# Operations And Security

## Environments

Recommended environments:

- local developer
- packaged desktop app
- internal demo
- private beta
- team or enterprise pilot
- production release

Each environment should have separate:

- configuration
- local app database
- model profiles
- connector credentials
- logs
- feature flags
- update channel

## Configuration

Configuration should be explicit and exportable:

- workspaces
- active studio surface
- LLM profiles
- model capability flags
- tool enablement
- tool risk policies
- indexing rules
- ignored paths
- artifact paths
- connector definitions
- UI preferences
- editor, data, analytics, document, operations, and artifact surface preferences
- feature flags

Runtime changes should be stored locally and optionally exportable as workspace config.

Studio surfaces can change which tools and panels are visible, but they must not change the underlying safety boundary. Code Studio, Data Studio, Analytics Studio, Document Studio, Operations Studio, and Artifact Studio all share the same workspace roots, path checks, approval rules, secret handling, and audit model.

The current implementation has an append-only local approval/action log for applied file writes, deletes, moves, artifact creation, scan-report creation, artifact archive, and artifact delete. It writes records under `.nexusdesk/approvals/log.json` and shows the latest entries in the workbench. Modal approval prompts now cover higher-risk file and artifact actions; Docker, database, and model-directed agent tool execution remain planned.

## Secrets

NexusDesk may store:

- LLM API keys
- search API keys
- database credentials
- marketing connector tokens
- Docker endpoint settings
- custom gateway credentials

Rules:

- never store secrets in plain workspace files by default
- use OS keychain or encrypted local storage when available
- never include secrets in model prompts
- redact secrets in logs
- mark suspected secret files as restricted
- require confirmation before sending sensitive content to remote models

## File System Security

Default rules:

- only access selected workspace roots
- block path traversal
- deny access to parent directories unless added as roots
- ignore system folders by default
- show file writes before applying
- show diffs for edits
- require approval for overwrite and delete
- cap file read size
- cap artifact output size

IDE-style convenience must remain scoped. Quick-open, editor tabs, project context packs, source previews, and artifact navigation should all resolve through the same rooted workspace APIs instead of reading arbitrary filesystem paths from the frontend.

Current implementation:

- workspace scans and previews are rooted in `app/internal/workspace/`
- create/update writes require a backend diff preview before apply
- deletes reject directories, symlinks, metadata paths, and traversal before frontend confirmation
- rename/move rejects traversal, metadata paths, directories, symlinks, same-path moves, directory-like targets, and overwrites
- direct `.nexusdesk/` metadata writes and deletes are blocked
- CSV query exports are created only from bounded query results and exclusive artifact writes
- chart artifacts are created only through bounded CSV aggregation and exclusive artifact writes
- scan-report artifacts are created from backend scan status and exclusive artifact writes
- artifact archive/delete actions validate workspace-relative artifact paths and move/remove sibling metadata sidecars through backend methods
- frontend commands call Wails bindings rather than reading or mutating arbitrary paths directly

## Database Security

Default database mode should be read-only.

Rules:

- list schemas and tables safely
- show generated SQL before execution in debug or approval mode
- block `DROP`, `DELETE`, `UPDATE`, `INSERT`, `ALTER`, `TRUNCATE`, and similar statements by default
- allow mutations only through explicit policy and approval
- cap result rows
- log query text and timing
- redact credentials in error messages

Data Studio and Analytics Studio should make read-only status visible near schema, query, chart, and report surfaces. Mutating SQL is a policy change, not a UI shortcut.

## Docker Security

Docker access is powerful and should be treated as high risk.

Low-risk actions:

- list containers
- list images
- inspect container
- read logs
- explain Dockerfile
- explain Compose file

High-risk actions:

- start container
- stop container
- remove container
- remove image
- build image
- run container
- change volume or network
- execute command in container

High-risk Docker actions require approval. The UI should show the exact planned action and affected resources.

Operations Studio can surface Docker state, logs, Compose files, and generated configs, but it should keep start, stop, build, run, exec, volume, and network actions behind the same high-risk approval flow. The current implementation parses selected Compose files into service names, images, ports, volumes, and dependencies without calling Docker or mutating local state.

## Network Security

Network tools should be controlled.

Rules:

- HTTP fetch tools use timeouts
- cap response size
- block local/private network access by default for remote prompts when needed
- allow search only through configured providers
- do not scrape search engines directly
- log outgoing tool requests
- let users disable network tools per workspace

## Privacy

NexusDesk should assume user workspaces may contain private data.

Protect:

- file paths
- document text
- spreadsheet data
- database results
- Docker logs
- marketing data
- chat history
- tool outputs

Privacy controls:

- local-only mode
- remote-model warning
- restricted files
- secret redaction
- per-workspace send-to-model policy
- per-surface context visibility
- chat/tool log cleanup
- artifact cleanup

Switching from one studio surface to another should not silently expand what gets sent to a model. The context pack preview remains the user's confirmation point for project, directory, document, dataset, and artifact context.

## Observability

Track locally:

- app start errors
- indexing runs
- files indexed/skipped/failed
- document extraction errors
- dataset profiling time
- search latency
- model latency
- tool run count
- approval count
- failed tool calls
- artifact creation count
- Docker connector status
- database connector status

For team or enterprise mode, metrics should be configurable and privacy-preserving.

## Backups

Back up:

- SQLite app database
- workspace metadata
- chats
- tool logs
- settings, excluding raw secrets unless encrypted
- artifact metadata
- custom prompts and policies

Workspace files themselves should not be silently backed up by NexusDesk unless the user explicitly opts in.

## Failure Modes

NexusDesk should degrade gracefully:

- if LLM is down, file browsing and previews still work
- if indexing fails, the workspace still opens
- if embeddings are unavailable, lexical search still works
- if PDF extraction fails, PDF preview still works
- if spreadsheet profiling fails, raw table preview can still work
- if a studio surface fails to load, the rest of the shell stays usable
- if Docker is unavailable, Docker panel shows connection status
- if database connection fails, saved config remains but queries are disabled
- if chart export fails, text summary still remains

## Release Discipline

Every release should include:

- migration review
- model prompt version review
- tool policy review
- risky action smoke tests
- file path traversal tests
- database safety tests
- Docker safety tests
- local data privacy review
- packaging test on Windows, macOS, and Linux
- rollback or downgrade notes
