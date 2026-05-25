# Delivery Plan

## Phase 0: Product Baseline

Goal: lock the product direction and preserve the strongest idea.

Deliverables:

- Nexus Augentic Studio product docs
- brand package
- UI wireframe
- core workflow definition
- LLM provider settings design
- tool risk model
- foundation scope guardrails

Exit criteria:

- product is clearly positioned as a local-first AI IDE, data studio, and analytics studio, not a prompt-only chatbot
- foundation modules are named
- risky actions have approval rules
- development stack is selected

## Phase 1: Project Foundation

Goal: create a usable local desktop studio with project browsing, editor tabs, data panels, artifacts, and grounded LLM assistance.

Deliverables:

- Wails desktop app shell: implemented
- frontend layout with project tree, editor area, and chat panel: implemented
- IDE/data/analytics studio positioning in app copy and docs: implemented
- visible studio surface vocabulary for code, data, documents, operations, and artifacts: first implementation
- local JSON app config for recent workspaces and LLM settings: implemented
- OS-protected API key credential storage: implemented
- local SQLite app database: initialized through a real driver with direct fresh-row metadata writes and compatibility mirroring
- controlled Markdown artifact writer: implemented
- workspace open/recent workspaces: implemented
- safe new text/code file draft creation through preview/apply: first implementation
- safe text/code file deletion with backend validation and confirmation: first implementation
- safe text/code rename/move with no-overwrite backend validation: first implementation
- file tree with ignore rules: implemented
- file tree scans up to 10 workspace levels by default: implemented
- workspace path/content search: implemented
- keyboard quick-open palette for workspace files, folders, and open tabs: implemented
- keyboard command palette for workspace, editor, context, data, artifact, and chat actions: first implementation
- editor find-in-file with highlighted matches: first implementation
- per-tab text edit drafts with dirty tab markers and close guard: first implementation
- Ctrl+S preview/apply shortcut for the current edit draft: first implementation
- Ctrl+F file find focus plus Ctrl+W and Ctrl+Tab editor tab shortcuts: first implementation
- workspace tree expand/collapse controls: implemented
- expandable tree state across refreshes: implemented
- fixed-height desktop shell with panel-level scrolling: implemented
- safe text/code file viewer: implemented
- bounded CSV table preview: implemented
- bounded CSV column profiles: implemented
- larger capped CSV profile sample: implemented
- persistent CSV/XLSX dataset profiles: first implementation
- bounded CSV query/filter flow: first implementation
- CSV query result export artifact flow: first implementation
- CSV-to-SVG bar chart artifact flow: first implementation
- image preview: implemented
- basic PDF preview: implemented
- PDF page text extraction: first implementation
- DOCX body text extraction: first implementation
- lightweight syntax highlighting: implemented
- UTF-8 BOM, UTF-16, and Windows-1251 text decoding: implemented
- Monaco editor integration: first edit-draft and read-only preview implementation
- LLM settings screen: implemented
- recommended local model dropdown capped at 26B: implemented
- LLM connection test for OpenAI-compatible `/models`: implemented
- LLM capability hints from provider model IDs: implemented
- Ollama runtime diagnostics for endpoint, selected model, and GPU/VRAM offload: implemented
- streaming chat with configured OpenAI-compatible LLM URL: implemented
- chat history per workspace: implemented with local JSON config
- read selected text file into chat context: implemented
- read selected CSV profile and sample into chat context: implemented
- pin multiple selected previews into a bounded chat context pack: implemented
- remove individual pinned context files: implemented
- preview backend-selected context pack files before sending chat: implemented
- read extracted PDF text into chat context when available: implemented
- reload selected preview from disk: implemented
- explain selected text/code preview through chat: implemented
- summarize selected context through chat and save a Markdown artifact: implemented
- create Markdown report artifact from selected preview: implemented
- save latest assistant answer as Markdown artifact: implemented
- artifact browser for generated Markdown reports: implemented
- frontend build smoke check: implemented

