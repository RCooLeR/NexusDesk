# Developer Experience

## Current Verification Loop

On the current Windows workstation, use this loop after backend, frontend, binding, or asset changes:

```powershell
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
go test ./...
npm.cmd run build
npm.cmd run smoke
npm.cmd run smoke:visual
wails build
```

Run Go commands from `app/`, frontend commands from `app/frontend/`, and Wails build commands from `app/`.

When Wails regenerates frontend bindings, `app/frontend/wailsjs/go/models.ts` can pick up whitespace-only changes. Clean those before committing if `git diff --check` reports trailing whitespace.

## Current Local Persistence

The current app still uses small JSON files in the user's config directory as the compatibility layer:

- `recent-workspaces.json`
- `llm-settings.json`
- `chat-history.json`

LLM API keys are not written into `llm-settings.json`. They are saved in a sidecar credential blob protected by the OS where available, while the JSON settings file keeps only a storage marker. Connector profile passwords/tokens follow the same rule: `connector-profiles.json` stores non-secret profile metadata and credential references, while secret material lives in a protected sidecar and is returned to the frontend only as a redacted marker. `EnsureSQLiteMetadataStore` now initializes `.nexusdesk/metadata/nexusdesk.sqlite` with `modernc.org/sqlite`, applies the workspace/chat/approval/artifact/tool-run plus dataset dependency/SQL run schema, and mirrors current JSON chat, approval, artifact, and tool-run records into SQLite. Once that store exists, fresh chat, approval, artifact, and tool-run rows are also appended directly to SQLite while JSON remains the compatibility fallback. Metadata search, dataset dependencies, and SQL run history live behind `app/internal/appmeta/` so the frontend can inspect history without reading workspace files directly.

## Chat Streaming

`AskLLMStream` emits `nexus:chat-stream` Wails events while `app/internal/llm/chat.go` reads OpenAI-compatible server-sent response chunks. The frontend listens in `NexusShell.tsx`, updates the in-flight assistant message per `delta`, then replaces it with the final persisted response or refreshed workspace chat history when the request completes. User and assistant messages in the same optimistic pair receive distinct timestamps because streaming updates use the assistant timestamp as their target key. LLM settings include a model context window and response reserve; Nexus Augentic Studio uses the remaining budget for selected-file and context-pack bytes, local Ollama-compatible requests include `num_ctx` and `num_predict`, and all compatible chat requests send `max_tokens` from the reserve.

Model selection is backed by `app/frontend/src/features/shell/llmModelCatalog.ts` and matching backend defaults in `app/internal/storage/llm_settings.go`. Choosing a curated local model immediately applies the largest configured context window for that model and derives the response reserve from that window. When the Ollama runtime probe reports a loaded model `context_length`, the frontend prefers that runtime value and saves the tuned setting, so the actual loaded runner wins over catalog guesses.

Selected directories and the workspace root also flow through the same streaming path. `app/internal/workspace/context.go` expands a selected directory or `.` into a capped set of previewable files, then `app/app.go` builds a context pack with a small manifest and file sections. The pack budget scales from the configured model context window, while still keeping ignored-folder, symlink, path traversal, encoding, PDF text, DOCX text, and table-dataset summary boundaries used by file previews.

The chat panel previews pinned context packs by calling `PreviewChatContextPack`, which uses the same backend collector as the send path. That keeps the visible file list aligned with what the model will actually receive, including truncation warnings when caps are reached. The bottom Activity Log records chat lifecycle steps such as request queued, stream listener attached, context budget reserved, first token received, stream completed, and failures, so long-running local model work has visible progress.

Chat history stores the source paths attached to each user/assistant pair so saved answer artifacts can use the answer's original context instead of whatever happens to be pinned later. Persisted assistant answers now append a compact source list when context paths exist, and saved Markdown answer artifacts include the same source citations in their header and footer. The chat header can retry the latest answered prompt or ask the current model/settings to compare against the latest answer using the same source paths. Assistant messages without explicit source context show a weak-evidence warning, and the composer shows a missing-context warning when nothing selectable or pinned can ground the next answer. The chat card also exposes local assistant memory and prompt-profile controls; the backend stores them in `app/internal/storage/assistant_profile.go` and prepends the active profile/memory as guidance, not source evidence.

## Local Models

The default local endpoint is `http://localhost:11434/v1`, targeting the `rcooler-ollama` Docker container on this workstation. The settings card recommends only local models at 26B parameters or below: `qwen3:4b-instruct`, `qwen3:8b`, `qwen3.5:9b`, `phi4:14b`, `phi4-reasoning:14b`, `gpt-oss:20b`, `mistral-small3.2:latest`, and `gemma4:26b`.

The current workstation runner is the sibling Compose stack at `../Llm/`. Keep the `ollama` service pinned to Ollama's CUDA 12 backend:

```powershell
OLLAMA_LLM_LIBRARY=cuda_v12
```

Without that pin, the current `ollama/ollama:latest` image can choose its CUDA 13 backend, fail with `CUDA driver version is insufficient for CUDA runtime version`, and silently load models on CPU with `size_vram: 0`.

GPU verification:

```powershell
cd ..\Llm
docker compose exec ollama nvidia-smi
docker compose logs ollama | Select-String "cuda_v12|offloaded|model weights"
Invoke-RestMethod http://localhost:11434/api/ps | ConvertTo-Json -Depth 10
```

