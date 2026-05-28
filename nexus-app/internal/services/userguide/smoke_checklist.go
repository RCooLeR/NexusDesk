package userguide

func CleanMachineSmokeChecklistGuide() Guide {
	return Guide{
		Title:   "Clean-Machine Smoke Checklist",
		Summary: "A release-candidate checklist for validating NexusDesk on a fresh Windows, macOS, or Linux machine without source-tree assumptions.",
		Sections: []Section{
			{
				Title: "Preflight",
				Body: []string{
					"Start from a fresh user profile or clean virtual machine with no source checkout, no developer environment variables, and no previous NexusDesk app data unless the test is explicitly an upgrade test.",
					"Record operating system version, architecture, installer/package type, app version, commit, build date, artifact SHA256, and whether code signing/notarization or package trust prompts appear.",
				},
			},
			{
				Title: "Install And Launch",
				Body: []string{
					"Install or unpack the release artifact using the documented path for the platform.",
					"Launch NexusDesk from the normal user entry point, verify About metadata, verify the app icon/window title, and confirm Home readiness renders without a workspace.",
				},
			},
			{
				Title: "Workspace And Editor",
				Body: []string{
					"Open a small trusted sample workspace, verify the project tree, recent workspace list, quick open, preview, search, Problems, and safe text edit/save/rollback flows.",
					"Open Markdown, JSON, Go or another supported code file, verify syntax mirror, document map, find/replace, formatting where supported, dirty markers, close guards, and split preview basics.",
				},
			},
			{
				Title: "Assistant And Safety",
				Body: []string{
					"Open Settings, configure a local or test provider, run Test connection, and verify provider/model failures produce understandable remediation.",
					"Run one Ask workflow with pinned context, inspect citations/source diagnostics, then run one low-risk Agent workflow and confirm approvals, audit, jobs, and rollback expectations.",
				},
			},
			{
				Title: "Data, Artifacts, Jobs, And Diagnostics",
				Body: []string{
					"Profile a small CSV/JSON/XLSX sample, run a bounded query or notebook cell, and verify Data grids, copy behavior, result tabs, and artifact output.",
					"Refresh Artifacts, preview/regenerate a safe artifact, inspect Jobs and History, run Diagnostics, and export a redacted issue report with no workspace contents by default.",
				},
			},
			{
				Title: "Platform-Specific Checks",
				Body: []string{
					"Windows: verify icon/resource metadata, DPAPI-protected secret persistence, UCRT64/CGO build expectation in developer docs, installer trust prompts, uninstall behavior, and common antivirus false-positive notes.",
					"macOS: verify app launch permissions, Keychain-backed secret behavior, signing/notarization state, quarantine behavior, and app data cleanup expectations.",
					"Linux: verify package dependencies, Secret Service/libsecret or explicit unsupported-secret refusal, desktop entry/icon behavior, Wayland/X11 launch behavior, and app data cleanup expectations.",
				},
			},
			{
				Title: "Upgrade, Uninstall, And Closeout",
				Body: []string{
					"For upgrade tests, install over a previous beta, confirm settings, recent workspaces, metadata, protected secrets, and rollback records remain readable or migrate with clear messaging.",
					"Uninstall or remove the app using the platform path, verify expected app files are removed, document retained user data locations, and record any cleanup steps users must perform manually.",
					"Close the smoke run only after release notes list validation coverage, known limitations, required user actions, and any platform-specific risks.",
				},
			},
		},
	}
}

func CleanMachineSmokeChecklistMarkdown() string {
	return FormatMarkdown(CleanMachineSmokeChecklistGuide())
}
