# NexusDesk Full Project Review And Roadmap

Date: 2026-05-29
Status: active consolidated review and execution guide
Primary app: `nexus-app/`
Reference app: `app-wails/`

This review combines the current Codex project review, the latest `claude-findings.md` static review, the Wails feature inventory, the production readiness plan, the agent tool registry, and the JetBrains-like UI target. It is intentionally direct: NexusDesk is close to Native Parity Beta, but it is not production-ready until the safety, performance, packaging, and UI-coherence risks below are closed or explicitly accepted.

`tracker.md` remains the task-level checklist. `docs/17_END_TO_END_PRODUCTION_PLAN.md` remains the product north star. This document is the current full-review snapshot that explains what the app can do, what is planned, and what the next LLM development sessions should prioritize.

## 1. Executive Status

Current estimate:

- Fyne-native migration: about 98% complete.
- Useful Wails-era parity: about 97% complete.
- Native Parity Beta readiness: about 96% complete.
- Overall production readiness: about 95% complete.
- Distribution and packaging readiness: about 80% complete.

High-level assessment:

- The Fyne migration remains the right architecture for a native local-first IDE/data/document/operations studio.
- `nexus-app/` is the active product. `app-wails/` is reference-only until explicit freeze/retirement.
- The service architecture is strong: domain and services are framework-free, UI owns Fyne, and import-boundary tests prevent Wails/webview or Fyne leakage from returning.
- The first-party agent toolbelt is already large and practical, with implemented registry validation and a planned-tool contract.
- The biggest remaining risks are not broad feature absence. They are production trust: data-loss prevention, untrusted-file caps, platform secret hardening, SSRF hardening, large-session performance, signed packaging, clean-machine smoke, and UI/controller complexity.
- The final UI direction is clear: compact JetBrains-like native workbench, not a browser dashboard and not a pile of debug panels.

## 2. Architecture Review

### 2.1 What Is Healthy

- `nexus-app/main.go` is thin and delegates lifecycle assembly.
- `internal/app` owns native lifecycle and window setup.
- `internal/domain` owns framework-free domain models.
- `internal/services` owns workspace, agent, tools, approvals, artifacts, metadata, jobs, assistant, settings, data, documents, operations, release, security, and persistence behavior.
- `internal/ui` owns Fyne shell, widgets, dialogs, panels, menus, and theme.
- `internal/architecture` has import-boundary tests for no Wails/webview, no Fyne in services/domain, and no UI imports from service/domain packages.
- Slow work has a documented job contract and multiple concrete job-backed workflows already exist.
- File mutation paths are rooted, rollback-aware, and approval-aware where risk requires it.
- Metadata is durable and recoverable through SQLite-backed stores and backup/export paths.
- Agent tools have risk descriptors, catalog validation, approval expectations, and diagnostics visibility.
- Release packaging now has a framework-free readiness evaluator for manifest, format, signing/trust, install/update/uninstall smoke, clean-machine smoke, protected-secret smoke, and antivirus/release-note evidence.

### 2.2 Architecture Risks

These should become near-term milestones rather than vague cleanup:

- `internal/ui/shell` remains the largest orchestration area and still carries controller pressure.
- Large files such as data, artifacts, assistant, diagnostics, and settings panels increase refactor risk and make UI polish harder.
- Some UI actions still need background dispatch/throttling to avoid main-thread stalls during large saves, streaming, activity updates, and data/search rendering.
- Some security-sensitive platform paths require runtime validation on real Windows/macOS/Linux machines, not just static tests.
- Packaging is now better specified, but signed installers, notarization, Linux packaging, update validation, and installer/uninstall smoke remain open.

### 2.3 Architecture Direction

- Keep services framework-free.
- Keep Fyne inside UI/app/theme only.
- Continue extracting shell controllers by responsibility: editor, data, artifacts, assistant, jobs, diagnostics, Git, settings.
- Treat every slow or risky workflow as a job with logs, cancellation, retry, audit, and output opening.
- Keep folder open cheap forever: no Git, Docker, OCR, connector pulls, dump imports, model calls, shell commands, browser automation, or deep indexing.
- Do not reintroduce Wails/webview to solve editor or browser problems.

## 3. Professional UI And Product Design Review

### 3.1 Current Design Strength

NexusDesk already has the important ingredients of a professional native studio:

