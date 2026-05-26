# Nexus Augentic Studio Tracker

This tracker is the working execution plan for Nexus Augentic Studio. It separates implemented repository state from the long-term product plan so we do not confuse created code with intended architecture.

Related docs:

- `README.md`: project overview
- `docs/01_PRODUCT_VISION.md`: product direction
- `docs/02_ARCHITECTURE.md`: system architecture
- `docs/06_AI_AGENT_AND_LLM_STRATEGY.md`: agent and model contract
- `docs/08_DELIVERY_PLAN.md`: delivery phases
- `docs/09_DEVELOPER_EXPERIENCE.md`: local verification and ownership notes
- `docs/10_STUDIO_ROADMAP.md`: long-range studio roadmap

## Current Status

Nexus Augentic Studio is a runnable Wails desktop application with a Go backend, React/TypeScript frontend, local workspace scanning, file previews, editor tabs, safe file writes, configurable OpenAI-compatible LLM settings, streaming chat, first agent runtime, first read-only git status/diff visibility, first data workflows, first artifact/approval metadata, and visual smoke coverage.

It is not yet a JetBrains-class IDE/data/analytics studio. Major planned capabilities are still missing: IDE-grade Workbench editing/refactoring, staged diff workflows, deeper database/data support, analytics connectors, document intelligence, operations tooling, and AI orchestration.

## Repository State

- [x] Product and engineering docs live under `docs/`.
- [x] Brand assets and source brand package live under `docs/brand/`.
- [x] Long-range studio roadmap lives at `docs/10_STUDIO_ROADMAP.md`.
- [x] Wails app scaffold lives under `app/`.
- [x] Go backend lives under `app/`.
- [x] React/TypeScript frontend lives under `app/frontend/`.
- [x] Runtime brand assets live under `app/frontend/src/assets/brand/`.
- [x] Product logos/app icon use the brand kit; UI glyphs use Font Awesome instead of repeated logos.
- [x] Frontend generated Wails bindings are isolated behind `app/frontend/src/api/wailsClient.ts`.
- [x] Shell resize and studio navigation state are extracted into focused hooks.
- [x] App-level metadata mirror orchestration is split into `app/app_metadata.go`.
- [x] Helper services placeholder lives under `services/`.
- [x] Repository ignore rules exist in `.gitignore`.
- [x] Workspace-local `.nexusdesk/` runtime state is ignored when this repository is opened as a test workspace.

## Verification Loop

Run from `app/` unless noted:

```powershell
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
go test ./...
cd frontend
npm.cmd run build
npm.cmd run smoke
npm.cmd run smoke:visual
cd ..
wails build
```

Optional icon regeneration:

```powershell
python scripts/generate_windows_icon.py
```

Local Ollama GPU verification from sibling `../Llm/`:

```powershell
docker compose exec ollama nvidia-smi
docker compose logs ollama | Select-String "offloaded|model weights|cuda_v12"
Invoke-RestMethod http://localhost:11434/api/ps | ConvertTo-Json -Depth 10
```

## Phase 0: Product Baseline

Goal: define Nexus Augentic Studio as a local-first AI workbench with IDE, data, analytics, document, and operations capabilities.

Status: mostly complete for planning.

Steps:

- [x] Define product vision and target users.
- [x] Define local-first/provider-agnostic/tool-mediated principles.
- [x] Define simplified product navigation: Workbench, Data & Analytics, Artifacts, and Settings.
- [x] Define safety principle: LLM requests tools; backend validates and runs tools.
- [x] Define artifact-first output model.
- [x] Add long-range studio roadmap.
- [ ] Keep roadmap updated after each major implementation batch.

Exit criteria:

- [x] Product docs explain why this is more than a chatbot.
- [x] First useful foundation and long-range goals are separated.
- [x] Risky operations require preview, approval, and audit in the design.

## Phase 1: Project Shell Foundation

Goal: make the desktop app usable for opening a local workspace, browsing files, previewing content, editing safely, and chatting with selected context.

Status: implemented as a first useful foundation.

Steps:

- [x] Create Wails desktop app.
- [x] Replace starter screen with Nexus Augentic Studio shell.
- [x] Add branded rail, navigator, workbench, assistant, and bottom drawer layout.
- [x] Keep whole app fixed to the window and move scrolling into panels.
- [x] Add resizable workspace navigator.
- [x] Add resizable assistant sidebar up to 50 percent of the window.
- [x] Add resizable bottom drawer up to 70 percent of the window.
- [x] Add workspace folder picker.
- [x] Add recent workspace storage.
- [x] Add workspace refresh and selected-file preservation.
- [x] Add safe backend workspace scanning.
- [x] Scan up to 10 workspace levels by default.
- [x] Skip noisy folders, symlinks, deep paths, and oversized listings.
- [x] Add expandable workspace tree.
- [x] Preserve expanded directories across refreshes.
- [x] Add workspace path/content search.
- [x] Merge search results with artifact metadata and chat history snippets.
- [x] Add quick-open palette for files, folders, and open tabs.
- [x] Add command palette for workspace/editor/data/artifact/chat actions.
- [x] Add route-driven main menu selection for Workbench, Data & Analytics, Artifacts, and Settings.
- [x] Make main menu selections swap the primary workspace surface, not only the bottom drawer tab.
- [x] Reuse existing drawer surfaces as temporary primary route surfaces until full studios land.
- [x] Add first Workbench utility surface for editor session, project status, repository status placeholder, and work queues.
- [x] Remove duplicated route-owned tabs from the bottom drawer; keep the drawer for Git, Approvals, and Activity.
- [x] Replace current tree visual treatment with denser IDE-style project tree foundation.

Exit criteria:

- [x] User can open a local workspace and select files without blanking the app.
- [x] Large folder and malformed scan responses are guarded.
- [x] Long trees scroll inside the navigator.

## Phase 2: File Preview And Editor Foundation

Goal: provide safe preview/edit workflows for common source, text, document, image, and data files.

Status: implemented as a first foundation.

Steps:

