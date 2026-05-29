# NexusDesk End-To-End Production Master Plan

Date: 2026-05-28
Status: Active master plan for Fyne-native production completion
Primary app: `nexus-app/`
Reference app: `app-wails/`

This document combines the latest project review, the Wails feature inventory, the production readiness plan, Claude's static findings, and the JetBrains-style UI references provided by the product owner. It exists so repeated development sessions keep the same product idea in view: NexusDesk is a native, local-first agentic workbench for code, data, documents, artifacts, operations, and assistant-assisted development.

`tracker.md` remains the task-level checklist. `docs/13_PRODUCTION_READINESS.md` remains the release-gate map. `docs/18_SAFE_AGENT_USER_GUIDE.md` is the private-beta safety guide for agent use, approvals, rollbacks, local data, connector credentials, jobs, diagnostics, and issue reports. `docs/19_BETA_FEEDBACK_AND_RELEASE_NOTES.md` defines the private-beta feedback and release-note loop. `docs/20_CLEAN_MACHINE_SMOKE_CHECKLIST.md` defines release-candidate clean-machine smoke coverage. `docs/21_APP_DATA_AND_UNINSTALL_CLEANUP.md` defines app data, uninstall, and manual cleanup behavior. `docs/22_RELEASE_HYGIENE_AND_ANTIVIRUS.md` defines release artifact discipline, signing/trust expectations, antivirus false-positive triage, release-note requirements, and do-not-ship rules. `docs/26_NATIVE_THEME_TOKENS.md` defines the Fyne-native JetBrains-like theme token baseline. This file is the end-to-end product and architecture plan that explains why each remaining task matters and what finished should look like.

## 1. Product North Star

NexusDesk should become a production-ready native desktop studio that feels closer to a JetBrains IDE plus local data/document/operations workbench than to a browser dashboard.

The core promise:

- Open a local project or workspace safely.
- Understand code, data, documents, artifacts, Git state, tasks, and operations evidence in one cohesive native environment.
- Ask or instruct an assistant that can cite sources, inspect local context, run safe tools, create artifacts, and request explicit approvals for higher-risk work.
- Keep user trust: no surprise model calls, no surprise shell commands, no surprise Docker/database mutations, no hidden external work on folder open.
- Preserve local-first privacy and reversibility through metadata, provenance, approvals, audit trails, rollbacks, diagnostics, and backups.
- Ship as a polished desktop product on Windows first, with clear macOS and Linux support paths.

NexusDesk is not intended to become:

- a webview shell with native branding;
- a loosely connected set of debug panels;
- a cloud-first IDE clone;
- an autonomous agent that silently changes files or systems;
- a data platform that mutates production databases without explicit guardrails;
- a toy editor that loses the workbench/assistant/data/artifact idea.

## 2. Visual And UX North Star

The supplied screenshots set the intended visual direction: a professional JetBrains-like dark IDE shell with dense, calm information architecture and tool-window discipline.

Target personality:

- Native, compact, serious, and productivity-oriented.
- Dark professional theme by default, with good contrast and subdued color accents.
- Clear hierarchy: top menu, toolbar, project rail, central editor, right assistant, bottom tool windows, status bar.
- Minimal visual noise, but enough density for power users.
- The assistant should feel integrated into the IDE, not bolted on as a chat website.
- Data grids should feel closer to DataGrip than to a generic table widget.
- Settings should feel like a desktop preferences dialog with grouped navigation, validation, and inline diagnostics.
- Empty states should teach the product without looking like onboarding marketing pages.

### 2.1 Target Shell Layout

The final shell should converge on this structure:

- Top menu: File, Edit, View, Navigate, Code, Refactor, Run, Tools, Help.
- Top toolbar: project/workspace selector, branch/status segment, run/task selector, quick actions, search, settings.
- Left tool-window rail: Project, Search, Problems, Git, Data, Artifacts, Operations, Jobs, Diagnostics.
- Left primary pane: Project tree by default, with switchable tool-window content.
- Center workspace: tabbed editor/artifact/data/documents area with split support.
- Right tool-window rail: Assistant, Sources, Artifact Lineage, Job Monitor, Inspector.
- Right primary pane: AI Chat/Agent by default with source diagnostics and run approvals.
- Bottom tool windows: Problems, Terminal/Tasks, Jobs, Git details, Data result details, Audit, Diagnostics.
- Status bar: workspace root, provider/model health, branch, job count, warning count, selection/path, encoding, line endings, app version.

### 2.2 JetBrains-Like Interaction Principles

- Tool windows should be grouped, pinnable/collapsible, and keyboard reachable.
- Bottom panels should not expose a long ungrouped row of tabs. Use modes/groups and contextual surfaces.
- Keyboard-first workflows matter: quick open, command palette, find, go to symbol, go to definition, references, run task, open settings, toggle assistant, toggle terminal/jobs.
- Context menus should be complete but not bloated; common actions first, dangerous actions visually separated and approval-gated.
- Settings should support search, grouped categories, provider test buttons, validation, and useful disabled-state explanations.
- Long workflows should become jobs with progress, logs, cancel, retry, and output opening.
- Assistant citations should be visible and actionable, with source open/pin actions.
- Errors should be recoverable: show what failed, why it probably failed, and what the user can do next.

### 2.3 UI Completion Bar

Native Parity Beta can accept the current Fyne editor strategy, but production UI should not stop at functional parity. The production UI bar is:

- No crowded debug-panel feeling.
- Major surfaces look designed as a single product.
- Dense grids and workbench panes remain readable on 1440p and laptop widths.
- Empty states are useful and calm.
- First-run flow gets a user to open workspace, configure model, understand local data, and run safe first actions.
- Settings, diagnostics, approvals, and job history build trust rather than confusion.

## 3. Architecture North Star

The Fyne migration is correct because the desired product is native, local-first, and Go-service-heavy. The architecture must stay strict.

### 3.1 Package Ownership

- `nexus-app/main.go`: thin executable entrypoint only.
- `nexus-app/internal/app`: app lifecycle, window setup, dependency assembly.
- `nexus-app/internal/domain`: framework-free domain models and value types.
- `nexus-app/internal/services`: framework-free business logic, file safety, metadata, jobs, assistants, connectors, artifacts, Git, tasks, operations.
- `nexus-app/internal/ui`: Fyne shell, widgets, layouts, dialogs, menus, theme, presentation logic.
- `nexus-app/internal/brand`: brand assets and theme constants.
- `app-wails/`: reference implementation only until explicit freeze/archive decision.
- Detailed package ownership is captured in `docs/23_INTERNAL_PACKAGE_OWNERSHIP.md` and exposed in-product through Help and the command palette.
- Contributor setup, coding standards, validation expectations, and ADR process are captured in `docs/24_CONTRIBUTOR_SETUP_AND_STANDARDS.md` and exposed in-product through Help and the command palette.
- The framework-free performance profiling harness and live startup/folder-open Diagnostics timing records are documented in `docs/25_PERFORMANCE_PROFILING_HARNESS.md` and implemented under `nexus-app/internal/services/perf`.

### 3.2 Non-Negotiable Architecture Rules

- Domain and services must not import Fyne.
- Active app must not reintroduce Wails/webview dependencies.
- UI packages may depend on services; services must not depend on UI packages.
- Slow work must be represented as jobs before being exposed as a production workflow.
- Workspace open must stay cheap and must not trigger Git, Docker, OCR, connector pulls, dump imports, model calls, shell commands, or deep indexing.
- File writes must stay rooted, audited, approval-aware, and rollback-backed where practical.
- Secrets must use OS-protected storage or explicit unsupported-platform refusal.
- External database work must default to read-only, bounded, redacted, cancelable, and auditable.
- Agent tools must be explicit, inspectable, bounded, approval-gated by risk, and recorded.
- Compatibility import from Wails metadata must stay asynchronous and never block folder open.

### 3.3 Current Architecture Health

Healthy:

- The executable root is thin.
- Services and domain are mostly clean and framework-free.
- Fyne is contained to app/UI/theme/test areas.
- Safety boundaries are service-owned.
- Durable metadata exists for chats, approvals, artifacts, jobs, SQL runs, dataset dependencies, agent runs, and tool runs.
- Compatibility import, metadata recovery, backup export, diagnostics, and job records are present.
- Most Wails useful-code parity has been ported or replaced with native equivalents.

Still risky:

- `internal/ui/shell` remains the largest orchestration area.
- Some UI panels still carry too much state and behavior.
- Data, artifacts, assistant, operations, and diagnostics have strong capability, but need final information-architecture polish.
- Production packaging/signing/install/update flows remain less complete than core app capability.
- Richer generated document/deck templates and cross-suite compatibility smoke are not complete.
- Deeper assistant retrieval/ranking and broader source quality remain important post-parity work.

## 4. Current Capability Inventory

This section records what the app can already do so future planning does not lose the main product idea.

### 4.1 Workspace And Workbench

Already available:

- Launches as a Fyne-native desktop app.
- Opens local folders with native dialogs.
- Maintains recent workspaces and a Home/open flow with a first-run readiness cockpit for workspace, provider/model, credentials, native toolchain, safety posture, and quick actions.
- Renders lazy project tree with ignored-path handling, entry caps, refresh, reveal, collapse, and context actions.
- Supports single-file open and quick-open workflows.
- Keeps folder open fast by avoiding automatic expensive or external operations.
- Provides JetBrains-like top menu groups, a workspace/branch/task/provider toolbar, keyboard-reachable first left/right tool-window rails with active-state highlighting, shortcuts, command palette baseline, activity, native status bar, bottom panels, and assistant integration.

Planned/remaining:

- More JetBrains-like assistant/data/artifact tool-window grouping and surface-specific polish. Assistant now has a first visible run/status strip for mode, route, context, evidence, source, and Agent job safety, plus bounded open/pin source actions and a native source digest for latest answers.
- Continue polishing the workspace home surface after the first-run readiness cockpit baseline.
- Continued global health indicator polish beyond the first native status bar.
- More consistent keyboard navigation/focus behavior across all panes.

### 4.2 Editor And Code Navigation

Already available:

- Draft editing with dirty markers, pinned tabs, close guards, save, revert, rollback, and explicit discard confirmation.
- Text/code preview for common code and config formats.
- Markdown source/rendered view.
- Find/replace and formatting actions for supported languages and text-like files.
- Breadcrumbs, split preview, outline, go-to-symbol, local definition, bounded workspace definition fallback, references search.
- Wails-derived language detection.
- Syntax mirror with bounded token analysis, line display, token coloring, active-line highlighting, and token/symbol status.
- Document Map replacing Monaco minimap value for Native Parity Beta.
- Live unsaved-draft diagnostics for markers, merge conflicts, JSON, Go, YAML, TOML, XML.
- Problems scans for saved JSON, Go, YAML, TOML, XML and marker/conflict detection.

Planned/remaining:

- Post-beta editable inline syntax styling only if safe, accessible, and performant.
- Packaged LSP spike behind feature flag with cancellation and failure isolation.
- Deeper cross-file language actions and semantic diagnostics after local fallback remains stable.
- More polished editor chrome, split groups, search result navigation, and document-map visuals.

### 4.3 Search, Problems, And Indexing

Already available:

- Workspace path/content search with snippets and multiple matches.
- Problems scanner with markers, conflicts, syntax diagnostics.
- Search snippets centered around match location.
- Entry caps and truncation UI for large directories.
- Shared candidate policy for preview/context behavior.
- Search metadata export/recovery: explicit workspace searches persist bounded `.nexusdesk/search/index-metadata.json` manifests with result paths/lines and scan counts, while corrupt search metadata is quarantined under `.nexusdesk/search/recovery/` before a clean manifest is written.

Planned/remaining:

- Long indexing as explicit durable job, never on folder open.
- Better ranking, grouping, and source weighting for assistant retrieval.

### 4.4 Assistant And Agent

Already available:

- Native Ask and Agent modes.
- OpenAI-compatible, Ollama, and custom endpoint settings.
- Protected API key storage and bearer auth propagation.
- Provider probe, model count, runtime diagnostics, curated model catalog, loaded-model context tuning.
- Streaming responses with cancellation handling.
- Context packs from selected workspace, directories, files, artifacts, and prior chat.
- Persisted chat history, search/history navigation, retry, compare, save latest answer.
- Assistant memory and prompt profiles.
- Separate Ask vs Agent prompt behavior.
- Source/model footer diagnostics plus a native source digest for latest-answer evidence quality, verified refs, unverified refs, cited sources, and uncited sources.
- In-product Safe Agent Guide from Help and the command palette.
- In-product Beta Feedback and Release Notes guide from Help and the command palette.
- In-product Clean-Machine Smoke Checklist from Help and the command palette.
- In-product App Data and Uninstall Cleanup guide from Help and the command palette.
- In-product Release Hygiene and Antivirus Notes guide from Help and the command palette.
- Line-aware citations, bounded snippets, evidence labels, weak-evidence warnings, stale-source warnings, cited/uncited coverage, unverified/out-of-context citation diagnostics.
- Agent runtime with plan updates, bounded observations, emergency backend loop guard, tool dispatcher, audit, final fallback.
- Agent tools for context, search, problems, Git, tasks, artifacts, datasets, SQLite, documents, operations, safe file mutations, rollback, web fetch, and artifact regeneration.
- High-risk mutation tools use per-call approval modal and audit records.
- Settings stores and exposes task-aware model defaults for coding, backend, database, analytics, research, vision/screenshot, balanced reasoning, and fast-coding routes while preserving the global model as the active fallback.
- Ask mode exposes a model-route selector with global fallback and sends selected routes through the assistant service.
- Git AI diff summaries and commit drafts resolve the main coding route through the assistant service; assistant results and saved chat-answer artifacts preserve route metadata/warnings.
- Agent mode resolves the selected model route in the framework-free agent service, sizes context packs against that route, surfaces fallback warnings, and records model/route provenance in the persisted agent audit.
- Assistant/Agent route selection now includes an auto mode that infers data, SQL, document, image/screenshot, and code routes from pinned/selected context or prompt signals before falling back to the global model.
- Assistant context controls show the effective auto/manual/global model route and approximate pre-run context budget, with safe fallback sizing when context/reserve settings are misconfigured.

Planned/remaining:

- Deeper retrieval/ranking quality beyond deterministic coverage diagnostics.
- Wire route-aware model resolution into future dedicated Data, analytics, research, vision/screenshot, and document model workflows with explicit fallback/availability warnings.
- Better source diagnostics for partial, stale, weak, contradictory, or missing evidence.
- Complete tool-run provenance coverage for every generated output type.
- Agent approved shell beyond discovered safe tasks only after shell policy, audit, job routing, and UX are mature.
- Deeper GPU/provider diagnostics.

### 4.5 Git, Tasks, And IDE Operations

Already available:

- Manual Git status refresh.
- Project-tree Git badges.
- Grouped changed files.
- Unified, split, and diff-only views.
- Hunk-windowed diffs and large-file elision.
- File/hunk stage and unstage with explicit actions.
- AI diff summary and commit draft.
- Read-only Git history/blame tools.
- Safe task discovery and argument-based safe task execution.
- Job-backed task runs with logs, cancellation, retry, and report artifacts.

Planned/remaining:

- More task kinds as safe argv-based runners.
- Optional terminal-like UI only if safety and process model are clear.
- More complete Git review/commit workflows after shell and packaging foundations are stable.

### 4.6 Data And Analytics

Already available:

- Profiles CSV, TSV, JSON, NDJSON/JSONL, XLSX, Parquet metadata, logs.
- Bounded row query/filter/order/limit for local datasets.
- SELECT-only dataset SQL with persisted history/dependencies.
- SQL notebooks with SQL/chart cells, directives, execution, save/load, export, result tabs, lineage.
- SQLite workspace connector with schema, views, indexes, relationships, row counts, samples, saved query snippets, CSV/Markdown artifacts.
- External DB profile storage, protected credentials, test/inspect/query/cancel/history for PostgreSQL, MySQL/MariaDB, SQL Server, file SQLite, and guarded DuckDB builds.
- Read-only SQL guard, single-statement enforcement, mutation/export blocking, engine allowlist tuning.
- Error redaction for DSN, URLs, query strings, JSON password/token/API-key/auth-header fields.
- Data-grid copy, keyboard navigation, dense render policy, adaptive width sampling, row caching, large-grid tuning.
- SVG line/bar charts, dashboard visuals, KPI cards, artifact generation.

Planned/remaining:

- Dump import jobs using isolated temporary environments.
- Connector sync job model with cancellation, credentials, redaction, retry, audit, history.
- Google Analytics connector/export importer.
- Ads platform exported-data importer and later API connectors.
- CRM/contact connectors such as Eloqua, Mautic, and similar systems.
- Cross-source analysis workflows with row/query/connector/artifact citations.
- More DataGrip-like grid polish and schema navigation.

### 4.7 Artifacts, Documents, And Provenance

Already available:

- Artifact writer, metadata sidecars, browser, search, preview, compare, archive, delete, restore, open source, context pinning.
- Source lineage, source fingerprints, freshness warnings, metadata rows.
- Artifact lineage graph import/export.
- Regeneration for dataset summary, query CSV, SQL report, chart, dashboard, SQL notebook, SQLite query, document report, document brief, DOCX export, workspace scan, document extraction, operations runbook, comparison, saved chat-answer refresh, presentation outline, packaged presentation zip, PPTX deck.
- Markdown, CSV, SVG/chart, SQL result, task report, notebook report, document report, document brief, DOCX, scan report, operations runbook, comparison, chat answer, extracted document, presentation outline, packaged presentation, and PPTX deck artifacts.
- DOCX/PPTX package validation metadata, theme baseline, branded output styling, preview metadata.

Planned/remaining:

- Richer DOCX/PPTX template variants.
- Cross-suite Office compatibility smoke.
- Better visual design polish for generated documents/decks.
- Regeneration coverage for future richer variants.
- More complete tool-run provenance coverage across all generated output types.

### 4.8 Documents Studio And OCR

Already available:

- TXT, Markdown, PDF, DOCX, XLSX, HTML, XML preview/extraction baseline.
- Document extraction/report artifacts.
- Document brief and DOCX export baselines.

Planned/remaining:

- OCR/text extraction for images and scanned PDFs.
- OCR durable job pipeline with cancellation and output artifacts.
- Broader document-set analysis with citations, summaries, comparisons, and version workflows.
- Report and presentation generation from document sets and data sources.

### 4.9 Operations Studio

Already available:

- Read-only inspection for Dockerfile, Compose, env/config, scripts, logs.
- Compose topology summary from dependencies, ports, volumes.
- Compose config validation through explicit safe job.
- Operations runbook artifacts with source citations.

Planned/remaining:

- Container/image/log workflows only after approval policy, durable jobs, audit, and mitigation are mature.
- Strict separation between read-only inspection and mutating Docker/system actions.
- Better operations-specific diagnostics and runbook polish.

### 4.10 Security, Audit, Jobs, Diagnostics

Already available:

- Approval queue and modal flows.
- Time-boxed full-project access with visible status.
- Per-call approval for high-risk agent mutations.
- Path-root, traversal, symlink, ignored-state, and `.nexusdesk` protection.
- Rollback snapshots for practical workspace mutations.
- OS-protected secrets on Windows, macOS Keychain, Linux Secret Service/libsecret with unsupported-platform refusal.
- SQLite metadata for chats, approvals, artifacts, jobs, SQL, dataset dependencies, agent/tool runs.
- Wails-era compatibility import for chats, approvals, artifacts, tool runs, SQL, dependencies.
- Corrupt metadata recovery and metadata backup export.
- Search metadata recovery/export for explicit user-triggered workspace searches.
- Diagnostics for provider, metadata health, jobs, tasks, SQL, agent failures, runtime state.
- Shared slow-workflow job contract and explicit user-start enforcement.

Planned/remaining:

- Audit coverage for future connector sync, OCR, dump import, arbitrary shell, and Docker mutation workflows.
- Issue-report bundle with secret redaction and opt-in workspace content inclusion. Implemented as a Diagnostics export that writes a redacted ZIP with diagnostics text, activity tail, environment metadata, workspace-state file names, and no workspace file contents by default.
- Crash/hang checks for folder open, malformed files, corrupt metadata, missing providers, and canceled long work.

### 4.11 Packaging, Platform, Release

Already available:

- Windows Fyne/CGO build helpers.
- Windows icon stamping.
- Build metadata validation through ldflags.
- CI matrix for Windows, macOS, and Linux formatting, tests, static analysis, and Fyne build smoke.
- Native release manifest generation with artifact SHA256/size metadata.
- Platform support matrix documentation.

