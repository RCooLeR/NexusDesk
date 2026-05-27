# Nexus Augentic Studio Tracker

This tracker is now centered on the Fyne migration. The Wails/React application is preserved as `app-wails/` and remains the reference implementation until feature parity is intentionally restored in `nexus-app/`.

## Current Decision

We are moving away from Wails because the product wants to become a native, local-first IDE/data/document/operations studio, and the browser bridge has been creating recurring friction:

- generated bindings and bridge glue make large refactors noisy;
- Wails/webview lifecycle issues have caused blank or gray windows on folder open;
- React shell state grew too large while backend use cases also stayed too close to `app.go`;
- desktop behaviors such as dialogs, menus, layout, process handling, and long-running jobs should be first-class Go concerns;
- a native Fyne app keeps the whole product in one Go module and makes modular internal packages easier to enforce.

This is a breaking migration, not an incremental UI refresh.

## Repository State

- [x] `app-wails/` preserves the existing Wails application and all current migration source code.
- [x] `nexus-app/` is the new Fyne-native application root.
- [x] `nexus-app/main.go` is the only executable root file.
- [x] `nexus-app/go.mod` owns the new Fyne dependency graph.
- [x] `nexus-app/internal/app/` owns desktop lifecycle and window setup.
- [x] `nexus-app/internal/domain/` owns framework-free domain models.
- [x] `nexus-app/internal/services/` owns UI-independent application services.
- [x] `nexus-app/internal/ui/` owns Fyne views, layouts, widgets, and theme.
- [x] `.gitignore` covers Wails legacy build output and new Fyne build output.

## Verification

Current shell verification that does not require a Windows CGO compiler:

```powershell
cd nexus-app
$env:GOFLAGS='-mod=readonly'
go test ./internal/domain ./internal/services/... ./internal/ui/shell ./internal/ui/theme ./internal/brand
```

Full Fyne app run/build requires CGO and a C compiler on Windows:

```powershell
cd nexus-app
$env:CGO_ENABLED='1'
go build -o build\nexusdesk.exe .
```

Current build reality on this workstation:

- MSYS2 is installed at `C:\msys64`, and UCRT64 GCC is available at `C:\msys64\ucrt64\bin\gcc.exe`.
- `nexus-app/scripts/dev-env.ps1` configures the current PowerShell session with the MSYS2 compiler path, `CGO_ENABLED=1`, and default readonly module flags.
- `CGO_ENABLED=1 go build -o build\nexusdesk.exe .` succeeds when that helper is used.
- `go run .` has been smoke-verified by staying alive for 5 seconds under the configured CGO toolchain.
- `CGO_ENABLED=0 go build .` still fails because Fyne's OpenGL binding excludes all Go files without CGO.

Use the helper for native service tests, full builds, and local runs:

```powershell
cd nexus-app
.\scripts\dev-env.ps1 -Test
.\scripts\dev-env.ps1 -Build
.\scripts\dev-env.ps1 -Run
```

## Ordering Notes

Some tracker items are intentionally out of phase order because they depend on missing foundations:

- Phase 2 is functionally wired for first native Ask and Agent modes; deeper agent tool coverage is still pending.
- Phase 3 AI diff summary and commit drafting are pending until the native assistant service exists.
- Destructive hunk mutations remain pending until native approval policy is integrated into those specific Git actions.
- Durable persisted jobs and task-run records now have a SQLite foundation, and completed native task runs write Markdown report artifacts linked from those records.

## Migration Principles

- Keep Wails code as read-only reference unless explicitly patching a critical source bug.
- Port services by capability, not by copying giant files.
- Port Wails-era functionality first, then build new features on top of the native architecture.
- Avoid giant source files. Prefer packages, small files by responsibility, and tests near the code they protect.
- Design code so external contributors can understand ownership quickly.
- Domain and service packages must not import Fyne.
- UI packages may import services and domain models, but business rules stay in services.
- Long-running work must be represented as jobs before it is wired to UI events.
- Opening a workspace must never start Git, Docker, OCR, connector pulls, dump imports, model calls, or shell commands.
- Approval, rollback, audit, and path safety remain backend service responsibilities.
- Do not chase feature parity blindly; rebuild only the workflows that fit the native product direction.

