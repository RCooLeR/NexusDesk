package workspace

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

const defaultPreviewMaxBytes = 64 * 1024
const defaultImagePreviewMaxBytes = 2 * 1024 * 1024
const defaultDocumentPreviewMaxBytes = 8 * 1024 * 1024
const csvPreviewMaxRows = 50
const csvPreviewMaxColumns = 30
const csvProfileMaxRows = 1000
const csvProfileMaxBytes = 1024 * 1024

type PreviewOptions struct {
	MaxBytes int
}

type TablePreview struct {
	Columns   []string        `json:"columns"`
	Rows      [][]string      `json:"rows"`
	Profiles  []ColumnProfile `json:"profiles"`
	TotalRows int             `json:"totalRows"`
	Truncated bool            `json:"truncated"`
}

type ColumnProfile struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Missing  int    `json:"missing"`
	Distinct int    `json:"distinct"`
	Min      string `json:"min,omitempty"`
	Max      string `json:"max,omitempty"`
}

type FilePreview struct {
	RelPath   string        `json:"relPath"`
	Name      string        `json:"name"`
	Kind      string        `json:"kind"`
	FileType  string        `json:"fileType"`
	Content   string        `json:"content"`
	Encoding  string        `json:"encoding"`
	Table     *TablePreview `json:"table,omitempty"`
	Truncated bool          `json:"truncated"`
	Message   string        `json:"message"`
	Size      int64         `json:"size"`
}

func Preview(root string, relPath string, options PreviewOptions) (FilePreview, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return FilePreview{}, err
	}

	cleanRel, err := cleanPreviewRelPath(relPath)
	if err != nil {
		return FilePreview{}, err
	}

	target := filepath.Join(absRoot, cleanRel)
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return FilePreview{}, err
	}

	if err := ensureInsideRoot(absRoot, absTarget); err != nil {
		return FilePreview{}, err
	}

	info, err := os.Lstat(absTarget)
	if err != nil {
		return FilePreview{}, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return FilePreview{}, errors.New("workspace preview cannot follow symlinks")
	}

	evalRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return FilePreview{}, err
	}
	evalTarget, err := filepath.EvalSymlinks(absTarget)
	if err != nil {
		return FilePreview{}, err
	}
	if err := ensureInsideRoot(evalRoot, evalTarget); err != nil {
		return FilePreview{}, err
	}

	preview := FilePreview{
		RelPath:  filepath.ToSlash(cleanRel),
		Name:     info.Name(),
		Kind:     "file",
		FileType: detectFileTypeName(info.Name(), info.IsDir()),
		Size:     info.Size(),
	}

	if info.IsDir() {
		preview.Kind = "directory"
		preview.Message = "Select a file inside this folder to preview its contents."
		return preview, nil
	}

	if strings.EqualFold(filepath.Ext(info.Name()), ".pdf") {
		content, err := readBinaryDataURLContent(absTarget, info.Size(), binaryPreviewLimit(options.MaxBytes, defaultDocumentPreviewMaxBytes), "application/pdf")
		if err != nil {
			return FilePreview{}, err
		}
		if content == "" {
			preview.Kind = "unsupported"
			preview.Message = "PDF is too large to preview inline."
			return preview, nil
		}

		preview.Kind = "pdf"
		preview.Content = content
		preview.Message = "PDF preview rendered from the approved workspace root."
		return preview, nil
	}

	if preview.FileType == "image" {
		mimeType, ok := imageMimeType(absTarget)
		if !ok {
			preview.Kind = "unsupported"
			preview.Message = "Image type is not supported for inline preview."
			return preview, nil
		}

		content, err := readBinaryDataURLContent(absTarget, info.Size(), binaryPreviewLimit(options.MaxBytes, defaultImagePreviewMaxBytes), mimeType)
		if err != nil {
			return FilePreview{}, err
		}
		if content == "" {
			preview.Kind = "unsupported"
			preview.Message = "Image is too large to preview inline."
			return preview, nil
		}

		preview.Kind = "image"
		preview.Content = content
		preview.Message = "Image preview rendered from the approved workspace root."
		return preview, nil
	}

	content, truncated, err := readPreviewContent(absTarget, previewLimit(options.MaxBytes))
	if err != nil {
		return FilePreview{}, err
	}

	normalized, encoding, ok := normalizePreviewText(content)
	if !ok || isLikelyBinary(normalized) {
		preview.Kind = "unsupported"
		preview.Message = "Binary or unsupported text encoding files are not previewed yet."
		return preview, nil
	}

	preview.Content = string(normalized)
	preview.Encoding = encoding
	preview.Truncated = truncated
	if strings.EqualFold(filepath.Ext(info.Name()), ".csv") {
		table, err := parseCSVPreview(preview.Content, csvPreviewMaxRows, csvPreviewMaxColumns)
		if err == nil && len(table.Columns) > 0 {
			preview.Table = &table
			profiles, err := profileCSVFile(absTarget, csvProfileMaxBytes, csvProfileMaxRows, csvPreviewMaxColumns)
			if err == nil && len(profiles) > 0 {
				preview.Table.Profiles = profiles
			}
			preview.Message = fmt.Sprintf("CSV table preview loaded with %d rows.", table.TotalRows)
			if table.Truncated || truncated {
				preview.Message = "CSV table preview truncated to keep the app responsive."
			}
		}
	}
	if truncated && preview.Table == nil {
		preview.Message = "Preview truncated to keep the app responsive."
	} else if encoding != "utf-8" && preview.Message == "" {
		preview.Message = fmt.Sprintf("Decoded as %s.", encoding)
	}

	return preview, nil
}

