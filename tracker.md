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

## Current Review

Latest full project review: `docs/12_PROJECT_REVIEW.md`.
Production-readiness gates: `docs/13_PRODUCTION_READINESS.md`.
Wails feature inventory and retirement blockers: `docs/15_WAILS_FEATURE_INVENTORY.md`.
Native editor parity decision: `docs/16_EDITOR_PARITY_STRATEGY.md`.
End-to-end production master plan and JetBrains-like UI target: `docs/17_END_TO_END_PRODUCTION_PLAN.md`.
Safe agent user guide: `docs/18_SAFE_AGENT_USER_GUIDE.md`.
Beta feedback and release notes guide: `docs/19_BETA_FEEDBACK_AND_RELEASE_NOTES.md`.
Clean-machine smoke checklist: `docs/20_CLEAN_MACHINE_SMOKE_CHECKLIST.md`.
App data and uninstall cleanup guide: `docs/21_APP_DATA_AND_UNINSTALL_CLEANUP.md`.
Release hygiene and antivirus guidance: `docs/22_RELEASE_HYGIENE_AND_ANTIVIRUS.md`.

Summary:

- The Fyne migration remains the correct direction and `nexus-app/` is the active product.
- Current estimate: Fyne-native migration is roughly 98% complete, Wails useful-code parity is roughly 97%, Native Parity Beta readiness is roughly 96%, and overall production readiness is roughly 95%.
- The architecture is healthy: thin executable root, framework-free domain/services, Fyne-only UI packages, explicit approvals, safe workspace mutation boundaries, manual Git/Docker actions, durable metadata, and local-first safety rules.
- The biggest remaining architectural risk is UI/orchestration complexity in `internal/ui/shell`; future UI work should keep extracting focused panels, controllers, and service-owned behavior.
- The highest-priority unfinished work is migration and production readiness, not new top-level studios: applying durable slow-job routing to the remaining slow workflows, richer DOCX/PPTX template variants and cross-suite smoke beyond current native export/theme/validation baselines, deeper assistant retrieval quality beyond deterministic source coverage, signed packaging/installer validation, and native UI polish.
- Final UI direction is a professional JetBrains-like native workbench: top menu/toolbar, left project/tool rail, central tabbed editor, right integrated assistant, grouped bottom tool windows, compact dark theme, strong keyboard workflows, DataGrip-style data surfaces, and trust-building settings/diagnostics.
- `app-wails/` should remain as reference until the remaining native parity blockers are completed or explicitly moved out of Native Parity Beta.

Production direction:

- Gate 1 is Native Parity Beta: editor parity, external DB profiles, protected secrets, assistant quality, Wails-only inventory, and native UI cleanup.
- Gate 2 is Safety And Reliability Beta: durable jobs, metadata recovery/export, diagnostics, audit coverage, and failure recovery.
- Gate 3 is Packaging And Platform Beta: repeatable signed Windows builds, CI, visual/manual smoke, platform support matrix, and release hygiene.
- Gate 4 is Private Beta: onboarding/readiness, issue-report bundles, safe-agent guidance, and user feedback/release notes.

Immediate execution order:

- Keep `docs/17_END_TO_END_PRODUCTION_PLAN.md` current as the product north star for all repeated development sessions.
- Validate macOS Keychain and Linux Secret Service/libsecret behavior during platform packaging smoke.
- Apply the durable slow-job contract to OCR, dump imports, connector pulls, long indexing, report generation, and long agent runs as those workflows are implemented.
- Keep the documented native editor parity strategy visible in Language Actions and continue post-beta LSP/inline-styling spikes without blocking Native Parity Beta.
- Polish richer generated document/deck exports beyond native DOCX/PPTX baselines, including template variants, cross-suite compatibility smoke, and richer visual design.
- Build signed release packaging and installer/update validation.
- Continue the focused UI polish pass on empty states, settings, diagnostics, and workflow hierarchy now that the first Home readiness/onboarding cockpit and Safe Agent Guide exist.

## Repository State

