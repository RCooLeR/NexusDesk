package shell

import (
	"strconv"
	"strings"
	"testing"

	fynetest "fyne.io/fyne/v2/test"
)

func TestAddActivityKeepsBoundedMarkdownBuffer(t *testing.T) {
	app := fynetest.NewTempApp(t)
	window := app.NewWindow("activity")
	defer window.Close()

	view := New(window)
	for index := 1; index <= activityHistoryLimit+25; index++ {
		view.addActivity("line-" + strconv.Itoa(index))
	}

	if len(view.activityLines) != activityHistoryLimit {
		t.Fatalf("expected bounded activity lines, got %d", len(view.activityLines))
	}
	sections := strings.Split(view.activityText, "\n\n")
	if len(sections) != activityHistoryLimit {
		t.Fatalf("expected markdown buffer to mirror bounded line history, got %d sections", len(sections))
	}
	if sections[0] != "line-26" {
		t.Fatalf("expected first retained line to be line-26, got %q", sections[0])
	}
	if sections[len(sections)-1] != "line-"+strconv.Itoa(activityHistoryLimit+25) {
		t.Fatalf("expected newest line at tail, got %q", sections[len(sections)-1])
	}
}
