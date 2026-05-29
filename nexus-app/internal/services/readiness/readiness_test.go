package readiness

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	settingsSvc "nexusdesk/internal/services/settings"
	startupSvc "nexusdesk/internal/services/startup"
)

func TestCollectFlagsFirstRunActions(t *testing.T) {
	snapshot := Collect(Options{
		Settings:                settingsSvc.Defaults(),
		Now:                     time.Date(2026, 5, 28, 12, 0, 0, 0, time.UTC),
		GOOS:                    "windows",
		LookupPath:              missingPath,
		ExternalAgentLookupPath: missingPath,
		Stat:                    missingStat,
		MSYS2UCRT64Bin:          `C:\missing\ucrt64\bin`,
	})

	assertItemStatus(t, snapshot, "workspace", StatusAction)
	assertItemStatus(t, snapshot, "model", StatusAction)
	assertItemStatus(t, snapshot, "toolchain", StatusAction)
	assertItemStatus(t, snapshot, "external-agents", StatusAction)
	assertItemStatus(t, snapshot, "failure-scenarios", StatusOK)
	if snapshot.SettingsLoaded != true {
		t.Fatalf("expected default settings to be treated as loaded")
	}
	if snapshot.ModelConfigured {
		t.Fatalf("expected empty default model to require setup")
	}
}

func TestCollectReportsExternalAgentCLIs(t *testing.T) {
	snapshot := Collect(Options{
		Settings:   settingsSvc.Settings{Model: "qwen3:8b"},
		GOOS:       "linux",
		LookupPath: fixedPath("/usr/bin/gcc"),
		ExternalAgentLookupPath: func(name string) (string, error) {
			if name == "codex" {
				return "/usr/local/bin/codex", nil
			}
			return "", os.ErrNotExist
		},
	})

	assertItemStatus(t, snapshot, "external-agents", StatusWarning)
	if !strings.Contains(FormatMarkdown(snapshot), "Codex CLI") {
		t.Fatalf("expected Codex CLI in readiness markdown")
	}
}

func TestFormatMarkdownIncludesFailureScenarioMatrix(t *testing.T) {
	snapshot := Collect(Options{
		Settings:   settingsSvc.Settings{Model: "qwen3-coder:30b"},
		GOOS:       "linux",
		LookupPath: fixedPath("/usr/bin/gcc"),
	})
	text := FormatMarkdown(snapshot)
	for _, expected := range []string{
		"Production failure gates",
		"Production failure scenarios",
		"folder-open-cheap",
		"canceled-long-work",
		"docs/20_CLEAN_MACHINE_SMOKE_CHECKLIST.md",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected %q in readiness markdown:\n%s", expected, text)
		}
	}
}

func TestCollectAcceptsWindowsGCCOnPath(t *testing.T) {
	snapshot := Collect(Options{
		WorkspaceRoot: "C:/work/project",
		WorkspaceName: "project",
		Settings: settingsSvc.Settings{
			Provider: "ollama",
			Protocol: settingsSvc.ProtocolOllamaOpenAICompatible,
			BaseURL:  "http://localhost:11434/v1",
			Model:    "qwen3:8b",
		},
		GOOS: "windows",
		LookupPath: func(name string) (string, error) {
			if name == "gcc.exe" {
				return `C:\msys64\ucrt64\bin\gcc.exe`, nil
			}
			return "", os.ErrNotExist
		},
	})

	assertItemStatus(t, snapshot, "workspace", StatusOK)
	assertItemStatus(t, snapshot, "model", StatusOK)
	assertItemStatus(t, snapshot, "toolchain", StatusOK)
	if snapshot.Toolchain.GCCPath == "" {
		t.Fatalf("expected gcc path in toolchain snapshot")
	}
}

func TestCollectRequiresAPIKeyForCustomProvider(t *testing.T) {
	snapshot := Collect(Options{
		Settings: settingsSvc.Settings{
			Provider: "custom-openai-compatible",
			Protocol: settingsSvc.ProtocolOpenAICompatible,
			BaseURL:  "https://api.openai.com/v1",
			Model:    "gpt-4.1",
		},
		GOOS:       "linux",
		LookupPath: fixedPath("/usr/bin/gcc"),
	})
	assertItemStatus(t, snapshot, "credentials", StatusAction)

	snapshot = Collect(Options{
		Settings: settingsSvc.Settings{
			Provider: "custom-openai-compatible",
			Protocol: settingsSvc.ProtocolOpenAICompatible,
			BaseURL:  "https://api.openai.com/v1",
			Model:    "gpt-4.1",
			APIKey:   settingsSvc.RedactedAPIKey,
		},
		GOOS:       "linux",
		LookupPath: fixedPath("/usr/bin/gcc"),
	})
	assertItemStatus(t, snapshot, "credentials", StatusOK)
	if !strings.Contains(FormatMarkdown(snapshot), "First-run readiness") {
		t.Fatalf("expected formatted markdown to include readiness title")
	}
}

func TestCollectWarnsForUnknownProvider(t *testing.T) {
	snapshot := Collect(Options{
		Settings: settingsSvc.Settings{
			Provider: "legacy-provider",
			Protocol: settingsSvc.ProtocolOpenAICompatible,
			BaseURL:  "http://localhost:9999/v1",
			Model:    "legacy-model",
		},
		GOOS:       "linux",
		LookupPath: fixedPath("/usr/bin/gcc"),
	})

	assertItemStatus(t, snapshot, "settings", StatusWarning)
}

func TestCollectWarnsWhenCGODisabled(t *testing.T) {
	snapshot := Collect(Options{
		Settings: settingsSvc.Settings{Model: "qwen3:8b"},
		GOOS:     "linux",
		Getenv: func(name string) string {
			if name == "CGO_ENABLED" {
				return "0"
			}
			return ""
		},
		LookupPath: fixedPath("/usr/bin/gcc"),
	})

	assertItemStatus(t, snapshot, "toolchain", StatusWarning)
	if !strings.Contains(snapshot.Toolchain.Detail, "CGO_ENABLED=0") {
		t.Fatalf("expected CGO warning, got %q", snapshot.Toolchain.Detail)
	}
}

func TestCollectWarnsAboutPreviousUncleanStartup(t *testing.T) {
	snapshot := Collect(Options{
		Settings:   settingsSvc.Settings{Model: "qwen3:8b"},
		GOOS:       "linux",
		LookupPath: fixedPath("/usr/bin/gcc"),
		StartupRecovery: startupSvc.Status{
			PreviousUnclean: true,
			Message:         "Previous NexusDesk run did not record a clean exit.",
		},
	})
	assertItemStatus(t, snapshot, "startup", StatusWarning)
	if !strings.Contains(FormatMarkdown(snapshot), "Startup recovery") {
		t.Fatalf("expected startup recovery in readiness markdown")
	}
}

func assertItemStatus(t *testing.T, snapshot Snapshot, id string, status string) {
	t.Helper()
	for _, item := range snapshot.Items {
		if item.ID == id {
			if item.Status != status {
				t.Fatalf("expected %s status %s, got %s (%s)", id, status, item.Status, item.Detail)
			}
			return
		}
	}
	t.Fatalf("missing readiness item %q", id)
}

func missingPath(string) (string, error) {
	return "", os.ErrNotExist
}

func fixedPath(path string) func(string) (string, error) {
	return func(string) (string, error) {
		return path, nil
	}
}

func missingStat(string) (os.FileInfo, error) {
	return nil, errors.New("missing")
}
