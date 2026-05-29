package userguide

func ContributorGuide() Guide {
	return Guide{
		Title:   "Contributor Setup And Standards",
		Summary: "A production-oriented contributor guide for building NexusDesk safely without weakening the Fyne-native architecture.",
		Sections: []Section{
			{
				Title: "Active App",
				Body: []string{
					"Develop in `nexus-app/`; it is the single active desktop product.",
					"Do not reintroduce deprecated runtime dependencies or generated UI bridge code into the active app.",
				},
			},
			{
				Title: "Local Setup",
				Body: []string{
					"Use Go with CGO enabled for Fyne builds. On Windows, use MSYS2 UCRT64 GCC and prefer `nexus-app/scripts/dev-env.ps1` for test/build/run setup.",
					"Keep module use readonly during validation when possible: `GOFLAGS=-mod=readonly` helps catch accidental dependency drift.",
					"Prefer `nexus-app/scripts/dev-env.ps1 -BuildCheck` for routine validation because it builds to a temporary folder and removes the unsigned executable immediately. Use `-Build` only when a local runnable artifact is intentionally needed.",
					"After raw `go build .` or `-Build`, remove generated binaries such as `nexusdesk.exe` before committing. Unsigned local Windows builds can trigger Norton or SmartScreen on every fresh hash; production trust needs signed release artifacts, not antivirus bypasses.",
				},
			},
			{
				Title: "Coding Standards",
				Body: []string{
					"Put business rules, path safety, query safety, approvals, rollbacks, redaction, and persistence in services before wiring UI.",
					"Keep services and domain packages framework-free. Fyne imports belong in app, UI, theme, brand presentation code, and UI tests only.",
					"Keep folder open cheap: no Git, Docker, OCR, connector pulls, dump imports, model calls, shell commands, or deep indexing on workspace open.",
					"Prefer small package-owned helpers and focused tests over adding orchestration to `internal/ui/shell`.",
				},
			},
			{
				Title: "Testing Standards",
				Body: []string{
					"Add focused tests for every milestone. Service tests should cover boundaries, caps, cancellation, redaction, metadata, and safety decisions.",
					"Use small deterministic fixtures. Do not start external services unless the test name and package make that dependency explicit.",
					"Run `gofmt`, `go test ./...`, `nexus-app/scripts/dev-env.ps1 -BuildCheck`, and `git diff --check` before committing a milestone.",
				},
			},
			{
				Title: "Documentation And Tracker Updates",
				Body: []string{
					"Update `tracker.md` and the relevant docs whenever behavior, architecture, production readiness, or UI/tooling plans change.",
					"Keep `docs/05_PLAN.md` as the broad roadmap and `docs/03_UI_WORKBENCH.md` as the UI target.",
					"Document new package ownership in `docs/01_ARCHITECTURE.md` before responsibilities become tribal knowledge.",
				},
			},
			{
				Title: "ADR Process",
				Body: []string{
					"Create an ADR for decisions that change architecture boundaries, persistence formats, security policy, connector behavior, packaging, or extension/plugin execution.",
					"Use `docs/adr/NNNN-short-title.md` with Context, Decision, Consequences, Status, and Date sections.",
					"Prefer reversible decisions and document migration or rollback expectations when persistence or generated artifacts are affected.",
				},
			},
			{
				Title: "Commit Discipline",
				Body: []string{
					"Commit logical milestones only after validation passes. Do not amend or force-push unless explicitly requested.",
					"Do not revert unrelated user changes. If unexpected unrelated changes appear, pause and ask how to proceed.",
					"Use direct commit messages such as `Document contributor standards`, `Harden native artifact regeneration`, or `Polish diagnostics health cards`.",
				},
			},
		},
	}
}

func ContributorMarkdown() string {
	return FormatMarkdown(ContributorGuide())
}
