# Features

Status: canonical feature inventory and planned capability map.

This document separates implemented capabilities from planned capabilities. If a capability is planned but not built, it must remain described as planned and must not appear as an executable agent tool or user promise until implemented and tested.

## 1. Product Areas

NexusDesk is one native workbench with these major product surfaces:

- Workbench and navigation.
- Editor and file operations.
- Search and problems.
- Assistant and agent.
- Agent tool registry.
- Data and analytics.
- Database connections.
- Artifacts and documents.
- Operations evidence.
- Jobs and approvals.
- Settings and model routing.
- Diagnostics and release readiness.

## 2. Implemented Capabilities

### 2.1 Workbench and navigation

Implemented:

- Native Fyne desktop app foundation.
- App menu, toolbar, rails, central canvas, assistant panel, left-sidebar tool windows, and status bar foundation.
- Native open workspace flow.
- Recent workspaces.
- Lazy workspace tree.
- Ignored-path handling.
- Refresh and reveal actions.
- Quick open.
- Command palette.
- Startup readiness summary.
- Diagnostics entry points.
- Workspace open policy that keeps open cheap and side-effect-free.
- JetBrains-style editor-centered shell layout.
- Thin icon-first rails with keyboard routing, active state, collapse behavior, and hover tooltips.
- Problems, Search, Git, Tasks, Jobs, Audit, Diagnostics, and Activity in left-sidebar tool windows.
- Stable split/resize behavior and per-tool width memory.
- Visual smoke coverage for supported shell states and sizes.
- Shell controller extraction for editor, assistant, data, Git, artifacts, jobs, diagnostics, approvals, and audit surfaces.

### 2.2 Editor and file operations

Implemented:

- Text/code preview.
- Draft editing.
- Dirty tabs.
- Pinned tabs.
- Close guard for dirty tabs.
- Save and revert flows.
- Rollback-backed writes.
- Safe create, delete, copy, move, rename, append, write, and patch operations.
- Markdown source/rendered view.
- Find/replace with match counts.
- Formatting for supported file types.
- Breadcrumbs.
- Split preview.
- Outline.
- Go-to-symbol.
- Local definition lookup.
- Workspace definition fallback.
- References.
- Syntax mirror.
- Document map.
- Token/symbol status.
- Active-line highlighting.
- Diagnostics for markers, merge conflicts, JSON, Go, YAML, TOML, XML.
- Preview support for text, code, images, CSV/TSV, DOCX text, PDF text, XLSX-derived content, and binary metadata.
- Encoding handling for common text formats.
- Save guard for partial/truncated previews.
- Off-UI-thread save flow with visible saving state.
- Cursor/scroll preservation and in-place editor refresh after save.
- Bounded hunk-based diffs for large files.
- Ambiguous encoding warning and explicit encoding selection before save.

Planned:

- Optional inline editable syntax styling after accessibility/performance proof.
- Feature-flagged LSP provider spike.
- Rename/code actions/test navigation after LSP design.

### 2.3 Search and problems

Implemented:

- Workspace path search.
- Workspace content search.
- Search snippets.
- Multiple matches per file.
- Problem scanning for TODO/FIXME-like markers.
- Conflict marker detection.
- Syntax diagnostics integration.
- Search/problem context for assistant.
- Streaming byte-level content and regex search that does not run full preview for every file.
- Cancellation/singleflight behavior while typing.
- Known-binary pre-skip list.
- Larger bounded per-file search cap.
- Incremental result streaming to the UI.

Planned:

- Saved search scopes.
- Semantic search after deterministic search is fast and trustworthy.

### 2.4 Assistant

Implemented:

- Ask mode.
- Agent mode.
- Streaming responses.
- Stop/cancel paths.
- OpenAI-compatible endpoint support.
- Ollama-oriented defaults and probing.
- Custom provider configuration.
- Provider/model probe and runtime diagnostics.
- Protected API key storage.
- Context packs from selected files/folders/artifacts/workspace.
- Source citations and line references.
- Evidence quality summaries.
- Weak-evidence warnings.
- Unverified citation diagnostics.
- Cited/uncited source coverage.
- Source freshness and stale-source warnings.
- Chat history persistence.
- Retry last answer.
- Compare assistant answers.
- Save assistant answer as artifact.
- Prompt profile and memory baseline.
- Task-aware model routing.
- Coalesced streaming render with final markdown parse.
- Source digest, source pane, lineage pane, inspector pane, tool timeline, and approval-card UI.
- Context budget visualization and route/model readiness guidance.