- JetBrains-like top menu groups: File, Edit, View, Navigate, Code, Refactor, Run, Tools, Help.
- Top toolbar for workspace, branch/status, tasks, provider/model, search, command palette, settings.
- Left and right tool-window rails with keyboard-reachable actions.
- Central tabbed editor and artifact/data/document surfaces.
- Right integrated assistant with Ask/Agent mode, source diagnostics, and run status.
- Bottom workbench panels for problems, Git, tasks/jobs, data details, diagnostics, and activity.
- Compact dark native theme tokens and density direction.
- DataGrip-like goals for grids, schema/data surfaces, and connector feedback.
- Searchable settings and diagnostics that explain provider, metadata, jobs, performance, startup, tool registry, artifact provenance, and release trust state.

### 3.2 UI Still Needed Before Production

- Reduce crowded/debug feeling in bottom panels by grouping tool windows more consistently.
- Extract UI controllers so polish can continue without growing the `View` god object.
- Make empty states calmer and more instructional across Data, Artifacts, Assistant, Jobs, Diagnostics, and Operations.
- Add stronger keyboard focus rules for every pane and dialog.
- Throttle streaming and activity rendering so long assistant/agent runs feel smooth.
- Move heavy save/search/diff operations off the UI thread where they can block Fyne.
- Polish DataGrip-like grids: schema navigation, row/column affordances, filtering, saved query flow, and result detail surfaces.
- Polish assistant hierarchy: clearer source cards, tool timeline, approvals, citations, and run recovery.
- Polish generated DOCX/PPTX template controls and preview surfaces.

## 4. What NexusDesk Can Do Already

### 4.1 Workspace And Workbench

- Launch as a native Fyne desktop app.
- Open local folders with native dialogs.
- Preserve recent workspaces.
- Show a Home/readiness cockpit for workspace open, provider/model setup, credentials, native toolchain, local safety posture, and quick actions.
- Browse lazy project trees with ignored-path handling, entry caps, refresh, reveal, collapse, context menus, and safe file operations.
- Keep workspace open bounded and cheap by design.
- Use quick open and command palette workflows.
- Use JetBrains-like menu/toolbar/rail/status shell structure.

### 4.2 Editor And Navigation

- Preview text, code, Markdown, images, CSV/TSV, DOCX, PDF text, XLSX-derived content, and binary metadata.
- Edit text/code drafts with dirty markers, pinned tabs, close guards, save, revert, rollback, and discard confirmation.
- Use safe write, append, copy, move, delete, rename, and folder-create services.
- Use Markdown source/rendered toggle.
- Use find/replace with match counts.
- Use formatting for Go, JSON, Markdown/config/SQL/Dockerfile/text and recognized whitespace-safe formats.
- Use breadcrumbs, split preview, outline, go-to-symbol, local go-to-definition, bounded workspace definition fallback, find references, document map, and syntax mirror.
- Use live draft diagnostics for markers, merge conflicts, JSON, Go, YAML, TOML, and XML.
- Use Problems scanning for saved files and markers.

### 4.3 Search And Problems

- Search workspace paths and file contents with snippets and multiple matches.
- Persist bounded search metadata manifests and quarantine corrupt search metadata.
- Scan TODO/FIXME/HACK/BUG markers, merge conflicts, and syntax diagnostics.
- Surface search/problems through bottom panels and assistant tools.

### 4.4 Assistant And Agent

- Configure OpenAI-compatible, Ollama, and custom provider endpoints.
- Store API keys in OS-protected storage or explicitly refuse unsupported platforms.
- Probe providers, count models, inspect runtime status, and show remediation guidance.
- Use Ask and Agent modes with streaming, cancellation, context packs, chat history, retry, compare, and save answer.
- Use prompt profiles and assistant memory.
- Use task-aware model defaults for coding, backend, database, analytics, research, vision/screenshot, balanced reasoning, and fast coding routes.
- Use auto/global/manual model routing with context-budget visibility.
- Persist model/route provenance in chat answers and agent audit.
- Show evidence quality, line-aware citations, source coverage, stale-source warnings, unverified/out-of-context citation diagnostics, and bounded snippets.
- Run agent plans with deterministic tools, bounded observations, approvals, audit, rollback, and final fallback behavior.

