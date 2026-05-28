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

func TestDefaultCreateFolderPathUsesSelectedDirectory(t *testing.T) {
	if got := defaultCreateFolderPath("docs"); got != "docs/new-folder" {
		t.Fatalf("unexpected create folder path: %q", got)
	}
	if got := defaultCreateFolderPath("docs/readme.md"); got != "docs/new-folder" {
		t.Fatalf("unexpected create folder path near selected file: %q", got)
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
	fileOptions := navigatorActionOptions("docs/readme.md", domain.NodeFile, true)
	for _, option := range []string{navigatorActionCreateDir, navigatorActionCopy, navigatorActionCut, navigatorActionPaste, navigatorActionRename, navigatorActionDelete, navigatorActionCopyPath, navigatorActionUseContext} {
		if !slices.Contains(fileOptions, option) {
			t.Fatalf("file options are missing %q: %#v", option, fileOptions)
		}
	}

	directoryOptions := navigatorActionOptions("docs", domain.NodeDirectory, true)
	if !slices.Contains(directoryOptions, navigatorActionCreateDir) {
		t.Fatalf("directory options should include folder creation: %#v", directoryOptions)
	}
	if !slices.Contains(directoryOptions, navigatorActionPaste) {
		t.Fatalf("directory options should include paste when clipboard is active: %#v", directoryOptions)
	}
	for _, option := range []string{navigatorActionCopy, navigatorActionCut, navigatorActionRename, navigatorActionDelete} {
		if slices.Contains(directoryOptions, option) {
			t.Fatalf("directory options should not include file-only action %q: %#v", option, directoryOptions)
		}
	}

	emptyClipboardOptions := navigatorActionOptions("docs", domain.NodeDirectory, false)
	if slices.Contains(emptyClipboardOptions, navigatorActionPaste) {
		t.Fatalf("directory options should hide paste without clipboard: %#v", emptyClipboardOptions)
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
	truncated := navigatorVisibilitySummary(false, domain.ScanSummary{Included: 600, EntryCap: 1})
	if truncated != "600 shown, 1 folder(s) clipped by entry cap" {
		t.Fatalf("unexpected truncated summary: %q", truncated)
	}
	truncatedWithIgnored := navigatorVisibilitySummary(false, domain.ScanSummary{Included: 600, Ignored: 3, EntryCap: 2})
	if truncatedWithIgnored != "600 shown, 3 ignored hidden, 2 folder(s) clipped by entry cap" {
		t.Fatalf("unexpected ignored truncated summary: %q", truncatedWithIgnored)
	}
}

func TestTreeStoreVisibleSummaryAggregatesLoadedEntryCaps(t *testing.T) {
	store := &treeStore{summaries: map[string]domain.ScanSummary{
		"":     {Included: 4, EntryCap: 1},
		"docs": {Included: 600, EntryCap: 1},
	}}
	summary := store.visibleSummary()
	if summary.Included != 4 || summary.EntryCap != 2 {
		t.Fatalf("unexpected visible summary: %#v", summary)
	}
}

func TestNavigatorPasteDirectoryUsesSelectionKind(t *testing.T) {
	workspace := domain.Workspace{Tree: []domain.WorkspaceNode{
		{RelPath: "docs", Kind: domain.NodeDirectory, Children: []domain.WorkspaceNode{
			{RelPath: "docs/readme.md", Kind: domain.NodeFile},
		}},
	}}
	if got := navigatorPasteDirectory(workspace, "docs"); got != "docs" {
		t.Fatalf("unexpected directory paste target: %q", got)
	}
	if got := navigatorPasteDirectory(workspace, "docs/readme.md"); got != "docs" {
		t.Fatalf("unexpected file paste target: %q", got)
	}
	if got := navigatorPasteDirectory(workspace, "README.md"); got != "" {
		t.Fatalf("unexpected root paste target: %q", got)
	}
}

func TestNavigatorUniqueCopyPathAvoidsExistingTreeNode(t *testing.T) {
	workspace := domain.Workspace{Tree: []domain.WorkspaceNode{
		{RelPath: "docs", Kind: domain.NodeDirectory, Children: []domain.WorkspaceNode{
			{RelPath: "docs/readme.md", Kind: domain.NodeFile},
			{RelPath: "docs/readme-copy-2.md", Kind: domain.NodeFile},
		}},
	}}
	if got := navigatorUniqueCopyPath(workspace, "docs/readme.md"); got != "docs/readme-copy-3.md" {
		t.Fatalf("unexpected unique path: %q", got)
	}
}