Task-aware model routing includes routes for:

- main coding;
- React, TypeScript, and JavaScript;
- Go backend;
- Python coding;
- PHP and Laravel;
- MySQL and PostgreSQL;
- Neo4j and Cypher;
- CSV and Excel data scripts;
- analytics explanations;
- research and summaries;
- image and screenshot understanding;
- balanced coding, reasoning, and vision;
- fastest practical 30B-class coding.

Planned:

- Image/screenshot understanding once model and UI policy are complete.
- Browser automation only after safety policy.
- MCP and plugin tool calls only after trust and approval model.

### 2.5 Agent

Implemented:

- Agent loop with planning and tool calls.
- Bounded observations.
- Approval-gated tool execution.
- Tool result audit.
- Final-answer fallback.
- Tool catalog with implemented/planned distinction.
- Risk classification.
- Full-project access policy with time bounds.
- Tool call visibility in UI.

Implemented tool categories:

- Tool registry inspection.
- Workspace context and file reads.
- Workspace search and problems.
- Definition, references, dependency graph, symbol index, project memory.
- Safe file write, append, copy, move, delete, patch.
- Rollback list and rollback application.
- Formatting and lint diagnostics.
- Git status, diff, history, blame.
- Git stage/unstage files and hunks.
- Commit staged changes.
- Create branch.
- Resolve conflicts.
- Revert approved changes.
- Task discovery.
- Safe task execution.
- Approved one-shot terminal command with argv, rooted cwd, timeout, output caps, shell/path blocking, and audit.
- Durable job list, logs, cancellation.
- Bounded web fetch.
- Dataset profile/query/SQL/chart.
- Workspace SQLite inspect/query.
- Document extraction.
- Read-only operations inspection.
- Runbook generation.
- Artifact lineage and supported regeneration.
- Redaction and approval helpers.
- External agent readiness planning.

Planned agent tools:

- Browser automation with URL policy, screenshot policy, download policy, and approvals.
- Interactive terminal sessions with durable supervision.
- Pull-request platform tools.
- MCP discovery and tool invocation.
- Scheduled automations.
- Image/screenshot description.
- Semantic workspace search.
- Connector sync job tools.
- Plugin-hosted tools.

### 2.6 Data and analytics

Implemented:

- CSV profiling.
- TSV profiling.
- JSON profiling.
- NDJSON profiling.
- XLSX profiling.
- Parquet metadata handling.
- Log profiling.
- Dataset query/filter/order/limit flows.
- SQL notebooks.
- SQL/chart notebook cells.
- Notebook save/load/run/export.
- Result tabs.
- Chart SVG artifacts.
- Dashboard artifacts.
- Workspace SQLite schema and query.
- SQLite views, indexes, samples, relationships.
- SQLite query history.
- SQLite cancellation.
- Result export paths.
- Schema tree browser with tables, views, columns, indexes, samples, and relationships.
- Query editor polish.
- Virtualized result grid with copy/export actions.
- Query history and profile inspector.

Planned:

- Dump import as an isolated durable job.
- Connector sync jobs.
- Better chart theming.
- Richer notebook cell types and skipped-cell reporting.

### 2.7 External databases

Implemented:

- External profiles for PostgreSQL.
- External profiles for MySQL/MariaDB.
- External profiles for SQL Server.
- External profiles for SQLite.
- Guarded DuckDB-oriented flows where supported.
- Protected connector credential storage.
- Read-only profile testing.
- Read-only schema inspection.
- Read-only bounded queries.
- Cancellation.
- Redacted errors.
- Query history.
- Encrypted transport defaults for PostgreSQL, MySQL/MariaDB, and SQL Server profiles.
- Explicit `development-plaintext` opt-in for local non-production database connections.
- Audited development-only plaintext opt-in.
- Connector pool reuse with cancellation, invalidation, and bounded idle lifetime.
- Profile inspector and Diagnostics transport status showing resolved TLS/read-only/plaintext state.
- Connection diagnostics and remediation hints.

Planned:

- Cross-platform credential smoke.

### 2.8 Artifacts and documents

Implemented:

- Artifact browser.
- Artifact preview.
- Artifact search/filter.
- Artifact archive/delete/restore.
- Artifact compare.
- Artifact lineage.
- Artifact freshness and stale-source warnings.
- Metadata sidecars.
- Regeneration for supported kinds.
- Artifact-to-assistant context pinning.
- Chart artifacts.
- Dashboard artifacts.
- Notebook artifacts.
- Document report artifacts.
- Document brief artifacts.
- DOCX document export.
- Workspace scan reports.
- Operations runbooks.
- Task reports.
- Chat answer artifacts.
- Comparison artifacts.
- Presentation outlines.
- Packaged presentation artifacts.
- PPTX deck outputs.
- Document text extraction.
- DOCX/PPTX package validation and theme metadata.
- Rollback/recovery snapshots for destructive artifact archive/restore/delete/regenerate flows.
- Expanded regeneration coverage.
- Polished DOCX templates.
- Polished PPTX templates.
- Cross-suite DOCX/PPTX smoke coverage in service and shell tests.
- Improved artifact freshness and lineage visualization.

Planned:

- OCR/scanned PDF/image extraction through jobs.

### 2.9 Operations

Implemented:

- Read-only Dockerfile inspection.
- Read-only Compose file inspection.
- Read-only env/config/script/log inspection.
- Secret redaction for operations evidence.
- Generated operations runbooks.
- Operations artifacts.

Planned:

- Better operations file tree.
- Environment comparison reports.
- Runbook template polish.
- Docker/system mutation workflows only after approval/job/audit/mitigation design.

### 2.10 Jobs and approvals

Implemented:

- Durable job ledger.
- Running/success/failed/canceled/timeout states.
- Job logs.
- Cancel running job.
- Retry/open-output patterns where supported.
- Approval log.
- Time-boxed full-project access policy.
- Approval UI baseline.
- Agent audit surface.
- Full log files for long outputs.
- Higher visible log tail cap.
- Open-full-log action and redacted issue-report log inclusion.
- Approval cards with details.
- Agent wall-clock limits, timeout UI/audit state, and repeated tool-loop stress coverage.

Planned:

- Better job progress modeling.
- Better cancel/retry semantics per job kind.

### 2.11 Settings and model routing

Implemented:

- Provider settings.
- Base URL and model settings.
- API key handling.
- Context and response reserve settings.
- Provider probe.
- Task-aware model routes.
- Curated model catalog.
- Connector profiles.
- Protected secrets baseline.
- Searchable settings shell with provider, route, credentials, connector, safety, UI, and diagnostics categories.
- Recommended model selectors and task-route test actions.
- Disabled-state explanations.
- First-run provider setup wizard and provider model auto-suggestion.

Planned:

- Model capability badges.
- Vision-capability gating.

### 2.12 Diagnostics and release readiness

Implemented:

- Startup recovery markers.
- Workspace readiness checks.
- Provider/model diagnostics.
- Metadata recovery/export.
- Redacted issue-report bundle.
- Packaging readiness evaluator.
- Release manifest support.
- CI scripts for platform checks.
- Diagnostics health cards and report sections.
- Metadata WAL, busy-timeout, foreign-key, and connection-pool visibility.
- Protected secret backend status.
- Release trust diagnostics.
- Windows zip and installer packaging.
- Release manifest, SBOM, provenance, and artifact evidence verification.
- Manual update check with no auto-download or auto-install.

Planned:

- Protected secret smoke on each platform.
- Signed release evidence.
- macOS/Linux package and clean-machine smoke evidence.

## 3. Intentional Non-Goals For v1

These are not v1 product goals:

- Hosted SaaS backend.
- Cloud sync.
- User accounts.
- Telemetry by default.
- Silent auto-update.
- Autonomous high-risk mutation without approval.
- Production database mutations.
- Docker/system mutations.
- Free-text shell execution from the model.
- Plugin marketplace.
- Multi-user/team mode.
- Background indexer that runs on workspace open.
- Hidden model calls.

## 4. Feature Readiness Summary

Approximate planning assessment:

- Native app foundation: very strong.
- Core workbench functionality: strong; remaining blockers are release validation, not the shell target.
- Editor functionality: broad and safety-hardened; final release still depends on clean-machine smoke and accessibility review.
- Assistant functionality: broad with source/trust UI; image/browser/MCP/plugin capabilities remain post-v1 unless separately designed.
- Agent toolbelt: broad for deterministic local tools; planned high-risk tools remain non-executable until design approval and tests.
- Data functionality: strong for v1 local/read-only workflows; dump import and connector sync remain post-v1.
- Artifact functionality: broad with DOCX/PPTX, lineage, freshness, regeneration, and rollback-aware destructive flows.
- Jobs/approvals: functional with durable logs, audit, approvals, and timeout coverage; richer progress modeling remains post-v1.
- Packaging: Windows zip/installer and release evidence exist; production release still needs signing, macOS/Linux artifacts, platform CI/smoke, and beta validation.
