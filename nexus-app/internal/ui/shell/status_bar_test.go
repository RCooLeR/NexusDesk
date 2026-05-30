package shell

import (
	"strings"
	"testing"
	"time"

	"nexusdesk/internal/buildinfo"
	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
	gitSvc "nexusdesk/internal/services/git"
	jobsSvc "nexusdesk/internal/services/jobs"
	settingsSvc "nexusdesk/internal/services/settings"
)

func TestStatusBarTextSummarizesWorkbenchHealth(t *testing.T) {
	text := statusBarText(statusBarSnapshot{
		Workspace: domain.Workspace{
			Name: "NexusDesk",
			Root: "E:/repo",
			Summary: domain.ScanSummary{
				Included:   42,
				Ignored:    3,
				Unreadable: 2,
			},
		},
		Settings: settingsSvc.Settings{
			Provider: "ollama",
			Model:    "qwen3-coder:30b",
		},
		GitStatus: gitSvc.Status{
			Available:   true,
			Branch:      "main",
			AheadBehind: "ahead 1",
		},
		SelectedPath: "internal/ui/shell/status_bar.go",
		SaveState:    "modified",
		Encoding:     "utf-8",
		LineEnding:   "LF",
		Jobs: []jobsSvc.Job{
			{ID: "running", Status: jobsSvc.StatusRunning, StartedAt: time.Now()},
			{ID: "failed", Status: jobsSvc.StatusFailed, StartedAt: time.Now()},
			{ID: "timeout", Status: jobsSvc.StatusTimedOut, StartedAt: time.Now()},
		},
		BuildInfo: buildinfo.Info{Version: "1.2.3"},
	})

	for _, expected := range []string{
		"Workspace: NexusDesk (42 indexed, 3 ignored, 2 unreadable)",
		"Provider: ollama/qwen3-coder:30b",
		"Branch: main ahead 1",
		"Jobs: 1 running, 2 failed",
		"Warnings: 4",
		"Selected: internal/ui/shell/status_bar.go",
		"Save: modified",
		"Encoding: utf-8",
		"Line: LF",
		"Version: 1.2.3",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected status bar text to contain %q, got %q", expected, text)
		}
	}
}

func TestStatusBarTextFallsBackForColdStart(t *testing.T) {
	text := statusBarText(statusBarSnapshot{
		Settings: settingsSvc.Settings{Provider: "ollama"},
	})

	for _, expected := range []string{
		"Workspace: none",
		"Provider: ollama/model not selected",
		"Branch: not loaded",
		"Jobs: 0 running, 0 failed",
		"Warnings: 1",
		"Selected: none",
		"Save: n/a",
		"Encoding: n/a",
		"Line: n/a",
		"Version: dev",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected cold-start status to contain %q, got %q", expected, text)
		}
	}
}

func TestEditorSaveStateText(t *testing.T) {
	if got := editorSaveStateText(editorSvc.Tab{Dirty: true}, &textEditorBinding{encodingExplicit: true}); got != "modified" {
		t.Fatalf("expected modified save state, got %q", got)
	}
	if got := editorSaveStateText(editorSvc.Tab{}, &textEditorBinding{saving: true, encodingExplicit: true}); got != "saving" {
		t.Fatalf("expected saving state, got %q", got)
	}
	if got := editorSaveStateText(editorSvc.Tab{}, &textEditorBinding{sourceEncoding: "utf-8", saveEncoding: "utf-16le", encodingExplicit: true}); got != "encoding changed" {
		t.Fatalf("expected encoding changed state, got %q", got)
	}
	if got := editorSaveStateText(editorSvc.Tab{}, &textEditorBinding{encodingExplicit: false}); got != "encoding required" {
		t.Fatalf("expected encoding required state, got %q", got)
	}
}

func TestDetectLineEnding(t *testing.T) {
	cases := map[string]string{
		"":             "n/a",
		"one\n":        "LF",
		"one\r\n":      "CRLF",
		"one\r":        "CR",
		"one\r\ntwo\n": "mixed",
	}
	for input, want := range cases {
		if got := detectLineEnding(input); got != want {
			t.Fatalf("detectLineEnding(%q) = %q, want %q", input, got, want)
		}
	}
}
