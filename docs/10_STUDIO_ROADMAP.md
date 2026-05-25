# Studio Roadmap

This document describes the target product depth beyond the current MVP. It is intentionally more ambitious than the implemented feature set. `tracker.md` remains the source of truth for what exists today.

## North Star

NexusDesk should feel like a JetBrains-class desktop studio for mixed engineering, data, analytics, documents, and operations work. Chat is one interaction model, but the durable product surface is a set of studios with real navigation, inspectors, history, diffs, tool runs, artifacts, and source-grounded AI.

## Main Studio Menu

The primary rail/main menu should become a real navigation system, not just labels:

- `Code Studio`: repository tree, editor tabs, git status, diffs, search, problems, symbols, tests, task runner, and code-aware AI actions.
- `Data Studio`: databases, CSV, Excel, JSON/NDJSON, Parquet, logs, SQL dumps, schema browsers, query notebooks, import/export jobs, and temporary database sandboxes.
- `Analytics Studio`: GA4, Search Console, ads platforms, CRM/marketing automation data, exported marketing datasets, attribution/funnel views, dashboards, and recurring report artifacts.
- `Documents Studio`: PDF, DOCX, TXT, MD, XLS/XLSX, HTML, presentations, OCR, document sets, cross-document analysis, generated reports, and presentation creation.
- `AI Assistant`: model/provider control, context packs, agent runs, tool plan review, memory, prompt profiles, citations, and cross-studio orchestration.
- `Ops Studio`: Docker, Compose, logs, environment files, local services, ports, health checks, build/run/debug actions, generated configs, and approval-governed mutations.
- `Artifacts`: generated reports, charts, exports, diffs, presentations, configs, lineage, comparisons, archive/delete, and reproducibility metadata.
- `Settings`: providers, credentials, policies, workspace rules, connector credentials, model context windows, UI preferences, and diagnostics.

Each studio should own its commands, tabs, side panels, and empty states. The bottom drawer can remain useful for secondary panes, but the main menu should select first-class work modes.

## Code Studio

Current Code Studio is still primitive. The target is an IDE-grade surface:

- filesystem tree that behaves like an IDE project tree: folders/files, indentation, disclosure arrows, icons, selection, drag/drop intent, context menu, rename/move/new file/new folder, cut/copy/paste, reveal current file, collapse all, and ignored-file controls
- git status in the tree: modified/added/deleted/renamed/untracked/ignored, branch indicator, changed-file grouping, and repository dirty summary
- git diff viewer: working tree diff, staged diff, per-file diff, inline/side-by-side modes, hunk navigation, stage/unstage/revert hunk with approval where destructive
- editor quality: tab pinning, split editor groups, breadcrumbs, minimap/outline, symbol search, go to definition where language services exist, diagnostics panel, problems panel, formatting, and file encoding controls
- code search: path search, text search, regex, replace preview, symbol search, and saved searches
- tests/tasks: detect package scripts, Go tests, npm scripts, Docker Compose tasks, run selected tasks with logs, and save run artifacts
- AI code actions: explain file/project, review changes, generate tests, propose patch, apply patch through diff preview, summarize git diff, create commit message, and create PR description later

## Data Studio

Data Studio should be a real local data workbench:

- file datasets: CSV/TSV, Excel/XLSX/XLS, JSON, NDJSON, Parquet, SQLite, database dumps, logs, and compressed exports
- database connectors: SQLite first, then PostgreSQL, MySQL/MariaDB, SQL Server, DuckDB, and external JDBC/ODBC-style adapters where practical
- dump import workflow: detect `.sql`, `.dump`, `.bak`, `.gz`, `.zip`, and vendor-specific dumps; create a temporary isolated Docker/database workspace; import the dump; inspect schema; run read-only queries; destroy or persist the sandbox by explicit user choice
- schema browser: databases, schemas, tables, views, indexes, keys, row counts, samples, column stats, relationships, and generated ERD views
- query notebook: multiple SQL cells, saved queries, result tabs, chart cells, explain-plan where supported, query history, cancellation, result caps, and export to CSV/Markdown/Parquet
- data profiling: missing values, distinct counts, distributions, ranges, date detection, outlier hints, duplicate detection, primary-key candidates, and join suggestions
- data cleaning: preview transformations, derive columns, normalize dates, split/merge columns, dedupe, and write cleaned artifacts only after approval
- LLM research over data: generate hypotheses, choose relevant tables/files, create analysis plan, run bounded read-only queries, cite rows/queries, build charts, and create reproducible report artifacts