For a healthy load, `/api/ps` should show nonzero `size_vram`, and the Ollama logs should include `offloaded ... layers to GPU` plus `model weights device=CUDA0`.

## Workspace Previews

`app/internal/workspace/preview.go` keeps text previews rooted and size-limited, decodes UTF-8, UTF-8 BOM, UTF-16, and Windows-1251 text variants, parses CSV, TSV, JSON, and NDJSON files into bounded table previews with lightweight column profiles from a larger capped sample, and renders common image/PDF files as capped data URLs for inline display. `app/internal/workspace/write.go` keeps file writes rooted, diff-previewed, and encoding-aware for UTF-8, UTF-8 BOM, UTF-16 LE/BE, and Windows-1251. PDFs also expose simple embedded text extraction by page when available, and DOCX files expose basic body text extraction. Chat context accepts text previews, DOCX text, extracted PDF text, and structured table profiles plus bounded samples, so binary payloads and data URLs are not sent to the model as source text.

`app/internal/workspace/search.go` owns the first workspace path/content search pass. It searches path names and previewable text content inside the same ignore and depth boundaries as scanning, and the advanced Wails search request can run regex and lightweight symbol matching for the Workbench search surface. `app/internal/workspace/symbols.go` owns the shared symbol extractor for Markdown, TypeScript/JavaScript, Go, CSS, JSON, and YAML style structures. `SearchWorkspace` now merges that result set with artifact metadata matches from `app/internal/artifact/` and persisted chat snippets from `app/internal/storage/chat_history.go`, so generated outputs and prior analysis are searchable from the same navigator surface. The Workbench utility panel also has a non-mutating replace preview for current search snippets; actual file writes still go through the editor diff/apply boundary. `app/internal/workspace/problems.go` owns the first read-only Problems scan for TODO/FIXME/HACK/BUG markers, merge-conflict markers, and invalid JSON. It uses the same bounded preview path and does not run compilers, language servers, shells, Git, Docker, or task commands. `app/internal/workspace/copy.go`, `move.go`, `delete.go`, and `write.go` own safe file mutations for the Workbench tree/editor. Cut/copy/paste in the tree is a clipboard intent until paste time, then the backend previews and applies the copy or move without overwriting targets or touching `.nexusdesk/`. `app/internal/workspace/dataset_query.go` owns the first table-dataset query flow for CSV, TSV, JSON, and NDJSON with bounded row results, text search, column filters, numeric comparisons, `contains`, `limit`, and simple `order by` clauses until a deeper DuckDB SQL layer is added. Dataset query exports rerun that same bounded query before writing a CSV artifact, so exported rows match the backend safety boundary rather than trusting frontend table state.

`app/internal/workspace/context.go` owns directory/project context expansion and context-pack previews. The UI can pin a selected directory or the workspace root, but the backend still decides which files are safe and useful enough to include. Table data candidates include CSV, TSV, JSON, JSONL, and NDJSON. `app/internal/workspace/freshness.go` owns the first file-change snapshot; the shell polls it to mark changed tree rows, warn when generated artifacts cite changed source paths, and flag dataset-derived views/snippets/reports when table or workbook source files change. The workbench can refresh a stale context preview from changed files and records that action in the local approval/metadata trail.

`app/app_git.go` owns the Wails-facing Git API types and bridge methods, while `app/git_service.go` owns the first Git service facade. `GetGitStatus`, `GetGitFileDiff`, `PreviewGitFileAction`, `ApplyGitFileAction`, `PreviewGitHunkAction`, and `ApplyGitHunkAction` keep their Wails contracts stable and dispatch to `GitService`, which runs bounded `git` commands against the active workspace root, detects the repository root, branch, short HEAD, ahead/behind text, porcelain changed-file rows, staged/unstaged groups, a capped staged diff, and a capped working-tree diff. `GetGitFileDiff` loads a capped read-only staged and unstaged diff for one selected changed file. `PreviewGitFileAction` plans stage/unstage commands and returns the command, current status, approval requirement, and message without mutating the repository; `ApplyGitFileAction` runs the same validated command only after frontend approval. Hunk actions are narrower: the backend rebuilds the selected hunk patch from the current Git diff by path, diff kind, and hunk index; unstaged hunks can be staged into the index or discarded from the working tree, and staged hunks can be unstaged or reverted from the index, only after the frontend approval modal confirms the action. No Git API performs full-file reset, checkout, or broad discard operations. The frontend does not run Git automatically on workspace open; the user must press Refresh git in the Git drawer or Workbench repository surface. Workbench consumes these through `useGitController`, which owns Git status, selected changed-file state, selected diff loading, file stage/unstage preview/apply state, hunk action preview/apply state, null-response normalization, and explicit refresh/actions. The bottom Git drawer tab owns selected changed-file review, directory-structured changed-file lists, file stage/unstage controls, read-only staged/unstaged diffs, unified/split/diff-only modes, icon hunk navigation, hunk selection state, approval-backed hunk stage/unstage/discard/revert controls, AI diff summaries, and AI commit-message drafts.

`app/app_tasks.go` owns workspace task discovery and the first safe task-run boundary. `ListWorkspaceTasks` scans bounded workspace paths, skips noisy folders such as `.git`, `.nexusdesk`, build output, dist output, and `node_modules`, parses `package.json` scripts, detects Go module test commands, and lists Docker Compose config-check tasks for compose files. `RunWorkspaceTask` does not accept arbitrary commands from the frontend; it re-discovers tasks, matches the requested task ID, validates the command shape, runs it from the discovered workspace-relative working directory with hidden Windows child-process flags, timeout, and capped stdout/stderr, then saves a Markdown task-run artifact plus approval/audit record. Task execution is user-triggered only and never runs on folder open.

