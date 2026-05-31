# NexusDesk Beta Install Report Template

Use this template once per beta tester/install. The five-user beta install test is complete only when five separate reports exist and every reported issue has been triaged within 48 hours.

Before closing the related tracker rows, save the completed reports and verify them from the repo root:

```powershell
.\nexus-app\scripts\verify-release-validation-reports.ps1 -BetaInstallReport path\to\beta-1.md,path\to\beta-2.md,path\to\beta-3.md,path\to\beta-4.md,path\to\beta-5.md -RequireFiveBetaReports
```

## Tester And Build

- Tester:
- Date and local time:
- Platform and OS version:
- Architecture:
- Machine type:
- Fresh install or upgrade:
- Artifact filename:
- Artifact SHA-256:
- App version:
- Commit:
- Build date:
- Signing/notarization/package trust state:

## Install Experience

- Install/unpack path:
- First launch path:
- Trust prompts or warnings:
- Antivirus/reputation warning:
- Start Menu/Desktop/App launcher integration:
- Time to first usable window:
- Crash or hang observed:
- Notes:

## Core Flow Result

| Flow | Result | Notes |
|---|---|---|
| Launch and About metadata |  |  |
| Open workspace |  |  |
| Preview/edit/save/revert file |  |  |
| Search workspace |  |  |
| Git/status/diff visibility if repo has Git |  |  |
| Provider setup and Test connection |  |  |
| Ask with source context and citations |  |  |
| Low-risk Agent with approval/audit/jobs |  |  |
| Dataset/SQLite profile or query |  |  |
| Artifact generation and preview |  |  |
| Diagnostics report |  |  |
| Redacted issue-report export |  |  |
| Uninstall/remove app files |  |  |
| User data retention understood |  |  |

## Feedback

- Top positive signal:
- Top confusing moment:
- Top blocker:
- Data-loss concern:
- Security/trust concern:
- Accessibility concern:
- Performance concern:
- Documentation gap:
- Would use again for a trusted local workspace: yes / no / unsure

## Triage

- Report received at:
- Triage due by:
- Triage completed at:
- Owner:
- Issues filed:
- Fix/defer/not-planned decision:
- User-facing release-note update needed: yes / no
- Closed within 48 hours: yes / no