Planned/remaining:

- Signed Windows installer and release flow.
- macOS packaging, signing, notarization path.
- Linux package strategy, likely AppImage/deb/rpm or documented portable package.
- Installer/update/uninstall validation.
- Antivirus false-positive mitigation notes.
- Release notes, smoke checklist, beta feedback loop.

## 5. Claude Findings Integration

Claude's report from `claude-findings.md` was valuable because it highlighted concrete safety, reliability, UX, and architecture risks from static reading. The tracker now records those findings as integrated. This section keeps the full integration posture visible without deleting Claude's original report.

### 5.1 Critical Findings Status

- API key not configurable in UI: resolved by native protected API-key settings and bearer auth propagation.
- Agent could never run shell/task tools: resolved for approved safe discovered tasks; arbitrary shell remains intentionally planned/later.
- Symlink read-path traversal: resolved by mirroring write-path protections across read/search/context/preview paths.
- Metadata SQLite store reopened DB and reran schema on hot paths: resolved by opening/caching store and avoiding repeated schema/file rewrites.

### 5.2 High Findings Status

- Unbounded activity text growth: resolved with bounded activity text/line behavior.
- Workspace search one match per file: resolved with multiple matches/snippet improvements.
- Fixed model dropdown: resolved with configurable/free-form model/provider settings and probe validation.
- Ambient write approval only: resolved with per-call approval modal for high-risk tools.
- Preview hard cap without graceful truncation: resolved with truncated previews for supported text.
- Preview/context mismatch: resolved through shared extension/basename policy.
- Windows-1251 hard fallback: improved to avoid locale-bound silent corruption.
- Approval log re-saves all records: resolved by persisting only new records and serializing writes.
- Folder-only browse: resolved with single-file open/quick-open.
- Bottom panel overload: improved by consolidation/grouping; continue UI polish toward JetBrains-like tool-window model.
- Large `data_panel.go`: split work has landed; continue extracting remaining large shell controllers.

### 5.3 Medium/Polish Findings Status

Resolved or covered:

- Append rollback cleanup and close errors.
- Entry-cap visibility.
- Dirty tab close confirmation.
- Targeted tree refresh.
- Hunk-windowed diffs and large-file elision.
- Empty role/action guardrails.
- Settings-level LLM connection test.
- Persistence failure surfacing.
- Streaming cancellation.
- Extensible provider configuration.
- Safe context-budget fallback.
- Task-aware model routing for different app workflows and model capabilities.
- Scoped copy shortcuts.
- Match-centered search snippets.
- `cleanRel` normalization.
- Append safety sampling/encoding checks.
- Welcome open action and welcome handling.
- Release-ready About wording.
- Non-Windows support matrix and CI plan.
- Default LLM model strategy.
- Mutation-observation-driven verification.
- Expanded safe task execution whitelist.
- Argument-based process invocation for discovered tasks.
- Schema rewrite avoidance.
- Ask vs Agent prompt separation.
- Quick-open keyboard workflow.

### 5.4 Claude-Inspired Watchlist

These remain important even if the original individual findings were closed:

- Keep reducing `internal/ui/shell` orchestration complexity.
- Keep arbitrary shell out of production until approval, job routing, audit, redaction, and UX are mature.
- Keep Docker/system mutations out of production until safety design is complete.
- Keep platform packaging smoke honest, especially macOS Keychain and Linux Secret Service behavior.
- Keep large-session memory/performance tests in the validation set.
- Keep assistant claims grounded in recorded tool outcomes and citations.
- Keep docs synchronized so the project does not drift back into ambiguous migration status.

## 6. Production Gates

### Gate 0: Documentation And Source Of Truth

Goal: make every repeated development session start from the same plan.

Done when:

- `tracker.md`, `docs/13_PRODUCTION_READINESS.md`, `docs/15_WAILS_FEATURE_INVENTORY.md`, and this master plan agree on current status.
- Claude findings are integrated and not treated as forgotten external notes.
- UI direction explicitly references the JetBrains-like native workbench target.
- Repeated-development prompt points future LLM sessions at the same docs.

### Gate 1: Native Parity Beta

Goal: `nexus-app/` is the only app needed for normal daily use.

Must be true:

- Wails-only useful workflows are ported, replaced, dropped, or explicitly deferred.
- Editor baseline is accepted according to `docs/16_EDITOR_PARITY_STRATEGY.md`.
- External DB profiles and protected secrets work in native app.
- Assistant/source quality parity has deterministic citation, evidence, source, model, and artifact metadata coverage.
- UI no longer feels like a crowded migration prototype.
- `app-wails/` is frozen as reference-only.

### Gate 2: Safety And Reliability Beta

Goal: prove the local-first trust model.

Must be true:

- Every slow workflow is job-backed, cancelable, retryable, inspectable, and explicit-user-start only.
- Metadata recovery and backup export are reliable.
- Diagnostics explain provider, metadata, job, task, SQL, and agent failures.
- Audit coverage is complete for supported high-risk workflows.
- Folder open stays bounded and cheap under malformed or huge workspaces.

### Gate 3: Packaging And Platform Beta

Goal: a clean machine can install and smoke-test the app.

Must be true:

- Windows signed installer path works.
- macOS and Linux packaging paths are explicit and smoke-tested.
- CI validates formatting, tests, static analysis, build smoke, version metadata, release manifest, and diff hygiene.
- Installer/update/uninstall behavior is validated.
- Release artifacts are versioned, hashed, and documented.

### Gate 4: Private Beta

Goal: real users can try NexusDesk safely.

Must be true:

- First-run onboarding exists through the native Home readiness cockpit.
- Model/provider setup is understandable and visible before agent workflows start.
- Permissions, approvals, local data, rollback, and connector credentials are documented in-product and in docs.
- Issue-report bundle exists with redaction.
- Release notes and feedback loop exist through the Beta Feedback and Release Notes guide.

### Gate 5: Production Release

Goal: no known critical/high blockers, polished UI, stable architecture, measured performance, and clear platform story.

Must be true:

- No known critical/high safety, data-loss, packaging, or startup bugs.
- Main workflows have tests or documented manual smoke coverage.
- Performance budgets are measured on representative large workspaces.
- UI passes a professional polish pass against JetBrains-like direction.
- Wails reference is archived/frozen according to explicit owner decision.
- Documentation is current and user-facing docs match actual app behavior.

## 7. Detailed End-To-End Tracker

The checklist below is intentionally large. `tracker.md` should keep task-level execution details; this section keeps the whole roadmap visible.

### 7.1 P0: Source-Of-Truth And Planning

- [x] Preserve `app-wails/` as read-only reference.
- [x] Establish `nexus-app/` as active Fyne-native product.
- [x] Maintain Wails inventory with port/replace/drop/later decisions.
- [x] Maintain production readiness gates.
- [x] Integrate Claude findings into active tracker.
- [x] Add master production plan with JetBrains-like UI target.
- [x] Add Safe Agent User Guide and expose it in-product.
- [x] Add beta feedback and release-notes guide and expose it in-product.
- [x] Add clean-machine smoke checklist and expose it in-product.
- [x] Add app data and uninstall cleanup guide and expose it in-product.
- [ ] Keep `docs/12_PROJECT_REVIEW.md`, `docs/13_PRODUCTION_READINESS.md`, `docs/15_WAILS_FEATURE_INVENTORY.md`, and `tracker.md` synchronized after every major milestone.
- [x] Add package ownership documentation for every major `internal/` area.
- [x] Add contributor setup, coding standards, and ADR index.

### 7.2 P0: Architecture And Codebase Health

- [x] Keep main executable thin.
- [x] Keep services framework-free.
- [x] Keep Fyne imports out of domain/services.
- [x] Keep Wails/webview out of active app.
- [x] Add dependency guard/static import-boundary tests for forbidden Wails/webview imports, Fyne leakage outside presentation packages, and UI imports from services/domain.
- [x] Split initial large Data panel responsibilities.
- [ ] Continue extracting `internal/ui/shell` controllers by responsibility.
- [ ] Define shell state ownership boundaries: workspace, editor, assistant, data, artifacts, jobs, diagnostics.
- [x] Add package-level architecture docs for services and UI shell.
- [x] Add performance profiling harness for shell redraw, activity log, data grid, large search, and large artifacts.

### 7.3 P0: Safety And Trust

- [x] Rooted path safety for reads/writes/search/context.
- [x] Symlink protections on read and write paths.
- [x] `.nexusdesk` metadata protection.
- [x] Rollback snapshots for practical file mutations.
- [x] Approval queue and full-access policy.
- [x] Per-call approval modal for high-risk agent mutations.
- [x] OS-protected provider API keys and connector credentials.
- [x] Agent mutation claims grounded in observed tool outcomes.
- [x] Issue-report bundle with redaction and opt-in workspace content.
- [ ] Audit coverage for future OCR, connector sync, dump import, shell, and Docker mutation workflows.
- [ ] Secret-storage smoke on macOS and Linux packaging targets.
- [ ] Threat-model document for agent tools, connectors, filesystem, jobs, and generated artifacts.

### 7.4 P1: JetBrains-Like UI Polish

- [x] Define final native theme tokens: backgrounds, borders, text hierarchy, accent colors, warnings, success/error, selection, focus ring.
- [x] Define compact density and comfortable density modes.
- [x] Polish top menu to include File, Edit, View, Navigate, Code, Refactor, Run, Tools, Help where supported.
- [x] Polish top toolbar with workspace, branch, run/task, provider/model status, search, settings.
- [x] Convert crowded bottom tabs into grouped tool windows.
- [x] Add first left tool-window rail with consistent icons, labels, and shortcut hints for Project/Search/Problems/Git/Tasks/Jobs/Data/Artifacts/Operations/Diagnostics.
- [x] Add first right-side rail for assistant, sources, lineage, monitor, and inspector surfaces.
- [x] Add keyboard shortcuts for left/right tool-window rail actions.
- [x] Add active-state polish for left/right tool-window rails when opened by rail actions, shortcuts, or grouped bottom-tab selection.
- [ ] Add deeper assistant/data/artifact tool-window grouping and surface-specific polish.
- [x] Add status bar with workspace, provider, branch from last manual Git refresh, jobs, warnings, selected path, encoding, line ending, and app version.
- [x] Improve Settings into searchable grouped preferences with inline validation.
- [x] Improve onboarding Home tab with recent workspaces, model setup, safety explanation, and first actions.
- [x] Add first assistant panel run/status strip for Ask/Agent mode, model route, context budget, selected/pinned context, evidence/source counts, route warnings, and Agent job/tool safety.
- [x] Add first Assistant source actions to open or pin latest-answer sources for follow-up inspection.
- [ ] Improve assistant panel layout beyond the status strip and first source actions: richer source drilldowns, approvals, diagnostics grouping, footer evidence actions, and run history.
- [ ] Improve data-grid visual polish: header hierarchy, row density, selection, loading states, error states.
- [ ] Improve artifact browser polish: type badges, lineage, freshness, regeneration, comparison, preview.
- [ ] Improve diagnostics polish: health cards, actions, explanations, export issue bundle.
- [ ] Add visual regression/manual screenshot checklist for core windows.