On Windows, external child processes launched by the app are configured as hidden/no-console processes. That keeps user-triggered read-only Git refreshes and approved agent shell commands from flashing transient console windows over the desktop UI.

`app/frontend/src/features/shell/HighlightedCode.tsx` remains as the dependency-free fallback highlighter for non-Monaco preview paths. Text/code source previews and edit drafts now use the Monaco-backed components listed below.

## Dataset Profiles

`app/dataset_service.go` owns the Wails-facing data workflow orchestration while preserving the existing frontend method names. It dispatches dataset profiling, bounded row queries, saved filters, saved SQL snippets, DuckDB-compatible dataset SQL, read-only SQLite connector queries, chart artifacts, query exports, SQL report artifacts, dataset summaries, SQL run records, dataset dependency records, and dependency rebuilds. `app/internal/dataset/` owns the first persistent dataset profile pass and saved query history. CSV, TSV, JSON, JSONL, and NDJSON files reuse the workspace preview profiles; XLSX files expose workbook sheet names plus sheet dimensions, formula counts, named ranges, table ranges, and pivot table names from package XML; Parquet files validate the fixed `PAR1` header/footer and persist bounded file/footer/data byte metadata without schema decoding or full columnar scans; log files persist a bounded sample profile with levels, timestamp counts, stack trace counts, and repeated normalized patterns. Legacy binary XLS parsing returns conversion guidance instead of attempting unsupported binary parsing. Profiles are stored under `.nexusdesk/datasets/profiles.json` inside the active workspace. Saved lightweight row filters and read-only SQL snippets are stored separately under `.nexusdesk/datasets/queries.json` and capped per dataset. `app/internal/workspace/chart.go` owns the first table chart model: one category column, optional numeric value column, bar or line chart mode, bounded points, and no arbitrary SQL or model-rendered pixels.

The workbench topbar now has functional Preview, Explain, Summarize, Edit, and Report actions. Preview reloads the selected workspace node from disk, Explain sends a predefined grounded prompt when text context is available, Summarize sends selected file/directory context through chat and saves the result as a Markdown artifact, Edit uses the diff/apply write flow, and Report creates a Markdown artifact. Workbench now has a route-owned toolbar, persisted route/drawer/sidebar layout state, project-tree context menu shell, git status badges in the tree, read-only git branch/dirty summary, staged/unstaged changed-file groups, a Workbench utility search panel that reuses the safe workspace path/text/artifact/chat search backend and opens file/artifact matches from results, and a read-only Tasks panel for detected npm scripts and Go tests. The project tree supports cut/copy/paste file intents; paste prompts for the target path, previews the copy/move through backend validation, requires approval, refreshes the tree, and opens the resulting file. Ignored path samples are available behind an explicit tree control instead of default chrome. Drag/drop remains intentionally unimplemented: the design rule is that a drag can only create a visible move/copy intent and must still go through the same paste-style preview and approval boundary. The bottom Git drawer tab shows changed files as a directory tree and capped staged/working-tree diffs for the selected change with unified/split rendering, hunk navigation, assistant diff summaries, and assistant commit-message drafts. The Data & Analytics route owns dataset profiling plus query/chart/SQL workflows: it can persist CSV/TSV/JSON/NDJSON/XLSX dataset metadata, run a bounded row query for the selected table dataset, save/reuse queries, export the bounded result as a CSV artifact, preview chart points, create deterministic SVG bar or line chart artifacts, create deterministic Markdown dataset summaries, surface XLSX workbook metadata counts, and show read-only data source cards from the already-bounded workspace tree. Editor previews and drafts now use Monaco with language detection for common code, document, data, and operations files. Drafts show dirty state, persist per tab while navigating, clear stale diff previews after edits, support revert before apply, guard dirty tab close, expose a save-as-encoding selector, and use Ctrl+S to preview/apply through the same write path. Editor tabs can be pinned to stay at the front of the tab strip, breadcrumbs expose the active workspace path and can reopen visible ancestor nodes, split editor mode opens a second read-only editor group from already-open tabs, outline navigation extracts common headings/functions/types/selectors/keys and jumps Monaco to the selected line, Monaco definition lookup is available for source previews/editors where language services can resolve it, draft formatting is available while editing and still requires preview/apply before disk writes, and a minimap toggle controls Monaco preview/edit minimaps. New files start as draft tabs from Ctrl+N or the command palette, then use the same preview/apply boundary to create the file. Editor keyboard shortcuts include Ctrl+F for in-file find, Ctrl+W for active-tab close, and Ctrl+Tab / Ctrl+Shift+Tab for tab cycling. Ctrl+Shift+P opens the command palette for common workspace, editor, context, data, artifact, and chat actions.

## Frontend Structure

The shell is now mostly orchestration. Feature panels own stable presentation, while a small frontend API adapter isolates generated Wails bindings from React UI code:

- `app/frontend/src/components/ui.tsx` contains reusable UI atoms such as buttons, cards, status badges, and branded state panels.
- `app/frontend/src/api/wailsClient.ts` is the only frontend source module that imports generated Wails bindings directly. Shell and feature components import backend calls through this adapter.
- `app/frontend/src/features/shell/NexusShell.tsx` owns the composed desktop workbench state, global quick-open/command-palette shortcuts, and cross-panel navigation wiring.
- `app/frontend/src/features/shell/useStudioNavigation.ts` owns active studio route state, bottom drawer tab state, temporary route-to-surface mapping, and best-effort local persistence for route/drawer state.
- `app/frontend/src/features/shell/useResizablePanels.ts` owns navigator, assistant, and bottom drawer sizing plus drag handlers and best-effort local persistence for layout dimensions.
- `app/frontend/src/features/shell/useGitController.ts` owns Git status refresh, selected changed-file state, selected-file diff loading, file stage/unstage preview/apply state, approval-backed hunk action calls, null-response normalization, and the manual-only Git refresh boundary.
- `app/frontend/src/features/shell/QuickOpenPalette.tsx` owns the keyboard quick-open palette for workspace nodes and open editor tabs.
- `app/frontend/src/features/shell/CommandPalette.tsx` owns the keyboard command palette for workspace, editor, assistant, data, and artifact actions.
- `app/frontend/src/features/shell/codeAiActions.ts` owns pure Code AI prompt builders and single-file unified-diff parsing for assistant patch drafts. Shell code may orchestrate model calls and safe write previews, but patch parsing and prompt templates should stay in this module.
- `app/frontend/src/features/shell/CodeStudioPanel.tsx` owns reusable Workbench session metrics, open tabs, workspace status, git branch/dirty summary, staged/unstaged changed-file groups, selected changed-file state, active-file and git-diff review/test/patch/dependency/PR draft triggers, the Workbench path/text/symbol/regex search utility panel with non-mutating replace previews, lightweight Problems results, read-only detected task listings, and placeholder queues for deeper code review records.
- `app/frontend/src/features/shell/GitDiffPanel.tsx` owns the bottom-drawer Git tab for working-tree review, including selected changed-file state, directory-structured changed-file lists, file stage/unstage controls, per-file read-only staged/unstaged diffs, unified/split/diff-only rendering, hunk navigation, hunk selection, approval-backed hunk stage/unstage/discard/revert controls, assistant review/test/summary/commit/PR draft actions, and refresh controls.
- `app/frontend/src/features/shell/MonacoFileEditor.tsx` owns the lazy-loaded Monaco edit surface, worker wiring, language detection, editor-local Ctrl+S forwarding for draft writes, go-to-definition dispatch, and safe draft formatting dispatch.
- `app/frontend/src/features/shell/MonacoCodePreview.tsx` owns read-only Monaco previews, search decorations for source files, and go-to-definition dispatch where Monaco can resolve a target.
- `app/frontend/src/features/shell/monacoRuntime.ts` owns shared Monaco lazy-loading, worker setup, theme definition, and file language detection.
- `app/frontend/src/features/shell/AgentChatCard.tsx` owns the expanded chat presentation, full conversation scroll area, OpenAI-style composer with absolute-positioned model and Ask/Agent controls, assistant memory/profile controls, context pack list, retry/compare/save-answer action surface, weak-evidence and missing-context warnings, and delegates provider calls/history/artifact actions back to the shell.
- `app/frontend/src/features/shell/llmModelCatalog.ts` owns the curated local model dropdown, per-model context defaults, runtime context override helpers, and response reserve derivation used by both Settings and Chat.
- `app/frontend/src/features/shell/AgentToolPlanCard.tsx` owns the first visible agent tool plan preview, dry-run/execute controls, and recent tool-run summaries using backend tool descriptors and active context.
- `app/frontend/src/features/shell/ChatMessageContent.tsx` renders safe dependency-free Markdown-style chat content, including headings, lists, tables, code fences, inline code, and bold text.
- `app/frontend/src/features/shell/LLMSettingsCard.tsx` owns the provider settings form, model context-window controls, response-reserve controls, and delegates persistence/probe actions back to the shell.
- `app/frontend/src/features/shell/ConnectorProfilesCard.tsx` owns the first connector profile settings card. It saves read-only profile metadata with caps/timeouts, displays saved profiles with redacted credential markers, and delegates persistence plus explicit PostgreSQL Test/Inspect actions to the shell.
- `app/frontend/src/features/shell/ToolTimeline.tsx` owns the visible tool event timeline presentation.
- `app/frontend/src/features/shell/BottomStudioPanel.tsx` owns reusable utility surfaces for Workbench, Settings, Data & Analytics, Tools, Artifacts, Git, Approvals, and Activity. Git, Approvals, and Activity are exposed as bottom drawer tabs; route-owned surfaces are rendered from the main nav instead of being duplicated in the drawer.
- `app/frontend/src/features/shell/DataOperationsPanel.tsx` owns the Data & Analytics route surface for dataset profiling, query/chart/SQL workflows, read-only SQLite connector queries, manual SQLite connector schema inspection, Operations inspector, metadata browser/search, and workspace freshness controls. Its source cards are bounded to the already-scanned workspace tree and expose only explicit user-triggered actions: Open, Profile for supported file datasets, Inspect for SQLite, and disabled planned actions for dump/import or conversion work.
- `app/frontend/src/features/shell/ArtifactStudioPanel.tsx` owns artifact browsing, metadata actions, comparison summaries, and selectable lineage graph presentation inside the Artifact route.
- `app/frontend/src/features/shell/WorkspaceNavigator.tsx` owns the workspace lockup, search controls, recent workspace list, fallback scaffold list, indexed IDE-style project tree presentation, ignored-path sample toggle, and file context menu, with depth guides, disclosure state, type badges, selected rows, cut/copy/paste actions, and changed-file markers inside the resizable sidebar. The default sidebar header shows the workspace name/path; scan counters stay out of the primary chrome and belong in scan reports or diagnostics. `NexusShell.tsx` owns the resizable navigator width state.
- `app/frontend/src/features/shell/WorkbenchPanel.tsx` owns the active context topbar, closeable and pinnable editor tab strip, breadcrumb path navigation, source preview/editor presentation, split editor group layout, read-only secondary tab preview, find-in-file, Markdown source/rendered switching, Monaco minimap toggle, active-file AI review trigger, safe edit/diff controls, and fallback workflow preview. Roadmap/studio-route metadata must stay out of the editor header. Git status and working-tree diffs must stay in the Workbench repository surface and bottom Git drawer, not above editor tabs in the workbench.
- `app/frontend/src/features/shell/EditorOutlinePanel.tsx` owns the editor outline side panel presentation and symbol selection UI.
- `app/frontend/src/features/shell/editorOutline.ts` owns lightweight outline extraction for Markdown, TypeScript/JavaScript, Go, CSS, JSON, and YAML until richer editor language-service hooks land.
- `app/frontend/src/features/shell/WorkspaceRail.tsx` owns the compact branded main menu for implemented surfaces: Workbench, Data & Analytics, Artifacts, and Settings. Rail selections change the primary workspace, while the bottom drawer remains contextual and the assistant remains always visible.
- `app/frontend/src/brand/assets.ts` owns product logo asset references, Font Awesome UI icon mapping, route labels, descriptions, command hints, hidden roadmap route metadata, and fallback route-to-surface mapping. Product logos stay reserved for app identity, while controls, route glyphs, tree chevrons, and file/data/document icons use Font Awesome.
- `app/frontend/src/features/shell/AgentPanel.tsx` composes only the grounded assistant header and chat card. `NexusShell.tsx` owns resizable right-sidebar width up to 50% of the window and resizable bottom-drawer height up to 70% of the window.

