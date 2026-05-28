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
- The remaining parity blockers are concentrated in editor maturity, deeper artifact regeneration coverage, and richer assistant/source quality polish.
- React/Wails shell code should not be embedded wholesale. Monaco-specific editor behavior should be replaced by a native editor strategy unless a future spike proves an embedded editor can be shipped cleanly without reviving the Wails/webview architecture.

## Inventory

| Area | Wails Evidence | Fyne Status | Decision | Native Target |
| --- | --- | --- | --- | --- |
| Wails app lifecycle, bridge bindings, React routing, Vite build, generated frontend assets | `app-wails/main.go`, `app-wails/wails.json`, `app-wails/frontend/` | Native lifecycle lives in `nexus-app/internal/app`; Fyne shell is active | `drop` | Keep `app-wails/` as reference until freeze, then remove only after explicit approval |
| Workspace open, refresh, preview, search, problems, file write/copy/move/delete, rollback | `app-wails/app.go`, `app-wails/internal/workspace/*` | Native services and shell panels exist with symlink, binary, encoding, and rollback hardening | `ported` | Continue production hardening only |
| Recent workspace management | `app-wails/internal/storage/recent_workspaces.go`, startup state UI | Native Home tab records opened workspaces and supports open/remove/clear recent entries | `ported` | Continue onboarding polish only |
| Monaco syntax highlighting, language workers, minimap/editor outline UX | `MonacoFileEditor.tsx`, `MonacoCodePreview.tsx`, `editorOutline.ts`, frontend `dist/assets/*Mode*` | Native text editor supports editing, preview, dirty-close safety, quick-open, find/replace with live Wails-style match counts, Wails-derived breadcrumbs, explicit save encoding controls, deterministic Go/JSON formatting, safe Markdown/config/SQL/Dockerfile/text whitespace formatting, Wails-style secondary split preview selection, a Fyne outline tab ported from the Wails outline rules, and searchable go-to-symbol navigation | `replace` | Finish syntax, minimap, and deeper language-action strategy; embed only after a focused spike proves packaging and accessibility |
| Command palette and quick-open | `CommandPalette.tsx`, `QuickOpenPalette.tsx` | Native quick-open keyboard workflow exists; command palette depth is not fully replicated | `ported` for quick-open, `later` for broader command palette | Keep quick-open native; revisit full command palette after editor/navigation polish |
| Git status, file diff, file/hunk stage and unstage | `app-wails/app_git.go`, `GitDiffPanel.tsx` | Native Git panel supports status, diff, hunk-windowing, file/hunk actions, and AI summary | `ported` | Continue destructive action policy separately |
| Git history and blame | `GetGitHistory`, `GetGitBlame`, Wails agent `read_git_history`/`read_git_blame` | Native Git service, Git panel, and deterministic agent tools expose read-only history/blame | `ported` | Continue broader Git AI/review work separately |
| Artifact writer, metadata, archive/delete, compare, source freshness | `app-wails/internal/artifact/*`, `ArtifactStudioPanel.tsx` | Native artifact browser/writer/compare/archive/restore/delete/source actions are implemented | `ported` | Continue regeneration work |
| Artifact lineage graph import/export and agent context | `GetArtifactLineage`, `ExportArtifactLineageJSON`, `ImportArtifactLineageJSON`, Wails `read_artifact_lineage` | Native workspace lineage graph export/import UI and read-only agent lineage tool are implemented | `ported` | Continue graph polish only |
| Artifact dependency rebuild/regeneration | `RebuildDatasetDependency` | Native can regenerate dataset query CSV, dataset SQL report, chart, and dashboard artifacts from dependency metadata | `ported` baseline | Continue broader regeneration coverage for notebooks, summaries, and future artifact kinds |
| Dataset profiling, SQL, notebooks, charts, dashboards, SQLite query artifacts | `dataset_service.go`, `DataStudioPanel.tsx`, `DataOperationsPanel.tsx` | Native Data panel covers profiles, query/SQL, notebook run/export, chart/dashboard artifacts, SQLite saved queries, history, and lineage | `ported` | Continue notebook/editor UX and dump import design |
| External database profiles and read-only query flows | `internal/dbconnector/*`, `ConnectorProfilesCard.tsx` | Native profile list/save/delete/test/inspect/query/cancel/history exists for PostgreSQL, MySQL/MariaDB, SQL Server, SQLite, and DuckDB guarded builds with protected Windows credential sidecars | `ported` for functional parity | macOS Keychain and Linux Secret Service remain future platform work |
| Protected secret storage | `app-wails/internal/storage/secret_windows.go`, connector sidecar handling | Native settings API keys and connector credentials use DPAPI-protected sidecars on Windows, redacted display values, and explicit unsupported-platform refusal elsewhere | `ported` Windows baseline | Add macOS Keychain and Linux Secret Service/libsecret before claiming full cross-platform secret support |
| LLM settings, provider probe, model catalog, runtime diagnostics | `LLMSettingsCard.tsx`, `llmModelCatalog.ts`, `internal/llm/probe.go` | Native settings include provider/protocol/model/protected API key, connection test, model count, runtime diagnostics, the Wails curated model catalog with context/reserve sizing, and loaded-model runtime context tuning | `ported` with follow-up | Deeper GPU/runtime hints remain backlog |
| Assistant chat, streaming, context pack, citations, chat persistence | `AskLLM*`, `PreviewChatContextPack`, `ChatMessageContent.tsx` | Native Ask/Agent, streaming, bounded context, context-path persistence, Wails-compatible context-to-source fallback parsing, source/model footer diagnostics, weak-evidence warning, stale-source chat-history warning, persisted chat/history, and diagnostics exist | `ported` baseline | Improve finer-grained citations beyond file-level sources |
| Assistant prompt profiles and memory | `internal/storage/assistant_profile.go`, `AgentPanel.tsx` | Native Fyne loads/saves the Wails-compatible assistant profile store, applies active prompt profiles to Ask requests, and exposes memory/profile controls | `ported` baseline | Add profile editing beyond default profiles if needed |
| Assistant retry, compare latest answer, save latest answer as artifact | `AgentPanel.tsx`, `CreateChatMarkdownArtifact` | Native Fyne surfaces retry, compare, and save-latest-answer; saved answers become `chat-answer` artifacts with prompt/model/context/source metadata | `ported` baseline | Continue polishing citation depth |
| Agent tool registry for file, Git, tasks, artifacts, data, SQLite, docs, operations | `agent_runtime.go`, `internal/agenttools/*` | Native agent dispatcher covers many read/write/data/artifact/task/document/operations tools with per-call approval and audit | `ported` baseline | Fill remaining read-only context tools as scoped work |
| Agent web fetch | `internal/webfetch/fetch.go`, Wails `agentWebFetch` | Native deterministic `web_fetch` is approval-gated and preserves Wails bounds for HTTP(S), redirects, size, content type, local-network blocking, and optional domain allow-lists | `ported` | Keep browser automation/screenshots out of scope until explicitly designed |
| Agent constrained shell | `agent_runtime_shell.go` | Native supports safe discovered task execution; arbitrary approved shell remains intentionally absent | `later` | Reconsider only after audit coverage and shell approval policy mature |
| Access policy card / broad workspace trust UX | `AccessPolicyCard.tsx` | Native uses scoped approvals, full-project access, and per-call high-risk modals | `replace` | Keep native approval model; avoid reintroducing broad opaque trust toggles |
| Approval log | `ApprovalLogPanel.tsx`, `internal/approval` | Native approvals panel and metadata repository exist | `ported` | Continue audit coverage for future high-risk operations |
| Metadata browser/search/history | `InspectMetadataStore`, `SearchMetadata`, `appmeta` | Native metadata history, diagnostics, backup/export, and repositories exist | `ported` baseline | Expand recovery and issue-report bundle later |
| Operations read-only inspection/runbooks | `OperationsInspector.tsx`, operations docs | Native operations panel scans Docker/Compose/env/config/log evidence and exports runbooks | `ported` baseline | Mutating Docker/system workflows stay blocked until job/audit maturity |
| Document extraction | `workspace/docx_text.go`, `pdf_text.go` | Native document extraction artifacts cover Markdown/TXT/HTML/XML/DOCX/PDF | `ported` baseline | OCR/scanned documents are later job-routed work |
| Presentation/report generation | Roadmap/docs references, artifact report foundation | Not a completed Wails-only production feature | `later` | Implement after artifact lineage and report-generation jobs are stable |
| Packaging/build docs for Wails | `app-wails/README.md`, Wails build scripts | Active app is `nexus-app/`; Wails instructions are reference-only | `drop` from primary path | Remove Wails build instructions from primary docs after freeze |

## Native Parity Blockers From This Inventory

1. Finish the editor parity strategy: syntax highlighting, minimap, and deeper language-action choices.
2. Continue assistant quality parity: finer-grained citations beyond file-level sources.
3. Expand artifact regeneration beyond the first dataset/query/chart rebuild baseline.
4. Add macOS Keychain and Linux Secret Service/libsecret support after the Windows secret-storage baseline.

## Retirement Decision

`app-wails/` should remain in the repository until the blockers above are either implemented or explicitly moved out of Native Parity Beta. After that point:

1. Freeze `app-wails/` as reference-only.
2. Remove Wails build instructions from primary docs.
3. Archive or delete `app-wails/` only after explicit approval.