Exit criteria:

- user can open a workspace and chat with a local or remote model
- selected files can be included in context
- file access stays inside workspace root
- app runs on at least one development platform

Current status:

- The desktop shell builds on Windows through Wails.
- Nexus Augentic Studio is now documented and presented as a local-first AI IDE, data studio, and analytics studio.
- The active center pane exposes a first studio surface indicator for Code Studio, Data Studio, Document Studio, Operations Studio, Artifact Studio, or Project Studio.
- The workspace browser can open, refresh, preview, remember, search, and expand/collapse local folders, scanning up to 10 levels deep by default.
- The shell has a keyboard quick-open palette for workspace files, folders, and already-open tabs, with parent directories expanded on selection.
- The shell has a keyboard command palette for common workspace, editor, context, data, artifact, and chat actions.
- The window shell stays fixed-height; long file trees, previews, chat, settings, and timelines scroll inside their own panels.
- Text preview stays inside the approved workspace root and refuses binary/unsafe paths.
- Text preview decodes common UTF-8, UTF-16, and Windows-1251 Cyrillic files.
- CSV files render as bounded table previews with lightweight column profiles from a larger capped sample while retaining raw text for selected chat context.
- Common image previews render inline as capped data URLs from inside the approved workspace root.
- PDF previews render inline as capped data URLs from inside the approved workspace root and expose extracted text by page when available.
- DOCX files expose extracted body text when the document XML is readable.
- Recent workspaces and LLM settings persist locally.
- API keys are masked before leaving backend settings storage and saved in OS-protected credential blobs where available.
- The LLM settings form defaults to `qwen3:8b` and offers installed local model choices no larger than 26B.
- The local `rcooler-ollama` endpoint on `localhost:11434` is verified with CUDA 12 GPU offload through the sibling `../Llm/` Compose stack.
- The LLM settings panel reports Ollama runtime details, including selected model, endpoint, and VRAM residency when available.
- Streaming chat works with the configured model and optional selected file context.
- CSV context is sent as a structured profile plus bounded row sample instead of only raw preview text.
- CSV datasets can be queried with bounded text search or `column=value` filters.
- CSV query results can be exported as timestamped CSV artifacts.
- CSV queries can be saved per dataset and reused from the Data Studio panel.
- CSV queries support text search, column filters, numeric comparisons, `contains`, `limit`, and simple `order by` clauses.
- CSV datasets can preview bar or line chart points before generating deterministic SVG chart artifacts from category counts or numeric sums.
- CSV datasets can generate deterministic Markdown summary artifacts with column profiles and suggested analysis questions.
- Multiple text, CSV, and extracted-PDF previews can be pinned into a bounded context pack for chat.
- Selected directories and the workspace root can be expanded into bounded streaming context packs.
- Pinned context packs show individual files and support removing one file at a time.
- The Preview button reloads the selected file, and the Explain button sends a grounded prompt for selected text/code previews.
- The Summarize button sends selected file, extracted document, or directory context through chat and saves the result as a Markdown artifact with provenance.
- The workbench keeps recently opened previews in closeable editor tabs so several files can stay loaded while browsing.
- Markdown editor tabs can switch between raw source and rendered preview.
- Text/code previews support a local find box with match counts and highlighted matches.
- Text/code previews use a read-only Monaco viewer with find decorations.
- Text edit drafts show dirty state, can be reverted to the loaded content, and clear stale diff previews when the draft changes.
- Text edit drafts are retained per editor tab while navigating, dirty tabs are marked, closing a dirty tab asks for confirmation, and Ctrl+S previews or applies through the same safe write flow.
- New text/code files can be drafted from the command palette or Ctrl+N, then created through the same diff/apply write flow.
- Selected text/code files can be deleted only after backend validation and frontend confirmation.
- Selected text/code files can be renamed or moved inside the workspace without overwriting existing files.
- Editor keyboard support includes Ctrl+F for the in-file finder, Ctrl+W for closing the active tab, and Ctrl+Tab / Ctrl+Shift+Tab for tab cycling.
- Text/code edit drafts use a Monaco-backed editor surface with language detection while preserving the diff/apply boundary.
- The chat panel has an expanded conversation area, full visible history, context pack list, and multiline prompt composer.
- Chat responses render common Markdown structures, including tables and code blocks, instead of flattening formatted model output into one paragraph.
- Persistent chat history works through local JSON config.
- richer document extraction/OCR and broader SQLite repository coverage are still planned.
- Markdown report artifacts can be created under `.nexusdesk/artifacts/` without overwriting existing files.
- Latest assistant answers can be saved as Markdown artifacts under `.nexusdesk/artifacts/` with their chat context recorded as metadata.
- Markdown artifacts now write sidecar provenance metadata with source, prompt, model, source paths, and creation timestamp.
- CSV query export artifacts now write sidecar provenance metadata with dataset source paths and query string.
- SVG chart artifacts now write sidecar provenance metadata with dataset source paths and chart configuration.
- The Artifact Studio route lists generated Markdown, CSV, and SVG artifacts, can reselect visible artifact files from that list, and shows artifact metadata when a generated artifact is active.
- Workspace scan reports can be saved as Markdown artifacts with scan counters and skipped/ignored path samples.
- Artifact metadata cards can open the source context, archive generated artifacts, or delete artifacts after approval.
- The AI Assistant route shows a first backend-driven tool plan with registered workspace, dataset, artifact, and operations tools plus risk/approval labels.
- Workspace search includes path/content matches, artifact metadata, and chat history snippets.
- Applied write/delete/move and artifact creation actions are recorded in `.nexusdesk/approvals/log.json` and shown in the bottom Approvals tab.
- Operations Studio parses selected Docker Compose files into service, image, port, volume, and dependency summaries without mutating Docker state.
- The frontend has a smoke check for the built entrypoint, generated Wails bindings, and core shell functionality markers.
- Playwright is installed as a frontend dev dependency and visual smoke captures desktop/mobile baselines from the production build.
- SQLite metadata initialization now applies the schema to `.nexusdesk/metadata/nexusdesk.sqlite`, while JSON stores remain the compatibility layer until repositories migrate.
- Read-only SQL uses the bounded CSV-compatible path by default and has a CGO-gated DuckDB driver path behind the `duckdb` build tag.
- Tool-run rows expose detail drawers with captured inputs, outputs/errors, approval references, replay, and target diff affordances.
- Assistant answers and saved answer artifacts include source citations from selected files and context packs.
- Artifact lineage can be built across chats, tools, source files, and generated artifacts.
- Workspace freshness polling marks changed files and generated artifacts that may be stale after source changes.
- SQLite metadata now mirrors current JSON chat, approval, artifact, and tool-run records when the metadata store is prepared or inspected.
- Chat history, approval log, artifact list, and tool-run list now prefer SQLite metadata reads after the store exists while retaining JSON compatibility writes.
- The workbench can inspect SQLite metadata tables, search/filter columns, copy sample rows, and view dataset SQL views.
- Chat messages and context-pack previews warn when cited files changed after the answer/context was created.
- Stale-context refresh can rebuild a context preview from changed files and records the refresh in the approval/metadata trail.
- Dataset dependency rebuild now removes the prior generated artifact before re-running so repeated refreshes avoid same-timestamp collisions.
- Data Studio clears visible query/chart/profile state when the selected dataset changes on disk.
- Workspace freshness reports dataset-derived views that need refresh when CSV/XLSX sources change.
- SQL query results can be exported as Markdown artifacts with SQL text, engine, row counts, preview rows, and dataset citations.
- Data Studio saves read-only SQL snippets separately from lightweight row filters.
- Artifact lineage has a selectable graph layout with relationship counts and source navigation.
- Playwright visual smoke now asserts navigator resizing, panel-level scrolling, tool-run details, metadata browser, lineage graph/filtering, SQL snippets, and freshness warnings using Wails-free mocks.
- richer document extraction/OCR and broader SQLite repository coverage are still planned.