`App.css` keeps the desktop shell fixed to the window and pushes overflow into the interactive surfaces that actually need it: workspace tree/search results, quick-open and command-palette results, source preview, dataset query results, capability list, chat thread, route surfaces, bottom Git/approvals/activity tabs, and tool timeline.

## Frontend Smoke Checks

`app/frontend/scripts/smoke.mjs` checks that the built frontend and key shell source files still expose the main foundation functionality: Wails bindings, simplified main routing, Workbench surface, IDE-style project tree, search, quick-open, command palette, Monaco preview/edit surfaces, find-in-file, context packs, file create/update/delete/move flows, dataset profiling/querying/saved queries/exporting/charting/summaries, read-only SQL, route-owned artifact actions/comparison/lineage, agent tool plan dry-run/execute controls, Compose parsing, approval log styling, resizable navigator/right-sidebar/bottom-drawer styling, and the production `dist/index.html` entrypoint. Run it after `npm.cmd run build`.

`app/frontend/scripts/visual-smoke.mjs` is now an enforced Playwright screenshot smoke with Wails-free mocks for workspace, dataset, metadata, chat, tool-run, artifact, lineage export, and metadata history flows. Shared mocks live in `app/frontend/scripts/visual-fixtures.mjs` so future Playwright scenarios can reuse the same workspace/data/metadata setup instead of copying a large inline fixture. It captures desktop and mobile screenshots plus `visual-baselines/manifest.json` from the built `dist/index.html`, and fails if the production build or Playwright dependency is missing. On this workstation, install/run with `$env:NODE_OPTIONS='--use-system-ca'` because npm needs the system CA store.

## Artifact Creation

`app/internal/artifact/` owns deterministic artifact writes, provenance sidecars, metadata lookup, artifact search, listing, archive/delete, comparison, and scan-report creation. The first flows create timestamped Markdown reports from selected previews, timestamped Markdown artifacts from assistant answers, timestamped CSV exports from dataset queries, timestamped SVG chart artifacts from CSV chart models, timestamped Markdown dataset summaries, and timestamped workspace scan reports under `.nexusdesk/artifacts/`, use exclusive file creation to avoid overwrites, and return the new workspace-relative path so the UI can refresh and select it. Each artifact also gets a sibling `.meta.json` file with kind, source, source paths, prompt/configuration, model when relevant, context path, and creation timestamp when available. Saved assistant answers preserve the model's Markdown and include source/context metadata before the generated body. The Artifact Studio route lists Markdown, CSV, and SVG artifacts from that folder so generated outputs remain visible after creation, shows metadata for the active generated artifact, and can open the artifact source context, archive the artifact, delete it through approval prompts, compare it with a prior artifact of the same kind, or inspect lineage.

## Approval Log

`app/internal/approval/` owns the first append-only action log. Applied text writes, deletes, moves, reports, saved chat artifacts, chart artifacts, query exports, dataset summaries, scan reports, artifact archives, and artifact deletes append records under `.nexusdesk/approvals/log.json`. The backend agent runtime also records approved high-impact write and shell actions here, and the bottom Approvals tab surfaces the current log.

