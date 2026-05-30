package userguide

func ProviderSetupWizardGuide() Guide {
	return Guide{
		Title:   "Provider Setup Wizard",
		Summary: "A short setup path for connecting NexusDesk to an OpenAI-compatible or Ollama provider before running Ask or Agent workflows.",
		Sections: []Section{
			{
				Title: "1. Choose Provider And Endpoint",
				Body: []string{
					"Open Model Settings, choose the provider profile, confirm the protocol, and verify the Base URL points to the intended local or compatible endpoint.",
					"For Ollama, start the runtime first and use the OpenAI-compatible endpoint, commonly `http://localhost:11434/v1`.",
				},
			},
			{
				Title: "2. Select Or Detect Model",
				Body: []string{
					"Choose a recommended model from the catalog or run Test connection to read provider models. If the configured model is blank or missing, NexusDesk suggests the first detected provider model.",
					"Review context tokens and response reserve after a probe because loaded runtime metadata can tune the context window automatically.",
				},
			},
			{
				Title: "3. Save Credentials Safely",
				Body: []string{
					"Enter an API key only when the provider requires one. Saved keys use protected OS storage where available and display as redacted values after save.",
					"Do not paste provider credentials into prompts, workspace files, issue reports, screenshots, or beta feedback.",
				},
			},
			{
				Title: "4. Test And Verify",
				Body: []string{
					"Run Test connection and read the status, model count, warnings, suggested model, runtime context, and guidance before starting long workflows.",
					"After saving, run Diagnostics to confirm provider, protected-secret, release trust, and workspace readiness signals are visible.",
				},
			},
			{
				Title: "5. Route Defaults",
				Body: []string{
					"Use Task Model Routes to set defaults for coding, research, data, vision, and balanced workflows. Test selected routes before relying on them.",
					"Keep route models explicit so future route-aware workflows do not silently fall back to an unintended model.",
				},
			},
		},
	}
}

func ProviderSetupWizardMarkdown() string {
	return FormatMarkdown(ProviderSetupWizardGuide())
}
