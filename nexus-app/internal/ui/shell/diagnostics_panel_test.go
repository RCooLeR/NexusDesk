package shell

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fynetest "fyne.io/fyne/v2/test"

	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
	perfSvc "nexusdesk/internal/services/perf"
	settingsSvc "nexusdesk/internal/services/settings"
	startupSvc "nexusdesk/internal/services/startup"
)

func TestDiagnosticsStatusLineReflectsProbeAndMetadataState(t *testing.T) {
	now := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	snapshot := diagnosticsSnapshot{
		CollectedAt:         now,
		ProbeResult:         &llmSvc.ProbeResult{OK: true},
		MetadataStatus:      &metadataSvc.Status{Path: "meta.sqlite"},
		InMemoryJobs:        4,
		InMemoryRunningJobs: 1,
		InMemoryFailedJobs:  2,
	}
	line := diagnosticsStatusLine(snapshot)
	for _, part := range []string{"provider ok", "metadata ok", "jobs 4 running 1 failed 2"} {
		if !strings.Contains(line, part) {
			t.Fatalf("expected %q in %q", part, line)
		}
	}
}

func TestFormatDiagnosticsSnapshotIncludesCoreSections(t *testing.T) {
	snapshot := diagnosticsSnapshot{
		CollectedAt:   time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC),
		WorkspaceRoot: "E:/workspace",
		Settings: settingsSvc.Settings{
			Provider: "ollama",
			BaseURL:  "http://localhost:11434/v1",
			Model:    "qwen2.5-coder:14b",
		},
		ProbeResult: &llmSvc.ProbeResult{
			OK:         true,
			Message:    "Connected to provider.",
			Endpoint:   "http://localhost:11434/v1/models",
			ModelCount: 3,
		},
		RuntimeSummary: []string{
			"Provider: ollama",
			"Selected model: qwen2.5-coder:14b (loaded=true, vram=4.00 GiB)",
		},
		MetadataStatus: &metadataSvc.Status{
			Path:    "E:/workspace/.nexusdesk/metadata/nexusdesk.sqlite",
			Tables:  []string{"jobs", "task_runs"},
			Message: "SQLite metadata store is active.",
		},
		InMemoryJobs:            2,
		InMemoryRunningJobs:     1,
		InMemoryFailedJobs:      0,
		RecentPersistedJobs:     5,
		RecentPersistedFailures: 1,
		RecentTaskRuns:          8,
		RecentTaskFailures:      2,
		RecentSQLRuns:           6,
		RecentSQLFailures:       1,
		RecentAgentRuns:         3,
		RecentAgentFailures:     1,
		RecentArtifacts:         4,
		RecentJobFailures:       []string{"job-9 [failed] Build docs: process exited 2"},
		RecentTaskFailuresList:  []string{"task-3 [failed] go test exit 1: tests failed"},
		RecentSQLFailuresList:   []string{"sql-2 [sqlite failed] sample.db: syntax error"},
		RecentAgentFailuresList: []string{"agent-5 [failed] iter 8 stop max_iterations: loop guard triggered"},
		ActivityTail:            []string{"Opened workspace E:/workspace", "Ran read-only SQLite query for sample.db"},
		ProviderGuidance:        []string{"For Ollama, start the runtime with \"ollama serve\" and verify installed models with \"ollama list\"."},
		StartupRecovery: startupSvc.Status{
			Path: "C:/Users/example/AppData/Roaming/NexusDesk/startup-session.json",
		},
		PerformanceTimings: []perfSvc.TimingRecord{
			{
				Name:         perfSvc.TimingStartupReady,
				Duration:     600 * time.Millisecond,
				Budget:       perfSvc.StartupReadyBudget,
				Detail:       "native shell content is ready",
				WithinBudget: true,
			},
		},
		RecommendedActions: []string{
			"Open Jobs and Agent Audit tabs to inspect recent failures and retry safe workloads.",
		},
	}
	text := formatDiagnosticsSnapshot(snapshot)
	for _, expected := range []string{
		"# Diagnostics",
		"## Provider",
		"Probe: ok",
		"## Provider Runtime",
		"## Provider Guidance",
		"ollama serve",
		"## Startup Recovery",
		"Status: ok - clean-exit markers are active.",
		"## Performance Timings",
		"startup-ready: 600ms",
		"## Metadata",
		"Status: ok",
		"## Jobs",
		"Task runs (recent): 8 total, 2 non-success",
		"## Recommended Actions",
		"1. Open Jobs and Agent Audit tabs to inspect recent failures and retry safe workloads.",
		"## Recent Failure Triage",
		"Jobs:",
		"## App Log Tail",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected %q in diagnostics text:\n%s", expected, text)
		}
	}
}

