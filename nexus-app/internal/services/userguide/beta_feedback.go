package userguide

func BetaFeedbackGuide() Guide {
	return Guide{
		Title:   "Beta Feedback And Release Notes",
		Summary: "How private beta users should read release notes and report issues without leaking workspace data or secrets.",
		Sections: []Section{
			{
				Title: "Before Reporting",
				Body: []string{
					"Check Home readiness, Diagnostics, Jobs, Agent Audit, History, and Rollbacks before rerunning a failed workflow.",
					"Confirm whether the issue happened in Workbench, Editor, Assistant, Agent, Data, Artifacts, Tasks, Jobs, Diagnostics, Settings, or packaging.",
				},
			},
			{
				Title: "What To Include",
				Body: []string{
					"Include app version, commit, build date, operating system, provider type, and whether the run created a job, approval, rollback, artifact, SQL run, or agent-audit record.",
					"Include the exact visible error after redaction, the user action that triggered it, whether retry/cancel worked, and whether Diagnostics can export a redacted issue report.",
				},
			},
			{
				Title: "What Not To Include",
				Body: []string{
					"Do not include API keys, passwords, bearer tokens, DSNs, connector secrets, private prompts, production data, or workspace files unless the report explicitly requires them and you have reviewed the contents.",
					"Do not paste raw logs from external systems before checking for credentials, query strings, headers, and customer data.",
				},
			},
			{
				Title: "Redacted Issue Reports",
				Body: []string{
					"Use Diagnostics to export a redacted issue report when possible. The default bundle should include diagnostics text, activity tail, environment metadata, and workspace-state file names without workspace file contents.",
					"Only opt into workspace content when you understand exactly what will be included and it is safe to share.",
				},
			},
			{
				Title: "Release Notes",
				Body: []string{
					"Read release notes before upgrading. Pay special attention to migration notes, packaging changes, provider/credential changes, new agent tools, and known limitations.",
					"Release notes should distinguish new capabilities, fixed issues, safety changes, known risks, validation coverage, and any required user action.",
				},
			},
			{
				Title: "Beta Feedback Loop",
				Body: []string{
					"Useful beta feedback explains the goal, expected result, actual result, reproducibility, affected workspace type, and whether the workaround is acceptable.",
					"Prioritize reports for data loss, startup failures, unsafe tool requests, missing approvals, unredacted secrets, packaging failures, stuck jobs, provider failures, and misleading assistant citations.",
				},
			},
		},
	}
}

func BetaFeedbackMarkdown() string {
	return FormatMarkdown(BetaFeedbackGuide())
}
