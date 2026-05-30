package userguide

import (
	"strings"
	"testing"
)

func TestReleaseHygieneGuideCoversProductionTrustTopics(t *testing.T) {
	markdown := ReleaseHygieneMarkdown()
	for _, expected := range []string{
		"Release Hygiene And Antivirus Notes",
		"SHA256",
		"manifest",
		"code-signing",
		"notarization",
		"Linux Package Trust",
		"nexusdesk-<platform>-sbom.json",
		"sha256sum",
		"Linux Runtime Dependencies",
		"OpenGL",
		"Wayland/X11",
		"Secret Service",
		"Release Verification Steps",
		"provenance",
		"Antivirus False-Positive Triage",
		"Never ask users to disable antivirus globally",
		"Opening a workspace must stay cheap",
		"approval-gated",
		"clean-machine smoke",
		"Do Not Ship If",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected release hygiene guide to contain %q, got:\n%s", expected, markdown)
		}
	}
}
