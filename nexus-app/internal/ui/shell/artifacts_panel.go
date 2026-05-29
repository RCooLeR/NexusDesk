package shell

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	documentsSvc "nexusdesk/internal/services/documents"
	jobsSvc "nexusdesk/internal/services/jobs"
	metadataSvc "nexusdesk/internal/services/metadata"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

const (
	artifactJobKindDocumentReport  = "artifact-document-report"
	artifactJobKindDocumentExtract = "artifact-document-extract"
	artifactJobKindDocumentBrief   = "artifact-document-brief"
	artifactJobKindDocumentExport  = "artifact-document-export"
	artifactJobKindScanReport      = "artifact-scan-report"
	artifactJobKindPresentation    = "artifact-presentation"
	artifactJobKindRegenerate      = "artifact-regenerate"
)

type artifactsController struct {
	view            *View
	results         *fyne.Container
	status          *widget.Label
	preview         *widget.Entry
	sourceStatus    *widget.Label
	sources         *fyne.Container
	includeArchived bool
	compareLeft     artifactsCompareSelection
	lastComparison  artifactsSvc.ArtifactComparison
}

func newArtifactsController(view *View) *artifactsController {
	preview := widget.NewMultiLineEntry()
	preview.TextStyle = fyne.TextStyle{Monospace: true}
	preview.Wrapping = fyne.TextWrapWord
	preview.Disable()
	return &artifactsController{
		view:         view,
		results:      container.NewVBox(widget.NewLabel("Refresh artifacts to inspect generated task reports.")),
		status:       widget.NewLabel("Artifacts have not been loaded."),
		preview:      preview,
		sourceStatus: widget.NewLabel("Artifact sources have not been loaded."),
		sources:      container.NewVBox(widget.NewLabel("Preview an artifact to inspect cited sources.")),
	}
}

func (v *View) newArtifactsPanel() fyne.CanvasObject {
	search := widget.NewEntry()
	search.SetPlaceHolder("Search artifacts by title, path, kind, source, job, or task")
	documentReport := widget.NewButtonWithIcon("Document report", theme.DocumentCreateIcon(), v.generateDocumentSetArtifact)
	documentExtract := widget.NewButtonWithIcon("Extract doc", theme.FileTextIcon(), v.generateDocumentExtractionArtifact)
	scanReport := widget.NewButtonWithIcon("Scan report", theme.SearchIcon(), v.generateWorkspaceScanReportArtifact)
	exportComparison := widget.NewButtonWithIcon("Export compare", theme.DocumentSaveIcon(), v.exportArtifactComparison)
	exportLineage := widget.NewButtonWithIcon("Export lineage", theme.DocumentSaveIcon(), v.exportArtifactLineageGraph)
	importLineage := widget.NewButtonWithIcon("Import lineage", theme.FolderOpenIcon(), v.importArtifactLineageGraph)
	showArchived := widget.NewCheck("Show archived", func(include bool) {
		v.artifacts.includeArchived = include
		v.refreshArtifactsWithQuery(search.Text)
	})
	showArchived.SetChecked(v.artifacts.includeArchived)
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		v.refreshArtifactsWithQuery(search.Text)
	})
	search.OnSubmitted = func(string) {
		v.refreshArtifactsWithQuery(search.Text)
	}
	header := container.NewBorder(nil, nil, v.artifacts.status, container.NewHBox(documentReport, documentExtract, scanReport, exportComparison, exportLineage, importLineage, showArchived, refresh), search)
	listScroll := container.NewScroll(v.artifacts.results)
	listScroll.SetMinSize(fyne.NewSize(260, 110))
	sourceScroll := container.NewVScroll(v.artifacts.sources)
	sourceScroll.SetMinSize(fyne.NewSize(320, 80))
	previewHeader := container.NewVBox(widget.NewLabel("Artifact preview and lineage"), v.artifacts.sourceStatus, sourceScroll)
	preview := container.NewBorder(previewHeader, nil, nil, nil, v.artifacts.preview)
	split := container.NewVSplit(listScroll, preview)
	split.Offset = 0.42
	return container.NewBorder(header, nil, nil, nil, split)
}

func (v *View) generateWorkspaceScanReportArtifact() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before generating a scan report.")
		return
	}
	jobLabel := workspaceScanReportJobLabel(workspace.Name)
	job, ctx := v.jobService.Start(artifactJobKindScanReport, jobLabel)
	v.jobService.AppendLog(job.ID, "Workspace: "+workspace.Root)
	v.artifacts.status.SetText("Running " + jobLabel + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	go func() {
		artifact, err := v.buildWorkspaceScanReportArtifact(ctx, workspace.Root)
		fyne.Do(func() {
			v.finishWorkspaceScanReportArtifactJob(job.ID, artifact, err)
		})
	}()
}

func (v *View) buildWorkspaceScanReportArtifact(ctx context.Context, workspaceRoot string) (artifactsSvc.Artifact, error) {
	report, err := v.workspaceService.ScanReport(ctx, workspaceRoot, workspaceSvc.ScanReportOptions{
		MaxDepth:   12,
		MaxEntries: 5000,
		MaxSamples: 12,
	})
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	artifact, err := store.WriteWorkspaceScanReport(workspaceScanArtifactInput(report))
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return artifact, nil
}