## Completed Batch: Studio Hardening And Inspectors

This batch kept momentum on real functionality while cleaning up the growing shell surface:

1. Modal approval requests now cover higher-risk file write/delete/move applies.
2. Workspace search results are grouped into file, artifact, and chat sections.
3. Data Studio, Artifact Studio, Approval Log, Operations inspector, and approval modal UI are split into focused components.
4. Scan status now reports included, ignored, depth-skipped, entry-capped, and unreadable paths.
5. CSV preview/query tables support sortable columns and bounded pagination.
6. Chart artifact metadata now has clearer configuration and inline SVG preview.
7. Operations Studio has a first read-only inspector for Docker/Compose and local service files.

## Completed Batch: Agent Tools And Workspace Intelligence

This batch made more of the studio inspectable and auditable without turning on autonomous tool execution yet:

1. Backend tool descriptors now live in `app/internal/agenttools/` with names, descriptions, risk levels, surfaces, and approval requirements.
2. The AI Assistant route shows a first proposed tool plan for the active file, dataset, artifact, or operations context.
3. Workspace scan reports can be saved as Markdown artifacts under `.nexusdesk/artifacts/`.
4. CSV queries now support numeric comparisons, `contains`, `limit`, and simple `order by` clauses.
5. Generated artifacts can open their source context, archive to `.nexusdesk/artifacts/archive/`, or be deleted through approval prompts.
6. Operations Studio parses selected Compose YAML into services, images, ports, volumes, and dependencies.
7. Frontend smoke coverage now checks the new tool-planning, artifact-action, scan-report, Compose parsing, and optional visual smoke surfaces.

