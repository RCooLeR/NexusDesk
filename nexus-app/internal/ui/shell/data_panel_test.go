package shell

import (
	"strings"
	"testing"

	datasetsSvc "nexusdesk/internal/services/datasets"
)

func TestFormatDatasetProfileIncludesColumnsAndJSONNotes(t *testing.T) {
	output := formatDatasetProfile(datasetsSvc.Profile{
		RelPath:   "data/events.json",
		Format:    "JSON",
		MediaType: "application/json",
		Size:      42,
		Rows:      2,
		Columns: []datasetsSvc.ColumnProfile{
			{Name: "channel", Type: "text", NonEmpty: 2, Samples: []string{"search", "email"}},
		},
		JSONProfile: &datasetsSvc.JSONProfile{TopLevel: "array", Count: 2, Notes: []string{"Array object fields are profiled across object elements."}},
	})
	if !strings.Contains(output, "Path: data/events.json") || !strings.Contains(output, "- channel | text") || !strings.Contains(output, "Top level: array") {
		t.Fatalf("profile output missing expected details:\n%s", output)
	}
}

func TestProfileStatusMarksTruncatedSample(t *testing.T) {
	status := profileStatus(datasetsSvc.Profile{RelPath: "data.csv", Format: "CSV", Rows: 50, Columns: make([]datasetsSvc.ColumnProfile, 3), Truncated: true})
	if !strings.Contains(status, "CSV sample") || !strings.Contains(status, "3 columns") {
		t.Fatalf("unexpected profile status: %q", status)
	}
}
