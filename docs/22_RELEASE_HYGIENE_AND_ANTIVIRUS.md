# NexusDesk Release Hygiene And Antivirus Notes

Status: active production-readiness guidance for release candidates and private-beta builds.

This document defines how NexusDesk should prepare, verify, communicate, and support release artifacts so users can trust the native app. It complements the production gates in [Production Readiness Plan](13_PRODUCTION_READINESS.md), the clean-machine smoke checklist in [Clean-Machine Smoke Checklist](20_CLEAN_MACHINE_SMOKE_CHECKLIST.md), the app data cleanup guide in [App Data And Uninstall Cleanup](21_APP_DATA_AND_UNINSTALL_CLEANUP.md), and the generated release manifest checks in `nexus-app/scripts`.

## 1. Release Artifact Rules

- Release artifacts must come from CI or the documented release pipeline, not from ad-hoc local builds.
- The source tree must be clean before packaging.
- The artifact must match a known commit and the release notes must name that commit.
- The app About dialog, build metadata, release manifest, and release notes must agree on version, commit, build date, platform, and architecture.
- The release manifest must record artifact name, platform, version, commit, build date, size, and SHA256.
- Generated binaries such as local `nexusdesk.exe` files must not be committed.
- Debug-only files, local logs, temporary package directories, secrets, test workspaces, and developer machine paths must not be bundled.

## 2. Signing And Platform Trust

Windows:

- Use the planned Windows code-signing path before public release.
- Expect Microsoft SmartScreen reputation to improve only after signed releases build download reputation.
- Record whether a release was unsigned, test-signed, or production-signed.
- If an installer is used, verify publisher identity, icon, version metadata, install location, uninstall entry, and retained app data behavior.

macOS:

- Use signing and notarization before public release.
- Record quarantine behavior, Gatekeeper prompts, notarization status, and Keychain-backed secret behavior.
- Verify the app bundle metadata, icon, version, and app data cleanup expectations.

Linux:

- Use the chosen package trust model once selected: archive, AppImage, deb/rpm, Flatpak, or repository packaging.
- Document desktop entry/icon behavior, runtime dependencies, Wayland/X11 launch behavior, and Secret Service/libsecret behavior or explicit unsupported-secret refusal.

## 3. Antivirus False-Positive Triage

False positives are a release-quality issue, not something users should be asked to work around blindly.

When a scanner flags a build, record:

- artifact file name;
- SHA256 and size;
- NexusDesk version, commit, build date, platform, and architecture;
- signing/notarization state;
- scanner or vendor name;
- detection name;
- download or distribution URL;
- whether the same CI pipeline reproduces the exact hash;
- whether a clean rebuild from the same commit changes the result;
- whether the scanner flags the unpacked app, installer, or both.

Recommended response:

- Verify the artifact hash against the release manifest.
- Run the clean-machine smoke checklist again.
- Inspect the package contents for unexpected files.
- Submit the artifact to the vendor false-positive portal when appropriate.
- Publish a release-note update that explains the current validation state.
- Rebuild, re-sign, and republish only when there is a real packaging/signing reason to do so.

Never:

- ask users to disable antivirus globally;
- tell users to ignore a high-confidence detection before triage is complete;
- repack or obfuscate binaries to evade a scanner;
- publish a binary whose source commit, manifest, or signing state is unknown.

## 4. Runtime Behaviors That Reduce Suspicion

NexusDesk should preserve trust by keeping potentially suspicious behavior explicit and auditable:

- Opening a workspace must not run Git, Docker, shell commands, OCR, connector pulls, dump imports, deep indexing, model calls, or background network activity.
- Git operations remain manual and visible.
- Shell-like task execution must use discovered safe tasks, jobs, logs, cancellation, and audit.
- File mutations must stay rooted, approval-aware where applicable, and rollback-backed where practical.
- Connector/database access must be user-started, bounded, read-only by default, redacted, cancelable, and auditable.
- Issue-report bundles must redact secrets and exclude workspace contents unless the user explicitly includes them.
- App data paths, secret storage, upgrade behavior, uninstall behavior, and manual cleanup must be documented.

## 5. Release Note Requirements

Every release candidate or private-beta release note should include:

- version, commit, build date, platform, architecture, artifact names, and SHA256 values;
- signing/notarization/package trust state;
- validation commands or CI workflow summary;
- clean-machine smoke coverage and any skipped items;
- known trust prompts, antivirus findings, or false-positive submissions;
- installer/update/uninstall behavior;
- app data retention and cleanup notes;
- protected-secret storage status per platform;
- known limitations and required user actions;
- support instructions for verifying artifact hashes and reporting scanner flags.

## 6. Do Not Ship If

Block the release when any of these are true:

- The artifact was built from an unknown commit.
- The release manifest is missing, invalid, or does not match the artifact hash/size.
- The About dialog metadata does not match release notes.
- The build comes from a dirty worktree or undocumented local packaging step.
- Clean-machine smoke fails on the target platform.
- A high-confidence malware detection is untriaged.
- The package contains secrets, local developer paths, test data, logs, or debug-only files.
- Installer/update/uninstall behavior is unknown for the release channel.
- Protected-secret storage or explicit unsupported-platform refusal is unverified for the target platform.

## 7. Current Status

- Native CI already validates formatting, tests, vet, CGO/Fyne builds, build metadata, release manifests, and whitespace checks.
- Windows icon stamping and release manifest generation exist.
- Clean-machine smoke, app data cleanup, safe agent guidance, and beta feedback docs are available in the app Help menu and command palette.
- Signed packaging, installer/update validation, macOS signing/notarization, and Linux package strategy remain open production milestones.