## Completed Batch: Agent Execution And Analytics Foundations

1. Backend agent tool plan rows can now be dry-run or executed through persisted tool-run records.
2. Medium/high-risk plan executions use modal approval before backend execution.
3. Tool run records persist input, output summary, risk, approval ID, duration, and errors under `.nexusdesk/tool-runs/`.
4. SQLite metadata schema preparation now writes a migration-compatible schema and manifest under `.nexusdesk/metadata/`.
5. Data Studio has a read-only DuckDB-compatible SQL surface over CSV datasets, using the bounded CSV query path until the real driver lands.
6. Artifact comparison shows added/removed line summaries and size delta between generated outputs.
7. Visual smoke now writes baseline screenshots and a manifest whenever Playwright is installed.

## Completed Batch: Context, Persistence, And Analytics Depth

1. SQLite metadata preparation now uses `modernc.org/sqlite` to create and migrate `.nexusdesk/metadata/nexusdesk.sqlite`.
2. DuckDB-backed SQL execution is implemented as a `database/sql` path behind the `duckdb` build tag for CGO-enabled systems, with bounded CSV SQL fallback in the default Windows loop.
3. Tool-run rows now expand into detail drawers with inputs, outputs/errors, approval IDs, replay, and target diff affordances.
4. Context-pack source citations now appear in persisted assistant answers and saved Markdown answer artifacts.
5. Artifact lineage can be built across chats, tool runs, source files, and generated outputs.
6. Workspace freshness polling detects changed files and flags generated artifacts that cite stale sources.
7. Playwright is now a dev dependency, visual smoke is enforced, and desktop/mobile visual baselines are captured.

## Completed Batch: Real Studio Workflows

1. SQLite metadata mirrors JSON chat, approval, artifact, and tool-run records into the active database.
2. Metadata Browser inspects SQLite metadata tables and dataset SQL views.
3. Artifact lineage filtering can focus source, chat, tool, or artifact relationships.
4. Chat messages and context-pack previews warn when cited files change.
5. Data Studio invalidates visible query/chart/profile state when the selected dataset changes on disk.
6. SQL result artifacts save SQL text, engine, row counts, preview rows, and source dataset citations.
7. Playwright visual smoke asserts navigator resizing, tool-run details, metadata browser, lineage filtering, panel scrolling, and freshness warnings.

