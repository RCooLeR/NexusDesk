# Architecture

Status: canonical architecture for the active NexusDesk app.

## 1. Product Shape

NexusDesk is a local-first Fyne desktop studio for code, data, documents, databases, artifacts, operations evidence, and AI-assisted work. It is intentionally built as a native desktop application with framework-free services and a Fyne-only UI shell.

The architecture has four core ideas:

- The UI is native and thin: it renders state, captures intent, shows progress, and delegates business rules to services.
- Services are reusable and testable: workspace, files, Git, data, database connectors, documents, artifacts, jobs, assistant, agent, approvals, settings, secrets, diagnostics, and release logic live outside Fyne.
- Risk is explicit: dangerous actions are mediated through deterministic tools, approvals, audit records, output bounds, redaction, cancellation, and rollback or mitigation.
- Local state is transparent: user-visible outputs live in the workspace under `.nexusdesk/`, while user-level settings and secrets live in OS-appropriate locations.

The app should remain a single product, but its internal design must allow services to be reused later by a CLI, tests, automation, or a future server process without dragging in UI dependencies.

## 2. Repository Shape

```text
NexusDesk/
  README.md                         Project overview
  tracker.md                        Short pointer to the canonical tracker
  docs/                             Canonical source of truth
    README.md                       Documentation index
    01_ARCHITECTURE.md              This document
    02_UI_WORKBENCH.md              Target UI and UX specification
    03_FEATURES.md                  Implemented and planned capabilities
    04_GOALS.md                     Product goal and release bar
    05_PLAN.md                      Phases, milestones, gates
    06_TRACKER.md                   Detailed execution checklist
  nexus-app/                        Active Fyne-native desktop app
    main.go                         Thin executable entrypoint
    go.mod / go.sum                 Single Go module
    scripts/                        CI, dev environment, icon/build helpers
    internal/
      app/                          App lifecycle, dependency assembly, window setup
      architecture/                 Import-boundary tests
      brand/                        Embedded app icon and logo resources
      buildinfo/                    Version, commit, app identity constants
      domain/                       Framework-free domain models and value types
      platform/                     Platform-specific filesystem/clipboard helpers
      release/                      Release manifest and packaging checks
      services/                     Framework-free application behavior
      ui/shell/                     Fyne workbench, panels, dialogs, shortcuts
      ui/theme/                     Fyne theme and visual tokens
  services/                         Optional local helper services for development
```

## 3. Layer Boundaries

### 3.1 Entry and app lifecycle

`nexus-app/main.go` must stay tiny. Its job is to call the app bootstrap and exit. It must not assemble services, own UI widgets, load workspace state, run background work, or contain product behavior.

`internal/app` owns app construction, startup session marker integration, Fyne app/window creation, dependency assembly, theme activation, app identity, initial window sizing, and clean shutdown coordination.

### 3.2 Domain layer

`internal/domain` owns pure value types. It avoids UI, persistence-driver, network, provider, and platform concerns unless the value object genuinely represents a cross-layer concept.

Domain rules:

- Prefer plain structs and explicit fields.
- Keep IO-dependent validation out of domain.
- Do not import Fyne, app packages, service implementations, or database drivers.

### 3.3 Services layer

`internal/services` owns nearly all product behavior. Services must remain framework-free. They may depend on other services when that dependency is stable and intentional, but they must not depend on Fyne or UI packages.

Major service areas:

- `workspace`: safe rooted path handling, listing, preview, search, problem scanning, context packs, project memory, file operations, rollback.
- `editor`: tab/session model, syntax strategy, formatting, diagnostics, outline, breadcrumbs, references, definition lookup.
- `assistant`: Ask-mode orchestration, context, source quality, chat result persistence.
- `agent`: agent loop, planning, tool call parsing, iteration limits, final-answer handling.
- `tools`: first-party agent tool dispatcher, catalog, risk model, tool handlers.
- `approvals`: approval records and full-project access policy.
- `metadata`: SQLite-backed store for chats, jobs, artifacts, approvals, task runs, SQL runs, audit trails, and recovery.
- `jobs`: durable job ledger, logs, cancellation, retry/open-output coordination.
- `tasks`: workspace task discovery and safe task execution.
- `git`: status, diff, staging, hunk operations, commit, branch, history, blame, conflict resolution.
- `datasets`: profiling and querying CSV, TSV, JSON, NDJSON, XLSX, Parquet metadata, logs, notebooks, charts.
- `dbconnector`: read-only external database profiles, schema inspection, bounded queries.
- `documents`: bounded text extraction and document handling.
- `spreadsheets`: bounded XLSX parsing.
- `artifacts`: artifact creation, metadata, lineage, freshness, archive/restore/delete, regeneration support.
- `operations`: read-only Dockerfile, Compose, env/config/script/log inspection and runbook generation.
- `llm`: OpenAI-compatible chat, streaming, probes, model/runtime capabilities.
- `settings`: provider settings, task model routes, persistence, defaults.
- `protectedsecret`: OS-protected secret storage.
- `webfetch`: approval-gated HTTP(S) text fetch with SSRF and size guards.
- `readiness`: first-run readiness and production failure scenario checks.
- `startup`: crash marker and recovery surface data.
- `issuereport`: redacted support bundle export.
- `release`: manifest, version, packaging readiness.
- `externalagents`: detection and readiness planning for external coding-agent CLIs.
- `security`: threat-model control vocabulary in code form.

