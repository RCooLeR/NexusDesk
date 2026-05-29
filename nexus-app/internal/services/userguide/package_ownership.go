package userguide

func PackageOwnershipGuide() Guide {
	return Guide{
		Title:   "Internal Package Ownership",
		Summary: "A contributor-facing ownership map for keeping the Fyne-native app modular, testable, and safe as NexusDesk moves toward production.",
		Sections: []Section{
			{
				Title: "Layer Rules",
				Body: []string{
					"`internal/app` owns application startup, dependency assembly, and Fyne window lifecycle. It should stay thin and should not absorb workflow rules.",
					"`internal/domain` owns framework-free domain types and invariants. It must not import Fyne, shell code, storage adapters, or service implementation details.",
					"`internal/services` owns UI-independent behavior: workspace safety, metadata, jobs, assistant/agent orchestration, connector rules, artifacts, Git, tasks, operations, settings, and release/user-guide models.",
					"`internal/ui` owns Fyne presentation: shell layout, menus, panels, dialogs, keyboard shortcuts, rendering, and user intent dispatch. It should call services instead of reimplementing business rules.",
				},
			},
			{
				Title: "Safety-Critical Services",
				Body: []string{
					"`services/workspace` owns rooted file reads, previews, search, context packs, problems, safe writes, file operations, rollback records, and metadata-safe path rules.",
					"`services/approvals`, `services/protectedsecret`, and `services/tools` own approval records, protected credential storage, deterministic tool dispatch, risk metadata, and high-risk action boundaries.",
					"`services/jobs`, `services/startup`, `services/metadata`, and `services/issuereport` own durable job state, startup recovery markers, SQLite metadata, compatibility import, backup, diagnostics evidence, and redacted issue-report bundles.",
					"Future OCR, dump import, connector sync, shell, and Docker mutation workflows must enter through services with jobs, approvals, audit, redaction, and rollback or mitigation design before UI exposure.",
				},
			},
			{
				Title: "Studio Workflow Services",
				Body: []string{
					"`services/editor` owns tab identity, dirty-state policy, pinned ordering, and close guards; UI widgets own only the visual editing surface.",
					"`services/git`, `services/tasks`, and `services/operations` own manual Git inspection/actions, safe discovered task execution, and read-only operations evidence/runbooks.",
					"`services/datasets`, `services/dbconnector`, `services/spreadsheets`, and `services/documents` own data profiling/querying, read-only connector execution, workbook parsing, and bounded document extraction.",
					"`services/artifacts` and `services/history` own generated files, sidecar metadata, lineage, freshness, archive/restore/delete, artifact search, comparison, regeneration metadata, and unified history composition.",
				},
			},
			{
				Title: "Assistant And Provider Services",
				Body: []string{
					"`services/llm` owns provider transport, streaming, model probes, Ollama runtime diagnostics, context-window handling, and request/response bounds.",
					"`services/assistant` owns Ask-mode request preparation and context packaging; `services/agent` owns Agent-mode loop behavior, plan state, observations, and final-answer handling.",
					"`services/settings`, `services/recentworkspaces`, `services/readiness`, and `services/webfetch` own non-secret settings, recent workspace persistence, readiness checks, and approval-gated bounded web text fetches.",
				},
			},
			{
				Title: "Presentation Packages",
				Body: []string{
					"`internal/ui/shell` owns the native workbench: project tree, editor tabs, assistant, Data, Artifacts, Git, Jobs, Diagnostics, History, Approvals, Settings, menus, shortcuts, and dialogs.",
					"`internal/ui/theme` owns presentation tokens and Fyne theme adaptation; `internal/brand` owns approved product assets used by native windows and packaging.",
					"`internal/architecture` owns import-boundary tests that prevent deprecated runtime dependencies and Fyne/UI leakage into services or domain packages.",
					"`internal/buildinfo` and `internal/release` own build metadata validation, About text, release manifest models, checksums, and release hygiene support.",
				},
			},
			{
				Title: "Change Checklist",
				Body: []string{
					"Put new business rules in services first, add focused service tests, then wire UI intent to the service result.",
					"Keep folder open cheap: no Git, Docker, OCR, connector pulls, dump imports, model calls, shell commands, or deep indexing on workspace open.",
					"Do not write into `.nexusdesk` through generic file mutation paths. Internal services may own explicit metadata/artifact/recovery writes with rooted path checks.",
					"Update `tracker.md`, production docs, and this ownership guide when a new package or cross-package responsibility becomes stable.",
				},
			},
		},
	}
}

func PackageOwnershipMarkdown() string {
	return FormatMarkdown(PackageOwnershipGuide())
}
