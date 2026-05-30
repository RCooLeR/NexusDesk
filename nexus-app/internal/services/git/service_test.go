package git

import (
	"os"
	"strings"
	"testing"
)

func TestTimeoutForOperationUsesOperationClass(t *testing.T) {
	cases := map[operationClass]struct {
		wantGreaterThanStatus bool
		want                  string
	}{
		operationStatus:   {want: "4s"},
		operationDiff:     {wantGreaterThanStatus: true, want: "8s"},
		operationHistory:  {wantGreaterThanStatus: true, want: "12s"},
		operationMutation: {wantGreaterThanStatus: true, want: "20s"},
	}
	status := timeoutForOperation(operationStatus)
	for class, test := range cases {
		got := timeoutForOperation(class)
		if got.String() != test.want {
			t.Fatalf("timeoutForOperation(%s) = %s, want %s", class, got, test.want)
		}
		if test.wantGreaterThanStatus && got <= status {
			t.Fatalf("timeoutForOperation(%s) = %s, want greater than status timeout %s", class, got, status)
		}
	}
	if got := timeoutForOperation(operationClass("unknown")); got != status {
		t.Fatalf("unknown operation timeout = %s, want status timeout %s", got, status)
	}
}

func TestNonInteractiveGitEnvDisablesPrompts(t *testing.T) {
	env := nonInteractiveGitEnv([]string{"PATH=" + os.Getenv("PATH"), "GIT_TERMINAL_PROMPT=1", "GCM_INTERACTIVE=Always"})
	for _, expected := range []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
		"GCM_INTERACTIVE=Never",
	} {
		if !hasEnvEntry(env, expected) {
			t.Fatalf("non-interactive env missing %q in %#v", expected, env)
		}
	}
}

func hasEnvEntry(env []string, expected string) bool {
	key := strings.SplitN(expected, "=", 2)[0] + "="
	found := ""
	for _, entry := range env {
		if strings.HasPrefix(entry, key) {
			found = entry
		}
	}
	return found == expected
}