### 4.5 Implemented Agent Toolbelt

The native agent can already use tools for:

- Tool registry inspection.
- Workspace reads, file reads, search, problems, definition, references, dependency graph, symbol index, and project memory update.
- Safe file writes, appends, copy, move, delete, patch, rollback listing, and rollback application.
- Editor formatting and lint diagnostics.
- Git status, diff, history, blame, file/hunk stage/unstage, commit staged changes, create branch, resolve conflicts, revert unstaged changes, and revert staged changes.
- Task discovery and safe task execution.
- One-shot approved terminal commands with argv, rooted cwd, timeout, output caps, shell/path blocking, and audit.
- Durable job listing, log reading, and cancellation.
- Web fetch for bounded HTTP(S) text retrieval.
- Artifact lineage and supported artifact regeneration.
- Dataset profiling, bounded queries, SELECT-only SQL, and chart artifact generation.
- Workspace SQLite inspection and read-only query.
- Document text extraction.
- Operations file inspection and runbook generation.
- Redaction, approval listing, and approval request records.
- External coding-agent CLI readiness detection and non-executing run plans for Codex, Claude Code, and OpenCode.

### 4.6 Git, Tasks, Jobs

- Refresh Git status manually.
- Show project tree Git badges.
- Group changed files.
- View unified/split/diff-only diffs.
- Use hunk-windowed large-file-aware diffs.
- Stage/unstage files and hunks.
- Generate AI diff summaries and commit drafts.
- Read Git history and blame.
- Discover safe tasks.
- Run tasks as jobs with logs, cancellation, retry, and report artifacts.
- Persist job/task records in metadata.

### 4.7 Data And Analytics

- Profile CSV, TSV, JSON, NDJSON, XLSX, Parquet metadata, and logs.
- Query, filter, order, and limit local datasets.
- Run SELECT-only dataset SQL with persisted run/dependency metadata.
- Use SQL notebook flows with SQL/chart cells, directives, save/load, run/export, result tabs, and artifacts.
- Browse workspace SQLite schema, views, indexes, relationships, row counts, samples, saved queries, query history, and exports.
- Manage external database profiles for PostgreSQL, MySQL/MariaDB, SQL Server, SQLite, and DuckDB guarded builds.
- Run read-only connector tests, inspections, queries, cancellation, redaction, history, and credential sidecars.
- Generate chart and dashboard SVG previews/artifacts.

### 4.8 Artifacts, Documents, Operations

- Browse, search, preview, archive, delete, restore, compare, and inspect artifacts.
- Track source lineage, fingerprints, freshness warnings, metadata sidecars, and provenance health.
- Export/import artifact lineage graph JSON.
- Pin artifacts into assistant/agent context.
- Regenerate supported artifacts from dependency/source metadata.
- Create chart, dashboard, notebook, document report, document brief, DOCX export, workspace scan, operations runbook, task report, chat answer, comparison, presentation outline, packaged presentation zip, and PPTX deck artifacts.
- Validate DOCX/PPTX package structure and theme metadata.
- Extract text from supported documents.
- Inspect Dockerfiles, Compose, env/config/script/log evidence read-only.
- Generate operations runbooks from redacted evidence.

### 4.9 Diagnostics, Safety, Release

- Persist chats, approvals, artifacts, jobs, SQL runs, dataset dependencies, agent runs, and tool runs.
- Import Wails-era metadata where useful.
- Recover corrupt metadata and export backups.
- Show diagnostics for providers, metadata, jobs, tasks, SQL, agent failures, app logs, performance, startup recovery, issue reports, tool registry health, artifact provenance, and release trust direction.
- Use protected secrets for API keys and connector credentials.
- Enforce path-root, traversal, symlink, ignored-state, and `.nexusdesk` protections.
- Record rollback snapshots for practical file mutations.
- Validate build metadata through ldflags.
- Run cross-platform native CI smoke for formatting, tests, vet, Fyne build, release manifest, and diff hygiene.
- Evaluate packaging readiness evidence for Windows/macOS/Linux.

## 5. Planned Functionality

### 5.1 Native Parity And Editor

- Post-beta editable inline syntax styling if it remains safe and accessible.
- Packaged LSP provider spike behind a feature flag.
- Deeper cross-file semantic navigation, rename, code actions, diagnostics, and test generation.
- More polished editor chrome, split groups, search result navigation, and document map visuals.
- Safer large/truncated file editing behavior.

