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

- product is clearly positioned as an AI workbench, not only an IDE
- MVP modules are named
- risky actions have approval rules
- development stack is selected

## Phase 1: Workspace MVP

Goal: create a usable local desktop app with project browsing and LLM chat.

Deliverables:

- Wails desktop app shell: implemented
- frontend layout with project tree, editor area, and chat panel: implemented
- local JSON app config for recent workspaces and LLM settings: implemented
- local SQLite app database: planned
- workspace open/recent workspaces: implemented
- file tree with ignore rules: implemented
- expandable tree state across refreshes: implemented
- safe text/code file viewer: implemented
- image preview: implemented
- basic PDF preview: implemented
- lightweight syntax highlighting: implemented
- UTF-8 BOM and UTF-16 text decoding: implemented
- Monaco editor integration
- LLM settings screen: implemented
- LLM connection test for OpenAI-compatible `/models`: implemented
- LLM capability hints from provider model IDs: implemented
- streaming chat with configured OpenAI-compatible LLM URL: implemented
- chat history per workspace: implemented with local JSON config
- read selected text file into chat context: implemented

Exit criteria:

- user can open a workspace and chat with a local or remote model
- selected files can be included in context
- file access stays inside workspace root
- app runs on at least one development platform

Current status:

- The desktop shell builds on Windows through Wails.
- The workspace browser can open, refresh, preview, and remember local folders.
- Text preview stays inside the approved workspace root and refuses binary/unsafe paths.
- Common image previews render inline as capped data URLs from inside the approved workspace root.
- PDF previews render inline as capped data URLs from inside the approved workspace root.
- Recent workspaces and LLM settings persist locally.
- API keys are masked before leaving backend settings storage, but OS credential storage is still pending.
- Streaming chat works with the configured model and optional selected file context.
- Persistent chat history works through local JSON config.
- Multi-file context packaging, Monaco, richer document extraction, and SQLite persistence are still planned.

## Phase 2: Files, Documents, And Artifacts

Goal: make NexusDesk useful for real documents and generated outputs.

Deliverables:

- Markdown/text extraction
- PDF text extraction where available
- document summary tool
- artifact manager
- create Markdown report tool
- create text/code file tool with approval
- overwrite protection
- tool call timeline in chat
- approval dialog
- file diff preview for edits
- artifact browser

Exit criteria:

- AI can create a report artifact from selected source files
- user can approve or reject file writes
- generated artifacts are linked to conversations and source context

## Phase 3: Excel, CSV, And Charts

Goal: support business and marketing analysis from structured data.

Deliverables:

- Excel workbook inspector
- CSV loader
- dataset profiles
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
- chat with configurable model
- read selected context safely
- analyze Excel/CSV
- create reports and charts
- log tool calls
- require approval for writes

Everything else should support those outcomes.