- [x] Add rooted file preview boundary.
- [x] Refuse unsafe traversal and unsupported binary content.
- [x] Decode UTF-8 BOM, UTF-16, BOM-less UTF-16, and Windows-1251 text.
- [x] Add preview metadata: type, encoding, size, truncation.
- [x] Add image previews as bounded data URLs.
- [x] Add PDF preview as bounded data URL.
- [x] Add PDF embedded text extraction by page when available.
- [x] Add DOCX body text extraction when readable.
- [x] Add CSV bounded table preview.
- [x] Add CSV column profiles.
- [x] Add larger capped CSV profile sample.
- [x] Add read-only Monaco preview for text/code.
- [x] Add Monaco edit surface for text/code drafts.
- [x] Add closeable editor tabs.
- [x] Add per-tab edit drafts.
- [x] Add dirty tab markers.
- [x] Add dirty-close guard.
- [x] Add find-in-file with Monaco decorations.
- [x] Add Markdown source/rendered toggle.
- [x] Add safe new file draft creation.
- [x] Add safe text/code edit preview with diff.
- [x] Add safe apply flow for file writes.
- [x] Add safe file delete.
- [x] Add safe rename/move.
- [x] Add Ctrl+S, Ctrl+F, Ctrl+W, Ctrl+Tab, and Ctrl+Shift+Tab editor shortcuts.
- [x] Add split editor groups.
- [x] Add pinned tabs.
- [x] Add breadcrumbs.
- [x] Add outline/symbol navigation.
- [x] Add minimap toggle.
- [x] Add go-to-definition hook where language services exist.
- [x] Add formatting hooks.
- [x] Add file encoding selector and save-as-encoding support.

Exit criteria:

- [x] User can preview and safely edit text/code files.
- [x] File writes route through diff preview and rooted backend validation.

## Phase 3: LLM, Chat, Context, And Artifacts

Goal: make the assistant useful with selected workspace context while keeping provenance and safety.

Status: implemented as a first foundation.

Steps:

- [x] Add local LLM settings storage.
- [x] Store API keys in sidecar credential blob protected by OS where available.
- [x] Redact API keys before returning settings to UI.
- [x] Add OpenAI-compatible `/models` connection probe.
- [x] Infer capability hints from model IDs.
- [x] Add curated local model dropdown capped at 26B.
- [x] Verify `rcooler-ollama` endpoint at `localhost:11434`.
- [x] Document CUDA 12 Ollama runner pin.
- [x] Add Ollama runtime diagnostics for model, endpoint, and GPU/VRAM offload.
- [x] Add non-streaming chat.
- [x] Add streaming chat with Wails events.
- [x] Persist chat history per workspace.
- [x] Use distinct user/assistant timestamps so streamed deltas do not overwrite prompts.
- [x] Add readable chat panel with Markdown-style rendering.
- [x] Add OpenAI-style composer with model, Ask/Agent mode, and submit controls.
- [x] Send selected text/code context.
- [x] Send extracted PDF text context.
- [x] Send DOCX text context.
- [x] Send structured CSV profile/sample context.
- [x] Add pinned multi-file context packs.
- [x] Add individual context pack removal.
- [x] Add backend preview for context packs.
- [x] Expand selected directories and workspace root into bounded context packs.
- [x] Add model context-window and response-reserve settings.
- [x] Add shared curated model catalog that maxes context/reserve when a model is selected.
- [x] Prefer loaded Ollama runtime `context_length` over catalog defaults when probing succeeds.
- [x] Scale context-pack budget from configured model context window.
- [x] Send `num_ctx` for local/Ollama-compatible chat requests.
- [x] Send `num_predict` and OpenAI-compatible `max_tokens` from the configured response reserve.
- [x] Add realtime Activity Log events for chat request, stream, first token, completion, and failures.
- [x] Mark bounded agent runs that hit the tool iteration budget and force a finalization pass instead of showing the raw iteration-limit fallback as the answer.
- [x] Replace the empty agent placeholder with the last one or two live model/tool events while `RunAgent` is still executing, then replace that activity text with the final answer.
- [x] Add Explain selected context action.
- [x] Add Summarize selected context action.
- [x] Save summaries as Markdown artifacts.
- [x] Save latest assistant answer as Markdown artifact.
- [x] Include source citations in assistant answers and saved artifacts.
- [x] Warn when cited source paths changed.
- [x] Show the latest agent activity in chat while keeping the detailed run trace in Activity/tool-run records.
- [x] Add model comparison/retry.
- [x] Add weak-evidence and missing-context UI.
- [ ] Add assistant memory and prompt profiles.

Exit criteria:

- [x] User can chat with local/remote OpenAI-compatible provider using selected safe context.
- [x] Generated answers can become artifacts with provenance.

## Phase 4: IDE-Grade Workbench

Goal: make Workbench feel and work like a serious IDE rather than a file preview shell.

Status: planned.

Step 4.1: Main code route

- [x] Add first-class Workbench route in primary menu.
- [x] Collapse visible rail to implemented top-level surfaces instead of roadmap-only studios.
- [x] Create first Workbench utility layout.
- [x] Persist route, drawer, and panel layout state independently from transient bottom drawer state.
- [x] Add Workbench toolbar and command set.
- [x] Keep editor surfaces available through the main route and git diff review available through the bottom drawer.

Step 4.2: Project tree

- [x] Replace current navigator feel with IDE project tree presentation.
- [x] Add indentation guides.
- [x] Add disclosure arrows.
- [x] Add file/folder icons by type.
- [x] Add selected/current file reveal.
- [x] Add collapse all and expand selected path.
- [x] Add context menu shell for new file, new folder, rename, move, delete, copy path, reveal in explorer.
- [ ] Add cut/copy/paste file operations with preview for mutations.
- [ ] Add ignored-file controls.
- [ ] Add drag/drop intent design before implementing mutation.

Step 4.3: Git integration

