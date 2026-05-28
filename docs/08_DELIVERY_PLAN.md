# Delivery Plan

## Native Migration Status

This document still preserves the long-horizon delivery plan and Wails-era implementation history. The active execution tracker is `tracker.md`: Wails/React is now preserved under `app-wails/`, the new native implementation lives under `nexus-app/`, and newly completed work should be described in native terms before older Wails-only capabilities are treated as complete.

Current native status: `nexus-app/` now has the active Fyne shell, native workspace tree, preview/edit/search/problems flows, safe file mutation and rollback services, assistant and agent services, approvals, metadata, Git status/diff/actions, task/jobs, Data & Analytics profiling/query/SQL/notebooks/charts, workspace SQLite inspection/query/export/cancellation, external database profile test/inspect/query flows, OS-protected provider/connector secret storage, artifacts, document extraction, operations scanning/runbooks, and history/audit panels. The largest remaining migration gaps are deeper assistant retrieval/ranking quality, richer generated document/full deck outputs, future slow-work job routing, dump/import design, signed packaging, onboarding, JetBrains-like UI polish, and continued shell complexity reduction.

Current estimate: Fyne-native migration is roughly 98% complete by useful Wails-era functionality, Wails useful-code parity is roughly 97%, Native Parity Beta readiness is roughly 96%, overall production readiness is roughly 93%, and distribution/packaging readiness is roughly 70-75%. Treat these as planning estimates, not release guarantees.

The production path is now defined in `docs/13_PRODUCTION_READINESS.md` and the combined product/architecture/UI north star is defined in `docs/17_END_TO_END_PRODUCTION_PLAN.md`. In short: finish Native Parity Beta first, then Safety/Reliability Beta, then Packaging/Platform Beta, then Private Beta. Do not treat the long historical sections below as production status; use `tracker.md`, the master plan, and the production-readiness gates for current execution.

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

- Fyne desktop app shell: implemented in `nexus-app/`
- native layout with project tree, editor area, assistant panel, and bottom workbench panels: implemented
- IDE/data/analytics studio positioning in app copy and docs: implemented
- visible product surface vocabulary for Workbench, Data & Analytics, Artifacts, and Settings: first implementation
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
- bounded CSV/TSV/JSON/NDJSON table preview: implemented
- bounded CSV/TSV/JSON/NDJSON column profiles: implemented
- larger capped table profile sample: implemented
- persistent CSV/TSV/JSON/NDJSON/XLSX/Parquet/log dataset profiles: first implementation, with XLSX workbook metadata, Parquet footer metadata, and bounded log summaries
- bounded CSV/TSV/JSON/NDJSON query/filter flow: first implementation
- dataset query result export artifact flow: first implementation
- dataset-to-SVG bar chart artifact flow: first implementation
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
- read selected table profile and sample into chat context: implemented
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

