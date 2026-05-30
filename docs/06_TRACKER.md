# Execution Tracker

Status: canonical checklist for finishing NexusDesk end to end.

Use this tracker for implementation slices. Work from the earliest phase with open items unless a later item blocks release safety. Check items only after implementation, tests, docs, and validation are complete.

Legend:

- `[ ]`: not done.
- `[x]`: done.
- `P0`: release blocker or data-loss/security risk.
- `P1`: important production readiness item.
- `P2`: polish or post-beta hardening.

## Phase 0: Documentation Reset

- [x] P1 Create canonical docs index.
- [x] P1 Create canonical architecture document.
- [x] P1 Create canonical UI workbench document.
- [x] P1 Create canonical feature inventory.
- [x] P1 Create canonical goals document.
- [x] P1 Create canonical production plan.
- [x] P1 Create detailed execution tracker.
- [x] P1 Remove duplicate docs asset clutter from `docs/`.
- [x] P1 Update root `tracker.md` to point to this canonical tracker.
- [x] P1 Update root `README.md` links if doc filenames changed.
- [x] P1 Run docs reference sweep.
- [x] P1 Remove obsolete web-runtime code, docs, and build metadata. (Repository sweeps removed stale web-runtime references and ignored build-output paths so the repo reflects the Fyne-native app layout.)
- [x] P1 Run `git diff --check`.

## Phase 1: Safety Lockdown

### 1.1 File preview and save safety

- [x] P0 Add top-level `Truncated` metadata to file preview domain model.
- [x] P0 Add top-level text byte count/source byte count metadata to file previews.
- [x] P0 Mark previews partial when read caps are hit.
- [x] P0 Disable inline save for partial previews.
- [x] P0 Show editor banner for partial/truncated files.
- [x] P0 Add explicit open-external or full-load option only if safe.
- [x] P0 Include truncation state in `read_file` tool observations.
- [x] P0 Ensure assistant context packs preserve truncation warnings.
- [x] P0 Add tests for large text file preview.
- [x] P0 Add tests that save is blocked for partial preview.
- [x] P0 Add tests that agent read observation exposes truncation.

### 1.2 Archive and document container caps

- [x] P0 Add total uncompressed cap for XLSX parsing.
- [x] P0 Add per-entry cap for XLSX shared strings.
- [x] P0 Add per-entry cap for XLSX worksheets.
- [x] P0 Add cap tests for malicious XLSX containers.
- [x] P0 Review DOCX extraction for unbounded zip entry reads.
- [x] P0 Add DOCX extraction caps.
- [x] P0 Review PPTX generation/validation for bounded zip handling.
- [x] P1 Document archive/container caps in architecture or safety notes.

### 1.3 Protected secret hardening

- [x] P0 Ensure macOS secret storage never passes secret value in process arguments.
- [x] P0 Add macOS command-argument exposure test or documented smoke.
- [x] P0 Add `runtime.KeepAlive` or safer wrapper for Windows DPAPI input buffers.
- [x] P0 Add Windows DPAPI stress test.
- [x] P0 Ensure Linux protected storage failure is explicit and actionable.
- [x] P1 Add diagnostics card for protected secret backend status. (Diagnostics now reports DPAPI/Keychain/Secret Service backend status, remediation, health-card state, and recommended actions.)
- [x] P1 Add cross-platform protected secret CI/smoke plan.

### 1.4 Network and connector safety

- [x] P0 Replace pre-lookup-only private IP checks in web fetch with actual dial-address checks.
- [x] P0 Cover IPv4 private, loopback, link-local, multicast ranges.
- [x] P0 Cover IPv6 loopback, link-local, multicast, and ULA ranges.
- [x] P0 Recheck safety after redirects.
- [x] P0 Add DNS rebinding regression tests.
- [x] P0 Default PostgreSQL profiles to encrypted transport.
- [x] P0 Default MySQL/MariaDB profiles to encrypted transport.
- [x] P0 Default SQL Server profiles to encrypted transport.
- [x] P0 Add explicit development-only plaintext toggle.
- [x] P0 Audit plaintext toggle changes.
- [x] P1 Show resolved transport mode in connector profile UI. (Data Sources now displays normalized transport state, including secure defaults, certificate-skip mode, and explicit development plaintext.)
- [x] P1 Show resolved transport mode in Diagnostics. (Diagnostics now includes a connector transport health card, report section, plaintext warning, and remediation action.)

### 1.5 Agent mutation honesty

- [x] P0 Ensure file mutation tools set `Mutated` accurately.
- [x] P0 Ensure formatter tool sets `Mutated` when it writes.
- [x] P0 Ensure Git staging tools set `Mutated` accurately.
- [x] P0 Ensure commit and branch tools set `Mutated` accurately.
- [x] P0 Ensure conflict resolution tools set `Mutated` accurately.
- [x] P0 Ensure artifact regeneration sets `Mutated` accurately.
- [x] P0 Ensure project memory updates set `Mutated` accurately.
- [x] P0 Make agent final verification use tool result mutation flags instead of fragile tool name lists.
- [x] P0 Add tests for verification messages after representative mutating tools.

### 1.6 Task and terminal safety

- [x] P0 Ensure discovered task execution uses argv, not shell strings.
- [x] P0 Add tests for task names containing shell metacharacters.
- [x] P0 Keep one-shot terminal command rooted to workspace or explicit safe cwd.
- [x] P0 Keep one-shot terminal command approval-gated.
- [x] P0 Keep one-shot terminal command output-capped.
- [x] P0 Keep one-shot terminal command timeout-bound.
- [x] P0 Keep one-shot terminal command audited.
- [x] P0 Reject shell interpreter shortcuts unless explicitly designed and approved.

### 1.7 Workspace path safety

- [x] P0 Re-audit read and write paths for symlink escape behavior.
- [x] P0 Add tests for symlink parent components.
- [x] P0 Add tests for symlink file targets outside root.
- [x] P0 Reject Windows alternate data stream path forms where relevant.
- [x] P0 Reject Windows reserved device names where relevant.
- [x] P1 Ensure error messages explain rejected path cause.