## Active Native Porting Plan

The phases below are the active path. They track what has already been ported from Wails/React into `nexus-app/`, what is intentionally deferred because a dependency is missing, and what must happen before we resume broad new feature work.

## Phase 0: Migration Baseline

Goal: preserve the old app, establish the native shell, and make the new architecture explicit.

- [x] Rename `app/` to `app-wails/`.
- [x] Create `nexus-app/` with Fyne dependency and native app entrypoint.
- [x] Add first native shell layout: rail, toolbar, navigator, editor tabs, assistant panel, bottom activity/git/approval tabs.
- [x] Add first framework-free workspace domain model.
- [x] Add first lazy workspace listing service with entry cap, ignored folders, symlink skip, traversal protection, and unreadable tracking.
- [x] Document CGO/Fyne toolchain requirement.
- [x] Install/configure a Windows CGO compiler and verify `go run .`.
- [x] Add app icon and brand assets from `docs/brand/`.
- [x] Add native main menu: File, Edit, View, Navigate, Tools, Help.
- [x] Add keyboard shortcut registry for common IDE actions.

Exit criteria:

- [x] `nexus-app` opens as a native Fyne desktop window on the workstation.
- [x] The old Wails implementation is still available as reference.
- [x] New code follows the root-thin/internal-structured rule.

## Phase 1: Native Workbench Foundation

Goal: recreate the useful local project workbench without Wails or React.

- [x] Add folder open flow using native Fyne dialog.
- [x] Render first workspace tree from the service scan.
- [x] Add lazy child loading for large workspace trees.
- [x] Add first native file preview service with rooted text preview, UTF-8/UTF-8 BOM/UTF-16/Windows-1251 decoding, binary detection, traversal protection, and size cap.
- [x] Add first native editor tab lifecycle with close cleanup and same-file tab reuse.
- [x] Add UI-independent dirty/pinned tab state model with dirty close guards.
- [x] Add native pinned-tab controls and dirty markers in the tab header/editor chrome.
- [x] Add text/code editor widget decision: Fyne text editor first, Scintilla/LSP-backed editor later if needed.
- [x] Add first draft-only text editor with Source/Preview tabs, automatic dirty state, disabled Save, and Revert Draft.
- [x] Add Markdown source/rendered toggle.
- [x] Add first native image preview surface for capped PNG/JPEG/GIF/BMP/SVG/WebP files.
- [x] Add first native capped CSV/TSV table preview surface.
- [x] Add first native DOCX text extraction preview.
- [x] Add first native PDF text extraction preview surface.
- [x] Add first native safe write preview/apply/append/rollback service port for text and code files.
- [x] Wire draft editor Save through the native safe write service and rollback log.
- [x] Add first native file create/delete/rename/move/copy operation services with rooted validation and rollback records.
- [x] Add first selected-item navigator action menu for safe file operations, relative-path copy, and assistant-context selection.
- [x] Replace the selected-item navigator action menu with true tree-row secondary-click context menus.
- [x] Add first project-tree reveal/collapse controls and ignored-path visibility affordances.
- [x] Add first native workspace path/content search service and bottom result panel.
- [x] Add first native Problems service and bottom panel from the bounded marker/JSON scanner.

Exit criteria:

- [ ] A user can open a real project, browse files, preview content, and safely edit text/code files.

## Phase 2: Native Assistant And Agent

Goal: port the LLM and agent runtime without recreating the Wails bridge problems.

