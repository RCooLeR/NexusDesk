# Developer Experience

## Current Verification Loop

Primary development has moved to the Fyne-native `nexus-app/`. The Wails app is preserved under `app-wails/` as the reference implementation and migration source.

Fyne framework-independent checks:

```powershell
cd nexus-app
$env:GOFLAGS='-mod=readonly'
go test ./internal/domain ./internal/services/... ./internal/ui/shell ./internal/ui/theme ./internal/brand
```

Full native app run/build requires CGO and a Windows C compiler:

```powershell
cd nexus-app
.\scripts\dev-env.ps1 -Build
.\scripts\dev-env.ps1 -Run
```

Current build status on this workstation: MSYS2 UCRT64 GCC is installed under `C:\msys64\ucrt64\bin`, `nexus-app/scripts/dev-env.ps1` configures the current shell, focused native package tests pass, full `go build -o build\nexusdesk.exe .` succeeds, and `go run .` has been smoke-verified under CGO. `CGO_ENABLED=0 go build .` still fails because the Fyne OpenGL binding has no buildable files without CGO.

Legacy Wails reference verification:

```powershell
cd app-wails
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
go test ./...
cd frontend
npm.cmd run build
npm.cmd run smoke
npm.cmd run smoke:visual
npm.cmd run smoke:gallery
cd ..
wails build
```

Run legacy Go commands from `app-wails/`, frontend commands from `app-wails/frontend/`, and Wails build commands from `app-wails/`. `smoke:gallery` writes ignored review artifacts under `.codex-ui-screenshots/loaded/`; it is the broader manual UI gallery and can be skipped for tiny non-UI backend-only changes.

When Wails regenerates frontend bindings, `app-wails/frontend/wailsjs/go/models.ts` can pick up whitespace-only changes. Clean those before committing if `git diff --check` reports trailing whitespace.

## Current Local Persistence

The current app still uses small JSON files in the user's config directory as the compatibility layer:

- `recent-workspaces.json`
- `llm-settings.json`
- `chat-history.json`

LLM API keys are not written into `llm-settings.json`. They are saved in a sidecar credential blob protected by the OS where available, while the JSON settings file keeps only a storage marker. Connector profile passwords/tokens follow the same rule: `connector-profiles.json` stores non-secret profile metadata and credential references, while secret material lives in a protected sidecar and is returned to the frontend only as a redacted marker. Windows uses DPAPI today; macOS and Linux builds refuse secret persistence until Keychain and Secret Service/libsecret backends exist. `EnsureSQLiteMetadataStore` now initializes `.nexusdesk/metadata/nexusdesk.sqlite` with `modernc.org/sqlite`, applies the workspace/chat/approval/artifact/tool-run plus dataset dependency/SQL run schema, and mirrors current JSON chat, approval, artifact, and tool-run records into SQLite. Once that store exists, fresh chat, approval, artifact, and tool-run rows are also appended directly to SQLite while JSON remains the compatibility fallback. Metadata search, dataset dependencies, and SQL run history live behind `app/internal/appmeta/` so the frontend can inspect history without reading workspace files directly.

## Chat Streaming

`AskLLMStream` emits `nexus:chat-stream` Wails events while `app/internal/llm/chat.go` reads OpenAI-compatible server-sent response chunks. The frontend listens in `NexusShell.tsx`, updates the in-flight assistant message per `delta`, then replaces it with the final cited response or refreshed workspace chat history when the request completes. User and assistant messages in the same optimistic pair receive distinct timestamps because streaming updates use the assistant timestamp as their target key. LLM settings include a model context window and response reserve; Nexus Augentic Studio uses the remaining budget for selected-file and context-pack bytes, local Ollama-compatible requests include `num_ctx` and `num_predict`, and all compatible chat requests send `max_tokens` from the reserve.

`prepareChat` also includes the most recent user/assistant history turns in `llm.ChatRequest.Conversation`, bounded by the selected model's context budget, before adding the current user prompt. `app/internal/llm/chat.go` accepts only user and assistant turns from history, ignores any other role, and keeps system instructions owned by Nexus.

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

`app/internal/workspace/preview.go` keeps text previews rooted and size-limited, decodes UTF-8, UTF-8 BOM, UTF-16, and Windows-1251 text variants, parses CSV, TSV, JSON, and NDJSON files into bounded table previews with lightweight column profiles from a larger capped sample, and renders common image/PDF files as capped data URLs for inline display. `app/internal/workspace/write.go` keeps text/code writes rooted, diff-previewed, and encoding-aware for UTF-8, UTF-8 BOM, UTF-16 LE/BE, and Windows-1251; it also exposes a separate binary write boundary for base64 payloads with size and SHA-256 preview metadata instead of fake text diffs, while blocking direct model-authored executable binary targets such as `.exe`, `.dll`, `.msi`, and `.scr`. `app/internal/workspace/patch.go` owns the agent-facing unified patch boundary for multi-file text/code edits: patches are size-limited, rooted, blocked from `.nexusdesk/`, matched against current file content hunk-by-hunk, previewed through the normal diff flow, and applied only after approval. PDFs also expose simple embedded text extraction by page when available, and DOCX files expose basic body text extraction. Chat context accepts text previews, DOCX text, extracted PDF text, and structured table profiles plus bounded samples, so binary payloads and data URLs are not sent to the model as source text.

The native Fyne migration ports these workspace behaviors capability-by-capability into `nexus-app/internal/services/workspace`. The native safe-write slice now keeps write path validation, text diff preview, append/apply, UTF-8/UTF-8 BOM/UTF-16/Windows-1251 encoding, and rollback snapshots in service files (`write.go`, `write_path.go`, `write_diff.go`, `write_encoding.go`, and `rollback.go`). Native file-operation services (`file_ops.go`, `file_ops_path.go`) cover create, delete, copy, move, and rename with rooted validation, metadata guards, operation previews, and rollback records. The native navigator exposes quick action buttons plus tree-row secondary-click context menus for create/copy/rename/delete, relative-path copy, and assistant-context selection; file mutations still route through the service layer. Its tree controls can expand the selected branch, collapse all branches, and opt into safe ignored-folder visibility while keeping `.nexusdesk` metadata and symlinks hidden. The native draft editor Save action calls this service, marks the saved tab clean through `internal/services/editor`, and must not write directly to workspace files from UI code.

