package shell

import (
	"path"
	"strings"

	gitSvc "nexusdesk/internal/services/git"
)

const gitChangedDirectoryBadge = "*"

func gitWorkspaceBadges(status gitSvc.Status) map[string]string {
	badges := map[string]string{}
	if !status.Available {
		return badges
	}
	for _, change := range status.ChangedFiles {
		addGitChangeBadge(badges, change.Path, gitChangeBadge(change))
		if change.OldPath != "" {
			addGitChangeBadge(badges, change.OldPath, "R")
		}
	}
	return badges
}

func addGitChangeBadge(badges map[string]string, relPath string, badge string) {
	relPath = strings.Trim(strings.TrimSpace(relPath), "/")
	if relPath == "" {
		return
	}
	if badge != "" {
		badges[relPath] = badge
	}
	for directory := path.Dir(relPath); directory != "." && directory != "/"; directory = path.Dir(directory) {
		if badges[directory] == "" {
			badges[directory] = gitChangedDirectoryBadge
		}
	}
}

func gitChangeBadge(change gitSvc.FileChange) string {
	if change.OldPath != "" || change.Index == "R" || change.Worktree == "R" {
		return "R"
	}
	if change.Index == "?" || change.Worktree == "?" {
		return "?"
	}
	for _, status := range []string{change.Worktree, change.Index} {
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case "A", "D", "M":
			return strings.ToUpper(strings.TrimSpace(status))
		}
	}
	return "!"
}
