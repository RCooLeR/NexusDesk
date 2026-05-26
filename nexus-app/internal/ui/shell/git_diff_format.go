package shell

import (
	"strings"

	gitSvc "nexusdesk/internal/services/git"
)

type gitDiffMode string

const (
	gitDiffModeUnified  gitDiffMode = "unified"
	gitDiffModeSplit    gitDiffMode = "split"
	gitDiffModeDiffOnly gitDiffMode = "diff-only"
)

func (m gitDiffMode) Label() string {
	switch m {
	case gitDiffModeSplit:
		return "Split"
	case gitDiffModeDiffOnly:
		return "Diff only"
	default:
		return "Unified"
	}
}

func gitDiffModeLabels() []string {
	return []string{gitDiffModeUnified.Label(), gitDiffModeSplit.Label(), gitDiffModeDiffOnly.Label()}
}

func gitDiffModeFromLabel(label string) gitDiffMode {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "split":
		return gitDiffModeSplit
	case "diff only":
		return gitDiffModeDiffOnly
	default:
		return gitDiffModeUnified
	}
}

func formatGitDiff(diff gitSvc.FileDiff, mode gitDiffMode) string {
	sections := gitDiffSections(diff)
	if len(sections) == 0 {
		return "No staged or unstaged diff for " + diff.Path + "."
	}

	var builder strings.Builder
	for index, section := range sections {
		if index > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(section.title)
		builder.WriteString(" / ")
		builder.WriteString(diff.Path)
		builder.WriteString("\n")
		switch mode {
		case gitDiffModeSplit:
			builder.WriteString(formatSplitDiff(section.text))
		case gitDiffModeDiffOnly:
			builder.WriteString(formatDiffOnly(section.text))
		default:
			builder.WriteString(section.text)
		}
		if !strings.HasSuffix(builder.String(), "\n") {
			builder.WriteString("\n")
		}
	}
	if diff.StagedDiffTruncated || diff.UnstagedDiffTruncated {
		builder.WriteString("\nDiff output was truncated.\n")
	}
	return builder.String()
}

type gitDiffSection struct {
	title string
	text  string
}

func gitDiffSections(diff gitSvc.FileDiff) []gitDiffSection {
	var sections []gitDiffSection
	if diff.StagedDiff != "" {
		sections = append(sections, gitDiffSection{title: "Staged diff", text: diff.StagedDiff})
	}
	if diff.UnstagedDiff != "" {
		sections = append(sections, gitDiffSection{title: "Unstaged diff", text: diff.UnstagedDiff})
	}
	return sections
}

func formatSplitDiff(diff string) string {
	var builder strings.Builder
	builder.WriteString("Old\tNew\n")
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case line == "":
			continue
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "index "):
			builder.WriteString(line)
			builder.WriteString("\n")
		case strings.HasPrefix(line, "@@"):
			builder.WriteString(line)
			builder.WriteString("\n")
		case strings.HasPrefix(line, "-"):
			builder.WriteString(strings.TrimPrefix(line, "-"))
			builder.WriteString("\t\n")
		case strings.HasPrefix(line, "+"):
			builder.WriteString("\t")
			builder.WriteString(strings.TrimPrefix(line, "+"))
			builder.WriteString("\n")
		case strings.HasPrefix(line, " "):
			text := strings.TrimPrefix(line, " ")
			builder.WriteString(text)
			builder.WriteString("\t")
			builder.WriteString(text)
			builder.WriteString("\n")
		default:
			builder.WriteString(line)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func formatDiffOnly(diff string) string {
	var builder strings.Builder
	builder.WriteString("Old\tNew\n")
	var deletes []string
	var adds []string
	flush := func() {
		maxRows := len(deletes)
		if len(adds) > maxRows {
			maxRows = len(adds)
		}
		for index := 0; index < maxRows; index++ {
			if index < len(deletes) {
				builder.WriteString(deletes[index])
			}
			builder.WriteString("\t")
			if index < len(adds) {
				builder.WriteString(adds[index])
			}
			builder.WriteString("\n")
		}
		deletes = nil
		adds = nil
	}
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "index "):
			continue
		case strings.HasPrefix(line, "@@"):
			flush()
			builder.WriteString(line)
			builder.WriteString("\n")
		case strings.HasPrefix(line, "-"):
			deletes = append(deletes, strings.TrimPrefix(line, "-"))
		case strings.HasPrefix(line, "+"):
			adds = append(adds, strings.TrimPrefix(line, "+"))
		default:
			flush()
		}
	}
	flush()
	if strings.TrimSpace(builder.String()) == "Old\tNew" {
		return "No changed lines in this diff view.\n"
	}
	return builder.String()
}