- [x] Add first native non-secret settings store for provider/model/context configuration.
- [x] Port OpenAI-compatible/Ollama client.
- [x] Add native provider/model settings page skeleton.
- [x] Add streaming assistant panel using Go channels/events instead of Wails events.
- [x] Port context-pack builder.
- [x] Add assistant context-pack UI affordances for pinning the workspace root, directories, and multiple files explicitly.
- [x] Add persisted native chat history and reload recent workspace turns into Ask mode.
- [x] Add first native chat search/history bottom panel backed by SQLite chat metadata.
- [x] Add first chat-history-to-Agent seed action with source path context pinning.
- [x] Add first unified native history navigation across chat, artifacts, jobs, and agent audit records.
- [x] Port agent runtime as an internal service, not a UI callback.
- [x] Unify registered tools and agent tools behind one dispatcher.
- [x] Add approval queue UI and full-access policy UI.
- [x] Add rollback browser for model-authored file mutations.
- [x] Add live activity tail with final-answer replacement behavior.
- [x] Add durable job and SQLite tool-run audit persistence for native agent runs.
- [x] Add native audit/history UI for persisted agent runs and tool runs.
- [x] Add agent-safe write/append/copy/move/delete/apply_patch tools gated by full-project access and rollback snapshots.

Exit criteria:

- [ ] The assistant can answer with selected workspace context and can request approved tools safely.

## Phase 3: Git And IDE Operations

Goal: make Workbench credible as an IDE-like surface.

- [x] Add first native Git status service under `nexus-app/internal/services/git`.
- [x] Add manual-only Git refresh panel.
- [x] Add changed-file tree grouped by directories.
- [x] Add first Workbench project-tree Git status badges from the last manual Git refresh.
- [x] Add first read-only Git file diff service and unified diff panel.
- [x] Add unified/split/diff-only diff views.
- [x] Add confirmed file-level staged/unstaged controls.
- [x] Add parsed hunk metadata and read-only hunk navigation.
- [x] Add hunk selection and approval-backed hunk stage/unstage actions.
- [ ] Add AI diff summary and commit draft once assistant service exists.
- [x] Add task discovery and safe task-run service.
- [x] Add first native task discovery/run panel.
- [x] Add native activity/job log for task output.

Exit criteria:

- [ ] Workbench can inspect repository state and run approved project tasks without command-window flashes.

## Phase 4: Data And Analytics

Goal: rebuild Data & Analytics as native data tooling, not a crowded web panel.

- [x] Port dataset profiling for CSV, TSV, JSON, NDJSON, XLSX, Parquet metadata, and logs.
- [x] Add first native sample-based data profiling slice for selected CSV, TSV, and JSON files.
- [x] Expand native profiling to NDJSON/JSONL, log line datasets, and lightweight Parquet footer metadata.
- [x] Add native bounded Parquet schema and row-group footer profiling without adding a heavy reader dependency.
- [x] Port first bounded row query/filter/order service for selected CSV, TSV, and JSON files.
- [x] Extend bounded row query/filter/order service to NDJSON/JSONL and log line datasets.
- [x] Add first SELECT-only native dataset SQL run over the selected dataset with persisted run/dependency metadata.
- [x] Promote SQL run/dependency history into the Data panel and unified History navigation.
- [x] Port first native SQL notebook model with per-dataset save/load, capped cells, and lineage metadata.
- [x] Add first native SQL notebook execution slice with multiline cell directives, SQL/chart cells, per-cell results, isolated failures, and SQL run lineage.
- [x] Add first native SQL notebook run Markdown artifact export with per-cell SQL, rows, plans, chart SVG, metadata, and source lineage.
- [x] Port first SQLite workspace connector browser with read-only schema, index, relationship, row-count, and capped-sample inspection.
- [ ] Port external DB profile storage and read-only query guards.
- [ ] Add native table/grid widget strategy.
- [x] Add chart preview/artifact generation.
- [x] Add automatic SVG line chart previews/artifacts for ordered date or numeric series.
- [x] Add richer dashboard SVG previews/artifacts with metrics, chart panel, and bounded-source notes.
- [ ] Add dump import job design before any Docker/database imports.

Exit criteria:

- [ ] A user can inspect local datasets and run bounded read-only analysis workflows.

## Phase 5: Artifacts, Documents, And Operations

Goal: restore generated-output workflows with provenance and native inspection.

