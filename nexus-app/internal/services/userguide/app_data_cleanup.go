package userguide

func AppDataCleanupGuide() Guide {
	return Guide{
		Title:   "App Data And Uninstall Cleanup",
		Summary: "Where NexusDesk stores local data, what uninstallers should remove, and what users may need to clean manually.",
		Sections: []Section{
			{
				Title: "Global App Configuration",
				Body: []string{
					"NexusDesk stores global settings under the operating system user config directory. The active paths include NexusDesk/settings.json, NexusDesk/recent-workspaces.json, NexusDesk/connector-profiles.json, and the legacy-named NexusAugenticStudio/assistant-profile.json profile store.",
					"These files contain preferences, recent workspace paths, connector profile metadata, and assistant memory/profile settings. They should be treated as user data and should not be removed by an app upgrade.",
				},
			},
			{
				Title: "Protected Secrets",
				Body: []string{
					"Provider API keys and connector credentials display as redacted values. Windows uses DPAPI-protected sidecar data; macOS uses Keychain; Linux uses Secret Service through secret-tool when available.",
					"Uninstallers must not dump secret values into logs. Cleanup instructions should mention both file sidecars and OS credential-store entries because platform uninstallers may not remove Keychain or Secret Service records automatically.",
				},
			},
			{
				Title: "Workspace State",
				Body: []string{
					"Each opened workspace can contain .nexusdesk/ state with SQLite metadata, schema files, artifacts, rollbacks, approvals, issue reports, task reports, data artifacts, and compatibility import markers.",
					"Workspace .nexusdesk/ is project-local user data. App uninstallers should not delete it automatically because it may contain rollback records, generated artifacts, audit records, and issue-report bundles the user still needs.",
				},
			},
			{
				Title: "What Uninstall Should Remove",
				Body: []string{
					"Normal uninstall should remove installed binaries, app bundles, shortcuts, desktop entries, Start menu entries, and packaged application resources.",
					"Normal uninstall should document whether user config, recent workspace lists, assistant profiles, connector profiles, protected secrets, and workspace .nexusdesk/ state are retained.",
				},
			},
			{
				Title: "Manual Cleanup",
				Body: []string{
					"To fully reset the app, remove the NexusDesk config directory from the OS user config location, remove the legacy NexusAugenticStudio assistant-profile directory if present, clear NexusDesk entries from the OS credential store, and delete .nexusdesk/ directories from workspaces you no longer need.",
					"Before deleting .nexusdesk/, export or review artifacts, rollbacks, approvals, issue reports, and metadata backups. Deleting workspace state can remove recovery and audit history.",
				},
			},
			{
				Title: "Upgrade And Backup Guidance",
				Body: []string{
					"Before upgrading between beta builds, export a workspace state backup from Diagnostics when practical and keep release notes available.",
					"After upgrading, verify settings, recent workspaces, protected credentials, connector profiles, assistant profiles, metadata, jobs, artifacts, approvals, and rollbacks still load or show clear migration guidance.",
				},
			},
		},
	}
}

func AppDataCleanupMarkdown() string {
	return FormatMarkdown(AppDataCleanupGuide())
}