### 7.5 P1: Editor And IDE Maturity

- [x] Native draft editor baseline.
- [x] Find/replace, formatting, breadcrumbs, split preview.
- [x] Syntax mirror and Document Map beta strategy.
- [x] Outline, go-to-symbol, local definition, workspace fallback, references.
- [x] Live draft diagnostics and Problems syntax scans.
- [ ] Continue document-map visual polish and keyboard navigation.
- [ ] Add stronger editor split groups and tab movement behavior.
- [ ] Add richer language action availability explanations.
- [ ] Spike editable inline syntax styling post-beta.
- [ ] Spike packaged LSP provider behind feature flag post-beta.
- [ ] Expand formatting and diagnostics for additional languages only with bounded behavior.

### 7.6 P1: Assistant And Source Quality

- [x] Source/model diagnostics.
- [x] Line-aware citations and snippets.
- [x] Evidence labels and source coverage.
- [x] Stale/weak/unverified citation warnings.
- [x] Chat-answer artifact metadata.
- [ ] Improve source retrieval/ranking beyond deterministic coverage.
- [ ] Add contradiction/ambiguity diagnostics when sources disagree.
- [ ] Add richer artifact-source cross-navigation.
- [x] Add better context budget visualization before a run.
- [x] Add task-aware model defaults in Settings for coding, backend, database, analytics, research, vision/screenshot, balanced reasoning, and fast-coding routes.
- [x] Add Ask-mode model-route selector with global fallback and assistant-service route metadata.
- [x] Wire task-aware model route resolution into Git AI diff summary/commit drafting with assistant-service fallback warnings and saved chat-answer route metadata.
- [x] Wire task-aware model route resolution into Agent workflows with selected-route context budgeting, fallback warnings, final-response metadata, and persisted agent-audit route/model provenance.
- [x] Add assistant/agent auto-routing by selected context for data, SQL, document, image/screenshot, and code tasks.
- [ ] Wire task-aware model route resolution into future dedicated Data, document, and vision/screenshot model workflows with fallback warnings and persisted route metadata.
- [ ] Add assistant answer quality smoke tests with fixture workspaces.
- [x] Add provider-specific guidance for common local model failures in Settings and Diagnostics, covering Ollama runtime startup, auth/base-URL failures, missing configured models, empty model lists, provider health, and unloaded runtime models.

### 7.7 P1: Artifacts And Documents

- [x] Artifact browser, metadata, lineage, freshness, compare, archive, restore.
- [x] Regeneration for major current artifact types.
- [x] DOCX/PPTX export baselines with validation and theme metadata.
- [ ] Richer DOCX template variants.
- [ ] Richer PPTX template variants.
- [ ] Cross-suite smoke: Microsoft Office, LibreOffice, Google Docs/Slides import where practical.
- [ ] Better document/deck visual design controls.
- [ ] Full provenance coverage for all current and future output types.
- [ ] OCR/scanned PDF/image extraction job pipeline.
- [ ] Document set comparison/version workflows.

### 7.8 P1: Data, Connectors, Operations

- [x] Local dataset profiling/query/SQL/notebooks/charts/dashboards.
- [x] SQLite browser/query/export.
- [x] External DB profiles, protected credentials, read-only queries, cancellation, redaction, history.
- [x] Operations read-only inspection and runbooks.
- [ ] Dump import job design and isolated execution.
- [ ] Connector sync job model.
- [ ] Google Analytics connector/export importer.
- [ ] Ads exported-data importer and later API connectors.
- [ ] CRM/contact connector importers.
- [ ] Cross-source analysis workflows with citations.
- [ ] DataGrip-like schema navigation and grid polish.
- [ ] Docker/container mutations only after mature approval, jobs, audit, and mitigation design.

### 7.9 P1: Jobs, Persistence, Diagnostics

- [x] SQLite metadata foundation.
- [x] Durable task/job monitor with progress, cancellation, retry, output.
- [x] Metadata backup and corrupt metadata recovery.
- [x] Compatibility import as async job with completion stamp and dedupe.
- [x] Slow-workflow job contract.
- [ ] Route every future slow workflow through jobs before UI exposure.
- [x] Add issue-report bundle.
- [x] Add crash/hang detector or startup recovery notes.
- [x] Add search index metadata and recovery/export.
- [x] Add job history retention controls and cleanup policy.
- [x] Add startup/folder-open performance timings to Diagnostics with over-budget warnings.

