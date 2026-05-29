package shell

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	fynetest "fyne.io/fyne/v2/test"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	dbconnectorSvc "nexusdesk/internal/services/dbconnector"
	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
	perfSvc "nexusdesk/internal/services/perf"
	protectedsecretSvc "nexusdesk/internal/services/protectedsecret"
	settingsSvc "nexusdesk/internal/services/settings"
	startupSvc "nexusdesk/internal/services/startup"
	toolsSvc "nexusdesk/internal/services/tools"
)

func TestDiagnosticsControllerInitialState(t *testing.T) {
	_ = fynetest.NewTempApp(t)

	view := &View{}
	controller := newDiagnosticsController(view)

	if controller.status.Text != "Open a workspace to run diagnostics." {
		t.Fatalf("expected initial diagnostics status, got %q", controller.status.Text)
	}
	if strings.TrimSpace(controller.detail.Text) != "" {
		t.Fatalf("expected empty initial diagnostics detail, got %q", controller.detail.Text)
	}
}

func TestDiagnosticsControllerRequiresWorkspace(t *testing.T) {
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("diagnostics-controller")
	defer window.Close()
	view := New(window)

	view.diagnostics.Refresh()

	if view.diagnostics.status.Text != "Open a workspace before running diagnostics." {
		t.Fatalf("expected missing workspace status, got %q", view.diagnostics.status.Text)
	}
	if view.diagnostics.detail.Text != "Diagnostics are scoped to an open workspace." {
		t.Fatalf("expected missing workspace detail, got %q", view.diagnostics.detail.Text)
	}
}

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
			Path:          "E:/workspace/.nexusdesk/metadata/nexusdesk.sqlite",
			JournalMode:   "wal",
			ForeignKeys:   true,
			BusyTimeoutMS: 5000,
			Tables:        []string{"jobs", "task_runs"},
			Message:       "SQLite metadata store is active.",
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
		ConnectorProfiles: []dbconnectorSvc.ConnectorProfile{
			{Name: "Warehouse", Kind: "postgres", SSLMode: "require"},
		},
		ConnectorPools: []dbconnectorSvc.ConnectorPoolStatus{
			{Name: "Warehouse", ProfileID: "warehouse", Kind: "postgres", Driver: "pgx", MaxOpenConnections: 4, OpenConnections: 1, Idle: 1},
		},
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
		"## Health Cards",
		"**[OK] Provider:** Connected to provider.",
		"**[OK] Metadata:** 2 table(s). journal=wal foreign_keys=true busy_timeout=5000ms. SQLite metadata store is active.",
		"**[ACTION] Jobs and runs:** 5 non-success item(s)",
		"**[OK] Issue report:** Redacted diagnostics export is available",
		"Probe: ok",
		"## Provider Runtime",
		"## Provider Guidance",
		"ollama serve",
		"## Startup Recovery",
		"Status: ok - clean-exit markers are active.",
		"## Performance Timings",
		"startup-ready: 600ms",
		"## Production Failure Gates",
		"folder-open-cheap",
		"## Agent Tool Registry",
		"planned tools are roadmap-only",
		"## Artifact Provenance",
		"No native artifacts are present yet.",
		"## Protected Secrets",
		"## Connector Transport",
		"Warehouse [postgres]: encrypted transport required",
		"## Connector Pools",
		"Warehouse [postgres/pgx]: open=1",
		"## Metadata",
		"Status: ok",
		"Journal mode: wal",
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

func TestFormatDiagnosticsConnectorPools(t *testing.T) {
	empty := formatDiagnosticsConnectorPools(nil)
	if !strings.Contains(empty, "No external connector pools") {
		t.Fatalf("unexpected empty connector pool text: %q", empty)
	}
	text := formatDiagnosticsConnectorPools([]dbconnectorSvc.ConnectorPoolStatus{{
		Name:               "Warehouse",
		ProfileID:          "warehouse",
		Kind:               "postgres",
		Driver:             "pgx",
		MaxOpenConnections: 4,
		OpenConnections:    2,
		InUse:              1,
		Idle:               1,
	}})
	for _, expected := range []string{"1 external connector pool(s) open", "Warehouse [postgres/pgx]: open=2 in_use=1 idle=1 max=4"} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected %q in connector pool diagnostics:\n%s", expected, text)
		}
	}
}

