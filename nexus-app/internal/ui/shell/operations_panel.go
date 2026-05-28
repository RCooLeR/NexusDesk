package shell

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	jobsSvc "nexusdesk/internal/services/jobs"
	operationsSvc "nexusdesk/internal/services/operations"
)

const (
	operationsScanJobKind    = "operations-scan"
	operationsInspectJobKind = "operations-inspect"
	operationsRunbookJobKind = "operations-runbook"
)

func (v *View) newOperationsPanel() fyne.CanvasObject {
	scanButton := widget.NewButtonWithIcon("Scan ops files", theme.SearchIcon(), v.scanOperationsFiles)
	inspectButton := widget.NewButtonWithIcon("Inspect selected", theme.DocumentIcon(), v.inspectSelectedOperationsFile)
	validateButton := widget.NewButtonWithIcon("Validate Compose", theme.ConfirmIcon(), v.validateSelectedComposeConfig)
	exportButton := widget.NewButtonWithIcon("Export runbook", theme.DocumentSaveIcon(), v.exportSelectedOperationsRunbook)
	actions := container.NewHBox(scanButton, inspectButton, validateButton, exportButton)
	header := container.NewBorder(nil, nil, nil, actions, v.operationsStatus)
	results := container.NewScroll(v.operationsResults)
	results.SetMinSize(fyne.NewSize(280, 120))
	detail := container.NewScroll(v.operationsDetail)
	detail.SetMinSize(fyne.NewSize(360, 140))
	return container.NewHSplit(results, container.NewBorder(header, nil, nil, nil, detail))
}

func (v *View) scanOperationsFiles() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.operationsStatus.SetText("Open a workspace before scanning operations files.")
		return
	}
	jobLabel := operationsScanJobLabel()
	job, ctx := v.jobService.Start(operationsScanJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Workspace: "+workspace.Root)
	v.operationsStatus.SetText("Scanning operations files as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		result, err := v.operationsService.ScanContext(ctx, root)
		fyne.Do(func() {
			v.finishOperationsScanJob(job.ID, result, err)
		})
	}()
}

func (v *View) inspectSelectedOperationsFile() {
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.operationsStatus.SetText("Select an operations file before inspecting it.")
		return
	}
	v.inspectOperationsFile(selected)
}

func (v *View) inspectOperationsFile(relPath string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.operationsStatus.SetText("Open a workspace before inspecting operations files.")
		return
	}
	jobLabel := operationsInspectJobLabel(relPath)
	job, ctx := v.jobService.Start(operationsInspectJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+strings.TrimSpace(relPath))
	v.operationsStatus.SetText("Inspecting operations file as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		inspection, err := v.operationsService.InspectContext(ctx, root, relPath)
		fyne.Do(func() {
			v.finishOperationsInspectJob(job.ID, relPath, inspection, err)
		})
	}()
}

func (v *View) validateSelectedComposeConfig() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.operationsStatus.SetText("Open a workspace before validating Compose config.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.operationsStatus.SetText("Select a Compose file before validating it.")
		return
	}
	inspection, err := v.operationsService.Inspect(workspace.Root, selected)
	if err != nil {
		v.operationsStatus.SetText("Compose validation could not inspect " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	if inspection.File.Kind != operationsSvc.FileKindCompose {
		v.operationsStatus.SetText("Select a Compose file before validating it.")
		return
	}
	task, ok, err := v.taskService.FindBySource(workspace.Root, "compose", inspection.File.RelPath)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	if !ok {
		v.operationsStatus.SetText("No safe Compose validation task found for " + inspection.File.RelPath + ".")
		return
	}
	message := "Run read-only `docker compose config` for " + inspection.File.RelPath + "?"
	dialog.ShowConfirm("Validate Compose config", message, func(confirm bool) {
		if !confirm {
			return
		}
		v.operationsStatus.SetText("Validating Compose config " + inspection.File.RelPath + " as a job.")
		v.runTask(task)
	}, v.window)
}

func (v *View) exportSelectedOperationsRunbook() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.operationsStatus.SetText("Open a workspace before exporting an operations runbook.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.operationsStatus.SetText("Select an operations file before exporting a runbook.")
		return
	}
	jobLabel := operationsRunbookJobLabel(selected)
	job, ctx := v.jobService.Start(operationsRunbookJobKind, jobLabel)
	v.jobService.AppendLog(job.ID, "Path: "+strings.TrimSpace(selected))
	v.operationsStatus.SetText("Exporting operations runbook as " + job.ID + ".")
	v.addActivity("Started " + job.ID + ": " + jobLabel + ".")
	v.refreshJobs()
	root := workspace.Root
	go func() {
		inspection, artifact, err := v.buildOperationsRunbookArtifact(ctx, root, selected)
		fyne.Do(func() {
			v.finishOperationsRunbookJob(job.ID, selected, inspection, artifact, err)
		})
	}()
}

