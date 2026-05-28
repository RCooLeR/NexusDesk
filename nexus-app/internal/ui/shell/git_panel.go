package shell

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	gitSvc "nexusdesk/internal/services/git"
)

type gitChangeGroup struct {
	Directory string
	Changes   []gitSvc.FileChange
}

type gitHunkTarget struct {
	Kind   gitSvc.DiffKind
	Index  int
	Header string
	Label  string
}

func (v *View) newGitPanel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh git", theme.ViewRefreshIcon(), v.refreshGitStatus)
	header := container.NewBorder(nil, nil, v.gitStatus, refresh)
	scroll := container.NewScroll(v.gitResults)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	diffMode := widget.NewSelect(gitDiffModeLabels(), func(label string) {
		v.gitDiffMode = gitDiffModeFromLabel(label)
		if v.gitLastDiff.Path != "" {
			v.gitDiffText.SetText(formatGitDiff(v.gitLastDiff, v.gitDiffMode))
		}
	})
	diffMode.SetSelected(gitDiffModeUnified.Label())
	stage := widget.NewButtonWithIcon("Stage", theme.ContentAddIcon(), func() {
		v.confirmGitFileAction(gitSvc.FileActionStage)
	})
	unstage := widget.NewButtonWithIcon("Unstage", theme.ContentRemoveIcon(), func() {
		v.confirmGitFileAction(gitSvc.FileActionUnstage)
	})
	stageHunk := widget.NewButtonWithIcon("Stage hunk", theme.ContentAddIcon(), func() {
		v.confirmGitHunkAction(gitSvc.HunkActionStage)
	})
	unstageHunk := widget.NewButtonWithIcon("Unstage hunk", theme.ContentRemoveIcon(), func() {
		v.confirmGitHunkAction(gitSvc.HunkActionUnstage)
	})
	summarize := widget.NewButtonWithIcon("AI summary", theme.InfoIcon(), v.summarizeSelectedGitDiff)
	draftCommit := widget.NewButtonWithIcon("Draft commit", theme.MailComposeIcon(), v.draftSelectedGitCommitMessage)
	history := widget.NewButtonWithIcon("History", theme.HistoryIcon(), v.openSelectedGitHistory)
	blame := widget.NewButtonWithIcon("Blame", theme.VisibilityIcon(), v.openSelectedGitBlame)
	prevHunk := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		v.moveGitHunk(-1)
	})
	prevHunk.Importance = widget.LowImportance
	nextHunk := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		v.moveGitHunk(1)
	})
	nextHunk.Importance = widget.LowImportance
	hunkNav := container.NewHBox(prevHunk, v.gitHunkStatus, nextHunk)
	actions := container.NewHBox(stage, unstage, stageHunk, unstageHunk, history, blame, summarize, draftCommit, diffMode)
	diffHeader := container.NewBorder(nil, nil, hunkNav, actions, v.gitDiffStatus)
	diff := container.NewBorder(diffHeader, nil, nil, nil, v.gitDiffText)
	split := container.NewVSplit(scroll, diff)
	split.Offset = 0.42
	return container.NewBorder(header, nil, nil, nil, split)
}