func (v *View) finishWorkspaceScanReportArtifactJob(jobID string, artifact artifactsSvc.Artifact, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Workspace scan report canceled.", nil)
			v.artifacts.status.SetText("Workspace scan report canceled.")
			v.addActivity("Workspace scan report canceled for " + jobID + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Workspace scan report failed.", err)
			v.artifacts.status.SetText("Workspace scan report failed.")
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	artifact.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+artifact.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+artifact.RelPath+".", nil)
	v.artifacts.preview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Created " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.persistArtifactRecord(artifact)
	v.refreshArtifactsWithQuery("kind:scan-report")
	v.refreshJobs()
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
	jobLabel := documentSetArtifactJobLabel(root)
	job, ctx := v.jobService.Start(artifactJobKindDocumentReport, jobLabel)
	v.jobService.AppendLog(job.ID, "Root: "+root)
	v.artifacts.status.SetText("Running " + jobLabel + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	workspaceRoot := workspace.Root
	go func() {
		artifact, err := v.buildDocumentSetArtifact(ctx, workspaceRoot, root)
		fyne.Do(func() {
			v.finishDocumentSetArtifactJob(job.ID, artifact, err)
		})
	}()
}

func (v *View) buildDocumentSetArtifact(ctx context.Context, workspaceRoot string, root string) (artifactsSvc.Artifact, error) {
	root = filepath.ToSlash(strings.TrimSpace(root))
	if root == "" {
		root = "."
	}
	return v.buildDocumentSetArtifactFromRoots(ctx, workspaceRoot, []string{root})
}

func (v *View) buildDocumentSetArtifactFromRoots(ctx context.Context, workspaceRoot string, roots []string) (artifactsSvc.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	roots = normalizeArtifactRegenerationSources(roots, true)
	if len(roots) == 0 {
		roots = []string{"."}
	}
	pack, err := v.workspaceService.BuildContextPack(workspaceRoot, roots, workspaceSvc.ContextPackOptions{
		ContextCollectOptions: workspaceSvc.ContextCollectOptions{
			MaxFiles:   24,
			MaxEntries: 1200,
			MaxDepth:   8,
		},
		MaxBytes: 128 * 1024,
	})
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	artifact, err := store.WriteDocumentSetReport(artifactsSvc.DocumentSetReport{
		Title:       documentSetArtifactTitleForRoots(roots),
		Roots:       append([]string{}, roots...),
		SourcePaths: pack.SourcePaths,
		Content:     pack.Content,
		Truncated:   pack.Truncated,
		GeneratedBy: "Nexus native Workbench",
	})
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return artifact, nil
}

func (v *View) finishDocumentSetArtifactJob(jobID string, artifact artifactsSvc.Artifact, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Document report canceled.", nil)
			v.artifacts.status.SetText("Document report canceled.")
			v.addActivity("Document report canceled for " + jobID + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Document report failed.", err)
			v.artifacts.status.SetText("Document report failed.")
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	artifact.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+artifact.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+artifact.RelPath+".", nil)
	v.artifacts.preview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Created " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.persistArtifactRecord(artifact)
	v.refreshArtifactsWithQuery("kind:document-report")
	v.refreshJobs()
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
	jobLabel := documentExtractionArtifactJobLabel(source)
	job, ctx := v.jobService.Start(artifactJobKindDocumentExtract, jobLabel)
	v.jobService.AppendLog(job.ID, "Source: "+source)
	v.artifacts.status.SetText("Running " + jobLabel + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	workspaceRoot := workspace.Root
	go func() {
		artifact, err := v.buildDocumentExtractionArtifact(ctx, workspaceRoot, source)
		fyne.Do(func() {
			v.finishDocumentExtractionArtifactJob(job.ID, artifact, err)
		})
	}()
}

func (v *View) buildDocumentExtractionArtifact(ctx context.Context, workspaceRoot string, source string) (artifactsSvc.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	extractor := documentsSvc.New(v.workspaceService)
	document, err := extractor.Extract(workspaceRoot, source)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	artifact, err := store.WriteDocumentExtractionReport(documentExtractionArtifactInput(document))
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return artifact, nil
}

func (v *View) finishDocumentExtractionArtifactJob(jobID string, artifact artifactsSvc.Artifact, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Document extraction canceled.", nil)
			v.artifacts.status.SetText("Document extraction canceled.")
			v.addActivity("Document extraction canceled for " + jobID + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Document extraction failed.", err)
			v.artifacts.status.SetText("Document extraction failed.")
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	artifact.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+artifact.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+artifact.RelPath+".", nil)
	v.artifacts.preview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Created " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.persistArtifactRecord(artifact)
	v.refreshArtifactsWithQuery("kind:document-extract")
	v.refreshJobs()
}

func (v *View) refreshArtifacts() {
	v.refreshArtifactsWithQuery("")
}

func (v *View) refreshArtifactsWithQuery(query string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.artifacts.status.SetText("Open a workspace before reading artifacts.")
		v.addActivity("Open a workspace before reading artifacts.")
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	artifacts, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: query, IncludeArchived: v.artifacts.includeArchived})
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	status := fmt.Sprintf("%d artifact(s)", len(artifacts))
	if strings.TrimSpace(query) != "" {
		status += " matching " + strings.TrimSpace(query)
	}
	v.artifacts.status.SetText(status)
	v.artifacts.results.Objects = artifactRows(artifacts, v.previewArtifact, v.pinArtifactForAssistantContext, v.compareArtifact, v.generateDocumentBriefFromArtifact, v.generatePresentationOutlineFromArtifact, v.regenerateArtifact, v.archiveArtifact, v.restoreArtifact, v.deleteArtifact)
	v.artifacts.results.Refresh()
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
	text, err := artifactPreviewText(store, artifact)
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
	v.artifacts.preview.SetText(artifactPreviewSummaryText(artifact) + "\n\n" + artifactLineageText(lineage) + "\n\n" + artifactFreshnessText(freshness) + "\n\n---\n\n" + text)
	v.refreshArtifactSources(freshness.Sources)
	v.artifacts.status.SetText("Previewing " + artifact.RelPath)
	v.addActivity("Previewed artifact " + artifact.RelPath + ".")
}

func (v *View) pinArtifactForAssistantContext(artifact artifactsSvc.Artifact) {
	if artifact.RelPath == "" {
		v.addActivity("Artifact has no workspace-relative path to pin.")
		return
	}
	v.pinAssistantContextPath(artifact.RelPath)
	v.artifacts.status.SetText("Pinned artifact context: " + artifact.RelPath)
}

func (v *View) compareArtifact(artifact artifactsSvc.Artifact) {
	if v.artifacts.compareLeft.RelPath == "" {
		v.artifacts.compareLeft = artifactsCompareSelection{
			RelPath: artifact.RelPath,
			Kind:    artifact.Kind,
			Title:   artifactTitle(artifact),
		}
		v.artifacts.lastComparison = artifactsSvc.ArtifactComparison{}
		v.artifacts.status.SetText("Compare base selected: " + artifact.RelPath)
		v.artifacts.preview.SetText("Select another " + artifact.Kind + " artifact to compare with:\n\n" + artifact.RelPath)
		v.refreshArtifactSources(nil)
		v.addActivity("Selected artifact compare base " + artifact.RelPath + ".")
		return
	}
	left := v.artifacts.compareLeft
	v.artifacts.compareLeft = artifactsCompareSelection{}
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
	v.artifacts.preview.SetText(formatArtifactComparison(comparison))
	v.refreshArtifactSources(nil)
	v.artifacts.lastComparison = comparison
	v.artifacts.status.SetText(comparison.Message)
	v.addActivity(comparison.Message)
}

func (v *View) generateDocumentBriefFromArtifact(artifact artifactsSvc.Artifact) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before generating document briefs.")
		return
	}
	if artifactCanGenerateDocumentExport(artifact) {
		v.generateDocumentExportFromBrief(artifact)
		return
	}
	if !artifactCanGenerateDocumentBrief(artifact) {
		v.artifacts.status.SetText("Document brief is not available for " + artifact.Kind + ".")
		v.addActivity("Select a report-like artifact before generating a document brief.")
		return
	}
	jobLabel := documentBriefJobLabel(artifact)
	job, ctx := v.jobService.Start(artifactJobKindDocumentBrief, jobLabel)
	v.jobService.AppendLog(job.ID, "Source artifact: "+artifact.RelPath)
	v.jobService.AppendLog(job.ID, "Source kind: "+artifact.Kind)
	v.artifacts.status.SetText("Generating document brief from " + artifact.RelPath + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		created, err := buildDocumentBriefArtifact(ctx, root, artifact)
		fyne.Do(func() {
			v.finishDocumentBriefJob(job.ID, artifact, created, err)
		})
	}()
}