Native workspace search lives in `nexus-app/internal/services/workspace/search.go`. It walks the workspace inside ignore/depth boundaries, returns capped path/content matches, uses preview-safe text extraction for content matches, supports optional regex mode, and is rendered by the Fyne bottom Search tab rather than by a background indexer.

Native Problems scanning lives in `nexus-app/internal/services/workspace/problems.go`. It uses the same bounded preview-safe read path as search and reports only lightweight local signals: TODO/FIXME/HACK/BUG markers, merge-conflict markers, and invalid JSON. The Fyne Problems tab triggers scans manually and opens matched files as preview/editor tabs.

Native dataset profiling starts in `nexus-app/internal/services/datasets`. The first slice profiles the selected workspace CSV, TSV, or JSON file through the existing preview-safe read boundary, reports sample row counts, field names, inferred lightweight types, empty/non-empty counts, sample values, JSON top-level shape, and truncation when the preview cap applies. The Fyne bottom Data tab owns only the "Profile selected" intent and read-only profile rendering; broader row queries, SQL notebooks, connectors, charts, dependency records, and large-file profiling remain planned native ports.

Native Git status, selected-file diffs, parsed hunks, file-level stage/unstage actions, and index-only hunk stage/unstage actions live in `nexus-app/internal/services/git`. The service is manual-refresh/manual-select only, parses `git status --porcelain=v1 --branch`, splits staged and unstaged rows, validates selected repository-relative paths, loads capped staged and unstaged diffs, parses unified diff hunk metadata, runs selected-file `git add -- path` and `git restore --staged -- path`, applies selected hunks with `git apply --cached` or `git apply --cached --reverse`, and suppresses Windows command windows for spawned Git processes. The Fyne Git panel renders changed files grouped by directory, projects compact status badges into the Workbench tree from the last manual refresh, shows read-only unified, split, and diff-only selected-file diff views, exposes previous/next hunk navigation, confirms file-level and hunk-level stage/unstage actions, then refreshes repository status and the selected diff. Destructive hunk discard/revert and broader approval-backed Git mutations remain later native ports.

Native task discovery and the first safe task-run boundary live in `nexus-app/internal/services/tasks`. Discovery is manual, bounded, skips noisy folders such as `.git`, `.nexusdesk`, build output, dist output, vendor, and `node_modules`, parses `package.json` scripts, detects Go module test commands, and lists Docker Compose config-check tasks. Running a task re-discovers the workspace task list, matches only a discovered task ID, validates the command shape, checks the working directory stays inside the workspace, runs with timeout and capped stdout/stderr, and hides Windows child-process windows. Native task runs now start through `nexus-app/internal/services/jobs`, which provides job IDs, status, log tail, completion state, cancellation contexts, and optional persistence. The native bottom Tasks tab can discover tasks, confirm a selected run, show status, and render the last captured stdout/stderr in a read-only output area; the Jobs tab lists recent task jobs and can request cancellation for running jobs. Completed task runs also write Markdown report artifacts and JSON sidecar metadata through `nexus-app/internal/services/artifacts`. The native Artifacts tab now uses that service for recursive generated-output listing, metadata search, archive/delete actions, bounded previews, task-report lineage rendering, same-kind generated-output comparison diffs, comparison report exports, artifact-to-assistant context pinning, cited-source open/pin actions, first document extraction report exports for Markdown/TXT/HTML/XML files through `nexus-app/internal/services/documents`, and first document-set Markdown report generation from the selected file, folder, or project root through the same bounded context-pack extraction path; it is still smaller than the full Wails Artifact Studio because richer non-Markdown artifact writers remain future ports.

Native metadata persistence starts in `nexus-app/internal/services/metadata`. Opening a workspace ensures `.nexusdesk/metadata/schema.sql`, `.nexusdesk/metadata/sqlite-manifest.json`, and `.nexusdesk/metadata/nexusdesk.sqlite` exist through `modernc.org/sqlite`; the shell attaches that store to the Jobs service so jobs reload per workspace and start/log/finish/cancel changes persist. Completed task runs are also recorded in `task_runs` with command, cwd, status, exit code, bounded stdout/stderr, timestamps, duration, and `artifact_path` when a report file was created. Native agent runs now persist through `agent_runs` plus ordered `tool_runs`, capturing the prompt, job ID, status, final message, plan, source paths, iterations, stop reason, tool arguments, observations, errors, risk, mutation flag, and timestamps.

Native workspace history starts in `nexus-app/internal/services/history`. It merges bounded chat search results, generated artifact metadata from the artifact store, persisted jobs, and persisted agent runs into one newest-first list with optional kind/query filters. The Fyne bottom History tab owns only list/detail rendering and jump actions: artifacts open in the Artifact preview/lineage flow, chats populate the Chat detail surface, jobs refresh the Jobs tab, and agent runs open through the Agent Audit detail flow. Artifact rows are still filesystem/sidecar backed until a native artifact SQLite repository lands.

Native non-secret settings live in `nexus-app/internal/services/settings`. The first store persists provider, base URL, model, context-window size, and response reserve to the user config directory, while the Fyne Settings tab owns form rendering and validation. API keys and other secrets remain out of this JSON settings file.

