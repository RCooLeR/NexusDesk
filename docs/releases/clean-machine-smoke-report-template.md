# NexusDesk Clean-Machine Smoke Report Template

Use this template once per clean Windows, macOS, or Linux machine. A clean-machine smoke is valid only when it starts from a fresh user profile or clean VM without a NexusDesk source checkout, developer environment variables, or previous NexusDesk app data unless the run is explicitly marked as an upgrade test.

Before closing the related tracker rows, save the completed report and verify it from the repo root:

```powershell
.\nexus-app\scripts\verify-release-validation-reports.ps1 -CleanMachineReport path\to\windows-report.md,path\to\macos-report.md,path\to\linux-report.md -RequireAllCleanMachinePlatforms
```

## Run Identity

- Tester:
- Date and local time:
- Platform: Windows / macOS / Linux
- OS version and build:
- Architecture:
- Machine type: physical / VM / cloud VM
- Fresh profile or clean VM: yes / no
- Upgrade test: yes / no
- Artifact filename:
- Artifact SHA-256:
- Manifest/SBOM/provenance sidecars present: yes / no
- App version:
- Commit:
- Build date:
- Signing/notarization/package trust state:
- Trust prompts, Gatekeeper prompts, antivirus/reputation prompts, or package-manager warnings:

## Preflight Evidence

- Artifact hash matches manifest: pass / fail / not checked
- About/version output matches manifest: pass / fail / not checked
- Packaged smoke command run: pass / fail / not checked
- Packaged smoke command:
- Packaged smoke workspace path:
- Packaged smoke JSON/output archived: yes / no

Expected command shape:

```text
nexusdesk --version
nexusdesk --smoke-check <empty-test-workspace>
```

Platform paths:

- Windows: run from the installer target directory.
- macOS: run `NexusDesk.app/Contents/MacOS/nexusdesk --smoke-check <empty-test-workspace>`.
- Linux: run `bin/nexusdesk --smoke-check <empty-test-workspace>` from the unpacked package.

## Manual UI Smoke

Record pass/fail plus notes for each item.

| Area | Result | Notes/evidence |
|---|---|---|
| Normal launch path opens NexusDesk |  |  |
| App icon/window title visible |  |  |
| Help > About metadata matches manifest |  |  |
| Home readiness renders with no workspace |  |  |
| Open trusted sample workspace |  |  |
| Project tree and recent workspace update |  |  |
| Quick open/search basics work |  |  |
| Preview supported text/Markdown file |  |  |
| Edit, save, dirty guard, rollback/revert |  |  |
| Configure local/test provider |  |  |
| Test connection and model suggestion behavior |  |  |
| Protected-secret backend status visible |  |  |
| Ask workflow with pinned context and citations |  |  |
| Low-risk Agent workflow shows approval/audit/jobs |  |  |
| Profile/query small CSV/JSON/XLSX or SQLite sample |  |  |
| Generate artifact and inspect preview/lineage/freshness |  |  |
| Run Diagnostics |  |  |
| Export redacted issue report |  |  |
| Issue report excludes workspace contents by default |  |  |
| Uninstall/remove app files |  |  |
| Expected user data locations retained/documented |  |  |

## Platform-Specific Evidence

### Windows

- Windows version/build:
- Installer path used:
- Start Menu shortcut created: yes / no
- DPAPI protected-secret smoke: pass / fail / not checked
- Authenticode signature state:
- Antivirus/reputation prompt state:
- Uninstall result:

### macOS

- macOS version/build:
- Package path used:
- Quarantine attribute present before first launch: yes / no / not checked
- Gatekeeper prompt state:
- Keychain protected-secret smoke: pass / fail / not checked
- Codesign verification result:
- Notarization/stapling state:
- App cleanup result:

### Linux

- Distribution/version:
- Desktop/session: Wayland / X11 / headless
- Package path used:
- Runtime dependency issues:
- Desktop entry/icon behavior:
- Secret Service/libsecret behavior:
- Unsupported-secret refusal behavior if no keyring exists:
- App cleanup result:

## Result

- Overall result: pass / fail
- Blocks release: yes / no
- Failed checklist rows:
- Logs, screenshots, diagnostics bundle paths:
- Follow-up issue links:
- Reviewer:
- Release owner sign-off:
