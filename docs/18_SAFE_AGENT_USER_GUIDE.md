# NexusDesk Safe Agent User Guide

Status: active user guidance for private beta readiness

This guide explains how to use NexusDesk's assistant and agent features safely. It is intentionally practical: the goal is to help users understand what the app can do, where approvals matter, where data stays local, and what to check when a run fails.

## 1. Start From A Trusted Workspace

- Open only project folders you trust.
- NexusDesk keeps folder open bounded and cheap.
- Opening a folder must not automatically run Git, Docker, shell commands, model calls, OCR, connector pulls, dump imports, or deep indexing.
- Use the Home readiness cockpit before long agent work. It surfaces workspace, provider/model, credential, and native build-toolchain gaps.
- Use Diagnostics when provider, metadata, job, or runtime behavior looks wrong.

## 2. Control Context Explicitly

- Ask mode and Agent mode work from selected workspace context, pinned files, pinned directories, artifacts, and relevant chat history.
- Pin only the files or directories that should influence the answer.
- Review source diagnostics before acting on an answer.
- Treat weak, stale, uncited, or unverified source warnings as a signal to inspect the cited files yourself.
- Prefer smaller, specific prompts when asking the agent to modify files.

## 3. Approvals And Risky Actions

- Agent tools are bounded and recorded.
- File mutations, high-risk workspace changes, shell-like task execution, connector work, and future system mutations must stay behind explicit approvals, audit records, and rollback or mitigation paths.
- Do not approve an action unless the target path, proposed operation, and reason are understandable.
- If an approval looks too broad, deny it and ask the assistant to narrow the request.
- Never approve a mutation because the assistant sounds confident. Approve only the concrete operation you are willing to run.

## 4. Rollbacks And Recovery

- Supported file writes create rollback records where practical.
- Use the Rollbacks panel to inspect and restore model-authored file changes.
- Jobs, agent runs, tool runs, artifacts, SQL runs, approvals, and chat messages are persisted as local metadata for later inspection.
- Use History, Jobs, Agent Audit, Diagnostics, and Rollbacks together when investigating failures.
- If a run fails halfway through, inspect what completed before retrying.

## 5. Local Data And Secrets

- NexusDesk is local-first.
- Workspace contents are not included in issue reports unless explicitly requested.
- Exported issue bundles redact secrets by default.
- Provider API keys and connector credentials use protected OS storage where available and display as redacted values in the UI.
- Avoid pasting API keys, passwords, bearer tokens, DSNs, or production credentials into prompts, files, SQL text, notebook cells, or artifact descriptions.
- Treat generated artifacts as workspace data. Review them before sharing outside the workspace.

## 6. Connectors And Databases

- External database work defaults to bounded, read-only, single-statement inspection.
- Query execution should remain cancelable, audited, and redacted in errors.
- Do not connect production systems until profile scope, credentials, query limits, and export expectations are clear.
- Mutation workflows should remain unavailable until their approval, audit, job, and rollback design is complete.
- If connector errors include secrets or DSNs, treat that as a bug and export a redacted issue report.

## 7. Slow Work And Jobs

- Long operations should appear as jobs with progress, logs, cancellation, retry, and output-opening paths.
- Folder open must never trigger slow or external work.
- Use Jobs to confirm whether work is still running before rerunning a task.
- If a workflow feels stuck, check Jobs, Diagnostics, Agent Audit, and History before retrying.
- Future OCR, connector sync, dump import, long indexing, report generation, long agent runs, shell workflows, and Docker mutations must continue through the durable job model.

## 8. Practical Private-Beta Checklist

- Open a trusted workspace.
- Confirm Home readiness shows the expected workspace, model/provider, credentials, and toolchain state.
- Open Settings and run provider connection checks before relying on model output.
- Pin only the files/directories/artifacts needed for the task.
- Read citations and source diagnostics before accepting an answer.
- Approve only bounded, understandable mutations.
- Check Rollbacks after file mutations.
- Use Diagnostics and redacted issue reports when reporting a problem.
- Do not share workspace content, generated artifacts, credentials, or local metadata unless you intentionally choose to.

## 9. What To Report

When reporting a beta issue, include:

- The platform and app version/build metadata.
- Whether the issue happened during Ask, Agent, Data, Artifacts, Tasks, Jobs, or Diagnostics.
- The visible error message after redaction.
- Whether a job, approval, rollback, artifact, or agent-audit record was created.
- A redacted issue-report bundle from Diagnostics when possible.

Do not include:

- API keys, passwords, tokens, DSNs, private prompts, or production data.
- Workspace files unless the report explicitly requires them and you have reviewed the contents.
