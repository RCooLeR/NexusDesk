package userguide

import (
	"strings"
	"testing"
)

func TestCleanMachineSmokeChecklistCoversReleaseValidation(t *testing.T) {
	markdown := CleanMachineSmokeChecklistMarkdown()
	for _, expected := range []string{
		"Clean-Machine Smoke Checklist",
		"Preflight",
		"Install And Launch",
		"Workspace And Editor",
		"Assistant And Safety",
		"Data, Artifacts, Jobs, And Diagnostics",
		"Platform-Specific Checks",
		"Upgrade, Uninstall, And Closeout",
		"Windows",
		"macOS",
		"Linux",
		"redacted issue report",
		"antivirus false-positive",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in smoke checklist markdown:\n%s", expected, markdown)
		}
	}
}
