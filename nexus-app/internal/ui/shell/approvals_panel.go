package shell

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	approvalsSvc "nexusdesk/internal/services/approvals"
)

func (v *View) newApprovalsPanel() fyne.CanvasObject {
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), v.refreshApprovals)
	grant := widget.NewButtonWithIcon("Grant 1h", theme.ConfirmIcon(), v.confirmGrantFullProjectAccess)
	revoke := widget.NewButtonWithIcon("Revoke", theme.CancelIcon(), v.revokeFullProjectAccess)
	actions := container.NewHBox(refresh, grant, revoke)
	header := container.NewBorder(nil, nil, container.NewVBox(v.accessStatus, v.approvalStatus), actions)
	scroll := container.NewScroll(v.approvalResults)
	scroll.SetMinSize(fyne.NewSize(260, 110))
	return container.NewBorder(header, nil, nil, nil, scroll)
}

func (v *View) refreshApprovals() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.accessStatus.SetText("Full project access: inactive")
		v.approvalStatus.SetText("Open a workspace before reading approvals.")
		v.approvalResults.Objects = []fyne.CanvasObject{widget.NewLabel("Open a workspace to inspect approval records.")}
		v.approvalResults.Refresh()
		return
	}
	policy, err := v.approvalService.LoadPolicy(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	records, err := v.approvalService.List(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.accessStatus.SetText(policyStatusText(policy))
	v.approvalStatus.SetText(fmt.Sprintf("%d approval record(s)", len(records)))
	v.approvalResults.Objects = approvalRows(records)
	v.approvalResults.Refresh()
}

func (v *View) confirmGrantFullProjectAccess() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before changing access policy.")
		return
	}
	message := "Grant full project access for this workspace for 1 hour?\n\nThis is scoped to the active workspace and does not enable arbitrary shell commands."
	dialog.ShowConfirm("Grant full project access", message, func(confirm bool) {
		if !confirm {
			return
		}
		policy, err := v.approvalService.GrantFullProjectAccess(workspace.Root, time.Hour)
		if err != nil {
			v.addActivity("Approval persistence failed: " + err.Error())
			dialog.ShowError(err, v.window)
			return
		}
		v.addActivity(policy.Message)
		v.refreshApprovals()
	}, v.window)
}

func (v *View) revokeFullProjectAccess() {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before changing access policy.")
		return
	}
	policy, err := v.approvalService.RevokeFullProjectAccess(workspace.Root)
	if err != nil {
		v.addActivity("Approval persistence failed: " + err.Error())
		dialog.ShowError(err, v.window)
		return
	}
	v.addActivity(policy.Message)
	v.refreshApprovals()
}

func approvalRows(records []approvalsSvc.Record) []fyne.CanvasObject {
	if len(records) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No approval records yet.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(records))
	for _, record := range records {
		rows = append(rows, approvalRow(record))
	}
	return rows
}

func approvalRow(record approvalsSvc.Record) fyne.CanvasObject {
	details := widget.NewLabel(approvalRecordDetails(record))
	details.Wrapping = fyne.TextWrapWord
	return widget.NewCard(
		firstNonEmpty(record.Action, "approval"),
		approvalRecordSubtitle(record),
		details,
	)
}

func approvalRecordSubtitle(record approvalsSvc.Record) string {
	parts := []string{}
	if target := strings.TrimSpace(record.Target); target != "" {
		parts = append(parts, "target: "+target)
	}
	if decision := strings.TrimSpace(record.Decision); decision != "" {
		parts = append(parts, "decision: "+decision)
	}
	if !record.CreatedAt.IsZero() {
		parts = append(parts, record.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return strings.Join(parts, " - ")
}

func approvalRecordDetails(record approvalsSvc.Record) string {
	lines := []string{}
	if risk := strings.TrimSpace(record.Risk); risk != "" {
		lines = append(lines, "Risk: "+risk)
	}
	if message := strings.TrimSpace(record.Message); message != "" {
		lines = append(lines, "Details: "+message)
	}
	if len(lines) == 0 {
		return "No additional details recorded."
	}
	return strings.Join(lines, "\n")
}

func policyStatusText(policy approvalsSvc.Policy) string {
	if policy.Active(time.Now().UTC()) {
		return fmt.Sprintf("Full project access: active until %s", policy.ExpiresAt.Local().Format("15:04:05"))
	}
	if policy.Message != "" {
		return "Full project access: inactive - " + policy.Message
	}
	return "Full project access: inactive"
}
