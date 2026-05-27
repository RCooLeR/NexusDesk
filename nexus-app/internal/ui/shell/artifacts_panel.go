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
	documentsSvc "nexusdesk/internal/services/documents"
	metadataSvc "nexusdesk/internal/services/metadata"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

func (v *View) newArtifactsPanel() fyne.CanvasObject {
	search := widget.NewEntry()
	search.SetPlaceHolder("Search artifacts by title, path, kind, source, job, or task")
	documentReport := widget.NewButtonWithIcon("Document report", theme.DocumentCreateIcon(), v.generateDocumentSetArtifact)
	documentExtract := widget.NewButtonWithIcon("Extract doc", theme.FileTextIcon(), v.generateDocumentExtractionArtifact)
	exportComparison := widget.NewButtonWithIcon("Export compare", theme.DocumentSaveIcon(), v.exportArtifactComparison)
	showArchived := widget.NewCheck("Show archived", func(include bool) {
		v.artifactIncludeArchived = include
		v.refreshArtifactsWithQuery(search.Text)
	})
	showArchived.SetChecked(v.artifactIncludeArchived)
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		v.refreshArtifactsWithQuery(search.Text)
	})
	search.OnSubmitted = func(string) {
		v.refreshArtifactsWithQuery(search.Text)
	}
	header := container.NewBorder(nil, nil, v.artifactStatus, container.NewHBox(documentReport, documentExtract, exportComparison, showArchived, refresh), search)
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
	v.persistArtifactRecord(artifact)
	v.refreshArtifactsWithQuery("kind:document-report")
}

func (v *View) generateDocumentExtractionArtifact() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before extracting a document.")
		return
	}
	source := selectedPathOrEmpty(v)
	if strings.TrimSpace(source) == "" {
		v.addActivity("Select a Markdown, TXT, HTML, XML, DOCX, or PDF file before extracting a document.")
		return
	}
	extractor := documentsSvc.New(v.workspaceService)
	document, err := extractor.Extract(workspace.Root, source)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	artifact, err := store.WriteDocumentExtractionReport(documentExtractionArtifactInput(document))
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifactPreview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifactStatus.SetText("Created " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.persistArtifactRecord(artifact)
	v.refreshArtifactsWithQuery("kind:document-extract")
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
	artifacts, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: query, IncludeArchived: v.artifactIncludeArchived})
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	status := fmt.Sprintf("%d artifact(s)", len(artifacts))
	if strings.TrimSpace(query) != "" {
		status += " matching " + strings.TrimSpace(query)
	}
	v.artifactStatus.SetText(status)
	v.artifactResults.Objects = artifactRows(artifacts, v.previewArtifact, v.pinArtifactForAssistantContext, v.compareArtifact, v.archiveArtifact, v.restoreArtifact, v.deleteArtifact)
	v.artifactResults.Refresh()
	v.persistArtifactRecords(artifacts)
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
	freshness, err := store.SourceFreshness(artifact.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifactPreview.SetText(artifactLineageText(lineage) + "\n\n" + artifactFreshnessText(freshness) + "\n\n---\n\n" + text)
	v.refreshArtifactSources(freshness.Sources)
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
	v.persistArtifactRecord(artifact)
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
		v.deleteArtifactRecord(artifact.RelPath)
		v.persistArtifactRecord(archived)
		v.addActivity("Archived artifact to " + archived.RelPath + ".")
		v.refreshArtifacts()
	}, v.window)
}

func (v *View) restoreArtifact(artifact artifactsSvc.Artifact) {
	dialog.ShowConfirm("Restore artifact", "Restore "+artifact.RelPath+"?", func(confirm bool) {
		if !confirm {
			return
		}
		store, err := artifactsSvc.NewStore(v.state.Workspace().Root)
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		restored, err := store.RestoreArtifact(artifact.RelPath)
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		v.artifactPreview.SetText("")
		v.refreshArtifactSources(nil)
		v.deleteArtifactRecord(artifact.RelPath)
		v.persistArtifactRecord(restored)
		v.addActivity("Restored artifact to " + restored.RelPath + ".")
		v.refreshArtifactsWithQuery(restored.RelPath)
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
		v.deleteArtifactRecord(artifact.RelPath)
		v.addActivity("Deleted artifact " + artifact.RelPath + ".")
		v.refreshArtifacts()
	}, v.window)
}

func (v *View) persistArtifactRecords(artifacts []artifactsSvc.Artifact) {
	for _, artifact := range artifacts {
		v.persistArtifactRecord(artifact)
	}
}

func (v *View) persistArtifactRecord(artifact artifactsSvc.Artifact) {
	if v.metadataStore == nil || artifact.RelPath == "" {
		return
	}
	if err := v.metadataStore.SaveArtifact(artifactMetadataRecord(artifact)); err != nil {
		v.addActivity("Could not persist artifact metadata: " + err.Error())
	}
}