- [x] Detect git repository root.
- [x] Show current branch.
- [x] Show dirty summary.
- [x] Avoid running Git automatically on workspace open.
- [x] Normalize unavailable Git responses so null changed-file arrays cannot blank the shell.
- [x] Hide external Git command windows on Windows desktop builds.
- [x] Show file status badges in tree.
- [x] Add changed-files panel.
- [x] Render Git changed files as a directory tree instead of a flat repeated path list.
- [x] Add working tree diff.
- [x] Add staged diff.
- [x] Move Git/working-tree diff review into a dedicated bottom drawer tab.
- [x] Add side-by-side diff viewer.
- [x] Add inline diff viewer.
- [x] Add diff-only side-by-side viewer for changed lines.
- [x] Add hunk navigation.
- [x] Replace text hunk controls with compact IDE-style icon controls.
- [ ] Add stage/unstage file.
- [ ] Add stage/unstage hunk.
- [ ] Add revert hunk with destructive approval.
- [x] Add AI diff summary.
- [x] Add AI commit message draft.

Step 4.4: Search, problems, and tasks

- [ ] Add path search panel.
- [ ] Add text search panel.
- [ ] Add regex search.
- [ ] Add replace preview.
- [ ] Add symbol search where language data exists.
- [ ] Add diagnostics/problems panel.
- [x] Detect package scripts.
- [x] Detect Go tests.
- [x] Detect npm scripts.
- [ ] Detect Docker Compose tasks.
- [ ] Run tasks with captured output.
- [ ] Save task/test runs as artifacts or metadata.

Step 4.5: Code AI actions

- [ ] Review current file.
- [ ] Review git diff.
- [ ] Generate tests for selected file/diff.
- [ ] Propose patch with diff preview.
- [ ] Apply accepted patch through safe write boundary.
- [ ] Explain dependency graph.
- [ ] Create PR summary draft.
- [ ] Create PR description draft.

Exit criteria:

- [ ] Workbench can be used for day-to-day project navigation and diff review.
- [ ] AI code changes remain previewed, reviewable, and auditable.

## Phase 5: Data & Analytics Expansion

Goal: make Data & Analytics a real local data workbench for files, databases, dumps, notebooks, marketing/CRM exports, profiling, charts, and LLM-assisted research.

Status: first CSV/SQLite foundation implemented; deeper work planned.

Implemented:

- [x] CSV table preview.
- [x] CSV column profiles.
- [x] Dataset profile persistence under `.nexusdesk/datasets/`.
- [x] Bounded CSV query/filter flow.
- [x] Numeric comparisons, contains, limit, and order by.
- [x] Saved lightweight row filters.
- [x] Saved read-only SQL snippets.
- [x] SQL run history.
- [x] CSV query export artifacts.
- [x] SVG chart artifacts.
- [x] Dataset summary artifacts.
- [x] DuckDB-compatible SQL surface with bounded fallback.
- [x] CGO-gated DuckDB driver path behind `duckdb` build tag.
- [x] Read-only SQLite workspace connector.
- [x] SQLite mutation keyword blocking and single-statement validation.
- [x] Dataset dependencies and rebuild actions.

Step 5.1: File dataset coverage

- [ ] Add TSV loader.
- [ ] Add richer XLSX/XLS workbook inspector for sheets, formulas, named ranges, pivots, and table ranges.
- [ ] Add JSON loader.
- [ ] Add NDJSON loader.
- [ ] Add Parquet inspection.
- [ ] Add SQLite file dataset cards separate from connector sessions.
- [ ] Add log dataset profiling.
- [ ] Add compressed export detection.
- [ ] Add SQL dump file classification.
- [ ] Add data source cards for each detected dataset.

Step 5.2: Database connector framework

- [ ] Define connector interface and metadata model.
- [ ] Add connection profiles with secure credential references.
- [ ] Expand SQLite schema browser.
- [ ] Add PostgreSQL read-only connector.
- [ ] Add MySQL/MariaDB read-only connector.
- [ ] Add SQL Server read-only connector.
- [ ] Add DuckDB connector.
- [ ] Add query cancellation.
- [ ] Add result caps and timeout controls per connector.
- [ ] Add connector error redaction.

Step 5.3: Schema and relationship explorer

- [ ] Show databases and schemas.
- [ ] Show tables and views.
- [ ] Show columns, types, nullable, defaults.
- [ ] Show indexes and keys.
- [ ] Show row counts and table samples.
- [ ] Infer relationships where metadata is absent.
- [ ] Generate ERD-like relationship view.
- [ ] Let AI explain schema with citations.

Step 5.4: Query notebook

- [ ] Add multi-cell SQL notebook UI.
- [ ] Add result tabs.
- [ ] Add chart cells.
- [ ] Add explain-plan display where connector supports it.
- [ ] Add saved notebooks.
- [ ] Add query history browser.
- [ ] Add visible result caps and timeout controls.
- [ ] Add export to CSV.
- [ ] Add export to Markdown report.
- [ ] Add export to Parquet when supported.
- [ ] Add query-to-artifact lineage.

Step 5.4b: Data profiling and cleaning

- [ ] Add missing-value profiles.
- [ ] Add distinct-count profiles.
- [ ] Add distribution charts.
- [ ] Add range and date detection.
- [ ] Add outlier hints.
- [ ] Add duplicate detection.
- [ ] Add primary-key candidate detection.
- [ ] Add join suggestions.
- [ ] Add preview transformations.
- [ ] Add derived columns.
- [ ] Add date normalization.
- [ ] Add split/merge column actions.
- [ ] Add dedupe preview.
- [ ] Write cleaned artifacts only after approval.

Step 5.5: Temporary dump import sandboxes

- [ ] Detect `.sql`, `.dump`, `.bak`, `.gz`, `.zip`, and vendor-specific dumps.
- [ ] Ask user to create temporary import sandbox.
- [ ] Choose matching database image when possible.
- [ ] Start isolated Docker Compose sandbox.
- [ ] Import dump with logs and progress.
- [ ] Enforce storage and runtime limits.
- [ ] Mark sandbox read-only for analysis.
- [ ] Destroy sandbox on request.
- [ ] Persist sandbox metadata only by explicit choice.
- [ ] Record all operations in approval/audit log.

