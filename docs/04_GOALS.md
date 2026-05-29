# Goals

Status: canonical product goal and release bar.

## 1. One-Line Promise

NexusDesk is a native, local-first IDE and data studio with an integrated AI assistant that can see the workspace, cite sources, run safe tools, generate artifacts, and never take risky action without explicit user control.

## 2. Full Product Goal

Build NexusDesk into a production-ready desktop studio for people whose work crosses code, data, documents, databases, operations evidence, and AI-assisted development.

The finished app should let a user open a local project, understand it, edit it, inspect data, query databases, ask grounded questions, run approval-gated agent workflows, generate durable artifacts, inspect every action, and recover from problems without trusting hidden cloud infrastructure or invisible automation.

The target is not merely feature completeness. The target is a professional, trustworthy workbench:

- It launches cleanly on Windows, macOS, and Linux.
- It opens a workspace quickly and safely.
- It feels like a serious IDE/data studio, not a prototype.
- It gives the assistant broad but controlled tools.
- It makes sources, approvals, jobs, artifacts, and errors visible.
- It protects secrets and user files.
- It ships with signed or clearly trusted release artifacts.

## 3. Product Vision

A user opens NexusDesk and sees a calm JetBrains-style workbench. The left rail holds tools. The center is the editor. The right panel is the assistant. The status bar tells the truth. The app does not start hidden work just because a folder was opened.

The user can:

- navigate a local workspace;
- preview and edit files;
- search and inspect problems;
- use Git workflows without leaving the app;
- profile and query datasets;
- inspect SQLite and read-only external databases;
- generate charts, reports, notebooks, documents, presentations, runbooks, and chat-answer artifacts;
- ask the assistant questions grounded in selected files and data;
- let the agent plan and call deterministic tools;
- approve or deny risky actions;
- see every tool call and job;
- roll back file mutations;
- export redacted diagnostics when something goes wrong.

The app should be dense, fast, and quiet. It should reward keyboard use. It should not surprise the user.

## 4. Target Users

Primary users:

- Developers who want an AI-aware local workbench with code, Git, files, tasks, and safe tool use.
- Data analysts who need to inspect CSV, Excel, JSON, logs, SQLite, and external databases.
- Technical founders and operators who work across code, reports, dashboards, config, and operations evidence.
- DevOps-adjacent users who need read-only help understanding Dockerfiles, Compose files, environment files, logs, and runbooks.
- Privacy-conscious teams who want local or private-model workflows.
- Power users who want to choose models, inspect context, see tools, approve changes, and own outputs.

Not the target:

- users who want a cloud-only IDE;
- users who want the AI to mutate systems without review;
- users who want a simple chat app;
- users who want hidden background indexing and telemetry;
- teams that need multi-user collaboration in v1.

## 5. Release Goal For v1

A non-developer should be able to install NexusDesk on a clean Windows 11 machine, launch it, open a workspace, configure a local or OpenAI-compatible model endpoint, and complete the core flow without reading source code.

Core v1 flow:

1. Install and launch.
2. Open a workspace.
3. See the project tree and editor canvas immediately.
4. Open, edit, save, and revert a file safely.
5. Search the workspace.
6. Inspect Git status and diff.
7. Configure an AI provider/model.
8. Ask a grounded question with citations.
9. Run an agent action that requires approval.
10. Inspect the tool timeline and approval log.
11. Profile/query a dataset or SQLite file.
12. Generate an artifact.
13. Inspect artifact lineage/freshness.
14. Export diagnostics if needed.

## 6. Product Goals

### 6.1 Workbench goals

- Create a native desktop workbench with a stable, professional layout.
- Keep the editor canvas as the main surface.
- Keep the assistant always reachable.
- Keep tool windows one click away.
- Make project, data, artifacts, Git, tasks, jobs, approvals, diagnostics, and activity coherent.
- Preserve keyboard-first navigation.
- Make resize behavior strong enough for laptop and desktop use.

### 6.2 Editor goals

- Support safe preview/edit/save/revert for ordinary text/code files.
- Make partial/truncated previews impossible to save accidentally.
- Preserve cursor and scroll across save when practical.
- Provide find/replace, formatting, diagnostics, breadcrumbs, outline, definition, references, and document map.
- Keep large file handling bounded.
- Make encoding and line-ending state visible.
- Add deeper language intelligence only after the foundation is safe and performant.

### 6.3 Assistant goals