func TestIsSuccessStatus(t *testing.T) {
	if !isSuccessStatus(" success ") {
		t.Fatal("expected trimmed success status to pass")
	}
	if isSuccessStatus("failed") {
		t.Fatal("expected failed status to be non-success")
	}
}

func TestDiagnosticsRuntimeSummaryIncludesLoadedModels(t *testing.T) {
	summary := diagnosticsRuntimeSummary(llmSvc.RuntimeStatus{
		Provider:            "ollama",
		Endpoint:            "http://localhost:11434/api/ps",
		Message:             "Runtime is available.",
		SelectedModel:       "qwen2.5-coder:14b",
		SelectedModelLoaded: true,
		SelectedModelVRAM:   2 * 1024 * 1024 * 1024,
		LoadedModels: []llmSvc.RuntimeModel{
			{Name: "qwen2.5-coder:14b", ContextLength: 32768, Size: 8 * 1024 * 1024 * 1024, SizeVRAM: 2 * 1024 * 1024 * 1024},
		},
	})
	joined := strings.Join(summary, "\n")
	for _, expected := range []string{
		"Provider: ollama",
		"Selected model: qwen2.5-coder:14b (loaded=true",
		"Loaded models: 1",
		"qwen2.5-coder:14b | ctx 32768",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in runtime summary:\n%s", expected, joined)
		}
	}
}

func TestDiagnosticsRecommendedActionsCoversProviderMetadataAndFailures(t *testing.T) {
	actions := diagnosticsRecommendedActions(diagnosticsSnapshot{
		ProbeError:              "dial tcp 127.0.0.1:11434: connect: connection refused",
		MetadataError:           "no such table: jobs",
		InMemoryFailedJobs:      1,
		RecentPersistedFailures: 2,
		RecentTaskFailures:      1,
		ActivityTail:            nil,
	})
	joined := strings.Join(actions, "\n")
	for _, expected := range []string{
		"Open Settings and verify provider base URL, credentials, and selected model.",
		"Inspect metadata health and recover .nexusdesk/metadata before continuing long runs.",
		"Open Jobs and Agent Audit tabs to inspect recent failures and retry safe workloads.",
		"Trigger one small task and rerun diagnostics to populate runtime activity context.",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in actions:\n%s", expected, joined)
		}
	}
}

func TestDiagnosticsRecommendedActionsIncludesStartupRecovery(t *testing.T) {
	actions := diagnosticsRecommendedActions(diagnosticsSnapshot{
		StartupRecovery: startupSvc.Status{
			PreviousUnclean: true,
			Message:         "Previous NexusDesk run did not record a clean exit.",
		},
	})
	joined := strings.Join(actions, "\n")
	if !strings.Contains(joined, "Review Startup Recovery") {
		t.Fatalf("expected startup recovery action, got:\n%s", joined)
	}
}

func TestDiagnosticsRecommendedActionsIncludesSlowPerformanceTiming(t *testing.T) {
	actions := diagnosticsRecommendedActions(diagnosticsSnapshot{
		PerformanceTimings: []perfSvc.TimingRecord{
			{
				Name:         perfSvc.TimingWorkspaceOpen,
				Duration:     3 * time.Second,
				Budget:       perfSvc.WorkspaceOpenBudget,
				WithinBudget: false,
			},
		},
	})
	joined := strings.Join(actions, "\n")
	if !strings.Contains(joined, "Review Performance Timings") {
		t.Fatalf("expected performance action, got:\n%s", joined)
	}
}

func TestCollectDiagnosticsSnapshotIncludesPerformanceTimings(t *testing.T) {
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("diagnostics-performance")
	defer window.Close()
	view := New(window)
	view.metadataStore = nil
	started := time.Now().Add(-3 * time.Second)
	view.recordPerformanceTiming(perfSvc.TimingWorkspaceOpen, started, time.Millisecond, "opened large workspace")

	snapshot := view.collectDiagnosticsSnapshot("E:/workspace", nil)

	if len(snapshot.PerformanceTimings) != 1 {
		t.Fatalf("expected performance timing in snapshot, got %#v", snapshot.PerformanceTimings)
	}
	joinedWarnings := strings.Join(snapshot.Warnings, "\n")
	if !strings.Contains(joinedWarnings, "Performance timing over budget") {
		t.Fatalf("expected performance warning in warnings: %v", snapshot.Warnings)
	}
}