func (v *View) refreshGitStatus() {
	workspace := v.state.Workspace()
	status, err := v.gitService.Status(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.applyGitStatus(status)
	v.addActivity(status.Message)
}

func (v *View) applyGitStatus(status gitSvc.Status) {
	v.gitStatus.SetText(gitStatusLabel(status))
	v.gitResults.Objects = v.gitRows(status)
	v.gitResults.Refresh()
	v.gitFileBadges = gitWorkspaceBadges(status)
	if v.state.Workspace().Root != "" {
		v.refreshNavigator()
	}
}

func (v *View) openGitDiff(path string) {
	workspace := v.state.Workspace()
	diff, err := v.gitService.FileDiff(workspace.Root, path)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.gitLastDiff = diff
	v.gitActiveHunk = 0
	v.gitDiffStatus.SetText(diff.Message)
	v.gitDiffText.SetText(formatGitDiff(diff, v.gitDiffMode))
	v.updateGitHunkStatus()
	v.addActivity(diff.Message)
}

func (v *View) openSelectedGitHistory() {
	target := v.currentGitTargetPath()
	workspace := v.state.Workspace()
	result, err := v.gitService.History(workspace.Root, target, gitSvc.DefaultHistoryLimit)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.gitDiffStatus.SetText(result.Message)
	v.gitDiffText.SetText(formatGitHistory(result))
	v.addActivity(result.Message)
}

func (v *View) openSelectedGitBlame() {
	target := v.currentGitTargetPath()
	if strings.TrimSpace(target) == "" {
		v.addActivity("Select a file before reading Git blame.")
		return
	}
	workspace := v.state.Workspace()
	result, err := v.gitService.Blame(workspace.Root, target, 1, gitSvc.DefaultHistoryLimit)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.gitDiffStatus.SetText(result.Message)
	v.gitDiffText.SetText(formatGitBlame(result))
	v.addActivity(result.Message)
}

func (v *View) currentGitTargetPath() string {
	if strings.TrimSpace(v.gitLastDiff.Path) != "" {
		return v.gitLastDiff.Path
	}
	return strings.TrimSpace(v.state.SelectedPath())
}

func (v *View) moveGitHunk(delta int) {
	targets := gitHunkTargets(v.gitLastDiff)
	if len(targets) == 0 {
		v.addActivity("No diff hunks are available for the selected file.")
		return
	}
	v.gitActiveHunk = (v.gitActiveHunk + delta + len(targets)) % len(targets)
	target := targets[v.gitActiveHunk]
	v.updateGitHunkStatus()
	v.addActivity("Selected " + target.Label + ".")
}

func (v *View) updateGitHunkStatus() {
	targets := gitHunkTargets(v.gitLastDiff)
	if len(targets) == 0 {
		v.gitHunkStatus.SetText("No hunks")
		return
	}
	if v.gitActiveHunk < 0 || v.gitActiveHunk >= len(targets) {
		v.gitActiveHunk = 0
	}
	v.gitHunkStatus.SetText(fmt.Sprintf("%d / %d %s", v.gitActiveHunk+1, len(targets), targets[v.gitActiveHunk].Label))
}

func (v *View) confirmGitFileAction(action gitSvc.FileAction) {
	if v.gitLastDiff.Path == "" {
		v.addActivity("Select a changed file before running a Git action.")
		return
	}
	title := "Stage file"
	message := "Stage " + v.gitLastDiff.Path + "?"
	if action == gitSvc.FileActionUnstage {
		title = "Unstage file"
		message = "Unstage " + v.gitLastDiff.Path + "?"
	}
	dialog.ShowConfirm(title, message, func(confirm bool) {
		if !confirm {
			return
		}
		v.applyGitFileAction(action)
	}, v.window)
}

func (v *View) applyGitFileAction(action gitSvc.FileAction) {
	workspace := v.state.Workspace()
	result, err := v.gitService.ApplyFileAction(workspace.Root, v.gitLastDiff.Path, action)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.applyGitStatus(result.Status)
	v.addActivity(result.Message)
	v.openGitDiff(result.Path)
}

func (v *View) confirmGitHunkAction(action gitSvc.HunkAction) {
	targets := gitHunkTargets(v.gitLastDiff)
	if v.gitLastDiff.Path == "" || len(targets) == 0 {
		v.addActivity("Select a changed file and hunk before running a Git hunk action.")
		return
	}
	if v.gitActiveHunk < 0 || v.gitActiveHunk >= len(targets) {
		v.gitActiveHunk = 0
	}
	target := targets[v.gitActiveHunk]
	if !validGitHunkAction(action, target.Kind) {
		v.addActivity("Selected hunk cannot be used with that action.")
		return
	}
	title := "Stage hunk"
	message := "Stage " + target.Label + " in " + v.gitLastDiff.Path + "?"
	if action == gitSvc.HunkActionUnstage {
		title = "Unstage hunk"
		message = "Unstage " + target.Label + " in " + v.gitLastDiff.Path + "?"
	}
	dialog.ShowConfirm(title, message, func(confirm bool) {
		if !confirm {
			return
		}
		v.applyGitHunkAction(target, action)
	}, v.window)
}

func (v *View) applyGitHunkAction(target gitHunkTarget, action gitSvc.HunkAction) {
	workspace := v.state.Workspace()
	result, err := v.gitService.ApplyHunkAction(workspace.Root, v.gitLastDiff.Path, target.Kind, target.Index, action)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	if result.Message != "" {
		v.addActivity(result.Message)
	}
	if result.Status.Available {
		v.applyGitStatus(result.Status)
	}
	v.openGitDiff(result.Path)
}

func validGitHunkAction(action gitSvc.HunkAction, kind gitSvc.DiffKind) bool {
	return (action == gitSvc.HunkActionStage && kind == gitSvc.DiffKindUnstaged) ||
		(action == gitSvc.HunkActionUnstage && kind == gitSvc.DiffKindStaged)
}

func gitStatusLabel(status gitSvc.Status) string {
	if !status.Available {
		return status.Message
	}
	head := status.Head
	if head == "" {
		head = "no HEAD"
	}
	return fmt.Sprintf("%s @ %s - %d changed", status.Branch, head, len(status.ChangedFiles))
}

func (v *View) gitRows(status gitSvc.Status) []fyne.CanvasObject {
	if !status.Available {
		return []fyne.CanvasObject{widget.NewLabel(status.Message)}
	}
	rows := []fyne.CanvasObject{
		widget.NewLabel(status.Message),
		widget.NewLabel(fmt.Sprintf("%d staged / %d unstaged", len(status.StagedFiles), len(status.UnstagedFiles))),
	}
	if status.AheadBehind != "" {
		rows = append(rows, widget.NewLabel(status.AheadBehind))
	}
	if len(status.ChangedFiles) == 0 {
		rows = append(rows, widget.NewLabel("Working tree is clean."))
		return rows
	}
	for _, group := range groupGitChanges(status.ChangedFiles) {
		rows = append(rows, gitDirectoryRow(group.Directory))
		for _, change := range group.Changes {
			rows = append(rows, gitChangeRow(change, v.openGitDiff))
		}
	}
	return rows
}

func groupGitChanges(changes []gitSvc.FileChange) []gitChangeGroup {
	grouped := map[string][]gitSvc.FileChange{}
	for _, change := range changes {
		directory := path.Dir(strings.TrimSpace(change.Path))
		if directory == "." || directory == "/" {
			directory = "Workspace root"
		}
		grouped[directory] = append(grouped[directory], change)
	}
	directories := make([]string, 0, len(grouped))
	for directory := range grouped {
		directories = append(directories, directory)
	}
	sort.Slice(directories, func(left int, right int) bool {
		if directories[left] == "Workspace root" {
			return true
		}
		if directories[right] == "Workspace root" {
			return false
		}
		return strings.ToLower(directories[left]) < strings.ToLower(directories[right])
	})
	groups := make([]gitChangeGroup, 0, len(directories))
	for _, directory := range directories {
		changes := grouped[directory]
		sort.Slice(changes, func(left int, right int) bool {
			return strings.ToLower(changes[left].Path) < strings.ToLower(changes[right].Path)
		})
		groups = append(groups, gitChangeGroup{Directory: directory, Changes: changes})
	}
	return groups
}

func gitDirectoryRow(directory string) fyne.CanvasObject {
	label := widget.NewLabel(directory)
	label.TextStyle = fyne.TextStyle{Bold: true}
	return container.NewBorder(nil, nil, widget.NewIcon(theme.FolderIcon()), nil, label)
}

func gitChangeRow(change gitSvc.FileChange, onOpen func(string)) fyne.CanvasObject {
	label := change.Path
	if change.OldPath != "" {
		label = change.OldPath + " -> " + change.Path
	}
	open := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		onOpen(change.Path)
	})
	open.Importance = widget.LowImportance
	text := widget.NewLabel(fmt.Sprintf("%s - %s", change.Summary, label))
	text.Truncation = fyne.TextTruncateEllipsis
	return container.NewBorder(nil, nil, open, nil, container.NewPadded(text))
}

