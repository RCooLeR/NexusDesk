package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const gitCommandTimeout = 4 * time.Second
const gitDiffMaxBytes = 220 * 1024

type GitStatus struct {
	Available     bool            `json:"available"`
	RepoRoot      string          `json:"repoRoot"`
	Branch        string          `json:"branch"`
	Head          string          `json:"head"`
	Dirty         bool            `json:"dirty"`
	ChangedFiles  []GitFileChange `json:"changedFiles"`
	Diff          string          `json:"diff"`
	DiffTruncated bool            `json:"diffTruncated"`
	AheadBehind   string          `json:"aheadBehind"`
	Message       string          `json:"message"`
	GeneratedAt   string          `json:"generatedAt"`
}

type GitFileChange struct {
	Path     string `json:"path"`
	OldPath  string `json:"oldPath"`
	Index    string `json:"index"`
	Worktree string `json:"worktree"`
	Summary  string `json:"summary"`
}

func (a *App) GetGitStatus() (GitStatus, error) {
	root := a.getWorkspaceRoot()
	if root == "" {
		return GitStatus{Available: false, Message: "Open a workspace before reading git status."}, nil
	}

	repoRoot, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return GitStatus{Available: false, Message: "Workspace is not inside a git repository.", GeneratedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}

	branch := strings.TrimSpace(mustGitOutput(root, "branch", "--show-current"))
	if branch == "" {
		branch = "detached"
	}
	head := strings.TrimSpace(mustGitOutput(root, "rev-parse", "--short", "HEAD"))
	statusText := mustGitOutput(root, "status", "--porcelain=v1", "--branch")
	changedFiles, aheadBehind := parseGitStatus(statusText)
	diff, truncated := gitDiff(root)

	return GitStatus{
		Available:     true,
		RepoRoot:      strings.TrimSpace(repoRoot),
		Branch:        branch,
		Head:          head,
		Dirty:         len(changedFiles) > 0,
		ChangedFiles:  changedFiles,
		Diff:          diff,
		DiffTruncated: truncated,
		AheadBehind:   aheadBehind,
		Message:       gitStatusMessage(branch, changedFiles),
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func gitDiff(root string) (string, bool) {
	diff := mustGitOutput(root, "diff", "--no-ext-diff", "--unified=80", "--")
	if len(diff) <= gitDiffMaxBytes {
		return diff, false
	}
	return diff[:gitDiffMaxBytes], true
}

func parseGitStatus(statusText string) ([]GitFileChange, string) {
	var changes []GitFileChange
	aheadBehind := ""
	for _, line := range strings.Split(statusText, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			aheadBehind = parseAheadBehind(line)
			continue
		}
		if len(line) < 4 {
			continue
		}
		index := strings.TrimSpace(line[0:1])
		worktree := strings.TrimSpace(line[1:2])
		pathText := strings.TrimSpace(line[3:])
		oldPath := ""
		if strings.Contains(pathText, " -> ") {
			parts := strings.SplitN(pathText, " -> ", 2)
			oldPath = strings.TrimSpace(parts[0])
			pathText = strings.TrimSpace(parts[1])
		}
		changes = append(changes, GitFileChange{
			Path:     pathText,
			OldPath:  oldPath,
			Index:    index,
			Worktree: worktree,
			Summary:  gitChangeSummary(index, worktree),
		})
	}
	return changes, aheadBehind
}

func parseAheadBehind(branchLine string) string {
	start := strings.Index(branchLine, "[")
	end := strings.LastIndex(branchLine, "]")
	if start < 0 || end <= start {
		return ""
	}
	return strings.TrimSpace(branchLine[start+1 : end])
}

func gitChangeSummary(index string, worktree string) string {
	status := strings.TrimSpace(index + worktree)
	switch {
	case strings.Contains(status, "??"):
		return "untracked"
	case strings.Contains(status, "R"):
		return "renamed"
	case strings.Contains(status, "A"):
		return "added"
	case strings.Contains(status, "D"):
		return "deleted"
	case strings.Contains(status, "M"):
		return "modified"
	default:
		return "changed"
	}
}

func gitStatusMessage(branch string, changes []GitFileChange) string {
	if len(changes) == 0 {
		return "Working tree clean on " + branch + "."
	}
	return strings.TrimSpace(branch + " has " + pluralize(len(changes), "changed file") + ".")
}

func pluralize(count int, label string) string {
	if count == 1 {
		return "1 " + label
	}
	return fmt.Sprintf("%d %ss", count, strings.TrimSuffix(label, "s"))
}

func gitOutput(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root}, args...)...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return "", errors.New(detail)
	}
	return stdout.String(), nil
}

func mustGitOutput(root string, args ...string) string {
	output, err := gitOutput(root, args...)
	if err != nil {
		return ""
	}
	return output
}
