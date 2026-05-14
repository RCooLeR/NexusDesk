# Developer Experience

## Current Verification Loop

On the current Windows workstation, use this loop after backend, frontend, binding, or asset changes:

```powershell
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
go test ./...
npm.cmd run build
npm.cmd run smoke
wails build
```

Run Go commands from `app/`, frontend commands from `app/frontend/`, and Wails build commands from `app/`.

When Wails regenerates frontend bindings, `app/frontend/wailsjs/go/models.ts` can pick up whitespace-only changes. Clean those before committing if `git diff --check` reports trailing whitespace.

## Current Local Persistence

The current app uses small JSON files in the user's config directory while SQLite is still pending:

- `recent-workspaces.json`
- `llm-settings.json`
- `chat-history.json`

LLM API keys are not written into `llm-settings.json`. They are saved in a sidecar credential blob protected by the OS where available, while the JSON settings file keeps only a storage marker. These stores live behind `app/internal/storage/` so the later SQLite migration can keep the same app-level boundaries.

## Chat Streaming

`AskLLMStream` emits `nexusdesk:chat-stream` Wails events while `app/internal/llm/chat.go` reads OpenAI-compatible server-sent response chunks. The frontend listens in `NexusDeskShell.tsx`, updates the in-flight assistant message per `delta`, then replaces it with the final persisted response or refreshed workspace chat history when the request completes.

Selected directories and the workspace root also flow through the same streaming path. `app/internal/workspace/context.go` expands a selected directory or `.` into a capped set of previewable files, then `app/app.go` builds a context pack with a small manifest and file sections. The current caps are 32 files and 96 KiB of packed context, with the same ignored-folder, symlink, path traversal, encoding, PDF text, DOCX text, and CSV-summary boundaries used by file previews.

The chat panel previews pinned context packs by calling `PreviewChatContextPack`, which uses the same backend collector as the send path. That keeps the visible file list aligned with what the model will actually receive, including truncation warnings when caps are reached.

Chat history stores the source paths attached to each user/assistant pair so saved answer artifacts can use the answer's original context instead of whatever happens to be pinned later.

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

`app/internal/workspace/preview.go` keeps text previews rooted and size-limited, decodes UTF-8, UTF-16, and Windows-1251 text variants, parses CSV files into bounded table previews with lightweight column profiles from a larger capped sample, and renders common image/PDF files as capped data URLs for inline display. PDFs also expose simple embedded text extraction by page when available, and DOCX files expose basic body text extraction. Chat context accepts text previews, DOCX text, extracted PDF text, and structured CSV profiles plus bounded samples, so binary payloads and data URLs are not sent to the model as source text.

`app/internal/workspace/search.go` owns the first workspace path/content search pass. It searches path names and previewable text content inside the same ignore and depth boundaries as scanning. `SearchWorkspace` now merges that result set with artifact metadata matches from `app/internal/artifact/` and persisted chat snippets from `app/internal/storage/chat_history.go`, so generated outputs and prior analysis are searchable from the same navigator surface. `app/internal/workspace/dataset_query.go` owns the first CSV query flow with bounded row results and simple `column=value` filters until a DuckDB SQL layer is added. Dataset query exports rerun that same bounded query before writing a CSV artifact, so exported rows match the backend safety boundary rather than trusting frontend table state.

`app/internal/workspace/context.go` owns directory/project context expansion and context-pack previews. The UI can pin a selected directory or the workspace root, but the backend still decides which files are safe and useful enough to include.

`app/frontend/src/features/shell/HighlightedCode.tsx` remains as the dependency-free fallback highlighter for non-Monaco preview paths. Text/code source previews and edit drafts now use the Monaco-backed components listed below.

## Dataset Profiles

`app/internal/dataset/` owns the first persistent dataset profile pass and saved query history. CSV files reuse the workspace preview profiles and XLSX files expose workbook sheet names, then profiles are stored under `.nexusdesk/datasets/profiles.json` inside the active workspace. Saved CSV queries are stored under `.nexusdesk/datasets/queries.json` and capped per dataset. `app/internal/workspace/chart.go` owns the first CSV chart model: one category column, optional numeric value column, bar or line chart mode, bounded points, and no arbitrary SQL or model-rendered pixels.

