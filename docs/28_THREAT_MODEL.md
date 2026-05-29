# NexusDesk Threat Model And Control Matrix

Status: Production-readiness control baseline for native Fyne app, agent tools, jobs, connectors, and generated artifacts.
Last updated: 2026-05-29.

This document turns NexusDesk's safety posture into an implementation checklist. The source code mirror lives in `nexus-app/internal/services/security` so future tools can share the same risk vocabulary instead of relying only on prose.

## Non-Negotiable Rules

- Workspace open must never start Git, Docker, shell, browser automation, connector pulls, OCR, dump imports, model calls, or deep indexing.
- Services remain framework-free. Fyne imports stay in UI/app/theme packages only.
- Planned tools must not appear as executable agent tools until validation, approvals, audit, redaction, cancellation, caps, and UI affordances exist.
- High-risk workflows require explicit user visibility and either rollback or a documented mitigation when rollback is impossible.
- Secrets must stay in protected storage or redacted memory paths; issue reports must not include workspace content unless explicitly requested.

## Control Vocabulary

| Control | Meaning |
| --- | --- |
| `approval` | User must explicitly approve the risky call or a bounded policy grant must already cover it. |
| `audit` | Action, target, actor, risk, and outcome must be recorded durably where practical. |
| `rollback-or-mitigation` | File mutations need rollback snapshots; irreversible/system actions need a mitigation note and recovery guidance. |
| `redaction` | Logs, observations, diagnostics, and issue reports must scrub secrets/tokens/credentials. |
| `rooted-scope` | Local paths must stay inside the selected workspace or an explicitly approved safe target. |
| `connector-scope` | Remote actions must be limited to configured profile scopes and must never expose secrets. |
| `timeout` | Calls must have bounded runtime. |
| `cancellation` | Slow or long-running calls must be cancelable. |
| `output-cap` | Observations, logs, previews, and extracts must be size-capped. |
| `durable-job` | Slow work must run as a job with status, logs, cancellation, and output linkage. |
| `secret-isolation` | Credentials must stay in protected storage or connector-specific secret boundaries. |
| `artifact-lineage` | Generated outputs must record source, prompt/query/tool/job metadata where practical. |
| `sandbox` | Browser, Docker, MCP/plugin, and dump-import workflows need isolation before execution. |
| `no-workspace-open` | Workflow must never be triggered automatically by opening or refreshing a workspace. |
| `preview` | Mutating or expensive workflows need a preview/dry-run/manifest where practical. |
| `user-visible-status` | The UI must show status/progress/failure and next recovery action. |

## Risk Defaults

- Low risk: rooted scope and output caps.
- Medium risk: rooted scope, approval, audit, timeout, cancellation, output caps, and redaction.
- High risk: medium controls plus rollback-or-mitigation and user-visible status.

These defaults are exposed in the native tool catalog so planned high-risk tools show their required control groups before implementation.

## Threat Families

| Family | Scope | Required emphasis |
| --- | --- | --- |
| Workspace filesystem | Reads, writes, patches, deletes, generated files, rollback-backed mutations | rooted scope, preview, approval, audit, rollback, redaction |
| External connectors and databases | Database profiles, analytics/CRM/cloud connectors, remote API actions | connector scope, protected secrets, durable jobs, redaction, lineage |
| Durable slow workflows | OCR, dump imports, connector pulls, reports, long indexing, long agent runs, packaged exports | durable job, no workspace-open trigger, cancellation, output caps, status |
| Terminal and shell execution | Approved one-shot commands and future interactive sessions | approval, audit, rooted cwd, durable jobs, redaction, cancellation |
| Rendered browser automation | Navigation, clicks, typing, screenshots, extraction, network logs | sandbox, URL policy, approval for data entry, screenshots/artifact lineage |
| Docker and system operations | Compose config/logs/lifecycle, containers, images, volumes | approval, sandbox/mitigation, durable jobs, no automatic execution |
| Generated artifacts and media | Reports, charts, DOCX/PPTX, images, regenerated artifacts, packaged outputs | provenance, output caps, preview, approval for writes, lineage |
| MCP and plugins | Third-party tool discovery, MCP calls, plugin install/execution | signed/verified trust, sandbox, approval, audit, secret isolation |

## Implementation Gate

Before any planned tool or workflow becomes executable, it needs:

1. Framework-free service implementation.
2. Scope validation: rooted local path, connector profile, or sandbox session.
3. Risk classification and controls from `internal/services/security`.
4. Explicit approval policy for medium/high-risk actions.
5. Timeout, cancellation, and output caps.
6. Redaction for logs, observations, issue reports, and persisted metadata.
7. Audit/lineage records for actions and generated outputs.
8. Rollback or mitigation for mutations where practical.
9. Focused tests for success, denial, invalid inputs, traversal/scope violations, timeout/cancel, and redaction.
10. UI affordance for preview, progress, cancellation, and recovery when the user needs to decide or inspect.
11. Tracker and production-plan updates.

## Current Status

Implemented foundations:

- Path-root and symlink safety across workspace reads/writes/search/context.
- Approval queue, full-project access policy, and per-call approval for high-risk agent mutations.
- Rollback-backed workspace/file mutations where practical.
- Durable jobs with cancellation/logs/output access for implemented slow workflows.
- Protected provider and connector credentials.
- Diagnostics, metadata backup, issue-report export, and job persistence warning visibility.
- Native tool catalog now annotates implemented/planned tools with risk-derived control requirements.
- Artifact provenance diagnostics now check generated outputs for readable metadata sidecars and lineage signals before release-candidate sharing.

Still planned:

- Connector sync job model and connector-specific audit coverage.
- OCR/scanned-image pipeline.
- Isolated dump-import workflow.
- Interactive terminal sessions.
- Rendered browser automation.
- Docker lifecycle workflows.
- MCP/plugin discovery and execution security model.
- Signed release and platform packaging smoke.