func gitHunkTargets(diff gitSvc.FileDiff) []gitHunkTarget {
	targets := make([]gitHunkTarget, 0, len(diff.StagedHunks)+len(diff.UnstagedHunks))
	for _, hunk := range diff.StagedHunks {
		targets = append(targets, gitHunkTargetFromDiff(hunk))
	}
	for _, hunk := range diff.UnstagedHunks {
		targets = append(targets, gitHunkTargetFromDiff(hunk))
	}
	return targets
}

func gitHunkTargetFromDiff(hunk gitSvc.DiffHunk) gitHunkTarget {
	kind := "Unstaged"
	if hunk.Kind == gitSvc.DiffKindStaged {
		kind = "Staged"
	}
	return gitHunkTarget{
		Kind:   hunk.Kind,
		Index:  hunk.Index,
		Header: hunk.Header,
		Label:  fmt.Sprintf("%s hunk %d (+%d/-%d)", kind, hunk.Index+1, hunk.AddedLines, hunk.DeletedLines),
	}
}

func formatGitHistory(result gitSvc.HistoryResult) string {
	var builder strings.Builder
	builder.WriteString("# Git History\n\n")
	if result.Path != "" {
		builder.WriteString("Path: ")
		builder.WriteString(result.Path)
		builder.WriteString("\n")
	}
	builder.WriteString(result.Message)
	builder.WriteString("\n\n")
	if len(result.Entries) == 0 {
		builder.WriteString("No commits found.\n")
		return builder.String()
	}
	for _, entry := range result.Entries {
		builder.WriteString("- ")
		builder.WriteString(firstNonEmptyString(entry.ShortHash, entry.Hash))
		builder.WriteString(" ")
		builder.WriteString(entry.Subject)
		builder.WriteString("\n  ")
		builder.WriteString(entry.Author)
		if entry.Email != "" {
			builder.WriteString(" <")
			builder.WriteString(entry.Email)
			builder.WriteString(">")
		}
		if entry.Date != "" {
			builder.WriteString(" | ")
			builder.WriteString(entry.Date)
		}
		builder.WriteString("\n")
	}
	if result.Truncated {
		builder.WriteString("\nHistory truncated by the native preview limit.\n")
	}
	return builder.String()
}

func formatGitBlame(result gitSvc.BlameResult) string {
	var builder strings.Builder
	builder.WriteString("# Git Blame\n\n")
	if result.Path != "" {
		builder.WriteString("Path: ")
		builder.WriteString(result.Path)
		builder.WriteString("\n")
	}
	builder.WriteString(result.Message)
	builder.WriteString("\n\n")
	if len(result.Lines) == 0 {
		builder.WriteString("No blame lines found.\n")
		return builder.String()
	}
	for _, line := range result.Lines {
		builder.WriteString(fmt.Sprintf("%4d  %-12s  %-18s  %s\n", line.Line, firstNonEmptyString(line.ShortHash, line.Hash), line.Author, line.Content))
		if line.Summary != "" || line.Date != "" {
			builder.WriteString("      ")
			builder.WriteString(strings.TrimSpace(line.Summary + " " + line.Date))
			builder.WriteString("\n")
		}
	}
	if result.Truncated {
		builder.WriteString("\nBlame truncated by the native preview limit.\n")
	}
	return builder.String()
}