- The active desktop shell builds on Windows through Fyne/CGO using `nexus-app/scripts/dev-env.ps1`.
- The preserved Wails shell remains under `app-wails/` as a reference implementation.
- Nexus Augentic Studio is now documented and presented as a local-first AI Workbench with Data & Analytics, Artifacts, Settings, and always-visible assistant surfaces.
- The primary rail is intentionally limited to implemented product surfaces instead of roadmap-only studios.
- Opening a workspace no longer triggers automatic Git refresh; Git status is manual so folder open cannot launch Git work or render malformed unavailable Git responses.
- The workspace browser can open, refresh, preview, remember, search, and expand/collapse local folders, scanning up to 10 levels deep by default.
- The shell has a keyboard quick-open palette for workspace files, folders, and already-open tabs, with parent directories expanded on selection.
- The shell has a keyboard command palette for common workspace, editor, context, data, artifact, and chat actions.
- The window shell stays fixed-height; long file trees, previews, chat, settings, and timelines scroll inside their own panels.
- Text preview stays inside the approved workspace root and refuses binary/unsafe paths.
- Text preview decodes common UTF-8, UTF-16, and Windows-1251 Cyrillic files.
- CSV, TSV, JSON, and NDJSON files render as bounded table previews with lightweight column profiles from a larger capped sample while retaining raw text for selected chat context.
- Parquet files can be profiled through a bounded footer/magic inspection that records file, data, and footer metadata byte counts without scanning full columnar data.
- Log files can be profiled through a bounded sample that records sampled lines, levels, timestamps, stack trace lines, and repeated patterns.
- Common image previews render inline as capped data URLs from inside the approved workspace root.
- PDF previews render inline as capped data URLs from inside the approved workspace root and expose extracted text by page when available.
- DOCX files expose extracted body text when the document XML is readable.
- Recent workspaces and LLM settings persist locally.
- API keys are masked before leaving backend settings storage and saved in OS-protected credential blobs where available.
- The LLM settings form defaults to `qwen3:8b` and offers installed local model choices no larger than 26B.
- The local `rcooler-ollama` endpoint on `localhost:11434` is verified with CUDA 12 GPU offload through the sibling `../Llm/` Compose stack.
- The LLM settings panel reports Ollama runtime details, including selected model, endpoint, and VRAM residency when available.
- Streaming chat works with the configured model and optional selected file context.
- CSV/TSV/JSON/NDJSON context is sent as a structured profile plus bounded row sample instead of only raw preview text.
- CSV/TSV/JSON/NDJSON datasets can be queried with bounded text search or `column=value` filters.
- Dataset query results can be exported as timestamped CSV artifacts.
- Dataset row queries can be saved per dataset and reused from the Data & Analytics panel.
- Dataset row queries support text search, column filters, numeric comparisons, `contains`, `limit`, and simple `order by` clauses.
- Table datasets can preview bar or line chart points before generating deterministic SVG chart artifacts from category counts or numeric sums.
- Table datasets can generate deterministic Markdown summary artifacts with column profiles and suggested analysis questions.
- Multiple text, table-data, and extracted-PDF previews can be pinned into a bounded context pack for chat.
- Selected directories and the workspace root can be expanded into bounded streaming context packs.
- Pinned context packs show individual files and support removing one file at a time.
- The Preview button reloads the selected file, and the Explain button sends a grounded prompt for selected text/code previews.
- The Summarize button sends selected file, extracted document, or directory context through chat and saves the result as a Markdown artifact with provenance.
- The Review button sends the active text/code file through a code-review prompt, streams the answer in chat, and saves the review as a Markdown artifact with source provenance.
- The Git drawer and Workbench utility panel can send the selected working-tree diff through a grounded review prompt and save the review artifact with changed-file citations.
- Code AI actions can generate test suggestions from the active file or selected diff, explain dependency relationships, draft PR summaries/descriptions, and request unified-diff patch proposals.
- Accepted single-file assistant patch diffs can be converted into edit drafts and previewed through the same safe file-write diff/apply boundary before approval.
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
- Dataset query export artifacts now write sidecar provenance metadata with dataset source paths and query string.
- SVG chart artifacts now write sidecar provenance metadata with dataset source paths and chart configuration.
- The Artifact Studio route lists generated Markdown, CSV, and SVG artifacts, can reselect visible artifact files from that list, and shows artifact metadata when a generated artifact is active.
- Workspace scan reports can be saved as Markdown artifacts with scan counters and skipped/ignored path samples.
- Artifact metadata cards can open the source context, archive generated artifacts, or delete artifacts after approval.
- The always-visible assistant shows a first backend-driven tool plan with registered workspace, dataset, artifact, and operations tools plus risk/approval labels.
- Workspace search includes path/content matches, artifact metadata, and chat history snippets.
- Applied write/delete/move and artifact creation actions are recorded in `.nexusdesk/approvals/log.json` and shown in the bottom Approvals tab.
- Operations capabilities parse selected Docker Compose files into service, image, port, volume, and dependency summaries without mutating Docker state.
- The frontend has a smoke check for the built entrypoint, generated Wails bindings, and core shell functionality markers.
- Playwright is installed as a frontend dev dependency and visual smoke captures desktop/mobile baselines from the production build.
- SQLite metadata initialization now applies the schema to `.nexusdesk/metadata/nexusdesk.sqlite`, while JSON stores remain the compatibility layer until repositories migrate.
- Read-only SQL uses the bounded dataset-compatible path by default and has a CGO-gated DuckDB driver path behind the `duckdb` build tag.
- Tool-run rows expose detail drawers with captured inputs, outputs/errors, approval references, replay, and target diff affordances.
- Assistant answers and saved answer artifacts include source citations from selected files and context packs.
- Artifact lineage can be built across chats, tools, source files, and generated artifacts.
- Workspace freshness polling marks changed files and generated artifacts that may be stale after source changes.
- SQLite metadata now mirrors current JSON chat, approval, artifact, and tool-run records when the metadata store is prepared or inspected.
- Chat history, approval log, artifact list, and tool-run list now prefer SQLite metadata reads after the store exists while retaining JSON compatibility writes.
- The workbench can inspect SQLite metadata tables, search/filter columns, copy sample rows, and view dataset SQL views.
- The Data & Analytics SQLite connector can inspect workspace SQLite tables, views, columns, indexes, row counts, and capped samples without executing user SQL.
- The Data & Analytics SQLite connector query surface has visible row cap, timeout, and cancel controls; backend requests enforce those settings, support request-ID cancellation, and redact connector errors before SQL run metadata is recorded.
- The Data & Analytics SQLite schema browser can select tables/views, preview rows through the same capped query path, save SQLite connector queries separately from dataset SQL snippets, and filter SQLite query history.
- SQLite connector query results can now be exported as CSV artifacts or Markdown reports with SQL text, cap, timeout, preview rows, source database citation, SQL run records, and dataset dependency lineage.
- Settings can save local connector profiles with read-only defaults, result caps, timeouts, and protected credential references; returned profiles redact passwords/tokens.
- Saved PostgreSQL connector profiles can now be explicitly tested and inspected from Settings. The backend resolves protected credentials only for that action, opens a read-only session, applies statement timeouts, rejects non-`SELECT` SQL for the guarded profile query method, and returns schema metadata with tables, views, columns, indexes, foreign keys, and relationship hints.
- Saved MySQL and MariaDB connector profiles use the same explicit Test/Inspect/query boundary, guarded read-only SQL validation, engine timeout settings, schema metadata, foreign-key metadata, and first inferred relationship hints.
- Saved SQL Server connector profiles now use the same explicit Test/Inspect/query boundary, guarded read-only SQL validation, timeout/lock-timeout settings, schema metadata, foreign-key metadata, and first inferred relationship hints.
- Saved DuckDB connector profiles now validate a local database file in default builds and expose the same explicit Test/Inspect/query runner behind the `duckdb` build tag with a read-only `access_mode=read_only` DSN.
- External connector profile query requests now carry request IDs, share cap/timeout normalization, use context-aware query functions, can be cancelled through the app bridge, and pass errors through the shared connector redaction helper.
- Connector schema browsing now uses a shared frontend metadata browser for workspace SQLite and saved external database profile inspections, so external profile metadata can be navigated by table/view and inspected for columns, indexes, capped samples, relationships, a first ERD-like relationship map, and AI schema explanations.
- Chat messages and context-pack previews warn when cited files changed after the answer/context was created.
- Stale-context refresh can rebuild a context preview from changed files and records the refresh in the approval/metadata trail.
- Dataset dependency rebuild now removes the prior generated artifact before re-running so repeated refreshes avoid same-timestamp collisions.
- Data & Analytics clears visible query/chart/profile state when the selected dataset changes on disk.
- Data & Analytics source cards show profiled Parquet footer/data byte summaries after metadata inspection.
- Data & Analytics source cards show profiled log level and sample summaries after log profiling.
- Data & Analytics source cards expose explicit Open, Profile, and SQLite Inspect actions while keeping dump/import and compressed-export workflows disabled until job-backed sandboxes exist.
- Workspace freshness reports dataset-derived views that need refresh when table/workbook sources change.
- SQL query results can be exported as Markdown artifacts with SQL text, engine, row counts, preview rows, and dataset citations.
- Data & Analytics saves read-only SQL snippets separately from lightweight row filters.
- Artifact lineage has a selectable graph layout with relationship counts and source navigation.
- Playwright visual smoke now asserts navigator resizing, panel-level scrolling, tool-run details, metadata browser, lineage graph/filtering, SQL snippets, and freshness warnings using Wails-free mocks.
- richer document extraction/OCR and broader SQLite repository coverage are still planned.

