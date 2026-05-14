# Domain Model

## Main Concepts

### Workspace

A workspace is the top-level studio container for files, data sources, chats, tools, settings, and artifacts.

Examples:

- a code repository
- a marketing analysis folder
- a client reporting project
- a Docker Compose app
- a mixed business workspace with docs, spreadsheets, and dashboards

Fields:

- workspace ID
- display name
- root path
- workspace type
- created at
- updated at
- indexing status
- permissions policy
- default model profile
- artifact root

### Project

A project is an optional logical unit inside a workspace. In the UI it should behave like an IDE/data-studio scope: a folder, dataset collection, report package, or operational environment the user can work inside.

Examples:

- frontend app
- backend service
- marketing report
- traffic analysis
- Docker environment
- database investigation

Fields:

- project ID
- workspace ID
- name
- description
- root path or scope
- tags
- active conversation ID

### File Node

A file node represents a file or directory inside an approved workspace root.

Fields:

- path
- normalized path
- workspace ID
- kind: file or directory
- detected type
- size
- modified time
- content hash
- preview mode
- index status
- ignored flag

### Document

A document is an extracted representation of a file that can be read or summarized.

Examples:

- Markdown file
- PDF
- DOCX
- text file
- HTML file
- presentation
- image with OCR text

Fields:

- file node ID
- document type
- title
- language
- extracted text
- extraction status
- extraction errors
- source hash
- generated summary
- summary prompt version

### Segment

A segment is a meaningful block extracted from a document.

Examples:

- heading
- paragraph
- table
- code block
- image caption
- PDF page
- spreadsheet sheet
- log section

Fields:

- document ID
- segment index
- role
- label
- text
- page number or sheet name
- row and column range when relevant
- weight
- text hash

### Chunk

A chunk is the searchable unit used for retrieval and context building.

Fields:

- workspace ID
- document ID
- segment ID
- chunk index
- path
- language
- text
- text hash
- token estimate
- source type
- embedding vector, optional
- FTS vector, optional

### Dataset

A dataset is structured data that can be profiled, queried, summarized, and charted.

Examples:

- Excel sheet
- CSV file
- JSON table
- database query result
- GA4 export
- Search Console export
- server log table

Fields:

- dataset ID
- workspace ID
- source type
- source path or connector ID
- display name
- schema
- row count
- column count
- profile status
- DuckDB table name
- source hash

### Studio Surface

A studio surface is a durable product mode for a specific kind of work. It is not a separate app; it is a focused view over the same workspace, tool, model, and artifact system.

Examples:

- Code Studio
- Data Studio
- Analytics Studio
- Document Studio
- Operations Studio
- Artifact Studio

Fields:

- surface ID
- workspace ID
- active project or scope
- active file, dataset, connector, or artifact
- open tabs
- selected context pack
- visible tools
- last activity

### Data Profile

A data profile describes a dataset for the user and the LLM.

Fields:

- dataset ID
- column names
- column types
- null counts
- distinct counts
- numeric summaries
- date ranges
- sample rows
- detected dimensions
- detected metrics
- warnings

### Connector

A connector is a configured link to an external system.

Examples:

- PostgreSQL
- MySQL
- SQLite
- Docker Engine
- GA4
- Search Console
- Google Ads
- Meta Ads
- HTTP search provider

Fields:

- connector ID
- workspace ID
- type
- display name
- config JSON
- encrypted credentials reference
- read-only flag
- enabled flag
- last health check

### Database Connection

A database connection is a connector specialized for SQL systems.

Fields:

- connector ID
- driver
- host or path
- database name
- read-only mode
- allowed schemas
- blocked statements
- connection status

### Docker Environment

A Docker environment represents Docker state visible to NexusDesk.

Fields:

- environment ID
- connector ID
- context name
- endpoint
- detected version
- containers
- images
- compose projects
- access mode
- last refresh time

### Model Profile

A model profile defines how NexusDesk should call an LLM.

Fields:

- profile ID
- provider type
- base URL
- model name
- API key reference
- temperature
- max context tokens
- supports streaming
- supports tools
- supports vision
- supports image generation
- supports embeddings
- timeout settings

### Conversation

A conversation is a chat thread tied to a workspace or project.

Fields:

- conversation ID
- workspace ID
- project ID, optional
- title
- model profile ID
- created at
- updated at
- archived flag

### Message

A message stores user, assistant, system, or tool content.

Fields:

- message ID
- conversation ID
- role
- content
- structured content JSON
- source references
- created at
- token estimate

### Tool Definition

A tool definition describes a callable backend capability.

Fields:

- tool name
- description
- input schema
- output schema
- risk level
- approval policy
- timeout
- max output size
- enabled flag

### Tool Run

A tool run is one executed tool call.

Fields:

- tool run ID
- conversation ID
- message ID
- tool name
- input JSON
- output JSON
- status
- risk level
- approval ID
- duration
- error
- created at

### Approval Request

An approval request is a user decision point for risky actions.

Examples:

- write file
- overwrite file
- delete file
- run Docker build
- stop container
- run database mutation
- execute shell command

Fields:

- approval ID
- workspace ID
- conversation ID
- action type
- description
- input JSON
- diff or preview
- status: pending, approved, rejected, expired
- created at
- resolved at

### Artifact

An artifact is a generated or exported result.

Examples:

- Markdown report
- PDF report
- PNG chart
- SVG chart
- SQL file
- generated source file
- Dockerfile
- docker-compose.yml
- cleaned Excel export
- HTML dashboard

Fields:

- artifact ID
- workspace ID
- conversation ID
- path
- kind
- title
- source tool run IDs
- created at
- source references
- content hash

## Relationship Overview

```mermaid
erDiagram
  WORKSPACE ||--o{ PROJECT : contains
  WORKSPACE ||--o{ FILE_NODE : owns
  WORKSPACE ||--o{ CONVERSATION : has
  WORKSPACE ||--o{ CONNECTOR : configures
  WORKSPACE ||--o{ DATASET : contains
  WORKSPACE ||--o{ ARTIFACT : creates
  WORKSPACE ||--o{ STUDIO_SURFACE : presents

  FILE_NODE ||--o| DOCUMENT : extracts_to
  DOCUMENT ||--o{ SEGMENT : has
  SEGMENT ||--o{ CHUNK : splits_into

  DATASET ||--o{ DATA_PROFILE : describes

  CONNECTOR ||--o| DATABASE_CONNECTION : may_be
  CONNECTOR ||--o| DOCKER_ENVIRONMENT : may_be

  CONVERSATION ||--o{ MESSAGE : contains
  MESSAGE ||--o{ TOOL_RUN : triggers
  TOOL_DEFINITION ||--o{ TOOL_RUN : implements
  TOOL_RUN ||--o| APPROVAL_REQUEST : may_require
  TOOL_RUN ||--o{ ARTIFACT : may_create

  MODEL_PROFILE ||--o{ CONVERSATION : used_by
```

## Design Rule

Store original source content and generated AI content separately.

Original files, extracted text, spreadsheet data, database results, and Docker logs should remain auditable. Summaries, insights, chart specs, generated reports, and model answers can be regenerated when prompts, models, or source data change.
