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
	diffHeader := container.NewBorder(nil, nil, v.gitDiffStatus, diffMode)
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
	v.gitStatus.SetText(gitStatusLabel(status))
	v.gitResults.Objects = v.gitRows(status)
	v.gitResults.Refresh()
	v.addActivity(status.Message)
}

func (v *View) openGitDiff(path string) {
	workspace := v.state.Workspace()
	diff, err := v.gitService.FileDiff(workspace.Root, path)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.gitLastDiff = diff
	v.gitDiffStatus.SetText(diff.Message)
	v.gitDiffText.SetText(formatGitDiff(diff, v.gitDiffMode))
	v.addActivity(diff.Message)
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