### 5.2 Assistant, Agent, And Tools

- Deeper retrieval/ranking quality beyond deterministic citation/source coverage.
- More source diagnostics for partial, stale, weak, contradictory, missing, or uncited evidence.
- Full tool-run provenance for every generated output type.
- Rendered browser automation: navigate, click, type, screenshot, extract page, inspect network metadata.
- Interactive terminal sessions with durable supervision, logs, cancellation, audit, and approvals.
- GitHub/PR connector tools: draft PR, create PR, read comments/reviews.
- Semantic search and indexed project memory.
- MCP discovery and permissioned MCP tool calls.
- Plugin discovery/install with signing/trust controls.
- Scheduled/recurring automations with visible ownership and audit.

### 5.3 Data, Documents, Connectors

- Dump import job design and isolated execution.
- Connector sync job model.
- Google Analytics, ads exports/importers, CRM/contact connectors, cloud storage connectors, and cross-source analysis workflows.
- Richer DOCX/PPTX template variants, cross-suite compatibility smoke, and visual design controls.
- OCR/scanned PDF/image extraction pipeline routed through durable jobs.
- Document set comparison/version workflows.
- Deeper DataGrip-like schema navigation and query UX.

### 5.4 Operations And Security

- Docker/system mutations only after mature approval, jobs, audit, redaction, mitigation, and UX design.
- Audit coverage for future OCR, connector sync, dump import, shell, terminal sessions, browser automation, and Docker workflows.
- Stronger connector TLS defaults and user-visible downgrade warnings.
- SSRF hardening for web fetch and future browser/network tools.
- Secret-storage runtime smoke on Windows, macOS, and Linux release targets.

### 5.5 Packaging, Release, Beta

- Signed Windows installer and release flow.
- macOS packaging, signing, and notarization.
- Linux package strategy and smoke: AppImage, deb/rpm, Flatpak, repository, or documented portable package.
- Installer/update/uninstall validation wired into the packaging readiness gate.
- SBOM/provenance for release artifacts.
- Update visibility, at minimum a check-for-updates action that opens or reads GitHub releases.
- Private beta clean-machine validation and release notes per platform.

## 6. Claude Findings Integration

The latest `claude-findings.md` is useful and should be treated as the current risk backlog. It also notes several older findings that are already resolved, including API key configurability, agent shell approval dead-end, symlink read protection, search one-match-per-file, activity buffer growth, bottom-tab overload baseline, and metadata DB reopen behavior.

### 6.1 Critical / P0 Findings To Prioritize

1. Preview truncation data-loss path: add top-level truncation metadata to `domain.FilePreview`, disable inline save for truncated files, and surface the partial-read state to agent observations.
2. XLSX/DOCX/PPTX zip decompression caps: implemented shared zip file-count, package-member, total-uncompressed, XLSX XML-member, DOCX body, and structured-preview compressed-size caps before parsing.
3. macOS Keychain argv secret leak: implemented Security.framework-backed storage so secrets are passed as private bytes rather than process arguments; macOS clean-machine smoke still needs to verify Keychain prompts and signing behavior.
4. Windows DPAPI buffer pinning: implemented `runtime.KeepAlive` after native calls, hardened `LocalFree` cleanup, and added Windows round-trip coverage.
5. DNS rebinding SSRF in `web_fetch`: implemented a guarded HTTP transport that re-resolves and rejects private, loopback, link-local, multicast, and unspecified targets at dial time, with redirect validation retained.
6. Workspace search performance: implemented bounded byte/text fast path with binary and structured-preview skips so search no longer full-decodes/classifies every candidate file.
7. External DB TLS defaults: require encrypted connections by default and make plaintext an explicit audited development-only choice.
8. Metadata SQLite WAL/busy timeout: reduce UI/agent write contention under long sessions.
9. Streaming/activity throttling: coalesce per-token and activity updates to avoid repeated full Markdown parsing.
10. Editor save off UI thread: run heavy save/diff/rollback work asynchronously and avoid rebuilding editor panels unnecessarily.
11. UI shell controller extraction: reduce `View` and large panel files by extracting controller/state ownership.

### 6.2 High / P1 Findings

