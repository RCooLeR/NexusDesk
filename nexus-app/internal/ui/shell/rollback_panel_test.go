package shell

import (
	"strings"
	"testing"
	"time"

	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	workspaceSvc "nexusdesk/internal/services/workspace"
)

func TestRollbackControllerInitialState(t *testing.T) {
	_ = fynetest.NewTempApp(t)

	view := &View{}
	controller := newRollbackController(view)

	if controller.status.Text != "Rollback records have not been loaded." {
		t.Fatalf("expected initial rollback status, got %q", controller.status.Text)
	}
	if len(controller.results.Objects) != 1 {
		t.Fatalf("expected one placeholder row, got %d", len(controller.results.Objects))
	}
}

func TestRollbackControllerRequiresWorkspace(t *testing.T) {
	_ = fynetest.NewTempApp(t)

	view := &View{
		state:         NewState(),
		activityLog:   widget.NewRichTextFromMarkdown("Ready."),
		activityText:  "Ready.",
		activityLines: []string{"Ready."},
	}
	controller := newRollbackController(view)
	view.rollbacks = controller

	controller.Refresh()

	if controller.status.Text != "Open a workspace before reading rollback records." {
		t.Fatalf("expected missing workspace status, got %q", controller.status.Text)
	}
	if got := view.activityLines[len(view.activityLines)-1]; got != "Open a workspace before reading rollback records." {
		t.Fatalf("expected missing workspace activity, got %q", got)
	}
}

func TestRollbackRowsEmpty(t *testing.T) {
	rows := rollbackRows(nil, func(workspaceSvc.RollbackRecord) {})
	if len(rows) != 1 {
		t.Fatalf("expected one empty rollback row, got %d", len(rows))
	}
}

func TestRollbackConfirmTextIncludesScope(t *testing.T) {
	text := rollbackConfirmText(workspaceSvc.RollbackRecord{
		Target:  "src/app.go",
		Message: "Rollback snapshot is available.",
		Entries: []workspaceSvc.RollbackEntry{
			{RelPath: "src/app.go", Existed: true},
			{RelPath: "src/app_test.go", Existed: false},
		},
	})

	for _, expected := range []string{"2 path(s)", "src/app.go", "Rollback snapshot is available."} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected rollback confirmation to contain %q, got %q", expected, text)
		}
	}
}

func TestRollbackRecordBodyAcceptsTimestamps(t *testing.T) {
	_ = fynetest.NewTempApp(t)

	body := rollbackRecordBody(workspaceSvc.RollbackRecord{
		Action:    "update",
		Target:    "README.md",
		Status:    "active",
		Message:   "Snapshot ready.",
		CreatedAt: time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC),
		Entries:   []workspaceSvc.RollbackEntry{{RelPath: "README.md"}},
	})
	if body == nil {
		t.Fatal("expected rollback record body")
	}
}
