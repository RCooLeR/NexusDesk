# NexusDesk Beta Release Notes

Date: 2026-05-30
Channel: private beta candidate

## What This Beta Is For

This beta is for validating the native Fyne desktop workbench before public distribution. Focus on startup, workspace open, provider setup, safe Ask/Agent workflows, local data inspection, artifacts, diagnostics, release evidence, and uninstall/app-data expectations.

Use trusted sample workspaces first. Do not test with production secrets, customer data, or destructive database/system workflows.

## New And Ready To Exercise

- Native first-run flow on Welcome with Provider Setup, Sample Workflow, workspace/file open, and Diagnostics entry points.
- Provider Setup Wizard with endpoint selection, detected model suggestions, protected credential guidance, Test connection, Diagnostics, and task-route setup.
- Settings model auto-suggestion from provider model probes when the selected model is blank or missing.
- Sample Workflow Guide for a safe end-to-end beta path across edit/revert, Ask, Agent, Data, Artifacts, Jobs, History, and Diagnostics.
- Known Limitations page covering packaging, provider/model setup, planned tools, connectors, platform coverage, and secret backends.
- Release evidence generation: manifest, CycloneDX SBOM, and provenance JSON sidecars.
- Release trust diagnostics in the Diagnostics report.
- Redacted beta feedback issue template.

## Install And Trust State

- Windows native build validation passed in the iteration 140 checkpoint, including gofmt, tests, vet, build metadata validation, native build, manifest, SBOM, provenance, and cleanup.
- Windows installer bundle generation is available through `nexus-app/scripts/package-windows-installer.ps1`; the bundle includes install/uninstall PowerShell scripts, the Windows payload zip, and installer-level manifest/SBOM/provenance sidecars.
- Windows installer uninstall/app-data behavior has a scripted smoke path through `nexus-app/scripts/smoke-windows-installer.ps1`; the smoke verifies app files are removed and workspace `.nexusdesk/` data is preserved.
- Public Windows code signing and installer signing are not complete.
- macOS signing/notarization and macOS package smoke are not complete.
- Linux package strategy is documented, but Linux package smoke is not complete.
- Treat unsigned beta artifacts as private test builds. Verify artifact SHA256 against the release manifest and keep the SBOM/provenance sidecars with the artifact.

## Validation To Run

- Launch the app on a clean user profile and verify Help > About Nexus shows the expected version, commit, and build date.
- Open Provider Setup Wizard, configure a test provider, run Test connection, and confirm model suggestion behavior.
- Open a trusted sample workspace and run the Sample Workflow Guide.
- Run Diagnostics and confirm provider, protected secrets, release trust, artifact provenance, jobs, metadata, and startup recovery sections are understandable.
- Export a redacted issue report after any failure.
- Uninstall or remove the app and record which app data, protected secrets, recent workspaces, and workspace `.nexusdesk/` state remain.

## Known Limitations

- Planned tools are roadmap-only until their design, approval, audit, rollback, and tests are complete.
- External database work is bounded and read-only; mutation workflows remain unavailable.
- Public packaging is not production-ready until signing/notarization/package strategy and clean-machine smoke are complete across target platforms.
- Provider answers still require source review. Weak, stale, uncited, or missing sources should block trust in the result.
- Linux protected-secret support depends on a usable Secret Service backend; missing backends should fail clearly.

## Feedback

Use the beta feedback issue template. Include app version, commit, build date, OS, provider, model, affected area, expected result, actual result, reproduction steps, and redacted diagnostics.

Do not include API keys, bearer tokens, DSNs, connector credentials, private workspace files, production logs, customer data, or screenshots with secrets.
