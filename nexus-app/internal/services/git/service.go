package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const diffMaxBytes = 220 * 1024
const diffContextLines = "3"
const DefaultHistoryLimit = 20
const historyMaxLimit = 80
const blameMaxLines = 220

type operationClass string

const (
	operationStatus   operationClass = "status"
	operationDiff     operationClass = "diff"
	operationHistory  operationClass = "history"
	operationMutation operationClass = "mutation"
)

const (
	statusTimeout   = 4 * time.Second
	diffTimeout     = 8 * time.Second
	historyTimeout  = 12 * time.Second
	mutationTimeout = 20 * time.Second
)

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
	repoRoot, err := gitOutputFor(absRoot, operationStatus, "rev-parse", "--show-toplevel")
	if err != nil {
		return unavailableStatus("Workspace is not inside a Git repository."), nil
	}
	branch := strings.TrimSpace(mustGitOutputFor(absRoot, operationStatus, "branch", "--show-current"))
	if branch == "" {
		branch = "detached"
	}
	head := strings.TrimSpace(mustGitOutputFor(absRoot, operationStatus, "rev-parse", "--short", "HEAD"))
	statusText := mustGitOutputFor(absRoot, operationStatus, "status", "--porcelain=v1", "--branch")
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

func (s *Service) FileDiff(root string, relPath string) (FileDiff, error) {
	generatedAt := time.Now().UTC()
	root = strings.TrimSpace(root)
	if root == "" {
		return FileDiff{Path: relPath, Message: "Open a workspace before reading Git diff.", GeneratedAt: generatedAt}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return FileDiff{}, err
	}
	cleanPath, err := cleanRelPath(relPath)
	if err != nil {
		return FileDiff{Path: relPath, Message: err.Error(), GeneratedAt: generatedAt}, nil
	}
	if _, err := gitOutputFor(absRoot, operationStatus, "rev-parse", "--show-toplevel"); err != nil {
		return FileDiff{Path: cleanPath, Message: "Workspace is not inside a Git repository.", GeneratedAt: generatedAt}, nil
	}
	unstagedDiff, unstagedTruncated := cappedGitOutputFor(absRoot, operationDiff, "diff", "--no-ext-diff", "--unified="+diffContextLines, "--", cleanPath)
	stagedDiff, stagedTruncated := cappedGitOutputFor(absRoot, operationDiff, "diff", "--cached", "--no-ext-diff", "--unified="+diffContextLines, "--", cleanPath)
	message := "No diff for " + cleanPath + "."
	if stagedDiff != "" || unstagedDiff != "" {
		message = "Loaded read-only diff for " + cleanPath + "."
	}
	return FileDiff{
		Path:                  cleanPath,
		StagedDiff:            stagedDiff,
		StagedDiffTruncated:   stagedTruncated,
		StagedHunks:           parseDiffHunks(DiffKindStaged, stagedDiff),
		UnstagedDiff:          unstagedDiff,
		UnstagedDiffTruncated: unstagedTruncated,
		UnstagedHunks:         parseDiffHunks(DiffKindUnstaged, unstagedDiff),
		Message:               message,
		GeneratedAt:           generatedAt,
	}, nil
}

func (s *Service) ApplyFileAction(root string, relPath string, action FileAction) (FileActionResult, error) {
	generatedAt := time.Now().UTC()
	root = strings.TrimSpace(root)
	if root == "" {
		return FileActionResult{Path: relPath, Action: action, Message: "Open a workspace before changing Git state.", GeneratedAt: generatedAt}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return FileActionResult{}, err
	}
	cleanPath, err := cleanRelPath(relPath)
	if err != nil {
		return FileActionResult{Path: relPath, Action: action, Message: err.Error(), GeneratedAt: generatedAt}, nil
	}
	if _, err := gitOutputFor(absRoot, operationStatus, "rev-parse", "--show-toplevel"); err != nil {
		return FileActionResult{Path: cleanPath, Action: action, Message: "Workspace is not inside a Git repository.", GeneratedAt: generatedAt}, nil
	}
	if err := runFileAction(absRoot, cleanPath, action); err != nil {
		return FileActionResult{}, err
	}
	status, err := s.Status(absRoot)
	if err != nil {
		return FileActionResult{}, err
	}
	return FileActionResult{
		Path:        cleanPath,
		Action:      action,
		Message:     fileActionMessage(cleanPath, action),
		Status:      status,
		GeneratedAt: generatedAt,
	}, nil
}

