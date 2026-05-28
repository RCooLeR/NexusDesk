# Project Review

Date: 2026-05-27

This review records the current state after the Wails-to-Fyne migration work. It is intentionally blunt: `app-wails/` remains valuable as a reference, but the active product is `nexus-app/`.

## Executive Summary

The product direction still makes sense. Nexus Augentic Studio should stay a native local-first IDE/data/document/operations workbench with an always-visible AI assistant, not a chatbot with side panels. The Fyne migration is the right architectural move because it removes the Wails bridge, generated frontend bindings, and webview lifecycle problems that caused blank or gray windows.

The new app is no longer a thin skeleton. `nexus-app/` now has real native services for workspace navigation, previews, safe file mutation, editor tabs, search, problems, Git, tasks, jobs, approvals, LLM chat, agent tooling, metadata, artifacts, datasets, SQLite inspection, document extraction, operations scanning, and history. The most important remaining work is not to invent new studios yet; it is to finish parity and make the native UI feel like a serious IDE-class product.

Approximate Wails-to-Fyne migration status: 92-93% of useful Wails-era backend/workflow capability has been migrated. The remaining gap is concentrated in IDE-grade editor behavior, deeper retrieval evidence, future artifact regeneration coverage, slow-work job routing, production packaging, and UI polish.

The production release path is tracked in `docs/13_PRODUCTION_READINESS.md`.

## Architecture Health

Healthy areas:

- Root structure is clean: the active module keeps executable code under `nexus-app/internal/`.
- UI-independent behavior mostly lives in `internal/services/*`; Fyne widgets live under `internal/ui/*`.
- Workspace open is still cheap by design. Git, Docker, OCR, connector pulls, model calls, dump imports, and shell commands are explicit user actions.
- Safe file operations, safe writes, rollback records, approvals, and agent mutation tools share service boundaries instead of duplicating filesystem rules in widgets.
- Data and artifact services are split by use case and covered by focused tests.
- The native app has a real SQLite metadata layer, compatibility importers, and history surfaces rather than relying only on JSON sidecars.

Risks to watch:

- `internal/ui/shell` is split into many files, which is good, but it still carries a lot of orchestration state. Future UI work should extract smaller controllers/models before adding deeper editor, connector, and assistant behavior.
- Several preserved Wails docs still describe active behavior using `app/internal` and React/Wails terms. They must be treated as reference-history until rewritten.
- The native editor is functional but not yet IDE-grade. Native outline, searchable go-to-symbol, local go-to-definition, jumpable Document Map, first lightweight Syntax tab, read-only highlighted syntax preview, breadcrumbs, explicit save encoding controls, deterministic Go/JSON format actions, safe Markdown/config/SQL/Dockerfile/text whitespace formatting, Wails-style secondary split preview selection, and live find match counts have started, while active-editor inline syntax styling and future LSP/deeper cross-file language actions still need deliberate follow-through.
- External database profile flows and Windows credential vault behavior have native parity baselines; connector sync jobs are still future work.
- Long-running work is only partially routed through durable jobs. OCR, dump imports, connector pulls, report generation, indexing, and long agent runs must not be wired directly to UI callbacks.

## Native Capability Snapshot

Implemented in `nexus-app/`:

- Native Fyne shell with brand icon/logo, menu, shortcuts, resizable assistant/sidebar and bottom workbench panes.
- Workspace tree with lazy loading, ignored-path controls, file operations, context menus, and Git badges from manual refresh.
- File previews for text/code, Markdown, images, CSV/TSV, DOCX, PDF text, XLSX-derived rows, and binary metadata.
- Editor tab lifecycle with pinned ordering, dirty markers, safe save, revert, and explicit discard confirmation for modified tabs.
- Search and Problems panels using bounded preview-safe reads.
- Git status/diff panel with directory-grouped changes, unified/split/diff-only views, hunk navigation, file-level stage/unstage, and hunk stage/unstage.
- Task discovery/run jobs for npm, Go tests, Python pytest, Cargo tests, and Docker Compose config validation.
- Data profiling/query/SQL/notebooks for CSV, TSV, JSON, NDJSON, XLSX, logs, Parquet metadata, and SQLite files.
- Chart/dashboard SVG preview and artifact generation.
- Artifact browser with metadata, lineage, comparison, archive/delete/restore, source freshness, context pinning, and job-routed workspace scan reports.
- Document extraction/report artifacts for Markdown, TXT, HTML, XML, DOCX, XLSX, and PDF preview text.
- Operations inspection for Dockerfiles, Compose, env/config/script/log files, Compose topology, safe config validation, and runbook artifacts.
- Settings, LLM transport, streaming Ask mode, Agent mode, deterministic tools, approval queue, full-project access policy, rollback browser, history, audit, source/model answer diagnostics, evidence-quality labels, line-aware citation refs, and durable metadata.

## Main Gaps

Priority migration gaps:

1. Native editor quality: active-editor inline syntax styling and future LSP/deeper cross-file language-aware navigation.
2. Native editor-adjacent artifact regeneration and source quality: future regeneration coverage and deeper retrieval/source evidence UX.
3. Job routing for slow workflows: long indexing, OCR, dump imports, connector pulls, report generation, and long agent runs.
4. Connector and dump workflows: temporary isolated database sandboxes, import lifecycle, storage limits, and read-only analysis.
5. Assistant maturity: model context accounting, runtime diagnostics, source-citation quality, and broader tool coverage.
6. UI polish: reduce crowded action strips, use structured tabs/dialogs/split panes, improve density, align with JetBrains-class workbench expectations, and add native visual checks.
7. Documentation cleanup: continue separating active Fyne behavior from preserved Wails history.

## Review Decision

Do not retire `app-wails/` yet. It still contains reference implementations for React/Monaco editor behavior and several mature Wails-era workflows.

Do not add new top-level studios yet. Keep the primary product surfaces as Workbench, Data & Analytics, Artifacts, and Settings with Assistant always visible. Documents, Operations, and analytics connectors should grow as capability domains inside those surfaces until they earn deeper native screens.

## Next Engineering Direction

The next batches should stay migration-first:

1. Keep closing the Wails-only feature inventory instead of adding new top-level studios.
2. Finish native editor parity and UI structure.
3. Add deeper assistant retrieval evidence and future artifact regeneration coverage.
4. Add macOS Keychain and Linux Secret Service/libsecret after the Windows protected-secret baseline.
5. Route remaining slow workflows through durable jobs.
6. Add dump import design and first safe job scaffold before any database mutation/import execution.
7. Add diagnostics and release-readiness checks before private beta.