## Phase 2: Performance Floor

### 2.1 Assistant and activity rendering

- [x] P1 Coalesce assistant stream deltas with a UI refresh ticker.
- [x] P1 Parse markdown once at final response where practical. Evidence: assistant streaming now renders interim deltas as plain RichText text segments and only parses the final assistant response as markdown; covered by focused assistant stream/response tests.
- [x] P1 Coalesce agent event rendering.
- [x] P1 Replace activity markdown rebuild with incremental list rendering. Evidence: activity UI is now a bounded VBox of label rows updated incrementally per event while preserving the capped text buffer for tests/audit; covered by `TestAddActivityKeepsBoundedMarkdownBuffer`.
- [x] P1 Cap activity display while preserving durable audit elsewhere.
- [x] P1 Add long streaming UI stress test or profiling harness.
- [x] P1 Add agent-event burst test.

### 2.2 Editor save performance

- [x] P1 Move save flow off UI thread.
- [x] P1 Show visible saving state.
- [x] P1 Keep tab dirty marker accurate during save.
- [x] P1 Avoid rebuilding whole editor panel after save.
- [x] P1 Preserve cursor after save.
- [x] P1 Preserve scroll after save.
- [x] P1 Add failure state and retry affordance for failed save.
- [x] P1 Add save performance test or benchmark.

### 2.3 Diff and rollback performance

- [x] P1 Replace large-file LCS diff with bounded hunk diff.
- [x] P1 Add deadline or size fallback for diff generation.
- [x] P1 Add tests for large-line-count diff.
- [x] P1 Add rollback storage usage diagnostics.
- [x] P1 Add rollback retention policy documentation.
- [ ] P2 Add content-addressed rollback storage.
- [ ] P2 Add deduplication tests for identical snapshots.

### 2.4 Workspace search performance

- [x] P1 Stop using full file preview for content search. (Content search now reads bounded safe text directly instead of going through capped preview text.)
- [x] P1 Add streaming byte-level literal search. (Search now scans UTF-8 text line-by-line up to an 8 MiB per-file cap and finds matches beyond the safe write-preview cap.)
- [x] P1 Add streaming regex search with bounds. (Regex content search uses the same streaming path with a per-line cap to bound matching work.)
- [x] P1 Skip known binary files before opening. (Search now rejects known binary/container extensions before content reads, with coverage for image/archive/wasm paths.)
- [x] P1 Increase per-file cap safely. (Search content cap increased to 8 MiB with extension pre-skip, binary sampling, and UTF-16 fallback kept at the safe write cap.)
- [x] P1 Add total search wall-clock limit. (Workspace search now applies a default 2s wall-clock budget, exits traversal cleanly, and records timed-out/truncated metadata.)
- [x] P1 Add cancellation/singleflight for search while typing. (Workspace search now accepts context cancellation, and the search panel cancels/suppresses stale searches before updating results.)
- [x] P1 Stream results to UI. (Workspace search now emits partial result callbacks, and the Search panel renders partial snapshots while preserving stale-search cancellation.)
- [x] P1 Add tests for matches beyond old preview cap.
- [x] P1 Add benchmark or synthetic large workspace test. (Workspace search now has a synthetic large-workspace test covering result caps, ignored directories, and metadata.)

### 2.5 Metadata store performance

- [x] P1 Enable SQLite WAL mode. (Metadata store now configures WAL on the cached SQLite connection and surfaces the active journal mode.)
- [x] P1 Enable busy timeout. (Metadata store now applies a 5s SQLite busy timeout and records it in status/diagnostics.)
- [x] P1 Enable foreign keys consistently. (Metadata store now enables foreign-key enforcement on the single cached SQLite connection and tests the runtime pragma.)
- [x] P1 Review connection pooling strategy. (Metadata store now uses one cached SQLite handle with `SetMaxOpenConns(1)`/`SetMaxIdleConns(1)` to keep connection-scoped pragmas consistent.)
- [x] P1 Avoid schema bootstrap work after first open. (Metadata store already caches `Ensure`; regression coverage verifies repeated open/ensure reuses the handle and leaves schema files untouched.)
- [x] P1 Surface active journal mode in Diagnostics. (Diagnostics metadata section and health card now show journal mode, foreign-key state, and busy timeout.)
- [x] P1 Add tests for concurrent read/write behavior where practical. (Metadata store now has a concurrent SaveJob/ListJobs stress test covering the WAL/busy-timeout connection configuration.)

### 2.6 Connector performance

- [x] P1 Cache external database pools by profile and version. (PostgreSQL, MySQL/MariaDB, and SQL Server connector test/query/inspect flows now reuse profile-keyed `database/sql` pools.)
- [x] P1 Invalidate pool when profile changes. (Pool version includes driver/profile DSN state; a changed profile version closes and replaces the old pool.)
- [x] P1 Respect context cancellation for pooled queries. (Pooled flows borrow connections with `db.Conn(ctx)` and query/inspect with context-aware calls; coverage verifies canceled context behavior.)
- [x] P1 Bound idle lifetime. (External connector pools now set max open/idle connections, idle lifetime, and max lifetime.)
- [x] P1 Surface pool/connection status in Diagnostics. (Diagnostics now reports open external connector pools, driver, open/in-use/idle counts, and last-used time.)
- [x] P1 Add tests with fake SQL driver or integration harness. (Connector pool tests use the SQLite driver as a lightweight integration harness for reuse, invalidation, stats, and cancellation.)

## Phase 3: Correctness And Audit Honesty

### 3.1 SQL safety correctness

- [x] P1 Extract shared SQL read-only analyzer.
- [x] P1 Replace dataset raw keyword blocklist with token-aware analyzer.
- [x] P1 Keep connector and dataset behavior consistent.
- [x] P1 Add string-literal keyword tests.
- [x] P1 Add comment keyword tests.
- [x] P1 Add multi-statement rejection tests.
- [x] P1 Add fuzz test seed corpus.

### 3.2 Agent bounds and context

- [x] P1 Add default wall-clock limit for agent runs.
- [x] P1 Make wall-clock limit configurable per request or setting.
- [x] P1 Propagate context deadline through model calls and tools.
- [x] P1 Surface timeout stop reason in UI.
- [x] P1 Persist timeout stop reason in audit.
- [x] P1 Replace fixed history count with token-aware packing.
- [x] P1 Add stress test for repeated tool-request loop.

### 3.3 Job logs

- [x] P1 Raise visible job log tail.
- [x] P1 Persist full logs under job directory.
- [x] P1 Add open-full-log action.
- [x] P1 Include relevant logs in issue report with redaction.
- [x] P1 Add tests for long job logs.

### 3.4 Artifact audit and rollback

- [x] P1 Snapshot artifact before archive where needed.
- [x] P1 Snapshot artifact before restore where needed.
- [x] P1 Snapshot artifact before delete.
- [x] P1 Snapshot artifact before regenerate.
- [x] P1 Persist artifact rollback metadata.
- [x] P1 Add UI recovery affordance.
- [x] P1 Add tests for artifact destructive flows.

### 3.5 Git robustness

- [x] P1 Split Git timeouts by operation class.
- [x] P1 Set non-interactive Git environment.
- [x] P1 Detect ownership/safety errors and show remediation.
- [x] P1 Avoid silent fetch/pull/network behavior.
- [x] P1 Add tests for timeout class selection.
- [x] P1 Add tests for error classification.

### 3.6 Encoding honesty

- [x] P1 Add charset detection strategy.
- [x] P1 Use lossless fallback when confidence is low.
- [x] P1 Surface ambiguous encoding warning.
- [x] P1 Disable save until encoding is explicit when ambiguity risks data loss.
- [x] P1 Add tests for Latin-1.
- [x] P1 Add tests for Windows-1251.
- [ ] P2 Add tests for Shift-JIS or other common encoding.

### 3.7 Recent workspace correctness

- [ ] P2 Stat recent workspace paths on list.
- [ ] P2 Mark missing paths.
- [ ] P2 Add remove missing workspace action.
- [ ] P2 Add tests for missing recent path.

## Phase 4: JetBrains-Style UI Refactor

### 4.1 Controller extraction

- [x] P1 Define shell controller interfaces.
- [x] P1 Add typed shell event bus.
- [x] P1 Extract editor controller. Evidence: `tabs.go` defines `editorController`, `View` owns it through `view.editor`, and editor chrome/tab tests exercise controller-owned tab state and UI actions.
- [x] P1 Extract assistant controller. Evidence: `assistant_panel.go` defines `assistantController` for assistant widgets/state, `View` initializes it through `newAssistantController`, and assistant controller tests cover initial state and run controls.
- [x] P1 Extract data controller. Evidence: `data_panel.go` defines `dataController` with owned panel widgets and connector/query state, initialized by `newDataController`, with data controller tests covering initial state.
- [x] P1 Extract Git controller. Evidence: `git_panel.go` defines `gitController` with panel/status/diff/hunk ownership and tests cover initial state plus status row/badge updates.
- [x] P1 Extract artifacts controller. Evidence: `artifacts_panel.go` defines `artifactsController` with results/status/preview/source state and tests cover initial state plus source refresh behavior.
- [x] P1 Extract jobs controller. Evidence: `jobs_panel.go` defines `jobsController` with job list/status/output controls and tests cover initial state, empty refresh, and missing output handling.
- [x] P1 Extract diagnostics controller. Evidence: `diagnostics_panel.go` defines `diagnosticsController` with panel state and refresh/export actions, with diagnostics controller tests covering initial and workspace-required states.
- [x] P1 Extract approvals/audit controller. Evidence: approvals and agent-audit panels now have dedicated `approvalsController` and `agentAuditController` ownership, with `View` retaining thin compatibility wrappers for workspace-open refresh paths (`go test ./internal/ui/shell -run "Approval|AgentAudit|Audit"`).
- [x] P1 Shrink `View` to layout and registry ownership. Evidence: remaining shared tool-window widgets now live behind the embedded `toolPanelWidgets` owner and `newToolPanelWidgets` factory instead of being hand-built in `NewWithStartupStatus`, keeping `View` focused on service wiring, layout state, and controller/registry setup (`go test ./internal/ui/shell -run "View|Controller|Tool|Panel|Task|History|Approval|AgentAudit"`).
- [x] P1 Add tests for controller event flow.

### 4.2 Tool-window framework

- [x] P1 Define tool-window registry type.
- [x] P1 Register Project tool window.
- [x] P1 Register Search tool window.
- [x] P1 Register Problems tool window.
- [x] P1 Register Git tool window.
- [x] P1 Register Data Sources tool window.
- [x] P1 Register Artifacts tool window.
- [x] P1 Register Operations tool window.
- [x] P1 Register Tasks tool window.
- [x] P1 Register Jobs tool window.
- [x] P1 Register History tool window.
- [x] P1 Register Approvals tool window.
- [x] P1 Register Diagnostics tool window.
- [x] P1 Register Activity tool window.
- [x] P1 Add keyboard shortcut routing through registry.

### 4.3 Left rail

- [x] P1 Implement thin icon-first rail.
- [x] P1 Add active icon state.
- [x] P1 Add collapse on active-icon click.
- [x] P1 Add hover tooltips.
- [x] P1 Add keyboard navigation.
- [x] P1 Remember active tool per workspace.
- [x] P1 Remember width per tool window. (The dock split stores/restores validated offsets by active tool key so each tool reopens at its last useful width.)
- [x] P1 Add resize handle with enlarged hit zone. Evidence: the left dock now wraps tool content with a draggable edge handle using the density `ResizeHandleHitWidth` token, clamps offsets to the existing safe range, and remembers per-tool widths; covered by `TestToolPanelResizeHandleUsesDensityHitZoneAndStoresOffset`.
- [x] P1 Add tests for rail state mapping.