Native LLM provider transport lives in `nexus-app/internal/services/llm`. It is UI-independent and ports the OpenAI-compatible chat/completions path, streaming SSE deltas, `/models` provider probing, Ollama `/api/ps` runtime diagnostics, context-window and response-reserve options, and workspace-context quoting with Nexus sentinel escaping. The package accepts a neutral `llm.Config` plus `ConfigFromSettings` for the current non-secret settings store, leaving secure API-key storage as a later platform service instead of putting secrets in the settings JSON.

Native assistant orchestration now starts in `nexus-app/internal/services/assistant`. The Fyne assistant panel dispatches Ask-mode prompts through that service instead of calling provider transport directly, streams deltas back to the response view with `fyne.Do`, keeps the user's prompt entry untouched, and includes explicit pinned context roots before falling back to the currently selected file, directory, or workspace context path as a bounded model context pack. The panel can pin the selected tree item, pin generated artifacts from the Artifacts tab, pin the workspace root, remove individual pins, and clear the pack; the service still owns deduplication, model-budget sizing, and context-pack construction for Ask mode. Ask mode persists successful user/assistant messages into `.nexusdesk/metadata/nexusdesk.sqlite`, reloads the latest workspace turns when a workspace opens, and sends those recent user/assistant turns before the current prompt so the native panel behaves conversationally without storing history in UI widgets. `nexus-app/internal/services/metadata` also exposes bounded chat search over role, content, model, and source-path metadata, and the Fyne bottom Chat tab renders searchable persisted chat rows plus a read-only detail pane. Chat history rows can seed a new Agent follow-up prompt and repin the original source paths, keeping prior answers useful without creating a second composer. The same assistant panel can switch to Agent mode, which now attaches the same pinned-or-selected context pack before running `nexus-app/internal/services/agent` with the native deterministic tool dispatcher, shows only the last one or two model/tool activity messages while the run is active, writes fuller events into the Activity tab and agent job log, persists the final run/tool audit in SQLite, refreshes the bottom Agent Audit tab for read-only run/tool history, and replaces the temporary tail with the final answer.

Native agent runtime now starts in `nexus-app/internal/services/agent`. It is UI-independent, resolves provider settings through the native settings store, calls the native LLM gateway, accepts an injected deterministic tool executor, parses ReAct-style `Action: tool({...})` and `Final Answer:` messages, handles built-in `update_plan`, emits model/tool/final events, keeps a backend emergency loop guard, preserves bounded observations, and appends a verification note when the model claims a workspace mutation without a successful mutating tool observation. `nexus-app/internal/services/tools` owns the first native deterministic dispatcher and descriptor list for agent-requested tools: context packs, file previews, workspace search, Problems, Git status, Git file diffs, discovered task listing, approval-gated task execution, approval-gated safe text/code `write_file` and `append_file`, approval-gated `copy_file`, `move_file`, `delete_file`, exact-match `apply_patch`, rollback listing, and approval-gated rollback application. Agent mutation calls use `nexus-app/internal/services/workspace` safe mutation methods, so full-project access is required and each write, append, copy, move, delete, or patch operation gets rollback coverage. The Fyne shell now wraps each Agent mode request in a durable job, saves the completed agent/tool audit through `nexus-app/internal/services/metadata`, and renders persisted agent runs plus per-run tool observations in the bottom Agent Audit tab.

Native approvals now start in `nexus-app/internal/services/approvals`. The service writes append-only approval records to `.nexusdesk/approvals/log.json`, persists full-project access policy in `.nexusdesk/approvals/policy.json`, expires project trust by timestamp, and records grant/revoke decisions. The Fyne bottom Approvals tab can refresh records, show full project access status, grant one hour of workspace-scoped project access, and revoke it. This policy is intentionally separate from shell execution and does not grant arbitrary command access.

Native context packing lives in `nexus-app/internal/services/workspace/context.go`. It accepts explicit workspace-relative files, directories, or `.` for the project root, skips ignored/noisy folders, rejects traversal and symlinks, caps scanned entries/depth/files/bytes, and builds a manifest plus preview-safe text sections from text, table, DOCX, and PDF previews.

`app/internal/workspace/search.go` owns the first workspace path/content search pass. It searches path names and previewable text content inside the same ignore and depth boundaries as scanning, and the advanced Wails search request can run regex and lightweight symbol matching for the Workbench search surface. `app/internal/workspace/symbols.go` owns the shared symbol extractor for Markdown, TypeScript/JavaScript, Go, CSS, JSON, and YAML style structures. `SearchWorkspace` now merges that result set with artifact metadata matches from `app/internal/artifact/` and persisted chat snippets from `app/internal/storage/chat_history.go`, so generated outputs and prior analysis are searchable from the same navigator surface. The Workbench utility panel also has a non-mutating replace preview for current search snippets; actual file writes still go through the editor diff/apply boundary. `app/internal/workspace/problems.go` owns the first read-only Problems scan for TODO/FIXME/HACK/BUG markers, merge-conflict markers, and invalid JSON. It uses the same bounded preview path and does not run compilers, language servers, shells, Git, Docker, or task commands. `app/internal/workspace/copy.go`, `move.go`, `delete.go`, `patch.go`, and `write.go` own safe file mutations for the Workbench tree/editor and agent. Cut/copy/paste in the tree is a clipboard intent until paste time, then the backend previews and applies the copy or move without overwriting targets or touching `.nexusdesk/`. `app/internal/workspace/dataset_query.go` owns the first table-dataset query flow for CSV, TSV, JSON, and NDJSON with bounded row results, text search, column filters, numeric comparisons, `contains`, `limit`, and simple `order by` clauses until a deeper DuckDB SQL layer is added. Dataset query exports rerun that same bounded query before writing a CSV artifact, so exported rows match the backend safety boundary rather than trusting frontend table state.

