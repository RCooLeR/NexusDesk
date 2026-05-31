# NexusDesk Release Blocker Disposition

Date: 2026-05-31
Status: current open P0/P1 release-blocker review.

This document records the remaining release blockers after CI package smoke and documentation reconciliation. It does not close any tracker item by itself; a row closes only when the required evidence exists in `docs/06_TRACKER.md`.

Use `docs/releases/clean-machine-smoke-report-template.md` or `.github/ISSUE_TEMPLATE/clean-machine-smoke.yml` for every clean-machine smoke run. Use `docs/releases/beta-install-report-template.md` or `.github/ISSUE_TEMPLATE/beta-install-report.yml` for every beta install report. Completed Markdown reports must pass `nexus-app/scripts/verify-release-validation-reports.ps1` before the related tracker rows are closed.

## Current State

- Tracker progress: 324 of 374 complete, 86.6%.
- Open P0: 3.
- Open P1: 12.
- Latest package-smoke CI evidence: hosted native CI run 26707224450 passed Windows, Linux, and macOS on commit `4d71dda`.
- Latest blocker documentation refresh: commit `05f4cf8`.

## P0 Disposition

| Tracker item | Disposition | Evidence required to close |
|---|---|---|
| Close all P0 issues | Open; depends on the rows below. | All remaining P0 rows closed with evidence. |
| Review all P1 issues and explicitly defer or fix | Open; no remaining P1 is safely deferrable for a production release without an explicit release-owner decision. | Each open P1 either fixed with evidence or marked deferred in tracker/release notes with owner approval and user-visible risk text. |
| Run full platform smoke | Open; CI package smoke exists, but clean-machine smoke is not complete. | Windows, macOS, and Linux clean-machine smoke reports covering install/launch/workspace/editor/assistant/data/artifact/diagnostics, plus any platform-specific trust prompts. |

## P1 Disposition

| Tracker item | Current disposition | Evidence required to close |
|---|---|---|
| Sign Windows executable | Open; signing hooks and JSON evidence sidecar generation exist, but certificate-backed signing has not run. | Signed `nexusdesk.exe`, valid Authenticode verification, timestamp evidence, artifact SHA-256, manifest/SBOM/provenance, and signer identity recorded. |
| Sign Windows installer | Open; installer script signing hooks and JSON evidence sidecar generation exist, but certificate-backed signing has not run. | Signed installer scripts or chosen installer artifact, valid Authenticode verification, timestamp evidence, artifact SHA-256, manifest/SBOM/provenance, and signer identity recorded. |
| Windows clean-machine launch smoke | Open; CI installer smoke is not a clean machine. | Fresh Windows 11 user/VM install, normal launch path, About metadata, icon/window title, first-run Home readiness, trust prompts recorded. |
| Windows open workspace smoke | Open. | Fresh Windows workspace-open run with project tree, recent workspace, quick-open/search basics, and no hidden side effects recorded. |
| Windows edit/save/revert smoke | Open. | Fresh Windows editor run showing preview, edit, save, dirty guard, rollback/revert, and retained file contents evidence. |
| Windows assistant setup smoke | Open. | Fresh Windows provider setup using local/test endpoint, Test connection result, model suggestion behavior, protected-secret status, and diagnostics visibility. |
| Windows data/artifact smoke | Open. | Fresh Windows CSV/JSON/XLSX or SQLite profile/query, artifact generation, artifact preview/readback, lineage/freshness visibility. |
| Windows diagnostics/export smoke | Open. | Fresh Windows Diagnostics run and redacted issue-report export with no default workspace contents or secrets. |
| macOS clean-machine smoke | Open; CI app-bundle smoke is not clean-machine Gatekeeper validation. | Fresh macOS account/VM package verification, quarantine/Gatekeeper/signing state, launch, workspace/editor/assistant/data/artifact/diagnostics smoke, Keychain behavior. |
| Linux clean-machine smoke | Open; CI package smoke is not desktop/runtime validation. | Fresh supported Linux distro install/unpack, runtime dependency notes, Wayland/X11 launch, desktop entry/icon behavior, workspace/editor/assistant/data/artifact/diagnostics smoke, Secret Service behavior. |
| Run five-user beta install test | Open; requires external beta users or machines. | Five separate beta install reports with OS, artifact hash, trust prompt state, core-flow outcome, diagnostics notes, and retained app-data expectations. |
| Triage beta feedback within 48 hours | Open; depends on beta feedback arriving. | Beta install reports and feedback log showing every issue reviewed, labeled, fixed/deferred, or documented within 48 hours. |

## Do Not Close From Automation Alone

The packaged `--smoke-check` command is strong CI evidence, but it is not a substitute for the clean-machine rows. It does not prove desktop launch integration, OS trust prompts, antivirus/reputation behavior, file-picker behavior, protected-secret integration under a real user session, screen-reader behavior, or upgrade/uninstall expectations outside the scripted install path.