## Analytics Studio

Analytics Studio should focus on business and marketing analysis rather than generic tables:

- API connectors: GA4, Google Search Console, Google Ads, Meta Ads, Microsoft Ads, LinkedIn Ads, HubSpot, Salesforce, Eloqua, Mautic, and CSV/export equivalents
- credential flow: secure local credential storage, scopes display, token refresh, connector test, and explicit workspace binding
- import profiles: traffic exports, campaign spend, UTM exports, CRM leads/opportunities, marketing automation contacts/events, landing-page exports, and call-tracking exports
- analysis surfaces: acquisition, funnel, cohort, retention, attribution, content/SEO, campaign ROI, channel mix, landing-page performance, lead quality, and anomaly detection
- dashboard builder: saved widgets, filters, date ranges, segment comparison, chart artifacts, and narrative report blocks
- LLM analytics workflows: find anomalies, explain channel shifts, compare campaigns, write client/internal reports, generate follow-up questions, and cite metrics, connector runs, and source rows

## Documents Studio

Documents Studio should treat business documents as first-class source material:

- file support: DOCX, PDF, TXT, MD/MDX, HTML, RTF, XLS/XLSX, CSV, PPTX, images with OCR, and document bundles/folders
- extraction: text, headings, tables, images, footnotes, comments, tracked changes when possible, metadata, page references, and OCR fallback
- document set analysis: compare documents, extract requirements, action items, risks, decisions, timelines, entities, contradictions, and unresolved questions
- generation: Markdown reports, DOCX briefs, slide decks/PPTX, executive summaries, comparison matrices, checklists, and source-cited research packs
- review workflows: redlines, comments, source citations, confidence/coverage indicators, and regeneration when source files change
- LLM document workflows: summarize a folder, build a presentation from source docs, answer questions with page/section citations, produce a brief, and create reusable templates

## AI Assistant

The AI Assistant should become the orchestration layer across all studios, not a narrow sidebar:

- context control: current file, selected files, folders, git diff, database schema/query result, analytics connector run, document set, operations logs, and artifacts
- model control: provider/model selection, context-window budget, response reserve, tool-calling support, streaming status, GPU/local runner diagnostics, and model suitability hints
- agent modes: Ask, Plan, Review, Edit, Research, Analyze, Debug Ops, Generate Artifact, and Report Builder
- tool planning: show proposed tool calls, inputs, risk levels, expected outputs, approvals, and dry-run previews before mutation
- memory: workspace facts, decisions, accepted answers, reusable prompts, preferred report style, and ignored paths/connectors
- provenance: citations for files, lines, rows, queries, connector runs, logs, and document pages; stale-source warnings; regenerate actions
- multi-step work: plan, execute read-only exploration, pause for approval, create artifacts, compare outputs, and summarize what changed
- quality controls: retry with different model, compare model outputs, ask for missing context, detect weak evidence, and mark unsupported claims

## Ops Studio

Ops Studio should make local and containerized systems inspectable and debuggable:

- Docker/Compose: containers, images, volumes, networks, Compose projects, services, ports, health, env, mounts, resource usage, and logs
- local services: port scanner, process/service list where allowed, endpoint checks, config file discovery, `.env` inspection with secret redaction, and generated runbooks
- log workbench: tail logs, search/filter logs, group errors, detect stack traces, summarize incidents, and link logs to services/config files
- safe operations: start/stop/restart/build/pull/up/down/exec only after approval with clear risk labels, command preview, environment preview, and audit records
- generated ops artifacts: Dockerfile, Compose files, `.env.example`, health-check scripts, deployment notes, troubleshooting guides, and incident reports
- LLM ops workflows: explain Compose topology, diagnose why a service fails, compare environment files, propose a minimal fix, generate a safe command plan, and summarize logs with citations

Ops Studio should default to read-only inspection. Anything that mutates containers, files, databases, networks, volumes, or shell state must go through the shared approval and audit model.

## Cross-Studio Foundations

The studio roadmap depends on shared foundations:

- real main-menu routing and per-studio state persistence
- workspace metadata database for tabs, runs, artifacts, connector jobs, git snapshots, and document indexes
- connector credential vault and policy UI
- job runner for long imports, OCR, dump restores, connector pulls, and report generation
- cancelable tasks with progress, logs, retry, and artifact output
- source-grounded search across files, docs, datasets, connector runs, logs, chats, and artifacts
- artifact lineage for every generated report, chart, deck, dataset export, config, and code patch
- visual smoke and behavior tests for each studio surface
