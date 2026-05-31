package workspace

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestPreviewFileWriteBuildsCreateDiff(t *testing.T) {
	root := t.TempDir()

	proposal, err := New().PreviewFileWrite(root, FileWriteRequest{
		RelPath: "docs/new.md",
		Content: "# New\n",
	})
	if err != nil {
		t.Fatalf("PreviewFileWrite returned error: %v", err)
	}
	if proposal.Action != "create" {
		t.Fatalf("expected create action, got %s", proposal.Action)
	}
	if !strings.Contains(proposal.Diff, "+++ b/docs/new.md") || !strings.Contains(proposal.Diff, "+# New") {
		t.Fatalf("unexpected diff: %s", proposal.Diff)
	}
}

func TestApplyFileWriteUpdatesTextFileAndCreatesRollback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "notes.md"), "old\n")

	proposal, err := New().ApplyFileWrite(root, FileWriteRequest{
		RelPath: "docs/notes.md",
		Content: "new\n",
	})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if proposal.Action != "update" || proposal.RollbackID == "" {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}
	assertFileContent(t, filepath.Join(root, "docs", "notes.md"), "new\n")

	rollbacks, err := New().ListRollbacks(root)
	if err != nil {
		t.Fatalf("ListRollbacks returned error: %v", err)
	}
	if len(rollbacks) != 1 || rollbacks[0].ID != proposal.RollbackID {
		t.Fatalf("expected committed rollback, got %#v", rollbacks)
	}
}

func TestRollbackStorageUsageSummarizesSnapshots(t *testing.T) {
	root := t.TempDir()
	service := New()
	writeFile(t, filepath.Join(root, "docs", "notes.md"), "old\n")

	if _, err := service.ApplyFileWrite(root, FileWriteRequest{RelPath: "docs/notes.md", Content: "new\n"}); err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	usage, err := service.RollbackStorageUsage(root)
	if err != nil {
		t.Fatalf("RollbackStorageUsage returned error: %v", err)
	}
	if usage.Records != 1 || usage.ActiveRecords != 1 || usage.Entries != 1 {
		t.Fatalf("unexpected rollback usage counts: %#v", usage)
	}
	if usage.SnapshotBytes != int64(len("old\n")) {
		t.Fatalf("expected source snapshot bytes, got %#v", usage)
	}
	if usage.StoredBytes < usage.SnapshotBytes {
		t.Fatalf("expected stored bytes to include snapshot/log bytes, got %#v", usage)
	}
}

func TestRollbackSnapshotsUseContentAddressedStorageForIdenticalContent(t *testing.T) {
	root := t.TempDir()
	service := New()
	path := filepath.Join(root, "docs", "notes.md")
	writeFile(t, path, "same\n")

	first, err := service.ApplyFileWrite(root, FileWriteRequest{RelPath: "docs/notes.md", Content: "first change\n"})
	if err != nil {
		t.Fatalf("first ApplyFileWrite returned error: %v", err)
	}
	writeFile(t, path, "same\n")
	second, err := service.ApplyFileWrite(root, FileWriteRequest{RelPath: "docs/notes.md", Content: "second change\n"})
	if err != nil {
		t.Fatalf("second ApplyFileWrite returned error: %v", err)
	}

	records, err := service.ListRollbacks(root)
	if err != nil {
		t.Fatalf("ListRollbacks returned error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected two rollback records, got %#v", records)
	}
	if records[0].ID != second.RollbackID || records[1].ID != first.RollbackID {
		t.Fatalf("unexpected rollback order: %#v", records)
	}
	firstBackup := records[1].Entries[0].BackupRelPath
	secondBackup := records[0].Entries[0].BackupRelPath
	if firstBackup == "" || firstBackup != secondBackup {
		t.Fatalf("expected identical snapshots to share one backup blob, got %q and %q", firstBackup, secondBackup)
	}
	if !strings.HasPrefix(firstBackup, rollbackBlobDirRelPath+"/") {
		t.Fatalf("expected content-addressed rollback blob path, got %q", firstBackup)
	}
	if got := countRollbackBlobFiles(t, root); got != 1 {
		t.Fatalf("expected one deduplicated rollback blob, got %d", got)
	}

	result, err := service.ApplyRollback(root, second.RollbackID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Restored) != 1 || result.Restored[0] != "docs/notes.md" {
		t.Fatalf("unexpected rollback result: %#v", result)
	}
	assertFileContent(t, path, "same\n")
}

