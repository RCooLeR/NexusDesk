package safearchive

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"
)

func TestValidateZipFilesRejectsTooManyFiles(t *testing.T) {
	err := ValidateZipFiles([]*zip.File{{}, {}, {}}, ZipLimits{MaxFiles: 2})
	if err == nil || !strings.Contains(err.Error(), "contains 3 files") {
		t.Fatalf("expected file count cap error, got %v", err)
	}
}

func TestValidateZipFilesRejectsOversizedMember(t *testing.T) {
	err := ValidateZipFiles([]*zip.File{{FileHeader: zip.FileHeader{Name: "xl/sharedStrings.xml", UncompressedSize64: 11}}}, ZipLimits{MaxMemberUncompressedBytes: 10})
	if err == nil || !strings.Contains(err.Error(), "xl/sharedStrings.xml") {
		t.Fatalf("expected member size cap error, got %v", err)
	}
}

func TestValidateZipFilesRejectsOversizedTotal(t *testing.T) {
	files := []*zip.File{
		{FileHeader: zip.FileHeader{Name: "a.xml", UncompressedSize64: 7}},
		{FileHeader: zip.FileHeader{Name: "b.xml", UncompressedSize64: 6}},
	}
	err := ValidateZipFiles(files, ZipLimits{MaxTotalUncompressedBytes: 12})
	if err == nil || !strings.Contains(err.Error(), "declares more than safety cap") {
		t.Fatalf("expected total size cap error, got %v", err)
	}
}

func TestReadZipFileRejectsOversizedMember(t *testing.T) {
	content := buildZip(t, "word/document.xml", strings.Repeat("a", 32))
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	_, err = ReadZipFile(reader.File[0], 8)
	if err == nil || !strings.Contains(err.Error(), "exceeding safety cap") {
		t.Fatalf("expected read cap error, got %v", err)
	}
}

func buildZip(t *testing.T, name string, content string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := entry.Write([]byte(content)); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	return buffer.Bytes()
}
