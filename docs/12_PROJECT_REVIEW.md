# Project Review

Date: 2026-05-28

This review records the current state of Nexus Augentic Studio after the latest Fyne migration and production-readiness work. It is intentionally explicit: `nexus-app/` is the active product, while `app-wails/` remains the reference implementation until remaining native parity blockers are either completed or deliberately moved out of Native Parity Beta.

## Executive Summary

Nexus Augentic Studio is now much closer to a production-grade native desktop studio than to a migration prototype. The Wails-to-Fyne move remains the right architectural decision: the active app is a local-first IDE/data/document/operations workbench with an always-visible assistant, not a web dashboard wrapped in a desktop shell.

Approximate current status:

- Fyne-native migration: 96-97% complete by useful Wails-era functionality.
- Wails useful-code parity: 94-96% complete.
- Native Parity Beta readiness: 90-93% complete.
- Overall production readiness: about 87% complete.
- Distribution/packaging readiness: about 65-70% complete.

The remaining work is concentrated in final editor parity, richer generated document/presentation outputs, deeper assistant evidence quality, durable routing for slow workflows, signed packaging, onboarding, platform smoke, and final UI polish.

## Architecture Health

The architecture is healthy and production-oriented.

Healthy areas:

- `nexus-app/main.go` is thin and delegates lifecycle to `internal/app`.
- `internal/domain` contains framework-free domain models.
- `internal/services` contains UI-independent application behavior.
- `internal/ui` owns Fyne widgets, layouts, dialogs, and theme code.
- Fyne imports are contained to app/UI/theme/test areas rather than leaking into service/domain packages.
- Workspace open remains bounded and cheap by design.
- Git, Docker, OCR, connector pulls, dump imports, model calls, shell commands, and deep indexing remain explicit user/job actions.
- File mutations, rollback records, approvals, and agent tools share service-owned safety rules.
- SQLite metadata is a first-class foundation for chats, approvals, jobs, artifacts, SQL, dataset dependencies, agent runs, and tool runs.
- Compatibility import from Wails-era metadata exists and has been moved off the workspace-open critical path.

Risks to watch:

- `internal/ui/shell` is modularized into many files, but it still carries substantial orchestration complexity.
- Future UI work should continue extracting focused controllers, panel state, and service-owned behavior before adding new large workflows.
- The native editor is strong for a Fyne baseline, but not yet Monaco/LSP-grade.
- Product breadth is high, so final polish now depends on cohesive workflows, onboarding, and confidence-building diagnostics rather than simply adding more panels.

## Current Native Capability Snapshot

The active `nexus-app/` can already do the following.

Workspace and shell:

- Launch as a Fyne-native desktop app.
- Open local workspaces through native dialogs.
- Maintain recent workspaces.
- Render a lazy workspace tree with ignored-path handling and entry caps.
- Reveal, refresh, collapse, and context-act on project tree nodes.
- Keep folder open fast and safe by avoiding automatic expensive external work.

Workbench and editor:

- Preview text, code, Markdown, images, CSV/TSV, DOCX, PDF text, XLSX-derived content, and binary metadata.
- Safely edit text/code through draft state, dirty markers, pinned tabs, close guards, save, revert, rollback, and explicit discard confirmation.
- Create, delete, copy, move, and rename files/folders through rooted validated services.
- Use quick open and command palette keyboard workflows.
- Use breadcrumbs, split preview, source/rendered Markdown, find/replace, and formatting actions.
- Detect languages using Wails-derived rules.
- Show bounded token analysis and a native syntax mirror/Highlight tab.
- Show cursor-aware active-line/token/symbol status.
- Show live unsaved-draft diagnostics for markers, merge conflicts, JSON, Go, YAML, TOML, and XML.
- Show outline, go-to-symbol, local definition, bounded workspace definition fallback, references search, and document map surfaces.

Search and problems:

- Search workspace paths and file contents with snippets.
- Scan TODO/FIXME/HACK/BUG markers, merge conflicts, and saved-file syntax diagnostics.
- Surface diagnostics in a bottom Problems panel.

Assistant and agent:

- Configure OpenAI-compatible, Ollama, and custom endpoint providers.
- Store OS-protected API keys and connector credentials.
- Probe providers, count models, and show runtime diagnostics.
- Stream Ask and Agent responses using native Go services.
- Use selected workspace, directory, file, and artifact context.
- Persist chat history and provide chat search/history navigation.
- Seed Agent runs from prior chat context.
- Use assistant memory and prompt profiles.
- Retry, compare, and save answers as artifacts.
- Persist source/model/footer diagnostics, line-aware citation refs, citation snippets, unverified/out-of-context citation warnings, cited/uncited coverage, evidence labels, and stale-source warnings.
- Run agent plans and deterministic tools with approval, audit, rollback, and final-answer fallback.

Git and IDE operations:

- Manually refresh Git status.
- Show project-tree Git badges.
- Render grouped changed files.
- Show unified, split, and diff-only diff views.
- Navigate hunks with large-file elision and hunk windowing.
- Stage/unstage files and hunks through explicit approval paths.
- Generate AI diff summaries and commit drafts.
- Show read-only Git history and blame.

Tasks and jobs:

- Discover safe project tasks.
- Run whitelisted tasks through jobs.
- Capture logs, status, cancellation, retry, and task-report artifacts.
- Persist jobs/task runs in SQLite metadata.

Data and analytics:

- Profile CSV, TSV, JSON, NDJSON, XLSX, Parquet metadata, and logs.
- Query/filter/order/limit bounded local datasets.
- Run SELECT-only dataset SQL with persisted run/dependency metadata.
- Use SQL notebooks with SQL/chart cells, directive parsing, save/load, run/export, result tabs, and artifact generation.
- Browse workspace SQLite databases with schema, views, indexes, relationships, row counts, samples, saved queries, and exports.
- Manage external DB profiles for PostgreSQL, MySQL/MariaDB, SQL Server, SQLite, and DuckDB guarded builds.
- Run read-only external connector tests, inspections, queries, cancellation, history, redaction, and credential sidecars.
- Generate chart/dashboard SVG previews and artifacts.

Artifacts and provenance:

- Browse, search, preview, archive, delete, restore, compare, and inspect generated artifacts.
- Track source lineage, source fingerprints, freshness warnings, and metadata sidecars.
- Export/import artifact lineage graph JSON.
- Pin artifacts into assistant/agent context.
- Regenerate dataset summary, dataset query, SQL report, chart, dashboard, SQL notebook, SQLite query, document report, workspace scan, document extraction, operations runbook, comparison, chat-answer refresh, and presentation-outline artifacts.

Documents and operations:

- Extract and preview TXT, Markdown, PDF, DOCX, XLSX, HTML, and XML style documents.
- Generate document extraction/report artifacts.
- Inspect Dockerfiles, Compose, env/config/script/log evidence.
- Summarize Compose topology.
- Validate Compose config through explicit safe jobs.
- Export operations runbook artifacts.

Diagnostics, safety, and persistence:

- Persist local metadata in SQLite.
- Import Wails-era chat, approval, artifact, tool-run, SQL, and dependency metadata.
- Recover corrupt metadata on workspace open.
- Export metadata backups and workspace state backups.
- Show diagnostics for providers, metadata health, jobs, tasks, SQL, agent failures, and runtime state.
- Queue approvals and expose full-project access status.
- Enforce path-root, traversal, symlink, ignored-state, and `.nexusdesk` protections.
- Record rollback snapshots for practical file mutations.

Build and CI:

- Build native Windows app through Fyne/CGO helper scripts.
- Stamp Windows executable icon resources.
- Validate build metadata through ldflags.
- Run cross-platform CI smoke for Windows, macOS, and Linux formatting, tests, static analysis, and Fyne build smoke.
- Track platform support in `docs/14_PLATFORM_SUPPORT.md`.

## Remaining Planned Functionality

Native parity and editor:

- Finish editable-widget inline syntax styling or deliberately accept the companion syntax mirror as the beta strategy.
- Decide the future LSP/deeper cross-file language action strategy.
- Continue go-to-definition, references, minimap/document-map, and formatting depth.

Assistant/source quality:

- Improve retrieval evidence beyond deterministic citation/source diagnostics.
- Add stronger source diagnostics for weak, stale, partial, or uncited context.
- Improve provenance consistency for all generated outputs.

Artifacts and generated outputs:

- Expand richer generated document artifacts.
- Add packaged presentation exports beyond the presentation-outline baseline.
- Extend regeneration coverage to those future output kinds.

Documents and OCR:

- Add OCR for images and scanned PDFs.
- Route OCR through durable cancelable jobs.
- Add broader document-set analysis with citations and comparison/version workflows.

Data and connectors:

- Design database dump import jobs using isolated temporary environments.
- Add connector sync jobs with cancellation, retry, credential handling, redaction, audit, and history.
- Add Google Analytics, ads-platform, CRM/contact-platform, and cross-source analysis workflows when the connector job model is ready.

Operations:

- Keep Docker/system mutation workflows blocked until approval, audit, rollback/mitigation, and durable jobs are mature enough.
- Preserve strict separation between read-only inspection and mutating operations.

Security/platform:

- Validate macOS Keychain and Linux Secret Service/libsecret behavior in platform packaging smoke.
- Extend audit coverage to future connector sync, OCR, dump import, shell, and Docker mutation flows.

Reliability and jobs:

- Apply the shared durable slow-workflow contract to OCR, dump imports, connector pulls, long indexing, long report generation, packaged exports, and long agent runs.
- Add crash/hang checks and issue-report bundles.
- Expand search index metadata and recovery/export flows.

Packaging and beta:

- Add signed Windows installer/release flow.
- Harden macOS and Linux packaging.
- Validate install/update/uninstall behavior.
- Add onboarding for workspace open, model setup, permissions, and local data policy.
- Add private beta release notes and feedback loop.

## Current Review Decision

`app-wails/` should not be retired yet. It remains useful for final editor behavior comparison, historical Wails workflows, and explicit parity decisions.

The active product path should remain migration-first and production-first. Avoid adding new top-level studios until Workbench, Data & Analytics, Artifacts, Settings, Assistant, diagnostics, and packaging feel coherent and reliable.

## Recommended Next Engineering Order

1. Apply the durable slow-workflow contract to concrete OCR, dump import, connector pull, long indexing, long report, and long agent run implementations.
2. Finalize the native editor parity strategy and explicitly document what counts for Native Parity Beta.
3. Expand richer document/presentation export workflows.
4. Build signed release packaging and installer/update validation.
5. Validate macOS Keychain and Linux Secret Service/libsecret behavior in platform packaging smoke.
6. Run a focused UI polish pass on onboarding, empty states, settings, diagnostics, and workflow hierarchy.
