package shell

import "testing"

func TestDirtyTabCloseMessageUsesTitleFallback(t *testing.T) {
	if got := dirtyTabCloseMessage("README.md"); got != "Discard unsaved changes in README.md?" {
		t.Fatalf("unexpected dirty close message: %q", got)
	}
	if got := dirtyTabCloseMessage(""); got != "Discard unsaved changes in this tab?" {
		t.Fatalf("unexpected dirty close fallback: %q", got)
	}
}
