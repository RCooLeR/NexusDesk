package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileDeleteProposal struct {
	RelPath string `json:"relPath"`
	Name    string `json:"name"`
	Action  string `json:"action"`
	Diff    string `json:"diff"`
	Size    int64  `json:"size"`
	Message string `json:"message"`
}

func PreviewFileDelete(root string, relPath string) (FileDeleteProposal, error) {
	absTarget, cleanRel, info, err := resolveDeleteTarget(root, relPath)
	if err != nil {
		return FileDeleteProposal{}, err
	}

	diff := ""
	if info.Size() <= writePreviewMaxBytes {
		if content, readErr := os.ReadFile(absTarget); readErr == nil {
			if normalized, _, ok := normalizePreviewText(content); ok && !isLikelyBinary(normalized) {
				diff = buildUnifiedDiff(filepath.ToSlash(cleanRel), string(normalized), "")
			}
		}
	}

	return FileDeleteProposal{
		RelPath: filepath.ToSlash(cleanRel),
		Name:    filepath.Base(cleanRel),
		Action:  "delete",
		Diff:    diff,
		Size:    info.Size(),
		Message: fmt.Sprintf("Preview ready to delete %s from the workspace.", filepath.ToSlash(cleanRel)),
	}, nil
}

func ApplyFileDelete(root string, relPath string) (FileDeleteProposal, error) {
	proposal, err := PreviewFileDelete(root, relPath)
	if err != nil {
		return FileDeleteProposal{}, err
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return FileDeleteProposal{}, err
	}
	cleanRel, err := cleanPreviewRelPath(relPath)
	if err != nil {
		return FileDeleteProposal{}, err
	}
	absTarget, err := filepath.Abs(filepath.Join(absRoot, cleanRel))
	if err != nil {
		return FileDeleteProposal{}, err
	}
	if err := ensureInsideRoot(absRoot, absTarget); err != nil {
		return FileDeleteProposal{}, err
	}
	if err := os.Remove(absTarget); err != nil {
		return FileDeleteProposal{}, err
	}

	proposal.Message = fmt.Sprintf("Deleted %s from the workspace.", proposal.RelPath)
	return proposal, nil
}

func resolveDeleteTarget(root string, relPath string) (string, string, os.FileInfo, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", nil, err
	}
	cleanRel, err := cleanPreviewRelPath(relPath)
	if err != nil {
		return "", "", nil, err
	}
	if strings.HasPrefix(filepath.ToSlash(cleanRel), ".nexusdesk/") {
		return "", "", nil, errors.New("direct deletes from NexusDesk metadata are not allowed")
	}

	absTarget, err := filepath.Abs(filepath.Join(absRoot, cleanRel))
	if err != nil {
		return "", "", nil, err
	}
	if err := ensureInsideRoot(absRoot, absTarget); err != nil {
		return "", "", nil, err
	}

	info, err := os.Lstat(absTarget)
	if os.IsNotExist(err) {
		return "", "", nil, errors.New("delete target does not exist")
	}
	if err != nil {
		return "", "", nil, err
	}
	if info.IsDir() {
		return "", "", nil, errors.New("delete target must be a file")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", nil, errors.New("delete target cannot be a symlink")
	}

	return absTarget, cleanRel, info, nil
}