func TestBuildUnifiedDiffUsesBoundedFallbackForLargeLineCount(t *testing.T) {
	beforeLines := make([]string, 0, 3200)
	afterLines := make([]string, 0, 3200)
	for index := 0; index < 3200; index++ {
		line := "line-" + strconv.Itoa(index)
		beforeLines = append(beforeLines, line)
		afterLines = append(afterLines, line)
	}
	afterLines[1600] = "line-1600 changed"

	diff := buildUnifiedDiff("docs/large.txt", strings.Join(beforeLines, "\n"), strings.Join(afterLines, "\n"))

	for _, expected := range []string{
		"@@ bounded diff: large input, 1 removed line(s), 1 added line(s) @@",
		" line-1599",
		"-line-1600",
		"+line-1600 changed",
		" line-1601",
	} {
		if !strings.Contains(diff, expected) {
			t.Fatalf("expected bounded diff to contain %q:\n%s", expected, diff)
		}
	}
	if len(diff) > 2000 {
		t.Fatalf("expected bounded diff to stay compact, got %d bytes", len(diff))
	}
}

func TestApplyRollbackRestoresUpdatedTextFile(t *testing.T) {
	root := t.TempDir()
	service := New()
	writeFile(t, filepath.Join(root, "docs", "notes.md"), "old\n")

	proposal, err := service.ApplyFileWrite(root, FileWriteRequest{RelPath: "docs/notes.md", Content: "new\n"})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	result, err := service.ApplyRollback(root, proposal.RollbackID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Restored) != 1 || result.Restored[0] != "docs/notes.md" {
		t.Fatalf("unexpected rollback result: %#v", result)
	}
	assertFileContent(t, filepath.Join(root, "docs", "notes.md"), "old\n")
}

func TestApplyRollbackRemovesCreatedFile(t *testing.T) {
	root := t.TempDir()
	service := New()

	proposal, err := service.ApplyFileWrite(root, FileWriteRequest{RelPath: "docs/new.md", Content: "# New\n"})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	result, err := service.ApplyRollback(root, proposal.RollbackID)
	if err != nil {
		t.Fatalf("ApplyRollback returned error: %v", err)
	}
	if len(result.Removed) != 1 || result.Removed[0] != "docs/new.md" {
		t.Fatalf("unexpected rollback result: %#v", result)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "new.md")); !os.IsNotExist(err) {
		t.Fatalf("expected created file to be removed, got err=%v", err)
	}
}

func TestApplyFileAppendDoesNotTruncateLargeFile(t *testing.T) {
	root := t.TempDir()
	large := strings.Repeat("a", 600*1024)
	writeFile(t, filepath.Join(root, "docs", "large.txt"), large)

	proposal, err := New().ApplyFileAppend(root, FileWriteRequest{
		RelPath: "docs/large.txt",
		Content: "\nappended\n",
	})
	if err != nil {
		t.Fatalf("ApplyFileAppend returned error: %v", err)
	}
	if proposal.Action != "append" || proposal.RollbackID == "" {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}
	assertFileContent(t, filepath.Join(root, "docs", "large.txt"), large+"\nappended\n")
}