Step 5.6: LLM data research

- [ ] Let assistant create analysis plan from schema/files.
- [ ] Let assistant propose read-only queries.
- [ ] Run bounded queries after user approval/policy.
- [ ] Cite rows, tables, queries, and connector runs.
- [ ] Generate charts from query results.
- [ ] Generate reproducible Markdown reports.
- [ ] Mark stale reports when source data changes.

Exit criteria:

- [ ] User can inspect and query real files, databases, and imported dumps safely.
- [ ] AI research over data is reproducible and source-cited.

## Phase 6: Analytics Capabilities

Goal: make Nexus Augentic Studio useful for marketing, traffic, CRM, and funnel analysis from APIs and exports.

Status: planned.

Step 6.1: Studio route and data model

- [x] Fold analytics planning into Data & Analytics instead of exposing a separate top-level route.
- [ ] Add native analytics connector/layout sections inside Data & Analytics.
- [ ] Define analytics source, connector run, metric, dimension, segment, and dashboard models.
- [ ] Bind analytics runs to workspace metadata.
- [ ] Add date range and segment selectors.

Step 6.2: Connectors

- [ ] Add GA4 connector.
- [ ] Add Google Search Console connector.
- [ ] Add Google Ads connector.
- [ ] Add Meta Ads connector.
- [ ] Add Microsoft Ads connector.
- [ ] Add LinkedIn Ads connector.
- [ ] Add HubSpot connector.
- [ ] Add Salesforce connector.
- [ ] Add Eloqua connector.
- [ ] Add Mautic connector.
- [ ] Add CSV/export equivalent import profiles for each connector family.

Step 6.3: Credential and policy layer

- [ ] Store connector credentials securely.
- [ ] Show scopes before connection.
- [ ] Support token refresh.
- [ ] Support connector test.
- [ ] Bind credentials to workspace.
- [ ] Add read-only connector policy by default.
- [ ] Log connector pulls.

Step 6.4: Analytics surfaces

- [ ] Acquisition dashboard.
- [ ] Channel mix dashboard.
- [ ] Campaign ROI dashboard.
- [ ] Funnel dashboard.
- [ ] Cohort/retention dashboard.
- [ ] Attribution dashboard.
- [ ] SEO/content dashboard.
- [ ] Landing-page performance dashboard.
- [ ] Lead quality dashboard.
- [ ] Anomaly detection view.
- [ ] Saved dashboard widgets.
- [ ] Dashboard filters and date ranges.
- [ ] Segment comparison.
- [ ] Narrative report blocks attached to charts.

Step 6.5: AI analytics workflows

- [ ] Explain performance changes.
- [ ] Find anomalies.
- [ ] Compare campaigns.
- [ ] Summarize channel mix.
- [ ] Generate client-ready report.
- [ ] Generate internal action plan.
- [ ] Cite metrics, connector runs, and source rows.

Exit criteria:

- [ ] User can connect or import at least one analytics data source.
- [ ] Analytics capabilities can produce cited charts and narrative reports.

## Phase 7: Document Capabilities

Goal: make documents first-class source material and support generated reports, briefs, and presentations.

Status: first PDF/DOCX text extraction exists; studio planned.

Implemented:

- [x] PDF preview and embedded text extraction.
- [x] DOCX body text extraction.
- [x] Markdown source/rendered preview.
- [x] Text preview and encoding support.
- [x] Summary-to-Markdown artifact flow.

Step 7.1: Studio route and document library

- [x] Keep document workflows contextual instead of exposing a separate top-level route.
- [ ] Add native document library/extraction sections when the workflow depth justifies it.
- [ ] Add document library view.
- [ ] Add document set/folder grouping.
- [ ] Add document metadata panel.
- [ ] Track extraction status and freshness.

Step 7.2: Extraction coverage

- [ ] Improve PDF text extraction.
- [ ] Add OCR fallback for image PDFs.
- [ ] Extract DOCX headings and tables.
- [ ] Extract document images where practical.
- [ ] Extract footnotes where practical.
- [ ] Extract DOCX comments where possible.
- [ ] Extract tracked changes where possible.
- [ ] Extract page references and section anchors.
- [ ] Extract document metadata.
- [ ] Add HTML/RTF extraction.
- [ ] Add PPTX text extraction.
- [ ] Add image OCR.
- [ ] Extract spreadsheet text/tables into document context.

Step 7.3: Document analysis workflows

- [ ] Summarize document.
- [ ] Summarize document set.
- [ ] Compare two documents.
- [ ] Extract action items.
- [ ] Extract decisions.
- [ ] Extract risks.
- [ ] Extract dates/entities.
- [ ] Detect contradictions across documents.
- [ ] Generate source-cited research pack.

Step 7.4: Generated document outputs

- [ ] Generate Markdown report.
- [ ] Generate DOCX brief.
- [ ] Generate PPTX presentation.
- [ ] Generate comparison matrix.
- [ ] Generate checklist.
- [ ] Store document output provenance.
- [ ] Regenerate stale document outputs.

Step 7.5: Review workflows

- [ ] Add redline/change review view.
- [ ] Add document comments view.
- [ ] Add confidence and coverage indicators for generated analysis.
- [ ] Add reusable document/report templates.
- [ ] Add page/section citation inspector.

Exit criteria:

- [ ] User can analyze a folder of documents and generate cited reports/decks.

## Phase 8: Operations Capabilities

Goal: make local and Docker operations inspectable, explainable, and safe.

Status: first Compose parser exists; studio planned.

Implemented:

- [x] Operations inspector parses Docker Compose files.
- [x] Compose services, images, ports, volumes, and dependencies are displayed.
- [x] Backend tool registry includes first operations inspect descriptors.

Step 8.1: Studio route and read-only inventory

