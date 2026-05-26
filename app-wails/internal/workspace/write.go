package workspace

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"golang.org/x/text/encoding/charmap"
)

const (
	writeDiffMaxBytes    = 256 * 1024
	writeContentMaxBytes = 2 * 1024 * 1024
	binaryWriteMaxBytes  = 10 * 1024 * 1024
)

type FileWriteRequest struct {
	RelPath  string `json:"relPath"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
}

type BinaryFileWriteRequest struct {
	RelPath       string `json:"relPath"`
	Base64Content string `json:"base64Content"`
	ContentType   string `json:"contentType"`
}

type FileWriteProposal struct {
	RelPath  string `json:"relPath"`
	Name     string `json:"name"`
	Action   string `json:"action"`
	Diff     string `json:"diff"`
	Encoding string `json:"encoding"`
	SHA256   string `json:"sha256"`
	Size     int    `json:"size"`
	Message  string `json:"message"`
}

func PreviewFileWrite(root string, request FileWriteRequest) (FileWriteProposal, error) {
	absRoot, absTarget, cleanRel, err := resolveWriteTarget(root, request.RelPath)
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

	existing, action, err := readExistingWriteTarget(absRoot, absTarget)
	if err != nil {
		return FileWriteProposal{}, err
	}

	return FileWriteProposal{
		RelPath:  filepath.ToSlash(cleanRel),
		Name:     filepath.Base(cleanRel),
		Action:   action,
		Diff:     buildUnifiedDiff(filepath.ToSlash(cleanRel), existing, request.Content),
		Encoding: encoding,
		Size:     len(encoded),
		Message:  fmt.Sprintf("Preview ready to %s %s as %s inside the workspace.", action, filepath.ToSlash(cleanRel), encoding),
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
	encoded, _, err := encodeWriteContent(request.Content, request.Encoding)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if err := os.WriteFile(absTarget, encoded, 0o644); err != nil {
		return FileWriteProposal{}, err
	}

	proposal.Message = fmt.Sprintf("%s applied for %s as %s.", titleAction(proposal.Action), proposal.RelPath, proposal.Encoding)
	return proposal, nil
}

func PreviewFileAppend(root string, request FileWriteRequest) (FileWriteProposal, error) {
	_, absTarget, cleanRel, err := resolveWriteTarget(root, request.RelPath)
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
	if info, err := os.Lstat(absTarget); err == nil {
		if info.IsDir() {
			return FileWriteProposal{}, errors.New("file append target must be a file")
		}
	}

	rel := filepath.ToSlash(cleanRel)
	return FileWriteProposal{
		RelPath:  rel,
		Name:     filepath.Base(cleanRel),
		Action:   "append",
		Diff:     buildAppendDiff(rel, request.Content),
		Encoding: encoding,
		Size:     len(encoded),
		Message:  fmt.Sprintf("Preview ready to append %d bytes to %s as %s inside the workspace.", len(encoded), rel, encoding),
	}, nil
}

func ApplyFileAppend(root string, request FileWriteRequest) (FileWriteProposal, error) {
	proposal, err := PreviewFileAppend(root, request)
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
	encoded, _, err := encodeWriteContent(request.Content, request.Encoding)
	if err != nil {
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

	proposal.Message = fmt.Sprintf("Append applied for %s as %s.", proposal.RelPath, proposal.Encoding)
	return proposal, nil
}

func PreviewBinaryFileWrite(root string, request BinaryFileWriteRequest) (FileWriteProposal, error) {
	_, absTarget, cleanRel, err := resolveWriteTarget(root, request.RelPath)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if isExecutableBinaryWriteTarget(cleanRel) {
		return FileWriteProposal{}, errors.New("binary writes to executable targets are not allowed; build executable artifacts from source through a signed build pipeline")
	}
	content, err := decodeBinaryWriteContent(request.Base64Content)
	if err != nil {
		return FileWriteProposal{}, err
	}

	action := "create"
	if _, err := os.Stat(absTarget); err == nil {
		action = "update"
	} else if !os.IsNotExist(err) {
		return FileWriteProposal{}, err
	}

	rel := filepath.ToSlash(cleanRel)
	checksum := fmt.Sprintf("%x", sha256.Sum256(content))
	contentType := strings.TrimSpace(request.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return FileWriteProposal{
		RelPath:  rel,
		Name:     filepath.Base(cleanRel),
		Action:   action,
		Diff:     fmt.Sprintf("Binary %s preview for %s\nSize: %d bytes\nSHA-256: %s\nContent-Type: %s\nNo text diff is available for binary content.", action, rel, len(content), checksum, contentType),
		Encoding: "base64",
		SHA256:   checksum,
		Size:     len(content),
		Message:  fmt.Sprintf("Preview ready to %s binary file %s (%d bytes, sha256 %s) inside the workspace.", action, rel, len(content), checksum),
	}, nil
}

func ApplyBinaryFileWrite(root string, request BinaryFileWriteRequest) (FileWriteProposal, error) {
	proposal, err := PreviewBinaryFileWrite(root, request)
	if err != nil {
		return FileWriteProposal{}, err
	}

	_, absTarget, _, err := resolveWriteTarget(root, request.RelPath)
	if err != nil {
		return FileWriteProposal{}, err
	}
	content, err := decodeBinaryWriteContent(request.Base64Content)
	if err != nil {
		return FileWriteProposal{}, err
	}
	if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
		return FileWriteProposal{}, err
	}
	if err := os.WriteFile(absTarget, content, 0o644); err != nil {
		return FileWriteProposal{}, err
	}

	proposal.Message = fmt.Sprintf("%s applied for binary file %s (%d bytes, sha256 %s).", titleAction(proposal.Action), proposal.RelPath, proposal.Size, proposal.SHA256)
	return proposal, nil
}

func decodeBinaryWriteContent(encoded string) ([]byte, error) {
	encoded = strings.TrimSpace(encoded)
	if encoded == "" {
		return nil, errors.New("binary file content is required")
	}
	content, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.New("binary file content must be standard base64")
	}
	if len(content) > binaryWriteMaxBytes {
		return nil, errors.New("binary file write is too large")
	}
	return content, nil
}

func isExecutableBinaryWriteTarget(relPath string) bool {
	switch strings.ToLower(filepath.Ext(relPath)) {
	case ".exe", ".dll", ".com", ".scr", ".sys", ".msi", ".msp", ".ocx", ".cpl":
		return true
	default:
		return false
	}
}

func encodeWriteContent(content string, requestedEncoding string) ([]byte, string, error) {
	encoding := normalizeWriteEncoding(requestedEncoding)
	switch encoding {
	case "utf-8":
		return []byte(content), encoding, nil
	case "utf-8-bom":
		return append([]byte{0xef, 0xbb, 0xbf}, []byte(content)...), encoding, nil
	case "utf-16le":
		return encodeUTF16(content, binary.LittleEndian, []byte{0xff, 0xfe}), encoding, nil
	case "utf-16be":
		return encodeUTF16(content, binary.BigEndian, []byte{0xfe, 0xff}), encoding, nil
	case "windows-1251":
		encoded, err := charmap.Windows1251.NewEncoder().Bytes([]byte(content))
		if err != nil {
			return nil, "", errors.New("content cannot be encoded as windows-1251")
		}
		return encoded, encoding, nil
	default:
		return nil, "", fmt.Errorf("unsupported write encoding %q", requestedEncoding)
	}
}

func normalizeWriteEncoding(value string) string {
	encoding := strings.ToLower(strings.TrimSpace(value))
	switch encoding {
	case "", "utf8", "utf-8":
		return "utf-8"
	case "utf8-bom", "utf-8-bom", "utf-8 bom":
		return "utf-8-bom"
	case "utf16le", "utf-16le", "utf-16 le":
		return "utf-16le"
	case "utf16be", "utf-16be", "utf-16 be":
		return "utf-16be"
	case "cp1251", "windows1251", "windows-1251":
		return "windows-1251"
	default:
		return encoding
	}
}

func encodeUTF16(content string, byteOrder binary.ByteOrder, bom []byte) []byte {
	values := utf16.Encode([]rune(content))
	encoded := make([]byte, 0, len(bom)+len(values)*2)
	encoded = append(encoded, bom...)
	buffer := make([]byte, 2)
	for _, value := range values {
		byteOrder.PutUint16(buffer, value)
		encoded = append(encoded, buffer...)
	}
	return encoded
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
	if len(content) > writeDiffMaxBytes {
		return "[existing file omitted: larger than inline diff limit]", "update", nil
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

	for _, line := range lcsDiffLines(splitDiffLines(before), splitDiffLines(after)) {
		builder.WriteString(line)
		builder.WriteString("\n")
	}

	return builder.String()
}

func lcsDiffLines(beforeLines []string, afterLines []string) []string {
	table := make([][]int, len(beforeLines)+1)
	for index := range table {
		table[index] = make([]int, len(afterLines)+1)
	}
	for left := len(beforeLines) - 1; left >= 0; left-- {
		for right := len(afterLines) - 1; right >= 0; right-- {
			if beforeLines[left] == afterLines[right] {
				table[left][right] = table[left+1][right+1] + 1
			} else if table[left+1][right] >= table[left][right+1] {
				table[left][right] = table[left+1][right]
			} else {
				table[left][right] = table[left][right+1]
			}
		}
	}

	diff := []string{}
	left := 0
	right := 0
	for left < len(beforeLines) && right < len(afterLines) {
		if beforeLines[left] == afterLines[right] {
			diff = append(diff, " "+beforeLines[left])
			left++
			right++
			continue
		}
		if table[left+1][right] >= table[left][right+1] {
			diff = append(diff, "-"+beforeLines[left])
			left++
		} else {
			diff = append(diff, "+"+afterLines[right])
			right++
		}
	}
	for left < len(beforeLines) {
		diff = append(diff, "-"+beforeLines[left])
		left++
	}
	for right < len(afterLines) {
		diff = append(diff, "+"+afterLines[right])
		right++
	}
	return diff
}

func buildAppendDiff(relPath string, appended string) string {
	var builder strings.Builder
	builder.WriteString("--- a/")
	builder.WriteString(relPath)
	builder.WriteString("\n+++ b/")
	builder.WriteString(relPath)
	builder.WriteString("\n@@ append @@\n")
	for _, line := range splitDiffLines(appended) {
		builder.WriteString("+")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	if appended == "" {
		builder.WriteString("+\n")
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