`app/internal/workspace/context.go` owns directory/project context expansion and context-pack previews. The UI can pin a selected directory or the workspace root, but the backend still decides which files are safe and useful enough to include. Table data candidates include CSV, TSV, JSON, JSONL, and NDJSON. `app/internal/workspace/freshness.go` owns the first file-change snapshot; the shell polls it to mark changed tree rows, warn when generated artifacts cite changed source paths, and flag dataset-derived views/snippets/reports when table or workbook source files change. The workbench can refresh a stale context preview from changed files and records that action in the local approval/metadata trail.

`app/app_git.go` owns the Wails-facing Git API aliases and bridge methods, while `app/internal/gitservice/` owns the Git implementation. `GetGitStatus`, `GetGitFileDiff`, `PreviewGitFileAction`, `ApplyGitFileAction`, `PreviewGitHunkAction`, and `ApplyGitHunkAction` keep their Wails contracts stable and dispatch to `gitservice.Service`, which runs bounded `git` commands against the active workspace root, detects the repository root, branch, short HEAD, ahead/behind text, porcelain changed-file rows, staged/unstaged groups, a capped staged diff, and a capped working-tree diff. `GetGitFileDiff` loads a capped read-only staged and unstaged diff for one selected changed file. `PreviewGitFileAction` plans stage/unstage commands and returns the command, current status, approval requirement, and message without mutating the repository; `ApplyGitFileAction` runs the same validated command only after frontend approval. Hunk actions are narrower: the backend rebuilds the selected hunk patch from the current Git diff by path, diff kind, and hunk index; unstaged hunks can be staged into the index or discarded from the working tree, and staged hunks can be unstaged or reverted from the index, only after the frontend approval modal confirms the action. No Git API performs full-file reset, checkout, or broad discard operations. The frontend does not run Git automatically on workspace open; the user must press Refresh git in the Git drawer or Workbench repository surface. Workbench consumes these through `useGitController`, which owns Git status, selected changed-file state, selected diff loading, file stage/unstage preview/apply state, hunk action preview/apply state, null-response normalization, and explicit refresh/actions. The bottom Git drawer tab owns selected changed-file review, directory-structured changed-file lists, file stage/unstage controls, read-only staged/unstaged diffs, unified/split/diff-only modes, icon hunk navigation, hunk selection state, approval-backed hunk stage/unstage/discard/revert controls, AI diff summaries, and AI commit-message drafts.

`app/app_tasks.go` owns workspace task discovery and the first safe task-run boundary. `ListWorkspaceTasks` scans bounded workspace paths, skips noisy folders such as `.git`, `.nexusdesk`, build output, dist output, and `node_modules`, parses `package.json` scripts, detects Go module test commands, and lists Docker Compose config-check tasks for compose files. `RunWorkspaceTask` does not accept arbitrary commands from the frontend; it re-discovers tasks, matches the requested task ID, validates the command shape, runs it from the discovered workspace-relative working directory with hidden Windows child-process flags, timeout, and capped stdout/stderr, then saves a Markdown task-run artifact plus approval/audit record. Task execution is user-triggered only and never runs on folder open.

On Windows, external child processes launched by the app are configured as hidden/no-console processes. That keeps user-triggered read-only Git refreshes and approved agent shell commands from flashing transient console windows over the desktop UI.

`app/frontend/src/features/shell/HighlightedCode.tsx` remains as the dependency-free fallback highlighter for non-Monaco preview paths. Text/code source previews and edit drafts now use the Monaco-backed components listed below.

## Dataset Profiles

`app/internal/artifactsvc/` owns artifact workflow orchestration while preserving the Wails-facing methods on `App`: Markdown reports, scan reports, generated Markdown artifacts, listing, metadata, archive, delete, and compare. `app/dataset_service.go` owns the Wails-facing data workflow orchestration while preserving the existing frontend method names. It dispatches dataset profiling, bounded row queries, saved filters, saved SQL snippets, saved SQL notebooks, DuckDB-compatible dataset SQL, read-only SQLite connector queries, chart artifacts, query exports, SQL report artifacts, dataset summaries, SQL run records, dataset dependency records, and dependency rebuilds. `app/internal/dataset/` owns the first persistent dataset profile pass and saved query history. CSV, TSV, JSON, JSONL, and NDJSON files reuse the workspace preview profiles; XLSX files expose workbook sheet names plus sheet dimensions, formula counts, named ranges, table ranges, and pivot table names from package XML; Parquet files validate the fixed `PAR1` header/footer and persist bounded file/footer/data byte metadata without schema decoding or full columnar scans; log files persist a bounded sample profile with levels, timestamp counts, stack trace counts, and repeated normalized patterns. Legacy binary XLS parsing returns conversion guidance instead of attempting unsupported binary parsing. Profiles are stored under `.nexusdesk/datasets/profiles.json` inside the active workspace. Saved lightweight row filters and read-only SQL snippets are stored separately under `.nexusdesk/datasets/queries.json` and capped per dataset. Saved SQL notebooks are stored under `.nexusdesk/datasets/notebooks.json`, capped per dataset, and contain only cell labels, cell kinds, SQL text, and timestamps. The first dataset SQL notebook shell manages multiple editable SQL/chart cells, can save/load durable notebooks for the selected dataset, sends SQL cells through the existing bounded read-only SQL runner, embeds bounded chart preview/create controls in chart cells, and displays rows, run summary, explain-plan lines, plus a SQL run history browser with status/text filters, selected-run details, and reuse/rerun actions. `app/internal/workspace/chart.go` owns the first table chart model: one category column, optional numeric value column, bar or line chart mode, bounded points, and no arbitrary SQL or model-rendered pixels.