- [x] Keep operations workflows contextual instead of exposing a separate top-level route.
- [ ] Add native operations layout sections when the workflow depth justifies it.
- [ ] List Docker containers.
- [ ] List Docker images.
- [ ] List Docker volumes.
- [ ] List Docker networks.
- [ ] List Compose projects.
- [ ] Inspect service health.
- [ ] Show ports and mounts.
- [ ] Show environment with secret redaction.
- [ ] Show Docker/Compose resource usage.
- [ ] List local processes/services where policy allows.

Step 8.2: Logs and diagnostics

- [ ] Add log viewer.
- [ ] Add tail mode.
- [ ] Add log search/filter.
- [ ] Group stack traces.
- [ ] Summarize errors.
- [ ] Link logs to services and configs.
- [ ] Save incident report artifact.

Step 8.3: Local services

- [ ] Add port scanner where policy allows.
- [ ] Add endpoint health checks.
- [ ] Add local config discovery.
- [ ] Add `.env` inspection with redaction.
- [ ] Add runbook generation.

Step 8.4: Safe operations

- [ ] Preview Docker start/stop/restart/build/pull/up/down/exec commands.
- [ ] Show environment preview before mutating operations.
- [ ] Require approval for every mutating Docker action.
- [ ] Require approval for shell execution.
- [ ] Capture stdout/stderr logs.
- [ ] Record operations in approval/audit metadata.
- [ ] Generate Dockerfile artifacts.
- [ ] Generate Compose artifacts.
- [ ] Generate `.env.example`.
- [ ] Generate health-check scripts.
- [ ] Generate deployment notes.
- [ ] Generate troubleshooting guides.

Step 8.5: AI ops workflows

- [ ] Explain Compose topology.
- [ ] Diagnose failed service.
- [ ] Compare environment files.
- [ ] Propose minimal safe fix.
- [ ] Generate command plan.
- [ ] Summarize logs with citations.

Exit criteria:

- [ ] User can inspect local/container operations and approve safe mutations with audit logs.

## Phase 9: AI Assistant Orchestration

Goal: promote AI Assistant from chat panel to cross-studio orchestration layer.

Status: first chat, context packs, and backend ReAct runtime exist; orchestration planned.

Implemented:

- [x] OpenAI-compatible chat.
- [x] Streaming chat.
- [x] Context packs for files/directories/workspace root.
- [x] Backend ReAct runtime under `app/internal/agent/`.
- [x] Wails `RunAgent` binding.
- [x] First safe Agent run button.
- [x] First tool plan UI, surfaced through the always-visible AI Assistant sidebar.
- [x] Tool run persistence.

Step 9.1: Assistant workspace

- [x] Keep right sidebar as the always-visible quick assistant output.
- [ ] Add optional native long-run assistant workspace when multi-step work outgrows the sidebar.
- [ ] Add full assistant workspace for long runs.
- [ ] Add run history.
- [ ] Add thread/session browser.
- [ ] Add model/provider status panel.
- [ ] Add model suitability hints for selected task/context.
- [ ] Add tool-calling support indicator.

Step 9.2: Context sources

- [ ] Add git diff context.
- [ ] Add changed-files context.
- [ ] Add database schema context.
- [ ] Add query result context.
- [ ] Add analytics connector run context.
- [ ] Add document set context.
- [ ] Add operations log context.
- [ ] Add artifact lineage context.

Step 9.3: Agent modes

- [ ] Ask.
- [ ] Plan.
- [ ] Review.
- [ ] Edit.
- [ ] Research.
- [ ] Analyze.
- [ ] Debug Ops.
- [ ] Generate Artifact.
- [ ] Report Builder.

Step 9.4: Tool planning and approval

- [ ] Show proposed tool sequence before execution.
- [ ] Show expected inputs and outputs.
- [ ] Show risk level per action.
- [ ] Dry-run read-only actions.
- [ ] Pause mid-run for approvals.
- [ ] Stream each tool call and observation.
- [ ] Resume after approval.
- [ ] Stop/cancel long runs.
- [ ] Summarize what changed after multi-step runs.
- [ ] Compare generated outputs from a run.

Step 9.5: Memory, citations, and quality

- [ ] Add workspace memory store.
- [ ] Store accepted facts.
- [ ] Store decisions.
- [ ] Store preferred report style.
- [ ] Store ignored paths/connectors.
- [ ] Add citation inspector.
- [ ] Add weak-evidence warnings.
- [ ] Add unsupported-claim warnings.
- [ ] Add retry with another model.
- [ ] Add compare model outputs.
- [ ] Add ask-for-missing-context prompts.

Exit criteria:

- [ ] AI Assistant can coordinate multi-step work across Workbench, Data & Analytics, document, operations, and artifact contexts with citations and approvals.

## Phase 10: Artifact Studio And Provenance

Goal: make generated outputs durable, comparable, reproducible, and easy to navigate.

Status: first artifact browser, metadata, comparison, and lineage implemented.

Implemented:

- [x] Markdown report artifacts.
- [x] Assistant answer artifacts.
- [x] CSV export artifacts.
- [x] SVG chart artifacts.
- [x] Dataset summary artifacts.
- [x] Workspace scan report artifacts.
- [x] Sidecar provenance metadata.
- [x] Artifact list in the Artifact Studio route surface.
- [x] Artifact metadata panel.
- [x] Archive artifact.
- [x] Delete artifact with approval.
- [x] Open source context.
- [x] Compare generated artifacts.
- [x] Artifact lineage graph.
- [x] Export/import lineage JSON preview.

Next steps:

- [x] Move Artifact Studio to a first-class route entry with primary fallback surface.
- [ ] Replace Artifact fallback surface with native route layout and filters.
- [ ] Add artifact type filters.
- [ ] Add artifact search.
- [ ] Add artifact tags.
- [ ] Add artifact version timeline.
- [ ] Add graph diff for imported lineage JSON.
- [ ] Add stale artifact regeneration.
- [ ] Add artifact templates.
- [ ] Add dashboard/report bundle artifacts.
- [ ] Add presentation artifacts.
- [ ] Add generated config artifacts.
- [ ] Add diff/patch artifacts.
- [ ] Add reproducibility action that replays source queries/context where safe.

