# Nexus Augentic Studio

Nexus Augentic Studio is a local-first AI IDE, data studio, and analytics studio for code, documents, datasets, marketing analytics, databases, Docker, and operations.

The goal is to give users one desktop studio where they can open a project or business workspace, inspect and edit files, analyze spreadsheets and documents, connect to data sources, generate reports and charts, and safely create or modify artifacts with AI assistance.

Nexus Augentic Studio is not only a chatbot. It should feel like a serious IDE-style studio with AI built into the project, data, analytics, document, and operations surfaces:

- browse and understand project folders
- open text, code, images, PDFs, spreadsheets, and common document formats
- connect to local or remote LLM endpoints as an integrated assistant layer
- analyze Excel, CSV, logs, traffic exports, marketing data, and database results
- create charts, reports, dashboards, code, SQL, Dockerfiles, and Compose files
- inspect Docker containers, images, logs, and compose projects
- keep AI actions visible, permissioned, and auditable

## Documentation

- [Product Vision](docs/01_PRODUCT_VISION.md)
- [Architecture](docs/02_ARCHITECTURE.md)
- [Domain Model](docs/03_DOMAIN_MODEL.md)
- [Workspace And Indexing](docs/04_WORKSPACE_AND_INDEXING.md)
- [Search, Context, And Ranking](docs/05_SEARCH_CONTEXT_AND_RANKING.md)
- [AI Agent And LLM Strategy](docs/06_AI_AGENT_AND_LLM_STRATEGY.md)
- [Operations And Security](docs/07_OPERATIONS_AND_SECURITY.md)
- [Delivery Plan](docs/08_DELIVERY_PLAN.md)
- [Developer Experience](docs/09_DEVELOPER_EXPERIENCE.md)
- [Studio Roadmap](docs/10_STUDIO_ROADMAP.md)
- [Implementation Tracker](tracker.md)

## Current Project Layout

The repository has its first runnable Wails app scaffold. The directories that exist now are:

```text
app/       Wails desktop app with Go backend and React/TypeScript frontend.
docs/      Product, engineering, and brand documentation.
services/  Development and testing helper services.
```

The fuller backend module, storage, indexing, connector, and tool layout is tracked in the developer experience doc as the target implementation shape.

## Core Principles

- Local-first: user files, chats, tool logs, and generated artifacts should live locally by default.
- Provider-agnostic: users should configure an LLM base URL, model, API key, and capabilities.
- Tool-mediated: the LLM requests actions; the Go backend validates and performs them.
- Source-grounded: analysis should cite files, sheets, rows, queries, logs, or tool outputs used.
- Permissioned: writes, deletes, Docker mutations, database mutations, and shell execution require approval.
- Multimodal: text, code, spreadsheets, PDFs, images, charts, and database results are first-class.
- Artifact-first: useful outputs should become real files, not just chat messages.
- Explainable: every tool call, search result, generated report, and file change should be inspectable.
- Modular: workspace, parsing, indexing, agent, LLM gateway, connectors, and UI are separate modules.
- Extensible: native tools come first; MCP and external plugin systems can be added later.

## Product Shape

Nexus Augentic Studio should feel like a unified studio:

```text
Project tree + editor tabs + data tables + analytics panels + Docker views + AI assistant
```

A typical workflow:

```text
Open workspace
  ->
Nexus Augentic Studio indexes files, documents, datasets, and metadata
  ->
User asks a question in chat
  ->
Agent searches relevant context
  ->
Agent requests tools when needed
  ->
Backend runs approved tools
  ->
Agent returns grounded answer
  ->
App creates artifacts such as reports, charts, files, or configs
```

## First Stable Focus

The first useful version should focus on:

- opening local workspaces
- configuring an LLM URL
- file tree, quick-open, editor tabs, find-in-file, and studio modes
- chat per workspace
- safe read-only tools
- Excel/CSV analysis
- PDF and image preview
- chart generation
- report artifacts
- simple Docker inspection

Avoid building everything at once. A reliable studio with a few strong tools is better than a broad agent that cannot be trusted.