The workbench topbar now has functional Preview, Explain, Summarize, Edit, and Report actions. Preview reloads the selected workspace node from disk, Explain sends a predefined grounded prompt when text context is available, Summarize sends selected file/directory context through chat and saves the result as a Markdown artifact, Edit uses the diff/apply write flow, and Report creates a Markdown artifact. Workbench now has a route-owned toolbar, persisted route/drawer/sidebar layout state, project-tree context menu shell, git status badges in the tree, read-only git branch/dirty summary, staged/unstaged changed-file groups, a Workbench utility search panel that reuses the safe workspace path/text/artifact/chat search backend and opens file/artifact matches from results, and a read-only Tasks panel for detected npm scripts and Go tests. The project tree supports cut/copy/paste file intents; paste prompts for the target path, previews the copy/move through backend validation, requires approval, refreshes the tree, and opens the resulting file. Ignored path samples are available behind an explicit tree control instead of default chrome. Drag/drop remains intentionally unimplemented: the design rule is that a drag can only create a visible move/copy intent and must still go through the same paste-style preview and approval boundary. The bottom Git drawer tab shows changed files as a directory tree and capped staged/working-tree diffs for the selected change with unified/split rendering, hunk navigation, assistant diff summaries, and assistant commit-message drafts. The Data & Analytics route owns dataset profiling plus query/chart/SQL workflows: it can persist CSV/TSV/JSON/NDJSON/XLSX dataset metadata, run a bounded row query for the selected table dataset, save/reuse queries, export the bounded result as a CSV artifact, preview chart points, create deterministic SVG bar or line chart artifacts, create deterministic Markdown dataset summaries, surface XLSX workbook metadata counts, and show read-only data source cards from the already-bounded workspace tree. Editor previews and drafts now use Monaco with language detection for common code, document, data, and operations files. Drafts show dirty state, persist per tab while navigating, clear stale diff previews after edits, support revert before apply, guard dirty tab close, expose a save-as-encoding selector, and use Ctrl+S to preview/apply through the same write path. Editor tabs can be pinned to stay at the front of the tab strip, breadcrumbs expose the active workspace path and can reopen visible ancestor nodes, split editor mode opens a second read-only editor group from already-open tabs, outline navigation extracts common headings/functions/types/selectors/keys and jumps Monaco to the selected line, Monaco definition lookup is available for source previews/editors where language services can resolve it, draft formatting is available while editing and still requires preview/apply before disk writes, and a minimap toggle controls Monaco preview/edit minimaps. New files start as draft tabs from Ctrl+N or the command palette, then use the same preview/apply boundary to create the file. Editor keyboard shortcuts include Ctrl+F for in-file find, Ctrl+W for active-tab close, and Ctrl+Tab / Ctrl+Shift+Tab for tab cycling. Ctrl+Shift+P opens the command palette for common workspace, editor, context, data, artifact, and chat actions.

## Frontend Structure

The shell is now mostly orchestration. Feature panels own stable presentation, while a small frontend API adapter isolates generated Wails bindings from React UI code:

- `app/frontend/src/components/ui.tsx` contains reusable UI atoms such as styled buttons, icon buttons, cards, status badges, route-local surface tabs, and branded state panels.
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
- `app/frontend/src/features/shell/ConnectorProfilesCard.tsx` owns the first connector profile settings card. It saves read-only profile metadata with caps/timeouts, displays saved profiles with redacted credential markers, delegates persistence plus explicit PostgreSQL/MySQL/MariaDB/SQL Server/DuckDB Test/Inspect actions to the shell, and renders inspected external schema metadata with the shared connector metadata browser and AI schema explanation action.
- `app/frontend/src/features/shell/ConnectorMetadataBrowser.tsx` owns connector schema navigation for inspected metadata. It can browse tables/views, column summaries, indexes, capped samples, textual relationships, and the first ERD-like relationship map for both workspace SQLite and saved external SQL profiles.
- `app/frontend/src/features/shell/AccessPolicyCard.tsx` owns Settings access and approval policy presentation. It exposes guarded versus full project-file access, guarded versus full read-only data-source access, shell policy intent, and reset-to-guarded controls while delegating approval-backed policy changes to the shell.
- `app/frontend/src/features/shell/ToolTimeline.tsx` owns the visible tool event timeline presentation.
- `app/frontend/src/features/shell/BottomStudioPanel.tsx` owns reusable utility surfaces for Workbench, Settings, Data & Analytics, Tools, Artifacts, Git, Approvals, and Activity. Git, Approvals, and Activity are exposed as bottom drawer tabs; route-owned surfaces are rendered from the main nav instead of being duplicated in the drawer. The Settings route uses Provider and Connectors tabs so runtime model settings and credential-backed connector profiles remain separate workflows.
- `app/frontend/src/features/shell/DataOperationsPanel.tsx` owns the Data & Analytics route surface for dataset profiling, query/chart/SQL workflows, read-only SQLite connector queries, manual SQLite connector schema inspection, Operations inspector, metadata browser/search, and workspace freshness controls. It separates those workflows into Sources, Operations, and Metadata route-local tabs so the main route stays scannable beside the assistant sidebar. Its source cards are bounded to the already-scanned workspace tree and expose only explicit user-triggered actions: Open, Profile for supported file datasets, Inspect for SQLite, and disabled planned actions for dump/import or conversion work. SQLite schema browsing now reuses `ConnectorMetadataBrowser` so external connector inspections and workspace database inspections share one presentation model.
- `app/frontend/src/features/shell/ArtifactStudioPanel.tsx` owns artifact browsing, metadata actions, comparison summaries, and selectable lineage graph presentation inside the Artifact route. It uses Library, Metadata, and Lineage route-local tabs instead of stacking every artifact workflow into one long panel.
- `app/frontend/src/features/shell/WorkspaceRail.tsx` owns the expandable desktop main navigation. The compact rail uses Font Awesome route icons; the expanded rail shows the horizontal Nexus logo plus route labels while preserving the same route state.
- `app/frontend/src/features/shell/WorkspaceNavigator.tsx` owns workspace search controls, recent workspace list, fallback scaffold list, indexed IDE-style project tree presentation, ignored-path sample toggle, and file context menu, with depth guides, disclosure state, type badges, selected rows, cut/copy/paste actions, Escape-to-close behavior, and changed-file markers inside the resizable sidebar. Product branding stays in the main rail instead of the file navigator; scan counters stay out of the primary chrome and belong in scan reports or diagnostics. `NexusShell.tsx` owns the resizable navigator width state.
- `app/frontend/src/features/shell/WorkbenchPanel.tsx` owns the active context topbar, closeable and pinnable editor tab strip, breadcrumb path navigation, source preview/editor presentation, split editor group layout, read-only secondary tab preview, find-in-file, Markdown source/rendered switching, Monaco minimap toggle, active-file AI review trigger, safe edit/diff controls, and fallback workflow preview. Roadmap/studio-route metadata must stay out of the editor header. Git status and working-tree diffs must stay in the Workbench repository surface and bottom Git drawer, not above editor tabs in the workbench.
- `app/frontend/src/features/shell/EditorOutlinePanel.tsx` owns the editor outline side panel presentation and symbol selection UI.
- `app/frontend/src/features/shell/editorOutline.ts` owns lightweight outline extraction for Markdown, TypeScript/JavaScript, Go, CSS, JSON, and YAML until richer editor language-service hooks land.
- `app/frontend/src/features/shell/WorkspaceRail.tsx` owns the compact branded main menu for implemented surfaces: Workbench, Data & Analytics, Artifacts, and Settings. Rail selections change the primary workspace, while the bottom drawer remains contextual and the assistant remains always visible.
- `app/frontend/src/brand/assets.ts` owns product logo asset references, Font Awesome UI icon mapping, route labels, descriptions, command hints, hidden roadmap route metadata, and fallback route-to-surface mapping. Product logos stay reserved for app identity, while controls, route glyphs, tree chevrons, and file/data/document icons use Font Awesome.
- `app/frontend/src/features/shell/AgentPanel.tsx` composes only the grounded assistant header and chat card. `NexusShell.tsx` owns resizable right-sidebar width up to 50% of the window and resizable bottom-drawer height up to 70% of the window.