- [ ] Port artifact writer, metadata, search, compare, archive, delete, and lineage.
- [x] Add native artifact browser for task-run report artifacts.
- [x] Expand native artifact browser to generic artifact metadata sidecars, metadata search, archive/delete actions, and task-report lineage.
- [x] Add first native document-set Markdown artifact writer from selected file/folder/project context with source lineage.
- [x] Add first same-kind artifact comparison surface with read-only generated-output diffs.
- [x] Add first artifact-to-assistant/agent context affordance so generated outputs can be cited in follow-up prompts.
- [x] Add first document-set artifact source actions for opening and pinning cited source files.
- [x] Add first artifact comparison report export with searchable metadata and source lineage.
- [x] Add first artifact source freshness warnings for missing or modified cited files.
- [x] Add source fingerprints to artifact metadata so freshness detects same-timestamp content changes.
- [x] Add first archive restore flow for generated artifacts with collision-safe restore paths.
- [x] Add document preview/extraction for Markdown, TXT, PDF, DOCX, XLSX, HTML/XML.
- [x] Add first native document extraction slice for Markdown, TXT, HTML, and XML source files with artifact export.
- [x] Extend native document extraction artifacts to DOCX and PDF preview text with PDF page metadata.
- [ ] Add presentation/report generation targets after artifact lineage is stable.
- [x] Add read-only operations scanners for Dockerfiles, Compose, env/config/logs.
- [x] Add Compose service topology summary from inspected Compose files.
- [x] Add first operations runbook artifact export from inspected Docker/Compose/env/config/log evidence.
- [ ] Add job-based OCR/document extraction before heavy parsing.

Exit criteria:

- [ ] Generated outputs are traceable to sources, chats, tool runs, and data queries.

## Phase 6: Job System And Persistence

Goal: make slow and durable workflows reliable.

- [x] Define first in-memory job model: id, kind, status, log tail, cancel, timestamps, and task output status.
- [x] Add SQLite primary metadata store in `nexus-app`.
- [x] Add durable SQLite repository for native jobs and task-run records.
- [x] Add task-run Markdown artifacts linked from persisted task-run records.
- [x] Add SQLite repository for native chat messages.
- [x] Add repositories for artifacts, SQL runs, and dataset dependencies.
- [x] Add first native SQLite artifact repository rows and history integration for explicit artifact writes, refreshes, archive/restore, and delete.
- [x] Add approval metadata repository coverage with JSON compatibility fallback.
- [x] Import Wails-era JSON chat, approval, artifact sidecar, and tool-run metadata into native SQLite on workspace open.
- [x] Migrate/import remaining Wails-era dataset SQL/dependency data from legacy SQLite metadata stores.
- [ ] Route long indexing, OCR, dump imports, connector pulls, report generation, and long agent runs through jobs.
- [x] Add native job monitor with cancel/retry/open-output actions.

Exit criteria:

- [ ] Slow work is cancelable, inspectable, and never blocks folder open.

## Phase 7: Retire Wails

Goal: remove the old app only after the Fyne app earns it.

- [ ] Identify any Wails-only features still missing in Fyne.
- [ ] Decide whether any React/Monaco code should be replaced, embedded, or permanently dropped.
- [ ] Freeze `app-wails` after feature parity milestone.
- [ ] Remove Wails build instructions from primary docs.
- [ ] Archive or delete `app-wails` after explicit approval.

Exit criteria:

- [ ] The default developer and user path is `nexus-app`.
- [ ] Wails is no longer needed for day-to-day development.

## Next Batch

1. Continue native SQL notebooks with visual cell controls, result tabs, run-history reuse/rerun, and richer explain output.
2. Port AI diff summary and commit drafting through the native assistant service.
3. Route long indexing, OCR, dump imports, connector pulls, report generation, and long agent runs through jobs.
4. Add dump import job design before any Docker/database imports.
5. Add SQLite connector cancellation, saved queries, and CSV/Markdown exports on top of the guarded query preview.

## Preserved Post-Port Backlog

The Fyne migration must not drop product ambition, but this section is intentionally at the end of the tracker. These are Wails-era planned or partial capabilities that still need to be ported, redesigned, or implemented after the native foundation is buildable and stable. We should port the needed Wails functionality first, then continue adding new features on the native architecture.