func (s *Service) CommitChanges(root string, message string, body string) (CommitResult, error) {
	generatedAt := time.Now().UTC()
	root = strings.TrimSpace(root)
	subject := strings.TrimSpace(message)
	body = strings.TrimSpace(body)
	if root == "" {
		return CommitResult{Subject: subject, Body: body, Message: "Open a workspace before committing Git changes.", GeneratedAt: generatedAt}, nil
	}
	if subject == "" {
		return CommitResult{Subject: subject, Body: body, Message: "Commit message is required.", GeneratedAt: generatedAt}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return CommitResult{}, err
	}
	if _, err := gitOutputFor(absRoot, operationStatus, "rev-parse", "--show-toplevel"); err != nil {
		return CommitResult{Subject: subject, Body: body, Message: "Workspace is not inside a Git repository.", GeneratedAt: generatedAt}, nil
	}
	stagedStat := strings.TrimSpace(mustGitOutputFor(absRoot, operationDiff, "diff", "--cached", "--stat"))
	if stagedStat == "" {
		return CommitResult{Subject: subject, Body: body, Message: "No staged changes are available to commit.", GeneratedAt: generatedAt}, nil
	}
	args := []string{"commit", "-m", subject}
	if body != "" {
		args = append(args, "-m", body)
	}
	if _, err := gitOutputFor(absRoot, operationMutation, args...); err != nil {
		return CommitResult{}, err
	}
	hash := strings.TrimSpace(mustGitOutputFor(absRoot, operationStatus, "rev-parse", "HEAD"))
	shortHash := strings.TrimSpace(mustGitOutputFor(absRoot, operationStatus, "rev-parse", "--short", "HEAD"))
	status, err := s.Status(absRoot)
	if err != nil {
		return CommitResult{}, err
	}
	return CommitResult{
		Hash:        hash,
		ShortHash:   shortHash,
		Subject:     subject,
		Body:        body,
		StagedStat:  stagedStat,
		Message:     "Committed staged changes.",
		Status:      status,
		GeneratedAt: generatedAt,
	}, nil
}

func (s *Service) CreateBranch(root string, branchName string, startPoint string, checkout bool) (BranchResult, error) {
	generatedAt := time.Now().UTC()
	root = strings.TrimSpace(root)
	branchName = strings.TrimSpace(branchName)
	startPoint = strings.TrimSpace(startPoint)
	if startPoint == "" {
		startPoint = "HEAD"
	}
	result := BranchResult{BranchName: branchName, StartPoint: startPoint, CheckedOut: checkout, GeneratedAt: generatedAt}
	if root == "" {
		result.Message = "Open a workspace before creating Git branches."
		return result, nil
	}
	if branchName == "" {
		result.Message = "branchName is required."
		return result, nil
	}
	if strings.HasPrefix(branchName, "-") || strings.HasPrefix(startPoint, "-") {
		result.Message = "Git branch names and start points must not start with '-'."
		return result, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return BranchResult{}, err
	}
	if _, err := gitOutputFor(absRoot, operationStatus, "rev-parse", "--show-toplevel"); err != nil {
		result.Message = "Workspace is not inside a Git repository."
		return result, nil
	}
	if _, err := gitOutputFor(absRoot, operationStatus, "check-ref-format", "--branch", branchName); err != nil {
		result.Message = "Invalid Git branch name."
		return result, nil
	}
	if branchExists(absRoot, branchName) {
		result.Message = "Git branch already exists."
		return result, nil
	}
	startSHA, err := gitOutputFor(absRoot, operationStatus, "rev-parse", "--verify", "--end-of-options", startPoint+"^{commit}")
	if err != nil {
		result.Message = "Start point is not a valid commit."
		return result, nil
	}
	result.StartPointSHA = strings.TrimSpace(startSHA)
	if _, err := gitOutputFor(absRoot, operationMutation, "branch", "--", branchName, result.StartPointSHA); err != nil {
		return BranchResult{}, err
	}
	if checkout {
		if _, err := gitOutputFor(absRoot, operationMutation, "switch", "--", branchName); err != nil {
			return BranchResult{}, err
		}
	}
	status, err := s.Status(absRoot)
	if err != nil {
		return BranchResult{}, err
	}
	result.Status = status
	if checkout {
		result.Message = "Created and switched to branch " + branchName + "."
	} else {
		result.Message = "Created branch " + branchName + "."
	}
	return result, nil
}