`App.css` keeps the desktop shell fixed to the window and pushes overflow into the interactive surfaces that actually need it: workspace tree/search results, quick-open and command-palette results, Workbench action rows on compact widths, source preview, dataset query results, capability list, chat thread, route surfaces, bottom Git/approvals/activity tabs, and tool timeline. Primary route surfaces use container-safe grids and route-local tabs so Workbench, Data & Analytics, Artifacts, and Settings do not create horizontal page scroll when the assistant sidebar and bottom drawer are visible. The Git drawer keeps repository commands in a horizontal action rail so the changed-file tree and diff viewer retain vertical room. The expandable desktop rail uses a CSS rail-width variable so navigator resizing remains accurate in compact and expanded modes. The compact mobile layout removes duplicate navigator branding and prioritizes workspace/search controls before the stacked main surface, assistant, and drawer panels.

## Frontend Smoke Checks

`app/frontend/scripts/smoke.mjs` checks that the built frontend and key shell source files still expose the main foundation functionality: Wails bindings, simplified main routing, Workbench surface, IDE-style project tree, search, quick-open, command palette, Monaco preview/edit surfaces, find-in-file, context packs, file create/update/delete/move flows, dataset profiling/querying/saved queries/exporting/charting/summaries, read-only SQL, route-owned artifact actions/comparison/lineage, agent tool plan dry-run/execute controls, Compose parsing, approval log styling, resizable navigator/right-sidebar/bottom-drawer styling, and the production `dist/index.html` entrypoint. Run it after `npm.cmd run build`.

`app/frontend/scripts/visual-smoke.mjs` is now an enforced Playwright screenshot smoke with Wails-free mocks for workspace, dataset, metadata, chat, tool-run, artifact, lineage export, approval modal, and metadata history flows. Shared mocks live in `app/frontend/scripts/visual-fixtures.mjs` so future Playwright scenarios can reuse the same workspace/data/metadata setup instead of copying a large inline fixture. It walks the route-local Data & Analytics, Artifact, and Settings tabs before taking desktop/mobile baselines, captures screenshots plus `visual-baselines/manifest.json` from the built `dist/index.html`, and fails if the production build or Playwright dependency is missing.

`app/frontend/scripts/visual-gallery.mjs` captures the broader manual UI gallery from the production build across Workbench compact/expanded rail states, project-tree context menu, approval modal, Git/Approvals/Activity drawer tabs, Agent composer write-access mode, Data & Analytics tabs, Artifact tabs, Settings Provider/Connectors/Access tabs, command palette, quick open, and compact mobile views. It writes a contact sheet plus individual screenshots under ignored `.codex-ui-screenshots/loaded/`, and asserts transient overlays such as the file-tree context menu are dismissed before later captures. On this workstation, install/run with `$env:NODE_OPTIONS='--use-system-ca'` because npm needs the system CA store.

## Artifact Creation

`app/internal/artifact/` owns deterministic artifact writes, provenance sidecars, metadata lookup, artifact search, listing, archive/delete, comparison, and scan-report creation. The first flows create timestamped Markdown reports from selected previews, timestamped Markdown artifacts from assistant answers, timestamped CSV exports from dataset queries, timestamped SVG chart artifacts from CSV chart models, timestamped Markdown dataset summaries, and timestamped workspace scan reports under `.nexusdesk/artifacts/`, use exclusive file creation to avoid overwrites, and return the new workspace-relative path so the UI can refresh and select it. Each artifact also gets a sibling `.meta.json` file with kind, source, source paths, prompt/configuration, model when relevant, context path, and creation timestamp when available. Saved assistant answers preserve the model's Markdown and include source/context metadata before the generated body. The Artifact Studio route lists Markdown, CSV, and SVG artifacts from that folder so generated outputs remain visible after creation, shows metadata for the active generated artifact, and can open the artifact source context, archive the artifact, delete it through approval prompts, compare it with a prior artifact of the same kind, or inspect lineage.

