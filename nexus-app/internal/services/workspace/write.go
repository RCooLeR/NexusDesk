package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	writeDiffMaxBytes    = 256 * 1024
	writeContentMaxBytes = 2 * 1024 * 1024
)

type FileWriteRequest struct {
	RelPath  string
	Content  string
	Encoding string
}

type FileWriteProposal struct {
	RelPath    string
	Name       string
	Action     string
	Diff       string
	Encoding   string
	Size       int
	RollbackID string
	Message    string
}

func (s *Service) PreviewFileWrite(root string, request FileWriteRequest) (FileWriteProposal, error) {
	_, absTarget, cleanRelPath, err := resolveWriteTarget(root, request.RelPath)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if len(request.Content) > writeContentMaxBytes {
		return FileWriteProposal{}, errors.New("file write preview is too large")
	}
	encoded, encoding, err := encodeWriteContent(request.Content, request.Encoding)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if len(encoded) > writeContentMaxBytes {
		return FileWriteProposal{}, errors.New("encoded file write preview is too large")
	}
	existing, action, err := readExistingWriteTarget(absTarget)
	if err != nil {
		return FileWriteProposal{}, err
	}

	relPath := filepath.ToSlash(cleanRelPath)
	return FileWriteProposal{
		RelPath:  relPath,
		Name:     filepath.Base(cleanRelPath),
		Action:   action,
		Diff:     buildUnifiedDiff(relPath, existing, request.Content),
		Encoding: encoding,
		Size:     len(encoded),
		Message:  fmt.Sprintf("Preview ready to %s %s as %s inside the workspace.", action, relPath, encoding),
	}, nil
}

func (s *Service) ApplyFileWrite(root string, request FileWriteRequest) (FileWriteProposal, error) {
	proposal, err := s.PreviewFileWrite(root, request)
	if err != nil {
		return FileWriteProposal{}, err
	}
	_, absTarget, cleanRelPath, err := resolveWriteTarget(root, request.RelPath)
	if err != nil {
		return FileWriteProposal{}, err
	}
	rollback, err := s.prepareRollback(root, proposal.Action, proposal.RelPath, []string{cleanRelPath})
	if err != nil {
		return FileWriteProposal{}, err
	}
	encoded, _, err := encodeWriteContent(request.Content, request.Encoding)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return FileWriteProposal{}, err
	}
	if err := os.WriteFile(absTarget, encoded, 0o644); err != nil {
		return FileWriteProposal{}, err
	}
	rollback, err = s.commitRollback(root, rollback)
	if err != nil {
		return FileWriteProposal{}, err
	}

	proposal.RollbackID = rollback.ID
	proposal.Message = fmt.Sprintf("%s applied for %s as %s. Rollback %s is available.", titleAction(proposal.Action), proposal.RelPath, proposal.Encoding, rollback.ID)
	return proposal, nil
}

func (s *Service) PreviewFileAppend(root string, request FileWriteRequest) (FileWriteProposal, error) {
	_, absTarget, cleanRelPath, err := resolveWriteTarget(root, request.RelPath)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if len(request.Content) > writeContentMaxBytes {
		return FileWriteProposal{}, errors.New("file append preview is too large")
	}
	encoded, encoding, err := encodeWriteContent(request.Content, request.Encoding)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if len(encoded) > writeContentMaxBytes {
		return FileWriteProposal{}, errors.New("encoded file append preview is too large")
	}
	if info, err := os.Lstat(absTarget); err == nil && info.IsDir() {
		return FileWriteProposal{}, errors.New("file append target must be a file")
	}
	if err := ensureAppendTargetSafe(absTarget); err != nil {
		return FileWriteProposal{}, err
	}

	relPath := filepath.ToSlash(cleanRelPath)
	return FileWriteProposal{
		RelPath:  relPath,
		Name:     filepath.Base(cleanRelPath),
		Action:   "append",
		Diff:     buildAppendDiff(relPath, request.Content),
		Encoding: encoding,
		Size:     len(encoded),
		Message:  fmt.Sprintf("Preview ready to append %d bytes to %s as %s inside the workspace.", len(encoded), relPath, encoding),
	}, nil
}

func (s *Service) ApplyFileAppend(root string, request FileWriteRequest) (FileWriteProposal, error) {
	proposal, err := s.PreviewFileAppend(root, request)
	if err != nil {
		return FileWriteProposal{}, err
	}
	_, absTarget, cleanRelPath, err := resolveWriteTarget(root, request.RelPath)
	if err != nil {
		return FileWriteProposal{}, err
	}
	rollback, err := s.prepareRollback(root, proposal.Action, proposal.RelPath, []string{cleanRelPath})
	if err != nil {
		return FileWriteProposal{}, err
	}
	encoded, _, err := encodeWriteContent(request.Content, request.Encoding)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return FileWriteProposal{}, err
	}
	file, err := os.OpenFile(absTarget, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return FileWriteProposal{}, err
	}
	defer file.Close()
	if _, err := file.Write(encoded); err != nil {
		return FileWriteProposal{}, err
	}
	rollback, err = s.commitRollback(root, rollback)
	if err != nil {
		return FileWriteProposal{}, err
	}

	proposal.RollbackID = rollback.ID
	proposal.Message = fmt.Sprintf("Append applied for %s as %s. Rollback %s is available.", proposal.RelPath, proposal.Encoding, rollback.ID)
	return proposal, nil
}

func ensureAppendTargetSafe(absTarget string) error {
	file, err := os.Open(absTarget)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()
	sample := make([]byte, 4096)
	read, err := file.Read(sample)
	if err != nil && read == 0 {
		return nil
	}
	sample = sample[:read]
	if looksBinary(sample) && !looksLikeUTF16LE(sample) && !looksLikeUTF16BE(sample) {
		return errors.New("existing file is not safe text")
	}
	return nil
}

func titleAction(action string) string {
	if action == "" {
		return "Write"
	}
	return strings.ToUpper(action[:1]) + action[1:]
}