func branchExists(root string, branchName string) bool {
	_, err := gitOutputFor(root, operationStatus, "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	return err == nil
}

func runFileAction(root string, relPath string, action FileAction) error {
	switch action {
	case FileActionStage:
		_, err := gitOutputFor(root, operationMutation, "add", "--", relPath)
		return err
	case FileActionUnstage:
		_, err := gitOutputFor(root, operationMutation, "restore", "--staged", "--", relPath)
		return err
	default:
		return fmt.Errorf("unsupported Git file action %q", action)
	}
}

func fileActionMessage(relPath string, action FileAction) string {
	switch action {
	case FileActionStage:
		return "Staged " + relPath + "."
	case FileActionUnstage:
		return "Unstaged " + relPath + "."
	default:
		return "Updated " + relPath + "."
	}
}

func unavailableStatus(message string) Status {
	return Status{Available: false, Message: message, GeneratedAt: time.Now().UTC()}
}

func gitOutput(root string, args ...string) (string, error) {
	return gitOutputFor(root, operationStatus, args...)
}

func gitOutputFor(root string, class operationClass, args ...string) (string, error) {
	timeout := timeoutForOperation(class)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", args...)
	command.Dir = root
	command.Env = nonInteractiveGitEnv(os.Environ())
	hideGitCommandWindow(command)
	output, err := command.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("git %s command timed out after %s", class, timeout)
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
	return mustGitOutputFor(root, operationStatus, args...)
}

func mustGitOutputFor(root string, class operationClass, args ...string) string {
	output, err := gitOutputFor(root, class, args...)
	if err != nil {
		return ""
	}
	return output
}

func cappedGitOutput(root string, args ...string) (string, bool) {
	return cappedGitOutputFor(root, operationDiff, args...)
}

func cappedGitOutputFor(root string, class operationClass, args ...string) (string, bool) {
	output, err := gitOutputFor(root, class, args...)
	if err != nil {
		return "", false
	}
	return windowUnifiedDiff(output)
}

func timeoutForOperation(class operationClass) time.Duration {
	switch class {
	case operationStatus:
		return statusTimeout
	case operationDiff:
		return diffTimeout
	case operationHistory:
		return historyTimeout
	case operationMutation:
		return mutationTimeout
	default:
		return statusTimeout
	}
}

func nonInteractiveGitEnv(base []string) []string {
	overrides := []string{
		"GIT_TERMINAL_PROMPT=0",
		"GIT_ASKPASS=",
		"SSH_ASKPASS=",
		"GCM_INTERACTIVE=Never",
	}
	overrideKeys := map[string]struct{}{}
	for _, entry := range overrides {
		key, _, _ := strings.Cut(entry, "=")
		overrideKeys[strings.ToUpper(key)] = struct{}{}
	}
	env := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		if _, overridden := overrideKeys[strings.ToUpper(key)]; overridden {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, overrides...)
	return env
}

func statusMessage(branch string, changes []FileChange) string {
	if len(changes) == 0 {
		return fmt.Sprintf("%s is clean.", branch)
	}
	return fmt.Sprintf("%s has %d changed file(s).", branch, len(changes))
}
