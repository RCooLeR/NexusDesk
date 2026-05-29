package git

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const (
	RevertScopeWorktree  = "worktree"
	RevertScopeUntracked = "untracked"
	RevertScopeStaged    = "staged"
)

func (s *Service) PlanRevertChanges(root string, relPath string, scope string) (RevertPlan, error) {
	generatedAt := time.Now().UTC()
	root = strings.TrimSpace(root)
	scope = normalizeRevertScope(scope)
	result := RevertPlan{Path: relPath, Scope: scope, GeneratedAt: generatedAt}
	if root == "" {
		result.Message = "Open a workspace before reverting Git changes."
		return result, nil
	}
	cleanPath, err := cleanRelPath(relPath)
	if err != nil {
		result.Message = err.Error()
		return result, nil
	}
	result.Path = cleanPath
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return RevertPlan{}, err
	}
	status, err := s.Status(absRoot)
	if err != nil {
		return RevertPlan{}, err
	}
	result.Status = status
	if !status.Available {
		result.Message = status.Message
		return result, nil
	}
	change, ok := findGitChange(status.ChangedFiles, cleanPath)
	if !ok {
		result.Message = "No changed Git path matched " + cleanPath + "."
		return result, nil
	}
	result.Change = change
	if change.Index == "?" || change.Worktree == "?" {
		if scope != RevertScopeUntracked {
			result.Message = "Untracked files require scope=untracked before deletion."
			return result, nil
		}
		result.Action = RevertActionDelete
		result.Message = "Prepared to delete untracked file " + cleanPath + " with rollback."
		return result, nil
	}
	if change.Index != "" {
		result.Message = "Staged changes are not reverted by this tool; unstage first or use a future staged-revert workflow."
		return result, nil
	}
	if change.Worktree == "" {
		result.Message = "No unstaged worktree changes are available for " + cleanPath + "."
		return result, nil
	}
	if scope != RevertScopeWorktree {
		result.Message = fmt.Sprintf("scope=%s is not valid for tracked worktree changes.", scope)
		return result, nil
	}
	content, err := gitOutput(absRoot, "show", "HEAD:"+cleanPath)
	if err != nil {
		result.Message = "Could not read HEAD content for " + cleanPath + ": " + err.Error()
		return result, nil
	}
	if strings.ContainsRune(content, '\x00') {
		result.Message = "Binary Git content is not supported by rollback-backed text revert."
		return result, nil
	}
	result.Action = RevertActionWrite
	result.Content = content
	result.Message = "Prepared to restore " + cleanPath + " from HEAD with rollback."
	return result, nil
}

func normalizeRevertScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	switch scope {
	case "", RevertScopeWorktree, "unstaged":
		return RevertScopeWorktree
	case RevertScopeUntracked:
		return RevertScopeUntracked
	default:
		return scope
	}
}

func (s *Service) PlanRevertStagedChanges(root string, relPath string, scope string) (RevertPlan, error) {
	generatedAt := time.Now().UTC()
	root = strings.TrimSpace(root)
	scope = strings.ToLower(strings.TrimSpace(scope))
	result := RevertPlan{Path: relPath, Scope: scope, GeneratedAt: generatedAt}
	if root == "" {
		result.Message = "Open a workspace before reverting staged Git changes."
		return result, nil
	}
	if scope != RevertScopeStaged {
		result.Message = "scope=staged is required before discarding staged changes."
		return result, nil
	}
	cleanPath, err := cleanRelPath(relPath)
	if err != nil {
		result.Message = err.Error()
		return result, nil
	}
	result.Path = cleanPath
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return RevertPlan{}, err
	}
	status, err := s.Status(absRoot)
	if err != nil {
		return RevertPlan{}, err
	}
	result.Status = status
	if !status.Available {
		result.Message = status.Message
		return result, nil
	}
	change, ok := findGitChange(status.ChangedFiles, cleanPath)
	if !ok {
		result.Message = "No changed Git path matched " + cleanPath + "."
		return result, nil
	}
	result.Change = change
	if change.Index == "" || change.Index == "?" {
		result.Message = "No staged changes are available for " + cleanPath + "."
		return result, nil
	}
	if change.Worktree != "" {
		result.Message = "Cannot safely discard staged changes for " + cleanPath + " while unstaged edits also exist."
		return result, nil
	}
	if change.OldPath != "" || change.Index == "R" {
		result.Message = "Staged rename discards are not supported by the safe single-file revert workflow."
		return result, nil
	}
	diff, truncated := cappedGitOutput(absRoot, "diff", "--cached", "--no-ext-diff", "--unified="+diffContextLines, "--", cleanPath)
	result.Diff = diff
	if truncated {
		result.Message = "Staged diff is too large for a safe discard preview."
		return result, nil
	}
	if change.Index == "A" {
		result.Action = RevertActionDelete
		result.Message = "Prepared to unstage and delete staged added file " + cleanPath + " with rollback."
		return result, nil
	}
	content, err := gitOutput(absRoot, "show", "HEAD:"+cleanPath)
	if err != nil {
		result.Message = "Could not read HEAD content for " + cleanPath + ": " + err.Error()
		return result, nil
	}
	if strings.ContainsRune(content, '\x00') {
		result.Message = "Binary Git content is not supported by rollback-backed staged revert."
		return result, nil
	}
	result.Action = RevertActionWrite
	result.Content = content
	result.Message = "Prepared to unstage and restore " + cleanPath + " from HEAD with rollback."
	return result, nil
}

func findGitChange(changes []FileChange, relPath string) (FileChange, bool) {
	for _, change := range changes {
		if change.Path == relPath || change.OldPath == relPath {
			return change, true
		}
	}
	return FileChange{}, false
}
