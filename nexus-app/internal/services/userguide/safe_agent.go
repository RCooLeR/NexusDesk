package userguide

import "strings"

type Section struct {
	Title string
	Body  []string
}

type Guide struct {
	Title    string
	Summary  string
	Sections []Section
}

func SafeAgentGuide() Guide {
	return Guide{
		Title:   "Safe Agent Use",
		Summary: "How NexusDesk keeps local agent work understandable, reversible, and private.",
		Sections: []Section{
			{
				Title: "Start From A Trusted Workspace",
				Body: []string{
					"Open only project folders you trust. NexusDesk keeps folder open cheap and does not run Git, Docker, shell commands, model calls, OCR, connector pulls, dump imports, or deep indexing just because a folder was opened.",
					"Use Home readiness and Diagnostics before long runs. They show workspace, provider/model, credential, and native toolchain gaps before agent work starts.",
				},
			},
			{
				Title: "Control Context Explicitly",
				Body: []string{
					"Ask mode answers from selected workspace context, pinned files, artifacts, and chat history. Pin only the files or folders that should influence the answer.",
					"Treat citations and source diagnostics as part of the answer. Weak, stale, uncited, or unverified sources mean the result needs review before acting on it.",
				},
			},
			{
				Title: "Approvals And Risky Actions",
				Body: []string{
					"Agent tools are bounded and recorded. File mutations, high-risk workspace changes, shell-like task execution, connector work, and future system mutations must stay behind explicit approvals, audit records, and rollback or mitigation paths.",
					"Do not approve a request unless the target path, proposed action, and reason make sense. When in doubt, deny it and ask the assistant to explain or narrow the operation.",
				},
			},
			{
				Title: "Rollbacks And Recovery",
				Body: []string{
					"Supported file writes create rollback records where practical. Use the Rollbacks panel to inspect and restore changes from model-authored mutations.",
					"Jobs, agent runs, tool runs, artifacts, SQL runs, approvals, and chat messages are persisted as local metadata so failures can be inspected after the fact.",
				},
			},
			{
				Title: "Local Data And Secrets",
				Body: []string{
					"NexusDesk is local-first. Workspace contents are not included in issue reports unless explicitly requested, and exported issue bundles redact secrets by default.",
					"Provider API keys and connector credentials use protected OS storage where available and display as redacted values in the UI. Avoid pasting secrets into prompts, files, SQL text, or artifact descriptions.",
				},
			},
			{
				Title: "Connectors And Databases",
				Body: []string{
					"External database work defaults to bounded, read-only, single-statement inspection with cancellation, query history, and redacted errors.",
					"Do not connect production systems until profile scope, credentials, query limits, and export expectations are clear. Mutation workflows should remain unavailable until their approval, audit, job, and rollback design is complete.",
				},
			},
			{
				Title: "Slow Work And Jobs",
				Body: []string{
					"Long operations should appear as jobs with progress, logs, cancellation, retry, and output-opening paths. Folder open should never be used as a trigger for slow or external work.",
					"If a workflow feels stuck, check Jobs, Diagnostics, Agent Audit, and History before rerunning it.",
				},
			},
		},
	}
}

func SafeAgentMarkdown() string {
	return FormatMarkdown(SafeAgentGuide())
}

func FormatMarkdown(guide Guide) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(strings.TrimSpace(guide.Title))
	builder.WriteString("\n\n")
	if summary := strings.TrimSpace(guide.Summary); summary != "" {
		builder.WriteString(summary)
		builder.WriteString("\n\n")
	}
	for _, section := range guide.Sections {
		title := strings.TrimSpace(section.Title)
		if title == "" {
			continue
		}
		builder.WriteString("## ")
		builder.WriteString(title)
		builder.WriteString("\n\n")
		for _, paragraph := range section.Body {
			paragraph = strings.TrimSpace(paragraph)
			if paragraph == "" {
				continue
			}
			builder.WriteString("- ")
			builder.WriteString(paragraph)
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String()) + "\n"
}
