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

func TestFormatDocumentFormatsMarkdownWhitespace(t *testing.T) {
	result, err := FormatDocument("README.md", "# Title\r\nKeep hard break   \r\nTrim tab\t\r\n\r\n")
	if err != nil {
		t.Fatalf("FormatDocument returned error: %v", err)
	}
	want := "# Title\nKeep hard break  \nTrim tab\n"
	if !result.Changed || result.Content != want {
		t.Fatalf("unexpected Markdown formatting:\n%q", result.Content)
	}
}

func TestFormatDocumentFormatsConfigWhitespace(t *testing.T) {
	result, err := FormatDocument("compose.yaml", "services:  \r\n  app:\t\r\n    image: nexus  \r\n\r\n")
	if err != nil {
		t.Fatalf("FormatDocument returned error: %v", err)
	}
	want := "services:\n  app:\n    image: nexus\n"
	if !result.Changed || result.Content != want {
		t.Fatalf("unexpected YAML formatting:\n%q", result.Content)
	}
}

func TestFormatDocumentFormatsDockerfileByName(t *testing.T) {
	result, err := FormatDocument("Dockerfile.dev", "FROM alpine  \r\nRUN echo hi\t\r\n")
	if err != nil {
		t.Fatalf("FormatDocument returned error: %v", err)
	}
	want := "FROM alpine\nRUN echo hi\n"
	if !result.Changed || result.Content != want {
		t.Fatalf("unexpected Dockerfile formatting:\n%q", result.Content)
	}
}

func TestFormatDocumentReportsUnsupportedExtensions(t *testing.T) {
	if _, err := FormatDocument("script.py", "print('hi')\n"); err == nil || !strings.Contains(err.Error(), "not available") {
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
