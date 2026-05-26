package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const gitCommandTimeout = 4 * time.Second
const gitDiffMaxBytes = 220 * 1024
const gitDiffContextLines = "3"
const gitFileActionStage = "stage"
const gitFileActionUnstage = "unstage"
const gitHunkActionStage = "stage"
const gitHunkActionUnstage = "unstage"
const gitHunkActionDiscard = "discard"
const gitHunkActionRevert = "revert"
const gitDiffKindStaged = "staged"
const gitDiffKindUnstaged = "unstaged"

type GitService struct {
	workspaceRoot func() string
}

func newGitService(workspaceRoot func() string) GitService {
	return GitService{workspaceRoot: workspaceRoot}
}

func (s GitService) Status() (GitStatus, error) {
	root := s.workspaceRoot()
	if root == "" {
		return unavailableGitStatus("Open a workspace before reading git status."), nil
	}

	repoRoot, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return unavailableGitStatus("Workspace is not inside a git repository."), nil
	}

	branch := strings.TrimSpace(mustGitOutput(root, "branch", "--show-current"))
	if branch == "" {
		branch = "detached"
	}
	head := strings.TrimSpace(mustGitOutput(root, "rev-parse", "--short", "HEAD"))
	statusText := mustGitOutput(root, "status", "--porcelain=v1", "--branch")
	changedFiles, aheadBehind := parseGitStatus(statusText)
	stagedFiles, unstagedFiles := splitGitChanges(changedFiles)
	unstagedDiff, unstagedTruncated := gitDiff(root)
	stagedDiff, stagedTruncated := gitStagedDiff(root)

	return GitStatus{
		Available:             true,
		RepoRoot:              strings.TrimSpace(repoRoot),
		Branch:                branch,
		Head:                  head,
		Dirty:                 len(changedFiles) > 0,
		ChangedFiles:          changedFiles,
		StagedFiles:           stagedFiles,
		UnstagedFiles:         unstagedFiles,
		Diff:                  unstagedDiff,
		DiffTruncated:         unstagedTruncated,
		StagedDiff:            stagedDiff,
		StagedDiffTruncated:   stagedTruncated,
		UnstagedDiff:          unstagedDiff,
		UnstagedDiffTruncated: unstagedTruncated,
		AheadBehind:           aheadBehind,
		Message:               gitStatusMessage(branch, changedFiles),
		GeneratedAt:           time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s GitService) FileDiff(relPath string) (GitFileDiff, error) {
	root := s.workspaceRoot()
	if root == "" {
		return GitFileDiff{Message: "Open a workspace before reading git diff."}, nil
	}

	cleanPath, err := cleanGitRelPath(relPath)
	if err != nil {
		return GitFileDiff{Path: relPath, Message: err.Error(), GeneratedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	if _, err := gitOutput(root, "rev-parse", "--show-toplevel"); err != nil {
		return GitFileDiff{Path: cleanPath, Message: "Workspace is not inside a git repository.", GeneratedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}

	unstagedDiff, unstagedTruncated := gitFileDiff(root, cleanPath)
	stagedDiff, stagedTruncated := gitStagedFileDiff(root, cleanPath)
	message := "No diff for " + cleanPath + "."
	if stagedDiff != "" || unstagedDiff != "" {
		message = "Loaded read-only diff for " + cleanPath + "."
	}
	return GitFileDiff{
		Path:                  cleanPath,
		StagedDiff:            stagedDiff,
		StagedDiffTruncated:   stagedTruncated,
		UnstagedDiff:          unstagedDiff,
		UnstagedDiffTruncated: unstagedTruncated,
		Message:               message,
		GeneratedAt:           time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s GitService) PreviewFileAction(request GitFileActionRequest) (GitFileActionPreview, error) {
	return s.prepareFileAction(request, false)
}

func (s GitService) ApplyFileAction(request GitFileActionRequest) (GitFileActionPreview, error) {
	return s.prepareFileAction(request, true)
}

func (s GitService) prepareFileAction(request GitFileActionRequest, apply bool) (GitFileActionPreview, error) {
	action := strings.ToLower(strings.TrimSpace(request.Action))
	preview := newGitFileActionPreview(request.Path, action)

	root := s.workspaceRoot()
	if root == "" {
		preview.Message = "Open a workspace before previewing Git actions."
		return preview, nil
	}

	cleanPath, err := cleanGitRelPath(request.Path)
	if err != nil {
		preview.Message = err.Error()
		return preview, nil
	}
	preview.Path = cleanPath

	command, err := gitFileActionCommand(action, cleanPath)
	if err != nil {
		preview.Message = err.Error()
		return preview, nil
	}
	preview.Command = command

	if _, err := gitOutput(root, "rev-parse", "--show-toplevel"); err != nil {
		preview.Message = "Workspace is not inside a git repository."
		return preview, nil
	}

	if apply {
		if _, err := gitOutput(root, command[1:]...); err != nil {
			return preview, err
		}
		status, err := s.Status()
		if err != nil {
			return preview, err
		}
		preview.Status = status
		preview.Message = "Applied " + action + " for " + cleanPath + "."
		return preview, nil
	}

	status, err := s.Status()
	if err != nil {
		return preview, err
	}
	preview.Status = status
	preview.Message = "Preview only. Approval is required before running " + strings.Join(command, " ") + "."
	return preview, nil
}

func newGitFileActionPreview(path string, action string) GitFileActionPreview {
	return GitFileActionPreview{
		Path:              strings.TrimSpace(path),
		Action:            action,
		Command:           []string{},
		RequiresApproval:  true,
		MutatesRepository: true,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
	}
}

func (s GitService) PreviewHunkAction(request GitHunkActionRequest) (GitHunkActionPreview, error) {
	return s.prepareHunkAction(request, false)
}

func (s GitService) ApplyHunkAction(request GitHunkActionRequest) (GitHunkActionPreview, error) {
	return s.prepareHunkAction(request, true)
}

func (s GitService) prepareHunkAction(request GitHunkActionRequest, apply bool) (GitHunkActionPreview, error) {
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	action := strings.ToLower(strings.TrimSpace(request.Action))
	diffKind := strings.ToLower(strings.TrimSpace(request.DiffKind))
	preview := GitHunkActionPreview{
		Path:              strings.TrimSpace(request.Path),
		Action:            action,
		DiffKind:          diffKind,
		HunkIndex:         request.HunkIndex,
		Command:           []string{},
		RequiresApproval:  true,
		MutatesRepository: true,
		GeneratedAt:       generatedAt,
	}

	root := s.workspaceRoot()
	if root == "" {
		preview.Message = "Open a workspace before previewing Git hunk actions."
		return preview, nil
	}
	cleanPath, err := cleanGitRelPath(request.Path)
	if err != nil {
		preview.Message = err.Error()
		return preview, nil
	}
	preview.Path = cleanPath
	if request.HunkIndex <= 0 {
		preview.Message = "Select a diff hunk before previewing this action."
		return preview, nil
	}
	command, err := gitHunkActionCommand(action, diffKind)
	if err != nil {
		preview.Message = err.Error()
		return preview, nil
	}
	preview.Command = command
	if _, err := gitOutput(root, "rev-parse", "--show-toplevel"); err != nil {
		preview.Message = "Workspace is not inside a git repository."
		return preview, nil
	}

	fullDiff := ""
	switch diffKind {
	case gitDiffKindStaged:
		fullDiff, _ = gitStagedFileDiff(root, cleanPath)
	case gitDiffKindUnstaged:
		fullDiff, _ = gitFileDiff(root, cleanPath)
	default:
		preview.Message = "Unsupported hunk diff kind."
		return preview, nil
	}

	patch, err := extractGitHunkPatch(fullDiff, request.HunkIndex)
	if err != nil {
		preview.Message = err.Error()
		return preview, nil
	}
	preview.Patch = patch

	if apply {
		if err := gitApplyPatch(root, command[2:], patch); err != nil {
			return preview, err
		}
		status, err := s.Status()
		if err != nil {
			return preview, err
		}
		preview.Status = status
		preview.Message = "Applied " + action + " for hunk " + fmt.Sprintf("%d", request.HunkIndex) + " in " + cleanPath + "."
		return preview, nil
	}

	status, err := s.Status()
	if err != nil {
		return preview, err
	}
	preview.Status = status
	preview.Message = "Preview only. Approval is required before running " + strings.Join(command, " ") + " for hunk " + fmt.Sprintf("%d", request.HunkIndex) + "."
	return preview, nil
}

func unavailableGitStatus(message string) GitStatus {
	return GitStatus{
		Available:     false,
		ChangedFiles:  []GitFileChange{},
		StagedFiles:   []GitFileChange{},
		UnstagedFiles: []GitFileChange{},
		Message:       message,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
	}
}

func gitDiff(root string) (string, bool) {
	diff := mustGitOutput(root, "diff", "--no-ext-diff", "--unified="+gitDiffContextLines, "--")
	return limitGitDiff(diff)
}

func gitStagedDiff(root string) (string, bool) {
	diff := mustGitOutput(root, "diff", "--cached", "--no-ext-diff", "--unified="+gitDiffContextLines, "--")
	return limitGitDiff(diff)
}

func gitFileDiff(root string, relPath string) (string, bool) {
	diff := mustGitOutput(root, "diff", "--no-ext-diff", "--unified="+gitDiffContextLines, "--", relPath)
	return limitGitDiff(diff)
}

func gitStagedFileDiff(root string, relPath string) (string, bool) {
	diff := mustGitOutput(root, "diff", "--cached", "--no-ext-diff", "--unified="+gitDiffContextLines, "--", relPath)
	return limitGitDiff(diff)
}

func gitFileActionCommand(action string, relPath string) ([]string, error) {
	switch action {
	case gitFileActionStage:
		return []string{"git", "add", "--", relPath}, nil
	case gitFileActionUnstage:
		return []string{"git", "restore", "--staged", "--", relPath}, nil
	default:
		return nil, errors.New("unsupported git file action")
	}
}

func gitHunkActionCommand(action string, diffKind string) ([]string, error) {
	switch {
	case action == gitHunkActionStage && diffKind == gitDiffKindUnstaged:
		return []string{"git", "apply", "--cached", "--whitespace=nowarn"}, nil
	case action == gitHunkActionUnstage && diffKind == gitDiffKindStaged:
		return []string{"git", "apply", "--cached", "--reverse", "--whitespace=nowarn"}, nil
	case action == gitHunkActionDiscard && diffKind == gitDiffKindUnstaged:
		return []string{"git", "apply", "--reverse", "--whitespace=nowarn"}, nil
	case action == gitHunkActionRevert && diffKind == gitDiffKindStaged:
		return []string{"git", "apply", "--cached", "--reverse", "--whitespace=nowarn"}, nil
	default:
		return nil, errors.New("unsupported git hunk action")
	}
}

func extractGitHunkPatch(diff string, hunkIndex int) (string, error) {
	if hunkIndex <= 0 {
		return "", errors.New("git hunk index must be greater than zero")
	}
	lines := strings.Split(strings.ReplaceAll(diff, "\r\n", "\n"), "\n")
	header := []string{}
	hunk := []string{}
	currentHunk := 0
	inTargetHunk := false
	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			currentHunk += 1
			if inTargetHunk {
				break
			}
			inTargetHunk = currentHunk == hunkIndex
		}
		if currentHunk == 0 {
			header = append(header, line)
			continue
		}
		if inTargetHunk {
			hunk = append(hunk, line)
		}
	}
	if len(header) == 0 || len(hunk) == 0 {
		return "", errors.New("selected hunk is no longer available in the current diff")
	}
	patchLines := append([]string{}, header...)
	patchLines = append(patchLines, hunk...)
	return strings.TrimRight(strings.Join(patchLines, "\n"), "\n") + "\n", nil
}

func gitApplyPatch(root string, args []string, patch string) error {
	ctx, cancel := context.WithTimeout(context.Background(), gitCommandTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "git", append([]string{"-C", root, "apply"}, args...)...)
	configureHiddenCommand(command)
	command.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func limitGitDiff(diff string) (string, bool) {
	if len(diff) <= gitDiffMaxBytes {
		return diff, false
	}
	return diff[:gitDiffMaxBytes], true
}

func splitGitChanges(changes []GitFileChange) ([]GitFileChange, []GitFileChange) {
	staged := []GitFileChange{}
	unstaged := []GitFileChange{}
	for _, change := range changes {
		if change.Index != "" && change.Index != "?" {
			staged = append(staged, change)
		}
		if change.Worktree != "" || change.Index == "?" {
			unstaged = append(unstaged, change)
		}
	}
	return staged, unstaged
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
	configureHiddenCommand(command)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
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

func cleanGitRelPath(value string) (string, error) {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = filepath.ToSlash(value)
	value = strings.TrimPrefix(value, "/")
	if value == "" || value == "." {
		return "", errors.New("git diff path is required")
	}
	if filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, "../") || strings.Contains(value, "/../") || strings.HasSuffix(value, "/..") || strings.HasPrefix(value, "-") {
		return "", errors.New("git diff path must stay inside the workspace")
	}
	return value, nil
}
