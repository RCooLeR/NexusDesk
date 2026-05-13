# Product Vision

## One-Line Vision

NexusDesk is the local AI workbench that helps developers, analysts, marketers, operators, and business teams understand their files, data, documents, code, and systems from one safe desktop workspace.

## Problem

Modern work is scattered across many formats and tools:

- source code lives in project folders and repositories
- marketing traffic data lives in GA4, Search Console, ads platforms, CSV exports, and spreadsheets
- business analysis lives in Excel, PDFs, presentations, dashboards, and documents
- databases require separate clients and SQL knowledge
- Docker and local services require terminal commands and logs
- AI tools usually work outside the user’s real workspace

Users need more than a chatbot. They need an AI assistant that can see the workspace structure, understand the right files, inspect data, generate reports, create charts, explain code, and interact with tools safely.

A general chatbot struggles because it cannot reliably access local files, cannot inspect a spreadsheet without a tool, and often loses the connection between answer and source. A normal IDE struggles because it is optimized for code, not business analysis, marketing data, PDFs, images, databases, and Docker.

NexusDesk solves this by combining a desktop workspace, local file intelligence, data analysis tools, configurable LLM providers, and a permissioned agent loop.

## Target Users

- Developers who want a local Codex-like workspace with file context, code understanding, and safe artifact creation.
- Data analysts who need to inspect Excel, CSV, JSON, logs, and database results.
- Marketers who need traffic, campaign, SEO, landing page, and funnel analysis.
- Founders and operators who want reports and dashboards without switching between many tools.
- DevOps users who need help with Dockerfiles, Compose files, containers, logs, and local services.
- Teams that want AI assistance without sending all workspace data to one fixed provider.
- Power users who want to connect custom LLM URLs, local models, and internal model gateways.

## Product Experience

NexusDesk should feel like a confident workspace analyst:

- The project tree is visible.
- Files open quickly in the right viewer.
- Code is highlighted and editable.
- Images and PDFs preview without leaving the app.
- Spreadsheets and datasets can be profiled, queried, charted, and exported.
- Chat is tied to the current workspace and selected files.
- The AI can ask to use tools, but the app controls permissions.
- Tool calls are visible in the chat timeline.
- Generated outputs become artifacts in the workspace.
- Risky actions require approval and show a preview or diff.

## Main Use Cases

### Code And IDE Assistance

- explain a file or project structure
- find bugs from selected files
- generate code files
- propose patches with diff preview
- create README, tests, Dockerfiles, and Compose files
- summarize dependencies and architecture

### Document Analysis

- summarize PDFs, DOCX files, Markdown files, and text files
- compare two documents
- extract action items, risks, decisions, dates, and entities
- create reports from multiple source documents

### Excel And Data Analysis

- inspect workbook sheets, headers, row counts, and formulas
- profile numeric, text, and date columns
- answer questions from spreadsheets
- create pivot-style summaries
- generate charts
- export cleaned or summarized datasets

### Marketing And Traffic Analytics

- analyze campaign exports
- compare traffic sources
- review SEO data
- inspect landing page screenshots
- summarize funnel performance
- create client-ready or internal reports

### Database Work

- connect to approved databases
- list schemas and tables
- describe table structure
- run read-only queries
- turn query results into charts and explanations
- export query output

### Docker And Operations

- inspect containers, images, volumes, and networks
- read logs
- explain Dockerfiles and Compose files
- generate Compose files
- suggest debugging steps
- perform start/stop/build actions only with approval

## What Makes NexusDesk Different

1. Workspace-native AI

   The app is built around real projects, files, datasets, chats, and generated artifacts. The workspace is the product surface, not just a prompt box.

2. Local-first control

   The user chooses the model endpoint. Ollama, Docker Model Runner, OpenAI-compatible APIs, internal gateways, and future providers can all fit behind one LLM interface.

3. Tool-mediated safety

   The model cannot directly read, write, delete, query, or run anything. It requests tools. NexusDesk validates the request, applies policy, shows approvals when needed, and logs the result.

4. Multi-domain workbench

   The app is useful for code, documents, Excel, images, marketing data, databases, Docker, and operations. It is broader than an IDE and more structured than a chatbot.

5. Artifact creation

   NexusDesk should create useful outputs: reports, charts, dashboards, generated files, SQL, code, Docker configs, and exports.

6. Explainable analysis

   The user should see which files, rows, sheets, queries, logs, or connectors were used to produce an answer.

## Success Criteria

- Users can open a workspace and start chatting with an LLM in minutes.
- The LLM URL, model, API key, and capabilities are configurable.
- File reading and dataset analysis work without sending the entire workspace blindly to the model.
- The app can analyze common Excel and CSV files and produce charts.
- Text, code, images, PDFs, and spreadsheets open in appropriate viewers.
- Generated files are saved as artifacts and visible in the UI.
- Risky actions require approval and are logged.
- Docker and database tools default to read-only or inspect-only behavior.
- A new developer can run the app locally from a clean checkout.
- The product remains modular enough to add MCP, team workspaces, and plugin systems later.
