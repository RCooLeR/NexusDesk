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

- Wails desktop app shell
- frontend layout with project tree, editor area, and chat panel
- local SQLite app database
- workspace open/recent workspaces
- file tree with ignore rules
- text/code file viewer
- Monaco editor integration
- image preview
- basic PDF preview
- LLM settings screen
- chat with configured LLM URL
- chat history per workspace
- read selected file into chat context

Exit criteria:

- user can open a workspace and chat with a local or remote model
- selected files can be included in context
- file access stays inside workspace root
- app runs on at least one development platform

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
