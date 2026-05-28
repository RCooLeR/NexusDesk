# Platform Support Matrix

This document defines the production support stance for the Fyne-native `nexus-app/` release path. It is intentionally conservative: Windows is the first supported desktop target, while Linux and macOS remain build-investigation targets until packaging, protected secrets, and visual smoke coverage are proven.

## Current Support Policy

| Platform | Status | Release Promise | Build Path | Notes |
| --- | --- | --- | --- | --- |
| Windows 10/11 x64 | Primary beta target | Supported first for local beta builds and release candidates | `nexus-app/scripts/dev-env.ps1 -Build` with CGO enabled and MSYS2 UCRT64 available | App icon resource generation is part of the helper path. Code signing, installer/update flow, and antivirus hygiene remain release gates. |
| Linux x64 | Investigation target | Not supported for end users yet | Future CI smoke using CGO-enabled Fyne build dependencies | Needs package/runtime dependency notes, Secret Service/libsecret decision, visual smoke, and artifact path verification. |
| macOS Apple Silicon / Intel | Investigation target | Not supported for end users yet | Future CI smoke using CGO-enabled Fyne build dependencies | Needs notarization/signing plan, Keychain-backed secret storage decision, app bundle packaging, and visual smoke. |

## Windows Release Gate

Windows can move from primary beta target to supported release target only when:

- `go test ./internal/domain ./internal/services/... ./internal/ui/shell ./internal/ui/theme ./internal/brand` passes in CI and locally.
- The Fyne app builds from a clean checkout through the documented helper.
- The executable carries the approved icon and version metadata.
- Manual smoke covers workspace open, quick-open, editor save/revert, assistant settings/probe, agent approvals, data preview/query, artifacts, jobs, Git, tasks, operations, rollback, and diagnostics.
- Protected secret storage has a Windows implementation or a clearly refused fallback for secret-bearing features.
- The release process includes signing, installer/update strategy, and antivirus false-positive notes.

## Linux And macOS Investigation Plan

The first non-Windows pass should be a build-smoke project, not a support promise.

1. Add CI jobs that install Fyne build prerequisites and run package-level tests.
2. Add smoke builds without release artifacts.
3. Document runtime/package dependencies discovered by the smoke builds.
4. Implement explicit unsupported-platform behavior for protected secrets before enabling saved connector credentials.
5. Add manual visual smoke checklists for native menus, dialogs, file pickers, keyboard shortcuts, charts, PDFs/images, and long-running jobs.
6. Decide whether each platform can enter beta, stay experimental, or remain source-only.

## CI Matrix Target

The intended matrix is:

- `windows-latest`: tests, formatting/static checks, Windows Fyne build smoke, icon/version metadata validation.
- `ubuntu-latest`: tests first, then CGO/Fyne build smoke once dependencies are documented.
- `macos-latest`: tests first, then app bundle build smoke once signing/notarization requirements are documented.

Until the Linux/macOS build-smoke jobs exist, user-facing release notes should state that only Windows builds are actively supported.