- LCS diff and rollback snapshots need large-file memory bounds.
- Bottom tool-window navigation still needs final grouping polish.
- SQL guard should avoid blocking valid keywords inside literals and needs more fuzz/property tests.
- Git command timeout should be command-aware rather than universally short.
- Mutating tool verification should trust `ToolResult.Mutated` and mark all mutating handlers correctly.
- Task execution should use safe argv forms rather than shell wrappers where possible.
- Connector pool lifecycle should avoid leaking handles between profile tests.
- Split-editor secondary previews should refresh after saves.
- Search should stop silently missing matches beyond preview caps.
- Windows reserved names, drive-like paths, and alternate streams need path-policy hardening.
- Agent runs need a wall-clock deadline.
- Job IDs should avoid predictable collision-prone counters.
- Connector profile save errors and protected-secret unavailability need clearer remediation.

### 6.3 Medium / P2 Findings

- Web fetch regex/title extraction performance and correctness.
- Job log retention and full-log access.
- Recent workspace existence validation.
- Issue-report filename UTC consistency.
- Welcome tab cleanup after first workspace open.
- Notebook unsupported-cell reporting.
- SSE compatibility documentation.
- Chart theme/color safety.
- Default model first-run selection flow.
- Generated docs/tool registry drift checks.
- Threat-model coverage for artifact regeneration.
- Freeze marker and CI guard for `app-wails/` once native parity is declared.
- Platform-specific protected-secret tests in CI.
- Long agent run stress tests.
- SBOM, provenance, update channel, and release artifact signing.

### 6.4 Complete Claude Findings Checklist

Every current `claude-findings.md` item is tracked here so no finding is lost while individual milestones move through `tracker.md`.

#### Critical

- [x] C-1.1 Workspace search no longer full-decodes every candidate file: content matching now uses a bounded byte/text fast path, skips binary and structured preview formats, preserves per-file match caps, and keeps matcher reuse; UI-level latest-request cancellation/debouncing remains a shell-controller follow-up if needed.
- [x] C-1.2 macOS Keychain no longer passes secrets on process argv: the darwin backend now uses Security.framework via cgo, passes secret bytes directly to Keychain APIs, has a no-cgo refusal path, and includes stubbed regression coverage that exercises the native backend interface without a real keychain.
- [x] C-1.3 XLSX/DOCX/PPTX zip preview has decompression caps: shared safe zip guards cap file count, total uncompressed size, package members, XLSX metadata/worksheet reads, DOCX body reads, and structured preview compressed package size; DOCX/PPTX generated export validation already enforces required-part/XML caps.
- [x] C-1.4 Windows DPAPI blob calls pin input buffers and harden `LocalFree` handling: `Protect`/`Unprotect` keep Go slices alive after native calls, propagate `LocalFree` failures when output cleanup fails, and cover Windows DPAPI round-trip behavior.
- [ ] C-1.5 UI shell `View` is a god object: Search, Jobs, Rollbacks, Diagnostics, Git, Artifacts, Assistant, and Editor panel state/actions are now isolated in controllers; continue extracting controllers/state for Data and Settings.

#### High

- [ ] H-2.1 Editor save runs on the UI thread: make save/diff/rollback async, show saving state, and avoid full panel rebuild after save.
- [ ] H-2.2 SQLite metadata store lacks WAL/busy timeout: open with WAL/foreign-key/busy-timeout pragmas and surface journal mode in Diagnostics.
- [ ] H-2.3 Streaming chat reparses Markdown per delta: coalesce updates, throttle UI refresh, and render final Markdown once.
- [x] H-2.4 Top-level text preview truncation is surfaced: `FilePreview.Truncated`/byte metadata exists, truncated editor previews are read-only/save-blocked, and agent `read_file` observations disclose partial reads.
- [ ] H-2.5 Activity log still reparses the whole bounded Markdown buffer: move to list/per-line rendering or throttled incremental rendering.
- [ ] H-2.6 External DB connector defaults can downgrade to plaintext: default PostgreSQL/MySQL/SQL Server to encrypted connections and require audited plaintext opt-in.
- [ ] H-2.7 Mutating tool verification whitelist is incomplete: trust `ToolResult.Mutated`, mark every mutating handler correctly, and remove fragile whitelist behavior.
- [x] H-2.8 `web_fetch` is hardened against DNS rebinding SSRF: URL validation and the guarded HTTP transport both reject private, loopback, link-local, multicast, and unspecified targets; redirects are revalidated and dial-time tests cover rebinding to loopback/multicast.
- [ ] H-2.9 LCS diff and rollback snapshots can consume too much memory/disk: add bounded diff strategy, content-addressable rollback storage, and rollback usage diagnostics.
- [ ] H-2.10 Two-layer bottom tab navigation still hides functions: continue bottom tool-window grouping and one-click discoverability polish.
- [ ] H-2.11 SQL guard can block valid keywords inside string literals: improve tokenizer/parser behavior and add fuzz/property tests.
- [ ] H-2.12 Git commands use a universal four-second timeout: make timeouts command-aware and improve long-repo error handling.

