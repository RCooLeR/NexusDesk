# Delivery Plan

## Phase 0: Product Baseline

Goal: lock the product direction and preserve the strongest idea.

Deliverables:

- NexusDesk product docs
- brand package
- UI wireframe
- core workflow definition
- LLM provider settings design
- tool risk model
- MVP scope guardrails

Exit criteria:

- product is clearly positioned as a local-first AI IDE, data studio, and analytics studio, not a prompt-only chatbot
- MVP modules are named
- risky actions have approval rules
- development stack is selected

## Phase 1: Workspace MVP

Goal: create a usable local desktop studio with project browsing, editor tabs, data panels, artifacts, and grounded LLM assistance.

Deliverables:

- Wails desktop app shell: implemented
- frontend layout with project tree, editor area, and chat panel: implemented
- IDE/data/analytics studio positioning in app copy and docs: implemented
- visible studio surface vocabulary for code, data, documents, operations, and artifacts: first implementation
- local JSON app config for recent workspaces and LLM settings: implemented
- OS-protected API key credential storage: implemented
- local SQLite app database: planned
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
- NexusDesk is now documented and presented as a local-first AI IDE, data studio, and analytics studio.
- The active center pane exposes a first studio surface indicator for Code Studio, Data Studio, Document Studio, Operations Studio, Artifact Studio, or Workspace Studio.
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
- CSV datasets can generate first SVG bar chart artifacts from category counts or numeric sums.
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
- richer document extraction/OCR and SQLite persistence are still planned.
- Markdown report artifacts can be created under `.nexusdesk/artifacts/` without overwriting existing files.
- Latest assistant answers can be saved as Markdown artifacts under `.nexusdesk/artifacts/` with their chat context recorded as metadata.
- Markdown artifacts now write sidecar provenance metadata with source, prompt, model, source paths, and creation timestamp.
- SVG chart artifacts now write sidecar provenance metadata with dataset source paths and chart configuration.
- The workbench lists generated Markdown artifacts and can reselect visible report files from that list.
- The frontend has a smoke check for the built entrypoint, generated Wails bindings, and core shell functionality markers.
- richer document extraction/OCR, richer approval dialogs, DuckDB SQL, and SQLite persistence are still planned.

## Phase 2: Files, Documents, And Artifacts

Goal: make NexusDesk useful for real documents and generated outputs.

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

Goal: connect NexusDesk to real business data sources.

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

Goal: make NexusDesk useful for Docker-based development and operations.

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
- core MVP remains fast and stable

## MVP Scope Guardrails

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