## Approval Log

`app/internal/approval/` owns the first append-only action log. Applied text writes, deletes, moves, reports, saved chat artifacts, chart artifacts, query exports, dataset summaries, scan reports, artifact archives, and artifact deletes append records under `.nexusdesk/approvals/log.json`. The backend agent runtime also records approved high-impact write and shell actions here, and the bottom Approvals tab surfaces the current log.

`app/internal/agent/` owns the first backend ReAct runtime. It builds the Nexus Augentic Studio agent prompt, runs Thought/Action/Observation loops without a frontend-supplied iteration cap, accepts `update_plan` steps, caps observations, prunes old working memory, emits live `nexus:agent-run` model/tool events, and returns final answers with ordered tool-call output. The agent can now request the same bounded context-pack builder used by chat through `read_context`, so files, directories, and the project root can be inspected through one rooted, capped observation before edits. It can also request `read_git_diff` for the existing bounded Git status and staged/unstaged diff context without mutating the repository, `read_changed_files` for capped previews of current Git working tree files, `read_git_history` for bounded repository or file commit history, `read_git_blame` for bounded line attribution, `read_problems` for the lightweight Problems scan, `list_tasks` for discovered npm/Go/Compose tasks, approval-gated `run_task` for the existing safe task runner, `list_artifacts`/`read_artifact` for generated outputs plus metadata, `read_artifact_lineage` for the source/chat/tool/artifact relationship graph, `profile_dataset`/`query_dataset`/`query_dataset_sql` for bounded dataset context, `inspect_sqlite`/`query_sqlite` for read-only workspace database schema and result context, `read_document_set` for bounded Markdown/TXT/PDF/DOCX/HTML/XML document sets, `inspect_operations` for read-only Dockerfile, Compose, environment, script, config, or log evidence with environment-like secret redaction, approval-gated `web_fetch` for capped HTTP(S) text sources, and `list_rollbacks`/`rollback_file_mutation` for undoing approved workspace file mutations. If the selected model context fills before a final answer, the runtime makes one wrap-up request using completed observations and marks the result with a stop reason so the UI can show an honest fallback instead of treating raw runtime text as the answer. A high emergency loop guard remains backend-only to protect the desktop app from a model repeating the same tool cycle indefinitely. While the run is active, chat shows only the last one or two activity messages; the bottom Activity tab and tool-run records keep the fuller trace. When the backend returns, the assistant placeholder is replaced by the final answer body. `app/agent_runtime.go` exposes `RunAgent` and maps model-requested tools to workspace-safe filesystem, Git intelligence, dataset, SQLite, document, operations-file, web-fetch, artifact, rollback, shell, and registered tool handlers.
Prompts that clearly request a persistent workspace change now trigger the same approval dialog as the manual Writes toggle when full project access is not active. Once write access is approved, the backend will not accept a final answer for that request until a successful workspace-write tool record exists, preventing answers that claim a file or code change was created without an auditable write.

Agent provider settings must be resolved with `LLMSettingsStore.ResolveForUse` before every model call; `Get` intentionally returns redacted values for the UI. The agent bridge is split by responsibility: `app/agent_runtime.go` keeps `RunAgent`, dispatch, and read/context tools; `app/agent_runtime_mutations.go` owns high-risk write/patch/copy/move/delete/rollback handlers; `app/agent_runtime_shell.go` owns argv parsing and shell allow-list policy; and `app/agent_runtime_format.go` owns observation formatting. Model-directed appends use `workspace.ApplyFileAppend` rather than rewriting a bounded preview. `app/internal/llm/chat.go` also quotes workspace context with Nexus sentinel delimiters and escapes fence/sentinel text so selected files cannot close a Markdown context block and impersonate the user prompt.

`app/internal/agenttools/` owns tool descriptors and tool run records. Dry-runs, explicit executions, and registered-tool calls from `RunAgent` persist under `.nexusdesk/tool-runs/log.json` with inputs, output summaries, risk, approval ID, duration, and errors. The registry currently includes workspace preview/context, Git diff/changed-file/history/blame context, Problems, tasks, file mutations, file rollback list/apply, dataset profile/query/SQL, SQLite inspect/query, document-set context, artifact list/read/lineage/create/archive, read-only operations inspection, and approval-gated web fetch descriptors. The agent panel can expand recent tool runs to inspect captured inputs, output/error text, approval reference, duration, and replay/diff affordances.

`app/internal/workspace/rollback.go` owns file-mutation rollback snapshots. Before approved text writes, binary writes, appends, patches, copies, moves, or deletes run through the workspace service or agent runtime, Nexus snapshots the previous file state for each affected path under `.nexusdesk/rollbacks/{id}/` and records the rollback in `.nexusdesk/rollbacks/log.json`. Applying a rollback is high-risk: it restores backed-up bytes with their original permission bits or removes files that did not exist before the mutation. Rollbacks never target `.nexusdesk/`, directories, or symlinks, and individual snapshots are capped to avoid silently archiving huge workspace files.

`app/internal/webfetch/` owns controlled web retrieval for the agent. It accepts only HTTP(S), rejects private/loopback/link-local hosts unless local access is explicitly enabled, optionally enforces domain allow-lists, limits redirects, reads only text-like content types, caps response bytes, strips simple HTML chrome, and returns URL/status/content metadata plus bounded text. It does not execute JavaScript, persist browser state, submit forms, download binaries, or automate a browser.

`app/internal/appmeta/` owns the SQLite metadata schema, manifest, real database initialization, metadata browser, JSON-to-SQLite mirror, direct fresh-row writes, metadata search, dataset dependency records, and SQL run history under `.nexusdesk/metadata/`. `app/app_metadata.go` owns the Wails-app-level orchestration that mirrors JSON stores into that database, records artifact/dataset/SQL metadata after user actions, and converts mirrored records back into UI-facing records. `InspectMetadataStore` returns table columns, row counts, sample rows, and dataset SQL view summaries for the workbench, where users can select tables, filter columns, and copy sample rows.

