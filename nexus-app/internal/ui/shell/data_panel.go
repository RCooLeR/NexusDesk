package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	datasetsSvc "nexusdesk/internal/services/datasets"
)

func (v *View) newDataPanel() fyne.CanvasObject {
	profileButton := widget.NewButtonWithIcon("Profile selected", theme.SearchIcon(), v.profileSelectedDataset)
	header := container.NewBorder(nil, nil, profileButton, nil, v.dataProfileStatus)
	detail := container.NewScroll(v.dataProfileDetail)
	detail.SetMinSize(fyne.NewSize(320, 130))
	return container.NewBorder(header, nil, nil, nil, detail)
}

func (v *View) profileSelectedDataset() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.dataProfileStatus.SetText("Open a workspace before profiling data.")
		return
	}
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.dataProfileStatus.SetText("Select a CSV, TSV, or JSON file first.")
		return
	}
	profile, err := v.datasetService.Profile(workspace.Root, selected)
	if err != nil {
		v.dataProfileStatus.SetText("Profile failed for " + selected)
		dialog.ShowError(err, v.window)
		return
	}
	v.dataProfileStatus.SetText(profileStatus(profile))
	v.dataProfileDetail.SetText(formatDatasetProfile(profile))
	v.addActivity("Profiled dataset " + profile.RelPath + ".")
}

func profileStatus(profile datasetsSvc.Profile) string {
	truncated := ""
	if profile.Truncated {
		truncated = " sample"
	}
	return fmt.Sprintf("%s: %s%s, %d rows, %d columns", profile.RelPath, profile.Format, truncated, profile.Rows, len(profile.Columns))
}

func formatDatasetProfile(profile datasetsSvc.Profile) string {
	var builder strings.Builder
	builder.WriteString("# Dataset Profile\n\n")
	builder.WriteString("Path: ")
	builder.WriteString(profile.RelPath)
	builder.WriteString("\nFormat: ")
	builder.WriteString(profile.Format)
	builder.WriteString("\nMedia type: ")
	builder.WriteString(profile.MediaType)
	builder.WriteString(fmt.Sprintf("\nSize: %d bytes\nRows: %d\nColumns: %d\n", profile.Size, profile.Rows, len(profile.Columns)))
	if profile.Truncated {
		builder.WriteString("Scope: preview sample is truncated by the safe preview cap\n")
	}
	if profile.JSONProfile != nil {
		builder.WriteString("\nJSON\n")
		builder.WriteString("- Top level: ")
		builder.WriteString(profile.JSONProfile.TopLevel)
		builder.WriteString(fmt.Sprintf("\n- Count: %d\n", profile.JSONProfile.Count))
		for _, note := range profile.JSONProfile.Notes {
			builder.WriteString("- ")
			builder.WriteString(note)
			builder.WriteString("\n")
		}
	}
	if len(profile.Columns) == 0 {
		builder.WriteString("\nNo tabular fields were found.\n")
		return builder.String()
	}
	builder.WriteString("\nFields\n")
	for _, column := range profile.Columns {
		builder.WriteString("- ")
		builder.WriteString(column.Name)
		builder.WriteString(" | ")
		builder.WriteString(column.Type)
		builder.WriteString(fmt.Sprintf(" | non-empty %d | empty %d", column.NonEmpty, column.Empty))
		if len(column.Samples) > 0 {
			builder.WriteString(" | samples: ")
			builder.WriteString(strings.Join(column.Samples, ", "))
		}
		builder.WriteString("\n")
	}
	return builder.String()
}