func TestCollectDiagnosticsSnapshotIncludesProbeErrorWarning(t *testing.T) {
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("diagnostics-probe-error")
	defer window.Close()
	view := New(window)
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	view.settingsStore = settingsSvc.NewFileStore(settingsPath)
	if err := view.settingsStore.Save(settingsSvc.Settings{
		Provider:              "ollama",
		BaseURL:               "http://localhost:11434/v1",
		Model:                 "qwen2.5-coder:14b",
		ContextTokens:         32768,
		ResponseReserveTokens: 4096,
	}); err != nil {
		t.Fatalf("Save settings failed: %v", err)
	}
	view.diagnosticsProber = diagnosticsProbeStub{
		err: errors.New("dial tcp 127.0.0.1:11434: connect: connection refused"),
	}

	snapshot := view.collectDiagnosticsSnapshot("E:/workspace", nil)

	if !strings.Contains(snapshot.ProbeError, "connection refused") {
		t.Fatalf("expected probe error to be captured, got %q", snapshot.ProbeError)
	}
	joinedWarnings := strings.Join(snapshot.Warnings, "\n")
	if !strings.Contains(joinedWarnings, "Provider probe failed:") {
		t.Fatalf("expected probe warning in warnings: %v", snapshot.Warnings)
	}
	joinedActions := strings.Join(snapshot.RecommendedActions, "\n")
	if !strings.Contains(joinedActions, "Open Settings and verify provider base URL, credentials, and selected model.") {
		t.Fatalf("expected provider remediation action, got %v", snapshot.RecommendedActions)
	}
	if !strings.Contains(joinedActions, "ollama serve") {
		t.Fatalf("expected Ollama-specific remediation action, got %v", snapshot.RecommendedActions)
	}
}

func TestCollectDiagnosticsSnapshotMarksNonOKProbeAsWarning(t *testing.T) {
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("diagnostics-probe-warning")
	defer window.Close()
	view := New(window)
	settingsPath := filepath.Join(t.TempDir(), "settings.json")
	view.settingsStore = settingsSvc.NewFileStore(settingsPath)
	if err := view.settingsStore.Save(settingsSvc.Settings{
		Provider:              "openai-compatible",
		BaseURL:               "http://localhost:11434/v1",
		Model:                 "qwen2.5-coder:14b",
		ContextTokens:         32768,
		ResponseReserveTokens: 4096,
	}); err != nil {
		t.Fatalf("Save settings failed: %v", err)
	}
	view.diagnosticsProber = diagnosticsProbeStub{
		result: llmSvc.ProbeResult{
			OK:         false,
			Message:    "Provider returned HTTP 401",
			Endpoint:   "http://localhost:11434/v1/models",
			ModelCount: 0,
		},
	}

	snapshot := view.collectDiagnosticsSnapshot("E:/workspace", nil)

	if snapshot.ProbeResult == nil || snapshot.ProbeResult.OK {
		t.Fatalf("expected non-OK probe result to be captured: %#v", snapshot.ProbeResult)
	}
	joinedWarnings := strings.Join(snapshot.Warnings, "\n")
	if !strings.Contains(joinedWarnings, "Provider probe returned non-OK status.") {
		t.Fatalf("expected non-OK warning in warnings: %v", snapshot.Warnings)
	}
	joinedActions := strings.Join(snapshot.RecommendedActions, "\n")
	if !strings.Contains(joinedActions, "Run provider probe again after checking model availability and endpoint health.") {
		t.Fatalf("expected rerun probe action, got %v", snapshot.RecommendedActions)
	}
	if !strings.Contains(joinedActions, "API key") {
		t.Fatalf("expected provider-specific auth guidance, got %v", snapshot.RecommendedActions)
	}
}

type diagnosticsProbeStub struct {
	result llmSvc.ProbeResult
	err    error
}

func (stub diagnosticsProbeStub) Probe(context.Context, llmSvc.Config) (llmSvc.ProbeResult, error) {
	if stub.err != nil {
		return llmSvc.ProbeResult{}, stub.err
	}
	return stub.result, nil
}
