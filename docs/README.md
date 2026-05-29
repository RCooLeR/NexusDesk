# NexusDesk Documentation

Status: canonical source of truth for NexusDesk product direction, architecture, user experience, delivery plan, and execution tracker.

NexusDesk is a Fyne-native, local-first AI workbench for code, data, documents, databases, artifacts, operations evidence, and safe agent-assisted work. The product target is a professional JetBrains-style desktop studio: compact, dark, keyboard-friendly, source-grounded, auditable, and safe by default.

## Canonical Documents

Read these in order:

1. [Architecture](01_ARCHITECTURE.md): system shape, package boundaries, service ownership, storage, safety model, and maintainability rules.
2. [UI Workbench](02_UI_WORKBENCH.md): the exact target workbench layout, visual language, interaction rules, resizing behavior, and screen acceptance criteria.
3. [Features](03_FEATURES.md): implemented capabilities, planned capabilities, agent tools, data surfaces, artifacts, diagnostics, and intentional non-goals.
4. [Goals](04_GOALS.md): the full product goal, release bar, success metrics, principles, and user promise.
5. [Production Plan](05_PLAN.md): phases, milestones, gates, sequencing, validation standards, and risk management.
6. [Execution Tracker](06_TRACKER.md): checkboxes for every known step needed to finish the project end to end.

## Documentation Rules

- These files replace scattered planning notes. If a plan is not reflected here, it is not the active plan.
- Keep documentation aligned with the active Fyne-native app under `nexus-app/`.
- Do not add obsolete runtime references, removed app paths, or historical migration narratives.
- Keep architecture claims grounded in code or explicitly label them as planned work.
- Keep tracker items small enough that one focused development slice can complete them with tests.
- When implementation changes behavior, update the relevant document and the tracker in the same slice.

## Current Product Direction

NexusDesk should become a production-ready desktop studio where a user can:

- open a local workspace without hidden side effects;
- inspect, edit, search, and safely mutate files;
- work with datasets, spreadsheets, SQLite, and read-only external databases;
- ask an AI assistant grounded in selected source context;
- run an approval-gated agent with deterministic local tools;
- generate reports, charts, notebooks, documents, decks, runbooks, and chat-answer artifacts;
- inspect every job, tool call, approval, source, and generated output;
- recover from crashes, export redacted diagnostics, and trust the release package.

The final app should feel serious, dense, calm, and native. The workbench is the product. The assistant is integrated into the workbench, not bolted onto it.