## Completed Batch: Studio Hardening And Inspectors

This batch kept momentum on real functionality while cleaning up the growing shell surface:

1. Modal approval requests now cover higher-risk file write/delete/move applies.
2. Workspace search results are grouped into file, artifact, and chat sections.
3. Data & Analytics, Artifacts, Approval Log, Operations inspector, and approval modal UI are split into focused components.
4. Scan reports now capture included, ignored, depth-skipped, entry-capped, and unreadable paths without crowding the default navigator header.
5. Dataset preview/query tables support sortable columns and bounded pagination.
6. Chart artifact metadata now has clearer configuration and inline SVG preview.
7. Operations capabilities have a first read-only inspector for Docker/Compose and local service files.

## Completed Batch: Agent Tools And Workspace Intelligence

This batch made more of the studio inspectable and auditable without turning on autonomous tool execution yet:

1. Backend tool descriptors now live in `app/internal/agenttools/` with names, descriptions, risk levels, surfaces, and approval requirements.
2. The always-visible assistant shows a first proposed tool plan for the active file, dataset, artifact, or operations context.
3. Workspace scan reports can be saved as Markdown artifacts under `.nexusdesk/artifacts/`.
4. Dataset row queries now support numeric comparisons, `contains`, `limit`, and simple `order by` clauses.
5. Generated artifacts can open their source context, archive to `.nexusdesk/artifacts/archive/`, or be deleted through approval prompts.
6. Operations capabilities parse selected Compose YAML into services, images, ports, volumes, and dependencies.
7. Frontend smoke coverage now checks the new tool-planning, artifact-action, scan-report, Compose parsing, and optional visual smoke surfaces.