`app/internal/agent/` owns the first backend ReAct runtime. It builds the Nexus Augentic Studio agent prompt, runs bounded Thought/Action/Observation loops, accepts `update_plan` steps, caps observations, prunes old working memory, emits live `nexus:agent-run` model/tool events, and returns final answers with ordered tool-call output. If the loop uses its iteration budget before a final answer, the runtime makes one no-tool finalization request and marks the result with a stop reason so the UI can show an honest fallback instead of treating the raw limit text as the answer. While the run is active, chat shows only the last one or two activity messages; the bottom Activity tab and tool-run records keep the fuller trace. When the backend returns, the assistant placeholder is replaced by the final answer body. `app/agent_runtime.go` exposes `RunAgent` and maps model-requested tools to workspace-safe filesystem, dataset, artifact, shell, and registered tool handlers.

`app/internal/agenttools/` owns tool descriptors and tool run records. Dry-runs, explicit executions, and registered-tool calls from `RunAgent` persist under `.nexusdesk/tool-runs/log.json` with inputs, output summaries, risk, approval ID, duration, and errors. The agent panel can expand recent tool runs to inspect captured inputs, output/error text, approval reference, duration, and replay/diff affordances.

`app/internal/appmeta/` owns the SQLite metadata schema, manifest, real database initialization, metadata browser, JSON-to-SQLite mirror, direct fresh-row writes, metadata search, dataset dependency records, and SQL run history under `.nexusdesk/metadata/`. `app/app_metadata.go` owns the Wails-app-level orchestration that mirrors JSON stores into that database, records artifact/dataset/SQL metadata after user actions, and converts mirrored records back into UI-facing records. `InspectMetadataStore` returns table columns, row counts, sample rows, and dataset SQL view summaries for the workbench, where users can select tables, filter columns, and copy sample rows.

`app/internal/analytics/` owns the first read-only SQL-style dataset query surface. It accepts a constrained `SELECT` subset, blocks mutation keywords, and executes through bounded table-dataset query primitives by default. A real DuckDB `database/sql` execution path is implemented behind the `duckdb` build tag for CGO-enabled machines; the current Windows verification loop keeps CGO disabled and therefore uses the safe fallback path. SQL results can be exported as Markdown artifacts that include SQL text, engine, row counts, preview rows, and source dataset citations.

`app/internal/dbconnector/` owns the first workspace SQLite connector and the first external PostgreSQL profile runner. SQLite opens `.sqlite`, `.sqlite3`, and `.db` files inside the active workspace in read-only mode, accepts bounded `SELECT`/`WITH` queries, blocks mutation-oriented SQL, and returns capped rows to the Data & Analytics connector panel. SQLite query requests carry an explicit request ID, result cap, and timeout; `app/dataset_service.go` registers request cancellation callbacks, applies the timeout context, records completed/failed SQL run metadata, and redacts connector errors before persistence. The Data & Analytics connector panel exposes the row cap, timeout, and cancel controls next to the query action. The connector package also exposes the first connector metadata model and a manual SQLite schema inspector for tables, views, columns, indexes, row counts, capped samples, declared foreign keys, and conservative `*_id` relationship hints without executing user-provided SQL. The frontend schema browser can select inspected objects, copy a quoted `SELECT *` query into the editor, run an explicit capped preview, ask the assistant to explain only the selected schema object from inspected metadata, save SQLite connector queries under the separate `sqlite-sql` kind, and filter SQLite query history from SQL run records. SQLite query exports use `app/internal/artifact/` to write CSV artifacts or Markdown reports with SQL, cap, timeout, preview rows, source database citation, and provenance metadata; `DatasetService` records matching SQL run and dependency rows for lineage. PostgreSQL profile actions are explicit settings actions today: credentials resolve only at test/inspect/query time, sessions are set read-only, statement timeouts are applied, mutation SQL is rejected, and schema inspection returns tables, views, columns, indexes, declared foreign keys, and first inferred relationship hints. MySQL/MariaDB, SQL Server, and DuckDB external runners remain planned.

`GetArtifactLineage` in `app/app.go` assembles lineage from artifact metadata, chat source paths, and persisted tool runs. It returns relationship counts for the Artifact Studio selectable graph layout, so users can filter by node kind, select nodes, inspect nearby relationships, and jump back to visible source files. The app can also export that graph as a JSON artifact and import a JSON graph preview for debugging and future sync work.

## Completed Batch: Agent Execution And Analytics Foundations

The Agent Execution And Analytics Foundations batch turned the tool planning surface into auditable controlled actions and added the first metadata/analytics foundations:

- Tool execution planner: proposed plan rows now map to backend dry-run and execute requests.
- Approval integration: medium/high-risk executions require the modal approval prompt.
- Tool run persistence: records include inputs, output summary, risk, approval ID, duration, and errors.
- SQLite metadata: `.nexusdesk/metadata/schema.sql` and a manifest prepare the migration-compatible schema.
- DuckDB-compatible analytics: Data & Analytics can run constrained read-only SQL over table previews.
- Artifact versions: generated artifacts can be compared for size delta and added/removed line summaries.
- Visual baselines: Playwright smoke writes desktop/mobile baselines and a manifest when installed.

## Completed Batch: Context, Persistence, And Analytics Depth

- SQLite metadata initialization now creates a real `.nexusdesk/metadata/nexusdesk.sqlite` database and applies the schema through `modernc.org/sqlite`.
- DuckDB query execution is available behind the `duckdb` build tag for CGO-enabled environments, with bounded dataset SQL fallback in the default loop.
- Tool-run details expose inputs, outputs/errors, approvals, replay, and target diff affordances.
- Assistant answers and saved Markdown answer artifacts include source citations.
- Artifact lineage links chats, tool runs, source files, and generated artifacts.
- Workspace freshness polling marks changed files and stale generated artifacts.
- Playwright is installed as a dev dependency, visual smoke is enforced, and visual baselines are captured.