`app/internal/analytics/` owns the first read-only SQL-style dataset query surface. It accepts a constrained `SELECT` subset, blocks mutation keywords, and executes through bounded table-dataset query primitives by default. A real DuckDB `database/sql` execution path is implemented behind the `duckdb` build tag for CGO-enabled machines; the current Windows verification loop keeps CGO disabled and therefore uses the safe fallback path. SQL results include plan lines: native DuckDB `EXPLAIN` when the optional driver path is available, or a deterministic fallback logical plan that lists validation, scan, projection, filter/order/limit, result counts, and native-explain availability. SQL results can be exported as Markdown artifacts that include SQL text, engine, row counts, preview rows, and source dataset citations.

`app/internal/dbconnector/` owns the first workspace SQLite connector and the first external SQL profile runners. SQLite opens `.sqlite`, `.sqlite3`, and `.db` files inside the active workspace in read-only mode, accepts bounded `SELECT`/`WITH` queries, blocks mutation-oriented SQL, and returns capped rows to the Data & Analytics connector panel. SQLite query requests carry an explicit request ID, result cap, and timeout; `app/dataset_service.go` registers request cancellation callbacks, applies the timeout context, records completed/failed SQL run metadata, and redacts connector errors before persistence. The Data & Analytics connector panel exposes the row cap, timeout, and cancel controls next to the query action. The connector package also exposes the first connector metadata model and a manual SQLite schema inspector for tables, views, columns, indexes, row counts, capped samples, declared foreign keys, and conservative `*_id` relationship hints without executing user-provided SQL. The shared frontend schema browser can select inspected SQLite or external profile objects, show tables/views, columns, indexes, capped samples, textual relationships, a compact clickable relationship map, and a grounded AI explanation action based only on inspected metadata; SQLite can additionally copy a quoted `SELECT *` query into the editor, run an explicit capped preview, save SQLite connector queries under the separate `sqlite-sql` kind, and filter SQLite query history from SQL run records. SQLite query exports use `app/internal/artifact/` to write CSV artifacts or Markdown reports with SQL, cap, timeout, preview rows, source database citation, and provenance metadata; `DatasetService` records matching SQL run and dependency rows for lineage. PostgreSQL, MySQL/MariaDB, and SQL Server profile actions are explicit settings actions today: credentials resolve only at test/inspect/query time, sessions are set read-only where supported, statement timeouts are applied, mutation SQL is rejected, and schema inspection returns tables, views, columns, indexes, declared foreign keys, capped sample rows, row estimates/counts where the provider exposes them, and first inferred relationship hints. External profile query calls now also accept request IDs and route through context-aware query functions so `CancelConnectorProfileQuery` can stop in-flight work. DuckDB profile actions follow the same explicit boundary and use a read-only `access_mode=read_only` DSN when the app is built with `-tags duckdb` and CGO; default builds keep a clear setup error so the main Windows verification loop remains compiler-free.

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

`app/internal/workspace/write.go` owns the first text and binary write approval boundaries. The frontend can draft edits for selected text files or create a new text/code file draft, preserve drafts per editor tab, request a diff preview, and only then apply the write through a rooted workspace method. Binary writes use a separate base64 path with size/SHA-256 preview metadata and no text diff. Changing a draft clears the existing diff proposal, so an apply action always corresponds to the current draft. `app/internal/workspace/delete.go` owns the first file delete boundary: selected files are backend-validated, metadata paths/directories/symlinks are rejected, and the frontend requires confirmation before removal. `app/internal/workspace/move.go` owns rename/move validation and rejects traversal, metadata targets, directories, symlinks, same-path moves, directory-like targets, and overwrites. Direct writes to `.nexusdesk/`, traversal paths, symlinks, directories, oversized text previews, oversized binary payloads, and unsafe binary text overwrites are rejected before apply.

## Goals

Nexus Augentic Studio should be easy to run, easy to test, easy to reason about as an IDE/data/analytics studio, and hard to accidentally make unsafe.

Current developer setup requires:

- Go
- Node.js or Bun for frontend development
- Wails
- optional Ollama or another LLM endpoint

Planned data/connector work will add:

- SQLite through `modernc.org/sqlite` for native metadata and future workspace database connectors
- optional DuckDB dependency behind the `duckdb` build tag and CGO
- Docker only for connector testing or packaging experiments

## Repository Shape

Current structure:

```text
nexus-app/                     Fyne-native desktop app
nexus-app/main.go              Native app entrypoint only
nexus-app/internal/app/        App lifecycle and window setup
nexus-app/internal/domain/     Framework-free domain models
nexus-app/internal/services/   UI-independent service packages
nexus-app/internal/ui/         Fyne shell, views, widgets, and theme
app-wails/                     Preserved Wails desktop app and migration reference
app-wails/internal/            Legacy backend packages to port capability by capability
app-wails/frontend/            Legacy React/TypeScript UI reference
docs/                          Product, engineering, and brand docs
docs/brand/                    Brand book, generated assets, and design tokens
services/                      Development and testing helper services
services/docker-compose.yml    Placeholder for helper service definitions
tracker.md                     Implementation tracker
```

Target Fyne internal structure as the backend grows:

```text
nexus-app/internal/app/        App lifecycle
nexus-app/internal/domain/     Domain entities and value types
nexus-app/internal/services/   Workspace, editor, git, assistant, agent, llm, artifacts, jobs, metadata, tasks, settings
nexus-app/internal/platform/   OS integration, process, secrets, filesystem adapters
nexus-app/internal/ui/         Fyne shell, panels, dialogs, widgets, theme
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
