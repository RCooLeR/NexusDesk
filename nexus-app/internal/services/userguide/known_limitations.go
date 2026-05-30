package userguide

func KnownLimitationsGuide() Guide {
	return Guide{
		Title:   "Known Limitations",
		Summary: "Current beta boundaries that users and testers should understand before relying on NexusDesk for important work.",
		Sections: []Section{
			{
				Title: "Packaging And Trust",
				Body: []string{
					"Public distribution is not complete until Windows signing, installer signing, macOS signing/notarization, Linux package strategy, release evidence, and clean-machine smoke are all recorded for the release.",
					"Unsigned local builds and fresh beta artifacts may trigger operating-system or antivirus reputation prompts. Use release manifest, SBOM, provenance, and release notes for verification.",
				},
			},
			{
				Title: "Provider And Model Setup",
				Body: []string{
					"Provider setup still depends on the user choosing a reachable local or compatible endpoint and model. Test connection and Diagnostics should be used before long assistant or agent runs.",
					"Answers can be incomplete or wrong when sources are weak, stale, uncited, or outside the model context window. Treat source warnings as review requirements.",
				},
			},
			{
				Title: "Agent And Tools",
				Body: []string{
					"Only implemented, approval-gated local tools should run. Planned tools remain roadmap-only until their design, tests, audit, rollback, and safety gates land.",
					"High-risk system mutation, connector mutation, shell orchestration, OCR, dump import, and external-agent workflows should remain unavailable until explicit release gates are met.",
				},
			},
			{
				Title: "Data And Connectors",
				Body: []string{
					"External database access is intended for bounded, read-only inspection unless a future mutation workflow adds approval, audit, rollback or mitigation, and redaction coverage.",
					"Large datasets, unusual Office files, corrupt archives, and slow network connectors may be capped, truncated, or rejected to keep the desktop responsive.",
				},
			},
			{
				Title: "Platform Coverage",
				Body: []string{
					"Windows has the strongest current build coverage. macOS and Linux package/build smoke must still be recorded on supported target machines before a production release.",
					"Protected-secret support depends on the platform backend: Windows DPAPI, macOS Keychain, or Linux Secret Service. Missing backends should fail clearly instead of silently storing secrets unsafely.",
				},
			},
		},
	}
}

func KnownLimitationsMarkdown() string {
	return FormatMarkdown(KnownLimitationsGuide())
}