func parseCSVPreview(content string, maxRows int, maxColumns int) (TablePreview, error) {
	records, err := readCSVRecords(content, 0)
	if err != nil {
		return TablePreview{}, err
	}
	if len(records) == 0 {
		return TablePreview{}, nil
	}

	columns := buildCSVColumns(records, maxColumns)
	rows := make([][]string, 0, minInt(len(records)-1, maxRows))
	totalRows := 0
	for _, record := range records[1:] {
		totalRows++
		if len(rows) >= maxRows {
			continue
		}
		rows = append(rows, trimRecordWidth(record, maxColumns))
	}

	return TablePreview{
		Columns:   columns,
		Rows:      rows,
		Profiles:  profileCSVColumns(columns, records[1:]),
		TotalRows: totalRows,
		Truncated: totalRows > len(rows) || recordsWiderThan(records, maxColumns),
	}, nil
}

func profileCSVFile(path string, maxBytes int, maxRows int, maxColumns int) ([]ColumnProfile, error) {
	content, _, err := readPreviewContent(path, maxBytes)
	if err != nil {
		return nil, err
	}

	normalized, _, ok := normalizePreviewText(content)
	if !ok || isLikelyBinary(normalized) {
		return nil, errors.New("csv profile content is not previewable text")
	}

	records, err := readCSVRecords(string(normalized), maxRows+1)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	columns := buildCSVColumns(records, maxColumns)
	return profileCSVColumns(columns, records[1:]), nil
}

