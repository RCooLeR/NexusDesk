# Production Readiness Plan

Date: 2026-05-27

This document defines what Nexus Augentic Studio still needs before it can be treated as a production desktop application. `tracker.md` remains the task-level execution source of truth; this document is the release-readiness map.

## Current State

The active product is `nexus-app/`, the Fyne-native application. `app-wails/` is preserved as a reference implementation until native parity is complete enough for daily development.

Approximate migration status:

- Native foundation and core services: mostly complete.
- Wails-era useful workflow parity: roughly 94-95% migrated.
- Production polish, packaging, cross-platform confidence, and advanced connector/editor features: still incomplete.

The app can already:

- open and browse real workspaces;
- preview common files and documents;
- safely edit text/code with rollback records;
- search paths/text and scan lightweight problems;
- inspect Git status/diffs and stage files or hunks through explicit actions;
- discover and run bounded project tasks;
- profile/query local datasets and workspace SQLite files;
- create chart, dashboard, notebook, document, workspace scan, operations, task, chat-answer, and comparison artifacts, with dependency/source rebuild coverage for dataset summary, query, SQL, chart, dashboard, SQL notebook, SQLite query, document-report, scan-report, document-extraction, operations-runbook, artifact-comparison, and chat-answer refresh outputs;
- run Ask and Agent modes against configured OpenAI-compatible or Ollama endpoints;
- let approved agent runs regenerate supported native artifacts from saved source/dependency metadata;
- persist chat, artifact, job, SQL, approval, and agent/tool audit metadata.

## Production Definition

Production-ready means:

1. A non-developer can install and run the app without local source/build knowledge.
2. Opening any normal folder is fast, bounded, and cannot start external tools or model calls.
3. File mutations, database access, Docker/system actions, and agent tools are permissioned, auditable, and reversible where practical.
4. The UI feels like a coherent IDE/data-studio product, not a collection of debug panels.
5. Data loss paths are covered by tests or explicit non-goals.
6. Crashes, hangs, provider failures, and corrupt metadata are visible and recoverable.
7. The preserved Wails app is no longer needed for day-to-day use.

## Release Gates

### Gate 1: Native Parity Beta

Goal: make `nexus-app/` the only app developers need during normal work.

Reference: [Wails Feature Inventory](15_WAILS_FEATURE_INVENTORY.md) records the explicit port/replace/drop/later decisions needed before freezing `app-wails`.

Required:

- IDE-grade editor baseline: first native lightweight syntax strategy, read-only highlighted syntax preview, language-action readiness for formatting/highlighting/outline/definition/reference/LSP status, Problems syntax diagnostics for JSON/Go/YAML/TOML/XML, bounded workspace go-to-definition fallback and references search, command palette baseline, future active-editor inline styling and LSP/cross-file language-action decisions, and continued outline/go-to-symbol/local-definition/document-map/breadcrumb/split/find/format polish.
- External database profile parity: PostgreSQL, MySQL/MariaDB, SQL Server, and DuckDB file/profile read-only query flows with cancellation, caps, redacted errors, and history.
- Native protected secret storage for Windows first is implemented for provider API keys and connector credentials; macOS/Linux keychain backends remain before full cross-platform secret support.
- Assistant quality parity: native Fyne now has weak-evidence warnings, retry/compare, Wails-compatible memory/profile storage, stale-source chat history warnings, Wails-compatible context-to-source fallback parsing, source/model footer diagnostics, line-aware citation refs, explicit unverified/out-of-context citation diagnostics, cited/uncited source coverage diagnostics, bounded citation snippets in saved answer artifacts, deterministic evidence-quality labels, curated model context sizing, loaded-model runtime context tuning, and save-latest-answer artifacts.
- Complete Wails-only feature inventory and explicit keep/drop/replace decisions.
- Native UI cleanup pass for Workbench, Data, Artifacts, Settings, assistant, and bottom panels.

Exit criteria:

- All normal development flows use `nexus-app`.
- `app-wails` is frozen as reference and no longer receives feature work.
- Full native test suite passes on Windows.

### Gate 2: Safety And Reliability Beta

Goal: prove local-first safety and slow-work reliability.

Required:

- Durable job routing for OCR, dump imports, connector pulls, long indexing, report generation, and long agent runs.
- Metadata recovery/export path for `.nexusdesk/metadata`.
- Backup/export flow for local-first workspace state.
- Diagnostics panel for app logs, provider status, metadata health, job history, GPU/model runtime, and recent failures.
- Audit coverage for connector jobs, OCR, dump imports, Docker mutations, shell tools, and future high-risk operations.
- Crash/hang checks for folder open, malformed files, corrupt metadata, missing providers, and canceled long work.

Exit criteria:

- Slow work is cancelable, inspectable, retryable, and never blocks folder open.
- Users can understand what failed and recover or export local state.

### Gate 3: Packaging And Platform Beta

Goal: produce repeatable signed builds.

Required:

- Repeatable Windows build pipeline with app icon, version metadata, installer/update plan, and code-signing path.
- First native CI smoke matrix for Windows, macOS, and Linux covers formatting, `go test ./...`, `go vet ./...`, CGO/Fyne build, ldflag-backed version/commit/build-date metadata validation, and `git diff --check`; signed release packaging and installer/update validation remain open.
- Windows visual/manual smoke checklist for every main surface.
- Linux/macOS build investigation and explicit support matrix, defined in [Platform Support Matrix](14_PLATFORM_SUPPORT.md).
- Antivirus false-positive mitigation notes and release-build hygiene.
- App data path documentation and cleanup/uninstall behavior.

Exit criteria:

- A clean machine can install, launch, open a workspace, run the smoke checklist, and uninstall without source tree access.

### Gate 4: Private Beta

Goal: put the app in front of real users while preserving trust.

Required:

- Onboarding flow for workspace open, model setup, permissions, and local data policy.
- First-run diagnostics for missing model endpoint, missing compiler/build toolchain, and unavailable provider.
- Issue-report bundle that redacts secrets and excludes workspace contents unless explicitly included.
- Documentation for safe agent use, approvals, rollbacks, local data, and connector credentials.
- Beta feedback loop and release notes.

Exit criteria:

- Private users can complete Workbench, Data, Artifact, and Assistant workflows without developer guidance.

## Must Not Ship Before

- Native protected secret storage or explicit refusal behavior is implemented.
- Wails-only connector/profile behavior is either ported or explicitly dropped.
- The agent cannot silently claim file/database/system changes without auditable tool records.
- Long-running jobs cannot freeze folder open or block the main UI.
- Destructive operations lack approval, audit, and rollback/mitigation where practical.
- Packaging lacks a repeatable build and versioned release process.

## Immediate Production-Oriented Next Batch

1. Close the remaining Wails inventory parity blockers: editor maturity, deeper retrieval evidence, and future artifact regeneration coverage for generated presentations.
2. Continue editor parity: active-editor inline syntax styling and future LSP/deeper cross-file language-action behavior.
3. Plan macOS Keychain and Linux Secret Service/libsecret after the Windows protected-secret baseline.
4. Define the durable job contract for OCR, dump imports, connector pulls, report generation, and long agent runs.
5. Continue diagnostics hardening with deeper provider/runtime checks and guided remediation.

## Documentation Rule

Every production-readiness item must be reflected in exactly one of:

- `tracker.md` for task execution;
- `docs/13_PRODUCTION_READINESS.md` for release gates;
- a focused design doc when implementation needs detailed architecture.

Avoid duplicating long checklists across multiple docs. Link back to this file instead.