func (v *View) deleteArtifactRecord(relPath string) {
	if v.metadataStore == nil || strings.TrimSpace(relPath) == "" {
		return
	}
	if err := v.metadataStore.DeleteArtifact(relPath); err != nil {
		v.addActivity("Could not delete artifact metadata: " + err.Error())
	}
}

func (v *View) refreshArtifactSources(sources []artifactsSvc.SourceFreshnessStatus) {
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
	v.artifactSourceStatus.SetText(artifactSourceStatusText(sources))
	for _, sourceStatus := range sources {
		sourceStatus := sourceStatus
		label := widget.NewLabel(artifactSourceLabel(sourceStatus))
		label.Truncation = fyne.TextTruncateEllipsis
		open := widget.NewButtonWithIcon("", theme.FileIcon(), func() {
			v.openArtifactSource(sourceStatus.RelPath)
		})
		open.Importance = widget.LowImportance
		if !sourceStatus.Exists {
			open.Disable()
		}
		pin := widget.NewButtonWithIcon("", theme.MailAttachmentIcon(), func() {
			v.pinAssistantContextPath(sourceStatus.RelPath)
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
	onRestore func(artifactsSvc.Artifact),
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
		restore := widget.NewButtonWithIcon("", theme.ContentUndoIcon(), func() {
			onRestore(artifact)
		})
		restore.Importance = widget.LowImportance
		if artifact.Archived {
			archive.Disable()
		} else {
			restore.Disable()
		}
		deleteButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			onDelete(artifact)
		})
		deleteButton.Importance = widget.LowImportance
		title := widget.NewLabel(artifactTitle(artifact))
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Truncation = fyne.TextTruncateEllipsis
		meta := widget.NewLabel(artifactMeta(artifact))
		meta.Truncation = fyne.TextTruncateEllipsis
		actions := container.NewHBox(preview, context, compare, archive, restore, deleteButton)
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

func documentExtractionArtifactInput(document documentsSvc.ExtractedDocument) artifactsSvc.DocumentExtractionReport {
	return artifactsSvc.DocumentExtractionReport{
		Title:     document.Title,
		RelPath:   document.RelPath,
		Format:    document.Format,
		MediaType: document.MediaType,
		Encoding:  document.Encoding,
		Content:   document.Text,
		Size:      document.Size,
		Lines:     document.Lines,
		Words:     document.Words,
		Pages:     document.Pages,
		Truncated: document.Truncated,
	}
}

func artifactMetadataRecord(artifact artifactsSvc.Artifact) metadataSvc.ArtifactRecord {
	return metadataSvc.ArtifactRecord{
		Kind:         artifact.Kind,
		Title:        artifact.Title,
		RelPath:      artifact.RelPath,
		MetadataPath: artifact.MetadataPath,
		Size:         artifact.Size,
		JobID:        artifact.JobID,
		TaskID:       artifact.TaskID,
		Source:       artifact.Source,
		SourcePaths:  append([]string{}, artifact.SourcePaths...),
		Archived:     artifact.Archived,
		CreatedAt:    artifact.CreatedAt,
		GeneratedAt:  artifact.GeneratedAt,
	}
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

func artifactFreshnessText(freshness artifactsSvc.SourceFreshness) string {
	var builder strings.Builder
	builder.WriteString("Source Freshness\n")
	builder.WriteString("- ")
	builder.WriteString(freshness.Message)
	builder.WriteString("\n")
	for _, source := range freshness.Sources {
		builder.WriteString("- ")
		builder.WriteString(artifactSourceLabel(source))
		builder.WriteString("\n")
	}
	return builder.String()
}

func artifactSourceStatusText(sources []artifactsSvc.SourceFreshnessStatus) string {
	changed := 0
	missing := 0
	unknown := 0
	for _, source := range sources {
		if source.Changed {
			changed++
		}
		if source.Unknown {
			unknown++
		} else if !source.Exists {
			missing++
		}
	}
	if changed > 0 || missing > 0 {
		return fmt.Sprintf("Sources: %d cited, %d changed, %d missing.", len(sources), changed, missing)
	}
	if unknown > 0 {
		return fmt.Sprintf("Sources: %d cited, %d unchecked.", len(sources), unknown)
	}
	return fmt.Sprintf("Sources: %d cited and current.", len(sources))
}

func artifactSourceLabel(source artifactsSvc.SourceFreshnessStatus) string {
	status := "current"
	switch {
	case source.Changed:
		status = "changed"
	case !source.Exists:
		status = "missing"
	case source.Unknown:
		status = "unchecked"
	}
	if source.Message != "" {
		return fmt.Sprintf("%s (%s: %s)", source.RelPath, status, source.Message)
	}
	return fmt.Sprintf("%s (%s)", source.RelPath, status)
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