func TestDiagnosticsHealthCardsSummarizeActionsAndWarnings(t *testing.T) {
	cards := diagnosticsHealthCards(diagnosticsSnapshot{
		ProbeError:              "connection refused",
		MetadataError:           "no such table: jobs",
		InMemoryFailedJobs:      1,
		RecentPersistedFailures: 2,
		RecentTaskFailures:      1,
		InMemoryRunningJobs:     1,
		StartupRecovery: startupSvc.Status{
			PreviousUnclean: true,
			Message:         "Previous run did not record a clean exit.",
		},
		PerformanceTimings: []perfSvc.TimingRecord{{
			Name:         perfSvc.TimingWorkspaceOpen,
			Duration:     3 * time.Second,
			Budget:       perfSvc.WorkspaceOpenBudget,
			WithinBudget: false,
		}},
	})
	joined := diagnosticsHealthCardText(cards)
	for _, expected := range []string{
		"Provider|action|Probe failed: connection refused|Verify base URL",
		"Metadata|action|no such table: jobs|Export metadata backup",
		"Jobs and runs|action|4 non-success item(s)",
		"Performance|warning|At least one startup or folder-open timing is over budget.",
		"Production failure gates|ok|5 scenario(s) cover crash/hang/provider/metadata/cancel release gates.",
		"Agent tool registry|ok|",
		"planned tools are not executable",
		"Artifact provenance|ok|No native artifacts are present yet.",
		"Protected secrets|warning|OS protected secret storage is unavailable.|Fix protected credential storage",
		"Connector transport|ok|No external connector profiles are configured.",
		"Startup recovery|warning|Previous run did not record a clean exit.",
		"Issue report|ok|Redacted diagnostics export is available",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected health cards to contain %q, got:\n%s", expected, joined)
		}
	}
}

func TestDiagnosticsProtectedSecretHealthCard(t *testing.T) {
	healthy := diagnosticsHealthCardText(diagnosticsHealthCards(diagnosticsSnapshot{
		ProtectedSecretStatus: protectedsecretSvc.BackendStatus{
			Backend:   "Windows DPAPI",
			Available: true,
			Message:   "Windows DPAPI protected storage is available.",
		},
	}))
	if !strings.Contains(healthy, "Protected secrets|ok|Windows DPAPI protected storage is available.") {
		t.Fatalf("expected protected secret OK card, got:\n%s", healthy)
	}

	warning := diagnosticsHealthCardText(diagnosticsHealthCards(diagnosticsSnapshot{
		ProtectedSecretStatus: protectedsecretSvc.BackendStatus{
			Backend:   "Linux Secret Service",
			Available: false,
			Message:   "Linux Secret Service protected storage is unavailable because secret-tool was not found in PATH.",
			Action:    "Install libsecret secret-tool.",
		},
	}))
	for _, expected := range []string{
		"Protected secrets|warning|Linux Secret Service protected storage is unavailable",
		"Install libsecret secret-tool.",
	} {
		if !strings.Contains(warning, expected) {
			t.Fatalf("expected protected secret warning %q, got:\n%s", expected, warning)
		}
	}

	report := formatDiagnosticsSnapshot(diagnosticsSnapshot{
		ProtectedSecretStatus: protectedsecretSvc.BackendStatus{
			Backend:   "Linux Secret Service",
			Available: false,
			Message:   "Linux Secret Service protected storage is unavailable because secret-tool was not found in PATH.",
			Action:    "Install libsecret secret-tool.",
		},
	})
	for _, expected := range []string{
		"## Protected Secrets",
		"Backend: Linux Secret Service",
		"Status: warning - Linux Secret Service protected storage is unavailable",
		"Next: Install libsecret secret-tool.",
	} {
		if !strings.Contains(report, expected) {
			t.Fatalf("expected %q in diagnostics report:\n%s", expected, report)
		}
	}
}