func readCSVRecords(content string, maxRecords int) ([][]string, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1
	records := [][]string{}
	for maxRecords <= 0 || len(records) < maxRecords {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if len(records) > 0 {
				break
			}
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

func buildCSVColumns(records [][]string, maxColumns int) []string {
	columnCount := minInt(len(widestRecord(records, maxColumns)), maxColumns)
	if columnCount == 0 {
		return nil
	}

	columns := make([]string, 0, columnCount)
	for index := 0; index < columnCount; index++ {
		if index < len(records[0]) {
			name := strings.TrimSpace(records[0][index])
			if name != "" {
				columns = append(columns, name)
				continue
			}
		}
		columns = append(columns, fmt.Sprintf("Column %d", index+1))
	}

	return columns
}

func profileCSVColumns(columns []string, records [][]string) []ColumnProfile {
	profiles := make([]ColumnProfile, 0, len(columns))
	for columnIndex, name := range columns {
		values := make(map[string]struct{})
		missing := 0
		numericCount := 0
		integerCount := 0
		nonMissingCount := 0
		var minValue float64
		var maxValue float64
		hasNumber := false

		for _, record := range records {
			value := ""
			if columnIndex < len(record) {
				value = strings.TrimSpace(record[columnIndex])
			}
			if value == "" {
				missing++
				continue
			}

			nonMissingCount++
			values[value] = struct{}{}
			number, err := strconv.ParseFloat(strings.ReplaceAll(value, ",", ""), 64)
			if err != nil {
				continue
			}

			numericCount++
			if float64(int64(number)) == number {
				integerCount++
			}
			if !hasNumber || number < minValue {
				minValue = number
			}
			if !hasNumber || number > maxValue {
				maxValue = number
			}
			hasNumber = true
		}

		profile := ColumnProfile{
			Name:     name,
			Type:     inferCSVColumnType(nonMissingCount, numericCount, integerCount),
			Missing:  missing,
			Distinct: len(values),
		}
		if hasNumber && numericCount == nonMissingCount {
			profile.Min = formatCSVNumber(minValue)
			profile.Max = formatCSVNumber(maxValue)
		}
		profiles = append(profiles, profile)
	}

	return profiles
}

func inferCSVColumnType(nonMissingCount int, numericCount int, integerCount int) string {
	if nonMissingCount == 0 {
		return "empty"
	}
	if numericCount == nonMissingCount {
		if integerCount == nonMissingCount {
			return "integer"
		}
		return "number"
	}
	return "text"
}

func formatCSVNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func trimRecordWidth(record []string, maxColumns int) []string {
	if len(record) <= maxColumns {
		return record
	}
	return record[:maxColumns]
}

func widestRecord(records [][]string, maxColumns int) []string {
	widest := []string{}
	for _, record := range records {
		if len(record) > len(widest) {
			widest = record
		}
	}
	return trimRecordWidth(widest, maxColumns)
}

func recordsWiderThan(records [][]string, maxColumns int) bool {
	for _, record := range records {
		if len(record) > maxColumns {
			return true
		}
	}
	return false
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func cleanPreviewRelPath(relPath string) (string, error) {
	if strings.TrimSpace(relPath) == "" {
		return "", errors.New("workspace preview path is required")
	}

	cleanRel := filepath.Clean(filepath.FromSlash(relPath))
	if cleanRel == "." || filepath.IsAbs(cleanRel) {
		return "", errors.New("workspace preview path must be relative")
	}

	parts := strings.Split(cleanRel, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return "", errors.New("workspace preview path must stay inside the workspace")
		}
	}

	return cleanRel, nil
}

func ensureInsideRoot(root string, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}

	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return errors.New("workspace preview path must stay inside the workspace")
	}

	return nil
}

func previewLimit(maxBytes int) int {
	if maxBytes <= 0 {
		return defaultPreviewMaxBytes
	}
	return maxBytes
}

func binaryPreviewLimit(maxBytes int, defaultMaxBytes int) int {
	if maxBytes <= 0 {
		return defaultMaxBytes
	}
	return maxBytes
}

func readPreviewContent(path string, maxBytes int) ([]byte, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()

	content, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		return nil, false, err
	}

	if len(content) <= maxBytes {
		return content, false, nil
	}

	return content[:maxBytes], true, nil
}

func readBinaryDataURLContent(path string, size int64, maxBytes int, mimeType string) (string, error) {
	if size > int64(maxBytes) {
		return "", nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(content) > maxBytes {
		return "", nil
	}

	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(content)), nil
}

func imageMimeType(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png", true
	case ".jpg", ".jpeg":
		return "image/jpeg", true
	case ".gif":
		return "image/gif", true
	case ".webp":
		return "image/webp", true
	case ".svg":
		return "image/svg+xml", true
	case ".ico":
		return "image/x-icon", true
	default:
		return "", false
	}
}

