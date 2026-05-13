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
- LLM profiles
- model capability flags
- tool enablement
- tool risk policies
- indexing rules
- ignored paths
- artifact paths
- connector definitions
- UI preferences
- feature flags

Runtime changes should be stored locally and optionally exportable as workspace config.

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
- chat/tool log cleanup
- artifact cleanup

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
