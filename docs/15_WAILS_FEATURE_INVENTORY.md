# Wails Feature Inventory

Date: 2026-05-28

This inventory records the explicit `port`, `replace`, `drop`, or `later` decisions needed before the preserved Wails app can be frozen. It is based on static inspection of `app-wails/`, `app-wails/frontend/src/`, and the current Fyne-native `nexus-app/` implementation.

## Decision Vocabulary

- `ported`: Already covered by `nexus-app/` with equivalent or better native behavior.
- `port`: Bring the Wails-era capability forward into native code.
- `replace`: Do not port the exact implementation; deliver equivalent value with a native design.
- `drop`: Remove from the active product path because it is Wails/React-specific or superseded.
- `later`: Keep as a roadmap idea, but do not block Native Parity Beta.

## Summary

- Most core backend workflows have native equivalents: workspace open/browse, safe file mutation, rollback records, search/problems, Git status/diff/hunk staging, task runs, artifacts, metadata, datasets, SQLite, external connector profile flows, settings, approvals, diagnostics, chat history, and agent audit.
- The remaining parity blockers are concentrated in editor maturity, protected secrets, assistant profile/memory UX, richer Git history/blame, and a few agent/artifact lineage affordances.
- React/Wails shell code should not be embedded wholesale. Monaco-specific editor behavior should be replaced by a native editor strategy unless a future spike proves an embedded editor can be shipped cleanly without reviving the Wails/webview architecture.

## Inventory

