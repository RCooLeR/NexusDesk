package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (s *Service) ApplyHunkAction(root string, relPath string, kind DiffKind, hunkIndex int, action HunkAction) (HunkActionResult, error) {
	generatedAt := time.Now().UTC()
	root = strings.TrimSpace(root)
	if root == "" {
		return HunkActionResult{Path: relPath, Action: action, DiffKind: kind, HunkIndex: hunkIndex, Message: "Open a workspace before changing Git state.", GeneratedAt: generatedAt}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return HunkActionResult{}, err
	}
	cleanPath, err := cleanRelPath(relPath)
	if err != nil {
		return HunkActionResult{Path: relPath, Action: action, DiffKind: kind, HunkIndex: hunkIndex, Message: err.Error(), GeneratedAt: generatedAt}, nil
	}
	if _, err := gitOutputFor(absRoot, operationStatus, "rev-parse", "--show-toplevel"); err != nil {
		return HunkActionResult{Path: cleanPath, Action: action, DiffKind: kind, HunkIndex: hunkIndex, Message: repositoryUnavailableMessage(absRoot, err), GeneratedAt: generatedAt}, nil
	}
	diff, err := hunkSourceDiff(absRoot, cleanPath, kind)
	if err != nil {
		return HunkActionResult{}, err
	}
	patch, err := extractHunkPatch(diff, hunkIndex)
	if err != nil {
		return HunkActionResult{Path: cleanPath, Action: action, DiffKind: kind, HunkIndex: hunkIndex, Message: err.Error(), GeneratedAt: generatedAt}, nil
	}
	args, err := hunkActionArgs(action, kind)
	if err != nil {
		return HunkActionResult{Path: cleanPath, Action: action, DiffKind: kind, HunkIndex: hunkIndex, Message: err.Error(), GeneratedAt: generatedAt}, nil
	}
	if err := applyGitPatch(absRoot, args, patch); err != nil {
		return HunkActionResult{}, err
	}
	status, err := s.Status(absRoot)
	if err != nil {
		return HunkActionResult{}, err
	}
	return HunkActionResult{
		Path:        cleanPath,
		Action:      action,
		DiffKind:    kind,
		HunkIndex:   hunkIndex,
		Patch:       patch,
		Message:     hunkActionMessage(cleanPath, action, hunkIndex),
		Status:      status,
		GeneratedAt: generatedAt,
	}, nil
}

func hunkSourceDiff(root string, relPath string, kind DiffKind) (string, error) {
	switch kind {
	case DiffKindUnstaged:
		return gitOutputFor(root, operationDiff, "diff", "--no-ext-diff", "--unified="+diffContextLines, "--", relPath)
	case DiffKindStaged:
		return gitOutputFor(root, operationDiff, "diff", "--cached", "--no-ext-diff", "--unified="+diffContextLines, "--", relPath)
	default:
		return "", fmt.Errorf("unsupported hunk diff kind %q", kind)
	}
}

func hunkActionArgs(action HunkAction, kind DiffKind) ([]string, error) {
	switch {
	case action == HunkActionStage && kind == DiffKindUnstaged:
		return []string{"--cached", "--whitespace=nowarn"}, nil
	case action == HunkActionUnstage && kind == DiffKindStaged:
		return []string{"--cached", "--reverse", "--whitespace=nowarn"}, nil
	default:
		return nil, errors.New("unsupported git hunk action")
	}
}

func extractHunkPatch(diff string, hunkIndex int) (string, error) {
	if hunkIndex < 0 {
		return "", errors.New("git hunk index must be zero or greater")
	}
	lines := strings.Split(strings.ReplaceAll(diff, "\r\n", "\n"), "\n")
	header := []string{}
	hunk := []string{}
	currentHunk := -1
	inTargetHunk := false
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			currentHunk++
			if inTargetHunk {
				break
			}
			inTargetHunk = currentHunk == hunkIndex
		}
		if currentHunk < 0 {
			header = append(header, line)
			continue
		}
		if inTargetHunk {
			hunk = append(hunk, line)
		}
	}
	if len(header) == 0 || len(hunk) == 0 {
		return "", errors.New("selected hunk is no longer available in the current diff")
	}
	patchLines := append([]string{}, header...)
	patchLines = append(patchLines, hunk...)
	return strings.TrimRight(strings.Join(patchLines, "\n"), "\n") + "\n", nil
}

func applyGitPatch(root string, args []string, patch string) error {
	timeout := timeoutForOperation(operationMutation)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", append([]string{"apply"}, args...)...)
	command.Dir = root
	command.Env = nonInteractiveGitEnv(os.Environ())
	hideGitCommandWindow(command)
	command.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	err := command.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("git apply timed out after %s", timeout)
	}
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func hunkActionMessage(relPath string, action HunkAction, hunkIndex int) string {
	switch action {
	case HunkActionStage:
		return fmt.Sprintf("Staged hunk %d in %s.", hunkIndex+1, relPath)
	case HunkActionUnstage:
		return fmt.Sprintf("Unstaged hunk %d in %s.", hunkIndex+1, relPath)
	default:
		return fmt.Sprintf("Updated hunk %d in %s.", hunkIndex+1, relPath)
	}
}
