# NexusDesk Beta Feedback And Release Notes Guide

Status: active private-beta feedback and release-note process

This guide defines how private beta users and maintainers should report issues, read release notes, and keep feedback safe. It complements the [Safe Agent User Guide](18_SAFE_AGENT_USER_GUIDE.md) and the [Production Readiness Plan](13_PRODUCTION_READINESS.md).

## 1. Feedback Goals

Beta feedback should help the team answer:

- What user goal was blocked?
- What exact action caused the issue?
- What was expected?
- What actually happened?
- Was the issue reproducible?
- Did the app create a job, approval, rollback, artifact, SQL run, chat record, or agent-audit record?
- Was data safety, credential safety, or user trust affected?
- Is there an acceptable workaround?

## 2. Highest Priority Reports

Report these first:

- Startup failure, crash, hang, blank window, or broken clean-machine launch.
- Data loss, missing rollback, unsafe file mutation, or unexpected workspace write.
- Unapproved shell/system/database/Docker action.
- Secret leakage in UI, logs, diagnostics, artifacts, or issue reports.
- Packaging, installer, update, uninstall, signing, or antivirus false-positive failure.
- Stuck, uncancelable, duplicated, or misleading job.
- Provider/model failure with unclear remediation.
- Misleading assistant citation, missing source warning, or fabricated source claim.
- External connector behavior that violates read-only, bounded, cancelable, or redacted expectations.

## 3. Before Filing A Report

- Check Home readiness for workspace, provider/model, credential, and toolchain state.
- Run Diagnostics when possible.
- Inspect Jobs for progress, cancellation, retry state, and logs.
- Inspect Agent Audit for tool calls and observations.
- Inspect History for related chats, artifacts, jobs, and agent runs.
- Inspect Rollbacks after file mutations.
- Try the smallest safe reproduction if the issue is not destructive.

## 4. What To Include

Include:

- App version, commit, build date, and operating system.
- Whether the issue happened in Workbench, Editor, Assistant, Agent, Data, Artifacts, Tasks, Jobs, Diagnostics, Settings, or packaging.
- Provider type and local/cloud endpoint category, without secrets.
- The exact visible error message after redaction.
- The action that triggered the issue.
- Whether a job, approval, rollback, artifact, SQL run, chat record, or agent-audit record exists.
- Whether retry, cancel, export, rollback, or restart helped.
- A redacted issue-report bundle from Diagnostics when possible.

## 5. What Not To Include

Do not include:

- API keys, passwords, bearer tokens, cookies, DSNs, connector credentials, SSH keys, signing keys, or recovery tokens.
- Raw external system logs before checking for credentials, headers, query strings, and customer data.
- Production data or private prompts.
- Workspace files unless the report explicitly requires them and you have reviewed the contents.
- Generated artifacts that contain private workspace or customer information unless intentionally shared.

## 6. Redacted Issue Reports

Use Diagnostics to export a redacted issue report when possible.

Default issue reports should include:

- diagnostics text;
- activity tail;
- app/runtime metadata;
- relevant metadata filenames and state summaries;
- no workspace file contents unless explicitly requested.

If workspace content is required:

- open the generated bundle before sharing;
- remove anything sensitive;
- prefer tiny synthetic reproduction workspaces over real projects.

## 7. Release Notes Policy

Every private-beta release note should separate:

- New capabilities.
- Fixed issues.
- Safety and trust changes.
- Packaging/platform changes.
- Migration or compatibility notes.
- Known limitations.
- Validation performed.
- Required user action.

Release notes should call out:

- new agent tools or changed approval behavior;
- provider/model/credential changes;
- connector changes;
- file mutation or rollback changes;
- packaging/signing/update/uninstall changes;
- known data-loss, startup, packaging, or trust risks.

## 8. Private-Beta Release Checklist

Before tagging a beta release:

- `go test ./...` passes.
- Native Fyne build passes on the target platform.
- Release manifest validates artifact name, size, SHA256, platform, version, commit, and build date.
- Generated binaries are not accidentally committed.
- Smoke checklist is updated for the release focus.
- Known blockers and limitations are listed.
- Feedback intake path is visible in release notes.
- Diagnostics issue-report export is working.
- Safe Agent Guide and in-product Help entries match current behavior.

## 9. Feedback Triage Labels

Use clear labels in issue trackers or release notes:

- `critical`: data loss, secret leakage, unsafe mutation, startup failure, or broken installer.
- `high`: blocked core workflow, stuck job, missing rollback, misleading safety state, or reproducible provider failure.
- `medium`: confusing UI, incomplete diagnostics, weak citation/source quality, or non-blocking packaging issue.
- `low`: visual polish, wording, minor keyboard/focus issue, or documentation improvement.

## 10. Closeout Expectations

A beta issue should not be considered closed until:

- the user-visible behavior is fixed or explicitly documented as unsupported;
- tests or manual smoke coverage exist where practical;
- release notes mention the fix if users may have encountered it;
- tracker and readiness docs reflect any changed risk or remaining work.
