package shell

import (
	"strings"
	"testing"
	"time"

	artifactsSvc "nexusdesk/internal/services/artifacts"
)

func TestArtifactMetaFormatsTaskReport(t *testing.T) {
	meta := artifactMeta(artifactsSvc.Artifact{
		Kind:      "task-report",
		Size:      1234,
		CreatedAt: time.Date(2026, 5, 27, 12, 30, 0, 0, time.UTC),
	})
	for _, expected := range []string{"task-report", "2026-05-27 12:30:00", "1234 bytes"} {
		if !strings.Contains(meta, expected) {
			t.Fatalf("artifact meta %q missing %q", meta, expected)
		}
	}
}

func TestArtifactTitleFallsBackToFilename(t *testing.T) {
	if got := artifactTitle(artifactsSvc.Artifact{RelPath: ".nexusdesk/artifacts/task-runs/report.md"}); got != "report.md" {
		t.Fatalf("unexpected fallback title: %q", got)
	}
	if got := artifactTitle(artifactsSvc.Artifact{Title: "Task report", RelPath: "ignored.md"}); got != "Task report" {
		t.Fatalf("unexpected explicit title: %q", got)
	}
}