## Completed Batch: Agent Execution And Analytics Foundations

1. Backend agent tool plan rows can now be dry-run or executed through persisted tool-run records.
2. Medium/high-risk plan executions use modal approval before backend execution.
3. Tool run records persist input, output summary, risk, approval ID, duration, and errors under `.nexusdesk/tool-runs/`.
4. SQLite metadata schema preparation now writes a migration-compatible schema and manifest under `.nexusdesk/metadata/`.
5. Data & Analytics has a read-only DuckDB-compatible SQL surface over table datasets, using the bounded dataset query path until the real driver lands.
6. Artifact comparison shows added/removed line summaries and size delta between generated outputs.
7. Visual smoke now writes baseline screenshots and a manifest whenever Playwright is installed.

## Completed Batch: Context, Persistence, And Analytics Depth

1. SQLite metadata preparation now uses `modernc.org/sqlite` to create and migrate `.nexusdesk/metadata/nexusdesk.sqlite`.
2. DuckDB-backed SQL execution is implemented as a `database/sql` path behind the `duckdb` build tag for CGO-enabled systems, with bounded dataset SQL fallback in the default Windows loop.
3. Tool-run rows now expand into detail drawers with inputs, outputs/errors, approval IDs, replay, and target diff affordances.
4. Context-pack source citations now appear in persisted assistant answers and saved Markdown answer artifacts.
5. Artifact lineage can be built across chats, tool runs, source files, and generated outputs.
6. Workspace freshness polling detects changed files and flags generated artifacts that cite stale sources.
7. Playwright is now a dev dependency, visual smoke is enforced, and desktop/mobile visual baselines are captured.

## Completed Batch: Real Studio Workflows