func (v *View) finishOperationsScanJob(jobID string, result operationsSvc.ScanResult, err error) {
	if err != nil {
		if isOperationsJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Operations scan cancelled.", nil)
			v.operationsStatus.SetText("Operations scan cancelled.")
			v.addActivity("Operations scan cancelled.")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Operations scan failed.", err)
			v.operationsStatus.SetText("Operations scan failed.")
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Files=%d compose=%d docker=%d", result.Summary.Files, result.Summary.Compose, result.Summary.Dockerfiles))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, firstNonEmptyString(result.Message, "Operations scan completed."), nil)
	v.operationsStatus.SetText(formatOperationsScanStatus(result))
	v.operationsResults.Objects = nil
	if len(result.Files) == 0 {
		v.operationsResults.Add(widget.NewLabel("No Docker, Compose, env, config, script, or log files found."))
	} else {
		for _, file := range result.Files {
			opsFile := file
			button := widget.NewButton(formatOperationsFileLabel(opsFile), func() {
				v.state.SetSelectedPath(opsFile.RelPath)
				v.inspectOperationsFile(opsFile.RelPath)
			})
			button.Alignment = widget.ButtonAlignLeading
			v.operationsResults.Add(button)
		}
	}
	v.operationsResults.Refresh()
	v.addActivity("Scanned workspace operations files.")
	v.refreshJobs()
}

func (v *View) finishOperationsInspectJob(jobID string, relPath string, inspection operationsSvc.Inspection, err error) {
	if err != nil {
		if isOperationsJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Operations inspection cancelled.", nil)
			v.operationsStatus.SetText("Operations inspection cancelled for " + relPath + ".")
			v.addActivity("Operations inspection cancelled for " + relPath + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Operations inspection failed.", err)
			v.operationsStatus.SetText("Operations inspection failed for " + relPath)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	v.jobService.AppendLog(jobID, fmt.Sprintf("Kind=%s services=%d warnings=%d", inspection.File.Kind, len(inspection.Services), len(inspection.Warnings)))
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Operations inspection completed for "+inspection.File.RelPath+".", nil)
	v.operationsStatus.SetText(formatOperationsInspectionStatus(inspection))
	v.operationsDetail.SetText(formatOperationsInspection(inspection))
	v.addActivity("Inspected operations file " + inspection.File.RelPath + ".")
	v.refreshJobs()
}

func (v *View) buildOperationsRunbookArtifact(ctx context.Context, root string, selected string) (operationsSvc.Inspection, artifactsSvc.Artifact, error) {
	inspection, err := v.operationsService.InspectContext(ctx, root, selected)
	if err != nil {
		return operationsSvc.Inspection{}, artifactsSvc.Artifact{}, err
	}
	select {
	case <-ctx.Done():
		return operationsSvc.Inspection{}, artifactsSvc.Artifact{}, ctx.Err()
	default:
	}
	store, err := artifactsSvc.NewStore(root)
	if err != nil {
		return operationsSvc.Inspection{}, artifactsSvc.Artifact{}, err
	}
	artifact, err := store.WriteOperationsRunbook(operationsRunbookArtifactInput(inspection))
	if err != nil {
		return operationsSvc.Inspection{}, artifactsSvc.Artifact{}, err
	}
	select {
	case <-ctx.Done():
		return operationsSvc.Inspection{}, artifactsSvc.Artifact{}, ctx.Err()
	default:
	}
	return inspection, artifact, nil
}

func (v *View) finishOperationsRunbookJob(jobID string, selected string, inspection operationsSvc.Inspection, artifact artifactsSvc.Artifact, err error) {
	if err != nil {
		if isOperationsJobCanceled(err) {
			v.jobService.Finish(jobID, jobsSvc.StatusCanceled, "Operations runbook export cancelled.", nil)
			v.operationsStatus.SetText("Operations runbook export cancelled for " + selected + ".")
			v.addActivity("Operations runbook export cancelled for " + selected + ".")
		} else {
			v.jobService.Finish(jobID, jobsSvc.StatusFailed, "Operations runbook export failed.", err)
			v.operationsStatus.SetText("Operations runbook export failed for " + selected)
			dialog.ShowError(err, v.window)
		}
		v.refreshJobs()
		return
	}
	artifact.JobID = jobID
	v.jobService.AppendLog(jobID, "Artifact: "+artifact.RelPath)
	v.jobService.Finish(jobID, jobsSvc.StatusSuccess, "Created operations runbook "+artifact.RelPath+".", nil)
	v.operationsStatus.SetText("Created operations runbook " + artifact.RelPath)
	v.operationsDetail.SetText(formatOperationsInspection(inspection))
	v.persistArtifactRecord(artifact)
	v.addActivity(artifact.Message)
	v.refreshArtifacts()
	v.refreshHistory("", "")
	v.refreshJobs()
}

func operationsScanJobLabel() string {
	return "Operations scan"
}

func operationsInspectJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "Operations inspect"
	}
	return "Operations inspect (" + relPath + ")"
}

func operationsRunbookJobLabel(relPath string) string {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return "Operations runbook export"
	}
	return "Operations runbook export (" + relPath + ")"
}