#### Medium

- [ ] M-3.1 `webfetch` recompiles regexes per call: move regexes to package-level compiled values.
- [ ] M-3.2 `web_fetch.extractTitle` returns first body line instead of HTML title: parse `<title>` before stripping HTML.
- [ ] M-3.3 Metadata store still has per-call connection/ensure contention: reduce locking with once/cached DB behavior and diagnostics.
- [ ] M-3.4 Job log tail is too short: raise tail size, persist full logs, and add "view full log" UX.
- [ ] M-3.5 Task runner can use shell wrappers for script names: switch discovered tasks to argv execution by task kind.
- [ ] M-3.6 Connector pool is not closed/cached cleanly between profile tests: add keyed TTL cache and invalidation on save.
- [ ] M-3.7 Split-editor secondary preview can go stale after save: refresh secondary pane on save or tab selection events.
- [ ] M-3.8 Search preview cap can silently miss matches past 64 KB: replace preview-based content search with bounded streaming search.
- [ ] M-3.9 `cleanRel` needs Windows drive/ADS/reserved-name hardening: reject colon paths and reserved device names where appropriate.
- [ ] M-3.10 Windows-1251 fallback remains too implicit: add encoding confidence/selection UX or stricter fallback diagnostics.
- [ ] M-3.11 Git dubious-ownership handling is generic: detect `safe.directory` failures and show targeted remediation.
- [ ] M-3.12 Agent runs lack a wall-clock limit: add deadline settings and graceful partial-result wrap-up.
- [ ] M-3.13 Job IDs can collide and have a 9,999 visual wall: switch to ULID or unbounded monotonic IDs.
- [ ] M-3.14 Connector profile plaintext sidecar errors are swallowed: preflight protected secret availability and show remediation.
- [ ] M-3.15 Streamed agent observer can race when events outrun UI queue: throttle/coalesce agent event updates.

#### Low / UX / Polish

- [ ] L-4.1 Rail button maps rely on UI-thread-only access: document or wrap with an explicit registry/mutex.
- [ ] L-4.2 Recent workspace list does not validate existence on load: tag or prune missing paths.
- [ ] L-4.3 Live editor syntax highlighting remains separate from the editable tab: keep as post-beta spike with accessibility/performance proof.
- [ ] L-4.4 Issue report filenames use local time while logs use UTC: use UTC timestamps for bundle filenames.
- [ ] L-4.5 Welcome tab persists after first workspace open in the latest static review: verify behavior and add regression coverage.
- [ ] L-4.6 Agent observation history is fixed at 10 entries: make history budget context-aware.
- [ ] L-4.7 Unsupported notebook cell kinds are silently dropped: record skipped cells and show a banner.
- [ ] L-4.8 Data connector active selection can stale after profile list rebuild: reconcile active ID after refresh.
- [ ] L-4.9 SSE reader assumes `data:` only: document provider compatibility and optionally handle `event:`/`id:` metadata.
- [ ] L-4.10 Agent prompt repeats ReAct format hints each iteration: send full format instruction once and trim later prompts.
- [ ] L-4.11 Chart rendering uses hardcoded colors: plumb theme tokens and validate safe color ranges.
- [ ] L-4.12 Fresh install has no default model selection flow: probe local provider and offer guided first model selection.

#### Documentation

