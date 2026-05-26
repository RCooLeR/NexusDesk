package main

import "testing"

func TestParseAllowedAgentShellCommandBlocksShellMetacharacters(t *testing.T) {
	if _, _, ok := parseAllowedAgentShellCommand("git status && del *"); ok {
		t.Fatal("expected shell metacharacters to be blocked")
	}
	if _, _, ok := parseAllowedAgentShellCommand("git config core.fsmonitor evil"); ok {
		t.Fatal("expected unsafe git subcommand to be blocked")
	}
	if _, _, ok := parseAllowedAgentShellCommand("git diff src/main.go"); !ok {
		t.Fatal("expected safe git diff command to be allowed")
	}
	if _, _, ok := parseAllowedAgentShellCommand("go test ./..."); !ok {
		t.Fatal("expected safe Go package pattern to be allowed")
	}
}

func TestParseAllowedAgentShellCommandBlocksEscapingPaths(t *testing.T) {
	blocked := []string{
		"npm run test ../outside",
		"python C:\\Users\\roman\\script.py",
		"node /tmp/script.js",
		"git show ~",
	}
	for _, command := range blocked {
		if _, _, ok := parseAllowedAgentShellCommand(command); ok {
			t.Fatalf("expected %q to be blocked", command)
		}
	}
}
