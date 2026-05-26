package git

import "strings"

func parseStatus(output string) ([]FileChange, string) {
	changes := []FileChange{}
	aheadBehind := ""
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if strings.HasPrefix(line, "## ") {
			aheadBehind = parseAheadBehind(line)
			continue
		}
		if len(line) < 3 {
			continue
		}
		index := strings.TrimSpace(line[:1])
		worktree := strings.TrimSpace(line[1:2])
		path := strings.TrimSpace(line[3:])
		oldPath := ""
		if strings.Contains(path, " -> ") {
			parts := strings.SplitN(path, " -> ", 2)
			oldPath = strings.TrimSpace(parts[0])
			path = strings.TrimSpace(parts[1])
		}
		changes = append(changes, FileChange{
			Path:     path,
			OldPath:  oldPath,
			Index:    index,
			Worktree: worktree,
			Summary:  changeSummary(index, worktree, oldPath),
		})
	}
	return changes, aheadBehind
}

func parseAheadBehind(line string) string {
	start := strings.Index(line, "[")
	end := strings.LastIndex(line, "]")
	if start < 0 || end <= start {
		return ""
	}
	return strings.TrimSpace(line[start+1 : end])
}

func splitChanges(changes []FileChange) ([]FileChange, []FileChange) {
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

func changeSummary(index string, worktree string, oldPath string) string {
	if oldPath != "" || index == "R" || worktree == "R" {
		return "renamed"
	}
	if index == "?" || worktree == "?" {
		return "untracked"
	}
	if index == "A" || worktree == "A" {
		return "added"
	}
	if index == "D" || worktree == "D" {
		return "deleted"
	}
	if index == "M" || worktree == "M" {
		return "modified"
	}
	return "changed"
}