Service rules:

- Accept context or cancellation where work can be slow.
- Return structured results, not UI-formatted strings, when behavior is reusable.
- Own validation and safety rules. UI can add guardrails, but cannot be the only guard.
- Enforce rooted paths and symlink-safe behavior for workspace file access.
- Create rollback for file mutation or explicitly document why rollback is impossible.
- Bound time, output, and resources for network, model, database, shell, Git, and parsing work.

### 3.4 UI layer

`internal/ui/shell` owns the Fyne workbench. It should evolve toward controllers instead of one large view object.

UI responsibilities:

- render workbench layout;
- manage visible panels and tool windows;
- capture user actions;
- show progress, loading, cancellation, errors, and recovery options;
- marshal worker results back onto Fyne's main goroutine;
- avoid blocking the UI thread;
- reflect service-owned safety decisions clearly.

UI must not re-implement path safety, SQL safety, approval policy, terminal policy, database mutation policy, or secret handling policy.

### 3.5 Theme and brand

`internal/ui/theme` owns visual tokens and Fyne theme behavior. Hardcoded colors should be treated as drift unless they are one-off graph/chart colors with documented reason.

`internal/brand` owns embedded app resources used by the application. Documentation describes brand use, not duplicated runtime assets.

## 4. Runtime Data Flow

```text
User event
  -> Fyne widget callback
  -> shell controller or View method
  -> service request object
  -> framework-free service
  -> bounded IO / model / database / tool / job work
  -> structured result or job update
  -> fyne.Do(...) back onto the UI thread
  -> visible state update, status, audit, artifact, or diagnostic
```

Long-running work:

```text
User starts workflow
  -> UI creates job or calls job-aware service
  -> service records job metadata
  -> worker does cancellable work
  -> service appends logs/status
  -> UI renders progress and cancel/retry/open-output controls
  -> result becomes artifact, editor state, data grid, diagnostic, or error
```

Agent work:

```text
User prompt + pinned context
  -> assistant/agent request
  -> model route selection
  -> model response stream
  -> parser extracts plan/tool/final answer
  -> tool dispatcher validates risk and arguments
  -> approval requested if required
  -> tool executes with bounds and audit
  -> observation returns to the model loop
  -> final answer rendered with citations, sources, and tool history
```

## 5. Storage Architecture

### 5.1 Workspace-local storage

Each workspace can contain a `.nexusdesk/` directory. This is the user's local, inspectable project state.

```text
<workspace>/.nexusdesk/
  approvals/
    log.json
    policy.json
  artifacts/
    reports/
    charts/
    dashboards/
    notebooks/
    chat-answers/
    documents/
    presentations/
    runbooks/
    archive/
    metadata sidecars
  datasets/
    notebooks/
    saved queries/filter state
  jobs/
    <job-id>/log.txt
  metadata/
    nexusdesk.sqlite
    schema.sql
    sqlite-manifest.json
    recovery/
  rollbacks/
    log.json
    <rollback-id>/
```

Workspace-local rules:

- Data must be readable outside the app where practical.
- Generated artifacts must have provenance metadata.
- Mutations must write enough metadata for audit, rollback, or recovery.
- Metadata corruption recovery must preserve the damaged file and explain the action.

### 5.2 User-level storage

User-level storage belongs under the OS user config directory, using a NexusDesk-specific folder.

```text
NexusDesk/
  settings.json
  connector-profiles.json
  recent-workspaces.json
  assistant-profile.json
  startup-session.json
  protected secret sidecars/tokens
```

User-level rules:

- API keys and connector credentials must use OS-protected storage.
- If protected storage is unavailable, the app must refuse to store secrets and show remediation.
- Issue reports must redact secrets and must not include workspace content unless the user opts in.

## 6. Safety Architecture

### 6.1 Workspace open safety

Opening a workspace must be cheap and side-effect-free. It may list bounded directory contents and initialize visible state. It must not trigger Git status, shell commands, Docker, model calls, connector pulls, OCR, imports, browser automation, dump processing, deep indexing, or task runs.

### 6.2 Path safety

Every workspace path must be normalized, rooted, symlink-aware, and platform-aware.

Required behavior:

- Reject `..` traversal.
- Reject absolute paths where a relative workspace path is expected.
- Evaluate symlink components before reads and writes.
- Reject paths that resolve outside the workspace root.
- Reject unsafe Windows-specific path forms where relevant.
- Keep preview/read/write behavior consistent.

### 6.3 Mutation safety

All file mutations must follow this chain:

```text
request -> normalize path -> preview/plan -> approval if required -> snapshot -> apply -> verify -> audit -> UI refresh
```

Required for file write, append, patch, create, delete, copy, move, rename, conflict resolution, formatting, and rollback application.