1. SQLite metadata mirrors JSON chat, approval, artifact, and tool-run records into the active database.
2. Metadata Browser inspects SQLite metadata tables and dataset SQL views.
3. Artifact lineage filtering can focus source, chat, tool, or artifact relationships.
4. Chat messages and context-pack previews warn when cited files change. Assistant messages without explicit source context show weak-evidence warnings, the composer warns when no selectable/pinned context is available, and the chat header can retry or compare the latest answered prompt with the same attached source paths.
5. Data & Analytics invalidates visible query/chart/profile state when the selected dataset changes on disk.
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
5. Data & Analytics has a first read-only SQLite workspace database connector surface.
6. Artifact lineage can be exported as JSON and imported for debugging/preview workflows.
7. Playwright visual smoke mocks moved into a reusable fixture helper.
8. Dataset dependency rebuild actions are now available in Data & Analytics for filter exports, SQL reports, charts, and summaries.
9. Dataset dependency rebuild is now collision-safe for rapid re-runs by replacing stale regenerated artifacts before writing a new one.

## Completed Batch: Studio Query And Connector Maturity

1. Add explicit single-statement SQL validation (including quote/comment-aware semicolon checks) for SQLite connector and dataset analytics SQL inputs.
2. Add explicit refresh/rebuild buttons for dataset dependencies so saved SQL reports, charts, summaries, and exports can be regenerated from recorded inputs.
3. Add connector approval policy docs/tests for read-only proofs, blocked SQL statements, result caps, and redacted errors.

## Completed Batch: Table Dataset Coverage

1. TSV files now use the same bounded table preview, column profiling, profile persistence, and row-query path as CSV.
2. JSON arrays/objects are flattened into deterministic table columns for preview, profile, context, and bounded row queries.
3. JSONL/NDJSON files are decoded record by record for preview, profile, context, and bounded row queries.
4. Workspace scanning and frontend draft file typing classify CSV, TSV, JSON, JSONL, and NDJSON as data files.

## Completed Batch: XLSX Workbook Metadata

1. XLSX profiles now include sheet dimensions, row/column summaries, formula counts, table ranges, named ranges, and pivot table names.
2. Data & Analytics profile summaries expose workbook formula/table/named-range/pivot counts.
3. Agent dataset-profile observations include workbook metadata counts when profiling Excel files.
4. Legacy binary XLS parsing is split into its own pending data-import task because it needs a different parser or conversion path.

## Completed Batch: Data Source Cards And Classification

1. Data & Analytics now shows read-only source cards for table files, modern workbooks, SQLite files, dump-like files, compressed exports, log-like files, and Parquet files found in the bounded workspace tree.
2. Source cards show persisted profile status for already-profiled CSV, TSV, JSON, NDJSON, and XLSX datasets.
3. SQLite database files appear as source cards separate from the read-only connector query panel.
4. SQL dumps and compressed exports are classified without starting imports, containers, or extraction jobs.
5. Legacy `.xls` files now show explicit conversion guidance instead of attempting unsupported binary parsing.

## Completed Batch: Parquet Metadata Inspection

1. Parquet profiles now validate the fixed `PAR1` header/footer and footer metadata length before persisting metadata.
2. Profiling reads only fixed header/footer bytes, avoiding full-file scans, schema decoding, external commands, or background work on folder open.
3. Persisted Parquet profiles record file size, data bytes, footer metadata bytes, magic marker, and an explicit schema-decoding-pending message.
4. Data & Analytics source cards and profile summaries show profiled Parquet footer/data byte summaries.

## Completed Batch: Log Dataset Profiling

1. `.log`, `.out`, and `.trace` files are now classified as Data & Analytics profiling candidates.
2. Log profiles read only a bounded sample, reject binary-looking content, and persist sampled bytes/lines plus truncation state.
3. Profiles capture log levels, timestamped line counts, stack trace line counts, and repeated normalized patterns.
4. Data & Analytics source cards and profile summaries show profiled log sample and level summaries.

## Completed Batch: Connector Metadata Foundation

1. `app/internal/dbconnector/` now exposes a connector metadata model that can represent connector identity, engine, read-only state, tables, views, columns, indexes, row counts, and capped samples.
2. Workspace SQLite files can be inspected in read-only mode without executing user-provided SQL.
3. Data & Analytics exposes a manual Inspect schema action next to the existing SQLite read-only query surface.
4. SQLite metadata inspection records a dataset dependency row so connector schema inspections remain visible in local metadata history.

## Completed Batch: Connector Profile Foundation

