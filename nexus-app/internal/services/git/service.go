package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const commandTimeout = 4 * time.Second

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Status(root string) (Status, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return unavailableStatus("Open a workspace before reading Git status."), nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Status{}, err
	}
	repoRoot, err := gitOutput(absRoot, "rev-parse", "--show-toplevel")
	if err != nil {
		return unavailableStatus("Workspace is not inside a Git repository."), nil
	}
	branch := strings.TrimSpace(mustGitOutput(absRoot, "branch", "--show-current"))
	if branch == "" {
		branch = "detached"
	}
	head := strings.TrimSpace(mustGitOutput(absRoot, "rev-parse", "--short", "HEAD"))
	statusText := mustGitOutput(absRoot, "status", "--porcelain=v1", "--branch")
	changes, aheadBehind := parseStatus(statusText)
	staged, unstaged := splitChanges(changes)
	return Status{
		Available:     true,
		RepoRoot:      strings.TrimSpace(repoRoot),
		Branch:        branch,
		Head:          head,
		Dirty:         len(changes) > 0,
		ChangedFiles:  changes,
		StagedFiles:   staged,
		UnstagedFiles: unstaged,
		AheadBehind:   aheadBehind,
		Message:       statusMessage(branch, changes),
		GeneratedAt:   time.Now().UTC(),
	}, nil
}

func unavailableStatus(message string) Status {
	return Status{Available: false, Message: message, GeneratedAt: time.Now().UTC()}
}

func gitOutput(root string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = root
	hideGitCommandWindow(command)
	output, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", errors.New("git command timed out")
	}
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			text = err.Error()
		}
		return "", errors.New(text)
	}
	return string(output), nil
}

func mustGitOutput(root string, args ...string) string {
	output, err := gitOutput(root, args...)
	if err != nil {
		return ""
	}
	return output
}

func statusMessage(branch string, changes []FileChange) string {
	if len(changes) == 0 {
		return fmt.Sprintf("%s is clean.", branch)
	}
	return fmt.Sprintf("%s has %d changed file(s).", branch, len(changes))
}
