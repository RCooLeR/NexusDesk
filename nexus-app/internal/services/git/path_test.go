package git

import "testing"

func TestCleanRelPath(t *testing.T) {
	got, err := cleanRelPath(`"app/frontend/src/App.tsx"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "app/frontend/src/App.tsx" {
		t.Fatalf("unexpected path: %q", got)
	}
	for _, value := range []string{"", ".", "..", "../outside.go", "app/../outside.go", "app/..", "-bad"} {
		if _, err := cleanRelPath(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