- [x] `app-wails/` preserves the existing Wails application and all current migration source code.
- [x] `nexus-app/` is the new Fyne-native application root.
- [x] `docs/17_END_TO_END_PRODUCTION_PLAN.md` documents the full end-to-end production plan, Claude findings integration, and JetBrains-like UI target.
- [x] `docs/18_SAFE_AGENT_USER_GUIDE.md` documents safe agent use, approvals, rollbacks, local data, connector credentials, jobs, diagnostics, and issue-report expectations.
- [x] `docs/19_BETA_FEEDBACK_AND_RELEASE_NOTES.md` documents private-beta feedback, release-note expectations, redacted reports, triage labels, and closeout rules.
- [x] `docs/20_CLEAN_MACHINE_SMOKE_CHECKLIST.md` documents clean-machine release-candidate smoke coverage for install, launch, workspace, editor, assistant, data, artifacts, jobs, diagnostics, platform-specific behavior, upgrade, uninstall, and cleanup.
- [x] `docs/21_APP_DATA_AND_UNINSTALL_CLEANUP.md` documents global config paths, protected secret storage, workspace `.nexusdesk/` state, normal uninstall expectations, full manual reset, and upgrade/backup guidance.
- [x] `docs/22_RELEASE_HYGIENE_AND_ANTIVIRUS.md` documents release artifact discipline, signing/trust expectations, antivirus false-positive triage, release-note requirements, and do-not-ship rules.
- [x] `nexus-app/main.go` is the only executable root file.
- [x] `nexus-app/go.mod` owns the new Fyne dependency graph.
- [x] `nexus-app/internal/app/` owns desktop lifecycle and window setup.
- [x] `nexus-app/internal/domain/` owns framework-free domain models.
- [x] `nexus-app/internal/services/` owns UI-independent application services.
- [x] `nexus-app/internal/ui/` owns Fyne views, layouts, widgets, and theme.
- [x] `nexus-app/internal/architecture` guards import boundaries so Wails/webview cannot return and Fyne/UI cannot leak into services/domain.
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
.\scripts\dev-env.ps1 -Build
```

Current build reality on this workstation:

- MSYS2 is installed at `C:\msys64`, and UCRT64 GCC is available at `C:\msys64\ucrt64\bin\gcc.exe`.
- `nexus-app/scripts/dev-env.ps1` configures the current PowerShell session with the MSYS2 compiler path, `CGO_ENABLED=1`, and default readonly module flags.
- `nexus-app/scripts/build-windows-icon.ps1` generates `resource_windows.syso` from the approved brand PNG so the Windows `.exe` is stamped with the app icon.
- `.\scripts\dev-env.ps1 -Build` succeeds and writes `build\nexusdesk.exe` with the executable icon resource.
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
- [x] Stamp the Windows executable icon resource during native builds.
- [x] Add native main menu: File, Edit, View, Navigate, Tools, Help.
- [x] Add keyboard shortcut registry for common IDE actions.
- [x] Add first resizable native shell layout with a labeled product rail, resizable assistant/sidebar split, resizable bottom workbench split, and grouped Data actions.

Exit criteria:

- [x] `nexus-app` opens as a native Fyne desktop window on the workstation.
- [x] The old Wails implementation is still available as reference.
- [x] New code follows the root-thin/internal-structured rule.

## Phase 1: Native Workbench Foundation

Goal: recreate the useful local project workbench without Wails or React.

- [x] Add folder open flow using native Fyne dialog.
- [x] Add native recent-workspace persistence with Home tab open/remove/clear actions.
- [x] Add first-run Home readiness cockpit for workspace open, model/provider setup, API-key state, native build toolchain status, local safety posture, and quick first actions.
- [x] Render first workspace tree from the service scan.
- [x] Add lazy child loading for large workspace trees.
- [x] Add first native file preview service with rooted text preview, UTF-8/UTF-8 BOM/UTF-16/Windows-1251 decoding, binary detection, traversal protection, and size cap.
- [x] Add first native editor tab lifecycle with close cleanup and same-file tab reuse.
- [x] Add UI-independent dirty/pinned tab state model with dirty close guards.
- [x] Add native pinned-tab controls and dirty markers in the tab header/editor chrome.
- [x] Add text/code editor widget decision: Fyne text editor first, Scintilla/LSP-backed editor later if needed.
- [x] Add first draft-only text editor with Source/Preview tabs, automatic dirty state, disabled Save, and Revert Draft.
- [x] Add Markdown source/rendered toggle.
- [x] Add first native lightweight syntax strategy: Wails-derived language detection, bounded token analysis, and a Fyne Syntax tab for common code/config languages.
- [x] Add native read-only highlighted syntax preview with Wails/Monaco-inspired token colors and bounded Fyne `TextGrid` styling.
- [x] Add cursor-aware native syntax mirror with active-line highlighting and token/symbol status for the current editor draft.
- [x] Add live native draft diagnostics tab for unsaved marker, merge-conflict, and JSON/Go/YAML/TOML/XML parser issues.
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
- [x] Add native Problems language diagnostics for Go, YAML, TOML, and XML syntax using bounded preview-safe reads.

Exit criteria:

- [x] A user can open a real project, browse files, preview content, and safely edit text/code files.

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

- [x] The assistant can answer with selected workspace context and can request approved tools safely.

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
- [x] Add selected-file AI diff summary and commit draft through the native assistant service.
- [x] Add task discovery and safe task-run service.
- [x] Add first native task discovery/run panel.
- [x] Add native activity/job log for task output.

Exit criteria:

- [x] Workbench can inspect repository state and run approved project tasks without command-window flashes.

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
- [x] Add first native SQL history reuse/rerun actions for selected dataset or SQLite sources.
- [x] Extend native SQL history reuse/rerun actions with connector-source fallback and connector rerun routing by profile id.
- [x] Add first native Data panel result tabs for notebook summary, rows, plan, and charts.
- [x] Add first native visual notebook controls for inserting SQL and chart cell templates.
- [x] Port first SQLite workspace connector browser with read-only schema, index, relationship, row-count, and capped-sample inspection.
- [x] Add native SQLite connector saved query snippets plus CSV and Markdown query-result artifacts with lineage metadata.
- [x] Port external DB profile storage and read-only query guards.
- [x] Add first native external connector profile workflows in Data panel: list/save/delete profiles, read-only SQL validation, profile test, profile inspection, run query, and query cancellation for PostgreSQL, MySQL/MariaDB, SQL Server, and file-based SQLite profiles.
- [x] Persist native external connector query runs into SQL history metadata with connector lineage dependencies.
- [x] Enforce workspace-scoped external connector profile filtering in native Data UI while retaining optional global profiles.
- [x] Add DuckDB profile test/query/inspection code paths with explicit driver-enabled-build guard and clear unsupported-build errors.
- [x] Add first native table/grid widget strategy for query-result rows in Data panel (dataset, SQLite, connector).
- [x] Add chart preview/artifact generation.
- [x] Add automatic SVG line chart previews/artifacts for ordered date or numeric series.
- [x] Add richer dashboard SVG previews/artifacts with metrics, chart panel, and bounded-source notes.
- [ ] Add dump import job design before any Docker/database imports.

Exit criteria:

- [x] A user can inspect local datasets and run bounded read-only analysis workflows.

## Phase 5: Artifacts, Documents, And Operations

Goal: restore generated-output workflows with provenance and native inspection.

- [x] Port artifact writer, metadata, search, compare, archive, delete, and lineage.
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
- [x] Port native workspace scan report artifacts from Wails behind an explicit cancellable job.
- [x] Add artifact-side regeneration for native workspace scan reports, document reports, document-extraction artifacts, operations runbooks, artifact comparison reports, and saved chat-answer refresh artifacts through durable Artifacts-panel jobs.
- [x] Add document preview/extraction for Markdown, TXT, PDF, DOCX, XLSX, HTML/XML.
- [x] Add first native document extraction slice for Markdown, TXT, HTML, and XML source files with artifact export.
- [x] Extend native document extraction artifacts to DOCX and PDF preview text with PDF page metadata.
- [x] Add first native presentation outline artifact target from generated reports/chat/runbook artifacts with source lineage.
- [x] Add first native packaged presentation zip export from presentation-outline artifacts with manifest, slide JSON/Markdown, README, source lineage, preview metadata, UI job routing, and agent/UI regeneration.
- [x] Add first native document brief artifact target from report-like artifacts with source lineage, Artifacts-panel job routing, and agent/UI regeneration.
- [x] Add first native DOCX document export artifact target from document briefs with OpenXML package output, preview metadata, source lineage, Artifacts-panel job routing, and agent/UI regeneration.
- [x] Add first native PPTX presentation deck export target from packaged presentation artifacts with OpenXML package output, preview metadata, source lineage, Artifacts-panel job routing, and agent/UI regeneration.
- [x] Add first release-grade Office package validation for native DOCX/PPTX exports: required part checks, XML well-formedness checks, PPTX slide relationship coverage, persisted validation metadata, and Artifacts preview status.
- [x] Add first native Office theme baseline for generated DOCX/PPTX exports: template/theme metadata, professional DOCX font/color styles, branded PPTX background/accent/footer treatment, and preview visibility.
- [x] Add read-only operations scanners for Dockerfiles, Compose, env/config/logs.
- [x] Add Compose service topology summary from inspected Compose files.
- [x] Add first operations runbook artifact export from inspected Docker/Compose/env/config/log evidence.
- [ ] Add job-based OCR/document extraction before heavy parsing.

Exit criteria:

- [ ] Generated outputs are traceable to sources, chats, tool runs, and data queries.

## Phase 6: Job System And Persistence

Goal: make slow and durable workflows reliable.

- [x] Define first in-memory job model: id, kind, status, log tail, cancel, timestamps, and task output status.
- [x] Define durable slow-workflow contract for OCR, dump imports, connector pulls, long indexing, report generation, long agent runs, and packaged exports, including explicit-start and no-workspace-open guardrails.
- [x] Add SQLite primary metadata store in `nexus-app`.
- [x] Add durable SQLite repository for native jobs and task-run records.
- [x] Add task-run Markdown artifacts linked from persisted task-run records.
- [x] Add SQLite repository for native chat messages.
- [x] Add repositories for artifacts, SQL runs, and dataset dependencies.
- [x] Add first native SQLite artifact repository rows and history integration for explicit artifact writes, refreshes, archive/restore, and delete.
- [x] Add approval metadata repository coverage with JSON compatibility fallback.
- [x] Import Wails-era JSON chat, approval, artifact sidecar, and tool-run metadata into native SQLite on workspace open.
- [x] Migrate/import remaining Wails-era dataset SQL/dependency data from legacy SQLite metadata stores.
- [ ] Apply durable job routing to concrete long indexing, OCR, dump import, connector pull, report generation, and long agent run implementations.
- [x] Add native job monitor with cancel/retry/open-output actions.

Exit criteria:

- [ ] Slow work is cancelable, inspectable, and never blocks folder open.

## Phase 7: Retire Wails

Goal: remove the old app only after the Fyne app earns it.

- [x] Identify any Wails-only features still missing in Fyne.
- [x] Decide whether any React/Monaco code should be replaced, embedded, or permanently dropped.
- [ ] Freeze `app-wails` after feature parity milestone.
- [ ] Remove Wails build instructions from primary docs.
- [ ] Archive or delete `app-wails` after explicit approval.

Exit criteria:

- [ ] The default developer and user path is `nexus-app`.
- [ ] Wails is no longer needed for day-to-day development.

## Next Batch

1. Use the Wails inventory to close remaining Native Parity blockers: deeper assistant retrieval/ranking quality, richer generated document/full deck outputs, and final UI polish.
2. Finish native editor/UI parity: richer inline syntax styling, future LSP/deeper cross-file language actions, and less cramped native panels.
3. Continue platform validation for protected secret storage now that Windows DPAPI, macOS Keychain, and Linux Secret Service/libsecret command-backed backends exist.
4. Apply the durable slow-workflow contract to concrete long indexing, OCR, dump imports, connector pulls, report generation, and long agent run implementations.
5. Add dump import job design before any Docker/database import execution.
6. Continue Diagnostics hardening with deeper provider-specific runtime/GPU checks and guided remediation workflows.
7. Keep cleaning Wails-era documentation wording so active docs clearly describe `nexus-app/` behavior and mark `app-wails/` as reference history.

## Production Readiness Checklist

### Gate 1: Native Parity Beta

- [x] Wails-only feature inventory with `port` / `replace` / `drop` / `later` decisions.
- [x] IDE-grade editor baseline for Native Parity Beta: native highlighted Syntax mirror, Document Map, outline/symbol navigation, local/workspace definition fallback, references, formatting, draft diagnostics, and explicit Language Actions decision are accepted as the beta replacement for Monaco inline/minimap behavior.
- [ ] Post-beta editor enhancements: active-editor inline styling and future LSP/deeper cross-file language actions.
- [ ] Native external database profiles for PostgreSQL, MySQL/MariaDB, SQL Server, and DuckDB with read-only guards.
- [x] Native protected secret storage for Windows DPAPI, macOS Keychain, and Linux Secret Service/libsecret, with explicit refusal on unsupported platforms.
- [x] Assistant quality parity: line-aware citation refs beyond file-level sources in native answer footers and saved chat-answer artifacts.
- [x] Assistant Wails parity slice: profile/memory store, active prompt profile injection, weak-evidence warning, retry/compare, and save-latest-answer `chat-answer` artifacts.
- [x] Assistant stale-source parity slice: chat context paths persist in native metadata and chat history warns when cited sources changed or disappeared.
- [x] Assistant source-quality UX slice: native answer footers and saved `chat-answer` artifacts classify evidence as weak, source-backed, or line-cited with metadata.
- [x] Assistant citation coverage diagnostics: answer footers and saved `chat-answer` artifacts now surface line refs that fall outside the attached source set instead of silently dropping them.
- [ ] Native UI cleanup pass across Workbench, Data, Artifacts, Settings, assistant, and bottom panels.

### Gate 2: Safety And Reliability Beta

- [x] Durable slow-workflow contract for OCR, dump imports, connector pulls, long indexing, report generation, long agent runs, and packaged exports.
- [ ] Concrete durable job routing for OCR, dump imports, connector pulls, long indexing, report generation, and long agent runs as those workflows are implemented.
- [x] Metadata recovery/export path for `.nexusdesk/metadata`.
- [x] Backup/export flow for local-first workspace state.
- [x] Diagnostics panel for provider status, metadata health, job history, app logs, GPU/model runtime, and recent failures.
- [ ] Audit coverage for connector jobs, OCR, dump imports, Docker mutations, shell tools, and future high-risk operations.
- [ ] Failure tests for folder open, malformed files, corrupt metadata, missing providers, and canceled long work.

### Gate 3: Packaging And Platform Beta

- [ ] Repeatable Windows build pipeline with icon, version metadata, installer/update plan, and code-signing path.
- [x] First native CI smoke matrix for Windows, macOS, and Linux with gofmt, `go test ./...`, `go vet ./...`, CGO/Fyne build, and `git diff --check`.
- [x] Add ldflag-backed app version/build metadata validation to native CI builds and About metadata.
- [x] Add release manifest generation/validation to native CI builds with artifact name, size, SHA256, platform, version, commit, and build date metadata.
- [ ] Expand CI into signed release packaging and installer/update validation.
- [x] Windows visual/manual smoke checklist for every main surface.
- [x] Linux/macOS build investigation and explicit support matrix.
- [x] Antivirus false-positive mitigation notes and release-build hygiene.
- [x] App data path and uninstall/cleanup documentation.

### Gate 4: Private Beta

- [x] Onboarding flow for workspace open, model setup, permissions, and local data policy.
- [x] First-run diagnostics for missing model endpoint, missing toolchain, and unavailable provider.
- [x] Redacted issue-report bundle that excludes workspace contents unless explicitly included.
- [x] User docs for safe agent use, approvals, rollbacks, local data, and connector credentials.
- [x] Beta feedback loop and release notes.

## Preserved Post-Port Backlog

The Fyne migration must not drop product ambition, but this section is intentionally at the end of the tracker. These are Wails-era planned or partial capabilities that still need to be ported, redesigned, or implemented after the native foundation is buildable and stable. We should port the needed Wails functionality first, then continue adding new features on the native architecture.

### Workbench / Code Studio

- [x] Native project tree baseline with lazy loading, ignored-path controls, file status badges, secondary-click context menus, reveal/collapse controls, safe copy/move/delete/rename, and file clipboard paste through copy/move services.
- [x] Native folder create flow with rooted validation, metadata guard, rollback removal for empty created folders, quick action, and tree context menu action.
- [ ] IDE-grade project tree polish with richer density, better icons/badges, keyboard navigation, and broader context actions.
- [x] First native multi-tab editor state with pinned ordering, dirty markers, safe save, revert, and explicit discard confirmation when closing modified tabs.
- [x] First native text-editor find/replace flow with Edit menu actions, keyboard shortcuts, next-match navigation, and replace-next/replace-all actions.
- [x] Native find match-count parity slice with Wails-style live match counts in the Fyne Find / Replace dialog.
- [x] Native editor outline parity slice with Wails-derived symbol rules for Markdown, Go, JavaScript/TypeScript, CSS, JSON, and YAML plus Fyne cursor navigation.
- [x] Native editor breadcrumb and save-encoding parity slice with Wails-derived breadcrumb rules, clickable Fyne breadcrumb navigation, and explicit UTF-8/UTF-16/Windows encoding selection for safe writes.
- [x] Native format-document parity slice with deterministic Go and JSON draft formatting from the Fyne editor.
- [x] Native split editor parity slice with Wails-derived secondary tab resolution and Fyne read-only secondary preview selection.
- [x] Native symbol navigation parity slice with a searchable Fyne go-to-symbol dialog backed by the Wails-derived outline rules.
- [x] Native command palette baseline with searchable workbench/editor/navigation actions and `Ctrl/Cmd+Shift+P`.
- [x] Native broader-format parity slice with safe Markdown/config/SQL/Dockerfile/text plus recognized code/markup whitespace formatting and JSON workspace formatting.
- [x] Native minimap replacement slice with a jumpable Fyne Document Map for symbols, TODO/FIXME/HACK/BUG markers, merge conflicts, and long-file anchors.
- [x] Native local go-to-definition parity slice with cursor-symbol resolution against the Wails-derived outline rules.
- [x] Native bounded workspace go-to-definition fallback for unresolved editor symbols using workspace search, preview-safe reads, and outline matching.
- [x] Native bounded find-references language action for cursor symbols using workspace search and jumpable preview-safe matches.
- [x] Native editor language-action readiness slice that surfaces available formatting, highlighting, outline, definition/reference fallback, and future LSP status per active file.
- [x] Native editor parity strategy decision surfaced in Language Actions and documented as the Native Parity Beta acceptance bar.
- [ ] Multi-tab editor polish with future LSP-backed cross-file go-to-definition where native language support is available.
- [x] First native lightweight syntax strategy for common languages, Markdown, SQL, JSON/YAML/XML/HTML, Docker/Compose, logs, and config files.
- [x] Native read-only highlighted syntax preview with Wails/Monaco-inspired token colors, line numbers, and bounded token styling.
- [x] Native Problems XML diagnostics for well-formed document/config markup.
- [ ] Post-beta rich inline syntax styling in the active editor widget and future LSP-backed diagnostics/actions.
- [x] Markdown source/rendered toggle.
- [x] Safe edit preview/apply/rollback for text, code, patches, appends, encoding-aware writes, and agent-safe mutation tools.
- [x] Workspace search over paths, previewable text, artifacts, chat history, and lightweight regex content matches.
- [x] Expand workspace search to return multiple content matches per file with bounded per-file caps.
- [x] Problems panel for TODO/FIXME/HACK/BUG markers, merge conflicts, JSON errors, and Go/YAML/TOML syntax diagnostics.
- [ ] Problems panel deeper semantic diagnostics beyond local syntax scans.
- [x] Git status, branch, changed-file tree, staged/unstaged groups, file diff, split/unified/diff-only views, hunk actions, AI diff summary, and commit draft.
- [x] Port native read-only Git history and blame service/UI from Wails.
- [x] Port native read-only Git history/blame agent context tools from Wails.
- [ ] Broader AI review, test suggestions, PR draft, and destructive revert/discard actions.
- [x] Task discovery and approved task runs for npm, Go, Python pytest, Cargo, and Docker Compose validation.

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
- [x] First native SQL history reuse/rerun actions for selected dataset or SQLite sources.
- [x] First native Data panel result tabs for notebook summary, rows, plan, and charts.
- [x] First native visual notebook controls for inserting SQL and chart cell templates without memorizing directives.
- [x] First native SQL notebook cell selector with move up/down, delete, editor outline refresh, and save-over-loaded-notebook identity.
- [ ] Full SQL notebook shell with richer explain output, notebook-level result navigation, per-cell run/export, and a dedicated editor surface beyond directive text.
- [x] First SQLite workspace database browser with schema, views, indexes, row counts, capped samples, and relationship hints.
- [x] First SQLite connector query preview with SELECT/WITH guard, single-statement validation, visible default row cap/timeout, SQL run metadata, dependency lineage, and read-only result rendering.
- [x] SQLite connector saved queries plus CSV/Markdown result exports.
- [x] SQLite connector query cancellation from the native Data panel using context-aware read-only query execution.
- [x] Native dataset artifact rebuild baseline for summary, query CSV, SQL report, chart, dashboard, SQL notebook, and SQLite query artifact dependency records.
- [ ] Richer SQLite connector lineage actions for opening sources and viewing dependent artifacts.
- [ ] External database profiles for PostgreSQL, MySQL/MariaDB, SQL Server, DuckDB files, and future engines with protected credentials plus live native execution paths.
- [x] Read-only SQL guard with comment/string-aware parsing, single-statement enforcement, and mutation/export blocking across live external connector execution.
- [x] Add connector runtime error redaction baseline for DSN/URL/query-string/JSON password fields before UI display.
- [x] Continue SQL guard hardening with engine-specific allowlist tuning.
- [x] Expand connector runtime error sanitization coverage for token/api-key/auth-header edge cases.
- [ ] Database dump import jobs using temporary isolated environments before any direct mutation workflows exist.
- [x] Add first grid copy affordances in Data panel (copy selected cell/row as clipboard text/TSV).
- [x] Add Data-grid keyboard/menu copy command (`Ctrl/Cmd+C` + Edit menu) scoped to active Data tab selection.
- [x] Add Data-grid keyboard navigation shortcuts (`Alt+Arrow`) with clamped selection and auto-scroll in the Rows tab.
- [x] Remove nested Rows-tab scroll wrapping, keep header row sticky, and add `Alt+PgUp/PgDn` step navigation for larger result sets.
- [x] Add Data-grid boundary navigation shortcuts (`Alt+Home/End`, `Alt+Shift+Home/End`) for fast row/column jumps in large tables.
- [x] Tune grid column-width estimation with bounded full-span row sampling so large-result tail values influence width without scanning all rows.
- [x] Add adaptive dense-grid render policy (truncation, column-cap tightening, separator hiding) plus explicit sampled-row hint in Data status.
- [x] Add adaptive width-sampling budgets and max-width early-exit so very large/wide result sets reduce column-estimation work.
- [x] Reuse a single Data rows table across result refreshes and apply only changed column widths to reduce redraw churn.
- [x] Add ultra-wide header-width mode and dense-result shallow row caching to reduce width-estimation and deep-copy overhead on very large grids.
- [x] Avoid duplicate row-sampling work and skip redundant container-object swaps/unselect calls during repeated Data-grid refreshes.
- [x] Expand native table/grid strategy with richer keyboard nav and larger-result virtualization tuning.
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

- [ ] Provider settings for Ollama/OpenAI-compatible endpoints, response reserve, GPU diagnostics, and provider probes.
- [x] Native provider settings for Ollama/OpenAI-compatible/custom endpoints, explicit protocol flags, provider probes, context-window options, response reserve, and Ollama runtime diagnostics.
- [x] Curated native local model catalog ported from Wails with recommended model labels, context-window sizing, and response reserve defaults in Fyne Settings.
- [x] Automatic loaded-model context-window tuning from Wails: provider probe applies Ollama runtime `context_length` to native context/reserve fields.
- [x] Add API key input/persistence in native settings and propagate bearer auth into OpenAI-compatible chat/probe config.
- [x] Add task-aware model defaults in Settings with user-configurable defaults for coding, React/TypeScript/JavaScript, Go backend, Python, PHP/Laravel, SQL, Neo4j/Cypher, CSV/Excel scripts, analytics explanations, research/summaries, vision/screenshot understanding, balanced reasoning/vision, and fastest 30B-class coding.
- [x] Add Ask-mode model-route selector with global fallback and assistant-service route metadata.
- [x] Wire task-aware model route resolution into Git AI diff summary/commit drafting with assistant-service fallback warnings and saved `chat-answer` route metadata.
- [x] Wire task-aware model route resolution into Agent mode with selected-route context budgeting, fallback warnings, final-response route metadata, and persisted agent-audit model provenance.
- [x] Add assistant/agent auto-routing by selected context so data, SQL, document, image/screenshot, and code contexts select the matching task-aware model route before falling back to global.
- [ ] Wire task-aware model route resolution into future dedicated Data, document, and vision/screenshot model workflows with fallback warnings and persisted route metadata.
- [ ] Deeper GPU diagnostics.
- [x] Streaming chat with selected files/directories/project context, token-budgeted history, persisted turns, and source-path context.
- [x] Native assistant source/model diagnostics parity slice: Wails-compatible context-label source fallback parsing, source/model/context answer footer, and effective source persistence for saved answer artifacts.
- [x] Finer-grained citation refs beyond file-level sources in native assistant UI and chat-answer artifact metadata.
- [x] Evidence-quality diagnostics for native assistant answers and `chat-answer` artifact metadata.
- [x] Persist bounded citation snippets in saved `chat-answer` artifacts and metadata for line-cited answers.
- [x] Persist unverified/out-of-context citation diagnostics in native assistant footers and `chat-answer` artifact metadata.
- [x] Add citation coverage diagnostics so native assistant footers and saved `chat-answer` metadata show cited/uncited source coverage.
- [x] Persist structured cited/uncited source coverage lists in `chat-answer` artifacts and metadata, and preserve them through chat-answer regeneration.
- [x] Local assistant memory and prompt profiles.
- [x] Agent runtime with plan updates, bounded observations, model-driven tool calls, no frontend iteration cap, emergency backend loop guard, and final-answer fallback behavior.
- [x] Agent runtime resolves selected task-aware model routes without UI/framework coupling and records model/route provenance in SQLite audit history.
- [x] Unified tool registry and dispatcher for deterministic tools and model-requested tools.
- [x] Agent tools for read context, workspace search, problems, Git status/diff, tasks, artifacts, datasets, SQLite, documents, operations files, safe writes, patches, copy/move/delete, and rollback.
- [x] Agent tools for read-only Git history/blame.
- [x] Agent tool for read-only artifact lineage context.
- [x] Agent tool for approval-gated artifact regeneration actions.
- [ ] Agent approved shell beyond discovered safe tasks.
- [x] Approval-gated native `web_fetch` agent tool with Wails-equivalent HTTP(S), redirect, size, content-type, allow-list, and local-network guards.
- [x] Live activity tail that shows compact model/tool progress while preserving persisted agent audit history.

### Artifacts And Provenance

- [x] Markdown, CSV, SVG/chart, SQL result, task report, notebook report, document report, document brief, DOCX document export, scan report, operations runbook, comparison, chat-answer, and extracted-document artifacts.
- [x] First native presentation outline artifacts in the Artifacts UI with source provenance.
- [x] First native packaged presentation zip exports in the Artifacts UI from presentation outlines, with manifest/package-file metadata and package preview text.
- [x] First native PPTX deck exports in the Artifacts UI from packaged presentation artifacts, with package-file metadata and package preview text.
- [x] First richer generated document brief artifacts in the Artifacts UI with source provenance.
- [x] First native DOCX document exports in the Artifacts UI from document briefs, with OpenXML package output, package-file metadata, and package preview text.
- [x] First DOCX/PPTX package validation metadata and preview status for native Office exports.
- [x] First DOCX/PPTX Office theme baseline with persisted template/theme metadata and branded generated output styling.
- [ ] Richer DOCX/PPTX template variants, cross-suite compatibility smoke, and visual design polish in native UI.
- [x] Provenance sidecars with source files, query IDs, generated timestamps, metadata rows, and freshness fingerprints for explicit artifact writes.
- [x] Chat-answer artifacts include prompt/model/context/source/citation/unverified-citation/snippet/evidence and structured cited/uncited source coverage metadata.
- [ ] Complete tool-run provenance coverage for every generated output type beyond chat-answer artifacts.
- [x] Artifact browser with search, metadata, preview, compare, archive, delete, restore, and open-source navigation.
- [x] Artifact lineage/freshness warnings for current native artifacts.
- [x] Artifact lineage graph import/export UI parity with workspace graph JSON artifacts.
- [x] First artifact regeneration workflow that reuses dataset summary, query, SQL, chart, dashboard, SQL notebook, and SQLite query export dependency metadata.
- [x] Expand artifact regeneration beyond dataset/query/chart/notebook/SQLite baseline with document-report, scan-report, document-extract, operations-runbook, and artifact-comparison rebuild actions.
- [x] Add approval-gated agent artifact regeneration for supported native rebuildable artifacts.
- [x] Add saved chat-answer refresh regeneration from persisted prompt/model/source/citation/evidence metadata without requiring a new model call.
- [x] Expand artifact regeneration to generated presentation outline artifacts.
- [x] Expand artifact regeneration to packaged presentation exports in both the Artifacts UI and approval-gated agent tool.
- [x] Expand artifact regeneration to first native document brief artifacts in both the Artifacts UI and approval-gated agent tool.
- [x] Expand artifact regeneration to first native DOCX document exports in both the Artifacts UI and approval-gated agent tool.
- [x] Expand artifact regeneration to first native PPTX presentation deck exports in both the Artifacts UI and approval-gated agent tool.
- [ ] Expand artifact regeneration to future richer DOCX/PPTX export variants beyond the native export baselines.

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
- [x] Path-root enforcement, traversal protection, ignored-state protection, and `.nexusdesk` protection across native workspace/file mutation services.
- [x] Rollback snapshots for approved native workspace mutations where practical.
- [x] OS-protected secrets on Windows, macOS Keychain, and Linux Secret Service/libsecret.
- [x] Append-only/persisted audit records for approvals, native agent/tool runs, file changes with rollback records, tasks, jobs, SQL runs, and artifacts.
- [ ] Extend audit coverage to future connector sync jobs, OCR, dump imports, shell, and Docker mutations.
- [x] Export/backup flows for local-first data.

### Jobs, Persistence, And Observability

- [x] SQLite-first metadata store for chats, approvals, artifacts, tool runs, jobs, SQL runs, and dataset dependencies.
- [x] Search index metadata and recovery/export flows: explicit workspace searches now persist a bounded `.nexusdesk/search/index-metadata.json` manifest with counts, caps, result paths/lines, and corrupt-manifest quarantine under `.nexusdesk/search/recovery/`.
- [x] JSON compatibility import from Wails-era workspaces for chat history, approvals, artifact sidecars, and tool-run logs.
- [x] Legacy Wails SQLite dataset SQL run and dataset dependency import into native SQLite metadata.
- [x] First durable job monitor with progress log tail, cancellation, retry from persisted task runs, and task-report output opening.
- [x] Shared slow-workflow job contract with explicit-user-start enforcement and workspace-open prohibition for OCR, dump imports, connector pulls, long indexing, report generation, long agent runs, and packaged exports.
- [x] First Diagnostics surface for provider probe/runtime status, metadata health, and recent persisted job/task/SQL/agent failure snapshots.
- [x] Diagnostics quick actions and recommended-remediation hints for provider/settings, metadata health, and recent failure triage.
- [x] Diagnostics redacted issue-report export bundle with diagnostics text, activity tail, environment metadata, workspace-state file names, path/secret redaction, and no workspace contents unless explicit relative paths are requested.
- [x] Route native document-report and document-extraction artifact generation through durable jobs with persisted job records and job-output opening.
- [x] Route external connector query execution through durable jobs with cancellation, job logs, and Jobs-panel output fallback for non-task job types.
- [x] Route external connector test/inspect flows through durable jobs with cancellable context propagation and Jobs-panel output fallback.
- [x] Persist external connector test/inspect job lineage into dataset dependency metadata for audit/history coverage.
- [x] Add shell-level regression tests for connector test/inspect lineage persistence, including canceled-path metadata.
- [x] Classify canceled dataset/SQLite/connector SQL runs as `canceled` (not `failed`) in persisted metadata, with regression tests.
- [x] Route operations scan/inspect/runbook export through durable jobs with cancellable context-aware service methods and consistent job output handling.
- [x] Route dataset profile/query/SQL actions through durable jobs with consistent cancel/status/output behavior in Data and Jobs panels.
- [x] Route SQL notebook run/export flows through durable jobs with cancellable context propagation and artifact/job linkage.
- [x] Route chart/dashboard preview+export and SQLite query artifact export through durable jobs with cancellation support and artifact/job linkage.
- [x] Add context-aware dataset service methods for profile/query/SQL so job cancellation propagates cleanly.
- [x] Add context-aware dataset notebook execution methods so cancellation propagates across notebook cell runs.
- [x] Add native metadata backup export (zipped `.nexusdesk/metadata` bundle) with Diagnostics quick action.
- [x] Add automatic corrupt-metadata recovery on workspace open (archive malformed SQLite DB, recreate clean store) with regression tests.
- [x] Move Wails-era compatibility metadata import off the workspace-open critical path (async with completion stamp) to keep folder open responsive.
- [x] Route async compatibility metadata import through durable jobs for visibility, status, and audit continuity.
- [x] Skip compatibility-import job scheduling after completion stamp exists to keep repeated workspace opens lightweight.
- [x] Deduplicate in-flight compatibility imports per workspace to prevent duplicate background import jobs on rapid reopen/refresh.
- [x] Harden compatibility import stamp handling: atomic writes + malformed-stamp quarantine fallback so startup never stalls on bad marker JSON.
- [x] Add context-aware compatibility import execution and route workspace import jobs through cancellable metadata-import context.
- [x] Add job history retention controls and cleanup policy: explicit Jobs-panel cleanup prunes successful/canceled completed jobs by count/age while preserving running jobs and failures/timeouts by default.
- [x] Add startup recovery markers and crash/hang triage visibility in Home readiness and Diagnostics; clean exits mark the session closed, while previous unclean exits warn users before repeating long work.
- [x] Folder open remains cheap: no Git, Docker, OCR, connector pulls, dump imports, model calls, shell commands, or deep indexing.
- [x] Diagnostics panel for app logs, provider status, GPU/model status, metadata health, and job history.

### Extensibility And Community

- [x] Package ownership docs for every major `internal/` area, with `docs/23_INTERNAL_PACKAGE_OWNERSHIP.md` and an in-app Help/command-palette guide covering layer rules, service ownership, UI boundaries, slow-work rules, and risky-action rules.
- [x] Contributor setup guide, coding standards, tests, and architecture decision records, with `docs/24_CONTRIBUTOR_SETUP_AND_STANDARDS.md`, `docs/adr/README.md`, and an in-app Help/command-palette guide.
- [ ] Plugin/MCP strategy after native core tools are stable.
- [ ] Stable service interfaces for community-contributed connectors and document parsers.
- [x] Add dependency guard/static check for forbidden Wails/webview imports, Fyne leakage, and UI imports from services/domain.
- [x] Native CI matrix for Windows, macOS, and Linux formatting, tests, static analysis, and Fyne build smoke.
- [x] Add native build metadata validation for version, commit, and build date through CI ldflags.
- [ ] Expand CI into signed release packaging and installer/update validation.

## Claude Findings Integration

- [x] Add API key to native settings and propagate `Authorization: Bearer` config into OpenAI-compatible chat/probe flows.
- [x] Unblock agent `run_task` execution path by allowing shell-task approval when requested by agent flow.
- [x] Mirror write-path symlink protections for all read paths (`PreviewFile`, context-pack, search) by rechecking resolved real paths.
- [x] Rework metadata store hot path so `Store` opens DB once and runs `Ensure` outside hot loops.
- [x] Replace fixed model dropdown with a configurable/free-form model selector and in-panel probe-driven model validation.
- [x] Add per-call agent approval modal for high-risk mutation tools instead of only ambient full-project approval.
- [x] Extend file preview to truncated text preview over cap and only reject truly non-previewable binaries.
- [x] Align preview/context candidate detection across extensions and filename basenames via a shared policy table.
- [x] Improve encoding fallback from hardcoded Windows-1251 for unsupported text encodings.
- [x] Avoid full-history replay in approval persistence; persist only the new record and serialize repository writes safely.
- [x] Add single-file-open entrypoint (file picker/quick-open) in addition to folder open flow.
- [x] Consolidate bottom-panel navigation to reduce tab discoverability and density issues.
- [x] Split `internal/ui/shell/data_panel.go` into smaller UI responsibility files.
- [x] Add performance profiling harness for shell redraw, activity log, data grid, large search, and large artifacts.
- [x] Add startup, workspace metadata, and folder-open performance timings to Diagnostics with over-budget warnings and remediation guidance.
- [x] Ensure `ApplyFileAppend` failure path cleans rollback snapshots cleanly and checks close errors.
- [x] Surface directory-entry truncation in navigator UI when entry cap clips folder contents.
- [x] Verify explicit discard confirmation path on dirty-tab close and add regression coverage.
- [x] Replace full-tree refresh on file operations with targeted tree node refresh helpers.
- [x] Implement hunk-windowed unified diffs with optional large-file elision.
- [x] Add guardrails for empty role/action string formatting in agent/activity path rendering.
- [x] Add settings-level LLM connection test action with model count and warning reporting.
- [x] Surface persistence failures from repository writes in activity/diagnostics.
- [x] Cancel streaming LLM response handlers promptly when request context is canceled.
- [x] Make provider configuration explicit and extensible beyond built-in options with protocol flags.
- [x] Clamp agent context budget fallback to safe non-trivial defaults when settings become misconfigured.
- [x] Scope global shortcuts so native editor copy remains reliable.
- [x] Improve search snippets to center on match location for long lines.
- [x] Normalize `cleanRel` output using `filepath.Clean` after traversal checks.
- [x] Cap `activityText` and `activityLines` growth so activity rendering remains bounded.
- [x] Expand append target safety sampling and encoding checks for UTF-16 and sparse/edge-case encodings.
- [x] Update welcome flow with immediate open action from startup screen.
- [x] Remove migration wording from About dialog and align with release messaging.
- [x] Define and document non-Windows CI/build support matrix and execution plan.
- [x] Revisit default LLM model strategy to avoid hard-coded defaults that rarely match local setups.
- [x] Replace locale-bound mutation-claim heuristics with mutation-observation-driven verification.
- [x] Expand safe task execution whitelist beyond current npm/go/compose-only support.
- [x] Replace shell string execution for discovered tasks with argument-based process invocation.
- [x] Avoid schema file rewrite on every metadata `Ensure` invocation.
- [x] Separate Ask vs Agent system prompt behavior for clearer model role control.
- [x] Add quick-open keyboard workflow (e.g., Ctrl+P) for direct file navigation.
- [x] Close welcome tab on first workspace open or repurpose into workspace-aware home surface.
