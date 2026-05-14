package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FileMoveRequest struct {
	SourceRelPath string `json:"sourceRelPath"`
	TargetRelPath string `json:"targetRelPath"`
}

type FileMoveProposal struct {
	SourceRelPath string `json:"sourceRelPath"`
	TargetRelPath string `json:"targetRelPath"`
	Name          string `json:"name"`
	Action        string `json:"action"`
	Size          int64  `json:"size"`
	Message       string `json:"message"`
}

func PreviewFileMove(root string, request FileMoveRequest) (FileMoveProposal, error) {
	_, _, cleanSource, cleanTarget, info, err := resolveMoveTargets(root, request)
	if err != nil {
		return FileMoveProposal{}, err
	}

	return FileMoveProposal{
		SourceRelPath: filepath.ToSlash(cleanSource),
		TargetRelPath: filepath.ToSlash(cleanTarget),
		Name:          filepath.Base(cleanTarget),
		Action:        "move",
		Size:          info.Size(),
		Message:       fmt.Sprintf("Preview ready to move %s to %s.", filepath.ToSlash(cleanSource), filepath.ToSlash(cleanTarget)),
	}, nil
}

func ApplyFileMove(root string, request FileMoveRequest) (FileMoveProposal, error) {
	proposal, err := PreviewFileMove(root, request)
	if err != nil {
		return FileMoveProposal{}, err
	}

	absSource, absTarget, _, _, _, err := resolveMoveTargets(root, request)
	if err != nil {
		return FileMoveProposal{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return FileMoveProposal{}, err
	}
	if err := os.Rename(absSource, absTarget); err != nil {
		return FileMoveProposal{}, err
	}

	proposal.Message = fmt.Sprintf("Moved %s to %s.", proposal.SourceRelPath, proposal.TargetRelPath)
	return proposal, nil
}

func resolveMoveTargets(root string, request FileMoveRequest) (string, string, string, string, os.FileInfo, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", "", nil, err
	}

	cleanSource, err := cleanPreviewRelPath(request.SourceRelPath)
	if err != nil {
		return "", "", "", "", nil, err
	}
	if strings.HasSuffix(strings.TrimSpace(request.TargetRelPath), "/") || strings.HasSuffix(strings.TrimSpace(request.TargetRelPath), "\\") {
		return "", "", "", "", nil, errors.New("move target must include a file name")
	}
	cleanTarget, err := cleanPreviewRelPath(request.TargetRelPath)
	if err != nil {
		return "", "", "", "", nil, err
	}
	if filepath.ToSlash(cleanSource) == filepath.ToSlash(cleanTarget) {
		return "", "", "", "", nil, errors.New("move target must be different from the source")
	}
	if strings.HasPrefix(filepath.ToSlash(cleanSource), ".nexusdesk/") || strings.HasPrefix(filepath.ToSlash(cleanTarget), ".nexusdesk/") {
		return "", "", "", "", nil, errors.New("direct moves into or out of NexusDesk metadata are not allowed")
	}

	absSource, err := filepath.Abs(filepath.Join(absRoot, cleanSource))
	if err != nil {
		return "", "", "", "", nil, err
	}
	absTarget, err := filepath.Abs(filepath.Join(absRoot, cleanTarget))
	if err != nil {
		return "", "", "", "", nil, err
	}
	if err := ensureInsideRoot(absRoot, absSource); err != nil {
		return "", "", "", "", nil, err
	}
	if err := ensureInsideRoot(absRoot, absTarget); err != nil {
		return "", "", "", "", nil, err
	}

	info, err := os.Lstat(absSource)
	if os.IsNotExist(err) {
		return "", "", "", "", nil, errors.New("move source does not exist")
	}
	if err != nil {
		return "", "", "", "", nil, err
	}
	if info.IsDir() {
		return "", "", "", "", nil, errors.New("move source must be a file")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", "", "", nil, errors.New("move source cannot be a symlink")
	}
	if targetInfo, err := os.Lstat(absTarget); err == nil {
		if targetInfo.IsDir() {
			return "", "", "", "", nil, errors.New("move target cannot be a directory")
		}
		return "", "", "", "", nil, errors.New("move target already exists")
	} else if !os.IsNotExist(err) {
		return "", "", "", "", nil, err
	}
	if err := ensureWriteParentInsideRoot(absRoot, absTarget); err != nil {
		return "", "", "", "", nil, err
	}

	return absSource, absTarget, cleanSource, cleanTarget, info, nil
}