The workbench topbar now has functional Preview, Explain, Summarize, Edit, Report, and Profile actions. Preview reloads the selected workspace node from disk, Explain sends a predefined grounded prompt when text context is available, Summarize sends selected file/directory context through chat and saves the result as a Markdown artifact, Edit uses the diff/apply write flow, Report creates a Markdown artifact, and Profile persists CSV/XLSX dataset metadata. The dataset panel can run a bounded CSV row query for the selected table, save/reuse queries, export the bounded result as a CSV artifact, preview chart points, create deterministic SVG bar or line chart artifacts, and create deterministic Markdown dataset summaries. The topbar also shows the active studio surface so code, data, document, operations, artifact, and workspace contexts are explicit. Editor previews and drafts now use Monaco with language detection for common code, document, data, and operations files. Drafts show dirty state, persist per tab while navigating, clear stale diff previews after edits, support revert before apply, guard dirty tab close, and use Ctrl+S to preview/apply through the same write path. New files start as draft tabs from Ctrl+N or the command palette, then use the same preview/apply boundary to create the file. Editor keyboard shortcuts include Ctrl+F for in-file find, Ctrl+W for active-tab close, and Ctrl+Tab / Ctrl+Shift+Tab for tab cycling. Ctrl+Shift+P opens the command palette for common workspace, editor, context, data, artifact, and chat actions.

## Frontend Structure

The shell is now mostly orchestration. Feature panels own stable presentation, while `NexusDeskShell.tsx` keeps workspace, preview, provider, and chat state wiring close to the Wails bindings:

- `app/frontend/src/components/ui.tsx` contains reusable UI atoms such as buttons, cards, status badges, and branded state panels.
- `app/frontend/src/features/shell/NexusDeskShell.tsx` owns the composed desktop workbench state, global quick-open/command-palette shortcuts, and cross-panel navigation wiring.
- `app/frontend/src/features/shell/QuickOpenPalette.tsx` owns the keyboard quick-open palette for workspace nodes and open editor tabs.
- `app/frontend/src/features/shell/CommandPalette.tsx` owns the keyboard command palette for workspace, editor, assistant, data, and artifact actions.
- `app/frontend/src/features/shell/MonacoFileEditor.tsx` owns the lazy-loaded Monaco edit surface, worker wiring, language detection, and editor-local Ctrl+S forwarding for draft writes.
- `app/frontend/src/features/shell/MonacoCodePreview.tsx` owns read-only Monaco previews and search decorations for source files.
- `app/frontend/src/features/shell/monacoRuntime.ts` owns shared Monaco lazy-loading, worker setup, theme definition, and file language detection.
- `app/frontend/src/features/shell/AgentChatCard.tsx` owns the expanded chat presentation, full conversation scroll area, multiline prompt composer, context pack list, save-answer action surface, and delegates provider calls/history/artifact actions back to the shell.
- `app/frontend/src/features/shell/ChatMessageContent.tsx` renders safe dependency-free Markdown-style chat content, including headings, lists, tables, code fences, inline code, and bold text.
- `app/frontend/src/features/shell/LLMSettingsCard.tsx` owns the provider settings form and delegates persistence/probe actions back to the shell.
- `app/frontend/src/features/shell/ToolTimeline.tsx` owns the visible tool event timeline presentation.
- `app/frontend/src/features/shell/WorkspaceNavigator.tsx` owns the workspace lockup, search controls, recent workspace list, fallback scaffold list, and indexed workspace tree presentation, with aligned rows inside the resizable sidebar. `NexusDeskShell.tsx` owns the resizable navigator width state.
- `app/frontend/src/features/shell/WorkbenchPanel.tsx` owns the active context topbar, active studio surface indicator, closeable editor tab strip, source preview/editor presentation, find-in-file, Markdown source/rendered switching, dataset query/chart panels, artifact metadata panel, first approval log, fallback workflow preview, and capability cards.
- `app/frontend/src/features/shell/WorkspaceRail.tsx` owns the compact branded rail and mode icons.
- `app/frontend/src/features/shell/AgentPanel.tsx` composes the grounded assistant header, chat card, provider settings, and tool timeline.

