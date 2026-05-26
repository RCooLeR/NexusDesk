package gitservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"NexusAugenticStudio/internal/processutil"
)

const gitCommandTimeout = 4 * time.Second
const gitDiffMaxBytes = 220 * 1024
const gitDiffContextLines = "3"
const DefaultHistoryLimit = 20
const gitHistoryMaxLimit = 80
const gitBlameMaxLines = 220
const gitFileActionStage = "stage"
const gitFileActionUnstage = "unstage"
const gitHunkActionStage = "stage"
const gitHunkActionUnstage = "unstage"
const gitHunkActionDiscard = "discard"
const gitHunkActionRevert = "revert"
const gitDiffKindStaged = "staged"
const gitDiffKindUnstaged = "unstaged"

type Service struct {
	workspaceRoot func() string
}

func New(workspaceRoot func() string) Service {
	return Service{workspaceRoot: workspaceRoot}
}

func (s Service) Status() (Status, error) {
	root := s.workspaceRoot()
	if root == "" {
		return unavailableStatus("Open a workspace before reading git status."), nil
	}

	repoRoot, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return unavailableStatus("Workspace is not inside a git repository."), nil
	}

	branch := strings.TrimSpace(mustGitOutput(root, "branch", "--show-current"))
	if branch == "" {
		branch = "detached"
	}
	head := strings.TrimSpace(mustGitOutput(root, "rev-parse", "--short", "HEAD"))
	statusText := mustGitOutput(root, "status", "--porcelain=v1", "--branch")
	changedFiles, aheadBehind := parseStatus(statusText)
	stagedFiles, unstagedFiles := splitGitChanges(changedFiles)
	unstagedDiff, unstagedTruncated := gitDiff(root)
	stagedDiff, stagedTruncated := gitStagedDiff(root)

	return Status{
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

func (s Service) FileDiff(relPath string) (FileDiff, error) {
	root := s.workspaceRoot()
	if root == "" {
		return FileDiff{Message: "Open a workspace before reading git diff."}, nil
	}

	cleanPath, err := cleanGitRelPath(relPath)
	if err != nil {
		return FileDiff{Path: relPath, Message: err.Error(), GeneratedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}
	if _, err := gitOutput(root, "rev-parse", "--show-toplevel"); err != nil {
		return FileDiff{Path: cleanPath, Message: "Workspace is not inside a git repository.", GeneratedAt: time.Now().UTC().Format(time.RFC3339)}, nil
	}

	unstagedDiff, unstagedTruncated := gitFileDiff(root, cleanPath)
	stagedDiff, stagedTruncated := gitStagedFileDiff(root, cleanPath)
	message := "No diff for " + cleanPath + "."
	if stagedDiff != "" || unstagedDiff != "" {
		message = "Loaded read-only diff for " + cleanPath + "."
	}
	return FileDiff{
		Path:                  cleanPath,
		StagedDiff:            stagedDiff,
		StagedDiffTruncated:   stagedTruncated,
		UnstagedDiff:          unstagedDiff,
		UnstagedDiffTruncated: unstagedTruncated,
		Message:               message,
		GeneratedAt:           time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s Service) History(request HistoryRequest) (HistoryResult, error) {
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	result := HistoryResult{
		Path:        cleanOptionalGitRelPath(request.Path),
		Limit:       boundedGitHistoryLimit(request.Limit),
		Entries:     []HistoryEntry{},
		GeneratedAt: generatedAt,
	}
	root := s.workspaceRoot()
	if root == "" {
		result.Message = "Open a workspace before reading Git history."
		return result, nil
	}
	if _, err := gitOutput(root, "rev-parse", "--show-toplevel"); err != nil {
		result.Message = "Workspace is not inside a git repository."
		return result, nil
	}
	args := []string{
		"log",
		fmt.Sprintf("--max-count=%d", result.Limit+1),
		"--date=iso-strict",
		"--pretty=format:%H%x1f%h%x1f%an%x1f%ae%x1f%ad%x1f%s",
	}
	if result.Path != "" {
		cleanPath, err := cleanGitRelPath(result.Path)
		if err != nil {
			result.Message = err.Error()
			return result, nil
		}
		result.Path = cleanPath
		args = append(args, "--", cleanPath)
	}
	output, err := gitOutput(root, args...)
	if err != nil {
		result.Message = "Could not read Git history: " + err.Error()
		return result, nil
	}
	result.Available = true
	result.Entries, result.Truncated = parseGitHistory(output, result.Limit)
	result.Message = fmt.Sprintf("Loaded %d Git history entr%s.", len(result.Entries), pluralSuffix(len(result.Entries), "y", "ies"))
	if result.Path != "" {
		result.Message = fmt.Sprintf("Loaded %d Git history entr%s for %s.", len(result.Entries), pluralSuffix(len(result.Entries), "y", "ies"), result.Path)
	}
	return result, nil
}

func (s Service) Blame(request BlameRequest) (BlameResult, error) {
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	result := BlameResult{
		Path:        cleanOptionalGitRelPath(request.Path),
		Lines:       []BlameLine{},
		GeneratedAt: generatedAt,
	}
	root := s.workspaceRoot()
	if root == "" {
		result.Message = "Open a workspace before reading Git blame."
		return result, nil
	}
	cleanPath, err := cleanGitRelPath(result.Path)
	if err != nil {
		result.Message = err.Error()
		return result, nil
	}
	result.Path = cleanPath
	if _, err := gitOutput(root, "rev-parse", "--show-toplevel"); err != nil {
		result.Message = "Workspace is not inside a git repository."
		return result, nil
	}
	startLine, endLine := boundedGitBlameRange(request.StartLine, request.EndLine)
	result.StartLine = startLine
	result.EndLine = endLine
	args := []string{"blame", "--line-porcelain"}
	if startLine > 0 {
		args = append(args, "-L", fmt.Sprintf("%d,%d", startLine, endLine))
	}
	args = append(args, "--", cleanPath)
	output, err := gitOutput(root, args...)
	if err != nil {
		result.Message = "Could not read Git blame: " + err.Error()
		return result, nil
	}
	result.Available = true
	result.Lines, result.Truncated = parseGitBlame(output, gitBlameMaxLines)
	result.Message = fmt.Sprintf("Loaded %d Git blame line%s for %s.", len(result.Lines), pluralizeWord(len(result.Lines)), cleanPath)
	return result, nil
}

func (s Service) PreviewFileAction(request FileActionRequest) (FileActionPreview, error) {
	return s.prepareFileAction(request, false)
}

func (s Service) ApplyFileAction(request FileActionRequest) (FileActionPreview, error) {
	return s.prepareFileAction(request, true)
}

func (s Service) prepareFileAction(request FileActionRequest, apply bool) (FileActionPreview, error) {
	action := strings.ToLower(strings.TrimSpace(request.Action))
	preview := newFileActionPreview(request.Path, action)

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

func newFileActionPreview(path string, action string) FileActionPreview {
	return FileActionPreview{
		Path:              strings.TrimSpace(path),
		Action:            action,
		Command:           []string{},
		RequiresApproval:  true,
		MutatesRepository: true,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
	}
}

func (s Service) PreviewHunkAction(request HunkActionRequest) (HunkActionPreview, error) {
	return s.prepareHunkAction(request, false)
}

func (s Service) ApplyHunkAction(request HunkActionRequest) (HunkActionPreview, error) {
	return s.prepareHunkAction(request, true)
}

func (s Service) prepareHunkAction(request HunkActionRequest, apply bool) (HunkActionPreview, error) {
	generatedAt := time.Now().UTC().Format(time.RFC3339)
	action := strings.ToLower(strings.TrimSpace(request.Action))
	diffKind := strings.ToLower(strings.TrimSpace(request.DiffKind))
	preview := HunkActionPreview{
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

func unavailableStatus(message string) Status {
	return Status{
		Available:     false,
		ChangedFiles:  []FileChange{},
		StagedFiles:   []FileChange{},
		UnstagedFiles: []FileChange{},
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
	processutil.ConfigureHiddenCommand(command)
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

func splitGitChanges(changes []FileChange) ([]FileChange, []FileChange) {
	staged := []FileChange{}
	unstaged := []FileChange{}
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

func parseStatus(statusText string) ([]FileChange, string) {
	var changes []FileChange
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
		changes = append(changes, FileChange{
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

func gitStatusMessage(branch string, changes []FileChange) string {
	if len(changes) == 0 {
		return "Working tree clean on " + branch + "."
	}
	return strings.TrimSpace(branch + " has " + pluralize(len(changes), "changed file") + ".")
}

func boundedGitHistoryLimit(limit int) int {
	if limit <= 0 {
		return DefaultHistoryLimit
	}
	if limit > gitHistoryMaxLimit {
		return gitHistoryMaxLimit
	}
	return limit
}

func boundedGitBlameRange(startLine int, endLine int) (int, int) {
	if startLine <= 0 {
		return 1, gitBlameMaxLines
	}
	if endLine < startLine {
		endLine = startLine
	}
	if endLine-startLine+1 > gitBlameMaxLines {
		endLine = startLine + gitBlameMaxLines - 1
	}
	return startLine, endLine
}

func parseGitHistory(output string, limit int) ([]HistoryEntry, bool) {
	entries := []HistoryEntry{}
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, "\x1f", 6)
		if len(parts) != 6 {
			continue
		}
		if len(entries) >= limit {
			return entries, true
		}
		entries = append(entries, HistoryEntry{
			Hash:      strings.TrimSpace(parts[0]),
			ShortHash: strings.TrimSpace(parts[1]),
			Author:    strings.TrimSpace(parts[2]),
			Email:     strings.TrimSpace(parts[3]),
			Date:      strings.TrimSpace(parts[4]),
			Subject:   strings.TrimSpace(parts[5]),
		})
	}
	return entries, false
}

func parseGitBlame(output string, maxLines int) ([]BlameLine, bool) {
	lines := []BlameLine{}
	var current BlameLine
	for _, rawLine := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(rawLine) == "" && current.Hash == "" {
			continue
		}
		if strings.HasPrefix(rawLine, "\t") {
			if len(lines) >= maxLines {
				return lines, true
			}
			current.Content = strings.TrimPrefix(rawLine, "\t")
			current.ShortHash = shortGitHash(current.Hash)
			lines = append(lines, current)
			current = BlameLine{}
			continue
		}
		parts := strings.SplitN(rawLine, " ", 2)
		key := parts[0]
		value := ""
		if len(parts) == 2 {
			value = strings.TrimSpace(parts[1])
		}
		switch key {
		case "author":
			current.Author = value
		case "author-time":
			if seconds, err := strconv.ParseInt(value, 10, 64); err == nil {
				current.Date = time.Unix(seconds, 0).UTC().Format(time.RFC3339)
			}
		case "summary":
			current.Summary = value
		default:
			header := strings.Fields(rawLine)
			if len(header) >= 3 && len(header[0]) >= 7 {
				current.Hash = header[0]
				if finalLine, err := strconv.Atoi(header[2]); err == nil {
					current.Line = finalLine
				}
			}
		}
	}
	return lines, false
}

func shortGitHash(hash string) string {
	hash = strings.TrimSpace(hash)
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func pluralSuffix(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func pluralizeWord(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
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
	processutil.ConfigureHiddenCommand(command)
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
	value = cleanOptionalGitRelPath(value)
	if value == "" || value == "." {
		return "", errors.New("git diff path is required")
	}
	if filepath.IsAbs(value) || value == ".." || strings.HasPrefix(value, "../") || strings.Contains(value, "/../") || strings.HasSuffix(value, "/..") || strings.HasPrefix(value, "-") {
		return "", errors.New("git diff path must stay inside the workspace")
	}
	return value, nil
}

func cleanOptionalGitRelPath(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"'`)
	value = filepath.ToSlash(value)
	value = strings.TrimPrefix(value, "/")
	if value == "." {
		return ""
	}
	return value
}
