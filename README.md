# NexusDesk

NexusDesk is a Fyne-native, local-first AI workbench for code, data, documents, artifacts, operations, and agent-assisted development.

The app should feel like a professional desktop IDE/data studio: JetBrains-style three-column workbench, compact top menu/toolbar, left tool stripe plus project/database tool window, large central editor canvas, right AI chat panel, subtle status bar, low-noise dark theme, and strong resize behavior.

## What It Does

- Open and browse local workspaces safely.
- Preview and edit text/code with rollback-backed safe writes.
- Search files, inspect problems, navigate symbols, definitions, and references.
- Work with Git status, diffs, hunks, staged commits, history, and blame.
- Use an integrated Ask/Agent assistant with local context, citations, source diagnostics, approvals, and audit.
- Run first-party tools for files, search, Git, tasks, jobs, data, SQLite, documents, artifacts, operations, and approved terminal commands.
- Profile/query datasets and inspect read-only database connections.
- Generate charts, dashboards, reports, DOCX/PPTX outputs, runbooks, task reports, and answer artifacts.
- Persist chats, jobs, approvals, artifacts, SQL runs, dataset dependencies, and agent/tool audit records locally.
- Export diagnostics and keep risky actions permissioned, auditable, redacted, and reversible where practical.

## Documentation

- [Architecture](docs/01_ARCHITECTURE.md)
- [JetBrains-Style UI Workbench](docs/02_UI_WORKBENCH.md)
- [Features](docs/03_FEATURES.md)
- [Goals](docs/04_GOALS.md)
- [Plan](docs/05_PLAN.md)
- [Tracker](docs/06_TRACKER.md)

## Project Layout

```text
nexus-app/            Active Fyne-native Go desktop app
nexus-app/internal/   App, domain, services, UI, theme, and brand packages
docs/                 Canonical architecture, UI, features, goals, plan, and tracker docs
services/             Local development helper services
tracker.md            Pointer to the canonical tracker in docs/
```

Generated runtime state such as `.nexusdesk/`, build output, local executables, and dependency folders are ignored.

## Core Principles

- Local-first: user files, chats, tool logs, and generated artifacts stay local by default.
- Provider-agnostic: users configure model providers, routes, context windows, and capabilities.
- Tool-mediated: the assistant requests actions; services validate and perform them.
- Source-grounded: answers cite files, rows, queries, logs, documents, or tool outputs.
- Permissioned: risky writes, terminal, database, operations, and future system actions require approval.
- Artifact-first: durable outputs become files with metadata and lineage.
- Explainable: every tool call, generated artifact, approval, and mutation should be inspectable.
- Modular: services own behavior; UI renders intent and results.

## Current Focus

1. Finish the JetBrains-style native shell: thin rails, compact left-sidebar tool windows, large center editor, right assistant hierarchy, status-only bottom bar, and resizing safety.
2. Harden long-session performance: streaming/activity throttling, async editor save/diff/rollback, metadata contention fixes.
3. Complete signed packaging and clean-machine smoke for Windows, macOS, and Linux.
4. Polish DataGrip-like data workflows, artifact regeneration, settings, diagnostics, and source quality.
5. Keep architecture boundaries strict: framework-free services, native UI only in app/UI/theme/brand, slow work through jobs, risky actions through approval/audit/rollback/redaction.
