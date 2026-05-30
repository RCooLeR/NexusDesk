package userguide

func SampleWorkflowGuide() Guide {
	return Guide{
		Title:   "Sample Workflow Guide",
		Summary: "A small end-to-end path beta users can run on a trusted sample workspace to verify NexusDesk without touching production data.",
		Sections: []Section{
			{
				Title: "Prepare A Safe Workspace",
				Body: []string{
					"Use a small local folder with a few Markdown, JSON, Go, CSV, and SQLite sample files. Do not use production secrets or customer data for the first workflow.",
					"Open the workspace, confirm Home readiness, and run Diagnostics before starting assistant or data work.",
				},
			},
			{
				Title: "Inspect And Edit",
				Body: []string{
					"Use Quick Open, Search, Problems, document map, and preview to inspect the sample files.",
					"Make a tiny text edit, save it, inspect the Git diff if available, then use Rollbacks or Revert Draft to verify recovery behavior.",
				},
			},
			{
				Title: "Ask With Sources",
				Body: []string{
					"Pin one or two relevant files and ask for a short summary or risk list. Check citations, source freshness, and warnings before trusting the answer.",
					"If the provider is not configured, open Settings, run Test connection, choose a detected model, and retry the Ask workflow.",
				},
			},
			{
				Title: "Run A Low-Risk Agent Task",
				Body: []string{
					"Ask the agent for a small documentation or formatting change that can be reviewed. Approve only actions with clear target paths and safe intent.",
					"After the run, inspect Jobs, Agent Audit, History, Git diff, and Rollbacks so the full chain of evidence is visible.",
				},
			},
			{
				Title: "Data And Artifacts",
				Body: []string{
					"Profile a sample CSV or JSON file, run one bounded read-only query, and export or copy a small result.",
					"Generate a simple artifact, inspect its metadata/lineage, regenerate it if safe, and confirm Artifacts and History tell the same story.",
				},
			},
			{
				Title: "Closeout",
				Body: []string{
					"Run Diagnostics again, export a redacted issue report if anything failed, and compare the report against the beta feedback template before sharing.",
					"Record app version, commit, build date, operating system, provider, model, and whether every step had clear status, cancellation, retry, or recovery behavior.",
				},
			},
		},
	}
}

func SampleWorkflowMarkdown() string {
	return FormatMarkdown(SampleWorkflowGuide())
}