1. `app/internal/storage/` now has a connector profile store for PostgreSQL, MySQL/MariaDB, SQL Server, DuckDB, and SQLite profile metadata.
2. Connector profile passwords/tokens are stored in a protected sidecar and represented in public JSON by credential references.
3. Wails exposes list/save/delete connector profile methods that return only redacted credentials.
4. Settings now includes a first connector profile card for saving read-only profile metadata, result caps, timeouts, and credential references.
5. PostgreSQL connector profiles can be tested and schema-inspected explicitly; the backend also exposes a guarded read-only query method for the next notebook surface.
6. MySQL/MariaDB connector profiles now share that explicit test, inspect, and guarded query path.
7. SQL Server connector profiles now share that explicit test, inspect, and guarded query path.
8. DuckDB connector profiles now share that explicit profile boundary through a default build guard and a CGO-tagged read-only execution path.
9. External profile queries now share request IDs, cancellation callbacks, cap/timeout normalization, and redacted error handling across supported database engines.
10. External profile inspections now render through the same connector metadata browser used by workspace SQLite schema inspection.
11. External profile inspections now include capped sample rows for PostgreSQL, MySQL/MariaDB, SQL Server, and DuckDB; DuckDB inspections also return live row counts when the optional `duckdb` build tag is active.
12. Connector metadata browsing now includes a clickable ERD-like relationship map with table/view nodes, primary-key hints, selected-object highlighting, and FK/inferred link rows.
13. Connector schema explanation prompts now use the shared inspected metadata shape, so saved external profile objects can be explained from Settings with the same grounded columns, indexes, samples, and relationship hints as workspace SQLite.
14. Data & Analytics now has a first multi-cell dataset SQL notebook shell: local SQL cells can be added, selected, edited, deleted, loaded from saved snippets, and executed through the existing bounded read-only SQL runner.
15. Dataset SQL notebook output now has result tabs for row previews, run summary, and SQL history/lineage.
16. The dataset SQL notebook now supports a first local chart cell type that embeds the existing bounded dataset chart preview/create controls.
17. Dataset SQL notebook output now includes a Plan tab. DuckDB builds can surface native explain output, while the bounded dataset fallback returns logical plan lines and explicit native-explain availability messaging.
18. Dataset SQL notebooks can now be saved and loaded per dataset. Saved notebooks persist SQL/chart cells under `.nexusdesk/datasets/notebooks.json` and record notebook dependencies for lineage/history surfaces.
19. Dataset SQL notebook History is now a browser over persisted SQL runs, with status/text filters, selected-run details, and Use SQL / Run Again actions.

## Completed Batch: Data Source Card Actions

1. Data source cards now expose explicit Open actions for all detected source-like files.
2. Table, workbook, Parquet, and log source cards route Profile to the existing bounded dataset profiling flow.
3. SQLite source cards route Inspect to the read-only schema inspector without starting connector work on folder open.
4. Dump/import, compressed-export, and legacy conversion workflows show disabled planned actions with clear lifecycle copy.

## Completed Batch: Connector Guardrails And Query Controls

1. SQLite connector queries now expose visible row cap and timeout controls in Data & Analytics.
2. SQLite query requests carry per-query cap, timeout, and request ID values instead of relying on hardcoded execution defaults.
3. In-flight SQLite connector queries can be canceled by request ID from the UI.
4. Connector errors are redacted before they are recorded in SQL run metadata, and redaction/cancellation paths have backend tests.
5. Connector SQL/dependency metadata is still recorded only after explicit user-triggered query completion or failure.

## Completed Batch: SQLite Schema Explorer Foundation

1. SQLite schema inspection now has selectable tables and views instead of a static object summary.
2. Selected schema nodes can run an explicit capped row preview through the guarded SQLite query path.
3. SQLite connector queries are saved under a separate `sqlite-sql` kind so they do not mix with dataset SQL snippets.
4. The connector panel shows saved SQLite queries and a filterable SQLite query history from persisted SQL run records.
5. Read-only status copy is visible near schema/query controls, and folder open still does not run connector work.

## Completed Batch: SQLite Connector Query Workflow

