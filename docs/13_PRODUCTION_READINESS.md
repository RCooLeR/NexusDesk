# Production Readiness Plan

Date: 2026-05-28

This document defines what Nexus Augentic Studio still needs before it can be treated as a production desktop application. `tracker.md` remains the task-level execution source of truth; this document is the release-readiness map. The broader end-to-end product plan, Claude findings integration, and JetBrains-like UI target live in [End-To-End Production Master Plan](17_END_TO_END_PRODUCTION_PLAN.md). Private-beta safe-agent guidance lives in [NexusDesk Safe Agent User Guide](18_SAFE_AGENT_USER_GUIDE.md), the feedback/release-note process lives in [NexusDesk Beta Feedback And Release Notes Guide](19_BETA_FEEDBACK_AND_RELEASE_NOTES.md), release-candidate smoke coverage lives in [NexusDesk Clean-Machine Smoke Checklist](20_CLEAN_MACHINE_SMOKE_CHECKLIST.md), app data cleanup behavior lives in [NexusDesk App Data And Uninstall Cleanup](21_APP_DATA_AND_UNINSTALL_CLEANUP.md), release hygiene/antivirus guidance lives in [NexusDesk Release Hygiene And Antivirus Notes](22_RELEASE_HYGIENE_AND_ANTIVIRUS.md), contributor package-boundary ownership lives in [NexusDesk Internal Package Ownership](23_INTERNAL_PACKAGE_OWNERSHIP.md), contributor setup/coding/ADR guidance lives in [NexusDesk Contributor Setup And Standards](24_CONTRIBUTOR_SETUP_AND_STANDARDS.md), performance smoke guidance lives in [NexusDesk Performance Profiling Harness](25_PERFORMANCE_PROFILING_HARNESS.md), and the implemented/planned first-party agent toolbelt lives in [NexusDesk Native Agent Tool Registry](27_AGENT_TOOL_REGISTRY.md).

## Current State

The active product is `nexus-app/`, the Fyne-native application. `app-wails/` is preserved as a reference implementation until native parity is complete enough for daily development.

Approximate migration status:

- Native foundation and core services: complete enough for sustained native development.
- Fyne-native migration: roughly 98% complete by useful Wails-era functionality.
- Wails-era useful workflow parity: roughly 97% complete.
- Native Parity Beta readiness: roughly 96% complete.
- Overall production readiness: roughly 95% complete.
- Distribution and packaging readiness: roughly 75-80% complete.

The app can already:

- open and browse real workspaces;
- preview common files and documents;
- safely edit text/code with rollback records;
- search paths/text and scan lightweight problems;
- use quick-open, command palette, native IDE-style status bar, breadcrumbs, split preview, find/replace, formatting, outline, document map, go-to-symbol, local definition, bounded workspace definition fallback, references search, syntax mirror, cursor-aware token/symbol status, and live draft diagnostics;
- inspect Git status/diffs and stage files or hunks through explicit actions;
- discover and run bounded project tasks;
- profile/query local datasets and workspace SQLite files with DataGrip-like row-grid feedback for visible rows/columns, density, sizing strategy, selection, hidden-column warnings, and copy hints;
- manage read-only external database profiles for PostgreSQL, MySQL/MariaDB, SQL Server, SQLite, and DuckDB guarded builds with cancellation, redaction, history, and OS-protected credential sidecars;
- create chart, dashboard, notebook, document, document-brief, validated/theme-backed DOCX document export, workspace scan, operations, task, chat-answer, comparison, presentation-outline, packaged presentation zip, and validated/theme-backed PPTX presentation deck artifacts, with dependency/source rebuild coverage for dataset summary, query, SQL, chart, dashboard, SQL notebook, SQLite query, document-report, document-brief, document-export, scan-report, document-extraction, operations-runbook, artifact-comparison, chat-answer refresh, presentation-outline, presentation-package, and presentation-deck outputs, plus artifact browser row badges and preview summaries for type, lineage, regeneration/export capability, comparison state, archive state, metadata, sources, and jobs;
- run Ask and Agent modes against configured OpenAI-compatible or Ollama endpoints;
- let approved agent runs regenerate supported native artifacts from saved source/dependency metadata;
- inspect the native tool catalog, let agent runs read local dataset profile/query/SELECT-only SQL context, extract workspace document text, inspect operations files read-only with redaction, inspect optional external coding-agent CLI readiness for Codex, Claude Code, and OpenCode, create non-executing run plans with approval/audit/cancel/output-capture requirements, and run first-party approved terminal commands through explicit argv, rooted cwd, timeout/output caps, shell/path blocking, and high-risk tool audit;
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

The professional UI target is a JetBrains-like native workbench: compact dark theme, top menu/toolbar, project and tool-window rails, central tabbed editor, right integrated assistant, grouped bottom tools, DataGrip-style data surfaces, searchable settings, visible diagnostics, and keyboard-first navigation. The native color/density baseline is defined in [NexusDesk Native Theme Tokens](26_NATIVE_THEME_TOKENS.md).

## Release Gates

### Gate 1: Native Parity Beta

Goal: make `nexus-app/` the only app developers need during normal work.

Reference: [Wails Feature Inventory](15_WAILS_FEATURE_INVENTORY.md) records the explicit port/replace/drop/later decisions needed before freezing `app-wails`. [Native Editor Parity Strategy](16_EDITOR_PARITY_STRATEGY.md) records the editor-specific beta replacement decision.

Required:

- IDE-grade editor baseline: first native lightweight syntax strategy, read-only highlighted syntax preview with cursor-aware active-line/token/symbol status, language-action readiness for formatting/highlighting/draft-diagnostics/outline/definition/reference/LSP status, live unsaved-draft diagnostics for markers/merge conflicts/JSON/Go/YAML/TOML/XML, Problems syntax diagnostics for saved JSON/Go/YAML/TOML/XML files, bounded workspace go-to-definition fallback and references search, command palette baseline, documented Syntax mirror/Document Map beta replacement decision, and continued outline/go-to-symbol/local-definition/document-map/breadcrumb/split/find/format polish.
- External database profile parity: PostgreSQL, MySQL/MariaDB, SQL Server, and DuckDB file/profile read-only query flows with cancellation, caps, redacted errors, and history.
- Native protected secret storage is implemented for provider API keys and connector credentials with Windows DPAPI, macOS Keychain, Linux Secret Service/libsecret via `secret-tool`, and explicit refusal on unsupported platforms.
- Assistant quality parity: native Fyne now has weak-evidence warnings, retry/compare, Wails-compatible memory/profile storage, stale-source chat history warnings, Wails-compatible context-to-source fallback parsing, source/model footer diagnostics, a visible Ask/Agent run status strip, bounded open/pin source actions, a native latest-answer source digest, line-aware citation refs, explicit unverified/out-of-context citation diagnostics, cited/uncited source coverage diagnostics, structured cited/uncited source coverage metadata in saved answer artifacts, bounded citation snippets in saved answer artifacts, deterministic evidence-quality labels, curated model context sizing, loaded-model runtime context tuning, and save-latest-answer artifacts.
- Task-aware model defaults are now stored and editable in searchable grouped Settings with inline readiness validation for coding, backend, database, analytics, research, vision/screenshot, balanced reasoning, and fast-coding routes while preserving a global fallback. Ask mode exposes an auto/global/manual route selector with pre-run route and context-budget visibility, assistant/agent auto-routing infers data, SQL, document, image/screenshot, and code routes from selected context or prompt signals, Git AI diff summary/commit drafting resolves the main coding route, Agent mode resolves the selected route with audit provenance, and saved chat-answer artifacts can preserve route metadata; future dedicated Data/document/vision model workflow resolution remains planned.
- Complete Wails-only feature inventory and explicit keep/drop/replace decisions.
- Native UI cleanup pass for Workbench, Data, Artifacts, Settings, assistant, and bottom panels.
- UI direction follows `docs/17_END_TO_END_PRODUCTION_PLAN.md` and the token baseline in `docs/26_NATIVE_THEME_TOKENS.md`: JetBrains-like menu/toolbar/status shell hierarchy, grouped tool windows, keyboard-reachable first left/right tool-window rails with active-state highlighting, professional settings, dense but readable data grids, and integrated assistant/source diagnostics.
- Import-boundary tests now enforce the active architecture: no Wails/webview imports, no Fyne imports in services/domain, and no UI imports from framework-free service/domain packages.

Remaining blockers:

- Post-beta editable-widget inline syntax styling if it preserves safe editing/accessibility.
- Future LSP/deeper cross-file language actions after a packaged provider spike proves reliability.
- Deeper assistant retrieval/ranking quality beyond deterministic citation/source coverage diagnostics.
- Richer DOCX/PPTX generated-output template variants, cross-suite compatibility smoke, and visual polish beyond the native validated/theme-backed document-export and presentation-deck baselines.
- Final UI polish for empty states, assistant hierarchy, data/artifact/diagnostics surfaces, and workflow hierarchy after the first native Home readiness/onboarding cockpit and left/right tool-window rail baselines.

Exit criteria:

- All normal development flows use `nexus-app`.
- `app-wails` is frozen as reference and no longer receives feature work.
- Full native test suite passes on Windows.

### Gate 2: Safety And Reliability Beta

Goal: prove local-first safety and slow-work reliability.

Required:

- Shared durable slow-workflow contract for OCR, dump imports, connector pulls, long indexing, report generation, long agent runs, and packaged exports.
- Concrete durable job routing for each slow workflow as it is implemented.
- Metadata recovery/export path for `.nexusdesk/metadata`.
- Search metadata recovery/export path for explicit workspace searches: bounded `.nexusdesk/search/index-metadata.json` manifests store result paths/lines and scan counts without snippets, and corrupt manifests are quarantined before replacement.
- Backup/export flow for local-first workspace state.
- Diagnostics panel for app logs, provider status, metadata health, job history, GPU/model runtime, recent failures, provider-specific model/runtime remediation guidance, and startup/folder-open performance timings with over-budget warnings.
- External coding-agent execution implementation for Codex, Claude Code, OpenCode, and similar CLIs remains optional and secondary to NexusDesk's own toolbelt: readiness and non-executing plans exist, but any future process launch must still be routed through approved jobs/shell policy, cancellation, audit, redaction, sandboxing, and artifact/output capture.
- Job history retention controls and cleanup policy: the Jobs panel can prune successful/canceled completed jobs by count/age while preserving running jobs and failures/timeouts by default.
- Startup recovery markers and crash/hang triage visibility: launch writes a local session marker, clean exit closes it, and Home/Diagnostics warn when the previous run did not shut down cleanly.
- Audit coverage for connector jobs, OCR, dump imports, Docker mutations, richer terminal sessions, and future high-risk operations.
- Crash/hang checks for folder open, malformed files, corrupt metadata, missing providers, and canceled long work.

Exit criteria:

- Slow work is cancelable, inspectable, retryable, and never blocks folder open.
- Users can understand what failed and recover or export local state.

### Gate 3: Packaging And Platform Beta

Goal: produce repeatable signed builds.

Required:

- Repeatable Windows build pipeline with app icon, version metadata, installer/update plan, and code-signing path.
- First native CI smoke matrix for Windows, macOS, and Linux covers formatting, `go test ./...`, `go vet ./...`, CGO/Fyne build, ldflag-backed version/commit/build-date metadata validation, release manifest generation/validation with artifact SHA256 and size, and `git diff --check`; signed release packaging and installer/update validation remain open.
- Windows visual/manual smoke checklist for every main surface. Implemented in `docs/20_CLEAN_MACHINE_SMOKE_CHECKLIST.md` and exposed in-product from Help and the command palette.
- Linux/macOS build investigation and explicit support matrix, defined in [Platform Support Matrix](14_PLATFORM_SUPPORT.md).
- Antivirus false-positive mitigation notes and release-build hygiene. Implemented in `docs/22_RELEASE_HYGIENE_AND_ANTIVIRUS.md` and exposed in-product from Help and the command palette.
- App data path documentation and cleanup/uninstall behavior. Implemented in `docs/21_APP_DATA_AND_UNINSTALL_CLEANUP.md` and exposed in-product from Help and the command palette.

Exit criteria:

- A clean machine can install, launch, open a workspace, run the smoke checklist, and uninstall without source tree access.

### Gate 4: Private Beta

Goal: put the app in front of real users while preserving trust.

Required:

- Onboarding flow for workspace open, model setup, permissions, and local data policy. Implemented as a native Home readiness cockpit with setup health, safety posture, and first actions.
- First-run diagnostics for missing model endpoint, missing compiler/build toolchain, and unavailable provider. Implemented in the Home readiness cockpit; Diagnostics now adds provider-specific remediation for common Ollama/OpenAI-compatible failures such as stopped local runtimes, missing models, unloaded models, auth failures, and bad `/v1` base URLs.
- Issue-report bundle that redacts secrets and excludes workspace contents unless explicitly included. Implemented in Diagnostics as a redacted ZIP export containing diagnostics text, activity tail, environment metadata, workspace-state file names, and no workspace file contents by default. Diagnostics reports now include health cards for provider, metadata, jobs/runs, performance, startup recovery, and issue-report readiness before detailed logs.
- Documentation for safe agent use, approvals, rollbacks, local data, and connector credentials. Implemented in `docs/18_SAFE_AGENT_USER_GUIDE.md` and exposed in-product from Help and the command palette.
- Beta feedback loop and release notes. Implemented in `docs/19_BETA_FEEDBACK_AND_RELEASE_NOTES.md` and exposed in-product from Help and the command palette.

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

1. Keep `docs/17_END_TO_END_PRODUCTION_PLAN.md`, `tracker.md`, this file, and the Wails inventory synchronized as the current source of truth.
2. Apply the durable slow-workflow contract to concrete OCR, dump import, connector pull, long indexing, report generation, and long agent run implementations.
3. Continue post-beta editor spikes for inline styling and packaged LSP while preserving the documented Syntax mirror/Document Map Native Parity Beta strategy.
4. Polish generated document/deck artifacts beyond native validated/theme-backed DOCX/PPTX baselines with richer template variants, cross-suite compatibility smoke, and visual design.
5. Build signed release packaging, installer/update validation, and artifact upload/signing around the generated release manifests.
6. Run a focused JetBrains-like UI polish pass on onboarding, empty states, settings, diagnostics, data grids, assistant, and workflow hierarchy.
7. Validate macOS Keychain/Linux Secret Service behavior in platform packaging smoke runs.

## Documentation Rule

Every production-readiness item must be reflected in exactly one of:

- `tracker.md` for task execution;
- `docs/13_PRODUCTION_READINESS.md` for release gates;
- `docs/17_END_TO_END_PRODUCTION_PLAN.md` for end-to-end product direction, UI north star, and combined roadmap;
- a focused design doc when implementation needs detailed architecture.

Avoid duplicating long checklists across multiple docs. Link back to this file instead.
