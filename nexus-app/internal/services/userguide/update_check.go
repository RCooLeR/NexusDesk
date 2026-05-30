package userguide

import "strings"

func UpdateCheckGuide(version string, commit string, buildDate string) Guide {
	version = firstUpdateValue(version, "unknown")
	commit = firstUpdateValue(commit, "unknown")
	buildDate = firstUpdateValue(buildDate, "unknown")
	return Guide{
		Title:   "Check For Updates",
		Summary: "Manual update guidance for private beta builds. NexusDesk does not auto-download or auto-install updates.",
		Sections: []Section{
			{
				Title: "Current Build",
				Body: []string{
					"Version: " + version,
					"Commit: " + commit,
					"Build date: " + buildDate,
				},
			},
			{
				Title: "How To Check",
				Body: []string{
					"Compare this build identity with the newest release notes supplied by the project maintainer.",
					"Use the release manifest, SBOM, and provenance sidecars to verify any downloaded artifact before replacing the current build.",
				},
			},
			{
				Title: "Update Policy",
				Body: []string{
					"Update information is shown in this normal Help tab so it does not interrupt active work.",
					"NexusDesk does not download update artifacts automatically.",
					"NexusDesk does not install updates automatically.",
				},
			},
			{
				Title: "Release Notes",
				Body: []string{
					"Read `docs/releases/beta-release-notes.md` before upgrading private beta builds.",
					"Release notes must call out packaging changes, signing/trust state, validation coverage, known limitations, migration notes, and any required user action.",
				},
			},
		},
	}
}

func UpdateCheckMarkdown(version string, commit string, buildDate string) string {
	return FormatMarkdown(UpdateCheckGuide(version, commit, buildDate))
}

func firstUpdateValue(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
