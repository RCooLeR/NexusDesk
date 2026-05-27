package shell

import (
	"fmt"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	artifactsSvc "nexusdesk/internal/services/artifacts"
)

func (v *View) newArtifactsPanel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refreshArtifacts)
	header := container.NewBorder(nil, nil, v.artifactStatus, refresh)
	listScroll := container.NewScroll(v.artifactResults)
	listScroll.SetMinSize(fyne.NewSize(260, 110))
	preview := container.NewBorder(widget.NewLabel("Task report preview"), nil, nil, nil, v.artifactPreview)
	split := container.NewVSplit(listScroll, preview)
	split.Offset = 0.42
	return container.NewBorder(header, nil, nil, nil, split)
}

func (v *View) refreshArtifacts() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.artifactStatus.SetText("Open a workspace before reading artifacts.")
		v.addActivity("Open a workspace before reading artifacts.")
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	reports, err := store.ListTaskRunReports()
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifactStatus.SetText(fmt.Sprintf("%d task report artifact(s)", len(reports)))
	v.artifactResults.Objects = artifactRows(reports, v.previewArtifact)
	v.artifactResults.Refresh()
	v.addActivity(fmt.Sprintf("Loaded %d task report artifact(s).", len(reports)))
}

func (v *View) previewArtifact(artifact artifactsSvc.Artifact) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	text, err := store.ReadArtifactText(artifact.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifactPreview.SetText(text)
	v.artifactStatus.SetText("Previewing " + artifact.RelPath)
	v.addActivity("Previewed artifact " + artifact.RelPath + ".")
}

func artifactRows(artifacts []artifactsSvc.Artifact, onPreview func(artifactsSvc.Artifact)) []fyne.CanvasObject {
	if len(artifacts) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No task report artifacts yet. Run a task to generate one.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifact := artifact
		preview := widget.NewButtonWithIcon("", theme.VisibilityIcon(), func() {
			onPreview(artifact)
		})
		preview.Importance = widget.LowImportance
		title := widget.NewLabel(artifactTitle(artifact))
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Truncation = fyne.TextTruncateEllipsis
		meta := widget.NewLabel(artifactMeta(artifact))
		meta.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, preview, nil, container.NewVBox(title, meta)))
	}
	return rows
}

func artifactTitle(artifact artifactsSvc.Artifact) string {
	if artifact.Title != "" {
		return artifact.Title
	}
	return filepath.Base(artifact.RelPath)
}

func artifactMeta(artifact artifactsSvc.Artifact) string {
	timestamp := "unknown time"
	if !artifact.CreatedAt.IsZero() {
		timestamp = artifact.CreatedAt.Format("2006-01-02 15:04:05")
	}
	return fmt.Sprintf("%s - %s - %d bytes", artifact.Kind, timestamp, artifact.Size)
}