### Workbench / Code Studio

- [ ] Native IDE-style project tree with lazy loading, ignored-path controls, file status badges, context menus, reveal/collapse controls, and safe copy/move/delete/rename.
- [ ] Multi-tab editor with pinned tabs, dirty state, close guards, split editor groups, breadcrumbs, outline, minimap, find, format, and go-to-definition where available.
- [ ] Syntax highlighting strategy for common languages, Markdown, SQL, JSON/YAML/XML, Docker/Compose, logs, and config files.
- [ ] Markdown source/rendered toggle.
- [ ] Safe edit preview/apply/rollback for text, code, patches, appends, encoding changes, and allowed binary writes.
- [ ] Workspace search over paths, text, symbols, artifacts, and chat history.
- [ ] Problems panel for TODO/FIXME/HACK/BUG markers, merge conflicts, JSON errors, and later language diagnostics.
- [ ] Git status, branch, changed-file tree, staged/unstaged groups, file diff, split/unified/diff-only views, hunk actions, history, blame, AI review, test suggestions, commit draft, and PR draft.
- [x] Task discovery and approved task runs for npm, Go, and Docker Compose validation.

### Data & Analytics Studio

- [x] Dataset profiling for CSV, TSV, JSON, NDJSON, XLSX, Parquet metadata, and logs.
- [x] Bounded native Parquet footer decoding for schema columns and row-group summaries without scanning values.
- [x] Bounded filter/query/order/limit workflows for table-like CSV, TSV, JSON, NDJSON, XLSX, and log files.
- [x] First SELECT-only native SQL run over the selected dataset with one predicate, order, limit, projection, execution plan, and metadata persistence.
- [x] SQL run and dataset dependency history surfaced in Data & Analytics plus unified History.
- [ ] DuckDB-capable SQL over datasets when the optional CGO-backed build is available.
- [x] First native saved SQL notebook model with per-dataset JSON persistence, capped cells, Data panel save/load actions, and dataset dependency lineage.
- [x] First native SQL notebook execution flow with multiline `-- cell:` / `-- chart:` directives, saved multi-cell notebooks, per-cell SQL/chart execution, isolated failures, and SQL run lineage.
- [x] First native SQL notebook run Markdown artifact export with cell SQL, tabular results, logical plans, chart SVG snippets, searchable artifact metadata, and source lineage.
- [ ] Full SQL notebook shell with visual cell controls, result tabs, run history, reuse/rerun, and richer explain output.
- [x] First SQLite workspace database browser with schema, views, indexes, row counts, capped samples, and relationship hints.
- [x] First SQLite connector query preview with SELECT/WITH guard, single-statement validation, visible default row cap/timeout, SQL run metadata, dependency lineage, and read-only result rendering.
- [ ] SQLite connector query cancellation, saved queries, CSV/Markdown exports, and richer lineage actions.
- [ ] External database profiles for PostgreSQL, MySQL/MariaDB, SQL Server, DuckDB files, and future engines with protected credentials.
- [ ] Read-only SQL guard with strong comment/string handling, mutation blocking, caps, timeouts, cancellation, and redacted errors.
- [ ] Database dump import jobs using temporary isolated environments before any direct mutation workflows exist.
- [ ] Native table/grid strategy suitable for large result sets.
- [x] First SVG bar chart preview/artifact generation from bounded query results.
- [x] Line chart previews/artifacts for ordered date or numeric query results.
- [x] Richer dashboard SVG visuals with KPI cards, chart panel, and dataset notes.

### Analytics Connectors

- [ ] Google Analytics API connector and exported-data importer.
- [ ] Ads platform exported-data importer and later API connectors.
- [ ] CRM/contact-platform connectors for Eloqua, Mautic, and similar systems.
- [ ] Connector job model for sync, cancellation, credentials, redaction, audit, and retry.
- [ ] Cross-source analysis workflows that can cite rows, queries, connector runs, and generated artifacts.

### Documents Studio / Document Intelligence

