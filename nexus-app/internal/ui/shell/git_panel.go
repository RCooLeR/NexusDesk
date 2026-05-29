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

type gitController struct {
	view       *View
	results    *fyne.Container
	status     *widget.Label
	diffText   *widget.Entry
	diffStatus *widget.Label
	diffMode   gitDiffMode
	lastDiff   gitSvc.FileDiff
	hunkStatus *widget.Label
	activeHunk int
}

func newGitController(view *View) *gitController {
	diffText := widget.NewMultiLineEntry()
	diffText.TextStyle = fyne.TextStyle{Monospace: true}
	diffText.Wrapping = fyne.TextWrapOff
	diffText.Disable()
	return &gitController{
		view:       view,
		results:    container.NewVBox(widget.NewLabel("Press Refresh git to inspect repository status.")),
		status:     widget.NewLabel("Git status has not been loaded."),
		diffText:   diffText,
		diffStatus: widget.NewLabel("Select a changed file to load a read-only diff."),
		diffMode:   gitDiffModeUnified,
		hunkStatus: widget.NewLabel("No hunk selected."),
	}
}

func (v *View) newGitPanel() fyne.CanvasObject {
	return v.git.Panel()
}

func (v *View) refreshGitStatus() {
	v.git.RefreshStatus()
}

func (v *View) applyGitStatus(status gitSvc.Status) {
	v.git.ApplyStatus(status)
}

func (v *View) openGitDiff(path string) {
	v.git.OpenDiff(path)
}

func (v *View) openSelectedGitHistory() {
	v.git.OpenSelectedHistory()
}

func (v *View) openSelectedGitBlame() {
	v.git.OpenSelectedBlame()
}

func (v *View) currentGitTargetPath() string {
	return v.git.CurrentTargetPath()
}

func (v *View) moveGitHunk(delta int) {
	v.git.MoveHunk(delta)
}

func (v *View) updateGitHunkStatus() {
	v.git.UpdateHunkStatus()
}

func (v *View) confirmGitFileAction(action gitSvc.FileAction) {
	v.git.ConfirmFileAction(action)
}

func (v *View) applyGitFileAction(action gitSvc.FileAction) {
	v.git.ApplyFileAction(action)
}

func (v *View) confirmGitHunkAction(action gitSvc.HunkAction) {
	v.git.ConfirmHunkAction(action)
}

func (v *View) applyGitHunkAction(target gitHunkTarget, action gitSvc.HunkAction) {
	v.git.ApplyHunkAction(target, action)
}

func (c *gitController) Panel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh git", theme.ViewRefreshIcon(), c.RefreshStatus)
	header := container.NewBorder(nil, nil, c.status, refresh)
	scroll := container.NewScroll(c.results)
	scroll.SetMinSize(fyne.NewSize(240, 110))
	diffMode := widget.NewSelect(gitDiffModeLabels(), func(label string) {
		c.diffMode = gitDiffModeFromLabel(label)
		if c.lastDiff.Path != "" {
			c.diffText.SetText(formatGitDiff(c.lastDiff, c.diffMode))
		}
	})
	diffMode.SetSelected(gitDiffModeUnified.Label())
	stage := widget.NewButtonWithIcon("Stage", theme.ContentAddIcon(), func() {
		c.ConfirmFileAction(gitSvc.FileActionStage)
	})
	unstage := widget.NewButtonWithIcon("Unstage", theme.ContentRemoveIcon(), func() {
		c.ConfirmFileAction(gitSvc.FileActionUnstage)
	})
	stageHunk := widget.NewButtonWithIcon("Stage hunk", theme.ContentAddIcon(), func() {
		c.ConfirmHunkAction(gitSvc.HunkActionStage)
	})
	unstageHunk := widget.NewButtonWithIcon("Unstage hunk", theme.ContentRemoveIcon(), func() {
		c.ConfirmHunkAction(gitSvc.HunkActionUnstage)
	})
	summarize := widget.NewButtonWithIcon("AI summary", theme.InfoIcon(), c.view.summarizeSelectedGitDiff)
	draftCommit := widget.NewButtonWithIcon("Draft commit", theme.MailComposeIcon(), c.view.draftSelectedGitCommitMessage)
	history := widget.NewButtonWithIcon("History", theme.HistoryIcon(), c.OpenSelectedHistory)
	blame := widget.NewButtonWithIcon("Blame", theme.VisibilityIcon(), c.OpenSelectedBlame)
	prevHunk := widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		c.MoveHunk(-1)
	})
	prevHunk.Importance = widget.LowImportance
	nextHunk := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
		c.MoveHunk(1)
	})
	nextHunk.Importance = widget.LowImportance
	hunkNav := container.NewHBox(prevHunk, c.hunkStatus, nextHunk)
	actions := container.NewHBox(stage, unstage, stageHunk, unstageHunk, history, blame, summarize, draftCommit, diffMode)
	diffHeader := container.NewBorder(nil, nil, hunkNav, actions, c.diffStatus)
	diff := container.NewBorder(diffHeader, nil, nil, nil, c.diffText)
	split := container.NewVSplit(scroll, diff)
	split.Offset = 0.42
	return container.NewBorder(header, nil, nil, nil, split)
}

func (c *gitController) RefreshStatus() {
	workspace := c.view.state.Workspace()
	status, err := c.view.gitService.Status(workspace.Root)
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	c.ApplyStatus(status)
	c.view.addActivity(status.Message)
}

