# Fyne Migration Plan

Nexus Augentic Studio is moving from Wails/React to a Fyne-native Go desktop application. This is a deliberate breaking change.

## Why This Makes Sense

The product is aiming at an IDE/data/document/operations studio, not a web dashboard wrapped in a desktop window. Wails got us a fast prototype, but the repeated gray/blank-window issues, generated binding churn, webview lifecycle concerns, and growing React shell complexity are now working against the architecture we want.

Fyne gives us:

- one Go application instead of Go plus React plus Wails bindings;
- native windows, menus, dialogs, shortcuts, and app lifecycle;
- simpler service boundaries because UI can call Go services directly;
- fewer moving parts for local-first security, approvals, jobs, and filesystem access;
- a cleaner path to package the app as a native desktop tool.

The tradeoff is real: we lose React/Monaco and must rebuild dense IDE UI patterns with native widgets or selected embedded components later. That is acceptable if we treat the Wails app as the reference implementation, not as code we must copy line-for-line.

## Repository Layout

```text
app-wails/                    Existing Wails/React implementation, preserved as reference
nexus-app/                    New Fyne-native application
nexus-app/main.go             Entrypoint only
nexus-app/go.mod              Fyne app module
nexus-app/internal/app/       Application lifecycle and windows
nexus-app/internal/domain/    Framework-free domain models
nexus-app/internal/services/  Workspace, Git, LLM, data, artifact, job services
nexus-app/internal/ui/        Fyne views, layouts, widgets, theme
docs/                         Product and architecture docs
tracker.md                    Fyne migration tracker
```

## Migration Strategy

1. Preserve `app-wails` exactly as the current behavior source.
2. Build `nexus-app` as a new app with a thin root and strongly grouped `internal/` packages.
3. Port stable backend services by capability: workspace, file preview, safe writes, Git, LLM, artifacts, data, jobs.
4. Rebuild UI natively, using the old UI only for workflow reference.
5. Keep folder open cheap and bounded. Expensive indexing, OCR, connector pulls, dump imports, and long agent work must become jobs.
6. Retire Wails only after native parity is good enough for day-to-day use.

## First Native Architecture

```mermaid
flowchart LR
  Main["main.go"] --> App["internal/app"]
  App --> Shell["internal/ui/shell"]
  Shell --> Services["internal/services"]
  Services --> Domain["internal/domain"]
  Services --> Workspace["Workspace Files"]
  Services --> Metadata["SQLite / .nexusdesk"]
  Services --> LLM["LLM Providers"]
  Services --> Jobs["Job Runner"]
```

Rules:

- `internal/domain` imports no UI packages.
- `internal/services` imports domain and infrastructure, not Fyne widgets.
- `internal/ui` imports Fyne plus services.
- Root keeps only `main.go`, `go.mod`, `go.sum`, and high-level project docs.

## Immediate Risks

- Fyne desktop builds need CGO and a C compiler on Windows.
- Monaco-grade editor behavior will need either careful native implementation, an embedded editor strategy, or a deliberately simpler first editor.
- Some visual polish from the React UI will need to be rebuilt with custom Fyne widgets.
- Wails generated bindings disappear, so frontend smoke tests must be replaced with Go service/UI tests and native visual checks.

## Current Baseline

The first `nexus-app` slice includes:

- native app lifecycle;
- branded dark theme foundation;
- embedded approved app icon and horizontal logo from `docs/brand/`;
- native main menu with File, Edit, View, Navigate, Tools, and Help groups;
- shortcut registry for open workspace, refresh, close tab, tab navigation, and settings;
- Workbench-style shell with rail, toolbar, navigator, editor tabs, assistant panel, and bottom tabs;
- shell UI split into focused files for panels, tabs, workspace actions, activity, tree, and preview;
- native folder-open dialog;
- lazy bounded workspace listing service with traversal protection, ignored folder handling, symlink skip, and entry caps;
- first rooted read-only file preview for capped UTF-8, UTF-16, and Windows-1251 text plus binary metadata;
- first editor tab lifecycle with same-file tab reuse and close cleanup;
- UI-independent editor tab session model with active tab, dirty state, pinned state, reuse, and close guards;
- native editor chrome for pinned tabs, dirty indicators, and state-driven tab labels/icons;
- first native draft-only text editor with Source/Preview tabs, automatic dirty tracking, disabled Save, and local draft revert;
- Markdown editor previews render Markdown in the Preview tab while non-Markdown text files stay in read-only source preview mode;
- first native image preview path for capped workspace image files using service-returned bytes and Fyne image rendering;
- first native capped CSV/TSV table preview path with service-side parsing and UI-side table rendering;
- first native DOCX text preview path using bounded service-side extraction from `word/document.xml`;
- first native PDF text preview path using bounded service-side literal text extraction and a read-only Fyne text surface;
- first native text/code safe write service with rooted diff previews, append/apply flows, encoding-aware writes, and rollback snapshots under `.nexusdesk/rollbacks`;
- draft editor Save wiring that applies through the native safe write service, marks the editor tab clean, and leaves a rollback record;
- first native file create, delete, copy, move, and rename services with rooted validation, metadata guards, operation previews, and rollback records;
- first native workspace search service and bottom result panel for bounded path/content search with preview-tab opening;
- first native Problems service and bottom panel for bounded TODO/FIXME/HACK/BUG, merge-conflict, and invalid JSON scanning;
- first native Git status service and manual bottom Git refresh panel with hidden Windows command execution;
- directory-grouped changed-file rendering in the native Git panel;
- first native read-only selected-file Git diff service with unified, split, and diff-only panel modes;
- first native parsed diff hunk metadata and previous/next hunk selection in the Git panel;
- first native confirmed file-level Git stage and unstage controls through the Git service boundary;
- first native rollback browser panel for safe-write and file-operation records with confirmation before apply;
- first native workspace-tree action strip for create, copy, rename/move, and delete through the file-operation services;
- first native non-secret settings store and provider/model settings page skeleton;
- framework-free workspace domain model.

Full execution is blocked in the current shell until CGO is enabled with a Windows C compiler. Non-driver internal packages can already be tested.

## Contributor-Grade Structure

The new app should be easy for outside contributors to navigate:

- one package should have one reason to change;
- UI code should be split by panel/dialog/widget, not accumulated in one shell file;
- service packages should expose small use-case methods and keep file/path/database/LLM safety inside services;
- every major package should get tests before it becomes a dependency of agent or UI workflows;
- long files are a smell. Split by responsibility before adding a second unrelated workflow to the same file.

Planned feature parity is preserved in `tracker.md` under the long-term functionality backlog.
