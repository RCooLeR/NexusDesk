package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const commandTimeout = 4 * time.Second
const diffMaxBytes = 220 * 1024
const diffContextLines = "3"

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Status(root string) (Status, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return unavailableStatus("Open a workspace before reading Git status."), nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Status{}, err
	}
	repoRoot, err := gitOutput(absRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return unavailableStatus("Workspace is not inside a Git repository."), nil
	}
	branch := strings.TrimSpace(mustGitOutput(absRoot, "branch", "--show-current"))
	if branch == "" {
		branch = "detached"
	}
	head := strings.TrimSpace(mustGitOutput(absRoot, "rev-parse", "--short", "HEAD"))
	statusText := mustGitOutput(absRoot, "status", "--porcelain=v1", "--branch")
	changes, aheadBehind := parseStatus(statusText)
	staged, unstaged := splitChanges(changes)
	return Status{
		Available:     true,
		RepoRoot:      strings.TrimSpace(repoRoot),
		Branch:        branch,
		Head:          head,
		Dirty:         len(changes) > 0,
		ChangedFiles:  changes,
		StagedFiles:   staged,
		UnstagedFiles: unstaged,
		AheadBehind:   aheadBehind,
		Message:       statusMessage(branch, changes),
		GeneratedAt:   time.Now().UTC(),
	}, nil
}

func (s *Service) FileDiff(root string, relPath string) (FileDiff, error) {
	generatedAt := time.Now().UTC()
	root = strings.TrimSpace(root)
	if root == "" {
		return FileDiff{Path: relPath, Message: "Open a workspace before reading Git diff.", GeneratedAt: generatedAt}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return FileDiff{}, err
	}
	cleanPath, err := cleanRelPath(relPath)
	if err != nil {
		return FileDiff{Path: relPath, Message: err.Error(), GeneratedAt: generatedAt}, nil
	}
	if _, err := gitOutput(absRoot, "rev-parse", "--show-toplevel"); err != nil {
		return FileDiff{Path: cleanPath, Message: "Workspace is not inside a Git repository.", GeneratedAt: generatedAt}, nil
	}
	unstagedDiff, unstagedTruncated := cappedGitOutput(absRoot, "diff", "--no-ext-diff", "--unified="+diffContextLines, "--", cleanPath)
	stagedDiff, stagedTruncated := cappedGitOutput(absRoot, "diff", "--cached", "--no-ext-diff", "--unified="+diffContextLines, "--", cleanPath)
	message := "No diff for " + cleanPath + "."
	if stagedDiff != "" || unstagedDiff != "" {
		message = "Loaded read-only diff for " + cleanPath + "."
	}
	return FileDiff{
		Path:                  cleanPath,
		StagedDiff:            stagedDiff,
		StagedDiffTruncated:   stagedTruncated,
		StagedHunks:           parseDiffHunks(DiffKindStaged, stagedDiff),
		UnstagedDiff:          unstagedDiff,
		UnstagedDiffTruncated: unstagedTruncated,
		UnstagedHunks:         parseDiffHunks(DiffKindUnstaged, unstagedDiff),
		Message:               message,
		GeneratedAt:           generatedAt,
	}, nil
}

func (s *Service) ApplyFileAction(root string, relPath string, action FileAction) (FileActionResult, error) {
	generatedAt := time.Now().UTC()
	root = strings.TrimSpace(root)
	if root == "" {
		return FileActionResult{Path: relPath, Action: action, Message: "Open a workspace before changing Git state.", GeneratedAt: generatedAt}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return FileActionResult{}, err
	}
	cleanPath, err := cleanRelPath(relPath)
	if err != nil {
		return FileActionResult{Path: relPath, Action: action, Message: err.Error(), GeneratedAt: generatedAt}, nil
	}
	if _, err := gitOutput(absRoot, "rev-parse", "--show-toplevel"); err != nil {
		return FileActionResult{Path: cleanPath, Action: action, Message: "Workspace is not inside a Git repository.", GeneratedAt: generatedAt}, nil
	}
	if err := runFileAction(absRoot, cleanPath, action); err != nil {
		return FileActionResult{}, err
	}
	status, err := s.Status(absRoot)
	if err != nil {
		return FileActionResult{}, err
	}
	return FileActionResult{
		Path:        cleanPath,
		Action:      action,
		Message:     fileActionMessage(cleanPath, action),
		Status:      status,
		GeneratedAt: generatedAt,
	}, nil
}

func runFileAction(root string, relPath string, action FileAction) error {
	switch action {
	case FileActionStage:
		_, err := gitOutput(root, "add", "--", relPath)
		return err
	case FileActionUnstage:
		_, err := gitOutput(root, "restore", "--staged", "--", relPath)
		return err
	default:
		return fmt.Errorf("unsupported Git file action %q", action)
	}
}

func fileActionMessage(relPath string, action FileAction) string {
	switch action {
	case FileActionStage:
		return "Staged " + relPath + "."
	case FileActionUnstage:
		return "Unstaged " + relPath + "."
	default:
		return "Updated " + relPath + "."
	}
}

func unavailableStatus(message string) Status {
	return Status{Available: false, Message: message, GeneratedAt: time.Now().UTC()}
}

func gitOutput(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = root
	hideGitCommandWindow(command)
	output, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("git command timed out")
	}
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		return "", errors.New(text)
	}
	return string(output), nil
}

func mustGitOutput(root string, args ...string) string {
	output, err := gitOutput(root, args...)
	if err != nil {
		return ""
	}
	return output
}

func cappedGitOutput(root string, args ...string) (string, bool) {
	output, err := gitOutput(root, args...)
	if err != nil {
		return "", false
	}
	return windowUnifiedDiff(output)
}

func statusMessage(branch string, changes []FileChange) string {
	if len(changes) == 0 {
		return fmt.Sprintf("%s is clean.", branch)
	}
	return fmt.Sprintf("%s has %d changed file(s).", branch, len(changes))
}