## Completed Batch: Studio Scale And Reliability

1. SQLite mirror reads now serve chat history, approvals, artifacts, and tool runs after the metadata store exists.
2. Metadata Browser now supports table search, column filtering, and copyable row samples.
3. Artifact lineage now has a graph layout with node selection, relationship counts, and source navigation.
4. Stale-context refresh controls rebuild context previews for changed files and record the refresh action.
5. Dataset dependency invalidation now flags dataset-derived views when source data files change.
6. SQL snippets are saved per dataset separately from lightweight row filters.
7. Playwright visual smoke now uses Wails-free mocked workspace, dataset, metadata, chat, and artifact fixtures.

## Completed Batch: Studio Depth And Connectors

1. Fresh chat, approval, artifact, and tool-run records now write directly into SQLite metadata when the store exists.
2. Metadata history search returns chat, artifact, and tool-run snippets backed by SQLite metadata queries.
3. Dataset lineage dependencies are recorded for saved SQL snippets, exported reports, chart artifacts, query exports, and summaries.
4. Saved SQL execution history records status, row counts, messages, and artifact links.
5. Data Studio has a first read-only SQLite workspace database connector surface.
6. Artifact lineage can be exported as JSON and imported for debugging/preview workflows.
7. Playwright visual smoke mocks moved into a reusable fixture helper.
8. Dataset dependency rebuild actions are now available in Data Studio for filter exports, SQL reports, charts, and summaries.
9. Dataset dependency rebuild is now collision-safe for rapid re-runs by replacing stale regenerated artifacts before writing a new one.

## Completed Batch: Studio Query And Connector Maturity

1. Add explicit single-statement SQL validation (including quote/comment-aware semicolon checks) for SQLite connector and CSV analytics SQL inputs.
2. Add explicit refresh/rebuild buttons for dataset dependencies so saved SQL reports, charts, summaries, and exports can be regenerated from recorded inputs.
3. Add connector approval policy docs/tests for read-only proofs, blocked SQL statements, result caps, and redacted errors.

## Prepared Batch: Studio Query And Connector Maturity

1. Add a richer metadata history tab with filters by kind, time, source path, and jump-to-chat/artifact/tool actions.
2. Expand the SQLite connector with schema browsing, table previews, saved connector queries, and clearer read-only status.
3. Add artifact lineage JSON import comparison in the UI, including validation errors and graph diff previews.
4. Promote dataset dependency and SQL run records into first-class UI navigation from Data Studio, Artifact Studio, and Metadata Browser.
5. Start a DuckDB multi-file workspace dataset surface for joins across CSV/XLSX-derived tables.
6. Split large shell orchestration state where connector/history flows start to crowd `NexusShell.tsx`.

## Strategic Studio Batches

These batches describe the next product direction. They are broader than the current foundation and should be broken down into smaller implementation batches before coding.

### Batch: Main Menu And Studio Routing

1. Turn the primary rail into a real main menu with Code, Data, Analytics, Documents, AI Assistant, Ops, Artifacts, and Settings routes.
2. Persist per-studio state: active resource, open tabs, filters, selected connector, selected artifact, and assistant context.
3. Give every studio a command surface, empty state, toolbar, and keyboard shortcuts that match its domain.
4. Keep the right sidebar focused on assistant output while studio-specific inspectors live in the selected studio or bottom drawer.
5. Update visual smoke to cover route switching and per-studio state restoration.

### Batch: IDE-Grade Code Studio