func (v *View) generateDocumentExportFromBrief(artifact artifactsSvc.Artifact) {
	jobLabel := documentExportJobLabel(artifact)
	job, ctx := v.jobService.Start(artifactJobKindDocumentExport, jobLabel)
	v.jobService.AppendLog(job.ID, "Source brief: "+artifact.RelPath)
	v.artifacts.status.SetText("Exporting DOCX from " + artifact.RelPath + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := v.state.Workspace().Root
	go func() {
		created, err := buildDocumentExportArtifact(ctx, root, artifact)
		fyne.Do(func() {
			v.finishDocumentExportJob(job.ID, artifact, created, err)
		})
	}()
}

func buildDocumentBriefArtifact(ctx context.Context, workspaceRoot string, source artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	text, err := store.ReadArtifactText(source.RelPath)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source.RelPath)
	if err == nil {
		source.Title = firstNonEmpty(metadata.Title, source.Title)
		source.Kind = firstNonEmpty(metadata.Kind, source.Kind)
		source.SourcePaths = append([]string{}, metadata.SourcePaths...)
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	report := artifactsSvc.BuildDocumentBriefReport(
		"",
		source.RelPath,
		artifactTitle(source),
		source.Kind,
		text,
		source.SourcePaths,
	)
	created, err := store.WriteDocumentBriefReport(report)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return created, nil
}

func buildDocumentExportArtifact(ctx context.Context, workspaceRoot string, source artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	text, err := store.ReadArtifactText(source.RelPath)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source.RelPath)
	if err == nil {
		source.Title = firstNonEmpty(metadata.Title, source.Title)
		source.Kind = firstNonEmpty(metadata.Kind, source.Kind)
		source.SourcePaths = append([]string{}, metadata.SourcePaths...)
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	report := artifactsSvc.BuildDocumentExportReport(
		"",
		source.RelPath,
		artifactTitle(source),
		source.Kind,
		text,
		source.SourcePaths,
	)
	created, err := store.WriteDocumentExportReport(report)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return created, nil
}

func (v *View) generatePresentationOutlineFromArtifact(artifact artifactsSvc.Artifact) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before generating presentation artifacts.")
		return
	}
	if artifactCanGeneratePresentationDeck(artifact) {
		v.generatePresentationDeckFromPackage(artifact)
		return
	}
	if artifactCanGeneratePresentationPackage(artifact) {
		v.generatePresentationPackageFromOutline(artifact)
		return
	}
	if !artifactCanGeneratePresentationOutline(artifact) {
		v.artifacts.status.SetText("Presentation outline is not available for " + artifact.Kind + ".")
		v.addActivity("Select a report-like artifact before generating a presentation outline.")
		return
	}
	jobLabel := presentationOutlineJobLabel(artifact)
	job, ctx := v.jobService.Start(artifactJobKindPresentation, jobLabel)
	v.jobService.AppendLog(job.ID, "Source artifact: "+artifact.RelPath)
	v.jobService.AppendLog(job.ID, "Source kind: "+artifact.Kind)
	v.artifacts.status.SetText("Generating presentation outline from " + artifact.RelPath + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		created, err := buildPresentationOutlineArtifact(ctx, root, artifact)
		fyne.Do(func() {
			v.finishPresentationOutlineJob(job.ID, artifact, created, err)
		})
	}()
}