func isOperationsJobCanceled(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func formatOperationsScanStatus(result operationsSvc.ScanResult) string {
	if strings.TrimSpace(result.Message) != "" {
		return result.Message
	}
	return fmt.Sprintf("%d operations files found.", len(result.Files))
}

func operationsRunbookArtifactInput(inspection operationsSvc.Inspection) artifactsSvc.OperationsRunbookReport {
	services := make([]artifactsSvc.OperationsServiceSummary, 0, len(inspection.Services))
	for _, service := range inspection.Services {
		services = append(services, artifactsSvc.OperationsServiceSummary{
			Name:      service.Name,
			Image:     service.Image,
			Ports:     append([]string{}, service.Ports...),
			Volumes:   append([]string{}, service.Volumes...),
			DependsOn: append([]string{}, service.DependsOn...),
		})
	}
	topologyEdges := make([]artifactsSvc.OperationsTopologyEdge, 0, len(inspection.Topology.Edges))
	for _, edge := range inspection.Topology.Edges {
		topologyEdges = append(topologyEdges, artifactsSvc.OperationsTopologyEdge{
			From:     edge.From,
			To:       edge.To,
			Relation: edge.Relation,
			Missing:  edge.Missing,
		})
	}
	exposedPorts := make([]artifactsSvc.OperationsPortExposure, 0, len(inspection.Topology.ExposedPorts))
	for _, exposure := range inspection.Topology.ExposedPorts {
		exposedPorts = append(exposedPorts, artifactsSvc.OperationsPortExposure{
			Service: exposure.Service,
			Port:    exposure.Port,
		})
	}
	warnings := append([]string{}, inspection.Warnings...)
	warnings = append(warnings, inspection.Topology.Warnings...)
	return artifactsSvc.OperationsRunbookReport{
		Title:           "Operations Runbook - " + inspection.File.Name,
		SourcePath:      inspection.File.RelPath,
		Kind:            string(inspection.File.Kind),
		Size:            inspection.File.Size,
		Content:         formatOperationsInspection(inspection),
		Services:        services,
		TopologySummary: inspection.Topology.Summary,
		TopologyEdges:   topologyEdges,
		ExposedPorts:    exposedPorts,
		NamedVolumes:    append([]string{}, inspection.Topology.NamedVolumes...),
		Warnings:        warnings,
		Truncated:       inspection.Truncated,
		GeneratedBy:     "Nexus native operations inspector",
	}
}

func formatOperationsFileLabel(file operationsSvc.File) string {
	return fmt.Sprintf("%s  [%s, %d bytes]", file.RelPath, file.Kind, file.Size)
}

func formatOperationsInspectionStatus(inspection operationsSvc.Inspection) string {
	servicePart := ""
	if len(inspection.Services) > 0 {
		servicePart = fmt.Sprintf(", %d Compose services", len(inspection.Services))
	}
	return fmt.Sprintf("%s inspected as %s%s.", inspection.File.RelPath, inspection.File.Kind, servicePart)
}

func formatOperationsInspection(inspection operationsSvc.Inspection) string {
	var builder strings.Builder
	builder.WriteString("# Operations Inspection\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(inspection.File.RelPath)
	builder.WriteString("\nKind: ")
	builder.WriteString(string(inspection.File.Kind))
	builder.WriteString(fmt.Sprintf("\nSize: %d bytes\n", inspection.File.Size))
	if inspection.Truncated {
		builder.WriteString("Scope: content was truncated for interactive inspection\n")
	}
	for _, warning := range inspection.Warnings {
		builder.WriteString("Warning: ")
		builder.WriteString(warning)
		builder.WriteString("\n")
	}
	if len(inspection.Services) > 0 {
		builder.WriteString("\nCompose Services\n")
		for _, service := range inspection.Services {
			builder.WriteString("- ")
			builder.WriteString(service.Name)
			if service.Image != "" {
				builder.WriteString(" | image: ")
				builder.WriteString(service.Image)
			}
			if len(service.Ports) > 0 {
				builder.WriteString(" | ports: ")
				builder.WriteString(strings.Join(service.Ports, ", "))
			}
			if len(service.Volumes) > 0 {
				builder.WriteString(" | volumes: ")
				builder.WriteString(strings.Join(service.Volumes, ", "))
			}
			if len(service.DependsOn) > 0 {
				builder.WriteString(" | depends on: ")
				builder.WriteString(strings.Join(service.DependsOn, ", "))
			}
			builder.WriteString("\n")
		}
	}
	if inspection.Topology.Summary != "" {
		builder.WriteString("\nCompose Topology\n")
		builder.WriteString(inspection.Topology.Summary)
		builder.WriteString("\n")
		if len(inspection.Topology.Edges) > 0 {
			builder.WriteString("Dependencies\n")
			for _, edge := range inspection.Topology.Edges {
				builder.WriteString("- ")
				builder.WriteString(edge.From)
				builder.WriteString(" -> ")
				builder.WriteString(edge.To)
				if edge.Relation != "" {
					builder.WriteString(" (")
					builder.WriteString(edge.Relation)
					builder.WriteString(")")
				}
				if edge.Missing {
					builder.WriteString(" [missing target]")
				}
				builder.WriteString("\n")
			}
		}
		if len(inspection.Topology.ExposedPorts) > 0 {
			builder.WriteString("Exposed Ports\n")
			for _, exposure := range inspection.Topology.ExposedPorts {
				builder.WriteString("- ")
				builder.WriteString(exposure.Service)
				builder.WriteString(" exposes ")
				builder.WriteString(exposure.Port)
				builder.WriteString("\n")
			}
		}
		if len(inspection.Topology.NamedVolumes) > 0 {
			builder.WriteString("Named Volumes: ")
			builder.WriteString(strings.Join(inspection.Topology.NamedVolumes, ", "))
			builder.WriteString("\n")
		}
		for _, warning := range inspection.Topology.Warnings {
			builder.WriteString("Topology Warning: ")
			builder.WriteString(warning)
			builder.WriteString("\n")
		}
	}
	builder.WriteString("\nContent\n\n")
	builder.WriteString(inspection.Text)
	return builder.String()
}
