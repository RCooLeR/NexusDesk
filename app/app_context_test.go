package main

import (
	"strings"
	"testing"

	"NexusDesk/internal/workspace"
)

func TestBuildChatContextContentUsesCSVSummary(t *testing.T) {
	content := buildChatContextContent(workspace.FilePreview{
		Name:    "report.csv",
		Kind:    "file",
		Content: "raw,csv\nalpha,10\n",
		Table: &workspace.TablePreview{
			Columns: []string{"name", "value"},
			Rows: [][]string{
				{"alpha", "10"},
				{"beta, quoted", "20"},
			},
			Profiles: []workspace.ColumnProfile{
				{Name: "name", Type: "text", Missing: 0, Distinct: 2},
				{Name: "value", Type: "integer", Missing: 0, Distinct: 2, Min: "10", Max: "20"},
			},
		},
	})

	if !strings.Contains(content, "CSV context summary") {
		t.Fatalf("expected CSV context summary, got %q", content)
	}
	if !strings.Contains(content, "value: integer, distinct=2, missing=0, range=10..20") {
		t.Fatalf("expected numeric profile, got %q", content)
	}
	if !strings.Contains(content, "\"beta, quoted\",20") {
		t.Fatalf("expected CSV sample rows to stay escaped, got %q", content)
	}
}

func TestBuildChatContextContentKeepsTextContent(t *testing.T) {
	content := buildChatContextContent(workspace.FilePreview{
		Name:    "notes.md",
		Kind:    "file",
		Content: "plain text",
	})

	if content != "plain text" {
		t.Fatalf("expected plain text context, got %q", content)
	}
}
