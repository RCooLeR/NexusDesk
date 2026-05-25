package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const writePreviewMaxBytes = 256 * 1024

type FileWriteRequest struct {
	RelPath string `json:"relPath"`
	Content string `json:"content"`
}

type FileWriteProposal struct {
	RelPath string `json:"relPath"`
	Name    string `json:"name"`
	Action  string `json:"action"`
	Diff    string `json:"diff"`
	Size    int    `json:"size"`
	Message string `json:"message"`
}

func PreviewFileWrite(root string, request FileWriteRequest) (FileWriteProposal, error) {
	absRoot, absTarget, cleanRel, err := resolveWriteTarget(root, request.RelPath)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if len(request.Content) > writePreviewMaxBytes {
		return FileWriteProposal{}, errors.New("file write preview is too large")
	}

	existing, action, err := readExistingWriteTarget(absRoot, absTarget)
	if err != nil {
		return FileWriteProposal{}, err
	}

	return FileWriteProposal{
		RelPath: filepath.ToSlash(cleanRel),
		Name:    filepath.Base(cleanRel),
		Action:  action,
		Diff:    buildUnifiedDiff(filepath.ToSlash(cleanRel), existing, request.Content),
		Size:    len([]byte(request.Content)),
		Message: fmt.Sprintf("Preview ready to %s %s inside the workspace.", action, filepath.ToSlash(cleanRel)),
	}, nil
}

func ApplyFileWrite(root string, request FileWriteRequest) (FileWriteProposal, error) {
	proposal, err := PreviewFileWrite(root, request)
	if err != nil {
		return FileWriteProposal{}, err
	}

	_, absTarget, _, err := resolveWriteTarget(root, request.RelPath)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return FileWriteProposal{}, err
	}
	if err := os.WriteFile(absTarget, []byte(request.Content), 0o644); err != nil {
		return FileWriteProposal{}, err
	}

	proposal.Message = fmt.Sprintf("%s applied for %s.", titleAction(proposal.Action), proposal.RelPath)
	return proposal, nil
}

func resolveWriteTarget(root string, relPath string) (string, string, string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", err
	}
	cleanRel, err := cleanPreviewRelPath(relPath)
	if err != nil {
		return "", "", "", err
	}
	if strings.HasPrefix(filepath.ToSlash(cleanRel), ".nexusdesk/") {
		return "", "", "", errors.New("direct writes to Nexus metadata are not allowed")
	}

	absTarget, err := filepath.Abs(filepath.Join(absRoot, cleanRel))
	if err != nil {
		return "", "", "", err
	}
	if err := ensureInsideRoot(absRoot, absTarget); err != nil {
		return "", "", "", err
	}
	if info, err := os.Lstat(absTarget); err == nil {
		if info.IsDir() {
			return "", "", "", errors.New("file write target must be a file")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", "", "", errors.New("file write target cannot be a symlink")
		}
	}
	if err := ensureWriteParentInsideRoot(absRoot, absTarget); err != nil {
		return "", "", "", err
	}

	return absRoot, absTarget, cleanRel, nil
}

func ensureWriteParentInsideRoot(absRoot string, absTarget string) error {
	evalRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return err
	}

	parent := filepath.Dir(absTarget)
	for {
		if info, err := os.Lstat(parent); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return errors.New("file write parent cannot be a symlink")
			}
			evalParent, err := filepath.EvalSymlinks(parent)
			if err != nil {
				return err
			}
			return ensureInsideRoot(evalRoot, evalParent)
		}

		next := filepath.Dir(parent)
		if next == parent {
			return errors.New("file write parent path is invalid")
		}
		parent = next
	}
}

func readExistingWriteTarget(absRoot string, absTarget string) (string, string, error) {
	if err := ensureInsideRoot(absRoot, absTarget); err != nil {
		return "", "", err
	}
	content, err := os.ReadFile(absTarget)
	if os.IsNotExist(err) {
		return "", "create", nil
	}
	if err != nil {
		return "", "", err
	}
	if len(content) > writePreviewMaxBytes {
		return "", "", errors.New("existing file is too large for write preview")
	}
	normalized, _, ok := normalizePreviewText(content)
	if !ok || isLikelyBinary(normalized) {
		return "", "", errors.New("existing file is not safe text")
	}
	return string(normalized), "update", nil
}

func buildUnifiedDiff(relPath string, before string, after string) string {
	var builder strings.Builder
	builder.WriteString("--- a/")
	builder.WriteString(relPath)
	builder.WriteString("\n+++ b/")
	builder.WriteString(relPath)
	builder.WriteString("\n")

	beforeLines := splitDiffLines(before)
	afterLines := splitDiffLines(after)
	maxLines := len(beforeLines)
	if len(afterLines) > maxLines {
		maxLines = len(afterLines)
	}

	for index := 0; index < maxLines; index++ {
		beforeLine := ""
		afterLine := ""
		if index < len(beforeLines) {
			beforeLine = beforeLines[index]
		}
		if index < len(afterLines) {
			afterLine = afterLines[index]
		}
		if beforeLine == afterLine {
			builder.WriteString(" ")
			builder.WriteString(beforeLine)
			builder.WriteString("\n")
			continue
		}
		if index < len(beforeLines) {
			builder.WriteString("-")
			builder.WriteString(beforeLine)
			builder.WriteString("\n")
		}
		if index < len(afterLines) {
			builder.WriteString("+")
			builder.WriteString(afterLine)
			builder.WriteString("\n")
		}
	}

	return builder.String()
}

func splitDiffLines(content string) []string {
	if content == "" {
		return []string{}
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.TrimSuffix(content, "\n")
	if content == "" {
		return []string{}
	}
	return strings.Split(content, "\n")
}

func titleAction(action string) string {
	if action == "" {
		return "Write"
	}
	return strings.ToUpper(action[:1]) + action[1:]
}