func TestApplyFileAppendDiscardsRollbackOnPreMutationFailure(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "notes.txt"), "original\n")

	if _, err := New().ApplyFileAppend(root, FileWriteRequest{
		RelPath:  "docs/notes.txt",
		Content:  "appended\n",
		Encoding: "unsupported-encoding",
	}); err == nil {
		t.Fatal("expected unsupported encoding to fail")
	}
	assertFileContent(t, filepath.Join(root, "docs", "notes.txt"), "original\n")
	rollbacks, err := New().ListRollbacks(root)
	if err != nil {
		t.Fatalf("ListRollbacks returned error: %v", err)
	}
	if len(rollbacks) != 0 {
		t.Fatalf("expected no committed rollback, got %#v", rollbacks)
	}
	entries, err := os.ReadDir(filepath.Join(root, ".nexusdesk", "rollbacks"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("ReadDir rollbacks failed: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf("expected prepared rollback directory to be discarded, found %s", entry.Name())
		}
	}
}

func TestPreviewFileAppendRejectsBinaryTarget(t *testing.T) {
	root := t.TempDir()
	writeBytes(t, filepath.Join(root, "blob.bin"), []byte{'a', 0x00, 'b'})

	if _, err := New().PreviewFileAppend(root, FileWriteRequest{RelPath: "blob.bin", Content: "text"}); err == nil {
		t.Fatal("expected binary append target to be rejected")
	}
}

func TestApplyFileAppendPreservesExistingUTF16LEEncoding(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "docs", "utf16.txt")
	writeBytes(t, path, encodeUTF16("hello", binary.LittleEndian, []byte{0xff, 0xfe}))

	proposal, err := New().ApplyFileAppend(root, FileWriteRequest{
		RelPath: "docs/utf16.txt",
		Content: "\nmore",
	})
	if err != nil {
		t.Fatalf("ApplyFileAppend returned error: %v", err)
	}
	if proposal.Encoding != encodingUTF16LE {
		t.Fatalf("expected append to preserve utf-16le, got %q", proposal.Encoding)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	text, encoding, err := decodeText(content)
	if err != nil {
		t.Fatalf("decodeText failed: %v", err)
	}
	if encoding != encodingUTF16LE || text != "hello\nmore" {
		t.Fatalf("unexpected appended UTF-16 content: encoding=%s text=%q bytes=%#v", encoding, text, content)
	}
	if strings.Count(string(content), string([]byte{0xff, 0xfe})) != 1 {
		t.Fatalf("expected a single UTF-16LE BOM, got %#v", content)
	}
}

func TestPreviewFileAppendRejectsEncodingMismatch(t *testing.T) {
	root := t.TempDir()
	writeBytes(t, filepath.Join(root, "docs", "utf16.txt"), encodeUTF16("hello", binary.LittleEndian, []byte{0xff, 0xfe}))

	if _, err := New().PreviewFileAppend(root, FileWriteRequest{
		RelPath:  "docs/utf16.txt",
		Content:  "more",
		Encoding: "utf-8",
	}); err == nil {
		t.Fatal("expected append encoding mismatch to be rejected")
	}
}

func TestPreviewFileAppendSamplesTailForBinaryData(t *testing.T) {
	root := t.TempDir()
	content := append([]byte(strings.Repeat("a", 5000)), 0x00, 0x01, 0x02)
	writeBytes(t, filepath.Join(root, "docs", "mixed.txt"), content)

	if _, err := New().PreviewFileAppend(root, FileWriteRequest{RelPath: "docs/mixed.txt", Content: "text"}); err == nil {
		t.Fatal("expected binary tail sample to be rejected")
	}
}