## Completed Batch: Real Studio Workflows

- SQLite metadata mirrors JSON chat, approval, artifact, and tool-run records into the active database.
- Metadata Browser inspects SQLite metadata tables and dataset SQL views.
- Artifact lineage filtering can focus source, chat, tool, or artifact relationships.
- Chat messages and context-pack previews warn when cited files change.
- Data & Analytics invalidates visible query/chart/profile state when the selected dataset changes on disk.
- SQL result artifacts save SQL text, engine, row counts, preview rows, and source dataset citations.
- Playwright visual smoke asserts navigator resizing, tool-run details, metadata browser, lineage filtering, panel scrolling, and freshness warnings.

## Completed Batch: Studio Scale And Reliability

- SQLite mirror rows now serve prepared reads for chat history, approvals, artifacts, and tool runs after the metadata store exists.
- Metadata Browser supports table selection, column filtering, and copyable row samples.
- Artifact lineage has a selectable graph layout with relationship counts and source navigation.
- Stale-context refresh rebuilds a context preview from changed files and records the refresh action.
- Dataset freshness now flags dataset-derived views/snippets/reports when source data files change.
- SQL snippets are saved separately from lightweight row filters per dataset.
- Playwright visual smoke uses Wails-free mocked workspace, dataset, metadata, chat, and artifact fixtures.

## Completed Batch: Studio Depth And Connectors

- Fresh chat, approval, artifact, and tool-run records now write directly into SQLite metadata when the store exists.
- Metadata history search returns chat, artifact, and tool-run snippets backed by SQLite metadata queries.
- Dataset lineage dependencies are recorded for saved SQL snippets, exported reports, chart artifacts, query exports, and summaries.
- Saved SQL execution history records status, row counts, messages, and artifact links.
- Data & Analytics has a first read-only SQLite workspace database connector surface.
- Artifact lineage can be exported as JSON and imported for debugging/preview workflows.
- Playwright visual smoke mocks moved into a reusable fixture helper.

## Prepared Next Batch

The next implementation batch should turn the new history/connector records into richer studio workflows:

- Add explicit refresh/rebuild buttons for dataset dependencies so saved SQL reports, charts, summaries, and exports can be regenerated from recorded inputs.
- Add a richer metadata history tab with filters by kind, time, source path, and jump-to-chat/artifact/tool actions.
- Expand the SQLite connector with relationship hints and explain-schema actions. Manual schema inspection, explicit query guardrails, saved connector queries, schema-node previews, query history, CSV exports, Markdown report exports, connector query lineage, declared/inferred relationship hints, and selected-object schema explanation are implemented.
- Add artifact lineage JSON import comparison in the UI, including validation errors and graph diff previews.
- Promote dataset dependency and SQL run records into first-class UI navigation from Data & Analytics, Artifacts, and Metadata Browser.
- Add connector approval policy docs/tests for read-only proofs, blocked SQL statements, result caps, and redacted errors.
- Start a DuckDB multi-file workspace dataset surface for joins across CSV/XLSX-derived tables.
- Split large shell orchestration state where connector/history flows start to crowd `NexusShell.tsx`.

## File Writes

`app/internal/workspace/write.go` owns the first text write approval boundary. The frontend can draft edits for selected text files or create a new text/code file draft, preserve drafts per editor tab, request a diff preview, and only then apply the write through a rooted workspace method. Changing a draft clears the existing diff proposal, so an apply action always corresponds to the current draft. `app/internal/workspace/delete.go` owns the first file delete boundary: selected files are backend-validated, metadata paths/directories/symlinks are rejected, and the frontend requires confirmation before removal. `app/internal/workspace/move.go` owns rename/move validation and rejects traversal, metadata targets, directories, symlinks, same-path moves, directory-like targets, and overwrites. Direct writes to `.nexusdesk/`, traversal paths, symlinks, directories, oversized previews, and binary existing files are rejected before apply.

## Goals

Nexus Augentic Studio should be easy to run, easy to test, easy to reason about as an IDE/data/analytics studio, and hard to accidentally make unsafe.

Current developer setup requires:

- Go
- Node.js or Bun for frontend development
- Wails
- optional Ollama or another LLM endpoint

Planned data/connector work will add:

- SQLite through `modernc.org/sqlite`
- optional DuckDB dependency behind the `duckdb` build tag and CGO
- Docker only for connector testing or packaging experiments

## Repository Shape

Current structure:

```text
app/                           Wails desktop app
app/app.go                     Wails-facing Go adapter with stable frontend bindings
app/workspace_service.go       Workspace open/search/preview/mutation/freshness service facade
app/artifact_service.go        Artifact report/list/metadata/archive/delete/compare service facade
app/dataset_service.go         Dataset profiling/query/SQL/connector/artifact service facade
app/app_tasks.go               Read-only workspace task discovery for npm scripts and Go tests
app/app_metadata.go            App-level metadata mirror and record orchestration
app/main.go                    Wails entrypoint
app/internal/artifact/         Markdown artifact creation, listing, provenance
app/internal/agenttools/       Backend tool descriptors for agent-capable actions
app/internal/dataset/          CSV/TSV/JSON/NDJSON/XLSX profile persistence and workbook metadata
app/internal/llm/              OpenAI-compatible probe, chat, streaming
app/internal/storage/          JSON stores for recent workspaces, LLM settings, chat history, assistant profile/memory
app/internal/workspace/        Safe scanning, preview, search, context packs, copy/move/delete/write file operations
app/frontend/                  React + TypeScript frontend
app/frontend/src/              Workbench UI source
app/frontend/wailsjs/          Generated Wails bindings
app/build/                     Wails packaging metadata and ignored binary output
docs/                          Product, engineering, and brand docs
docs/brand/                    Brand book, generated assets, and design tokens
services/                      Development and testing helper services
services/docker-compose.yml    Placeholder for helper service definitions
tracker.md                     Implementation tracker
```