func (v *View) generatePresentationPackageFromOutline(artifact artifactsSvc.Artifact) {
	jobLabel := presentationPackageJobLabel(artifact)
	job, ctx := v.jobService.Start(jobsSvc.KindPackagedExport, jobLabel)
	v.jobService.AppendLog(job.ID, "Source outline: "+artifact.RelPath)
	v.artifacts.status.SetText("Packaging presentation from " + artifact.RelPath + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := v.state.Workspace().Root
	go func() {
		created, err := buildPresentationPackageArtifact(ctx, root, artifact)
		fyne.Do(func() {
			v.finishPresentationPackageJob(job.ID, artifact, created, err)
		})
	}()
}

func (v *View) generatePresentationDeckFromPackage(artifact artifactsSvc.Artifact) {
	jobLabel := presentationDeckJobLabel(artifact)
	job, ctx := v.jobService.Start(jobsSvc.KindPackagedExport, jobLabel)
	v.jobService.AppendLog(job.ID, "Source package: "+artifact.RelPath)
	v.artifacts.status.SetText("Exporting PPTX from " + artifact.RelPath + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := v.state.Workspace().Root
	go func() {
		created, err := buildPresentationDeckFromPackageArtifact(ctx, root, artifact)
		fyne.Do(func() {
			v.finishPresentationDeckJob(job.ID, artifact, created, err)
		})
	}()
}

func buildPresentationOutlineArtifact(ctx context.Context, workspaceRoot string, source artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	text, err := store.ReadArtifactText(source.RelPath)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source.RelPath)
	if err == nil {
		source.Title = firstNonEmpty(metadata.Title, source.Title)
		source.Kind = firstNonEmpty(metadata.Kind, source.Kind)
		source.SourcePaths = append([]string{}, metadata.SourcePaths...)
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	report := artifactsSvc.BuildPresentationOutlineReport(
		"",
		source.RelPath,
		artifactTitle(source),
		source.Kind,
		text,
		source.SourcePaths,
	)
	created, err := store.WritePresentationOutlineReport(report)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return created, nil
}

func buildPresentationPackageArtifact(ctx context.Context, workspaceRoot string, source artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	text, err := store.ReadArtifactText(source.RelPath)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source.RelPath)
	if err == nil {
		source.Title = firstNonEmpty(metadata.Title, source.Title)
		source.Kind = firstNonEmpty(metadata.Kind, source.Kind)
		source.SourcePaths = append([]string{}, metadata.SourcePaths...)
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	report := artifactsSvc.BuildPresentationPackageReport(
		"",
		source.RelPath,
		artifactTitle(source),
		source.Kind,
		text,
		source.SourcePaths,
	)
	created, err := store.WritePresentationPackageReport(report)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return created, nil
}

func buildPresentationDeckArtifact(ctx context.Context, workspaceRoot string, source artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	text, err := store.ReadArtifactText(source.RelPath)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source.RelPath)
	if err == nil {
		source.Title = firstNonEmpty(metadata.Title, source.Title)
		source.Kind = firstNonEmpty(metadata.Kind, source.Kind)
		source.SourcePaths = append([]string{}, metadata.SourcePaths...)
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	report := artifactsSvc.BuildPresentationDeckReport(
		"",
		source.RelPath,
		artifactTitle(source),
		source.Kind,
		text,
		source.SourcePaths,
	)
	created, err := store.WritePresentationDeckReport(report)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return created, nil
}

func buildPresentationDeckFromPackageArtifact(ctx context.Context, workspaceRoot string, artifact artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(artifact.RelPath)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	outline := strings.TrimSpace(metadata.Source)
	if outline == "" {
		return artifactsSvc.Artifact{}, fmt.Errorf("presentation package artifact %s has no source outline metadata", artifact.RelPath)
	}
	outlineMetadata, err := store.ReadArtifactMetadata(outline)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	sourceArtifact := artifactsSvc.Artifact{
		Kind:        outlineMetadata.Kind,
		Title:       outlineMetadata.Title,
		RelPath:     outline,
		Source:      outlineMetadata.Source,
		SourcePaths: append([]string{}, outlineMetadata.SourcePaths...),
	}
	return buildPresentationDeckArtifact(ctx, workspaceRoot, sourceArtifact)
}

func buildPresentationOutlineRefreshArtifact(ctx context.Context, workspaceRoot string, artifact artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	source, ok := artifactRegenerationSource(artifact)
	if !ok {
		return artifactsSvc.Artifact{}, fmt.Errorf("presentation outline artifact %s has no source artifact metadata", artifact.RelPath)
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	sourceArtifact := artifactsSvc.Artifact{
		Kind:        metadata.Kind,
		Title:       metadata.Title,
		RelPath:     source,
		Source:      metadata.Source,
		SourcePaths: append([]string{}, metadata.SourcePaths...),
	}
	return buildPresentationOutlineArtifact(ctx, workspaceRoot, sourceArtifact)
}

func buildPresentationPackageRefreshArtifact(ctx context.Context, workspaceRoot string, artifact artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	source, ok := artifactRegenerationSource(artifact)
	if !ok {
		return artifactsSvc.Artifact{}, fmt.Errorf("presentation package artifact %s has no source outline metadata", artifact.RelPath)
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	sourceArtifact := artifactsSvc.Artifact{
		Kind:        metadata.Kind,
		Title:       metadata.Title,
		RelPath:     source,
		Source:      metadata.Source,
		SourcePaths: append([]string{}, metadata.SourcePaths...),
	}
	return buildPresentationPackageArtifact(ctx, workspaceRoot, sourceArtifact)
}

func buildPresentationDeckRefreshArtifact(ctx context.Context, workspaceRoot string, artifact artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	source, ok := artifactRegenerationSource(artifact)
	if !ok {
		return artifactsSvc.Artifact{}, fmt.Errorf("presentation deck artifact %s has no source outline metadata", artifact.RelPath)
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	sourceArtifact := artifactsSvc.Artifact{
		Kind:        metadata.Kind,
		Title:       metadata.Title,
		RelPath:     source,
		Source:      metadata.Source,
		SourcePaths: append([]string{}, metadata.SourcePaths...),
	}
	return buildPresentationDeckArtifact(ctx, workspaceRoot, sourceArtifact)
}

func buildDocumentBriefRefreshArtifact(ctx context.Context, workspaceRoot string, artifact artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	source, ok := artifactRegenerationSource(artifact)
	if !ok {
		return artifactsSvc.Artifact{}, fmt.Errorf("document brief artifact %s has no source artifact metadata", artifact.RelPath)
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	sourceArtifact := artifactsSvc.Artifact{
		Kind:        metadata.Kind,
		Title:       metadata.Title,
		RelPath:     source,
		Source:      metadata.Source,
		SourcePaths: append([]string{}, metadata.SourcePaths...),
	}
	return buildDocumentBriefArtifact(ctx, workspaceRoot, sourceArtifact)
}

func buildDocumentExportRefreshArtifact(ctx context.Context, workspaceRoot string, artifact artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	source, ok := artifactRegenerationSource(artifact)
	if !ok {
		return artifactsSvc.Artifact{}, fmt.Errorf("document export artifact %s has no source brief metadata", artifact.RelPath)
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(source)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	sourceArtifact := artifactsSvc.Artifact{
		Kind:        metadata.Kind,
		Title:       metadata.Title,
		RelPath:     source,
		Source:      metadata.Source,
		SourcePaths: append([]string{}, metadata.SourcePaths...),
	}
	return buildDocumentExportArtifact(ctx, workspaceRoot, sourceArtifact)
}

func (v *View) finishDocumentBriefJob(jobID string, source artifactsSvc.Artifact, created artifactsSvc.Artifact, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Document brief generation canceled.", nil)
			v.artifacts.status.SetText("Document brief generation canceled.")
			v.addActivity("Canceled document brief generation for " + source.RelPath + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Document brief generation failed.", err)
			v.artifacts.status.SetText("Document brief generation failed for " + source.RelPath)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	created.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+created.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+created.RelPath+".", nil)
	v.artifacts.preview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Created " + created.RelPath)
	v.addActivity("Generated document brief from " + source.RelPath + " at " + created.RelPath + ".")
	v.persistArtifactRecord(created)
	v.refreshArtifactsWithQuery("kind:document-brief")
	v.refreshJobs()
}

func (v *View) finishDocumentExportJob(jobID string, source artifactsSvc.Artifact, created artifactsSvc.Artifact, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Document export canceled.", nil)
			v.artifacts.status.SetText("Document export canceled.")
			v.addActivity("Canceled document export for " + source.RelPath + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Document export failed.", err)
			v.artifacts.status.SetText("Document export failed for " + source.RelPath)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	created.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+created.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+created.RelPath+".", nil)
	v.artifacts.preview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Created " + created.RelPath)
	v.addActivity("Exported document brief " + source.RelPath + " to " + created.RelPath + ".")
	v.persistArtifactRecord(created)
	v.refreshArtifactsWithQuery("kind:document-export")
	v.refreshJobs()
}

func (v *View) finishPresentationOutlineJob(jobID string, source artifactsSvc.Artifact, created artifactsSvc.Artifact, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Presentation outline generation canceled.", nil)
			v.artifacts.status.SetText("Presentation outline generation canceled.")
			v.addActivity("Canceled presentation outline generation for " + source.RelPath + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Presentation outline generation failed.", err)
			v.artifacts.status.SetText("Presentation outline generation failed for " + source.RelPath)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	created.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+created.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+created.RelPath+".", nil)
	v.artifacts.preview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Created " + created.RelPath)
	v.addActivity("Generated presentation outline from " + source.RelPath + " at " + created.RelPath + ".")
	v.persistArtifactRecord(created)
	v.refreshArtifactsWithQuery("kind:presentation-outline")
	v.refreshJobs()
}

func (v *View) finishPresentationPackageJob(jobID string, source artifactsSvc.Artifact, created artifactsSvc.Artifact, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Presentation packaging canceled.", nil)
			v.artifacts.status.SetText("Presentation packaging canceled.")
			v.addActivity("Canceled presentation packaging for " + source.RelPath + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Presentation packaging failed.", err)
			v.artifacts.status.SetText("Presentation packaging failed for " + source.RelPath)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	created.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+created.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+created.RelPath+".", nil)
	v.artifacts.preview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Created " + created.RelPath)
	v.addActivity("Packaged presentation outline " + source.RelPath + " at " + created.RelPath + ".")
	v.persistArtifactRecord(created)
	v.refreshArtifactsWithQuery("kind:presentation-package")
	v.refreshJobs()
}

func (v *View) finishPresentationDeckJob(jobID string, source artifactsSvc.Artifact, created artifactsSvc.Artifact, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Presentation deck export canceled.", nil)
			v.artifacts.status.SetText("Presentation deck export canceled.")
			v.addActivity("Canceled presentation deck export for " + source.RelPath + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Presentation deck export failed.", err)
			v.artifacts.status.SetText("Presentation deck export failed for " + source.RelPath)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	created.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+created.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created "+created.RelPath+".", nil)
	v.artifacts.preview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Created " + created.RelPath)
	v.addActivity("Exported presentation package " + source.RelPath + " to " + created.RelPath + ".")
	v.persistArtifactRecord(created)
	v.refreshArtifactsWithQuery("kind:presentation-deck")
	v.refreshJobs()
}

func (v *View) regenerateArtifact(artifact artifactsSvc.Artifact) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before regenerating artifacts.")
		return
	}
	if !artifactCanRegenerate(artifact) {
		v.artifacts.status.SetText("Artifact kind cannot be regenerated yet: " + artifact.Kind)
		v.addActivity("Artifact regeneration is not available for " + artifact.Kind + ".")
		return
	}
	jobLabel := artifactRegenerationJobLabel(artifact)
	job, ctx := v.jobService.Start(artifactJobKindRegenerate, jobLabel)
	v.jobService.AppendLog(job.ID, "Artifact: "+artifact.RelPath)
	v.jobService.AppendLog(job.ID, "Kind: "+artifact.Kind)
	v.artifacts.status.SetText("Regenerating " + artifact.RelPath + " as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		rebuilt, err := v.buildRegeneratedArtifact(ctx, root, artifact)
		fyne.Do(func() {
			v.finishArtifactRegenerationJob(job.ID, artifact, rebuilt, err)
		})
	}()
}

func (v *View) buildRegeneratedArtifact(ctx context.Context, workspaceRoot string, artifact artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	switch strings.TrimSpace(artifact.Kind) {
	case "scan-report":
		return v.buildWorkspaceScanReportArtifact(ctx, workspaceRoot)
	case "document-report":
		roots, ok := artifactRegenerationSources(artifact)
		if !ok {
			return artifactsSvc.Artifact{}, fmt.Errorf("document report artifact %s has no source path metadata", artifact.RelPath)
		}
		return v.buildDocumentSetArtifactFromRoots(ctx, workspaceRoot, roots)
	case "document-extract":
		source, ok := artifactRegenerationSource(artifact)
		if !ok {
			return artifactsSvc.Artifact{}, fmt.Errorf("document extraction artifact %s has no source path metadata", artifact.RelPath)
		}
		return v.buildDocumentExtractionArtifact(ctx, workspaceRoot, source)
	case "operations-runbook":
		source, ok := artifactRegenerationSource(artifact)
		if !ok {
			return artifactsSvc.Artifact{}, fmt.Errorf("operations runbook artifact %s has no source path metadata", artifact.RelPath)
		}
		_, rebuilt, err := v.buildOperationsRunbookArtifact(ctx, workspaceRoot, source)
		return rebuilt, err
	case "artifact-comparison":
		left, right, ok := artifactRegenerationPair(artifact)
		if !ok {
			return artifactsSvc.Artifact{}, fmt.Errorf("comparison artifact %s has no compared artifact path metadata", artifact.RelPath)
		}
		return buildArtifactComparisonReport(ctx, workspaceRoot, left, right)
	case "chat-answer":
		return buildChatAnswerRefreshArtifact(ctx, workspaceRoot, artifact)
	case "document-brief":
		return buildDocumentBriefRefreshArtifact(ctx, workspaceRoot, artifact)
	case "document-export":
		return buildDocumentExportRefreshArtifact(ctx, workspaceRoot, artifact)
	case "presentation-outline":
		return buildPresentationOutlineRefreshArtifact(ctx, workspaceRoot, artifact)
	case "presentation-package":
		return buildPresentationPackageRefreshArtifact(ctx, workspaceRoot, artifact)
	case "presentation-deck":
		return buildPresentationDeckRefreshArtifact(ctx, workspaceRoot, artifact)
	default:
		return artifactsSvc.Artifact{}, fmt.Errorf("artifact kind %q cannot be regenerated yet", artifact.Kind)
	}
}

func buildChatAnswerRefreshArtifact(ctx context.Context, workspaceRoot string, artifact artifactsSvc.Artifact) (artifactsSvc.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	metadata, err := store.ReadArtifactMetadata(artifact.RelPath)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if strings.TrimSpace(metadata.Kind) != "chat-answer" {
		return artifactsSvc.Artifact{}, fmt.Errorf("artifact %s metadata kind %q is not chat-answer", artifact.RelPath, metadata.Kind)
	}
	text, err := store.ReadArtifactText(artifact.RelPath)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	content := artifactsSvc.ExtractChatAnswerContent(text)
	if strings.TrimSpace(content) == "" {
		return artifactsSvc.Artifact{}, fmt.Errorf("chat answer artifact %s has no answer content", artifact.RelPath)
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return store.WriteChatAnswer(artifactsSvc.ChatAnswerReport{
		Title:                  metadata.Title,
		Prompt:                 metadata.Prompt,
		Content:                content,
		Source:                 metadata.Source,
		ContextRelPath:         metadata.ContextRelPath,
		Model:                  metadata.Model,
		SourcePaths:            append([]string{}, metadata.SourcePaths...),
		CitationRefs:           append([]string{}, metadata.CitationRefs...),
		UnverifiedCitationRefs: append([]string{}, metadata.UnverifiedCitationRefs...),
		CitationSnippets:       append([]string{}, metadata.CitationSnippets...),
		CitedSourcePaths:       append([]string{}, metadata.CitedSourcePaths...),
		UncitedSourcePaths:     append([]string{}, metadata.UncitedSourcePaths...),
		EvidenceQuality:        metadata.EvidenceQuality,
		EvidenceSummary:        metadata.EvidenceSummary,
	})
}

func buildArtifactComparisonReport(ctx context.Context, workspaceRoot string, left string, right string) (artifactsSvc.Artifact, error) {
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	store, err := artifactsSvc.NewStore(workspaceRoot)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	comparison, err := store.CompareArtifacts(left, right)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	rebuilt, err := store.WriteArtifactComparisonReport(comparison)
	if err != nil {
		return artifactsSvc.Artifact{}, err
	}
	if err := ctx.Err(); err != nil {
		return artifactsSvc.Artifact{}, err
	}
	return rebuilt, nil
}

func (v *View) finishArtifactRegenerationJob(jobID string, original artifactsSvc.Artifact, rebuilt artifactsSvc.Artifact, err error) {
	if err != nil {
		if errors.Is(err, context.Canceled) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Artifact regeneration canceled.", nil)
			v.artifacts.status.SetText("Artifact regeneration canceled.")
			v.addActivity("Canceled artifact regeneration for " + original.RelPath + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Artifact regeneration failed.", err)
			v.artifacts.status.SetText("Artifact regeneration failed for " + original.RelPath)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	rebuilt.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+rebuilt.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Regenerated "+rebuilt.RelPath+".", nil)
	v.artifacts.preview.SetText("")
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Regenerated " + rebuilt.RelPath)
	v.addActivity("Regenerated " + original.RelPath + " to " + rebuilt.RelPath + ".")
	v.persistArtifactRecord(rebuilt)
	v.refreshArtifactsWithQuery("kind:" + rebuilt.Kind)
	v.refreshJobs()
}

func (v *View) exportArtifactComparison() {
	if !artifactComparisonReady(v.artifacts.lastComparison) {
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
	artifact, err := store.WriteArtifactComparisonReport(v.artifacts.lastComparison)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifacts.status.SetText("Exported " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.persistArtifactRecord(artifact)
	v.refreshArtifactsWithQuery("kind:artifact-comparison")
}

func (v *View) exportArtifactLineageGraph() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before exporting artifact lineage.")
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	lineage, err := store.LineageGraph(artifactsSvc.ListOptions{IncludeArchived: v.artifacts.includeArchived})
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	artifact, err := store.WriteLineageGraphArtifact(lineage)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifacts.preview.SetText(artifactLineageText(lineage))
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText("Exported " + artifact.RelPath)
	v.addActivity(artifact.Message)
	v.persistArtifactRecord(artifact)
	v.refreshArtifactsWithQuery("kind:artifact-lineage")
}

func (v *View) importArtifactLineageGraph() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before importing artifact lineage.")
		return
	}
	relPath := selectedPathOrEmpty(v)
	if strings.TrimSpace(relPath) == "" {
		v.addActivity("Select an artifact lineage JSON file before importing lineage.")
		return
	}
	preview, err := v.workspaceService.PreviewFile(workspace.Root, relPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	result, err := artifactsSvc.ParseLineageJSON(preview.Text, preview.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.artifacts.preview.SetText(artifactLineageText(result.Lineage))
	v.refreshArtifactSources(nil)
	v.artifacts.status.SetText(result.Message)
	v.addActivity(result.Message)
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
		v.artifacts.preview.SetText("")
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
		v.artifacts.preview.SetText("")
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
		v.artifacts.preview.SetText("")
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
	if v.artifacts.sources == nil || v.artifacts.sourceStatus == nil {
		return
	}
	v.artifacts.sources.Objects = nil
	if len(sources) == 0 {
		v.artifacts.sourceStatus.SetText("Sources: none available for this artifact.")
		v.artifacts.sources.Add(widget.NewLabel("No cited source files."))
		v.artifacts.sources.Refresh()
		return
	}
	v.artifacts.sourceStatus.SetText(artifactSourceStatusText(sources))
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
		v.artifacts.sources.Add(container.NewBorder(nil, nil, container.NewHBox(open, pin), nil, label))
	}
	v.artifacts.sources.Refresh()
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
	onDocumentBrief func(artifactsSvc.Artifact),
	onPresentation func(artifactsSvc.Artifact),
	onRegenerate func(artifactsSvc.Artifact),
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
		documentBrief := widget.NewButtonWithIcon("", theme.FileTextIcon(), func() {
			onDocumentBrief(artifact)
		})
		documentBrief.Importance = widget.LowImportance
		presentation := widget.NewButtonWithIcon("", theme.DocumentCreateIcon(), func() {
			onPresentation(artifact)
		})
		presentation.Importance = widget.LowImportance
		regenerate := widget.NewButtonWithIcon("", theme.MediaReplayIcon(), func() {
			onRegenerate(artifact)
		})
		regenerate.Importance = widget.LowImportance
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
			regenerate.Disable()
		} else {
			restore.Disable()
		}
		if !artifactCanRegenerate(artifact) {
			regenerate.Disable()
		}
		if !artifactCanGenerateDocumentArtifact(artifact) {
			documentBrief.Disable()
		}
		if !artifactCanGeneratePresentationArtifact(artifact) {
			presentation.Disable()
		}
		deleteButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			onDelete(artifact)
		})
		deleteButton.Importance = widget.LowImportance
		title := widget.NewLabel(artifactTitle(artifact))
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Truncation = fyne.TextTruncateEllipsis
		badges := widget.NewLabel(artifactBadgeLine(artifact))
		badges.Truncation = fyne.TextTruncateEllipsis
		meta := widget.NewLabel(artifactMeta(artifact))
		meta.Truncation = fyne.TextTruncateEllipsis
		actions := container.NewHBox(preview, context, compare, documentBrief, presentation, regenerate, archive, restore, deleteButton)
		rows = append(rows, container.NewBorder(nil, nil, actions, nil, container.NewVBox(title, badges, meta)))
	}
	return rows
}

func artifactCanGenerateDocumentBrief(artifact artifactsSvc.Artifact) bool {
	if artifact.Archived || strings.TrimSpace(artifact.RelPath) == "" {
		return false
	}
	switch strings.TrimSpace(artifact.Kind) {
	case "document-report", "document-extract", "operations-runbook", "artifact-comparison", "chat-answer", "scan-report", "sql-notebook-run", "presentation-outline":
		return true
	default:
		return false
	}
}

func artifactCanGenerateDocumentExport(artifact artifactsSvc.Artifact) bool {
	return !artifact.Archived &&
		strings.TrimSpace(artifact.RelPath) != "" &&
		strings.TrimSpace(artifact.Kind) == "document-brief"
}

func artifactCanGenerateDocumentArtifact(artifact artifactsSvc.Artifact) bool {
	return artifactCanGenerateDocumentBrief(artifact) || artifactCanGenerateDocumentExport(artifact)
}

func artifactCanGeneratePresentationOutline(artifact artifactsSvc.Artifact) bool {
	if artifact.Archived || strings.TrimSpace(artifact.RelPath) == "" {
		return false
	}
	switch strings.TrimSpace(artifact.Kind) {
	case "document-report", "document-extract", "operations-runbook", "artifact-comparison", "chat-answer", "scan-report", "sql-notebook-run":
		return true
	default:
		return false
	}
}

func artifactCanGeneratePresentationPackage(artifact artifactsSvc.Artifact) bool {
	return !artifact.Archived &&
		strings.TrimSpace(artifact.RelPath) != "" &&
		strings.TrimSpace(artifact.Kind) == "presentation-outline"
}

func artifactCanGeneratePresentationDeck(artifact artifactsSvc.Artifact) bool {
	return !artifact.Archived &&
		strings.TrimSpace(artifact.RelPath) != "" &&
		strings.TrimSpace(artifact.Kind) == "presentation-package"
}

func artifactCanGeneratePresentationArtifact(artifact artifactsSvc.Artifact) bool {
	return artifactCanGeneratePresentationOutline(artifact) || artifactCanGeneratePresentationPackage(artifact) || artifactCanGeneratePresentationDeck(artifact)
}

func artifactCanRegenerate(artifact artifactsSvc.Artifact) bool {
	if artifact.Archived {
		return false
	}
	switch strings.TrimSpace(artifact.Kind) {
	case "scan-report":
		return true
	case "document-report":
		_, ok := artifactRegenerationSources(artifact)
		return ok
	case "document-extract":
		_, ok := artifactRegenerationSource(artifact)
		return ok
	case "operations-runbook":
		_, ok := artifactRegenerationSource(artifact)
		return ok
	case "artifact-comparison":
		_, _, ok := artifactRegenerationPair(artifact)
		return ok
	case "chat-answer":
		return strings.TrimSpace(artifact.RelPath) != "" && strings.TrimSpace(artifact.MetadataPath) != ""
	case "document-brief":
		source, ok := artifactRegenerationSource(artifact)
		return ok && strings.HasPrefix(source, ".nexusdesk/artifacts/")
	case "document-export":
		source, ok := artifactRegenerationSource(artifact)
		return ok && strings.HasPrefix(source, ".nexusdesk/artifacts/document-briefs/")
	case "presentation-outline":
		source, ok := artifactRegenerationSource(artifact)
		return ok && strings.HasPrefix(source, ".nexusdesk/artifacts/")
	case "presentation-package":
		source, ok := artifactRegenerationSource(artifact)
		return ok && strings.HasPrefix(source, ".nexusdesk/artifacts/presentations/")
	case "presentation-deck":
		source, ok := artifactRegenerationSource(artifact)
		return ok && strings.HasPrefix(source, ".nexusdesk/artifacts/presentations/")
	default:
		return false
	}
}

func artifactRegenerationSource(artifact artifactsSvc.Artifact) (string, bool) {
	sources := normalizeArtifactRegenerationSources(append(append([]string{}, artifact.SourcePaths...), artifact.Source), false)
	if len(sources) == 0 {
		return "", false
	}
	return sources[0], true
}

func artifactRegenerationSources(artifact artifactsSvc.Artifact) ([]string, bool) {
	var candidates []string
	if strings.TrimSpace(artifact.Source) != "" {
		if strings.Contains(artifact.Source, ",") {
			candidates = strings.Split(artifact.Source, ",")
		} else {
			candidates = append(candidates, artifact.Source)
		}
	}
	sources := normalizeArtifactRegenerationSources(candidates, true)
	if len(sources) == 0 {
		sources = normalizeArtifactRegenerationSources(artifact.SourcePaths, true)
	}
	return sources, len(sources) > 0
}

func normalizeArtifactRegenerationSources(candidates []string, allowProjectRoot bool) []string {
	seen := make(map[string]struct{}, len(candidates))
	sources := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.ToSlash(strings.TrimSpace(candidate))
		if candidate == "" || strings.Contains(candidate, ",") {
			continue
		}
		if candidate == "." && !allowProjectRoot {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		sources = append(sources, candidate)
	}
	return sources
}

func artifactRegenerationPair(artifact artifactsSvc.Artifact) (string, string, bool) {
	candidates := append([]string{}, artifact.SourcePaths...)
	if len(candidates) < 2 {
		candidates = nil
		if strings.Contains(artifact.Source, ",") {
			candidates = strings.Split(artifact.Source, ",")
		}
	}
	paths := make([]string, 0, 2)
	for _, candidate := range candidates {
		candidate = filepath.ToSlash(strings.TrimSpace(candidate))
		if candidate == "" || candidate == "." || strings.Contains(candidate, ",") {
			continue
		}
		paths = append(paths, candidate)
		if len(paths) == 2 {
			break
		}
	}
	if len(paths) != 2 || paths[0] == paths[1] {
		return "", "", false
	}
	return paths[0], paths[1], true
}

func artifactRegenerationJobLabel(artifact artifactsSvc.Artifact) string {
	title := artifactTitle(artifact)
	if strings.TrimSpace(title) == "" {
		title = artifact.Kind
	}
	return "Regenerate artifact (" + title + ")"
}

func presentationOutlineJobLabel(artifact artifactsSvc.Artifact) string {
	title := artifactTitle(artifact)
	if strings.TrimSpace(title) == "" {
		title = artifact.Kind
	}
	return "Presentation outline (" + title + ")"
}

func documentBriefJobLabel(artifact artifactsSvc.Artifact) string {
	title := artifactTitle(artifact)
	if strings.TrimSpace(title) == "" {
		title = artifact.Kind
	}
	return "Document brief (" + title + ")"
}

func documentExportJobLabel(artifact artifactsSvc.Artifact) string {
	title := artifactTitle(artifact)
	if strings.TrimSpace(title) == "" {
		title = artifact.Kind
	}
	return "Document DOCX export (" + title + ")"
}

func presentationPackageJobLabel(artifact artifactsSvc.Artifact) string {
	title := artifactTitle(artifact)
	if strings.TrimSpace(title) == "" {
		title = artifact.Kind
	}
	return "Presentation package (" + title + ")"
}

func presentationDeckJobLabel(artifact artifactsSvc.Artifact) string {
	title := artifactTitle(artifact)
	if strings.TrimSpace(title) == "" {
		title = artifact.Kind
	}
	return "Presentation PPTX deck (" + title + ")"
}

func artifactTitle(artifact artifactsSvc.Artifact) string {
	if artifact.Title != "" {
		return artifact.Title
	}
	return filepath.Base(artifact.RelPath)
}

func documentSetArtifactTitle(root string) string {
	return documentSetArtifactTitleForRoots([]string{root})
}

func documentSetArtifactTitleForRoots(roots []string) string {
	roots = normalizeArtifactRegenerationSources(roots, true)
	if len(roots) == 0 || (len(roots) == 1 && roots[0] == ".") {
		return "Project Document Set Report"
	}
	if len(roots) == 1 {
		return "Document Set Report - " + roots[0]
	}
	return fmt.Sprintf("Document Set Report - %d sources", len(roots))
}

func documentSetArtifactJobLabel(root string) string {
	root = strings.TrimSpace(root)
	if root == "" || root == "." {
		return "Document report (project)"
	}
	return "Document report (" + root + ")"
}

func workspaceScanReportJobLabel(workspaceName string) string {
	workspaceName = strings.TrimSpace(workspaceName)
	if workspaceName == "" {
		return "Workspace scan report"
	}
	return "Workspace scan report (" + workspaceName + ")"
}

func documentExtractionArtifactJobLabel(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return "Document extraction"
	}
	return "Document extraction (" + source + ")"
}

func workspaceScanArtifactInput(report workspaceSvc.ScanReport) artifactsSvc.WorkspaceScanReport {
	return artifactsSvc.WorkspaceScanReport{
		WorkspaceName:  report.Name,
		Included:       report.Included,
		Ignored:        report.Ignored,
		DepthSkipped:   report.DepthSkipped,
		EntrySkipped:   report.EntrySkipped,
		Unreadable:     report.Unreadable,
		MaxDepth:       report.MaxDepth,
		MaxEntries:     report.MaxEntries,
		Truncated:      report.Truncated,
		IgnoredSamples: append([]string{}, report.IgnoredSamples...),
		SkippedSamples: append([]string{}, report.SkippedSamples...),
		Message:        report.Message(),
	}
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

func artifactPreviewSummaryText(artifact artifactsSvc.Artifact) string {
	var builder strings.Builder
	builder.WriteString("Artifact Summary\n")
	builder.WriteString("- Type: ")
	builder.WriteString(artifactKindLabel(artifact.Kind))
	builder.WriteString("\n")
	builder.WriteString("- Path: ")
	builder.WriteString(firstNonEmpty(strings.TrimSpace(artifact.RelPath), "(no path)"))
	builder.WriteString("\n")
	if title := strings.TrimSpace(artifactTitle(artifact)); title != "" {
		builder.WriteString("- Title: ")
		builder.WriteString(title)
		builder.WriteString("\n")
	}
	builder.WriteString("- Status: ")
	builder.WriteString(strings.Join(artifactCapabilityBadges(artifact), ", "))
	builder.WriteString("\n")
	if len(artifact.SourcePaths) > 0 {
		builder.WriteString("- Sources: ")
		builder.WriteString(fmt.Sprintf("%d", len(artifact.SourcePaths)))
		builder.WriteString("\n")
	}
	if strings.TrimSpace(artifact.JobID) != "" {
		builder.WriteString("- Job: ")
		builder.WriteString(artifact.JobID)
		builder.WriteString("\n")
	}
	return builder.String()
}

func artifactBadgeLine(artifact artifactsSvc.Artifact) string {
	badges := append([]string{artifactKindLabel(artifact.Kind)}, artifactCapabilityBadges(artifact)...)
	return strings.Join(badges, " | ")
}

func artifactKindLabel(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return "Unknown artifact"
	}
	labels := map[string]string{
		"artifact-comparison":   "Comparison report",
		"chat-answer":           "Assistant answer",
		"dataset-dashboard":     "Dashboard",
		"dataset-query-csv":     "Dataset CSV",
		"dataset-sql-report":    "SQL report",
		"dataset-summary":       "Dataset summary",
		"document-brief":        "Document brief",
		"document-export":       "DOCX export",
		"document-extract":      "Extracted document",
		"document-report":       "Document report",
		"operations-runbook":    "Operations runbook",
		"presentation-deck":     "PPTX deck",
		"presentation-outline":  "Presentation outline",
		"presentation-package":  "Presentation package",
		"scan-report":           "Workspace scan",
		"sql-notebook-run":      "SQL notebook",
		"sqlite-query-csv":      "SQLite CSV",
		"sqlite-query-markdown": "SQLite report",
		"task-report":           "Task report",
	}
	if label, ok := labels[kind]; ok {
		return label
	}
	parts := strings.Split(kind, "-")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func artifactCapabilityBadges(artifact artifactsSvc.Artifact) []string {
	badges := []string{}
	if artifact.Archived {
		badges = append(badges, "archived")
	} else {
		badges = append(badges, "active")
	}
	if len(artifact.SourcePaths) > 0 || strings.TrimSpace(artifact.Source) != "" {
		badges = append(badges, "lineage")
	} else {
		badges = append(badges, "no lineage")
	}
	if artifactCanRegenerate(artifact) {
		badges = append(badges, "regenerable")
	}
	if artifactCanGenerateDocumentArtifact(artifact) {
		badges = append(badges, "doc export")
	}
	if artifactCanGeneratePresentationArtifact(artifact) {
		badges = append(badges, "deck export")
	}
	if strings.TrimSpace(artifact.Kind) == "artifact-comparison" {
		badges = append(badges, "comparison")
	}
	if strings.TrimSpace(artifact.MetadataPath) != "" {
		badges = append(badges, "metadata")
	}
	return badges
}

func artifactLineageText(lineage artifactsSvc.Lineage) string {
	if len(lineage.Nodes) == 0 {
		return "Lineage: no metadata available."
	}
	var builder strings.Builder
	builder.WriteString("Lineage\n")
	if strings.TrimSpace(lineage.Message) != "" {
		builder.WriteString("- ")
		builder.WriteString(lineage.Message)
		builder.WriteString("\n")
	}
	if len(lineage.RelationshipCounts) > 0 {
		builder.WriteString("- Relationships by type: ")
		labels := make([]string, 0, len(lineage.RelationshipCounts))
		for label := range lineage.RelationshipCounts {
			labels = append(labels, label)
		}
		sort.Strings(labels)
		for index, label := range labels {
			if index > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(label)
			builder.WriteString("=")
			builder.WriteString(fmt.Sprintf("%d", lineage.RelationshipCounts[label]))
		}
		builder.WriteString("\n")
	}
	for _, node := range lineage.Nodes {
		builder.WriteString("- ")
		builder.WriteString(node.Kind)
		builder.WriteString(": ")
		builder.WriteString(node.Label)
		if node.RelPath != "" {
			builder.WriteString(" (")
			builder.WriteString(node.RelPath)
			builder.WriteString(")")
		}
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
	case source.Unknown:
		status = "unchecked"
	case !source.Exists:
		status = "missing"
	}
	if source.Message != "" {
		return fmt.Sprintf("%s (%s: %s)", source.RelPath, status, source.Message)
	}
	return fmt.Sprintf("%s (%s)", source.RelPath, status)
}

func artifactPreviewText(store *artifactsSvc.Store, artifact artifactsSvc.Artifact) (string, error) {
	if strings.TrimSpace(artifact.Kind) == "document-export" {
		metadata, err := store.ReadArtifactMetadata(artifact.RelPath)
		if err != nil {
			return "", err
		}
		var builder strings.Builder
		builder.WriteString("# ")
		builder.WriteString(firstNonEmpty(metadata.Title, artifactTitle(artifact)))
		builder.WriteString("\n\n")
		builder.WriteString("DOCX document export\n\n")
		writeArtifactComparisonKV(&builder, "Artifact", artifact.RelPath)
		writeArtifactComparisonKV(&builder, "Format", firstNonEmpty(metadata.ExportFormat, "docx"))
		writeArtifactComparisonKV(&builder, "Template", metadata.ExportTemplate)
		writeArtifactComparisonKV(&builder, "Theme", metadata.ThemeName)
		writeArtifactComparisonKV(&builder, "Source brief", metadata.Source)
		writeArtifactPackageValidation(&builder, metadata.PackageValidation)
		if len(metadata.PackageFiles) > 0 {
			builder.WriteString("\n## Package Files\n\n")
			for _, file := range metadata.PackageFiles {
				builder.WriteString("- ")
				builder.WriteString(file)
				builder.WriteString("\n")
			}
		}
		builder.WriteString("\nOpen the DOCX artifact from the artifact path to inspect the generated document.")
		return builder.String(), nil
	}
	if strings.TrimSpace(artifact.Kind) == "presentation-deck" {
		metadata, err := store.ReadArtifactMetadata(artifact.RelPath)
		if err != nil {
			return "", err
		}
		var builder strings.Builder
		builder.WriteString("# ")
		builder.WriteString(firstNonEmpty(metadata.Title, artifactTitle(artifact)))
		builder.WriteString("\n\n")
		builder.WriteString("PPTX presentation deck export\n\n")
		writeArtifactComparisonKV(&builder, "Artifact", artifact.RelPath)
		writeArtifactComparisonKV(&builder, "Format", firstNonEmpty(metadata.ExportFormat, "pptx"))
		writeArtifactComparisonKV(&builder, "Template", metadata.ExportTemplate)
		writeArtifactComparisonKV(&builder, "Theme", metadata.ThemeName)
		writeArtifactComparisonKV(&builder, "Source outline", metadata.Source)
		writeArtifactPackageValidation(&builder, metadata.PackageValidation)
		if len(metadata.PackageFiles) > 0 {
			builder.WriteString("\n## Package Files\n\n")
			for _, file := range metadata.PackageFiles {
				builder.WriteString("- ")
				builder.WriteString(file)
				builder.WriteString("\n")
			}
		}
		builder.WriteString("\nOpen the PPTX artifact from the artifact path to inspect the generated deck.")
		return builder.String(), nil
	}
	if strings.TrimSpace(artifact.Kind) != "presentation-package" {
		return store.ReadArtifactText(artifact.RelPath)
	}
	metadata, err := store.ReadArtifactMetadata(artifact.RelPath)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(firstNonEmpty(metadata.Title, artifactTitle(artifact)))
	builder.WriteString("\n\n")
	builder.WriteString("Packaged presentation export\n\n")
	writeArtifactComparisonKV(&builder, "Artifact", artifact.RelPath)
	writeArtifactComparisonKV(&builder, "Format", firstNonEmpty(metadata.ExportFormat, "zip"))
	writeArtifactComparisonKV(&builder, "Source outline", metadata.Source)
	if len(metadata.PackageFiles) > 0 {
		builder.WriteString("\n## Package Files\n\n")
		for _, file := range metadata.PackageFiles {
			builder.WriteString("- ")
			builder.WriteString(file)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\nOpen the package file from the artifact path to inspect the bundled manifest, outline, and slide payloads.")
	return builder.String(), nil
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

func writeArtifactPackageValidation(builder *strings.Builder, validation *artifactsSvc.PackageValidation) {
	if validation == nil {
		writeArtifactComparisonKV(builder, "Package validation", "not recorded")
		return
	}
	status := "failed"
	if validation.Valid {
		status = "passed"
	}
	details := status
	if strings.TrimSpace(validation.Message) != "" {
		details += " - " + strings.TrimSpace(validation.Message)
	}
	writeArtifactComparisonKV(builder, "Package validation", details)
	if validation.XMLFiles > 0 {
		writeArtifactComparisonKV(builder, "Validated XML parts", fmt.Sprintf("%d", validation.XMLFiles))
	}
	if validation.SlideCount > 0 {
		writeArtifactComparisonKV(builder, "Validated slides", fmt.Sprintf("%d", validation.SlideCount))
	}
}