func TestApplyFileWritePreservesRequestedUTF16LEEncoding(t *testing.T) {
	root := t.TempDir()

	proposal, err := New().ApplyFileWrite(root, FileWriteRequest{
		RelPath:  "docs/notes.txt",
		Content:  "hello\n",
		Encoding: "utf-16le",
	})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if proposal.Encoding != encodingUTF16LE {
		t.Fatalf("expected utf-16le proposal encoding, got %q", proposal.Encoding)
	}

	content, err := os.ReadFile(filepath.Join(root, "docs", "notes.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(content) < 4 || content[0] != 0xff || content[1] != 0xfe || content[2] != 'h' || content[3] != 0x00 {
		t.Fatalf("expected utf-16le content with BOM, got %#v", content)
	}
}

func TestApplyFileWritePreservesRequestedUTF16BEEncoding(t *testing.T) {
	root := t.TempDir()

	proposal, err := New().ApplyFileWrite(root, FileWriteRequest{
		RelPath:  "docs/notes.txt",
		Content:  "hello\n",
		Encoding: "utf-16be",
	})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if proposal.Encoding != encodingUTF16BE {
		t.Fatalf("expected utf-16-be proposal encoding, got %q", proposal.Encoding)
	}

	content, err := os.ReadFile(filepath.Join(root, "docs", "notes.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(content) < 4 || content[0] != 0xfe || content[1] != 0xff || content[2] != 0x00 || content[3] != 'h' {
		t.Fatalf("expected utf-16be content with BOM, got %#v", content)
	}
	text, encoding, err := decodeText(content)
	if err != nil {
		t.Fatalf("decodeText failed: %v", err)
	}
	if encoding != encodingUTF16BE || text != "hello\n" {
		t.Fatalf("unexpected UTF-16BE round trip: encoding=%s text=%q bytes=%#v", encoding, text, content)
	}
}

func TestApplyFileWritePreservesRequestedWindows1252Encoding(t *testing.T) {
	root := t.TempDir()

	proposal, err := New().ApplyFileWrite(root, FileWriteRequest{
		RelPath:  "docs/notes.txt",
		Content:  "café",
		Encoding: "windows-1252",
	})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if proposal.Encoding != encodingWindows1252 {
		t.Fatalf("expected windows-1252 proposal encoding, got %q", proposal.Encoding)
	}

	content, err := os.ReadFile(filepath.Join(root, "docs", "notes.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "caf\xe9" {
		t.Fatalf("expected windows-1252 bytes, got %#v", content)
	}
}

func TestApplyFileWritePreservesRequestedLatin1Encoding(t *testing.T) {
	root := t.TempDir()
	service := New()
	proposal, err := service.ApplyFileWrite(root, FileWriteRequest{
		RelPath:  "docs/notes.txt",
		Content:  "café",
		Encoding: "iso-8859-1",
	})
	if err != nil {
		t.Fatalf("ApplyFileWrite returned error: %v", err)
	}
	if proposal.Encoding != encodingLatin1 {
		t.Fatalf("expected iso-8859-1 proposal encoding, got %q", proposal.Encoding)
	}
	content, err := os.ReadFile(filepath.Join(root, "docs", "notes.txt"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != "caf\xe9" {
		t.Fatalf("expected Latin-1 bytes, got %#v", content)
	}
}

func TestPreviewFileWriteRejectsUnsafeTargets(t *testing.T) {
	root := t.TempDir()
	service := New()

	if _, err := service.PreviewFileWrite(root, FileWriteRequest{RelPath: "../outside.txt", Content: "x"}); err == nil {
		t.Fatal("expected traversal write to be rejected")
	}
	if _, err := service.PreviewFileWrite(root, FileWriteRequest{RelPath: ".nexusdesk/config.json", Content: "{}"}); err == nil {
		t.Fatal("expected metadata write to be rejected")
	}
	if _, err := service.PreviewFileWrite(root, FileWriteRequest{RelPath: "bad.txt", Content: "x", Encoding: "koi8-r"}); err == nil {
		t.Fatal("expected unsupported encoding to be rejected")
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(content) != want {
		t.Fatalf("expected %q, got %q", want, string(content))
	}
}

func countRollbackBlobFiles(t *testing.T, root string) int {
	t.Helper()
	blobRoot := filepath.Join(root, filepath.FromSlash(rollbackBlobDirRelPath))
	count := 0
	err := filepath.WalkDir(blobRoot, func(_ string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WalkDir rollback blobs failed: %v", err)
	}
	return count
}
