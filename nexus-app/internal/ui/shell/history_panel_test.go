package shell

import (
	"strings"
	"testing"
	"time"

	historySvc "nexusdesk/internal/services/history"
)

func TestHistoryKindFromLabel(t *testing.T) {
	cases := map[string]historySvc.Kind{
		"All":       "",
		"Chat":      historySvc.KindChat,
		"Artifacts": historySvc.KindArtifact,
		"Jobs":      historySvc.KindJob,
		"Agent":     historySvc.KindAgent,
	}
	for label, want := range cases {
		if got := historyKindFromLabel(label); got != want {
			t.Fatalf("historyKindFromLabel(%q) = %q, want %q", label, got, want)
		}
	}
}

func TestFormatHistoryItemIncludesDetail(t *testing.T) {
	output := formatHistoryItem(historySvc.Item{
		Kind:    historySvc.KindArtifact,
		Ref:     ".nexusdesk/artifacts/report.md",
		Title:   "Report",
		Summary: "Generated output",
		Detail:  "Path: .nexusdesk/artifacts/report.md",
		When:    time.Date(2026, 5, 27, 12, 0, 0, 0, time.UTC),
	})
	for _, expected := range []string{"ARTIFACT - Report", "Generated output", "Path: .nexusdesk/artifacts/report.md"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("history detail missing %q:\n%s", expected, output)
		}
	}
}

func TestHistoryStatusTextIncludesFilter(t *testing.T) {
	status := historyStatusText("nexus", historySvc.KindChat, 3)
	if status != `History / chat / "nexus": 3 record(s).` {
		t.Fatalf("unexpected history status: %q", status)
	}
}
