package shell

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

func (v *View) newArtifactsPanel() fyne.CanvasObject {
	search := widget.NewEntry()
	search.SetPlaceHolder("Search artifacts by title, path, kind, source, job, or task")
	documentReport := widget.NewButtonWithIcon("Document report", theme.DocumentCreateIcon(), v.generateDocumentSetArtifact)
	exportComparison := widget.NewButtonWithIcon("Export compare", theme.DocumentSaveIcon(), v.exportArtifactComparison)
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		v.refreshArtifactsWithQuery(search.Text)
	})
	search.OnSubmitted = func(string) {
		v.refreshArtifactsWithQuery(search.Text)
	}
	header := container.NewBorder(nil, nil, v.artifactStatus, container.NewHBox(documentReport, exportComparison, refresh), search)
	listScroll := container.NewScroll(v.artifactResults)
	listScroll.SetMinSize(fyne.NewSize(260, 110))
	sourceScroll := container.NewVScroll(v.artifactSources)
	sourceScroll.SetMinSize(fyne.NewSize(320, 80))
	previewHeader := container.NewVBox(widget.NewLabel("Artifact preview and lineage"), v.artifactSourceStatus, sourceScroll)
	preview := container.NewBorder(previewHeader, nil, nil, nil, v.artifactPreview)
	split := container.NewVSplit(listScroll, preview)
	split.Offset = 0.42
	return container.NewBorder(header, nil, nil, nil, split)
}

func (v *View) generateDocumentSetArtifact() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before generating a document report.")
		return
	}
	root := selectedPathOrEmpty(v)
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	pack, err := v.workspaceService.BuildContextPack(workspace.Root, []string{root}, workspaceSvc.ContextPackOptions{
		ContextCollectOptions: workspaceSvc.ContextCollectOptions{
			MaxFiles:   24,
			MaxEntries: 1200,
			MaxDepth:   8,
		},
		MaxBytes: 128 * 1024,
	})
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	artifact, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       documentSetArtifactTitle(root),
		Roots:       []string{root},
		SourcePaths: pack.SourcePaths,
		Content:     pack.Content,
		Truncated:   pack.Truncated,
		GeneratedBy: "Nexus native Workbench",
	})
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifactPreview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifactStatus.SetText("Created " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.refreshArtifactsWithQuery("kind:document-report")
}

func (v *View) refreshArtifacts() {
	v.refreshArtifactsWithQuery("")
}

func (v *View) refreshArtifactsWithQuery(query string) {
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
	artifacts, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: query})
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	status := fmt.Sprintf("%d artifact(s)", len(artifacts))
	if strings.TrimSpace(query) != "" {
		status += " matching " + strings.TrimSpace(query)
	}
	v.artifactStatus.SetText(status)
	v.artifactResults.Objects = artifactRows(artifacts, v.previewArtifact, v.pinArtifactForAssistantContext, v.compareArtifact, v.archiveArtifact, v.deleteArtifact)
	v.artifactResults.Refresh()
	v.addActivity(fmt.Sprintf("Loaded %d artifact(s).", len(artifacts)))
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
	lineage, err := store.Lineage(artifact.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifactPreview.SetText(artifactLineageText(lineage) + "\n\n---\n\n" + text)
	v.refreshArtifactSources(artifactSourcePaths(artifact, lineage))
	v.artifactStatus.SetText("Previewing " + artifact.RelPath)
	v.addActivity("Previewed artifact " + artifact.RelPath + ".")
}

func (v *View) pinArtifactForAssistantContext(artifact artifactsSvc.Artifact) {
	if artifact.RelPath == "" {
		v.addActivity("Artifact has no workspace-relative path to pin.")
		return
	}
	v.pinAssistantContextPath(artifact.RelPath)
	v.artifactStatus.SetText("Pinned artifact context: " + artifact.RelPath)
}

