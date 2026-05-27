package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	operationsSvc "nexusdesk/internal/services/operations"
)

func (v *View) newOperationsPanel() fyne.CanvasObject {
	scanButton := widget.NewButtonWithIcon("Scan ops files", theme.SearchIcon(), v.scanOperationsFiles)
	inspectButton := widget.NewButtonWithIcon("Inspect selected", theme.DocumentIcon(), v.inspectSelectedOperationsFile)
	actions := container.NewHBox(scanButton, inspectButton)
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
	result, err := v.operationsService.Scan(workspace.Root)
	if err != nil {
		v.operationsStatus.SetText("Operations scan failed.")
		dialog.ShowError(err, v.window)
		return
	}
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
	inspection, err := v.operationsService.Inspect(workspace.Root, relPath)
	if err != nil {
		v.operationsStatus.SetText("Operations inspection failed for " + relPath)
		dialog.ShowError(err, v.window)
		return
	}
	v.operationsStatus.SetText(formatOperationsInspectionStatus(inspection))
	v.operationsDetail.SetText(formatOperationsInspection(inspection))
	v.addActivity("Inspected operations file " + inspection.File.RelPath + ".")
}

func formatOperationsScanStatus(result operationsSvc.ScanResult) string {
	if strings.TrimSpace(result.Message) != "" {
		return result.Message
	}
	return fmt.Sprintf("%d operations files found.", len(result.Files))
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
	builder.WriteString("\nContent\n\n")
	builder.WriteString(inspection.Text)
	return builder.String()
}