Target internal structure as the backend grows:

```text
internal/app/                  App lifecycle and Wails bindings
internal/config/               Typed config and validation
internal/settings/             User settings and model profiles
internal/workspace/            Workspace registration, roots, policies
internal/files/                File tree, safe paths, preview detection
internal/documents/            Text/PDF/Office/image extraction
internal/datasets/             Excel, CSV, DuckDB, profiles
internal/search/               Workspace search and context building
internal/agent/                Agent loop and tool orchestration
internal/llm/                  LLM gateway and provider adapters
internal/tools/                Built-in tool definitions and execution
internal/artifacts/            Reports, charts, generated files
internal/connectors/           DB, Docker, marketing, web/search connectors
internal/security/             Approvals, policy, redaction, risk levels
internal/storage/              SQLite repositories and migrations
internal/observability/        Logs, metrics, diagnostics
frontend/                      React/Svelte app
frontend/src/components/       UI components
frontend/src/features/         Workspace, editor, chat, data, Docker
frontend/src/lib/              API client and shared types
migrations/                    SQLite migrations
docs/                          Product and engineering docs
app/frontend/src/components/   Shared UI components
app/frontend/src/features/     Workspace, editor, chat, data, Docker
app/frontend/src/lib/          API client and shared types
app/migrations/                SQLite migrations
app/examples/                  Example workspaces and configs
app/scripts/                   Build, test, package, fixtures
```

Keep implementation notes and planning docs aligned with directories that exist. Do not document future directories as existing until they are created.

## Coding Principles

- Keep business rules out of Wails handlers.
- Keep file path safety in one shared module.
- Keep tool risk levels explicit.
- Keep model provider details behind the LLM gateway.
- Keep prompts versioned and testable.
- Keep generated AI text separate from source content.
- Keep original files auditable.
- Prefer typed structs over loosely typed maps at service boundaries.
- Prefer small interfaces for tools, storage, models, and connectors.
- Prefer deterministic tools over model-only behavior.
- Every risky action should pass through the approval system.

## Backend Interfaces

Example LLM interface:

```go
type LLMProvider interface {
    Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
    StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
    Capabilities(ctx context.Context) ProviderCapabilities
}
```

Example tool interface:

```go
type Tool interface {
    Name() string
    RiskLevel() RiskLevel
    InputSchema() json.RawMessage
    Run(ctx context.Context, input json.RawMessage, scope ToolScope) (ToolResult, error)
}
```

Example approval rule:

```go
type ApprovalPolicy interface {
    Evaluate(ctx context.Context, request ToolRequest) (ApprovalDecision, error)
}
```

## Testing Strategy

Unit tests:

- safe path resolution
- ignore rule matching
- file type detection
- document chunking
- dataset profiling
- SQL safety checks
- tool schema validation
- tool risk policies
- LLM response parsing
- context pack building
- artifact path generation

Integration tests:

- open fixture workspace
- index fixture files
- preview text, image, PDF, and spreadsheet files
- chat with fake model provider
- run tool loop with fake tools
- create artifact with approval
- query DuckDB dataset
- inspect fake Docker connector
- run database read-only query against fixture database

Evaluation tests:

- code explanation questions
- document summary questions
- spreadsheet analysis questions
- marketing report questions
- Docker log questions
- database schema questions
- path traversal attempts
- risky write requests
- weak-context questions

## Local Commands

Current command set:

```powershell
cd app
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
npm.cmd install
npm.cmd run build
go test ./...
wails build
```

The `NODE_OPTIONS` value is needed on this Windows workstation because Node/npm does not trust the registry certificate chain without the system CA store. Do not replace this with disabled TLS verification.

Planned command set:

```bash
wails dev
go run ./cmd/nexus migrate
go run ./cmd/nexus index --workspace ./examples/workspace
go run ./cmd/nexus eval --suite ./examples/eval/basic.yaml
```

## Debugging Tools

Developers and internal users need:

- workspace index report
- file extraction preview
- chunk viewer
- dataset profile viewer
- search result explanation JSON
- context pack preview
- prompt preview
- model response raw view
- tool call timeline
- approval log
- artifact source chain
- database query inspector
- Docker connector inspector

## Fixtures

Keep small test fixtures:

```text
examples/workspace-code/
examples/workspace-docs/
examples/workspace-excel/
examples/workspace-marketing/
examples/workspace-docker/
examples/workspace-database/
```

Each fixture should include:

- sample files
- expected index result
- example questions
- expected sources
- expected safe tool behavior

## Documentation Rule

Every module should document:

- what it owns
- what it does not own
- key inputs and outputs
- failure behavior
- config it depends on
- security assumptions
- tests that protect it

This keeps Nexus Augentic Studio maintainable as it grows from a local prototype into a serious desktop studio.