1. Replace the current navigator feel with a JetBrains-style project tree: indentation, disclosure arrows, icons, context menus, reveal current file, collapse all, and ignored-file controls.
2. Add git repository detection, branch display, dirty summary, and file-level status badges.
3. Add working tree/staged diff views with side-by-side and inline modes, hunk navigation, and stage/unstage/revert affordances.
4. Add problems/search panels: path search, text/regex search, replace preview, diagnostics, and task/test output.
5. Add editor improvements: split editor groups, pinned tabs, breadcrumbs, outline/symbol navigation, encoding controls, and format action hooks.
6. Add AI code actions for review diff, explain change, generate tests, propose patch, create commit message, and summarize branch.

### Batch: Data Studio Expansion

1. Expand file dataset support to TSV, JSON, NDJSON, Parquet, logs, compressed exports, and SQL/database dump files.
2. Build database connector framework for SQLite, PostgreSQL, MySQL/MariaDB, SQL Server, and DuckDB with read-only defaults.
3. Add schema browser with tables, views, columns, keys, indexes, row counts, samples, and generated relationship views.
4. Add query notebook with multiple cells, result tabs, saved queries, cancellation, query history, charts, and export actions.
5. Add temporary Docker-backed import sandboxes for SQL dumps with explicit lifecycle, storage limits, and read-only analysis mode.
6. Add LLM research workflow that creates an analysis plan, runs bounded read-only queries, cites rows/queries, and produces reproducible reports.

### Batch: Analytics Studio Connectors

1. Add connector framework for GA4, Google Search Console, Google Ads, Meta Ads, Microsoft Ads, LinkedIn Ads, HubSpot, Salesforce, Eloqua, and Mautic.
2. Add secure credential storage, scope display, connector test, token refresh, and workspace binding.
3. Add import profiles for campaign exports, UTM exports, CRM leads/opportunities, marketing automation events, landing-page exports, and call-tracking exports.
4. Add dashboard widgets for acquisition, channel mix, funnels, cohorts, campaign ROI, content/SEO, landing-page performance, and anomaly detection.
5. Add LLM analytics workflows for explaining performance changes, finding anomalies, comparing campaigns, and generating client/internal reports with metric citations.

### Batch: Documents Studio

1. Expand document extraction for PDF, DOCX, TXT, MD/MDX, HTML, RTF, XLS/XLSX, CSV, PPTX, and image OCR.
2. Add document set indexing with page/section/table/entity metadata and source citations.
3. Add comparison workflows for versions, contradictions, requirements, decisions, dates, risks, and action items.
4. Add generated outputs: Markdown reports, DOCX briefs, presentation decks, comparison matrices, checklists, and research packs.
5. Add stale-source regeneration for generated reports and presentations.

### Batch: Ops Studio

1. Expand Docker/Compose inspection to containers, images, volumes, networks, Compose projects, services, ports, health, env, mounts, resource usage, and logs.
2. Add local service views for ports, endpoint checks, config discovery, and redacted `.env` inspection where policy allows.
3. Add log workbench with tail, search, filters, stack trace grouping, error summaries, and incident report artifacts.
4. Add approval-governed start/stop/restart/build/pull/up/down/exec actions with command preview and audit records.
5. Add generated ops artifacts: Dockerfiles, Compose files, `.env.example`, health-check scripts, runbooks, deployment notes, and troubleshooting guides.

### Batch: AI Assistant Orchestration

1. Promote Assistant from chat panel to cross-studio orchestrator with explicit context, model, agent mode, tool plan, memory, and citation controls.
2. Add context sources for git diffs, database schemas, query results, analytics connector runs, document sets, operations logs, and artifacts.
3. Add agent modes: Ask, Plan, Review, Edit, Research, Analyze, Debug Ops, Generate Artifact, and Report Builder.
4. Add model comparison/retry, weak-evidence warnings, missing-context prompts, and source freshness indicators.
5. Add persistent workspace memory for accepted facts, decisions, preferred report style, and reusable prompts.

## Phase 2: Files, Documents, And Artifacts

Goal: make Nexus Augentic Studio useful for real documents and generated outputs.

Deliverables:

- Markdown/text extraction
- PDF text extraction where available
- document summary tool: first selected-context summarize-to-artifact flow implemented
- artifact manager: first Markdown artifact list implemented
- artifact provenance sidecars: first implementation
- create Markdown report tool: first controlled artifact flow implemented
- create chat answer artifact tool: first controlled artifact flow implemented
- create text/code file tool with approval: first edit flow implemented
- delete text/code file tool with confirmation: first implementation
- rename/move text/code file tool with no-overwrite validation: first implementation
- overwrite protection: first diff/apply flow implemented
- tool call timeline in chat
- approval dialog
- file diff preview for edits
- artifact browser: first Markdown report browser implemented

Exit criteria:

- AI can create a report artifact from selected source files
- user can approve or reject text file creates and updates after reviewing a diff
- user can delete a selected workspace file only after backend validation and confirmation
- user can rename or move a selected workspace file without overwriting existing files
- generated artifacts are linked to conversations and source context: first sidecar provenance flow implemented

## Phase 3: Excel, CSV, And Charts

Goal: support business and marketing analysis from structured data.

Deliverables:

- Excel workbook inspector
- CSV loader
- CSV table preview: implemented
- bounded CSV column profiles: implemented
- larger capped CSV profile sample: implemented
- dataset profiles beyond the preview window: first CSV/XLSX profile store implemented
- structured CSV chat context: implemented
- first CSV query result export artifacts: implemented
- first CSV-to-SVG bar chart artifacts: implemented
- DuckDB local analytics
- query dataset tool
- table preview
- chart spec tool
- chart rendering
- export chart as PNG/SVG
- export summary as Markdown
- basic marketing analysis templates

Exit criteria:

- user can analyze an Excel or CSV file through chat
- app can generate a chart artifact
- app can create a written report citing dataset sources
- large datasets are summarized instead of blindly sent to the model

## Phase 4: Databases And Marketing Connectors

Goal: connect Nexus Augentic Studio to real business data sources.

Deliverables:

- database connector framework
- SQLite connector
- PostgreSQL connector
- MySQL connector, optional
- read-only SQL policy
- schema explorer
- query-to-chart flow
- manual marketing CSV import templates
- GA4 connector prototype
- Search Console connector prototype
- connector credential storage

Exit criteria:

- user can connect to a database in read-only mode
- schema can be browsed and queried safely
- query results can become charts and reports
- marketing data can be analyzed from at least one connector or import format

## Phase 5: Docker And Operations

Goal: make Nexus Augentic Studio useful for Docker-based development and operations.

Deliverables:

- Docker connector
- container list
- image list
- container inspect
- log viewer
- Dockerfile explanation
- Compose file explanation
- generate Dockerfile/Compose artifact
- start/stop/build actions with approval
- Docker risk policies
- operations assistant mode

Exit criteria:

- app can inspect Docker environments
- AI can explain logs and Compose files
- risky Docker actions require approval
- generated Docker configs are saved as artifacts

## Phase 6: Advanced Agent And Plugin Layer

Goal: make the system extensible while preserving safety.

Potential deliverables:

- MCP client support
- external tool registry
- custom tool definitions
- embeddings and semantic workspace search
- project memory
- prompt/profile management
- multi-model comparison
- reusable report templates
- dashboard builder
- team/shared workspace mode
- Docker Desktop extension

Exit criteria:

- external tools can be added without breaking native safety rules
- advanced features remain optional
- core foundation remains fast and stable

## Foundation Scope Guardrails

Do not build everything at once.

Protect the core:

- open workspace
- inspect files
- jump to files, folders, and tabs quickly
- keep multiple files open in editor tabs
- find text inside the active file
- preserve unsaved drafts while switching tabs
- chat with configurable model
- read selected context safely
- analyze Excel/CSV
- create reports and charts
- expose clear studio surfaces for code, data, analytics, documents, operations, and artifacts
- log tool calls
- require approval for writes

Everything else should support those outcomes.