| Area | Wails Evidence | Fyne Status | Decision | Native Target |
| --- | --- | --- | --- | --- |
| Wails app lifecycle, bridge bindings, React routing, Vite build, generated frontend assets | `app-wails/main.go`, `app-wails/wails.json`, `app-wails/frontend/` | Native lifecycle lives in `nexus-app/internal/app`; Fyne shell is active | `drop` | Keep `app-wails/` as reference until freeze, then remove only after explicit approval |
| Workspace open, refresh, preview, search, problems, file write/copy/move/delete, rollback | `app-wails/app.go`, `app-wails/internal/workspace/*` | Native services and shell panels exist with symlink, binary, encoding, and rollback hardening | `ported` | Continue production hardening only |
| Recent workspace management | `app-wails/internal/storage/recent_workspaces.go`, startup state UI | Native Home tab records opened workspaces and supports open/remove/clear recent entries | `ported` | Continue onboarding polish only |
| Monaco syntax highlighting, language workers, minimap/editor outline UX | `MonacoFileEditor.tsx`, `MonacoCodePreview.tsx`, `editorOutline.ts`, frontend `dist/assets/*Mode*` | Native text editor supports editing, preview, dirty-close safety, quick-open, and find/replace; IDE-grade language UX remains incomplete | `replace` | Define Fyne-native syntax/breadcrumb/outline strategy; embed only after a focused spike proves packaging and accessibility |
| Command palette and quick-open | `CommandPalette.tsx`, `QuickOpenPalette.tsx` | Native quick-open keyboard workflow exists; command palette depth is not fully replicated | `ported` for quick-open, `later` for broader command palette | Keep quick-open native; revisit full command palette after editor/navigation polish |
| Git status, file diff, file/hunk stage and unstage | `app-wails/app_git.go`, `GitDiffPanel.tsx` | Native Git panel supports status, diff, hunk-windowing, file/hunk actions, and AI summary | `ported` | Continue destructive action policy separately |
| Git history and blame | `GetGitHistory`, `GetGitBlame`, Wails agent `read_git_history`/`read_git_blame` | Native Git service currently focuses on status/diff/actions | `port` | Add read-only history/blame services, UI rows, and agent context tools |
| Artifact writer, metadata, archive/delete, compare, source freshness | `app-wails/internal/artifact/*`, `ArtifactStudioPanel.tsx` | Native artifact browser/writer/compare/archive/restore/delete/source actions are implemented | `ported` | Continue lineage graph and regeneration work |
| Artifact lineage graph import/export and dependency rebuild | `GetArtifactLineage`, `ExportArtifactLineageJSON`, `ImportArtifactLineageJSON`, `RebuildDatasetDependency` | Native lineage metadata and freshness exist, but graph import/export and rebuild workflow are not complete | `port` | Add lineage graph import/export UI and regeneration actions after artifact metadata stabilizes |
| Dataset profiling, SQL, notebooks, charts, dashboards, SQLite query artifacts | `dataset_service.go`, `DataStudioPanel.tsx`, `DataOperationsPanel.tsx` | Native Data panel covers profiles, query/SQL, notebook run/export, chart/dashboard artifacts, SQLite saved queries, history, and lineage | `ported` | Continue notebook/editor UX and dump import design |
| External database profiles and read-only query flows | `internal/dbconnector/*`, `ConnectorProfilesCard.tsx` | Native profile list/save/delete/test/inspect/query/cancel/history exists for PostgreSQL, MySQL/MariaDB, SQL Server, SQLite, and DuckDB guarded builds | `ported` for functional parity | Move credentials to protected storage before production |
| Protected secret storage | `app-wails/internal/storage/secret_windows.go`, connector sidecar handling | Native stores non-secret settings and redacted connector profiles; OS-protected secret storage is still a gate | `port` | Windows protected storage first; explicit unsupported-platform refusal/fallback |
| LLM settings, provider probe, model catalog, runtime diagnostics | `LLMSettingsCard.tsx`, `llmModelCatalog.ts`, `internal/llm/probe.go` | Native settings include provider/protocol/model/API key, connection test, model count, and runtime diagnostics | `ported` with follow-up | Curated model catalog and deeper GPU/runtime hints remain backlog |
| Assistant chat, streaming, context pack, citations, chat persistence | `AskLLM*`, `PreviewChatContextPack`, `ChatMessageContent.tsx` | Native Ask/Agent, streaming, bounded context, source citations, persisted chat/history, and diagnostics exist | `ported` baseline | Improve weak-evidence warnings, retry/compare, and memory/profile UX |
| Assistant prompt profiles and memory | `internal/storage/assistant_profile.go`, `AgentPanel.tsx` | Native assistant has persistent conversation history, but not full prompt profile/memory parity | `port` | Add prompt profiles and user-visible memory/profile migration plan |
| Assistant retry, compare latest answer, save latest answer as artifact | `AgentPanel.tsx`, `CreateChatMarkdownArtifact` | Native has chat history and artifacts, but the complete retry/compare/save-latest UX is not fully surfaced | `port` | Add assistant answer retry/compare controls and artifact export |
| Agent tool registry for file, Git, tasks, artifacts, data, SQLite, docs, operations | `agent_runtime.go`, `internal/agenttools/*` | Native agent dispatcher covers many read/write/data/artifact/task/document/operations tools with per-call approval and audit | `ported` baseline | Fill remaining read-only context tools as scoped work |
| Agent web fetch | `internal/webfetch/fetch.go`, Wails `agentWebFetch` | Not part of current native safe default tool set | `port` | Add approval-gated, bounded, allow-listed web text fetch with audit |
| Agent constrained shell | `agent_runtime_shell.go` | Native supports safe discovered task execution; arbitrary approved shell remains intentionally absent | `later` | Reconsider only after audit coverage and shell approval policy mature |
| Access policy card / broad workspace trust UX | `AccessPolicyCard.tsx` | Native uses scoped approvals, full-project access, and per-call high-risk modals | `replace` | Keep native approval model; avoid reintroducing broad opaque trust toggles |
| Approval log | `ApprovalLogPanel.tsx`, `internal/approval` | Native approvals panel and metadata repository exist | `ported` | Continue audit coverage for future high-risk operations |
| Metadata browser/search/history | `InspectMetadataStore`, `SearchMetadata`, `appmeta` | Native metadata history, diagnostics, backup/export, and repositories exist | `ported` baseline | Expand recovery and issue-report bundle later |
| Operations read-only inspection/runbooks | `OperationsInspector.tsx`, operations docs | Native operations panel scans Docker/Compose/env/config/log evidence and exports runbooks | `ported` baseline | Mutating Docker/system workflows stay blocked until job/audit maturity |
| Document extraction | `workspace/docx_text.go`, `pdf_text.go` | Native document extraction artifacts cover Markdown/TXT/HTML/XML/DOCX/PDF | `ported` baseline | OCR/scanned documents are later job-routed work |
| Presentation/report generation | Roadmap/docs references, artifact report foundation | Not a completed Wails-only production feature | `later` | Implement after artifact lineage and report-generation jobs are stable |
| Packaging/build docs for Wails | `app-wails/README.md`, Wails build scripts | Active app is `nexus-app/`; Wails instructions are reference-only | `drop` from primary path | Remove Wails build instructions from primary docs after freeze |

## Native Parity Blockers From This Inventory

1. Finish the editor parity strategy: syntax highlighting, breadcrumbs/outline, encoding controls, split/editor layout decision.
2. Implement native protected secret storage on Windows with explicit unsupported-platform behavior.
3. Add assistant prompt profiles/memory plus retry/compare/save-answer UX.
4. Port read-only Git history/blame into the native Git panel and agent context tools.
5. Add artifact lineage import/export and regeneration workflows.
6. Add approval-gated agent web fetch if still desired for parity.

## Retirement Decision

`app-wails/` should remain in the repository until the blockers above are either implemented or explicitly moved out of Native Parity Beta. After that point:

1. Freeze `app-wails/` as reference-only.
2. Remove Wails build instructions from primary docs.
3. Archive or delete `app-wails/` only after explicit approval.