func (v *View) compareArtifact(artifact artifactsSvc.Artifact) {
	if v.artifactCompareLeft.RelPath == "" {
		v.artifactCompareLeft = artifactsCompareSelection{
			RelPath: artifact.RelPath,
			Kind:    artifact.Kind,
			Title:   artifactTitle(artifact),
		}
		v.artifactLastComparison = artifactsSvc.ArtifactComparison{}
		v.artifactStatus.SetText("Compare base selected: " + artifact.RelPath)
		v.artifactPreview.SetText("Select another " + artifact.Kind + " artifact to compare with:\n\n" + artifact.RelPath)
		v.refreshArtifactSources(nil)
		v.addActivity("Selected artifact compare base " + artifact.RelPath + ".")
		return
	}
	left := v.artifactCompareLeft
	v.artifactCompareLeft = artifactsCompareSelection{}
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	comparison, err := store.CompareArtifacts(left.RelPath, artifact.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifactPreview.SetText(formatArtifactComparison(comparison))
	v.refreshArtifactSources(nil)
	v.artifactLastComparison = comparison
	v.artifactStatus.SetText(comparison.Message)
	v.addActivity(comparison.Message)
}

func (v *View) exportArtifactComparison() {
	if !artifactComparisonReady(v.artifactLastComparison) {
		v.addActivity("Compare two artifacts before exporting a comparison report.")
		return
	}
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before exporting artifact comparison reports.")
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	artifact, err := store.WriteArtifactComparisonReport(v.artifactLastComparison)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifactStatus.SetText("Exported " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.refreshArtifactsWithQuery("kind:artifact-comparison")
}

func artifactComparisonReady(comparison artifactsSvc.ArtifactComparison) bool {
	return strings.TrimSpace(comparison.LeftPath) != "" &&
		strings.TrimSpace(comparison.RightPath) != "" &&
		strings.TrimSpace(comparison.Diff) != ""
}

func (v *View) archiveArtifact(artifact artifactsSvc.Artifact) {
	dialog.ShowConfirm("Archive artifact", "Archive "+artifact.RelPath+"?", func(confirm bool) {
		if !confirm {
			return
		}
		store, err := artifactsSvc.NewStore(v.state.Workspace().Root)
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		archived, err := store.ArchiveArtifact(artifact.RelPath)
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		v.artifactPreview.SetText("")
		v.refreshArtifactSources(nil)
		v.addActivity("Archived artifact to " + archived.RelPath + ".")
		v.refreshArtifacts()
	}, v.window)
}

func (v *View) deleteArtifact(artifact artifactsSvc.Artifact) {
	dialog.ShowConfirm("Delete artifact", "Permanently delete "+artifact.RelPath+"?", func(confirm bool) {
		if !confirm {
			return
		}
		store, err := artifactsSvc.NewStore(v.state.Workspace().Root)
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		if err := store.DeleteArtifact(artifact.RelPath); err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		v.artifactPreview.SetText("")
		v.refreshArtifactSources(nil)
		v.addActivity("Deleted artifact " + artifact.RelPath + ".")
		v.refreshArtifacts()
	}, v.window)
}

func (v *View) refreshArtifactSources(sources []string) {
	if v.artifactSources == nil || v.artifactSourceStatus == nil {
		return
	}
	v.artifactSources.Objects = nil
	if len(sources) == 0 {
		v.artifactSourceStatus.SetText("Sources: none available for this artifact.")
		v.artifactSources.Add(widget.NewLabel("No cited source files."))
		v.artifactSources.Refresh()
		return
	}
	v.artifactSourceStatus.SetText(fmt.Sprintf("Sources: %d cited file(s).", len(sources)))
	for _, source := range sources {
		source := source
		label := widget.NewLabel(source)
		label.Truncation = fyne.TextTruncateEllipsis
		open := widget.NewButtonWithIcon("", theme.FileIcon(), func() {
			v.openArtifactSource(source)
		})
		open.Importance = widget.LowImportance
		pin := widget.NewButtonWithIcon("", theme.MailAttachmentIcon(), func() {
			v.pinAssistantContextPath(source)
		})
		pin.Importance = widget.LowImportance
		v.artifactSources.Add(container.NewBorder(nil, nil, container.NewHBox(open, pin), nil, label))
	}
	v.artifactSources.Refresh()
}

func (v *View) openArtifactSource(relPath string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before opening artifact sources.")
		return
	}
	preview, err := v.workspaceService.PreviewFile(workspace.Root, relPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.state.SetSelectedPath(relPath)
	v.openPreviewTab(preview)
	v.refreshAssistantContextPins()
	v.addActivity("Opened artifact source " + relPath + ".")
}

func artifactRows(
	artifacts []artifactsSvc.Artifact,
	onPreview func(artifactsSvc.Artifact),
	onContext func(artifactsSvc.Artifact),
	onCompare func(artifactsSvc.Artifact),
	onArchive func(artifactsSvc.Artifact),
	onDelete func(artifactsSvc.Artifact),
) []fyne.CanvasObject {
	if len(artifacts) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No artifacts yet. Run a task or generate an output to create one.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifact := artifact
		preview := widget.NewButtonWithIcon("", theme.VisibilityIcon(), func() {
			onPreview(artifact)
		})
		preview.Importance = widget.LowImportance
		context := widget.NewButtonWithIcon("", theme.MailAttachmentIcon(), func() {
			onContext(artifact)
		})
		context.Importance = widget.LowImportance
		compare := widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			onCompare(artifact)
		})
		compare.Importance = widget.LowImportance
		archive := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
			onArchive(artifact)
		})
		archive.Importance = widget.LowImportance
		deleteButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			onDelete(artifact)
		})
		deleteButton.Importance = widget.LowImportance
		title := widget.NewLabel(artifactTitle(artifact))
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Truncation = fyne.TextTruncateEllipsis
		meta := widget.NewLabel(artifactMeta(artifact))
		meta.Truncation = fyne.TextTruncateEllipsis
		actions := container.NewHBox(preview, context, compare, archive, deleteButton)
		rows = append(rows, container.NewBorder(nil, nil, actions, nil, container.NewVBox(title, meta)))
	}
	return rows
}