`App.css` keeps the desktop shell fixed to the window and pushes overflow into the interactive surfaces that actually need it: workspace tree/search results, quick-open and command-palette results, source preview, dataset query results, capability list, chat thread, provider settings, and tool timeline.

## Frontend Smoke Checks

`app/frontend/scripts/smoke.mjs` checks that the built frontend and key shell source files still expose the main MVP functionality: Wails bindings, search, quick-open, command palette, Monaco preview/edit surfaces, find-in-file, context packs, file create/update/delete/move flows, dataset profiling/querying/saved queries/exporting/charting/summaries, artifact metadata, approval log styling, resizable navigator styling, and the production `dist/index.html` entrypoint. Run it after `npm.cmd run build`.

## Artifact Creation

`app/internal/artifact/` owns deterministic artifact writes, provenance sidecars, metadata lookup, artifact search, and listing. The first flows create timestamped Markdown reports from selected previews, timestamped Markdown artifacts from assistant answers, timestamped CSV exports from dataset queries, timestamped SVG chart artifacts from CSV chart models, and timestamped Markdown dataset summaries under `.nexusdesk/artifacts/`, use exclusive file creation to avoid overwrites, and return the new workspace-relative path so the UI can refresh and select it. Each artifact also gets a sibling `.meta.json` file with kind, source, source paths, prompt/configuration, model when relevant, context path, and creation timestamp when available. Saved assistant answers preserve the model's Markdown and include source/context metadata before the generated body. The workbench lists Markdown, CSV, and SVG artifacts from that folder so generated outputs remain visible after creation, and it shows metadata for the active generated artifact.

## Approval Log

`app/internal/approval/` owns the first append-only action log. Applied text writes, deletes, moves, reports, saved chat artifacts, chart artifacts, query exports, and dataset summaries append records under `.nexusdesk/approvals/log.json`. This is not the final modal policy engine yet, but it gives the studio an auditable local trail while higher-risk approval dialogs are designed.

## File Writes

`app/internal/workspace/write.go` owns the first text write approval boundary. The frontend can draft edits for selected text files or create a new text/code file draft, preserve drafts per editor tab, request a diff preview, and only then apply the write through a rooted workspace method. Changing a draft clears the existing diff proposal, so an apply action always corresponds to the current draft. `app/internal/workspace/delete.go` owns the first file delete boundary: selected files are backend-validated, metadata paths/directories/symlinks are rejected, and the frontend requires confirmation before removal. `app/internal/workspace/move.go` owns rename/move validation and rejects traversal, metadata targets, directories, symlinks, same-path moves, directory-like targets, and overwrites. Direct writes to `.nexusdesk/`, traversal paths, symlinks, directories, oversized previews, and binary existing files are rejected before apply.

## Goals

NexusDesk should be easy to run, easy to test, easy to reason about as an IDE/data/analytics studio, and hard to accidentally make unsafe.

Current developer setup requires:

- Go
- Node.js or Bun for frontend development
- Wails
- optional Ollama or another LLM endpoint

Planned data/connector work will add:

- SQLite
- DuckDB dependency
- Docker only for connector testing or packaging experiments

## Repository Shape

Current structure:

```text
app/                           Wails desktop app
app/app.go                     Go application state and frontend bindings
app/main.go                    Wails entrypoint
app/internal/artifact/         Markdown artifact creation, listing, provenance
app/internal/dataset/          CSV/XLSX profile persistence
app/internal/llm/              OpenAI-compatible probe, chat, streaming
app/internal/storage/          JSON stores for recent workspaces, LLM settings, chat history
app/internal/workspace/        Safe scanning, preview, search, context packs, file operations
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
go run ./cmd/nexusdesk migrate
go run ./cmd/nexusdesk index --workspace ./examples/workspace
go run ./cmd/nexusdesk eval --suite ./examples/eval/basic.yaml
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

This keeps NexusDesk maintainable as it grows from a local prototype into a serious desktop studio.