- Support local and OpenAI-compatible providers.
- Let users choose default models per task type.
- Stream answers smoothly.
- Ground answers in selected context.
- Show citations and source coverage.
- Warn on weak evidence.
- Save useful answers as artifacts.
- Keep chat history durable and inspectable.

### 6.4 Agent goals

- Give the agent a broad first-party toolbelt comparable to modern coding agents, but safer.
- Keep every tool deterministic and validated.
- Require approval for risky actions.
- Show plan, tool calls, observations, errors, and final answer.
- Bound iterations, wall-clock time, output size, and resource use.
- Make every mutation auditable and reversible where practical.
- Keep planned tools non-executable until safety design is done.

### 6.5 Data goals

- Treat datasets as first-class project assets.
- Profile common files quickly.
- Query local data safely.
- Provide SQL notebooks and chart/dashboard outputs.
- Inspect SQLite and read-only external databases.
- Keep database credentials protected.
- Default network database connections to encrypted transport.
- Move long data workflows into jobs.

### 6.6 Artifact goals

- Turn useful outputs into real files.
- Store metadata sidecars with source paths, timestamps, prompts, routes, models, and dependency information.
- Show freshness/staleness.
- Support preview, compare, archive, restore, delete, regenerate.
- Keep artifact mutations auditable and reversible where practical.
- Make generated DOCX/PPTX outputs polished enough for real use.

### 6.7 Operations goals

- Inspect operations evidence read-only.
- Generate runbooks from redacted evidence.
- Avoid system mutations in v1.
- Make future system mutation workflows job-backed, approval-gated, audited, and mitigated.

### 6.8 Safety goals

- Workspace open is cheap and safe.
- No surprise model calls.
- No surprise shell commands.
- No surprise Docker/system/database mutations.
- File mutations have previews and rollback where practical.
- Secrets use protected storage or explicit refusal.
- External calls are bounded, cancellable, redacted, and visible.
- Diagnostics explain failures and recovery.

### 6.9 Architecture goals

- Keep services framework-free.
- Keep the UI native and thin around service-owned behavior.
- Move slow work through jobs.
- Keep agent tools cataloged, bounded, auditable, and approval-gated by risk.
- Add tests near the package that owns behavior.
- Preserve import-boundary tests.
- Refactor the shell toward controllers so the UI can keep growing safely.

### 6.10 Packaging goals

- Provide repeatable builds.
- Produce release manifests and hashes.
- Add SBOM/provenance evidence.
- Sign Windows releases.
- Decide and implement macOS notarization/signing strategy.
- Provide Linux package strategy.
- Run clean-machine smoke on all supported platforms.
- Avoid leaving unsigned local build artifacts unless explicitly requested.

## 7. Success Metrics

| Area | Metric | Target |
|---|---|---|
| Workspace open | Time to visible navigator on 5,000-file repo | Under 1.5 s |
| Search | Time to first result on 2,000-file repo | Under 600 ms |
| Editor save | UI blocking time on 500 KB file | Under 50 ms, target non-blocking |
| Streaming | Visible stutter during 2,000-token response | None visible |
| Agent | Default wall-clock limit | 8 min or configurable |
| Mutation safety | File mutations with rollback or documented mitigation | 100% |
| Secret safety | Plaintext API keys/connector passwords on disk | 0 |
| Diagnostics | Crash marker visible on next launch | 100% |
| CI | Main branch tests/checks green | 100% before release |
| Release | Clean-machine smoke across target platforms | 100% before v1 |

## 8. Non-Goals For v1

- Cloud-hosted workspaces.
- User accounts.
- Team sync.
- Telemetry by default.
- Silent auto-update.
- Autonomous high-risk mutations.
- Production database mutation.
- Docker/system mutation.
- Plugin marketplace.
- Full multi-language IDE parity with dedicated language IDEs.
- Background indexing that starts on workspace open.
- Hidden model calls.

## 9. Principles

- Local-first: user files, chats, artifacts, and metadata live locally by default.
- Provider-agnostic: users can choose local or private endpoints and route tasks to different models.
- Tool-mediated: the LLM asks; deterministic services validate and act.
- Source-grounded: answers cite sources or warn when evidence is weak.
- Permissioned: risky actions require explicit user control.
- Reversible: file mutations can be rolled back where practical.
- Artifact-first: useful outputs become files with provenance.
- Explainable: every tool call, job, approval, and artifact is inspectable.
- Modular: services do not depend on the UI.
- Calm: density, clarity, and speed matter more than decoration.
