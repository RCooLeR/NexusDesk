package shell

import (
	"slices"
	"testing"

	"nexusdesk/internal/domain"
)

func TestDefaultCreatePathUsesSelectedDirectory(t *testing.T) {
	if got := defaultCreatePath("docs"); got != "docs/new-file.txt" {
		t.Fatalf("unexpected create path: %q", got)
	}
	if got := defaultCreatePath("docs/readme.md"); got != "docs/new-file.txt" {
		t.Fatalf("unexpected create path near selected file: %q", got)
	}
}

func TestDefaultCopyPathAddsCopySuffix(t *testing.T) {
	if got := defaultCopyPath("docs/readme.md"); got != "docs/readme-copy.md" {
		t.Fatalf("unexpected copy path: %q", got)
	}
	if got := defaultCopyPath("README"); got != "README-copy" {
		t.Fatalf("unexpected root copy path: %q", got)
	}
}

func TestNavigatorActionOptionsRespectSelectionKind(t *testing.T) {
	fileOptions := navigatorActionOptions("docs/readme.md", domain.NodeFile)
	for _, option := range []string{navigatorActionCopy, navigatorActionRename, navigatorActionDelete, navigatorActionCopyPath, navigatorActionUseContext} {
		if !slices.Contains(fileOptions, option) {
			t.Fatalf("file options are missing %q: %#v", option, fileOptions)
		}
	}

	directoryOptions := navigatorActionOptions("docs", domain.NodeDirectory)
	for _, option := range []string{navigatorActionCopy, navigatorActionRename, navigatorActionDelete} {
		if slices.Contains(directoryOptions, option) {
			t.Fatalf("directory options should not include file-only action %q: %#v", option, directoryOptions)
		}
	}
}

func TestNavigatorContextMenuItemsDispatchAction(t *testing.T) {
	var selected string
	items := navigatorContextMenuItems([]string{navigatorActionCreate, navigatorActionCopyPath}, func(action string) {
		selected = action
	})
	if len(items) != 3 {
		t.Fatalf("expected action, separator, action menu items, got %d", len(items))
	}
	if items[0].Label != navigatorActionCreate {
		t.Fatalf("unexpected first menu item label: %q", items[0].Label)
	}
	items[2].Action()
	if selected != navigatorActionCopyPath {
		t.Fatalf("unexpected dispatched action: %q", selected)
	}
}

func TestNavigatorSelectionSummary(t *testing.T) {
	if got := navigatorSelectionSummary(""); got != "No file selected" {
		t.Fatalf("unexpected empty selection summary: %q", got)
	}
	if got := navigatorSelectionSummary("docs/readme.md"); got != "docs/readme.md" {
		t.Fatalf("unexpected selected summary: %q", got)
	}
}

func TestNavigatorVisibilitySummary(t *testing.T) {
	hidden := navigatorVisibilitySummary(false, domain.ScanSummary{Included: 4, Ignored: 2})
	if hidden != "4 shown, 2 ignored hidden" {
		t.Fatalf("unexpected hidden summary: %q", hidden)
	}
	visible := navigatorVisibilitySummary(true, domain.ScanSummary{Included: 6, Ignored: 2})
	if visible != "6 shown, 2 ignored visible where safe" {
		t.Fatalf("unexpected visible summary: %q", visible)
	}
}
