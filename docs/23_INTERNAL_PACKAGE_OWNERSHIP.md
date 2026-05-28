# NexusDesk Internal Package Ownership

Date: 2026-05-28

This document is the contributor-facing ownership map for the Fyne-native `nexus-app/` module. It exists to keep NexusDesk modular as it moves toward production readiness: services stay framework-free, the Fyne shell stays presentation-focused, slow work goes through jobs, and risky actions keep approvals, audit, rollback or mitigation, and redaction.

## Layer Rules

| Area | Owner | Must Own | Must Not Own |
| --- | --- | --- | --- |
| `internal/app` | Native app lifecycle | Dependency assembly, app/window creation, startup/shutdown hooks, top-level wiring | Workspace business rules, file/database safety, artifact generation, long workflow logic |
| `internal/domain` | Framework-free product model | Shared value types and invariants that can be used by services | Fyne imports, UI state, SQLite/file adapters, network clients |
| `internal/services` | UI-independent application behavior | Workspace safety, metadata, jobs, assistant/agent logic, connectors, artifacts, Git, tasks, operations, settings, release/user-guide models | Fyne widgets/dialogs, shell layout, direct user interaction state |
| `internal/ui` | Native presentation | Fyne shell, panels, menus, dialogs, shortcuts, rendering, user intent dispatch | Business rules for file writes, database safety, approvals, connector validation, Docker/system safety |
| `internal/brand` | Product assets | Approved icons/logos and brand constants used by native UI and packaging | Workflow behavior or package-specific UI state |
| `internal/architecture` | Guardrails | Import-boundary tests that prevent Wails/webview reintroduction and Fyne leakage into services/domain | Runtime app behavior |
| `internal/buildinfo` and `internal/release` | Release metadata | Version/About metadata, manifest models, artifact hash/size validation, release hygiene support | Installer-specific mutation workflows |

## Service Ownership

### Safety-Critical Services

| Package | Owns | Notes |
| --- | --- | --- |
| `services/workspace` | Rooted file reads, previews, search, context packs, problems, safe writes, patches, file operations, rollback records, search metadata export/recovery, `.nexusdesk` path guards | Generic file mutation paths must not write into `.nexusdesk`; explicit internal metadata/artifact/recovery writers must use rooted path checks. |
| `services/approvals` | Approval records, full-project access policy, approval persistence fallback | UI may request approvals, but risk policy belongs here. |
| `services/protectedsecret` | OS-protected secret storage via Windows DPAPI, macOS Keychain, Linux Secret Service/libsecret, and unsupported-platform refusal | UI can display redacted values only. |
| `services/tools` | Deterministic agent tool descriptors, risk metadata, argument parsing, dispatch, approval-gated high-risk tool execution | Arbitrary shell remains out of scope until explicitly designed. |
| `services/jobs` | Job IDs, status, logs, cancellation, retry state, retention policy, repository hooks | Slow workflows should route through jobs before UI exposure. |
| `services/startup` | App-session recovery markers and previous unclean-exit detection | Home/Diagnostics render this state; they do not own marker semantics. |
| `services/metadata` | SQLite metadata, schema, jobs/tasks/chats/approvals/artifacts/SQL/agent/tool records, Wails compatibility import, backup/recovery | UI and agent code should not read SQLite directly. |
| `services/issuereport` | Redacted diagnostics bundles and workspace-state evidence export | Workspace contents remain opt-in, not default. |

### Studio Workflow Services

| Package | Owns | Notes |
| --- | --- | --- |
| `services/editor` | Tab identity, dirty state, pinned ordering, close guards | Editing widgets render state; they do not decide save safety. |
| `services/git` | Manual Git status, diffs, hunk parsing, stage/unstage, hidden Windows child processes | Git work must stay manual and user-triggered. |
| `services/tasks` | Safe discovered task inventory and execution for known task kinds | UI must execute rediscovered task IDs, not arbitrary command strings. |
| `services/operations` | Read-only Dockerfile, Compose, env/config/script/log inspection, topology summaries, runbook evidence | Mutating Docker/system actions require future job/audit/approval design. |
| `services/datasets` | Dataset profiling, bounded queries, SELECT-only dataset SQL, notebooks, chart/dashboard models | Long imports and connector syncs remain job-backed future work. |
| `services/dbconnector` | Read-only workspace SQLite and external SQL connector inspection/query/cancellation/redaction | Credentials flow through protected secret services. |
| `services/spreadsheets` | Dependency-light XLSX/OpenXML bounded workbook reads | Keep parsing framework-free and capped. |
| `services/documents` | Bounded document text extraction for preview/report workflows | OCR/scanned PDFs/images must be job-backed before UI exposure. |
| `services/artifacts` | Generated artifacts, metadata sidecars, lineage, freshness, search, preview, compare, archive/delete/restore, regeneration source data | UI owns intent; this package owns generated-file behavior. |
| `services/history` | Unified history composition across metadata/artifacts/jobs/agent audit | UI should render the feed and jump actions only. |

