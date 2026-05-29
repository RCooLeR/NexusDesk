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

Needs finish:

- Final JetBrains-style shell layout.
- Thin icon-first rails.
- Move Problems, Search, Git, Tasks, Jobs, Audit, Diagnostics, and Activity fully into left-sidebar tool windows.
- Stronger resize behavior.
- Visual smoke coverage.
- Controller extraction from large UI objects.

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

Planned:

- FilePreview top-level truncation flag and save guard for partial previews.
- Off-UI-thread save/diff/rollback flow.
- Cursor/scroll preservation after save.
- Hunk-based diff for large files.
- Ambiguous encoding warning and explicit encoding selection before save.
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

Planned:

- Streaming byte-level content search that does not run full preview for every file.
- Singleflight/cancel-old-query behavior while typing.
- Better binary skip list.
- Larger but bounded per-file search cap.
- Search result virtualization for large result sets.
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

- Better retrieval and ranking beyond deterministic citations.
- Richer source diagnostics UI.
- Stream render coalescing and final markdown parse only once.
- Tool timeline redesign.
- Better approval recovery UI.
- Better context budget controls.
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

Planned:

- Zip/decompression caps across spreadsheet and document containers.
- More DataGrip-like schema browser.
- Query editor polish.
- Result grid virtualization.
- Query explain/read-only diagnostics.
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

Planned:

- Encrypted transport defaults for all network databases.
- Loud audited development-only plaintext opt-in.
- Connector pool reuse with short TTL.
- Better profile inspector showing resolved TLS/read-only mode.
- Cross-platform credential smoke.
- Connection diagnostics and remediation hints.

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

Planned:

- Artifact rollback parity for archive/restore/delete/regenerate.
- More regeneration coverage.
- Richer DOCX templates.
- Richer PPTX templates.
- Cross-suite smoke for Word/PowerPoint/LibreOffice.
- OCR/scanned PDF/image extraction through jobs.
- Better artifact gallery and lineage visualization.

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

Planned:

- Full job log files for long outputs.
- Higher visible tail cap.
- Better job progress modeling.
- Better cancel/retry semantics per job kind.
- Better approval details and recovery UI.
- Long agent run wall-clock limits.
- Stress tests for long sessions.

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

Planned:

- Searchable Settings dialog with categories.
- Per-task default model UI polish.
- Model capability badges.
- Vision-capability gating.
- Route test actions.
- Better disabled-state explanations.
- First-run provider setup wizard.

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

Planned:

- Diagnostics health cards redesign.
- Metadata WAL/busy-timeout visibility.
- Tool registry drift visibility.
- Protected secret smoke on each platform.
- Signed release evidence.
- SBOM/provenance evidence.
- Clean-machine smoke evidence.
- Update check with no auto-install.

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
- Core workbench functionality: strong but UI layout needs polish.
- Editor functionality: broad, needs performance and truncation/encoding hardening.
- Assistant functionality: broad, needs smoother streaming and better source/tool timeline UX.
- Agent toolbelt: broad, needs remaining high-risk safety polish and planned tools kept non-executable until ready.
- Data functionality: strong, needs zip caps, grid polish, and connector TLS/pool hardening.
- Artifact functionality: broad, needs rollback parity and template polish.
- Jobs/approvals: functional, needs long-run stress and better log/progress UX.
- Packaging: not production-complete until signing, notarization/package strategy, SBOM/provenance, and clean-machine smoke are done.