### 6.4 Secret safety

Secrets include model API keys, connector credentials, future remote tokens, and any credential-like value discovered in logs or reports.

Rules:

- Store secrets in OS-protected storage.
- Never write plaintext secrets into settings JSON.
- Never include secrets in command-line arguments.
- Redact secrets in diagnostics, errors, jobs, issue reports, and tool observations.
- Show precise remediation if protected storage is unavailable.

### 6.5 Network safety

Network-capable features must be explicit, bounded, and explainable.

Rules:

- Model calls happen only after provider/model setup and explicit user action.
- `web_fetch` allows HTTP(S) text retrieval only under SSRF, redirect, content-type, and size guards.
- External database connections default to encrypted transport unless the user explicitly opts into plaintext development mode.
- XLSX, DOCX, and generated Office package validation use bounded ZIP member, file-count, and total-uncompressed-size caps before reading package contents.
- Browser automation is planned only after a separate policy for URLs, cookies, screenshots, downloads, storage, and approvals.

### 6.6 Database safety

External databases and workspace SQLite flows are read-only by default.

Rules:

- Accept only read-only SQL forms.
- Tokenize SQL rather than using raw keyword matching alone.
- Bound result rows, columns, time, and memory.
- Support cancellation.
- Redact connection errors.
- Audit profile tests, queries, exports, and failures.

### 6.7 Agent safety

The LLM never directly mutates state. The model proposes actions; deterministic tools validate and execute them.

Required controls per tool:

- catalog entry;
- risk classification;
- argument validation;
- rooted path checks where applicable;
- output caps;
- cancellation or timeout where applicable;
- approval for medium/high risk;
- audit record;
- redaction;
- rollback or mitigation for high-risk mutations;
- visible UI timeline.

## 7. Job Architecture

Slow workflows must route through durable jobs before broad exposure.

Job-owned workflows include or should include:

- data profiling on large files;
- SQL notebooks and exports;
- connector inspection/query when slow;
- artifact regeneration;
- document and presentation generation;
- OCR/scanned document extraction;
- dump import;
- browser automation;
- terminal/task execution;
- long agent workflows.

Each job must support stable ID, type, title, status, progress where measurable, logs, cancellation, retry when safe, output path/open action, persisted metadata, and issue-report inclusion.

## 8. Tool Architecture

The tool registry is the contract between the agent and the application.

Implemented tool categories should remain first-party and deterministic:

- registry and diagnostics;
- workspace read/search/context;
- file mutations and rollback;
- editor formatting/diagnostics;
- Git state and explicit Git mutations;
- tasks and approved argv terminal commands;
- jobs;
- web fetch;
- datasets and SQLite;
- external database inspection/query when routed safely;
- documents;
- artifacts and regeneration;
- operations inspection/runbooks;
- redaction and approvals;
- external agent readiness planning.

Planned tool categories must remain non-executable until safety design lands:

- browser automation;
- interactive terminal sessions;
- pull-request platform tools;
- MCP tools;
- scheduled automations;
- image/screenshot understanding;
- richer semantic search;
- connector sync jobs;
- plugin-hosted tools.

## 9. UI Architecture Target

Target shell structure:

```text
View
  - owns window-level layout, rails, status bar, theme hooks
  - owns controller registry
  - owns cross-controller event bus

Controllers
  - projectController
  - editorController
  - assistantController
  - dataController
  - gitController
  - searchController
  - problemsController
  - artifactsController
  - jobsController
  - approvalsController
  - diagnosticsController
  - settingsController
  - operationsController
```

Controller rules:

- Each controller owns its widgets and small local state.
- Controllers communicate through typed events, not by reaching into each other's fields.
- Services remain outside controllers and are injected.
- Workers marshal UI changes with `fyne.Do`.
- `View` should become layout registry plus common state, not a god object.

Acceptance targets:

- `View` has fewer than 25 pointer fields.
- Large panel files are under 600 lines each or split by behavior.
- Adding a tool window requires one registry entry plus a controller, not edits across unrelated files.
- Resize behavior is centralized and tested.

## 10. Release Architecture

Production releases require repeatable, auditable packaging.

Required release components:

- version metadata embedded in the app;
- release manifest with hashes;
- SBOM;
- provenance evidence;
- Windows signing;
- macOS signing/notarization decision and smoke;
- Linux package strategy;
- clean-machine smoke checklist;
- issue-report export path;
- clear release notes.

Windows-specific concern:

- Local unsigned build artifacts can trigger antivirus products. Development build scripts should support build-check mode that removes generated executables immediately after validation unless a runnable build is explicitly requested.

## 11. Architecture Non-Negotiables

- Native Fyne shell only for the desktop app.
- Services stay framework-free.
- Fyne imports stay in app, UI, theme, brand, and tests that explicitly target UI.
- Workspace open remains cheap and side-effect-free.
- Risky actions are approval-gated and audited.
- File mutations are reversible where practical.
- Secrets never land in plaintext configuration.
- Slow workflows use durable jobs.
- Generated outputs are real artifacts with provenance.