### 7.10 P1: Packaging, CI, Release

- [x] Cross-platform CI smoke for formatting, tests, static analysis, Fyne build.
- [x] Build metadata validation.
- [x] Release manifest with hashes/sizes.
- [x] Windows icon stamping.
- [ ] Signed Windows installer.
- [ ] macOS build, signing, notarization path.
- [ ] Linux package strategy and smoke.
- [ ] Installer/update/uninstall validation.
- [x] Antivirus false-positive mitigation notes.
- [x] App data path and uninstall/cleanup documentation.
- [x] Release notes and beta feedback loop.
- [x] Clean-machine smoke checklist for each supported OS.

### 7.11 P2: Extensibility And Community

- [ ] Stable service interfaces for contributed connectors and parsers.
- [ ] Plugin/MCP strategy after native core tools are stable.
- [x] Contributor setup, coding standards, and architecture decision records.
- [ ] Test fixture policy for community contributions.
- [ ] Extension security model before third-party code execution.

## 8. Performance Targets

Production should be measured, not guessed.

Initial targets:

- App launch: feels immediate on a normal developer workstation.
- Folder open: bounded and responsive for large repos; no external work starts automatically.
- Project tree: lazy loading prevents huge-folder freezes.
- Search: bounded result count, cancellation for future indexed/deep search, clear caps.
- Editor preview: large files truncate gracefully or show clear unsupported state.
- Data grid: large result sets use sampling, caps, caching, and responsive navigation.
- Activity log: bounded memory and redraw cost.
- Assistant: streaming cancel stops promptly; long agent runs are auditable and eventually job-backed where needed.
- Metadata: repeated chat/tool/job writes avoid reopening/rebootstrapping hot paths.

Performance work to add:

- [x] Large workspace smoke fixture baseline through `internal/services/perf`.
- [x] Large CSV/query/grid fixture baseline through `internal/services/perf`.
- [ ] Long chat/agent session fixture.
- [x] Large artifact directory fixture baseline through `internal/services/perf`.
- [x] Startup/folder-open timing logs in diagnostics.
- [ ] Memory snapshot or profiling recipe for release candidates.

## 9. Validation Strategy

Every logical milestone should run the appropriate validation set.

Code milestones:

```powershell
cd nexus-app
gofmt -w <changed-go-files>
go test ./...
go build .
git diff --check
Remove-Item .\nexusdesk.exe -ErrorAction SilentlyContinue
```

Docs-only milestones:

```powershell
git diff --check
```

Packaging milestones:

- Windows build helper.
- macOS build smoke.
- Linux build smoke.
- Release manifest validation.
- Installer/update/uninstall smoke.
- Clean-machine manual smoke checklist.

UI milestones:

- Manual screenshot pass against JetBrains-like shell reference.
- Open workspace, open file, search, assistant, data grid, artifacts, jobs, settings, diagnostics.
- Verify keyboard shortcuts and focus behavior.
- Verify empty states and error states.

Security/safety milestones:

- Path traversal and symlink tests.
- Approval denial/allow tests.
- Rollback integrity tests.
- Secret redaction tests.
- Connector read-only guard tests.
- Agent tool audit tests.

## 10. Definition Of Done For Production

NexusDesk is production-ready when:

- A non-developer can install, launch, open a workspace, configure a model, and use core workflows without source/build knowledge.
- The main UI looks like a coherent native professional workbench, not a migration prototype.
- No known critical/high data-loss, security, packaging, startup, or trust blockers remain.
- Wails useful-code parity is complete or intentionally documented as replaced/dropped/later.
- `app-wails/` is frozen or archived by explicit decision.
- Folder open is always safe and cheap.
- Supported file writes are approval-aware, audited, and rollback-backed where practical.
- Supported connector/database actions are read-only, bounded, redacted, cancelable, and audited.
- Supported agent tools are bounded, explicit, recorded, and approval-gated by risk.
- Slow workflows are jobs with progress, cancellation, retry, and outputs.
- Metadata recovery, backup, diagnostics, and issue-report bundle exist.
- Windows signed release path works, and macOS/Linux status is explicit and smoke-tested.
- Docs, tracker, release notes, and in-product messaging match actual behavior.

## 11. Repeated Development Prompt

Use this prompt for repeated production-development sessions:

```text
Continue NexusDesk development toward a production-ready Fyne-native app. Use app-wails/ only as reference and keep nexus-app/ as the active product. Start by reviewing tracker.md, docs/17_END_TO_END_PRODUCTION_PLAN.md, docs/13_PRODUCTION_READINESS.md, docs/15_WAILS_FEATURE_INVENTORY.md, and claude-findings.md. Pick the next highest-value unchecked milestone toward full Wails parity, JetBrains-like native UI polish, safety, performance, packaging, or production readiness. Implement one logical milestone end-to-end with focused tests and docs/tracker updates. Preserve architecture boundaries: no Wails/webview in the active app, services stay framework-free, Fyne stays in UI/app/theme packages, slow work uses jobs, and all risky actions keep approvals/audit/rollback/redaction. Validate with gofmt, go test ./..., go build ., git diff --check, remove generated binaries, then commit and push only when the milestone is clean. Report what changed, validation, commit hash, remaining blockers, and overall progress.
```