func (c *gitController) ApplyStatus(status gitSvc.Status) {
	c.view.gitStatusSnapshot = status
	c.status.SetText(gitStatusLabel(status))
	c.results.Objects = c.rows(status)
	c.results.Refresh()
	c.view.gitFileBadges = gitWorkspaceBadges(status)
	if c.view.state.Workspace().Root != "" {
		c.view.refreshNavigator()
	}
	c.view.refreshStatusBar()
}

func (c *gitController) OpenDiff(path string) {
	workspace := c.view.state.Workspace()
	diff, err := c.view.gitService.FileDiff(workspace.Root, path)
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	c.lastDiff = diff
	c.activeHunk = 0
	c.diffStatus.SetText(diff.Message)
	c.diffText.SetText(formatGitDiff(diff, c.diffMode))
	c.UpdateHunkStatus()
	c.view.addActivity(diff.Message)
}

func (c *gitController) OpenSelectedHistory() {
	target := c.CurrentTargetPath()
	workspace := c.view.state.Workspace()
	result, err := c.view.gitService.History(workspace.Root, target, gitSvc.DefaultHistoryLimit)
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	c.diffStatus.SetText(result.Message)
	c.diffText.SetText(formatGitHistory(result))
	c.view.addActivity(result.Message)
}

func (c *gitController) OpenSelectedBlame() {
	target := c.CurrentTargetPath()
	if strings.TrimSpace(target) == "" {
		c.view.addActivity("Select a file before reading Git blame.")
		return
	}
	workspace := c.view.state.Workspace()
	result, err := c.view.gitService.Blame(workspace.Root, target, 1, gitSvc.DefaultHistoryLimit)
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	c.diffStatus.SetText(result.Message)
	c.diffText.SetText(formatGitBlame(result))
	c.view.addActivity(result.Message)
}

func (c *gitController) CurrentTargetPath() string {
	if strings.TrimSpace(c.lastDiff.Path) != "" {
		return c.lastDiff.Path
	}
	return strings.TrimSpace(c.view.state.SelectedPath())
}

func (c *gitController) MoveHunk(delta int) {
	targets := gitHunkTargets(c.lastDiff)
	if len(targets) == 0 {
		c.view.addActivity("No diff hunks are available for the selected file.")
		return
	}
	c.activeHunk = (c.activeHunk + delta + len(targets)) % len(targets)
	target := targets[c.activeHunk]
	c.UpdateHunkStatus()
	c.view.addActivity("Selected " + target.Label + ".")
}

func (c *gitController) UpdateHunkStatus() {
	targets := gitHunkTargets(c.lastDiff)
	if len(targets) == 0 {
		c.hunkStatus.SetText("No hunks")
		return
	}
	if c.activeHunk < 0 || c.activeHunk >= len(targets) {
		c.activeHunk = 0
	}
	c.hunkStatus.SetText(fmt.Sprintf("%d / %d %s", c.activeHunk+1, len(targets), targets[c.activeHunk].Label))
}

func (c *gitController) ConfirmFileAction(action gitSvc.FileAction) {
	if c.lastDiff.Path == "" {
		c.view.addActivity("Select a changed file before running a Git action.")
		return
	}
	title := "Stage file"
	message := "Stage " + c.lastDiff.Path + "?"
	if action == gitSvc.FileActionUnstage {
		title = "Unstage file"
		message = "Unstage " + c.lastDiff.Path + "?"
	}
	dialog.ShowConfirm(title, message, func(confirm bool) {
		if !confirm {
			return
		}
		c.ApplyFileAction(action)
	}, c.view.window)
}

func (c *gitController) ApplyFileAction(action gitSvc.FileAction) {
	workspace := c.view.state.Workspace()
	result, err := c.view.gitService.ApplyFileAction(workspace.Root, c.lastDiff.Path, action)
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	c.ApplyStatus(result.Status)
	c.view.addActivity(result.Message)
	c.OpenDiff(result.Path)
}

func (c *gitController) ConfirmHunkAction(action gitSvc.HunkAction) {
	targets := gitHunkTargets(c.lastDiff)
	if c.lastDiff.Path == "" || len(targets) == 0 {
		c.view.addActivity("Select a changed file and hunk before running a Git hunk action.")
		return
	}
	if c.activeHunk < 0 || c.activeHunk >= len(targets) {
		c.activeHunk = 0
	}
	target := targets[c.activeHunk]
	if !validGitHunkAction(action, target.Kind) {
		c.view.addActivity("Selected hunk cannot be used with that action.")
		return
	}
	title := "Stage hunk"
	message := "Stage " + target.Label + " in " + c.lastDiff.Path + "?"
	if action == gitSvc.HunkActionUnstage {
		title = "Unstage hunk"
		message = "Unstage " + target.Label + " in " + c.lastDiff.Path + "?"
	}
	dialog.ShowConfirm(title, message, func(confirm bool) {
		if !confirm {
			return
		}
		c.ApplyHunkAction(target, action)
	}, c.view.window)
}

func (c *gitController) ApplyHunkAction(target gitHunkTarget, action gitSvc.HunkAction) {
	workspace := c.view.state.Workspace()
	result, err := c.view.gitService.ApplyHunkAction(workspace.Root, c.lastDiff.Path, target.Kind, target.Index, action)
	if err != nil {
		dialog.ShowError(err, c.view.window)
		return
	}
	if result.Message != "" {
		c.view.addActivity(result.Message)
	}
	if result.Status.Available {
		c.ApplyStatus(result.Status)
	}
	c.OpenDiff(result.Path)
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

func (c *gitController) rows(status gitSvc.Status) []fyne.CanvasObject {
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
			rows = append(rows, gitChangeRow(change, c.OpenDiff))
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
