package userguide

func ReleaseHygieneGuide() Guide {
	return Guide{
		Title:   "Release Hygiene And Antivirus Notes",
		Summary: "A production release checklist for reducing false positives, preserving user trust, and keeping NexusDesk builds traceable across Windows, macOS, and Linux.",
		Sections: []Section{
			{
				Title: "Release Artifact Discipline",
				Body: []string{
					"Ship only clean release artifacts produced by CI or the documented release pipeline. Do not publish ad-hoc local binaries, debug builds, unsigned test packages, or artifacts produced from a dirty worktree.",
					"Every release artifact should have version, commit, build date, platform, architecture, size, SHA256, and manifest metadata that matches the About dialog and release notes.",
				},
			},
			{
				Title: "Signing And Trust",
				Body: []string{
					"Windows releases should use the planned code-signing path before public distribution. macOS releases should use signing and notarization, and Linux packages should follow the chosen package trust model.",
					"Unsigned or freshly signed binaries can still trigger reputation warnings. Treat those prompts as release risks to document, not as user education problems.",
				},
			},
			{
				Title: "Linux Package Trust",
				Body: []string{
					"Until a signed Linux repository is available, Linux private-beta packages should use a hash-first trust path: publish the artifact with `nexusdesk-<platform>-manifest.json`, `nexusdesk-<platform>-sbom.json`, and `nexusdesk-<platform>-provenance.json` sidecars from the release pipeline.",
					"Verification should compare the artifact SHA256 from the manifest with `sha256sum`, confirm the provenance subject matches the version/commit/build date in About, and record whether the package was delivered as AppImage, tarball, deb, rpm, or zip.",
					"Do not ask users to bypass Linux desktop trust prompts blindly. Release notes should explain the package format, hash/provenance path, repository-signing state if deb/rpm packages are used, and any expected first-launch prompts.",
				},
			},
			{
				Title: "Linux Runtime Dependencies",
				Body: []string{
					"Linux release notes should list tested distributions, architecture, package format, OpenGL/graphics expectations, Wayland/X11 launch coverage, desktop entry/icon behavior, and whether sandboxed packaging is supported.",
					"Protected-secret verification should name Secret Service/libsecret/secret-tool support when available and the explicit unsupported-secret refusal path when a desktop keyring is missing.",
					"Data connectors should remain opt-in. Document any package-level dependencies needed for local SQLite, external database drivers, desktop portals, file dialogs, and clipboard behavior on the supported distro target.",
				},
			},
			{
				Title: "Antivirus False-Positive Triage",
				Body: []string{
					"Keep release binaries stable, reproducible, and minimally packed. Avoid obfuscation, self-modifying behavior, unexpected child processes, hidden network activity, and bundling unrelated tools.",
					"If a scanner flags a build, record the exact artifact SHA256, scanner/vendor, detection name, platform, signing state, download URL, and whether the same source commit reproduces the flagged binary.",
					"Never ask users to disable antivirus globally. Prefer vendor false-positive submission, signed rebuilds when appropriate, clear release notes, and temporary download warnings only when the team has verified the artifact.",
				},
			},
			{
				Title: "Runtime Behavior That Builds Trust",
				Body: []string{
					"Opening a workspace must stay cheap and must not start Git, Docker, shell commands, OCR, connector pulls, dump imports, indexing, model calls, or background network activity.",
					"External network calls, connector access, shell-like tasks, and high-risk file/database/system actions must remain explicit, bounded, approval-gated where needed, audited, and redacted in diagnostics.",
				},
			},
			{
				Title: "Release Notes And Support",
				Body: []string{
					"Release notes should list platform coverage, signing/notarization state, known trust prompts, scanner findings if any, validation results, supported upgrade/uninstall behavior, and required user actions.",
					"Support responses for trust prompts should point to the release manifest, SHA256 verification, signing state, clean-machine smoke results, and vendor submission status rather than asking users to bypass protections.",
				},
			},
			{
				Title: "Release Verification Steps",
				Body: []string{
					"Before publishing, verify the built artifact launches on the target platform and that About shows the same version, commit, and build date recorded in the release manifest.",
					"Verify the artifact hash against the manifest, open the SBOM to confirm dependency evidence exists, open provenance to confirm manifest/SBOM hashes are recorded, and archive all three sidecars with the release artifact.",
					"Run the packaged executable's `--version` and `--smoke-check` commands before manual UI smoke, then run clean-machine smoke, protected-secret smoke, uninstall/app-data cleanup smoke, and Diagnostics release trust checks. Record any remaining blocker as a known limitation or do not ship the artifact.",
				},
			},
			{
				Title: "Do Not Ship If",
				Body: []string{
					"A release artifact was built from an unknown commit, has no matching manifest, has unexpected size/hash drift, fails clean-machine smoke, contains generated debug files, or triggers an untriaged high-confidence malware detection.",
					"Signing, installer, update, uninstall, protected-secret, issue-report redaction, and app-data cleanup behavior must be documented before private-beta users receive the build.",
				},
			},
		},
	}
}

func ReleaseHygieneMarkdown() string {
	return FormatMarkdown(ReleaseHygieneGuide())
}
