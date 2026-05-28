# NexusDesk App Data And Uninstall Cleanup

Status: active packaging and support guidance

This guide documents where NexusDesk stores local data, what normal uninstall should remove, what should be retained as user data, and how to perform a full manual reset. It complements the [Clean-Machine Smoke Checklist](20_CLEAN_MACHINE_SMOKE_CHECKLIST.md).

## 1. Data Classes

NexusDesk has three broad data classes:

- Installed application files: binaries, app bundles, packaged resources, shortcuts, desktop entries, Start menu entries, and package-manager records.
- Global user configuration: settings, recent workspace history, connector profile metadata, assistant profile/memory, and protected-secret references.
- Workspace-local state: `.nexusdesk/` metadata, artifacts, approvals, rollbacks, jobs, issue reports, and compatibility-import state inside each workspace.

Normal uninstall should remove installed application files. It should not silently delete global user configuration, protected secrets, or workspace-local state unless the user explicitly chooses a full cleanup.

## 2. Global User Configuration

The active app uses the operating system user config directory.

Known config files:

- `NexusDesk/settings.json`: provider, model, context, and non-secret settings.
- `NexusDesk/settings.json.secret`: protected sidecar token or DPAPI-protected provider API-key data, depending on platform behavior.
- `NexusDesk/recent-workspaces.json`: recent workspace list.
- `NexusDesk/connector-profiles.json`: non-secret connector profile metadata.
- `NexusDesk/connector-profiles.json.secrets`: protected connector credential sidecar/token data.
- `NexusAugenticStudio/assistant-profile.json`: assistant memory and prompt profile settings, currently using the legacy product directory name for compatibility.

These files may contain private workspace paths, endpoint names, database names, usernames, assistant memory, and profile instructions. Treat them as user data.

## 3. Protected Secrets

Provider API keys and connector credentials display as redacted values.

Current secret behavior:

- Windows uses DPAPI-protected sidecar data.
- macOS uses Keychain through the `security` command.
- Linux uses Secret Service/libsecret through `secret-tool` when available.
- Unsupported secret storage must fail explicitly rather than store plaintext secrets silently.

Uninstallers and support scripts must not print secret values. Full cleanup instructions should mention both sidecar files and OS credential-store records because platform uninstallers may not remove Keychain or Secret Service entries automatically.

## 4. Workspace-Local State

Each opened workspace can contain `.nexusdesk/`.

Common contents include:

- `.nexusdesk/metadata/nexusdesk.sqlite`: SQLite metadata for chats, jobs, artifacts, approvals, SQL runs, dependencies, agent runs, tool runs, and history.
- `.nexusdesk/metadata/schema.sql`: metadata schema snapshot.
- `.nexusdesk/artifacts/`: generated reports, datasets, charts, notebooks, document exports, deck exports, comparisons, and sidecar metadata.
- `.nexusdesk/rollbacks/`: rollback snapshots for supported file mutations.
- `.nexusdesk/approvals/`: approval policy and compatibility approval logs.
- `.nexusdesk/issue-reports/`: exported redacted issue-report bundles.
- Other compatibility or backup files created by recovery/import/export flows.

Workspace-local state is intentionally project-local. It may be needed for audit, rollback, reproducibility, diagnostics, and generated work. Do not delete it automatically during app uninstall.

## 5. Normal Uninstall Expectations

Normal uninstall should remove:

- installed binaries or app bundles;
- packaged icons/resources;
- desktop entries, shortcuts, Start menu entries, or launch services records created by the installer;
- package-manager records where applicable.

Normal uninstall should document whether it keeps:

- global `NexusDesk/` config files;
- legacy `NexusAugenticStudio/assistant-profile.json`;
- protected OS credential-store entries;
- workspace `.nexusdesk/` directories.

If the installer offers "remove user data", it must clearly explain that this may remove settings, recent workspace history, connector profiles, assistant memory, protected credential references, generated artifacts, rollbacks, approvals, metadata history, and issue reports.

## 6. Full Manual Reset

To fully reset NexusDesk:

- Quit the app.
- Uninstall or remove the installed app package.
- Remove the `NexusDesk/` directory under the OS user config directory.
- Remove the legacy `NexusAugenticStudio/assistant-profile.json` file if present.
- Clear NexusDesk provider API-key and connector credential records from Windows DPAPI sidecars, macOS Keychain, or Linux Secret Service/libsecret as appropriate.
- Delete `.nexusdesk/` directories only from workspaces where you no longer need artifacts, rollbacks, approvals, jobs, issue reports, metadata, or audit history.

Before deleting `.nexusdesk/`, export or copy anything you may need later.

## 7. Upgrade And Backup Guidance

Before upgrading between beta builds:

- Read release notes.
- Export a workspace state backup from Diagnostics when practical.
- Export a redacted issue report if you are upgrading to reproduce a bug.
- Keep a copy of important generated artifacts outside `.nexusdesk/` if they are part of deliverables.

After upgrading:

- Confirm settings load.
- Confirm recent workspaces load.
- Confirm protected credentials still resolve or fail with clear remediation.
- Confirm connector profiles load.
- Confirm assistant profiles and memory load.
- Confirm workspace metadata, jobs, artifacts, approvals, and rollbacks are visible.
- Run Diagnostics if anything is missing.

## 8. Support Notes

Support and beta reports should never ask users to paste raw secret sidecars, Keychain exports, Secret Service records, or workspace `.nexusdesk/metadata/nexusdesk.sqlite` contents into public channels.

When debugging app data issues, prefer:

- redacted issue reports;
- screenshots with secrets hidden;
- file names and directory listings instead of file contents;
- synthetic workspaces;
- narrow exported backups reviewed by the user before sharing.
