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
- [x] P1 Remove Wails/webview-era code, docs, and build metadata. (Repository sweep found no Wails/WailsJS/webview references; stale `frontend/dist`, `build/bin`, and `app/build/bin` ignore entries were removed so the repo reflects the Fyne-native app layout.)
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
- [ ] P1 Parse markdown once at final response where practical.
- [x] P1 Coalesce agent event rendering.
- [ ] P1 Replace activity markdown rebuild with incremental list rendering.
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
- [ ] P1 Extract editor controller.
- [ ] P1 Extract assistant controller.
- [ ] P1 Extract data controller.
- [ ] P1 Extract Git controller.
- [ ] P1 Extract artifacts controller.
- [ ] P1 Extract jobs controller.
- [ ] P1 Extract diagnostics controller.
- [ ] P1 Extract approvals/audit controller.
- [ ] P1 Shrink `View` to layout and registry ownership.
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
- [ ] P1 Add resize handle with enlarged hit zone.
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

- [ ] P1 Improve Data Sources tree icons and hierarchy.
- [ ] P1 Add schema/table browser polish.
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

- [ ] P1 Polish DOCX report templates.
- [ ] P1 Polish PPTX deck templates.
- [ ] P1 Add cross-suite DOCX smoke.
- [ ] P1 Add cross-suite PPTX smoke.
- [ ] P1 Expand artifact regeneration coverage.
- [ ] P1 Improve artifact freshness visualization.
- [ ] P1 Improve lineage visualization.
- [ ] P2 Add OCR/scanned document extraction job.

### 5.4 Assistant quality

- [ ] P1 Improve source ranking.
- [ ] P1 Improve source coverage UI.
- [ ] P1 Improve uncited source warnings.
- [ ] P1 Improve stale source prompts.
- [ ] P1 Add model route recommendations in Settings.
- [ ] P1 Add context budget visualization.
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
- [ ] P1 Produce Windows zip artifact.
- [ ] P1 Produce Windows installer artifact.
- [ ] P1 Produce macOS app/package artifact.
- [ ] P1 Produce Linux package artifact.
- [x] P1 Generate SHA-256 manifest. (`scripts/ci-windows.ps1` generated `build/nexusdesk-windows-manifest.json` with artifact SHA-256 during the iteration 80 Windows checkpoint.)
- [ ] P1 Embed version metadata.
- [ ] P1 Add About dialog version string.

### 6.2 Signing and trust

- [ ] P1 Decide Windows certificate strategy.
- [ ] P1 Sign Windows executable.
- [ ] P1 Sign Windows installer.
- [ ] P1 Decide macOS signing/notarization strategy.
- [ ] P1 Implement macOS signing/notarization if chosen.
- [ ] P1 Document Linux trust/package dependencies.
- [ ] P1 Add release trust diagnostics.

### 6.3 SBOM and provenance

- [ ] P1 Generate SBOM for release.
- [ ] P1 Generate provenance evidence.
- [ ] P1 Store release evidence next to artifacts.
- [ ] P1 Document verification steps.

### 6.4 CI matrix

- [x] P1 Windows CI: gofmt check. (`scripts/ci-windows.ps1` passed gofmt verification on Windows in iteration 80.)
- [x] P1 Windows CI: tests. (`scripts/ci-windows.ps1` ran `go test ./...` successfully on Windows in iteration 80.)
- [x] P1 Windows CI: build check. (`scripts/ci-windows.ps1` built the native Windows executable successfully on Windows in iteration 80.)
- [x] P1 Windows CI: release manifest check. (`scripts/ci-windows.ps1` generated the release manifest successfully on Windows in iteration 80.)
- [ ] P1 Linux CI: tests.
- [ ] P1 Linux CI: build/package smoke.
- [ ] P1 macOS CI: tests.
- [ ] P1 macOS CI: build/package smoke.
- [ ] P1 Cross-platform protected secret smoke.
- [ ] P1 CI avoids leaving generated binaries in workspace.

### 6.5 Clean-machine smoke

- [ ] P1 Windows clean-machine launch smoke.
- [ ] P1 Windows open workspace smoke.
- [ ] P1 Windows edit/save/revert smoke.
- [ ] P1 Windows assistant setup smoke.
- [ ] P1 Windows data/artifact smoke.
- [ ] P1 Windows diagnostics/export smoke.
- [ ] P1 macOS clean-machine smoke.
- [ ] P1 Linux clean-machine smoke.
- [ ] P1 Uninstall/app-data cleanup smoke.

### 6.6 Update visibility

- [ ] P2 Add Help > Check for updates.
- [ ] P2 Show available update non-modally.
- [ ] P2 Do not auto-download.
- [ ] P2 Do not auto-install.
- [ ] P2 Add release notes link.

## Phase 7: Private Beta

- [ ] P1 Create first-run onboarding flow.
- [ ] P1 Add provider setup wizard.
- [ ] P1 Add model auto-suggestion from detected provider models.
- [ ] P1 Add sample workflow guide.
- [ ] P1 Add safe-agent user guide.
- [ ] P1 Add beta feedback issue template.
- [ ] P1 Add crash recovery banner actions.
- [ ] P1 Add known limitations page.
- [ ] P1 Prepare beta release notes.
- [ ] P1 Run five-user beta install test.
- [ ] P1 Triage beta feedback within 48 hours.

## Phase 8: Release Candidate

- [ ] P0 Freeze v1 feature scope.
- [ ] P0 Close all P0 issues.
- [ ] P0 Review all P1 issues and explicitly defer or fix.
- [ ] P0 Run full test suite in CI.
- [ ] P0 Run full platform smoke.
- [ ] P0 Run security/safety review.
- [ ] P0 Run performance review.
- [ ] P0 Run accessibility review.
- [ ] P0 Verify no hidden workspace-open side effects.
- [ ] P0 Verify no known file data-loss path.
- [ ] P0 Verify no known plaintext secret storage path.
- [ ] P0 Verify release artifacts and hashes.
- [ ] P0 Verify docs match shipped behavior.
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