1. SQLite connector query results can be exported as CSV artifacts through the guarded query path.
2. SQLite connector query results can be exported as Markdown reports with SQL, result cap, timeout, row preview, and source database citation.
3. SQLite connector exports record SQL run rows and dataset dependency rows that link source database, query, engine, and generated artifact.
4. SQLite schema inspection now returns relationship hints from declared foreign keys and conservative `*_id` column matches.
5. The connector panel exposes a user-triggered Explain schema action for the selected table/view; the prompt is bounded to inspected columns, indexes, sample rows, and relationship hints.

## Prepared Batch: Architecture Hardening Before Deeper Studios

1. Extract chat/context/agent orchestration into a `useChatController` frontend hook and a backend `ChatService`.
2. Extract artifact route state/actions into `useArtifactController`; finish ArtifactService ownership for lineage and regeneration.
3. Extract dataset/SQL/connector route state/actions into `useDatasetController` before adding notebooks or connector profiles.
4. Keep folder open bounded and prove it cannot start Git, Docker, OCR, connector pulls, dump imports, long indexing, or shell execution.
5. Add focused tests for assistant unified-diff patch parsing before expanding multi-file patch workflows.
6. Add a durable job model for slow work and surface job progress in Activity before connecting OCR, dump restore, connector pulls, or long agent runs.
7. Promote SQLite repositories to primary persistence only after migration, fallback, and corruption/export recovery tests exist.

## Prepared Batch: Studio Query And Connector Maturity

1. Add a richer metadata history tab with filters by kind, time, source path, and jump-to-chat/artifact/tool actions.
2. Expand the SQLite connector with schema browsing, table previews, saved connector queries, and clearer read-only status. First manual schema inspection, explicit query guardrails, saved connector queries, query history, and schema-node previews are implemented.
3. Add artifact lineage JSON import comparison in the UI, including validation errors and graph diff previews.
4. Promote dataset dependency and SQL run records into first-class UI navigation from Data & Analytics, Artifacts, and Metadata Browser.
5. Start a DuckDB multi-file workspace dataset surface for joins across CSV/XLSX-derived tables.
6. Continue splitting large shell orchestration state before connector/history flows grow again.

## Strategic Studio Batches

These batches describe the next product direction. They are broader than the current foundation and should be broken down into smaller implementation batches before coding.

### Batch: Main Menu And Product Surfaces

1. Keep the primary rail focused on implemented surfaces: Workbench, Data & Analytics, Artifacts, and Settings.
2. Keep AI Assistant as the always-visible right-side orchestrator instead of a separate rail destination.
3. Keep Analytics, Documents, and Ops as capability domains until they have native layouts deep enough to justify top-level navigation.
4. Persist per-surface state: active resource, open tabs, filters, selected connector, selected artifact, and assistant context.
5. Keep the right sidebar focused on assistant output while surface-specific inspectors live in the selected surface or bottom drawer.
6. Update visual smoke to cover route switching and per-surface state restoration.

### Batch: IDE-Grade Workbench

1. Replace the current navigator feel with a JetBrains-style project tree: indentation, disclosure arrows, icons, context menus, reveal current file, collapse all, ignored-file controls, and preview-backed cut/copy/paste file operations.
2. Add git repository detection, branch display, dirty summary, and file-level status badges.
3. Add working tree/staged diff views with side-by-side and inline modes, hunk navigation, and stage/unstage/revert affordances.
4. Add problems/search panels: path search, text/symbol/regex search, replace preview, diagnostics, and task/test output. Path/text/symbol search, regex search, replace preview, lightweight TODO/FIXME/conflict/JSON diagnostics, read-only Compose task detection, user-triggered discovered task runs, captured output, and task-run artifacts are now implemented.
5. Continue editor improvements toward search/problems/refactoring depth. Split editor groups, pinned tabs, breadcrumbs, outline/symbol navigation, local go-to-definition, the jumpable Document Map, safe draft formatting, and encoding-aware save are now implemented in the native editor-quality foundation; deeper Monaco language-worker behavior is being replaced with deliberate Fyne-native or future LSP-backed equivalents.
6. Add AI code actions for review diff, explain change, generate tests, propose patch, create commit message, and summarize branch.

### Batch: Data & Analytics Expansion

