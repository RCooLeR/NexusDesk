package editor

import (
	"strings"
	"testing"
)

func TestFormatDocumentFormatsGo(t *testing.T) {
	result, err := FormatDocument("main.go", "package main\nfunc main(){println(\"hi\")}\n")
	if err != nil {
		t.Fatalf("FormatDocument returned error: %v", err)
	}
	if !result.Changed || !strings.Contains(result.Content, "func main()") {
		t.Fatalf("expected formatted Go content, got %#v", result)
	}
}

func TestFormatDocumentFormatsJSON(t *testing.T) {
	result, err := FormatDocument("settings.json", `{"name":"nexus","enabled":true}`)
	if err != nil {
		t.Fatalf("FormatDocument returned error: %v", err)
	}
	want := "{\n  \"name\": \"nexus\",\n  \"enabled\": true\n}\n"
	if !result.Changed || result.Content != want {
		t.Fatalf("unexpected JSON formatting:\n%s", result.Content)
	}
}

func TestFormatDocumentReportsUnsupportedExtensions(t *testing.T) {
	if _, err := FormatDocument("README.md", "# Title\n"); err == nil || !strings.Contains(err.Error(), "not available") {
		t.Fatalf("expected unsupported format error, got %v", err)
	}
}

func TestFormatDocumentReportsUnchanged(t *testing.T) {
	content := "{\n  \"name\": \"nexus\"\n}\n"
	result, err := FormatDocument("settings.json", content)
	if err != nil {
		t.Fatalf("FormatDocument returned error: %v", err)
	}
	if result.Changed || result.Content != content {
		t.Fatalf("expected unchanged formatted document, got %#v", result)
	}
}