func isLikelyBinary(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	for _, value := range content {
		if value == 0 {
			return true
		}
	}

	return false
}

func normalizePreviewText(content []byte) ([]byte, string, bool) {
	if bytes.HasPrefix(content, []byte{0xef, 0xbb, 0xbf}) {
		content = bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})
	}

	if decoded, ok := decodeUTF16(content); ok {
		return decoded.content, decoded.encoding, true
	}

	if utf8.Valid(content) {
		return content, "utf-8", true
	}

	trimmed := content
	for i := 0; i < utf8.UTFMax-1; i++ {
		if len(trimmed) == 0 {
			return nil, "", false
		}
		trimmed = trimmed[:len(trimmed)-1]
		if utf8.Valid(trimmed) {
			return trimmed, "utf-8", true
		}
	}

	if decoded, ok := decodeWindows1251(content); ok {
		return decoded, "windows-1251", true
	}

	return nil, "", false
}

type decodedText struct {
	content  []byte
	encoding string
}

func decodeUTF16(content []byte) (decodedText, bool) {
	byteOrder, encoding, content, ok := detectUTF16ByteOrder(content)
	if !ok {
		return decodedText{}, false
	}

	if len(content) < 2 {
		return decodedText{encoding: encoding}, true
	}
	if len(content)%2 != 0 {
		content = content[:len(content)-1]
	}

	values := make([]uint16, 0, len(content)/2)
	for index := 0; index < len(content); index += 2 {
		values = append(values, byteOrder.Uint16(content[index:index+2]))
	}

	decoded := []byte(string(utf16.Decode(values)))
	if !isMostlyPrintableText(decoded) {
		return decodedText{}, false
	}

	return decodedText{content: decoded, encoding: encoding}, true
}

func detectUTF16ByteOrder(content []byte) (binary.ByteOrder, string, []byte, bool) {
	switch {
	case bytes.HasPrefix(content, []byte{0xff, 0xfe}):
		return binary.LittleEndian, "utf-16le", content[2:], true
	case bytes.HasPrefix(content, []byte{0xfe, 0xff}):
		return binary.BigEndian, "utf-16be", content[2:], true
	}

	if len(content) < 4 {
		return nil, "", nil, false
	}

	evenZeros := 0
	oddZeros := 0
	for index, value := range content {
		if value != 0 {
			continue
		}
		if index%2 == 0 {
			evenZeros++
		} else {
			oddZeros++
		}
	}

	pairs := len(content) / 2
	if oddZeros*100 >= pairs*60 && evenZeros*100 <= pairs*10 {
		return binary.LittleEndian, "utf-16le", content, true
	}
	if evenZeros*100 >= pairs*60 && oddZeros*100 <= pairs*10 {
		return binary.BigEndian, "utf-16be", content, true
	}

	return nil, "", nil, false
}

func decodeWindows1251(content []byte) ([]byte, bool) {
	if bytes.Contains(content, []byte{0}) {
		return nil, false
	}

	decoded, err := charmap.Windows1251.NewDecoder().Bytes(content)
	if err != nil {
		return nil, false
	}
	if !containsCyrillic(decoded) || !isMostlyPrintableText(decoded) {
		return nil, false
	}

	return decoded, true
}

func containsCyrillic(content []byte) bool {
	count := 0
	for _, value := range string(content) {
		if unicode.Is(unicode.Cyrillic, value) {
			count++
		}
	}

	return count >= 2
}

func isMostlyPrintableText(content []byte) bool {
	if len(content) == 0 {
		return true
	}

	printable := 0
	total := 0
	for _, value := range string(content) {
		total++
		if value == '\n' || value == '\r' || value == '\t' || unicode.IsPrint(value) {
			printable++
		}
	}

	return total > 0 && printable*100/total >= 90
}