- [x] Native preview and text extraction for TXT, Markdown, PDF, DOCX, XLSX, HTML, and XML.
- [ ] Native OCR/text extraction for images and broader office-like files.
- [ ] OCR job pipeline for scanned PDFs/images.
- [ ] Document set analysis with bounded context, source citations, summary artifacts, and lineage.
- [ ] Report and presentation generation from document sets and data sources.
- [ ] Comparison/version workflows for generated and source documents.

### AI Assistant And Agent

- [ ] Provider settings for Ollama/OpenAI-compatible endpoints, curated local model catalog, runtime context-window detection, response reserve, GPU diagnostics, and provider probes.
- [ ] Streaming chat with selected files/directories/project context, token-budgeted history, source citation, weak-evidence warnings, retries, and answer comparison.
- [ ] Local assistant memory and prompt profiles.
- [ ] Agent runtime with plan updates, bounded observations, model-driven tool calls, no frontend iteration cap, emergency backend loop guard, and final-answer fallback when context is exhausted.
- [ ] Unified tool registry and dispatcher for deterministic tools and model-requested tools.
- [ ] Agent tools for read context, read changed files, git diff/history/blame, problems, tasks, artifacts, artifact lineage, datasets, SQL, documents, operations files, web fetch, safe writes, patches, copy/move/delete, rollback, and approved shell.
- [ ] Live activity tail that shows the last one or two model/tool steps while preserving full trace in Activity.

### Artifacts And Provenance

- [ ] Markdown, CSV, SVG/chart, SQL result, scan report, task report, chat answer, and future presentation artifacts.
- [ ] Provenance sidecars with source files, chat IDs, tool run IDs, dataset/query IDs, and generated timestamps.
- [ ] Artifact browser with search, metadata, preview, compare, archive, delete, restore, and open-source navigation.
- [ ] Artifact lineage graph import/export and stale-source warnings.
- [ ] Regeneration workflows that reuse original source context and parameters.

### Operations Studio

- [x] Read-only Dockerfile, Compose, env/config, script, and log inspection.
- [x] Compose service topology summary from inspected service dependencies, port exposures, and named volumes.
- [x] Compose config validation through an explicit Operations action that runs the safe `docker compose config` task as a job.
- [ ] Container/image/log workflows only after approval policy and job model are mature.
- [x] First runbook artifacts and operations summaries with source citations.
- [ ] Strict separation between read-only inspection and mutating Docker/system actions.

### Security, Access, And Audit

- [x] Native approval queue and modal flows for high-risk actions.
- [x] Full-access project policy with clear scope, expiration, and visible status.
- [ ] Path-root enforcement, traversal protection, ignored-state protection, and `.nexusdesk` protection.
- [ ] Rollback snapshots for approved mutations where practical.
- [ ] OS-protected secrets on Windows, macOS Keychain, and Linux Secret Service/libsecret.
- [ ] Append-only audit records for approvals, tool runs, file changes, tasks, connector queries, jobs, and artifacts.
- [ ] Export/backup flows for local-first data.

### Jobs, Persistence, And Observability

- [ ] SQLite-first metadata store for chats, approvals, artifacts, tool runs, jobs, SQL runs, dataset dependencies, and search metadata.
- [x] JSON compatibility import from Wails-era workspaces for chat history, approvals, artifact sidecars, and tool-run logs.
- [x] Legacy Wails SQLite dataset SQL run and dataset dependency import into native SQLite metadata.
- [x] First durable job monitor with progress log tail, cancellation, retry from persisted task runs, and task-report output opening.
- [ ] Folder open remains cheap: no Git, Docker, OCR, connector pulls, dump imports, model calls, shell commands, or deep indexing.
- [ ] Diagnostics panel for app logs, provider status, GPU/model status, metadata health, and job history.

### Extensibility And Community

- [ ] Package ownership docs for every major `internal/` area.
- [ ] Contributor setup guide, coding standards, tests, and architecture decision records.
- [ ] Plugin/MCP strategy after native core tools are stable.
- [ ] Stable service interfaces for community-contributed connectors and document parsers.
- [ ] CI matrix for Windows first, then Linux/macOS once platform secrets and Fyne builds are ready.
