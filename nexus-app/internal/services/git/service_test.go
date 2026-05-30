package git

import (
	"errors"
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

func TestRepositoryUnavailableMessageClassifiesOwnershipSafety(t *testing.T) {
	err := errors.New("fatal: detected dubious ownership in repository at 'C:/work/repo'\nTo add an exception for this directory, call:\n\n\tgit config --global --add safe.directory C:/work/repo")
	message := repositoryUnavailableMessage(`C:\work\repo`, err)
	for _, expected := range []string{"ownership is not trusted", "safe.directory", `"C:\work\repo"`} {
		if !strings.Contains(message, expected) {
			t.Fatalf("ownership message missing %q:\n%s", expected, message)
		}
	}
}

func TestRepositoryUnavailableMessageKeepsNonRepoFallback(t *testing.T) {
	message := repositoryUnavailableMessage(`C:\work\repo`, errors.New("fatal: not a git repository"))
	if message != "Workspace is not inside a Git repository." {
		t.Fatalf("unexpected non-repo message: %q", message)
	}
}

func TestRejectSilentNetworkGitCommand(t *testing.T) {
	for _, command := range []string{"fetch", "pull", "push", "clone", "ls-remote"} {
		err := rejectSilentNetworkGitCommand([]string{command, "origin"})
		if err == nil || !strings.Contains(err.Error(), "explicit network workflow") {
			t.Fatalf("expected %s to require explicit network workflow, got %v", command, err)
		}
	}
	if err := rejectSilentNetworkGitCommand([]string{"status", "--porcelain=v1"}); err != nil {
		t.Fatalf("local status command should be allowed: %v", err)
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
