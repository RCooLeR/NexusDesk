# NexusDesk Clean-Machine Smoke Checklist

Status: active release-candidate smoke checklist

This checklist defines the minimum manual validation for a release candidate on a fresh Windows, macOS, or Linux machine. It complements the CI matrix and release manifest checks; it does not replace signed installer, notarization, package-manager, or update validation.

## 1. Preflight

- Use a clean VM, clean user profile, or machine with previous NexusDesk data removed unless this is explicitly an upgrade test.
- Do not rely on the source tree, developer shell profile, local build cache, or repository-relative assets.
- Record operating system version, architecture, installer/package type, app version, commit, build date, artifact SHA256, and signing/notarization/package trust state.
- Keep the release notes, Safe Agent Guide, Beta Feedback Guide, and this checklist available during the run.
- Prepare a tiny trusted sample workspace with code, Markdown, JSON, CSV/JSON/XLSX data, and one safe task script if practical.

## 2. Install And Launch

- Install or unpack the release artifact using the normal user path for the platform.
- Launch from the normal desktop/start/menu entry, not from a developer terminal unless the platform requires it.
- Verify the app icon, window title, About metadata, and release version.
- Verify Home readiness renders without an open workspace.
- Confirm missing provider/model/toolchain guidance is understandable and does not crash.
- Confirm Help opens Safe Agent Guide, Beta Feedback & Release Notes, and Clean-Machine Smoke Checklist.

## 3. Workspace And Editor Smoke

- Open the sample workspace with the native folder dialog.
- Verify project tree rendering, lazy folders, recent workspace entry, refresh, quick open, and file preview.
- Open Markdown and a supported code/config file.
- Verify dirty markers, save, revert, close guard, rollback record, find/replace, formatting where supported, syntax mirror, document map, Problems, and search.
- Confirm folder open did not automatically run Git, Docker, shell, OCR, connector pulls, dump imports, model calls, or deep indexing.

## 4. Assistant And Safety Smoke

- Open Settings, configure a local or test provider, and run Test connection.
- Verify provider/model failures produce actionable guidance.
- Run one Ask request with pinned context and verify citations, source diagnostics, and response cancellation behavior.
- Run one low-risk Agent request and verify approvals, Jobs, Agent Audit, History, and Rollbacks are understandable.
- Deny at least one risky approval and verify the app reports the denial clearly.
- Confirm issue-report export redacts secrets and excludes workspace file contents by default.

## 5. Data, Artifacts, Jobs, And Diagnostics Smoke

- Profile a small CSV/JSON/XLSX file.
- Run a bounded data query or notebook cell.
- Verify Data grid rendering, row/cell copy, result tabs, and chart/artifact output where applicable.
- Refresh Artifacts and preview at least one generated artifact.
- Regenerate one safe artifact if source metadata exists.
- Inspect Jobs for completed/canceled/failed state behavior.
- Run Diagnostics and verify provider, metadata, job, runtime, and app-log sections are readable.
- Export a redacted issue report and inspect the ZIP contents before deleting it.

## 6. Platform-Specific Smoke

### Windows

- Verify executable icon/resource metadata and About metadata.
- Verify protected API-key storage through Windows DPAPI behavior.
- Verify install/uninstall path, Start menu/Desktop entries if provided, and app data retention/cleanup notes.
- Record SmartScreen, signing, or antivirus false-positive behavior.
- Verify the app launches without requiring a developer MSYS2 shell.

### macOS

- Verify app launch permissions, quarantine behavior, signing/notarization status, and Gatekeeper prompts.
- Verify protected API-key storage through Keychain behavior.
- Verify app bundle icon, About metadata, menu behavior, and app data cleanup notes.

### Linux

- Verify package dependencies, desktop entry/icon behavior, Wayland/X11 launch behavior, and About metadata.
- Verify Secret Service/libsecret behavior or explicit unsupported-secret refusal.
- Verify app data cleanup notes for the chosen package format.

## 7. Upgrade, Uninstall, And Cleanup

- For upgrade tests, install over a previous beta and confirm settings, recent workspaces, metadata, protected secrets, chat history, artifacts, jobs, approvals, and rollback records remain readable or migrate with clear messaging.
- Uninstall or remove the app through the platform path.
- Verify expected app files are removed.
- Document retained user data locations and manual cleanup steps.
- Reinstall after uninstall and verify the app launches cleanly.

## 8. Closeout

- Attach the release manifest and artifact SHA256 to the smoke record.
- Record which workflows passed, failed, were skipped, or require platform follow-up.
- Confirm release notes mention validation coverage, known limitations, required user actions, and platform-specific risks.
- File issues for any crash, data loss, missing rollback, missing approval, unredacted secret, stuck job, misleading citation, packaging failure, installer/update/uninstall failure, or antivirus false-positive.
- Do not mark the release candidate ready until every critical/high smoke failure is fixed or explicitly documented as a blocker.