func TestDiagnosticsConnectorTransportHealthCard(t *testing.T) {
	cards := diagnosticsHealthCards(diagnosticsSnapshot{
		ConnectorProfiles: []dbconnectorSvc.ConnectorProfile{
			{Name: "Warehouse", Kind: "postgres", SSLMode: "require"},
			{Name: "Local cache", Kind: "sqlite"},
		},
	})
	joined := diagnosticsHealthCardText(cards)
	if !strings.Contains(joined, "Connector transport|ok|2 profile(s) checked: 1 encrypted, 1 local, 0 plaintext.") {
		t.Fatalf("expected connector transport OK card, got:\n%s", joined)
	}

	warningCards := diagnosticsHealthCards(diagnosticsSnapshot{
		ConnectorProfiles: []dbconnectorSvc.ConnectorProfile{
			{Name: "Dev MySQL", Kind: "mysql", SSLMode: dbconnectorSvc.ConnectorSSLModeDevelopmentPlaintext},
		},
	})
	warning := diagnosticsHealthCardText(warningCards)
	for _, expected := range []string{
		"Connector transport|warning|1 profile(s) use development plaintext transport",
		"switch production connector profiles to encrypted transport",
	} {
		if !strings.Contains(warning, expected) {
			t.Fatalf("expected connector transport warning %q, got:\n%s", expected, warning)
		}
	}

	report := formatDiagnosticsConnectorTransport(diagnosticsSnapshot{
		ConnectorProfiles: []dbconnectorSvc.ConnectorProfile{
			{Name: "Dev MySQL", Kind: "mysql", SSLMode: dbconnectorSvc.ConnectorSSLModeDevelopmentPlaintext},
		},
	})
	for _, expected := range []string{
		"Status: warning - 1 profile(s), 0 encrypted, 1 plaintext",
		"Dev MySQL [mysql]: development plaintext, encryption disabled",
	} {
		if !strings.Contains(report, expected) {
			t.Fatalf("expected %q in connector transport report:\n%s", expected, report)
		}
	}
}

func TestDiagnosticsHealthCardsHealthySnapshot(t *testing.T) {
	cards := diagnosticsHealthCards(diagnosticsSnapshot{
		ProbeResult: &llmSvc.ProbeResult{OK: true, Message: "Connected."},
		MetadataStatus: &metadataSvc.Status{
			JournalMode:   "wal",
			ForeignKeys:   true,
			BusyTimeoutMS: 5000,
			Tables:        []string{"jobs"},
			Message:       "SQLite metadata store is active.",
		},
		PerformanceTimings: []perfSvc.TimingRecord{{
			Name:         perfSvc.TimingWorkspaceOpen,
			Duration:     time.Second,
			Budget:       perfSvc.WorkspaceOpenBudget,
			WithinBudget: true,
		}},
	})
	joined := diagnosticsHealthCardText(cards)
	for _, expected := range []string{
		"Provider|ok|Connected.|",
		"Metadata|ok|1 table(s). journal=wal foreign_keys=true busy_timeout=5000ms. SQLite metadata store is active.|",
		"Jobs and runs|ok|0 recent/in-memory",
		"Performance|ok|1 timing record(s) captured and within budget.",
		"Production failure gates|ok|5 scenario(s) cover crash/hang/provider/metadata/cancel release gates.",
		"Agent tool registry|ok|",
		"Artifact provenance|ok|No native artifacts are present yet.",
		"Startup recovery|ok|Clean-exit markers are active.",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected healthy cards to contain %q, got:\n%s", expected, joined)
		}
	}
}