func artifactTitle(artifact artifactsSvc.Artifact) string {
	if artifact.Title != "" {
		return artifact.Title
	}
	return filepath.Base(artifact.RelPath)
}

func documentSetArtifactTitle(root string) string {
	root = strings.TrimSpace(root)
	if root == "" || root == "." {
		return "Project Document Set Report"
	}
	return "Document Set Report - " + root
}

func artifactMeta(artifact artifactsSvc.Artifact) string {
	timestamp := "unknown time"
	if !artifact.GeneratedAt.IsZero() {
		timestamp = artifact.GeneratedAt.Format("2006-01-02 15:04:05")
	} else if !artifact.CreatedAt.IsZero() {
		timestamp = artifact.CreatedAt.Format("2006-01-02 15:04:05")
	}
	details := fmt.Sprintf("%s - %s - %d bytes", artifact.Kind, timestamp, artifact.Size)
	if artifact.JobID != "" {
		details += " - job " + artifact.JobID
	}
	if artifact.Archived {
		details += " - archived"
	}
	return details
}

func artifactLineageText(lineage artifactsSvc.Lineage) string {
	if len(lineage.Nodes) == 0 {
		return "Lineage: no metadata available."
	}
	var builder strings.Builder
	builder.WriteString("Lineage\n")
	for _, node := range lineage.Nodes {
		builder.WriteString("- ")
		builder.WriteString(node.Kind)
		builder.WriteString(": ")
		builder.WriteString(node.Label)
		builder.WriteString("\n")
	}
	if len(lineage.Edges) > 0 {
		builder.WriteString("\nRelationships\n")
		for _, edge := range lineage.Edges {
			builder.WriteString("- ")
			builder.WriteString(edge.From)
			builder.WriteString(" --")
			builder.WriteString(edge.Label)
			builder.WriteString("--> ")
			builder.WriteString(edge.To)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func artifactSourcePaths(artifact artifactsSvc.Artifact, lineage artifactsSvc.Lineage) []string {
	seen := map[string]bool{}
	sources := make([]string, 0, len(artifact.SourcePaths))
	for _, source := range artifact.SourcePaths {
		source = strings.TrimSpace(source)
		if source == "" || seen[source] {
			continue
		}
		seen[source] = true
		sources = append(sources, source)
	}
	if len(sources) > 0 {
		return sources
	}
	for _, node := range lineage.Nodes {
		if node.Kind != "source" {
			continue
		}
		source := strings.TrimSpace(node.Label)
		if source == "" || seen[source] {
			continue
		}
		seen[source] = true
		sources = append(sources, source)
	}
	return sources
}

func formatArtifactComparison(comparison artifactsSvc.ArtifactComparison) string {
	var builder strings.Builder
	builder.WriteString("Artifact Comparison\n")
	writeArtifactComparisonKV(&builder, "Kind", comparison.Kind)
	writeArtifactComparisonKV(&builder, "Left", comparison.LeftPath)
	writeArtifactComparisonKV(&builder, "Right", comparison.RightPath)
	writeArtifactComparisonKV(&builder, "Same", fmt.Sprintf("%t", comparison.Same))
	builder.WriteString("\n")
	builder.WriteString(comparison.Message)
	builder.WriteString("\n\n---\n\n")
	builder.WriteString(comparison.Diff)
	return builder.String()
}

func writeArtifactComparisonKV(builder *strings.Builder, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "-"
	}
	builder.WriteString("- ")
	builder.WriteString(key)
	builder.WriteString(": ")
	builder.WriteString(value)
	builder.WriteString("\n")
}
