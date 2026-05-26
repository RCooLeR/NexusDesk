package editor

import "testing"

func TestOpenFileReusesExistingTab(t *testing.T) {
	session := NewSession()

	first := session.OpenFile("docs/readme.md", "readme.md")
	second := session.OpenFile("docs/readme.md", "README.md")

	if first.ID != second.ID {
		t.Fatalf("expected same tab id, got %q and %q", first.ID, second.ID)
	}
	if len(session.Tabs()) != 1 {
		t.Fatalf("expected one tab, got %d", len(session.Tabs()))
	}
	if session.ActiveID() != first.ID {
		t.Fatalf("expected reused tab to stay active")
	}
}

func TestCloseDirtyTabRequiresForce(t *testing.T) {
	session := NewSession()
	tab := session.OpenFile("main.go", "main.go")
	session.MarkDirty(tab.ID, true)

	if current, ok := session.Tab(tab.ID); !ok || !current.Dirty {
		t.Fatalf("expected tab lookup to show dirty state")
	}
	if _, ok := session.Close(tab.ID, false); ok {
		t.Fatalf("expected clean close to reject dirty tab")
	}
	if len(session.Tabs()) != 1 {
		t.Fatalf("expected dirty tab to remain open")
	}
	if _, ok := session.Close(tab.ID, true); !ok {
		t.Fatalf("expected forced close to remove dirty tab")
	}
	if len(session.Tabs()) != 0 {
		t.Fatalf("expected no tabs after forced close")
	}
}

func TestPinnedTabsStayBeforeUnpinnedTabs(t *testing.T) {
	session := NewSession()
	first := session.OpenFile("a.go", "a.go")
	second := session.OpenFile("b.go", "b.go")

	if _, ok := session.TogglePinned(second.ID); !ok {
		t.Fatalf("expected pin toggle to succeed")
	}

	tabs := session.Tabs()
	if tabs[0].ID != second.ID || tabs[1].ID != first.ID {
		t.Fatalf("expected pinned tab before unpinned tab, got %#v", tabs)
	}
}
