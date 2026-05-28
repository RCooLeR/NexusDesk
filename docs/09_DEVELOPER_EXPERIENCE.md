# Developer Experience

Primary development now happens in the Fyne-native `nexus-app/`. The Wails/React implementation is preserved under `app-wails/` as a reference and migration source, not as the active app.

## Current Verification Loop

Focused native tests:

```powershell
cd nexus-app
$env:GOFLAGS='-mod=readonly'
go test ./internal/domain ./internal/services/... ./internal/ui/shell ./internal/ui/theme ./internal/brand
```

Full native tests/build on Windows require CGO and a C compiler:

```powershell
cd nexus-app
.\scripts\dev-env.ps1 -Test
.\scripts\dev-env.ps1 -Build
.\scripts\dev-env.ps1 -Run
```

Current workstation status:

- MSYS2 UCRT64 GCC is installed under `C:\msys64\ucrt64\bin`.
- `nexus-app/scripts/dev-env.ps1` configures `PATH`, `CGO_ENABLED=1`, readonly module flags, and local Go cache/temp paths.
- `.\scripts\dev-env.ps1 -Build` stamps the approved brand icon into `resource_windows.syso`, injects version/commit/build-date metadata through Go ldflags, and writes `build\nexusdesk.exe`.
- `CGO_ENABLED=0 go build .` is not expected to work because Fyne's OpenGL binding requires CGO-backed files.

Legacy Wails reference checks, only when deliberately touching `app-wails/`:

```powershell
cd app-wails
$env:NODE_OPTIONS='--use-system-ca --dns-result-order=ipv4first'
go test ./...
cd frontend
npm.cmd run build
npm.cmd run smoke
npm.cmd run smoke:visual
cd ..
wails build
```

## Repository Shape

```text
nexus-app/                     Fyne-native desktop app
nexus-app/main.go              Native app entrypoint only
nexus-app/internal/app/        App lifecycle and window setup
nexus-app/internal/domain/     Framework-free domain models
nexus-app/internal/services/   UI-independent service packages
nexus-app/internal/ui/         Fyne shell, views, widgets, and theme
app-wails/                     Preserved Wails desktop app and migration reference
app-wails/internal/            Legacy backend packages to port capability by capability
app-wails/frontend/            Legacy React/TypeScript UI reference
docs/                          Product, engineering, and brand docs
docs/brand/                    Brand book, generated assets, and design tokens
services/                      Development and testing helper services
tracker.md                     Implementation tracker
```

Target native growth:

```text
nexus-app/internal/app/        App lifecycle
nexus-app/internal/domain/     Domain entities and value types
nexus-app/internal/services/   Workspace, editor, git, assistant, agent, llm, artifacts, jobs, metadata, tasks, settings, data, documents, operations
nexus-app/internal/platform/   OS integration, process, secrets, filesystem adapters, when needed
nexus-app/internal/ui/         Fyne shell, panels, dialogs, widgets, theme
```

Do not document future directories as existing until they are created.

## Architecture Rules

- Keep `main.go` thin.
- Keep framework-free domain types in `internal/domain`.
- Keep business rules, path safety, query safety, approvals, rollbacks, metadata, and file/database operations in `internal/services`.
- Keep Fyne widgets, layouts, dialogs, menus, and keyboard shortcuts in `internal/ui`.
- UI code may dispatch service use cases, but it must not reimplement workspace-root checks, SQL guards, approval policy, rollback behavior, or artifact metadata rules.
- Long work must become a job before it is attached to UI actions.
- Workspace open must remain cheap: no Git, Docker, OCR, connector pulls, dump imports, model calls, shell commands, or deep indexing on folder open.

## Native Runtime State

The native app uses workspace-local `.nexusdesk/` state:

- `.nexusdesk/metadata/nexusdesk.sqlite` for chats, approvals, artifacts, tool runs, jobs, task runs, SQL runs, dataset dependencies, and agent audit rows where repositories exist.
- `.nexusdesk/artifacts/` for generated reports, task outputs, charts, notebooks, document extracts, comparisons, and runbooks.
- `.nexusdesk/rollbacks/` for rollback snapshots created by safe workspace mutations.
- `.nexusdesk/datasets/` for saved dataset queries and SQL notebooks.
- JSON sidecars remain for compatibility imports and for human-readable artifact metadata.

Secret storage is now a shared native service. Provider API keys and connector credentials use Windows DPAPI, macOS Keychain through `security`, or Linux Secret Service/libsecret through `secret-tool`; unsupported platforms must refuse secret saves rather than writing raw credentials.

## Native Feature Boundaries

Workspace:

- `internal/services/workspace` owns scanning, listing, preview, encoding detection, context packs, search, problems, safe writes, patches, file operations, and rollbacks.
- `internal/services/editor` owns tab identity, dirty state, pinned ordering, and close guards.

Assistant and agent:

- `internal/services/llm` owns provider transport, streaming, probes, Ollama diagnostics, context windows, and response reserves.
- `internal/services/assistant` owns Ask-mode request preparation.
- `internal/services/agent` owns Agent-mode loop behavior.
- `internal/services/tools` owns deterministic tool descriptors and dispatch.
- `internal/services/approvals` owns approval records and full-project access policy.

Data and artifacts:

- `internal/services/datasets` owns profiling, bounded queries, SELECT-only dataset SQL, notebooks, and chart/dashboard data models.
- `internal/services/dbconnector` owns read-only workspace SQLite inspection/query/cancellation.
- `internal/services/artifacts` owns writes, metadata, search, preview, compare, archive/delete/restore, source freshness, and lineage.
- `internal/services/documents` owns bounded document extraction.

Workbench and operations:

- `internal/services/git` owns manual Git status, diffs, hunk parsing, stage/unstage, and hidden Windows child processes.
- `internal/services/tasks` owns task discovery and safe task execution.
- `internal/services/jobs` owns job status, cancellation, retry, and log tails.
- `internal/services/operations` owns read-only Dockerfile/Compose/env/config/script/log inspection and runbook evidence.

## Testing Strategy

Unit tests should protect:

- rooted path resolution and traversal rejection
- ignored path and `.nexusdesk` protection
- text encoding and binary detection
- file operation previews and rollback records
- patch parsing and exact hunk matching
- dataset profiling/query/SQL guard behavior
- SQLite connector read-only validation and cancellation
- artifact metadata, freshness, archive/delete/restore
- LLM streaming parsing and context escaping
- agent tool argument parsing and approval gating
- UI model state that can be tested without launching the desktop window

Integration tests should use small fixtures and avoid starting external services unless the test explicitly names that dependency.

## Current Review Findings

See `docs/12_PROJECT_REVIEW.md` for the latest full project review and `docs/13_PRODUCTION_READINESS.md` for the production gates. The key engineering direction is:

- finish Fyne parity before adding new top-level studios;
- keep extracting `internal/ui/shell` state as workflows grow;
- port external database profiles and credential handling carefully;
- route OCR, dump imports, connector pulls, report generation, deeper indexing, and long agent runs through jobs;
- keep `app-wails/` until native parity is enough for day-to-day use.
