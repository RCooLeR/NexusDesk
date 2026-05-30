package userguide

import (
	"strings"
	"testing"
)

func TestSampleWorkflowGuideCoversEndToEndBetaPath(t *testing.T) {
	markdown := SampleWorkflowMarkdown()
	for _, expected := range []string{
		"Sample Workflow Guide",
		"Prepare A Safe Workspace",
		"Home readiness",
		"Quick Open",
		"Rollbacks",
		"Ask With Sources",
		"Test connection",
		"Agent Audit",
		"Data And Artifacts",
		"metadata/lineage",
		"redacted issue report",
		"app version",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in sample workflow markdown:\n%s", expected, markdown)
		}
	}
}
