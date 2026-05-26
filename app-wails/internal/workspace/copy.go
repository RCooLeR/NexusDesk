package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const fileCopyMaxBytes = 32 * 1024 * 1024

type FileCopyRequest struct {
	SourceRelPath string `json:"sourceRelPath"`
	TargetRelPath string `json:"targetRelPath"`
}

type FileCopyProposal struct {
	SourceRelPath string `json:"sourceRelPath"`
	TargetRelPath string `json:"targetRelPath"`
	Name          string `json:"name"`
	Action        string `json:"action"`
	Size          int64  `json:"size"`
	Message       string `json:"message"`
}

func PreviewFileCopy(root string, request FileCopyRequest) (FileCopyProposal, error) {
	_, _, cleanSource, cleanTarget, info, err := resolveCopyTargets(root, request)
	if err != nil {
		return FileCopyProposal{}, err
	}

	return FileCopyProposal{
		SourceRelPath: filepath.ToSlash(cleanSource),
		TargetRelPath: filepath.ToSlash(cleanTarget),
		Name:          filepath.Base(cleanTarget),
		Action:        "copy",
		Size:          info.Size(),
		Message:       fmt.Sprintf("Preview ready to copy %s to %s.", filepath.ToSlash(cleanSource), filepath.ToSlash(cleanTarget)),
	}, nil
}

func ApplyFileCopy(root string, request FileCopyRequest) (FileCopyProposal, error) {
	proposal, err := PreviewFileCopy(root, request)
	if err != nil {
		return FileCopyProposal{}, err
	}

	absSource, absTarget, _, _, info, err := resolveCopyTargets(root, request)
	if err != nil {
		return FileCopyProposal{}, err
	}
	content, err := os.ReadFile(absSource)
	if err != nil {
		return FileCopyProposal{}, err
	}
	if int64(len(content)) != info.Size() {
		return FileCopyProposal{}, errors.New("copy source changed while preparing the operation")
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return FileCopyProposal{}, err
	}
	if err := os.WriteFile(absTarget, content, info.Mode().Perm()); err != nil {
		return FileCopyProposal{}, err
	}

	proposal.Message = fmt.Sprintf("Copied %s to %s.", proposal.SourceRelPath, proposal.TargetRelPath)
	return proposal, nil
}

func resolveCopyTargets(root string, request FileCopyRequest) (string, string, string, string, os.FileInfo, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", "", "", nil, err
	}

	cleanSource, err := cleanPreviewRelPath(request.SourceRelPath)
	if err != nil {
		return "", "", "", "", nil, err
	}
	if strings.HasSuffix(strings.TrimSpace(request.TargetRelPath), "/") || strings.HasSuffix(strings.TrimSpace(request.TargetRelPath), "\\") {
		return "", "", "", "", nil, errors.New("copy target must include a file name")
	}
	cleanTarget, err := cleanPreviewRelPath(request.TargetRelPath)
	if err != nil {
		return "", "", "", "", nil, err
	}
	if filepath.ToSlash(cleanSource) == filepath.ToSlash(cleanTarget) {
		return "", "", "", "", nil, errors.New("copy target must be different from the source")
	}
	if strings.HasPrefix(filepath.ToSlash(cleanSource), ".nexusdesk/") || strings.HasPrefix(filepath.ToSlash(cleanTarget), ".nexusdesk/") {
		return "", "", "", "", nil, errors.New("direct copies into or out of Nexus metadata are not allowed")
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
		return "", "", "", "", nil, errors.New("copy source does not exist")
	}
	if err != nil {
		return "", "", "", "", nil, err
	}
	if info.IsDir() {
		return "", "", "", "", nil, errors.New("copy source must be a file")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", "", "", "", nil, errors.New("copy source cannot be a symlink")
	}
	if info.Size() > fileCopyMaxBytes {
		return "", "", "", "", nil, errors.New("copy source is too large for interactive file operations")
	}
	if targetInfo, err := os.Lstat(absTarget); err == nil {
		if targetInfo.IsDir() {
			return "", "", "", "", nil, errors.New("copy target cannot be a directory")
		}
		return "", "", "", "", nil, errors.New("copy target already exists")
	} else if !os.IsNotExist(err) {
		return "", "", "", "", nil, err
	}
	if err := ensureWriteParentInsideRoot(absRoot, absTarget); err != nil {
		return "", "", "", "", nil, err
	}

	return absSource, absTarget, cleanSource, cleanTarget, info, nil
}
