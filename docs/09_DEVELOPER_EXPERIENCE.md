# Developer Experience

## Current Verification Loop

On the current Windows workstation, use this loop after backend, frontend, binding, or asset changes:

```powershell
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
go test ./...
npm.cmd run build
wails build
```

Run Go commands from `app/`, frontend commands from `app/frontend/`, and Wails build commands from `app/`.

When Wails regenerates frontend bindings, `app/frontend/wailsjs/go/models.ts` can pick up whitespace-only changes. Clean those before committing if `git diff --check` reports trailing whitespace.

## Goals

NexusDesk should be easy to run, easy to test, easy to reason about, and hard to accidentally make unsafe.

Developer setup should require:

- Go
- Node.js or Bun for frontend development
- Wails
- SQLite
- DuckDB dependency
- optional Ollama or another LLM endpoint
- Docker only for connector testing or packaging experiments

## Repository Shape

Current structure:

```text
app/                           Wails desktop app
app/app.go                     Go application state and frontend bindings
app/main.go                    Wails entrypoint
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

This keeps NexusDesk maintainable as it grows from a local prototype into a serious workbench.
