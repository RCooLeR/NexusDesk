package shell

import (
	"testing"

	"nexusdesk/internal/domain"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

func TestTreeStoreBranchPathForSelection(t *testing.T) {
	store := newTreeStore(domain.Workspace{
		Tree: []domain.WorkspaceNode{
			{ID: "docs", RelPath: "docs", Name: "docs", Kind: domain.NodeDirectory},
		},
	}, workspaceSvc.New(), nil)
	store.setChildren("docs", []domain.WorkspaceNode{
		{ID: "docs/guides", ParentID: "docs", RelPath: "docs/guides", Name: "guides", Kind: domain.NodeDirectory},
		{ID: "docs/readme.md", ParentID: "docs", RelPath: "docs/readme.md", Name: "readme.md", Kind: domain.NodeFile},
	})
	store.setChildren("docs/guides", []domain.WorkspaceNode{
		{ID: "docs/guides/setup.md", ParentID: "docs/guides", RelPath: "docs/guides/setup.md", Name: "setup.md", Kind: domain.NodeFile},
	})

	if got := store.branchPathForSelection("docs/guides/setup.md"); !sameTreeBranches(got, []string{"docs", "docs/guides"}) {
		t.Fatalf("unexpected file branches: %#v", got)
	}
	if got := store.branchPathForSelection("docs/guides"); !sameTreeBranches(got, []string{"docs", "docs/guides"}) {
		t.Fatalf("unexpected directory branches: %#v", got)
	}
}

func TestNavigatorParentID(t *testing.T) {
	cases := map[string]string{
		"":                  "",
		"README.md":         "",
		"docs/readme.md":    "docs",
		"docs/guides":       "docs",
		"/docs/guides/a.md": "docs/guides",
	}
	for input, want := range cases {
		if got := navigatorParentID(input); got != want {
			t.Fatalf("navigatorParentID(%q) = %q, want %q", input, got, want)
		}
	}
}

func sameTreeBranches(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