### 4.4 Center editor UI

- [x] P1 Replace first-launch dashboard/cockpit feeling with editor-like empty state.
- [x] P1 Keep tabs compact.
- [x] P1 Keep breadcrumbs compact.
- [x] P1 Keep document map optional and unobtrusive.
- [x] P1 Add stable split behavior.
- [x] P1 Show save state in status bar.
- [x] P1 Show encoding/line endings in status bar.
- [x] P1 Keep editor width priority during resize.

### 4.5 Right assistant UI

- [x] P1 Redesign assistant header hierarchy.
- [x] P1 Add visible mode/model/route state.
- [x] P1 Add source digest above composer or messages.
- [x] P1 Add clearer tool timeline.
- [x] P1 Add approval cards with details.
- [x] P1 Keep composer pinned to bottom.
- [x] P1 Add stop/cancel visibility during runs.
- [x] P1 Add Sources secondary pane.
- [x] P1 Add Lineage secondary pane.
- [x] P1 Add Inspector secondary pane.

### 4.6 Remove bottom tool panel

- [x] P1 Remove horizontal bottom tool panel from the target shell. (The utility tabs now live in a collapsible left dock beside the editor instead of a vertical split below the workbench.)
- [x] P1 Keep the bottom region as status bar only. (`View.Canvas` now returns the workbench with only `newStatusBar()` in the bottom border region.)
- [x] P1 Move Problems to the left-sidebar tool registry.
- [x] P1 Move Search Results to the left-sidebar tool registry.
- [x] P1 Move Git Diff to the left-sidebar tool registry.
- [x] P1 Move Tasks to the left-sidebar tool registry.
- [x] P1 Move Jobs to the left-sidebar tool registry.
- [x] P1 Move Audit to the left-sidebar tool registry.
- [x] P1 Move Diagnostics to the left-sidebar tool registry.
- [x] P1 Move Activity to the left-sidebar tool registry.
- [x] P1 Add keyboard shortcuts for the moved tool windows.
- [x] P1 Ensure the moved tool windows keep per-tool width memory and collapse behavior. (Moved tools now use the shared dock collapse path while preserving per-tool split widths.)

### 4.7 Settings UI

