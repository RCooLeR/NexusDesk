# NexusDesk Contributor Setup And Standards

Date: 2026-05-28

This guide defines the default contributor workflow for NexusDesk while the active product is the Fyne-native `nexus-app/`. It complements [Internal Package Ownership](23_INTERNAL_PACKAGE_OWNERSHIP.md), which maps package responsibility in detail.

## Active Product

- Develop in `nexus-app/`.
- Use `app-wails/` only as a reference implementation until the explicit freeze/archive milestone is complete.
- Port useful Wails-era behavior capability-by-capability.
- Do not add Wails, webview, React bridge, generated frontend, or browser-runtime dependencies to the active Fyne app.

## Local Setup

### Windows

Use the bundled scripts whenever possible:

```powershell
cd nexus-app
.\scripts\dev-env.ps1 -Test
.\scripts\dev-env.ps1 -BuildCheck
.\scripts\dev-env.ps1 -Build
.\scripts\dev-env.ps1 -Run
```

Windows Fyne builds require CGO and a C compiler. The current development path expects MSYS2 UCRT64 GCC under:

```text
C:\msys64\ucrt64\bin
```

The helper script configures `PATH`, `CGO_ENABLED=1`, readonly module flags, local Go cache/temp paths, build metadata, and Windows icon stamping. Prefer `-BuildCheck` for routine validation because it writes the unsigned executable to a temporary folder and removes it immediately. Use `-Build` only when you intentionally need a local runnable artifact in `nexus-app/build/`.

### macOS And Linux

- Install a Go toolchain compatible with the module.
- Install the native compiler/toolchain required by Fyne/CGO.
- Validate platform-specific secret storage during packaging smoke: macOS Keychain and Linux Secret Service/libsecret via `secret-tool`.
- Keep unsupported secret-storage behavior explicit rather than silently writing raw credentials.

## Verification Loop

Run focused tests while developing:

```powershell
cd nexus-app
go test ./internal/services/userguide ./internal/ui/shell
```

Run full validation before committing:

```powershell
cd nexus-app
gofmt -w <changed-go-files>
go test ./...
.\scripts\dev-env.ps1 -BuildCheck
cd ..
git diff --check
```

If you intentionally run `go build .` or `.\scripts\dev-env.ps1 -Build`, remove generated binaries such as:

```text
nexus-app/nexusdesk.exe
nexus-app/nexusdesk
nexus-app/build/nexusdesk.exe
```

Unsigned local Windows builds can trigger Norton, SmartScreen, or other reputation-based scanners on every commit because each build has a fresh hash and no production signing reputation. Do not ask users to disable antivirus globally. For local development, prefer `-BuildCheck`, keep generated binaries out of the source tree, and reserve signed CI/release artifacts for sharing outside the development machine.

## Coding Standards

- Keep `main.go` thin.
- Put business rules in services first.
- Keep domain and services framework-free.
- Keep Fyne imports in `internal/app`, `internal/ui`, `internal/ui/theme`, `internal/brand`, and UI tests only.
- Keep workspace open cheap: no Git, Docker, OCR, connector pulls, dump imports, model calls, shell commands, or deep indexing on folder open.
- Do not write into `.nexusdesk` through generic file-mutation paths.
- Preserve approvals, audit, rollback or mitigation, and redaction for risky workflows.
- Prefer focused package helpers and tests over growing `internal/ui/shell` orchestration.
- Do not reintroduce arbitrary shell execution; discovered safe tasks and future shell policies are separate workflows.

## Test Standards

Every logical milestone should add or update focused tests unless it is strictly documentation-only.

Test priorities:

- rooted path safety and traversal rejection;
- ignored path and `.nexusdesk` protection;
- symlink and binary/text boundaries;
- approval, rollback, audit, and redaction behavior;
- job cancellation, retry, retention, output, and metadata persistence;
- SQL/query mutation blocking and connector credential handling;
- artifact metadata, lineage, freshness, regeneration, and archive/restore behavior;
- LLM streaming parsing, context escaping, source/citation diagnostics, and agent tool argument parsing;
- UI model helpers, command-palette entries, guide rendering, and shell behavior that can be tested without launching the desktop window.

Integration tests should use small deterministic fixtures and avoid starting external services unless the test name and package explicitly document that dependency.

## Documentation Standards

Update documentation in the same milestone when behavior changes:

- `tracker.md` for task-level execution state;
- `docs/17_END_TO_END_PRODUCTION_PLAN.md` for broad roadmap and JetBrains-like product target;
- `docs/13_PRODUCTION_READINESS.md` for release gates;
- `docs/15_WAILS_FEATURE_INVENTORY.md` when Wails parity decisions change;
- `docs/23_INTERNAL_PACKAGE_OWNERSHIP.md` when package responsibilities change.

In-product Help guides live in `nexus-app/internal/services/userguide` and should have focused tests. Expose important production/contributor guides through the Help menu and command palette.

## ADR Process

Use an Architecture Decision Record when a change affects:

- package boundaries;
- persistence formats;
- security or approval policy;
- connector credentials or query behavior;
- artifact formats or generated-output compatibility;
- job model semantics;
- packaging/signing/distribution;
- plugin, MCP, extension, or third-party code execution.

Create ADRs under `docs/adr/` using the template in [ADR Index](adr/README.md).

ADR files should be named:

```text
NNNN-short-title.md
```

Required sections:

- Status;
- Date;
- Context;
- Decision;
- Consequences;
- Validation;
- Follow-ups.

Prefer reversible decisions and document migration or rollback expectations when persistence, generated artifacts, or user data are affected.

## Commit Discipline

- Commit and push only clean logical milestones.
- Use direct commit messages, for example `Document contributor standards`, `Harden native artifact regeneration`, or `Polish diagnostics health cards`.
- Do not amend commits unless explicitly requested.
- Do not force-push.
- Do not revert unrelated user changes.
- If unexpected unrelated changes appear, stop and ask how to proceed.

## Pre-Commit Checklist

- `git status --short --branch` reviewed.
- Relevant docs and tracker updated.
- Focused tests passed.
- `gofmt` applied.
- `go test ./...` passed.
- `.\scripts\dev-env.ps1 -BuildCheck` passed, or a documented platform build check passed.
- Generated binaries removed.
- `git diff --check` passed.
- Commit message describes one logical milestone.