Exit criteria:

- [ ] User can trust and reproduce generated work.

## Phase 11: Metadata, Indexing, Search, And Reliability

Goal: make the app robust enough for large workspaces and long-lived projects.

Status: first SQLite metadata store and search foundations implemented.

Implemented:

- [x] SQLite metadata schema under `app/internal/appmeta/`.
- [x] `.nexusdesk/metadata/nexusdesk.sqlite` initialization through `modernc.org/sqlite`.
- [x] JSON compatibility mirroring.
- [x] Direct fresh-row writes for chat, approval, artifact, and tool-run records once metadata store exists.
- [x] Metadata browser for tables, columns, row counts, samples, and dataset SQL views.
- [x] Metadata history search across chat, artifacts, and tool runs.
- [x] SQL run history.
- [x] Dataset dependencies.
- [x] Workspace freshness polling.
- [x] Changed-file indicators.
- [x] Stale artifact/dataset warnings.
- [x] Stale context refresh action.
- [x] Shared redaction/truncation helpers under `app/internal/safety`.
- [x] Redacted provider and SQL errors.

Next steps:

- [ ] Move more local JSON stores to SQLite primary repositories.
- [ ] Make SQLite the primary store for chats, approvals, artifacts, tool runs, SQL runs, and dataset dependencies.
- [ ] Keep JSON stores only as compatibility, migration, backup, or export fallback after SQLite repositories are primary.
- [ ] Add migrations with versioned schema changes.
- [ ] Add full-text search.
- [ ] Add semantic search/embeddings when provider/model is configured.
- [ ] Add task/job table.
- [ ] Add connector run table.
- [ ] Add tab/session state table.
- [ ] Add document index table.
- [ ] Add git snapshot table.
- [ ] Add artifact lineage indexes for reports, charts, decks, data exports, configs, and code patches.
- [ ] Add metrics dashboard for provider failures by kind, root path, and workspace.
- [ ] Add index rebuild controls.
- [ ] Add large-workspace performance budgets.
- [ ] Add corruption recovery and export.

Step 11.1: Long-running job runner

- [ ] Add job runner for imports, OCR, dump restores, connector pulls, report generation, and large indexing work.
- [ ] Route slow/external work through jobs instead of blocking the shell: indexing, OCR, imports, connector pulls, dump restore, report generation, and long agent runs.
- [ ] Add cancelable task progress.
- [ ] Add task logs.
- [ ] Add retry failed task.
- [ ] Link task output to generated artifacts.

Exit criteria:

- [ ] Nexus Augentic Studio can maintain durable, searchable workspace memory over long-lived projects.

## Phase 12: Settings, Policies, Credentials, And Security

Goal: centralize user control over providers, connectors, tool permissions, credentials, and workspace policies.

Status: first LLM settings implemented; broader policy work planned.

Implemented:

- [x] Settings route surface for LLM provider.
- [x] API key redaction.
- [x] OS-protected credential sidecar where available.
- [x] Model dropdown.
- [x] Context-window and reserve controls.
- [x] Per-model context/reserve defaults shared between Chat and Settings.
- [x] Runtime context tuning from Ollama loaded-model diagnostics.
- [x] Connection probe.
- [x] Ollama diagnostics.
- [x] Approval log.
- [x] Modal approval foundation for high-risk UI actions.

Next steps:

- [x] Move Settings to a first-class route entry with primary fallback surface.
- [ ] Replace Settings fallback surface with native route layout.
- [ ] Add provider profiles.
- [ ] Add connector credential vault UI.
- [ ] Add per-workspace policy settings.
- [ ] Add tool allow/deny list.
- [ ] Add approval policy levels.
- [ ] Add shell execution policy.
- [ ] Add Docker mutation policy.
- [ ] Add database mutation policy.
- [ ] Add secret scanner settings.
- [ ] Add audit export.
- [ ] Add UI preferences.

Exit criteria:

- [ ] User can understand and control what Nexus Augentic Studio may read, run, write, and send to models/connectors.

## Phase 13: Testing, Packaging, And Release Readiness

Goal: keep the app stable as the studio scope grows.

Status: first verification loop implemented.

Implemented:

- [x] Go unit tests across backend packages.
- [x] Frontend production build.
- [x] Frontend smoke script.
- [x] Playwright visual smoke with Wails-free mocks.
- [x] Desktop Wails build.
- [x] Windows icon generation script.
- [x] Production build at `app/build/bin/app.exe`.

Next steps:

- [ ] Add behavior tests for main studio routing.
- [ ] Add behavior tests for IDE tree and git diffs.
- [ ] Add regression coverage that opening a workspace never auto-runs Git, long indexing, OCR, connector pulls, or other external/slow work.
- [ ] Add behavior tests for Data & Analytics notebooks/connectors.
- [ ] Add behavior tests for document extraction flows.
- [ ] Add behavior tests for operations safe actions.
- [ ] Add backend integration tests with temporary SQLite/Postgres containers.
- [ ] Add connector contract tests.
- [ ] Add fixture workspaces.
- [ ] Add crash/hang regression tests for folder open.
- [ ] Add release packaging notes.
- [ ] Add signed build plan.
- [ ] Add update strategy.

Exit criteria:

- [ ] A release candidate can be built, smoke-tested, and installed predictably.

## Phase 14: Extensibility And Team Future

Goal: support plugins, MCP, shared workspaces, and enterprise controls without weakening local-first safety.

Status: future.

Steps:

- [ ] Add MCP client support.
- [ ] Add external tool registry.
- [ ] Add custom tool definitions.
- [ ] Add plugin manifest model.
- [ ] Add team/shared workspace model.
- [ ] Add policy export/import.
- [ ] Add central model gateway support.
- [ ] Add audit bundle export.
- [ ] Add Docker Desktop extension investigation.
- [ ] Add marketplace-style template packs for reports/dashboards.

Exit criteria:

- [ ] Nexus Augentic Studio can be extended without giving external tools direct authority over files, shell, Docker, or databases.

## Phase 15: Architecture Hardening

Goal: keep the product architecture simple to reason about as features deepen, with small Wails adapters, focused frontend controllers, primary SQLite persistence, job-based long work, and a restrained product menu.

Status: first hardening slice expanded with Git backend/frontend extraction, preview-only stage/unstage planning, generated Wails binding isolation, and editor-outline extraction from the Workbench panel.

Step 15.1: Backend service facades

- [ ] Keep Wails method names stable while moving orchestration out of `app/app.go`.
- [ ] Create a `WorkspaceService` facade for open/refresh/search/preview/context/freshness/file operations.
- [ ] Create an `ArtifactService` facade for report creation, metadata, lineage, archive/delete, comparison, and regeneration.
- [ ] Create a `DatasetService` facade for profiling, queries, SQL runs, dependencies, charts, summaries, and connector routing.
- [ ] Create a `ChatService` facade for settings-aware chat requests, streaming, context packs, history persistence, and saved answer artifacts.
- [x] Create a `GitService` facade for read-only status/diffs first, then approval-governed stage/unstage/revert actions.
- [ ] Keep `app/app.go` as a thin Wails adapter that validates runtime availability and dispatches to services.
- [ ] Move helper functions from `app/app.go` into owned service packages when they stop being bridge-specific.

Step 15.2: Frontend controller hooks

- [ ] Split workspace state and actions from `NexusShell.tsx` into `useWorkspaceController`.
- [x] Split Git state and actions into `useGitController`.
- [ ] Split artifact state and actions into `useArtifactController`.
- [ ] Split dataset/SQL/connector state and actions into `useDatasetController`.
- [ ] Split chat/context/agent state and actions into `useChatController`.
- [ ] Keep `NexusShell.tsx` focused on layout composition, cross-panel wiring, modals, global shortcuts, and route/drawer placement.
- [x] Add smoke checks that generated Wails bindings remain isolated behind `app/frontend/src/api/wailsClient.ts`.

Step 15.3: Persistence simplification

- [ ] Promote SQLite metadata repositories to primary reads/writes for chats, approvals, artifacts, tool runs, SQL runs, and dataset dependencies.
- [ ] Add one migration path from existing JSON compatibility stores into SQLite.
- [ ] Keep JSON compatibility stores as export/backup/fallback only after primary SQLite reads are stable.
- [ ] Add repository tests proving SQLite and JSON fallback records do not diverge during migration.
- [ ] Add corruption/export recovery path before removing JSON-first assumptions.

Step 15.4: Job-based slow work

- [ ] Define a job model with ID, kind, workspace root, status, progress, log tail, started/completed timestamps, artifact outputs, and cancel state.
- [ ] Route long indexing, OCR, dump imports, connector pulls, report generation, and long agent runs through the job runner.
- [ ] Surface jobs in a Workbench/Activity panel with cancel, retry, inspect logs, and open artifact actions.
- [ ] Ensure folder open only scans the bounded tree needed for first render; deeper indexing must run as a cancelable job.
- [ ] Persist job records in SQLite and link outputs to artifacts/lineage.

Step 15.5: Navigation discipline

- [ ] Keep the primary rail limited to Workbench, Data & Analytics, Artifacts, and Settings until a capability has enough native workflow depth.
- [ ] Keep analytics connectors inside Data & Analytics until they justify a separate surface.
- [ ] Keep document intelligence contextual through Workbench, Artifacts, and Assistant until it justifies a separate surface.
- [ ] Keep operations capabilities contextual/read-only first, then promote only if Docker/log/runbook workflows become deep enough.
- [ ] Keep AI Assistant always visible as orchestration, not a default top-level route.

Exit criteria:

- [ ] `app/app.go` is a thin Wails adapter instead of the main use-case owner.
- [ ] `NexusShell.tsx` composes focused controllers instead of owning most application workflows.
- [ ] SQLite is the primary metadata store for durable workspace records.
- [ ] Slow and external work is cancelable, logged, and never blocks folder open.
- [ ] Product navigation remains small while capability depth grows inside existing surfaces.

## Next Logical Batch

Completed batch: Navigation Cleanup And Git Diff Review.

Steps:

1. [x] Reduce the primary rail to Workbench, Data & Analytics, Artifacts, and Settings.
2. [x] Keep AI Assistant as an always-visible orchestration sidebar instead of a separate route.
3. [x] Remove roadmap metadata and scan diagnostics from the default visible workbench chrome.
4. [x] Move Git/working-tree diff review into the bottom Git drawer.
5. [x] Render changed Git files as a directory tree.
6. [x] Add staged/unstaged per-file diffs.
7. [x] Add Unified, Split, and Diff Only review modes.
8. [x] Replace visible hunk text buttons with compact icon controls.
9. [x] Avoid automatic Git refresh on folder open to prevent desktop command-window flashes.
10. [x] Normalize unavailable Git state so opening non-git folders cannot crash the Workbench or Git drawer.
11. [x] Keep docs, tracker, smoke checks, and visual baselines aligned.

Recommended next batch: Safe Git Mutations And Workbench Utility Panels.

Steps:

1. [x] Extract first backend service facade without changing Wails contracts, starting with Git or Workspace.
2. [x] Extract first frontend controller hook from `NexusShell.tsx`, starting with `useGitController` or `useWorkspaceController`.
3. [x] Design the first Git mutation approval boundary: preview commands now report `requiresApproval` and `mutatesRepository`, while execution remains blocked until an approval-governed apply path exists.
4. [x] Add backend Git command planning for stage/unstage file with dry-run/status refresh outputs.
5. [x] Add preview-only frontend controls for stage/unstage file in the bottom Git drawer.
6. [x] Add hunk-level selection model without mutation first.
7. [x] Add destructive approval flow for revert/discard hunk.
8. [x] Add Workbench path/text search panel as a real utility panel, not only command palette search.
9. [x] Detect npm scripts and Go tests into a Tasks panel with read-only listing.
10. Keep visual smoke focused on no blank screen, no whole-window scroll, Git drawer behavior, route switching, and no slow/external work on folder open.

Reasoning: read-only navigation and diff review are now credible, stage/unstage previews establish the approval boundary without mutating the repository, hunk selection exists as UI state, hunk discard/revert now goes through the approval modal before backend patch application, Workbench has a real search utility panel backed by the existing safe workspace search flow, task/script detection now lists npm scripts plus Go test commands without running external processes, and Phase 2 now has split editor groups, outline navigation, Monaco go-to-definition dispatch, safe draft formatting, and encoding-aware save. The next correction path can move to Phase 4 Workbench depth or Phase 15 architecture hardening.

## Directory Ownership Notes

`app/internal/workspace/` owns safe workspace scanning, previews, search, context expansion, dataset queries, freshness, and file operations.

`app/internal/artifact/` owns deterministic artifact writes, provenance sidecars, listing, search, comparison, archive/delete, and scan-report creation.

`app/internal/agent/` owns the backend ReAct runtime, system prompt, action parsing, plan updates, observation handling, and working-memory pruning.

`app/agent_runtime.go` exposes `RunAgent` and maps model-requested tools to workspace-safe handlers.

`app/app_git.go` owns Wails-facing Git API types and bridge methods. `app/git_service.go` owns the first Git service facade for read-only status/diff operations, preview-only stage/unstage command planning, and approval-backed selected-hunk discard/revert patch application while preserving existing Wails contracts.

`app/app_tasks.go` owns Wails-facing read-only workspace task discovery. It scans bounded workspace paths, skips noisy output/dependency folders, parses `package.json` scripts, detects Go module test commands, and returns task metadata only. It does not execute npm, Go, shell, Docker, or other external commands.

`app/internal/agenttools/` owns deterministic tool descriptors and tool-run persistence.

`app/internal/appmeta/` owns SQLite metadata schema, migrations, metadata browser queries, metadata search, dataset dependencies, and SQL run history. `app/app_metadata.go` owns application-level mirroring between JSON compatibility stores, app actions, and that SQLite metadata store.

`app/internal/analytics/` owns read-only SQL-style dataset querying and DuckDB-compatible execution paths.

`app/internal/dbconnector/` owns workspace database connector surfaces. Today that means read-only SQLite files; future phases add server databases and dump sandboxes.

`app/internal/approval/` owns append-only approval/action records.

`app/process_windows.go` and `app/process_other.go` own platform-specific child process configuration. Windows desktop builds hide app-launched child processes so user-triggered Git refreshes and approved shell tools do not flash console windows.

`app/internal/storage/` owns local app config such as recent workspaces and non-secret LLM settings. Secret values must stay in credential storage or protected sidecars.

`app/frontend/src/api/wailsClient.ts` is the frontend boundary for generated Wails bindings.

`app/frontend/src/features/shell/NexusShell.tsx` is still the large shell orchestrator, but Wails imports, resize state, and studio navigation state have been extracted. It should keep shrinking as workspace, chat, artifact, and data controllers move into focused hooks.

`app/frontend/src/features/shell/useStudioNavigation.ts` owns active studio route state, primary route surface mapping, and contextual bottom drawer tab state.

`app/frontend/src/features/shell/useResizablePanels.ts` owns navigator, assistant, and bottom drawer sizing plus resize drag handlers.

`app/frontend/src/features/shell/useGitController.ts` owns Git status refresh, selected changed-file state, selected-file diff loading, preview-only stage/unstage action state, hunk action preview/apply state, null-response normalization, and the manual-only Git refresh boundary.

Workspace scan counters are diagnostic data, not primary navigation content. Keep them in scan reports/diagnostics instead of the always-visible sidebar header.

`app/frontend/src/features/shell/WorkbenchPanel.tsx` currently owns the editor/preview surface, pinned tabs, breadcrumbs, split editor group presentation, Monaco minimap control, and composition of editor-adjacent panels. Git status, working-tree diff output, and roadmap/studio-route metadata should not render above the editor tabs; those surfaces belong to Workbench utility panels, the bottom Git drawer, or documentation.

`app/frontend/src/features/shell/EditorOutlinePanel.tsx` owns the editor outline side panel presentation and symbol-selection callbacks.

`app/frontend/src/features/shell/editorOutline.ts` owns lightweight outline extraction for Markdown, TypeScript/JavaScript, Go, CSS, JSON, and YAML while richer language-service-backed symbol search remains pending.

`app/frontend/src/features/shell/CodeStudioPanel.tsx` owns the first reusable Workbench utility surface for editor session metrics, open tabs, workspace status, git branch/dirty summary, changed-file list, Workbench search results/actions, read-only detected task listings, and placeholders that will receive problem/review data.

`app/frontend/src/features/shell/GitDiffPanel.tsx` owns the bottom-drawer Git tab for selected changed-file review, preview-only stage/unstage controls, hunk selection state, approval-backed hunk discard/revert controls, and read-only staged/unstaged working-tree diffs.

`app/frontend/src/features/shell/DataOperationsPanel.tsx` currently owns Data route workflows.

`app/frontend/src/features/shell/ArtifactStudioPanel.tsx` currently owns Artifact Studio route workflows.

`app/frontend/src/features/shell/BottomStudioPanel.tsx` currently hosts reusable Settings, Data, Tools, Artifacts, Git, Approvals, Activity, and Code surfaces. The visible bottom drawer exposes Git, Approvals, and Activity; route-owned surfaces are rendered through the main nav until native studio layouts land.

`app/frontend/src/features/shell/WorkspaceRail.tsx` owns the compact branded main studio menu, active route selection, route accessibility state, and pending-route markers.

`app/frontend/src/brand/assets.ts` owns studio route labels, descriptions, command hints, pending-route metadata, and temporary route-to-surface mapping.

`services/` is reserved for development/test services. Runtime workspace state belongs under ignored `.nexusdesk/` folders inside user workspaces, not in this repository.