- [x] P1 Add searchable settings shell. (`settings_panel.go` filters titled settings sections by title, summary, and keywords.)
- [x] P1 Add provider settings category. (Provider/runtime endpoint, protocol, model, and token budget controls live under Provider & Runtime.)
- [x] P1 Add task model routing category. (Task Model Routes edits per-route model defaults and detail text.)
- [x] P1 Add credentials category. (Secrets & Credentials owns the redacted API-key field and protected-storage note.)
- [x] P1 Add connector profiles category. (Connector Profiles links users to the Data tool where connector editing and schema/query testing live.)
- [x] P1 Add safety/approval category. (Safety & Approvals links to approval records and explains approval policy entry points.)
- [x] P1 Add UI density/theme category. (UI Density & Theme documents the current Fyne/system-theme behavior and disabled customization state.)
- [x] P1 Add diagnostics/test category. (Diagnostics & Tests now groups provider and route probe actions with shared probe status.)
- [x] P1 Add route test actions. (Settings can test the currently selected task route using that route's model/budget overrides.)
- [x] P1 Add disabled-state explanations. (Validation, route-test errors, and category notes explain missing model, disabled customization, and delegated connector/profile actions.)

### 4.8 Theme and visual polish

- [x] P1 Replace hardcoded colors with theme tokens. (Shell syntax highlighting now consumes `internal/ui/theme` palette tokens; the remaining `color.NRGBA` literals are centralized theme definitions/tests.)
- [x] P1 Define panel/editor/raised/status colors. (`Palette` includes background, panel, raised panel, editor, status bar, border, and shadow tokens.)
- [x] P1 Define semantic success/warning/error colors. (`Palette` includes success/warning/error foreground and background pairs mapped through the Fyne theme adapter.)
- [x] P1 Normalize spacing and row height. (`Density` now carries padding, inner padding, row height, and resize-hit-width tokens for compact/comfortable modes.)
- [x] P1 Normalize focus rings. (`Density` and `Palette` expose focus ring width/color tokens.)
- [x] P1 Normalize active tab underline. (`Palette` and `Density` expose active-tab underline color/height tokens.)
- [x] P1 Audit contrast. (`PaletteDiagnostics` checks core text, semantic, and syntax contrast ratios.)
- [x] P1 Add theme drift check or diagnostic. (`PaletteDiagnostics(JetBrainsDarkPalette())` is covered by production palette tests.)

### 4.9 Resize and visual smoke

- [x] P1 Test 1280 x 820 default. (`TestVisualSmokeSupportedWindowSizes/default` renders the live shell at 1280 x 820.)
- [x] P1 Test 1024 x 640 minimum working size. (`TestVisualSmokeSupportedWindowSizes/minimum` renders the live shell at 1024 x 640.)
- [x] P1 Test 1600 x 900 desktop size. (`TestVisualSmokeSupportedWindowSizes/desktop` renders the live shell at 1600 x 900.)
- [x] P1 Test first launch no workspace screenshot. (`TestVisualSmokeFirstLaunchNoWorkspace` verifies deterministic first-launch render markup and status state.)
- [x] P1 Test workspace open screenshot. (`TestVisualSmokeWorkspaceAndEditorStates` renders the shell after a temp workspace is loaded and verifies navigator/status state.)
- [x] P1 Test editor screenshot. (`TestVisualSmokeWorkspaceAndEditorStates` opens `README.md`, renders the shell, and verifies editor tab/draft content.)
- [x] P1 Test assistant streaming screenshot. (`TestVisualSmokeCoreToolStates` renders the assistant shell with a simulated streaming run-status header.)
- [x] P1 Test data source screenshot. (`TestVisualSmokeCoreToolStates` selects the Data tool and renders the shell.)
- [x] P1 Test artifacts screenshot. (`TestVisualSmokeCoreToolStates` selects the Artifacts tool and renders the shell.)
- [x] P1 Test settings screenshot. (`TestVisualSmokeCoreToolStates` opens the Settings editor tab and verifies the Diagnostics & Tests settings section.)
- [x] P1 Test diagnostics screenshot. (`TestVisualSmokeCoreToolStates` selects the Diagnostics tool and renders the shell.)
- [x] P1 Test approval dialog screenshot. (`TestVisualSmokeApprovalDialogState` renders the grant-access confirmation dialog and verifies its title/message.)

## Phase 5: Data, Artifact, Assistant Maturity

### 5.1 Data workbench

- [x] P1 Improve Data Sources tree icons and hierarchy. Evidence: SQLite and external connector metadata summaries now include an explicit schema tree with ASCII role markers for tables, views, columns, indexes, and foreign-key relationships; covered by `TestFormatSQLiteMetadataIncludesSchemaIndexesSamplesAndRelationships` and `TestFormatConnectorMetadata`.
- [x] P1 Add schema/table browser polish. (`formatSQLiteMetadata` and `formatConnectorMetadata` show tables, views, columns, indexes, samples, and relationships with focused tests.)
- [x] P1 Improve query editor layout. (The query editor now uses a labeled, taller, monospace, no-wrap scrolling surface with tests for the durable editor settings.)
- [x] P1 Improve result grid scrolling and virtualization. (`data_rows_grid.go` uses a virtualized `widget.Table`, visible-column clipping, row sampling for sizing, density policy, and scroll-to-selection behavior with tests.)
- [x] P1 Add result copy/export polish. (Data actions expose copy cell/row, CSV/SQL/SQLite exports, and status copy guidance; `data_rows_grid_test.go` covers copy/status behavior.)
- [x] P1 Improve query history UX. (The History tab exposes SQL history, latest-query reuse/rerun, saved SQLite queries, and dependency rebuild actions.)
- [x] P1 Add connector profile inspector. (External DB actions include Inspect profile, metadata formatting, audit records, and connector inspect tests.)
- [ ] P2 Add saved query folders/tags.

### 5.2 Dump import and sync jobs

- [ ] P2 Design dump import job safety model.
- [ ] P2 Implement isolated dump import job.
- [ ] P2 Add dump import progress/logs/cancel.
- [ ] P2 Add connector sync job design.
- [ ] P2 Implement first connector sync job after safety review.

### 5.3 Document and artifact outputs

- [x] P1 Polish DOCX report templates. Evidence: `internal/services/artifacts/document_export.go` writes themed DOCX packages with title/heading/body styles, source/package metadata, and Office validation; `document_export_test.go` asserts template/theme metadata and required package parts.
- [x] P1 Polish PPTX deck templates. Evidence: `internal/services/artifacts/presentation_deck.go` writes themed 16:9 PPTX decks with accent rail, typed title/body/footer shapes, source/package metadata, and Office validation; `presentation_deck_test.go` asserts themed slide XML and validation metadata.
- [x] P1 Add cross-suite DOCX smoke. Evidence: `internal/ui/shell/artifacts_panel_test.go` covers document brief -> DOCX export generation and regeneration from source metadata, while `internal/services/artifacts/document_export_test.go` validates the generated DOCX package.
- [x] P1 Add cross-suite PPTX smoke. Evidence: `internal/ui/shell/artifacts_panel_test.go` covers outline/package -> PPTX deck generation and regeneration from source metadata, while `internal/services/artifacts/presentation_deck_test.go` validates the generated PPTX package.
- [x] P1 Expand artifact regeneration coverage. Evidence: `internal/ui/shell/artifacts_panel_test.go` covers regeneration for document briefs/exports, presentation outlines/packages/decks, chat answers, comparison reports, cancellation, and source metadata preservation.
- [x] P1 Improve artifact freshness visualization. Evidence: artifact preview now includes `artifactFreshnessText`, source status rows, changed/missing source messages, and tests in `artifacts_panel_test.go` plus `internal/services/artifacts/freshness_test.go`.
- [x] P1 Improve lineage visualization. Evidence: artifact preview/export/import paths include `artifactLineageText`, lineage graph JSON support, provenance health diagnostics, and tests in `artifacts_panel_test.go` plus `lineage_graph_test.go`.
- [ ] P2 Add OCR/scanned document extraction job.

### 5.4 Assistant quality

- [x] P1 Improve source ranking. Evidence: assistant source actions, source pane, and source digest now use citation-ranked sources first with uncited sources after them; covered by `TestAssistantActionableSourcePathsRanksCitedSourcesFirst`.
- [x] P1 Improve source coverage UI. Evidence: assistant footer/status/sidebar/source digest report source count, verified refs, unverified refs, cited/uncited coverage, and lineage; covered by assistant source pane/digest tests.
- [x] P1 Improve uncited source warnings. Evidence: assistant diagnostics compute and display uncited source paths in evidence summaries and source digest; covered by `TestAssistantEvidenceDiagnosticReportsPartialCitationCoverage`.
- [x] P1 Improve stale source prompts. Evidence: chat history rows/details flag changed or missing original sources and seeded prompts tell users original sources are pinned; covered by `chat_history_panel_test.go`.
- [x] P1 Add model route recommendations in Settings. Evidence: Settings exposes recommended global and task-route model selectors backed by `settings.RecommendedModelOptions`, updates context/reserve budgets from recommendations, and route tests cover catalog helpers.
- [x] P1 Add context budget visualization. Evidence: assistant run/context status shows active route and approximate context budget, Settings validation shows token budget readiness, and tests cover routed/fallback budget lines.
- [ ] P2 Add image/screenshot understanding.

### 5.5 Planned tool designs

- [ ] P2 Write browser automation safety design.
- [ ] P2 Write interactive terminal session safety design.
- [ ] P2 Write PR platform tools design.
- [ ] P2 Write MCP client/tools design.
- [ ] P2 Write scheduled automation design.
- [ ] P2 Write plugin trust/signing design.
- [ ] P2 Only implement each planned tool after design approval and tests.

## Phase 6: Packaging And Release Trust

### 6.1 Release build pipeline

- [x] P1 Define release build command.
- [x] P1 Produce Windows zip artifact. Evidence: `scripts/package-windows-zip.ps1` builds `nexusdesk.exe`, generates manifest/SBOM/provenance evidence, and writes `nexusdesk-windows-<version>.zip`; iteration 149 smoke produced a 28,556,734-byte zip containing `nexusdesk.exe`, `nexusdesk-windows-manifest.json`, `nexusdesk-windows-sbom.json`, and `nexusdesk-windows-provenance.json`.
- [x] P1 Produce Windows installer artifact. Evidence: `scripts/package-windows-installer.ps1` now creates `nexusdesk-windows-installer-<version>.zip` containing install/uninstall PowerShell scripts, README, and the Windows payload zip, then writes installer-level manifest/SBOM/provenance sidecars; iteration 154 smoke produced a 28,409,202-byte installer zip with SHA-256 `89f9139cbc1d265d9633362dd5ed09b53100b30750bb2e7ed52175bde79dff67` (`go test ./internal/release ./cmd/release-manifest -run "Evidence|SBOM|Provenance|Manifest"` plus packaging smoke).
- [ ] P1 Produce macOS app/package artifact.
- [ ] P1 Produce Linux package artifact.
- [x] P1 Generate SHA-256 manifest. (`scripts/ci-windows.ps1` generated `build/nexusdesk-windows-manifest.json` with artifact SHA-256 during the iteration 80 Windows checkpoint.)
- [x] P1 Embed version metadata. Evidence: `internal/buildinfo` owns version/commit/build-date metadata, Windows CI injects it through `-ldflags`, build metadata validation runs in `scripts/ci-windows.ps1`, and status bar text includes the active version.
- [x] P1 Add About dialog version string. Evidence: Help > About calls `buildinfo.AboutText()`, which includes Version, Commit, and Build fields; covered by `TestAboutTextIncludesReleaseIdentity`.

### 6.2 Signing and trust

- [x] P1 Decide Windows certificate strategy. Evidence: `docs/adr/0001-release-signing-and-notarization.md` accepts an organization-owned OV/EV Windows code-signing certificate strategy with hardware-backed key storage preferred, `signtool`, RFC 3161 timestamping, and signed executable/installer release evidence requirements.
- [ ] P1 Sign Windows executable.
- [ ] P1 Sign Windows installer.
- [x] P1 Decide macOS signing/notarization strategy. Evidence: `docs/adr/0001-release-signing-and-notarization.md` accepts Developer ID Application signing for app bundles, Developer ID Installer signing for `.pkg`, Apple notarization/stapling for public artifacts, and documented quarantine/trust state for private beta builds.
- [ ] P1 Implement macOS signing/notarization if chosen.
- [x] P1 Document Linux trust/package dependencies. Evidence: Help > Release Hygiene now includes Linux Package Trust and Linux Runtime Dependencies sections covering hash-first manifest/SBOM/provenance verification, package format recording, repository-signing state, OpenGL/Wayland/X11 expectations, Secret Service/libsecret/secret-tool behavior, desktop entry/icon behavior, and package-level dependency notes (`go test ./internal/services/userguide -run "ReleaseHygiene"`).
- [x] P1 Add release trust diagnostics. Evidence: `release.RuntimeTrustDiagnostics` now reports unstamped build metadata plus missing manifest/signing/trust/smoke/SBOM/provenance evidence, and Diagnostics renders a Release trust health card plus report section (`go test ./internal/release ./internal/ui/shell -run "ReleaseTrust|RuntimeTrust|PackagingReadiness|Diagnostics"`).

### 6.3 SBOM and provenance

- [x] P1 Generate SBOM for release. Evidence: `cmd/release-manifest` now writes a CycloneDX 1.5 JSON SBOM from Go build info embedded in the release artifact, including the application component, artifact SHA-256, and Go module components (`go test ./internal/release ./cmd/release-manifest -run "Evidence|SBOM|Provenance|Manifest"`; CLI smoke generated `nexusdesk-windows-sbom.json`).
- [x] P1 Generate provenance evidence. Evidence: `release.WriteEvidenceSet` writes provenance JSON with subject build metadata, artifact hash/size, repository/workflow/source commit fields, and hashed manifest/SBOM evidence entries; the release-manifest CLI smoke generated `nexusdesk-windows-provenance.json`.
- [x] P1 Store release evidence next to artifacts. Evidence: manifest, SBOM, and provenance paths derive from the manifest path (`nexusdesk-<platform>-manifest.json`, `nexusdesk-<platform>-sbom.json`, `nexusdesk-<platform>-provenance.json`), and both Windows/Unix CI scripts verify and clean the sidecar evidence files.
- [x] P1 Document verification steps. Evidence: Help > Release Hygiene now includes Release Verification Steps covering About metadata matching, artifact SHA256 manifest verification, SBOM/provenance review, sidecar archiving, clean-machine smoke, protected-secret smoke, uninstall/app-data cleanup smoke, and Diagnostics release trust checks (`go test ./internal/services/userguide -run "ReleaseHygiene"`).

### 6.4 CI matrix

- [x] P1 Windows CI: gofmt check. Evidence: `scripts/ci-windows.ps1` passed gofmt verification in the iteration 140 full Windows checkpoint.
- [x] P1 Windows CI: tests. Evidence: `scripts/ci-windows.ps1` ran `go test ./...` successfully in the iteration 140 full Windows checkpoint.
- [x] P1 Windows CI: build check. Evidence: `scripts/ci-windows.ps1` built the native Windows executable successfully in the iteration 140 full Windows checkpoint.
- [x] P1 Windows CI: release manifest check. Evidence: `scripts/ci-windows.ps1` generated manifest/SBOM/provenance release evidence successfully in the iteration 140 full Windows checkpoint.
- [ ] P1 Linux CI: tests.
- [ ] P1 Linux CI: build/package smoke.
- [ ] P1 macOS CI: tests.
- [ ] P1 macOS CI: build/package smoke.
- [ ] P1 Cross-platform protected secret smoke.
- [x] P1 CI avoids leaving generated binaries in workspace. Evidence: iteration 140 full Windows checkpoint completed and the cleanup block removed `build/nexusdesk.exe`, `build/nexusdesk-windows-manifest.json`, `build/nexusdesk-windows-sbom.json`, and `build/nexusdesk-windows-provenance.json`; only the tracked `build/windows-resource` directory remained.

### 6.5 Clean-machine smoke

- [ ] P1 Windows clean-machine launch smoke.
- [ ] P1 Windows open workspace smoke.
- [ ] P1 Windows edit/save/revert smoke.
- [ ] P1 Windows assistant setup smoke.
- [ ] P1 Windows data/artifact smoke.
- [ ] P1 Windows diagnostics/export smoke.
- [ ] P1 macOS clean-machine smoke.
- [ ] P1 Linux clean-machine smoke.
- [x] P1 Uninstall/app-data cleanup smoke. Evidence: `scripts/smoke-windows-installer.ps1` builds the installer bundle, installs into an isolated temp directory without Start Menu shortcuts, verifies executable/release-evidence files, runs the uninstaller, confirms application files are removed, and confirms workspace `.nexusdesk` data is preserved; iteration 156 smoke passed with `nexusdesk-windows-installer-0.0.0-iter156.zip`.

### 6.6 Update visibility

- [x] P2 Add Help > Check for updates. Evidence: Help menu and command palette expose `Check for Updates`, opening a normal Updates guide tab with current version/commit/build-date (`go test ./internal/services/userguide ./internal/ui/shell -run "UpdateCheck|MainMenuHelpGroupExposesUpdateCheck|CommandPaletteIncludesSafeAgentGuide"`).
- [x] P2 Show available update non-modally. Evidence: update guidance opens through `addPlaceholderTab("Updates", ...)` instead of a blocking dialog, so users can compare release notes without interrupting active work.
- [x] P2 Do not auto-download. Evidence: `UpdateCheckGuide` explicitly states NexusDesk does not download update artifacts automatically and exposes only manual verification guidance.
- [x] P2 Do not auto-install. Evidence: `UpdateCheckGuide` explicitly states NexusDesk does not install updates automatically and routes users through release notes plus manifest/SBOM/provenance verification.
- [x] P2 Add release notes link. Evidence: update guidance points users to `docs/releases/beta-release-notes.md`, and the command palette detail advertises release notes as part of the update-check path.

## Phase 7: Private Beta

- [x] P1 Create first-run onboarding flow. Evidence: Welcome now includes a compact First Run flow with provider setup/Test connection, trusted workspace, Sample Workflow, and Diagnostics/redacted issue-report steps, plus direct action buttons for Provider Setup, Sample Workflow, workspace/file open, and Diagnostics (`go test ./internal/ui/shell -run "Welcome|Settings"`).
- [x] P1 Add provider setup wizard. Evidence: Help, command palette, and Welcome now expose Provider Setup Wizard, backed by `userguide.ProviderSetupWizardMarkdown()` with provider/endpoint, detected model suggestion, protected credential, Test connection, Diagnostics, and route-default steps (`go test ./internal/services/userguide ./internal/ui/shell -run "ProviderSetup|CommandPaletteIncludesSafeAgentGuide|Welcome"`).
- [x] P1 Add model auto-suggestion from detected provider models. Evidence: Settings Test connection now suggests the first detected provider model when the chat model is blank or missing, applies that suggestion to the global model field, reports it in the probe summary, and does the same for selected task routes (`go test ./internal/ui/shell -run "Settings"`).
- [x] P1 Add sample workflow guide. Evidence: Help and the command palette expose Sample Workflow Guide, backed by `userguide.SampleWorkflowMarkdown()` with a safe end-to-end beta path covering workspace readiness, edit/revert, Ask with sources, low-risk Agent, Data/Artifacts, Diagnostics, and redacted issue-report closeout (`go test ./internal/services/userguide ./internal/ui/shell -run "SampleWorkflow|KnownLimitations|CommandPaletteIncludesSafeAgentGuide"`).
- [x] P1 Add safe-agent user guide. Evidence: Help and the command palette expose Safe Agent Guide, backed by `userguide.SafeAgentMarkdown()` with approval, rollback, secret, connector, job, and diagnostic safety guidance (`go test ./internal/services/userguide ./internal/ui/shell -run "SafeAgent|CommandPaletteIncludesSafeAgentGuide"`).
- [x] P1 Add beta feedback issue template. Evidence: `.github/ISSUE_TEMPLATE/beta-feedback.yml` captures app version/commit/build date, affected area, goal/expected/actual result, reproduction steps, redacted diagnostics notes, and a required secret/workspace-data redaction checklist.
- [x] P1 Add crash recovery banner actions. Evidence: Welcome renders a Crash Recovery card when startup recovery detects an unclean previous session, with direct actions for Diagnostics, Jobs, Agent Audit, and History plus guidance to export a redacted issue report before retrying long workflows (`go test ./internal/ui/shell -run "Welcome|Startup|Crash"`).
- [x] P1 Add known limitations page. Evidence: Help and the command palette expose Known Limitations, backed by `userguide.KnownLimitationsMarkdown()` with beta boundaries for packaging/trust, provider/model setup, planned tools, connector/data limits, platform coverage, and protected-secret backends (`go test ./internal/services/userguide ./internal/ui/shell -run "SampleWorkflow|KnownLimitations|CommandPaletteIncludesSafeAgentGuide"`).
- [x] P1 Prepare beta release notes. Evidence: `docs/releases/beta-release-notes.md` documents private-beta scope, ready-to-exercise features, install/trust state, release evidence verification, validation steps, known limitations, uninstall/app-data expectations, and redacted feedback instructions.
- [ ] P1 Run five-user beta install test.
- [ ] P1 Triage beta feedback within 48 hours.

## Phase 8: Release Candidate

- [x] P0 Freeze v1 feature scope. Evidence: `docs/releases/v1-scope-freeze.md` freezes the v1 promise, in-scope surface, post-v1 deferrals, release blockers that do not expand scope, and change-control rule; docs index links it as release-specific source of truth.
- [ ] P0 Close all P0 issues.
- [ ] P0 Review all P1 issues and explicitly defer or fix.
- [ ] P0 Run full test suite in CI.
- [ ] P0 Run full platform smoke.
- [x] P0 Run security/safety review. Evidence: `docs/releases/security-safety-review.md` records the v1 security/safety review scope, verified controls, residual release blockers, and pass-with-blockers decision; focused safety matrix passed across security, tools, agent, tasks, workspace, connectors, issue reports, protected secrets, and shell UI (`go test ./internal/services/security ./internal/services/tools ./internal/services/agent ./internal/services/tasks ./internal/services/workspace ./internal/services/dbconnector ./internal/services/issuereport ./internal/services/protectedsecret ./internal/ui/shell -run "Risk|Threat|Control|Approval|Safety|Mutation|Timeout|Loop|Shell|Path|Rollback|Redact|Secret|Plaintext|WorkspaceOpen|ReadOnly|Unsafe"`).
- [ ] P0 Run performance review.
- [ ] P0 Run accessibility review.
- [x] P0 Verify no hidden workspace-open side effects. Evidence: workspace-open policy tests allow only metadata/history/audit/approval/navigator/pin refresh actions, reject slow/heavy workflow kinds, and `TestOpenWorkspaceDoesNotStartHiddenJobs` verifies opening a workspace does not create hidden jobs or job-start activity (`go test ./internal/ui/shell -run "WorkspaceOpen|OpenWorkspace"`).
- [x] P0 Verify no known file data-loss path. Evidence: focused file-safety matrix passed for safe write proposals, rollback creation/application, unsafe target rejection, full safe text reads, partial-preview save blocking, ambiguous-encoding save blocking, encoding state tracking, save-state visibility, and in-place post-save refresh (`go test ./internal/services/workspace ./internal/ui/shell -run "ApplyFileWrite|ApplyRollback|PreviewFileWrite|PreviewFileAppend|ReadTextFile|EditorSaveAllowed|TextEditorBinding|EditorSaveState|RefreshEditorAfterSave"`).
- [x] P0 Verify no known plaintext secret storage path. Evidence: focused secret-safety matrix passed for Windows DPAPI/protected-secret round trips, provider API-key protected sidecar storage and redacted display, connector credential references and redaction, issue-report redaction, provider error redaction, operations redaction, diagnostics protected-secret/connector plaintext status, and explicit development-plaintext transport audit paths (`go test ./internal/services/protectedsecret ./internal/services/settings ./internal/services/dbconnector ./internal/services/issuereport ./internal/services/llm ./internal/services/tools ./internal/services/operations ./internal/ui/shell -run "Secret|APIKey|Credential|Redact|Redacted|Protected|Plaintext|AuthHeader"`).
- [x] P0 Verify release artifacts and hashes. Evidence: `scripts/verify-release-evidence.ps1` validates artifact name, byte size, SHA-256, manifest JSON, SBOM CycloneDX component hash, provenance subject identity, and provenance evidence hashes; iteration 162 generated `nexusdesk-windows-installer-0.0.0-iter162.zip` and verified SHA-256 `664c3aa92978e107c73d539572720406a36815e4d53d5c73b3e4195fbb561f9c` against manifest/SBOM/provenance.
- [x] P0 Verify docs match shipped behavior. Evidence: `docs/03_FEATURES.md` now reflects completed shell, editor, search, assistant, data, connector, artifact, jobs/settings/diagnostics behavior instead of stale planned labels; obsolete web-runtime path skips were removed from task discovery; repository sweep found no obsolete web-runtime references (`rg -n "Wails|wails|WailsJS|webview|frontend/dist|app/frontend/dist|app/build/bin|build/bin" . -g "!docs/brand/logos/png/**"`), `go test ./internal/services/tasks -run "Discover|RunRejectsUnknownTask|Find"` passed, and `git diff --check` passed.
- [ ] P0 Publish release notes.

## Ongoing Rules For Every Development Slice

- [ ] Check git status before editing.
- [ ] Avoid overwriting unrelated user changes.
- [ ] Choose one logical milestone.
- [ ] Add or update focused tests for code changes.
- [ ] Update docs and this tracker when behavior or plan changes.
- [ ] Run `gofmt` for Go changes.
- [ ] Run focused tests first.
- [ ] Run `go test ./...` before milestone completion when local toolchain supports it.
- [ ] Run `go build .` or platform build-check when local toolchain supports it.
- [ ] Run `git diff --check`.
- [ ] Remove generated binaries unless a runnable local build was explicitly requested.
- [ ] Commit only when the user asks or the current workflow explicitly requires it.
- [ ] Never force-push.

## Progress Estimate

Planning-only estimate after the documentation reset:

- Architecture foundation: 90% to production target.
- Core feature breadth: 90% to production target.
- Safety hardening: 75% to production target.
- Performance hardening: 70% to production target.
- UI target polish: 60% to production target.
- Packaging/release trust: 45% to production target.
- Overall v1 readiness: 75-80%.

These numbers are intentionally conservative. The app has many capabilities, but production readiness depends on safety, polish, packaging, and smoke evidence, not only feature count.
