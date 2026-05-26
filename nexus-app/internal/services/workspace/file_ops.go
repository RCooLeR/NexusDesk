package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const fileOperationMaxBytes = 32 * 1024 * 1024

type FileCreateRequest struct {
	RelPath string
}

type FileCopyRequest struct {
	SourceRelPath string
	TargetRelPath string
}

type FileMoveRequest struct {
	SourceRelPath string
	TargetRelPath string
}

type FileOperationProposal struct {
	SourceRelPath string
	TargetRelPath string
	Name          string
	Action        string
	Size          int64
	RollbackID    string
	Message       string
}

func (s *Service) PreviewFileCreate(root string, request FileCreateRequest) (FileOperationProposal, error) {
	_, _, cleanRelPath, err := resolveNewFileTarget(root, request.RelPath, "create")
	if err != nil {
		return FileOperationProposal{}, err
	}
	relPath := filepath.ToSlash(cleanRelPath)
	return FileOperationProposal{
		TargetRelPath: relPath,
		Name:          filepath.Base(cleanRelPath),
		Action:        "create",
		Message:       fmt.Sprintf("Preview ready to create %s.", relPath),
	}, nil
}

func (s *Service) ApplyFileCreate(root string, request FileCreateRequest) (FileOperationProposal, error) {
	proposal, err := s.PreviewFileCreate(root, request)
	if err != nil {
		return FileOperationProposal{}, err
	}
	_, absTarget, cleanRelPath, err := resolveNewFileTarget(root, request.RelPath, "create")
	if err != nil {
		return FileOperationProposal{}, err
	}
	rollback, err := s.prepareRollback(root, proposal.Action, proposal.TargetRelPath, []string{cleanRelPath})
	if err != nil {
		return FileOperationProposal{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return FileOperationProposal{}, err
	}
	file, err := os.OpenFile(absTarget, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return FileOperationProposal{}, err
	}
	if err := file.Close(); err != nil {
		return FileOperationProposal{}, err
	}
	rollback, err = s.commitRollback(root, rollback)
	if err != nil {
		return FileOperationProposal{}, err
	}
	proposal.RollbackID = rollback.ID
	proposal.Message = fmt.Sprintf("Created %s. Rollback %s is available.", proposal.TargetRelPath, rollback.ID)
	return proposal, nil
}

func (s *Service) PreviewFileDelete(root string, relPath string) (FileOperationProposal, error) {
	_, cleanRelPath, info, err := resolveExistingFile(root, relPath, "delete")
	if err != nil {
		return FileOperationProposal{}, err
	}
	cleanRelPath = filepath.ToSlash(cleanRelPath)
	return FileOperationProposal{
		SourceRelPath: cleanRelPath,
		Name:          filepath.Base(cleanRelPath),
		Action:        "delete",
		Size:          info.Size(),
		Message:       fmt.Sprintf("Preview ready to delete %s.", cleanRelPath),
	}, nil
}

func (s *Service) ApplyFileDelete(root string, relPath string) (FileOperationProposal, error) {
	proposal, err := s.PreviewFileDelete(root, relPath)
	if err != nil {
		return FileOperationProposal{}, err
	}
	absTarget, cleanRelPath, _, err := resolveExistingFile(root, relPath, "delete")
	if err != nil {
		return FileOperationProposal{}, err
	}
	rollback, err := s.prepareRollback(root, proposal.Action, proposal.SourceRelPath, []string{cleanRelPath})
	if err != nil {
		return FileOperationProposal{}, err
	}
	if err := os.Remove(absTarget); err != nil {
		return FileOperationProposal{}, err
	}
	rollback, err = s.commitRollback(root, rollback)
	if err != nil {
		return FileOperationProposal{}, err
	}
	proposal.RollbackID = rollback.ID
	proposal.Message = fmt.Sprintf("Deleted %s. Rollback %s is available.", proposal.SourceRelPath, rollback.ID)
	return proposal, nil
}

func (s *Service) PreviewFileCopy(root string, request FileCopyRequest) (FileOperationProposal, error) {
	_, _, cleanSource, cleanTarget, info, err := resolveTransferTargets(root, request.SourceRelPath, request.TargetRelPath, "copy")
	if err != nil {
		return FileOperationProposal{}, err
	}
	sourceRelPath := filepath.ToSlash(cleanSource)
	targetRelPath := filepath.ToSlash(cleanTarget)
	return FileOperationProposal{
		SourceRelPath: sourceRelPath,
		TargetRelPath: targetRelPath,
		Name:          filepath.Base(cleanTarget),
		Action:        "copy",
		Size:          info.Size(),
		Message:       fmt.Sprintf("Preview ready to copy %s to %s.", sourceRelPath, targetRelPath),
	}, nil
}

func (s *Service) ApplyFileCopy(root string, request FileCopyRequest) (FileOperationProposal, error) {
	proposal, err := s.PreviewFileCopy(root, request)
	if err != nil {
		return FileOperationProposal{}, err
	}
	absSource, absTarget, _, cleanTarget, info, err := resolveTransferTargets(root, request.SourceRelPath, request.TargetRelPath, "copy")
	if err != nil {
		return FileOperationProposal{}, err
	}
	rollback, err := s.prepareRollback(root, proposal.Action, proposal.TargetRelPath, []string{cleanTarget})
	if err != nil {
		return FileOperationProposal{}, err
	}
	content, err := os.ReadFile(absSource)
	if err != nil {
		return FileOperationProposal{}, err
	}
	if int64(len(content)) != info.Size() {
		return FileOperationProposal{}, errors.New("copy source changed while preparing the operation")
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return FileOperationProposal{}, err
	}
	if err := os.WriteFile(absTarget, content, info.Mode().Perm()); err != nil {
		return FileOperationProposal{}, err
	}
	rollback, err = s.commitRollback(root, rollback)
	if err != nil {
		return FileOperationProposal{}, err
	}
	proposal.RollbackID = rollback.ID
	proposal.Message = fmt.Sprintf("Copied %s to %s. Rollback %s is available.", proposal.SourceRelPath, proposal.TargetRelPath, rollback.ID)
	return proposal, nil
}

func (s *Service) PreviewFileMove(root string, request FileMoveRequest) (FileOperationProposal, error) {
	return s.previewFileTransfer(root, request, "move")
}

func (s *Service) ApplyFileMove(root string, request FileMoveRequest) (FileOperationProposal, error) {
	return s.applyFileTransfer(root, request, "move")
}

func (s *Service) PreviewFileRename(root string, request FileMoveRequest) (FileOperationProposal, error) {
	return s.previewFileTransfer(root, request, "rename")
}

func (s *Service) ApplyFileRename(root string, request FileMoveRequest) (FileOperationProposal, error) {
	return s.applyFileTransfer(root, request, "rename")
}

func (s *Service) previewFileTransfer(root string, request FileMoveRequest, action string) (FileOperationProposal, error) {
	_, _, cleanSource, cleanTarget, info, err := resolveTransferTargets(root, request.SourceRelPath, request.TargetRelPath, action)
	if err != nil {
		return FileOperationProposal{}, err
	}
	sourceRelPath := filepath.ToSlash(cleanSource)
	targetRelPath := filepath.ToSlash(cleanTarget)
	return FileOperationProposal{
		SourceRelPath: sourceRelPath,
		TargetRelPath: targetRelPath,
		Name:          filepath.Base(cleanTarget),
		Action:        action,
		Size:          info.Size(),
		Message:       fmt.Sprintf("Preview ready to %s %s to %s.", action, sourceRelPath, targetRelPath),
	}, nil
}

func (s *Service) applyFileTransfer(root string, request FileMoveRequest, action string) (FileOperationProposal, error) {
	proposal, err := s.previewFileTransfer(root, request, action)
	if err != nil {
		return FileOperationProposal{}, err
	}
	absSource, absTarget, cleanSource, cleanTarget, _, err := resolveTransferTargets(root, request.SourceRelPath, request.TargetRelPath, "move")
	if err != nil {
		return FileOperationProposal{}, err
	}
	rollback, err := s.prepareRollback(root, proposal.Action, proposal.TargetRelPath, []string{cleanSource, cleanTarget})
	if err != nil {
		return FileOperationProposal{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return FileOperationProposal{}, err
	}
	if err := os.Rename(absSource, absTarget); err != nil {
		return FileOperationProposal{}, err
	}
	rollback, err = s.commitRollback(root, rollback)
	if err != nil {
		return FileOperationProposal{}, err
	}
	proposal.RollbackID = rollback.ID
	proposal.Message = fmt.Sprintf("%s %s to %s. Rollback %s is available.", operationPastTense(action), proposal.SourceRelPath, proposal.TargetRelPath, rollback.ID)
	return proposal, nil
}

func operationPastTense(action string) string {
	switch action {
	case "move":
		return "Moved"
	case "rename":
		return "Renamed"
	default:
		return titleAction(action)
	}
}