1. Expand file dataset support to TSV, JSON, NDJSON, Parquet, logs, compressed exports, and SQL/database dump files. TSV/JSON/NDJSON table profiling, Parquet footer metadata inspection, and bounded log profiling are implemented; imports remain planned.
2. Build database connector framework for SQLite, PostgreSQL, MySQL/MariaDB, SQL Server, and DuckDB with read-only defaults. First SQLite connector metadata shape and read-only schema inspection are implemented.
3. Add schema browser with tables, views, columns, keys, indexes, row counts, samples, and generated relationship views.
4. Add query notebook with multiple cells, result tabs, saved queries, cancellation, query history, charts, and export actions.
5. Add temporary Docker-backed import sandboxes for SQL dumps with explicit lifecycle, storage limits, and read-only analysis mode.
6. Add LLM research workflow that creates an analysis plan, runs bounded read-only queries, cites rows/queries, and produces reproducible reports.

### Batch: Analytics Connectors

1. Add connector framework for GA4, Google Search Console, Google Ads, Meta Ads, Microsoft Ads, LinkedIn Ads, HubSpot, Salesforce, Eloqua, and Mautic.
2. Add secure credential storage, scope display, connector test, token refresh, and workspace binding.
3. Add import profiles for campaign exports, UTM exports, CRM leads/opportunities, marketing automation events, landing-page exports, and call-tracking exports.
4. Add dashboard widgets for acquisition, channel mix, funnels, cohorts, campaign ROI, content/SEO, landing-page performance, and anomaly detection.
5. Add LLM analytics workflows for explaining performance changes, finding anomalies, comparing campaigns, and generating client/internal reports with metric citations.

### Batch: Document Capabilities

1. Expand document extraction for PDF, DOCX, TXT, MD/MDX, HTML, RTF, XLS/XLSX, CSV, PPTX, and image OCR.
2. Add document set indexing with page/section/table/entity metadata and source citations.
3. Add comparison workflows for versions, contradictions, requirements, decisions, dates, risks, and action items.
4. Add generated outputs: Markdown reports, DOCX briefs, presentation decks, comparison matrices, checklists, and research packs.
5. Add stale-source regeneration for generated reports and presentations.

### Batch: Operations Capabilities

1. Expand Docker/Compose inspection to containers, images, volumes, networks, Compose projects, services, ports, health, env, mounts, resource usage, and logs.
2. Add local service views for ports, endpoint checks, config discovery, and redacted `.env` inspection where policy allows.
3. Add log workbench with tail, search, filters, stack trace grouping, error summaries, and incident report artifacts.
4. Add approval-governed start/stop/restart/build/pull/up/down/exec actions with command preview and audit records.
5. Add generated ops artifacts: Dockerfiles, Compose files, `.env.example`, health-check scripts, runbooks, deployment notes, and troubleshooting guides.

### Batch: AI Assistant Orchestration

1. Promote Assistant from chat panel to cross-studio orchestrator with explicit context, model, agent mode, tool plan, memory, and citation controls.
2. Add context sources for git diffs, database schemas, query results, analytics connector runs, document sets, operations logs, and artifacts.
3. Add agent modes: Ask, Plan, Review, Edit, Research, Analyze, Debug Ops, Generate Artifact, and Report Builder.
4. Local assistant memory and prompt profiles now exist alongside model comparison/retry, weak-evidence warnings, missing-context prompts, and source freshness indicators.
5. Add persistent workspace memory for accepted facts, decisions, preferred report style, reusable prompts, and source-linked decisions.

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
- user can approve or reject text/code file creates and updates after reviewing a diff, and binary writes after reviewing size/SHA-256 metadata
- user can delete a selected workspace file only after backend validation and confirmation
- user can rename or move a selected workspace file without overwriting existing files
- generated artifacts are linked to conversations and source context: first sidecar provenance flow implemented

## Phase 3: Excel, CSV, And Charts

Goal: support business and marketing analysis from structured data.

Deliverables:

- Excel workbook inspector: richer XLSX metadata implemented; legacy XLS pending
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
- SQL Server connector
- DuckDB connector behind the optional CGO build tag
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