func TestDiagnosticsArtifactProvenanceHealthCardWarnsOnMissingLineage(t *testing.T) {
	cards := diagnosticsHealthCards(diagnosticsSnapshot{
		ArtifactProvenance: artifactsSvc.ProvenanceSummary{
			ArtifactCount:   2,
			WithMetadata:    2,
			WithLineage:     1,
			MissingLineage:  1,
			MissingMetadata: 0,
			Issues: []artifactsSvc.ProvenanceIssue{{
				RelPath: ".nexusdesk/artifacts/manual/weak.md",
				Kind:    "manual-report",
				Message: "metadata is missing source, job, prompt, query, package, or tool-run lineage",
			}},
		},
	})
	joined := diagnosticsHealthCardText(cards)
	for _, expected := range []string{
		"Artifact provenance|warning|2 artifact(s) checked, 1 provenance issue(s) found.",
		"Open Artifacts, inspect missing metadata/lineage",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected provenance warning to contain %q, got:\n%s", expected, joined)
		}
	}
	text := formatDiagnosticsSnapshot(diagnosticsSnapshot{
		ArtifactProvenance: artifactsSvc.ProvenanceSummary{
			ArtifactCount:  1,
			WithMetadata:   1,
			MissingLineage: 1,
			Issues: []artifactsSvc.ProvenanceIssue{{
				RelPath: ".nexusdesk/artifacts/manual/weak.md",
				Kind:    "manual-report",
				Message: "metadata is missing source, job, prompt, query, package, or tool-run lineage",
			}},
		},
	})
	if !strings.Contains(text, "## Artifact Provenance") || !strings.Contains(text, "manual/weak.md") {
		t.Fatalf("expected provenance diagnostics section, got:\n%s", text)
	}
}

func TestDiagnosticsToolCatalogHealthCardWarnsOnRegistryDrift(t *testing.T) {
	cards := diagnosticsHealthCards(diagnosticsSnapshot{
		ToolCatalogHealth: toolsSvc.ToolCatalogHealth{
			ImplementedCount: 10,
			PlannedCount:     2,
			Violations:       []string{"planned tool \"browser_navigate\" is registered as executable"},
		},
	})
	joined := diagnosticsHealthCardText(cards)
	for _, expected := range []string{
		"Agent tool registry|warning|1 registry violation(s): planned tool \"browser_navigate\" is registered as executable",
		"Fix catalog/dispatcher drift",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected tool-registry warning to contain %q, got:\n%s", expected, joined)
		}
	}
	text := formatDiagnosticsSnapshot(diagnosticsSnapshot{
		ToolCatalogHealth: toolsSvc.ToolCatalogHealth{
			ImplementedCount: 10,
			PlannedCount:     2,
			Violations:       []string{"planned tool \"browser_navigate\" is registered as executable"},
		},
	})
	if !strings.Contains(text, "## Agent Tool Registry") || !strings.Contains(text, "Status: warning") {
		t.Fatalf("expected warning registry section, got:\n%s", text)
	}
}

func TestDiagnosticsFailureGatesHealthCardWarnsOnInvalidMatrix(t *testing.T) {
	cards := diagnosticsHealthCards(diagnosticsSnapshot{
		FailureScenarioIssue: "production failure scenario matrix is empty",
	})
	joined := diagnosticsHealthCardText(cards)
	for _, expected := range []string{
		"Production failure gates|warning|Failure-scenario matrix is incomplete: production failure scenario matrix is empty",
		"Update readiness failure scenarios",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected failure-gates warning to contain %q, got:\n%s", expected, joined)
		}
	}
}

func TestDiagnosticsHealthCardsSurfaceJobPersistenceIssue(t *testing.T) {
	cards := diagnosticsHealthCards(diagnosticsSnapshot{
		ProbeResult:         &llmSvc.ProbeResult{OK: true, Message: "Connected."},
		JobPersistenceIssue: "job-0003: disk full",
		MetadataStatus: &metadataSvc.Status{
			JournalMode:   "wal",
			ForeignKeys:   true,
			BusyTimeoutMS: 5000,
			Tables:        []string{"jobs"},
			Message:       "SQLite metadata store is active.",
		},
	})
	joined := diagnosticsHealthCardText(cards)
	for _, expected := range []string{
		"Jobs and runs|warning|Latest job metadata save failed: job-0003: disk full",
		"Check disk space and metadata health",
	} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected job persistence warning card to contain %q, got:\n%s", expected, joined)
		}
	}
}

func diagnosticsHealthCardText(cards []diagnosticsHealthCard) string {
	lines := make([]string, 0, len(cards))
	for _, card := range cards {
		lines = append(lines, card.Label+"|"+card.Status+"|"+card.Detail+"|"+card.Action)
	}
	return strings.Join(lines, "\n")
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
