package workspace

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
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
	if info, err := os.Lstat(absTarget); err == nil && info.IsDir() {
		return FileWriteProposal{}, errors.New("file append target must be a file")
	}
	existingEncoding, hasExistingContent, err := inspectAppendTarget(absTarget)
	if err != nil {
		return FileWriteProposal{}, err
	}
	encoded, encoding, err := encodeAppendContent(request.Content, request.Encoding, existingEncoding, hasExistingContent)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if len(encoded) > writeContentMaxBytes {
		return FileWriteProposal{}, errors.New("encoded file append preview is too large")
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
	existingEncoding, hasExistingContent, err := inspectAppendTarget(absTarget)
	if err != nil {
		_ = s.discardPreparedRollback(root, rollback)
		return FileWriteProposal{}, err
	}
	encoded, _, err := encodeAppendContent(request.Content, request.Encoding, existingEncoding, hasExistingContent)
	if err != nil {
		_ = s.discardPreparedRollback(root, rollback)
		return FileWriteProposal{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		_ = s.discardPreparedRollback(root, rollback)
		return FileWriteProposal{}, err
	}
	file, err := os.OpenFile(absTarget, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		_ = s.discardPreparedRollback(root, rollback)
		return FileWriteProposal{}, err
	}
	written, writeErr := file.Write(encoded)
	closeErr := file.Close()
	if writeErr != nil {
		if written > 0 {
			return s.commitAppendRollbackAfterFailure(root, proposal, rollback, errors.Join(writeErr, closeErr))
		}
		_ = s.discardPreparedRollback(root, rollback)
		return FileWriteProposal{}, errors.Join(writeErr, closeErr)
	}
	if closeErr != nil {
		return s.commitAppendRollbackAfterFailure(root, proposal, rollback, closeErr)
	}
	rollback, err = s.commitRollback(root, rollback)
	if err != nil {
		return FileWriteProposal{}, err
	}

	proposal.RollbackID = rollback.ID
	proposal.Message = fmt.Sprintf("Append applied for %s as %s. Rollback %s is available.", proposal.RelPath, proposal.Encoding, rollback.ID)
	return proposal, nil
}

func (s *Service) commitAppendRollbackAfterFailure(root string, proposal FileWriteProposal, rollback RollbackRecord, cause error) (FileWriteProposal, error) {
	committed, err := s.commitRollback(root, rollback)
	if err != nil {
		return FileWriteProposal{}, errors.Join(cause, err)
	}
	proposal.RollbackID = committed.ID
	proposal.Message = fmt.Sprintf("Append did not complete cleanly for %s. Rollback %s is available.", proposal.RelPath, committed.ID)
	return proposal, fmt.Errorf("%s: %w", proposal.Message, cause)
}

func inspectAppendTarget(absTarget string) (string, bool, error) {
	file, err := os.Open(absTarget)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", false, err
	}
	if info.Size() == 0 {
		return "", false, nil
	}
	head, tail, err := readAppendSamples(file, info.Size())
	if err != nil {
		return "", false, err
	}
	encoding, err := appendSampleEncoding(head)
	if err != nil {
		return "", false, err
	}
	if err := ensureAppendSampleSafe(head, encoding); err != nil {
		return "", false, err
	}
	if len(tail) > 0 {
		if err := ensureAppendSampleSafe(tail, encoding); err != nil {
			return "", false, err
		}
	}
	return encoding, true, nil
}

func readAppendSamples(file *os.File, size int64) ([]byte, []byte, error) {
	const sampleSize = 4096
	headSize := int64(sampleSize)
	if size < headSize {
		headSize = size
	}
	head := make([]byte, headSize)
	read, err := file.ReadAt(head, 0)
	if err != nil && read == 0 {
		return nil, nil, err
	}
	head = head[:read]
	if size <= sampleSize {
		return head, nil, nil
	}
	tail := make([]byte, sampleSize)
	read, err = file.ReadAt(tail, size-sampleSize)
	if err != nil && read == 0 {
		return nil, nil, err
	}
	return head, tail[:read], nil
}

func appendSampleEncoding(sample []byte) (string, error) {
	if len(sample) == 0 {
		return "", nil
	}
	switch {
	case strings.HasPrefix(string(sample), "\xef\xbb\xbf"):
		return encodingUTF8BOM, nil
	case len(sample)%2 != 0 && (looksLikeUTF16LE(sample) || looksLikeUTF16BE(sample)):
		return "", errors.New("existing UTF-16 file has an invalid byte length")
	case strings.HasPrefix(string(sample), "\xff\xfe") || looksLikeUTF16LE(sample):
		return encodingUTF16LE, nil
	case strings.HasPrefix(string(sample), "\xfe\xff") || looksLikeUTF16BE(sample):
		return encodingUTF16BE, nil
	default:
		if looksBinary(sample) {
			return "", errors.New("existing file is not safe text")
		}
		if _, encoding, err := decodeText(sample); err == nil {
			return encoding, nil
		}
		return "", errors.New("existing file text encoding is unsupported")
	}
}

func ensureAppendSampleSafe(sample []byte, encoding string) error {
	if len(sample) == 0 {
		return nil
	}
	switch encoding {
	case encodingUTF16LE:
		text, err := decodeAppendUTF16Sample(sample, binary.LittleEndian, []byte{0xff, 0xfe})
		if err != nil || !appendDecodedTextSafe(text) {
			return errors.New("existing UTF-16 append target has mixed or invalid encoding")
		}
	case encodingUTF16BE:
		text, err := decodeAppendUTF16Sample(sample, binary.BigEndian, []byte{0xfe, 0xff})
		if err != nil || !appendDecodedTextSafe(text) {
			return errors.New("existing UTF-16 append target has mixed or invalid encoding")
		}
	default:
		if looksBinary(sample) {
			return errors.New("existing file is not safe text")
		}
	}
	return nil
}

func decodeAppendUTF16Sample(sample []byte, order binary.ByteOrder, bom []byte) (string, error) {
	if len(sample)%2 != 0 {
		return "", errors.New("invalid UTF-16 byte length")
	}
	if len(sample) >= 2 && sample[0] == bom[0] && sample[1] == bom[1] {
		sample = sample[2:]
	}
	return decodeUTF16(sample, order)
}

func appendDecodedTextSafe(text string) bool {
	if text == "" {
		return true
	}
	controls := 0
	for _, char := range text {
		if unicode.IsControl(char) && char != '\n' && char != '\r' && char != '\t' {
			controls++
		}
	}
	return controls*100/len([]rune(text)) <= 10
}

func titleAction(action string) string {
	if action == "" {
		return "Write"
	}
	return strings.ToUpper(action[:1]) + action[1:]
}
