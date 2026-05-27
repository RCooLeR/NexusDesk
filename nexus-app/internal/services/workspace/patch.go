package workspace

import (
	"fmt"
	"os"
	"path/filepath"
)

const unifiedPatchMaxBytes = 2 * 1024 * 1024

type UnifiedPatchRequest struct {
	Patch string
}

type UnifiedPatchFileResult struct {
	RelPath string
	Action  string
	Before  string
	After   string
	Diff    string
	Message string
}

type UnifiedPatchProposal struct {
	Files      []UnifiedPatchFileResult
	FileCount  int
	RollbackID string
	Message    string
}

func (s *Service) PreviewUnifiedPatch(root string, request UnifiedPatchRequest) (UnifiedPatchProposal, error) {
	files, err := parseUnifiedPatch(request.Patch)
	if err != nil {
		return UnifiedPatchProposal{}, err
	}
	results := make([]UnifiedPatchFileResult, 0, len(files))
	for _, file := range files {
		before, after, err := applyPatchFile(root, file)
		if err != nil {
			return UnifiedPatchProposal{}, err
		}
		results = append(results, UnifiedPatchFileResult{
			RelPath: file.relPath,
			Action:  file.action,
			Before:  before,
			After:   after,
			Diff:    buildUnifiedDiff(file.relPath, before, after),
			Message: fmt.Sprintf("Preview ready to %s %s from unified patch.", file.action, file.relPath),
		})
	}
	return UnifiedPatchProposal{
		Files:     results,
		FileCount: len(results),
		Message:   fmt.Sprintf("Preview ready to apply unified patch to %d file(s).", len(results)),
	}, nil
}

func (s *Service) ApplyUnifiedPatch(root string, request UnifiedPatchRequest) (UnifiedPatchProposal, error) {
	proposal, err := s.PreviewUnifiedPatch(root, request)
	if err != nil {
		return UnifiedPatchProposal{}, err
	}
	rollbackPaths := make([]string, 0, len(proposal.Files))
	for _, file := range proposal.Files {
		rollbackPaths = append(rollbackPaths, file.RelPath)
	}
	rollback, err := s.prepareRollback(root, "patch", "unified patch", rollbackPaths)
	if err != nil {
		return UnifiedPatchProposal{}, err
	}
	for _, file := range proposal.Files {
		_, absTarget, _, err := resolveWriteTarget(root, file.RelPath)
		if err != nil {
			return UnifiedPatchProposal{}, err
		}
		if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
			return UnifiedPatchProposal{}, err
		}
		if err := os.WriteFile(absTarget, []byte(file.After), 0o644); err != nil {
			return UnifiedPatchProposal{}, err
		}
	}
	rollback, err = s.commitRollback(root, rollback)
	if err != nil {
		return UnifiedPatchProposal{}, err
	}
	proposal.RollbackID = rollback.ID
	proposal.Message = fmt.Sprintf("Applied unified patch to %d file(s). Rollback %s is available.", proposal.FileCount, rollback.ID)
	for index := range proposal.Files {
		proposal.Files[index].RollbackMessage(rollback.ID)
	}
	return proposal, nil
}

func (r *UnifiedPatchFileResult) RollbackMessage(rollbackID string) {
	r.Message = fmt.Sprintf("Patch applied for %s. Rollback %s is available.", r.RelPath, rollbackID)
}
