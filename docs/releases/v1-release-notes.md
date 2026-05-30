# NexusDesk v1 Release Notes

Date: 2026-05-30
Channel: v1 release candidate notes

## Release Status

NexusDesk v1 is a Fyne-native local desktop workbench for code, data, documents, databases, artifacts, operations evidence, and safe AI-assisted work.

These release notes are published for the current v1 release candidate. They are not a claim that production distribution is complete. Public production release remains blocked until the tracker closes signing, platform packages, CI, clean-machine smoke, protected-secret smoke, beta validation, and final P0/P1 disposition.

## Highlights

- Native Fyne workbench with a compact editor-centered layout, left tool windows, right assistant, status bar, menus, command palette, first-run onboarding, and crash-recovery entry points.
- Safe workspace open policy with no hidden model calls, shell work, connector pulls, deep indexing, Docker/system work, OCR, browser automation, dump processing, or task runs.
- File preview/edit/save/revert flow with large-file partial-preview blocking, safe write proposals, rollback snapshots, ambiguous-encoding warnings, visible save state, and bounded diffs.
- Workspace search with streaming bounded reads, binary/container skips, cancellation, result caps, and partial UI updates.
- Source-grounded assistant with model routing, provider setup, context budget visibility, source coverage warnings, chat history, artifact save paths, and non-modal manual update guidance.
- Approval-gated deterministic agent tools with audit records, mutation verification, timeouts, cancellation, output caps, redaction, and planned high-risk tools kept non-executable until design approval and tests exist.
- Data workbench for CSV/JSON/XLSX/SQLite/read-only external database inspection, query history, result grids, exports, transport-state visibility, and connector pool diagnostics.
- Artifacts for reports, charts, notebooks, DOCX/PPTX-oriented outputs, runbooks, freshness checks, lineage, regeneration, package validation, and destructive-flow rollback.
- Jobs, approvals, agent audit, activity, diagnostics, redacted issue reports, protected-secret backend status, release trust diagnostics, and app-data cleanup guidance.
- Windows zip and installer packaging scripts with manifest, CycloneDX SBOM, provenance, SHA-256 verification, and installer smoke support.

## Validation Published With This Candidate

- Security/safety review: `docs/releases/security-safety-review.md`.
- Performance review: `docs/releases/performance-review.md`.
- Accessibility review: `docs/releases/accessibility-review.md`.
- v1 scope freeze: `docs/releases/v1-scope-freeze.md`.
- Private beta notes and feedback guidance: `docs/releases/beta-release-notes.md`.
- Windows release evidence verifier: `nexus-app/scripts/verify-release-evidence.ps1`.
- Windows installer smoke script: `nexus-app/scripts/smoke-windows-installer.ps1`.

## Install And Trust State

- Windows native build and installer packaging are available from repository scripts.
- Release evidence sidecars are generated next to artifacts: manifest, SBOM, and provenance.
- Artifact hashes must be checked against the release manifest before use.
- Windows executable and installer signing are not complete.
- macOS package/signing/notarization work is not complete.
- Linux package work is not complete.
- Unsigned or freshly built artifacts may trigger operating-system or antivirus reputation warnings. Treat those prompts as release risks and verify artifacts with manifest/SBOM/provenance evidence.

## Known Limitations

- Browser automation, interactive terminal sessions, PR platform tools, MCP/plugin invocation, scheduled automations, semantic search, connector sync jobs, image/screenshot understanding, and Docker/system mutation remain post-v1 or planned until their safety designs and tests are complete.
- External database mutation is not a v1 feature.
- Provider answers can be incomplete or wrong when source coverage is weak, stale, uncited, or outside the model context budget.
- Linux protected-secret support depends on an available Secret Service backend.
- App-specific text zoom beyond Fyne/system scaling has not been validated as a supported v1 control.
- Platform-specific clean-machine smoke is incomplete until the tracker says otherwise.

## Required User Actions

- Read these notes and the beta notes before upgrading.
- Verify the artifact SHA-256 against the manifest.
- Keep the manifest, SBOM, and provenance sidecars with the artifact.
- Use trusted sample workspaces for first validation.
- Do not test with production secrets, production customer data, or destructive database/system workflows.
- Export a redacted diagnostics report when filing feedback.

## Remaining Release Blockers

The release candidate is not complete while these tracker items remain open:

- Close all remaining P0 issues.
- Review all remaining P1 issues and explicitly defer or fix them.
- Run full test suite in CI.
- Run full platform smoke.
- Produce macOS and Linux artifacts.
- Sign Windows executable and installer.
- Complete macOS signing/notarization if macOS remains in the supported target set.
- Complete Linux/macOS CI and package smoke.
- Complete cross-platform protected-secret smoke.
- Complete Windows/macOS/Linux clean-machine smoke.
- Complete five-user beta install validation and feedback triage.