### Assistant, Provider, And Configuration Services

| Package | Owns | Notes |
| --- | --- | --- |
| `services/llm` | OpenAI-compatible transport, streaming, model probes, Ollama diagnostics, context-window and response-reserve bounds | Provider failures should be surfaced with actionable diagnostics. |
| `services/assistant` | Ask-mode request preparation and selected-context packaging | It should remain UI-independent. |
| `services/agent` | Agent loop behavior, plan updates, observations, loop guards, final-answer handling | Tool execution stays in `services/tools`. |
| `services/settings` | Non-secret provider/model/context settings and protected-secret references | Secrets themselves stay in `services/protectedsecret`. |
| `services/recentworkspaces` | Recent workspace persistence | Workspace open remains cheap. |
| `services/readiness` | Home readiness checks and setup status models | Checks should be bounded and non-invasive. |
| `services/webfetch` | Approval-gated bounded HTTP(S) text fetches with local-network and content limits | Browser automation/screenshot capture is not part of this service. |
| `services/userguide` | In-app Help guide models and Markdown generation | Guides should mirror stable docs and remain test-covered. |

## UI Ownership

`internal/ui/shell` owns the native workbench presentation:

- project tree, quick open, editor tabs, split preview, syntax mirror, document map, find/replace, and dirty-state prompts;
- assistant panel, chat history, agent audit, approvals, jobs, diagnostics, history, operations, Git, Data, Artifacts, Settings, and bottom tool windows;
- menus, command palette, keyboard shortcuts, dialogs, status/activity text, and user intent routing.

Rules for shell work:

- UI code may collect intent and render structured service results.
- UI code must not duplicate path traversal checks, SQL mutation blocking, protected-secret handling, rollback semantics, or job retention policy.
- As `internal/ui/shell` grows, extract focused files/controllers by workflow rather than creating a new monolithic bridge.
- Fyne imports belong in `internal/app`, `internal/ui`, `internal/ui/theme`, `internal/brand`, and UI tests only.

## Slow Workflow Rule

The following workflows must not run on folder open and must not block the UI:

- long indexing;
- OCR and scanned-document extraction;
- database dump imports;
- connector sync/pull jobs;
- long report generation;
- long agent runs;
- packaged export operations that can take noticeable time;
- shell, Docker, or system mutation workflows if they are ever introduced.

They need a durable job model with progress, logs, cancellation, retry/output-open behavior, persisted metadata, redaction, and audit continuity before becoming visible UI actions.

## Risky Action Rule

Risky actions must preserve user trust:

- file writes, patches, deletes, moves, and rollbacks go through `services/workspace`;
- agent high-risk tools go through `services/tools` and `services/approvals`;
- connector credentials go through `services/protectedsecret`;
- SQL/database execution remains read-only unless a future mutation design adds approvals, audit, rollback/mitigation, and clear scope;
- Docker/system/shell mutations remain unavailable until their policy, job, audit, and mitigation design is complete;
- issue-report and diagnostics exports redact secrets by default and exclude workspace content unless explicitly included.

## Change Checklist

Before adding or moving functionality:

1. Decide the owning package before coding.
2. Put business rules in services first.
3. Add focused service tests for path safety, bounds, redaction, cancellation, or metadata behavior.
4. Wire UI intent to service methods after service behavior is stable.
5. Keep `app-wails/` as reference only; do not reintroduce Wails/webview dependencies into `nexus-app/`.
6. Update `tracker.md`, `docs/17_END_TO_END_PRODUCTION_PLAN.md`, and this document when ownership changes.
7. Run import-boundary tests through `go test ./internal/architecture` when touching package seams.

## Open Ownership Work

- Continue extracting `internal/ui/shell` controllers by responsibility.
- Define final shell state ownership boundaries for workspace, editor, assistant, data, artifacts, jobs, diagnostics, and settings.
- Add contributor setup, coding standards, ADR index, and test-fixture policy.
- Define stable service interfaces for future contributed connectors and parsers.
- Define the extension security model before any third-party code execution.