- [ ] D-5.1 Production-readiness percentages are unverifiable: replace or back them with generated gate/item counts.
- [ ] D-5.2 Tracker open items need priority/gate labels: add "Now in flight" and P0/P1/P2/gate tags for open work.
- [ ] D-5.3 Agent tool registry docs can drift from code: generate docs/table from `internal/services/tools/catalog.go` and check drift in CI.
- [ ] D-5.4 Threat model needs explicit generated-artifact mutation row for `regenerate_artifact`: add controls for approval, lineage, rollback/mitigation, and redaction.
- [ ] D-5.5 LLM strategy provider docs drift from implemented provider profiles: update provider type language.
- [ ] D-5.6 `app-wails/` freeze state is not explicit: add freeze marker and CI guard once parity is declared.

#### Architecture / Long-Term Design

- [ ] A-6.1 Evaluate extracting a `nexus-core` Go module for services/tools/domain and optional CLI/server reuse.
- [ ] A-6.2 Replace repeated panel/list/detail patterns with a typed tool-window abstraction.
- [ ] A-6.3 Introduce a small shell event bus to decouple controllers and panel refreshes.
- [ ] A-6.4 Add structured/native tool-call agent protocol behind model-route feature flags while retaining ReAct fallback.
- [ ] A-6.5 Add rollback/snapshot behavior for artifact archive/restore/delete/regeneration mutations.
- [ ] A-6.6 Extend `protectedsecret` into the single vault for future GitHub, browser, MCP, plugin, and connector secrets.

#### Testing And CI

- [ ] T-7.1 Exercise platform-specific protected-secret paths in CI with Linux `secret-tool` and macOS test keychain coverage.
- [ ] T-7.2 Add fuzz/property tests for SQL guard tokenizer behavior.
- [ ] T-7.3 Add integration test proving workspace open runs no shell, Git, Docker, connector, OCR, browser, model, or deep-index work.
- [ ] T-7.4 Add long-agent-run stress test with repeated tool requests, wall-clock stop, and safe wrap-up.

#### Operational / Packaging

- [ ] O-8.1 Add SBOM, provenance attestation, signed release artifacts, and reproducible build guidance.
- [ ] O-8.2 Define and implement macOS notarization plus Linux package strategy and smoke.
- [ ] O-8.3 Keep no-telemetry posture but add opt-in quick diagnostics sharing via user-controlled URL/bundle flow.
- [ ] O-8.4 Add update visibility: minimal GitHub releases link first, later optional semver check service.

## 7. Immediate Execution Order

Future development sessions should pick one logical milestone from this order:

1. Continue UI shell controller extraction with Data and Settings controllers; the Search, Jobs, Rollbacks, Diagnostics, Git, Artifacts, Assistant, and Editor controller slices are complete.
2. Enforce safer connector TLS defaults with explicit audited plaintext opt-in.
3. Add metadata WAL/busy timeout and diagnostics visibility.
4. Throttle assistant streaming, agent events, and activity rendering.
5. Move editor save/diff/rollback off the UI thread and preserve editor state after save.
6. Wire signed packaging, installer/update/uninstall evidence into CI/release using the packaging readiness gate.
7. Run macOS clean-machine Keychain smoke to verify Security.framework prompts, signing/notarization behavior, and no-cgo refusal messaging.
8. Continue JetBrains-like UI polish once the above trust/performance risks are under control.

## 8. Keep-Going Prompt

Use this prompt repeatedly until production readiness is complete:

```text
Continue NexusDesk toward a production-ready Fyne-native app. Use nexus-app/ as active product and app-wails/ only as reference. Review tracker.md, docs/29_FULL_PROJECT_REVIEW_AND_ROADMAP.md, docs/17_END_TO_END_PRODUCTION_PLAN.md, docs/13_PRODUCTION_READINESS.md, docs/15_WAILS_FEATURE_INVENTORY.md, and claude-findings.md. Pick the highest-value unchecked milestone, prioritizing P0 safety/data-loss/performance findings, Wails parity, JetBrains-like UI polish, agent tool completeness, durable jobs, and signed packaging. Implement one logical slice end-to-end with focused tests and docs/tracker updates. Preserve boundaries: no Wails/webview in nexus-app, services framework-free, Fyne only in app/UI/theme, slow work via jobs, risky actions approval/audit/rollback/redaction. Validate with gofmt, go test ./..., go build ., git diff --check, remove generated binaries, then commit and push only when clean. Report changes, validation, commit hash, remaining blockers, and progress.
```
