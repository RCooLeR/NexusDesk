package shell

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	metadataSvc "nexusdesk/internal/services/metadata"
)

type agentAuditController struct {
	view *View
}

func newAgentAuditController(view *View) *agentAuditController {
	return &agentAuditController{view: view}
}

func (v *View) newAgentAuditPanel() fyne.CanvasObject {
	if v.agentAudit == nil {
		v.agentAudit = newAgentAuditController(v)
	}
	return v.agentAudit.NewPanel()
}

func (c *agentAuditController) NewPanel() fyne.CanvasObject {
	v := c.view
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), c.Refresh)
	header := container.NewBorder(nil, nil, v.agentAuditStatus, refresh)
	listScroll := container.NewScroll(v.agentAuditResults)
	listScroll.SetMinSize(fyne.NewSize(260, 110))
	detail := container.NewBorder(widget.NewLabel("Agent run and tool audit detail"), nil, nil, nil, v.agentAuditDetail)
	split := container.NewVSplit(listScroll, detail)
	split.Offset = 0.48
	return container.NewBorder(header, nil, nil, nil, split)
}

func (v *View) refreshAgentAudit() {
	if v.agentAudit == nil {
		v.agentAudit = newAgentAuditController(v)
	}
	v.agentAudit.Refresh()
}

func (v *View) previewAgentAuditRun(run metadataSvc.AgentRunRecord) {
	if v.agentAudit == nil {
		v.agentAudit = newAgentAuditController(v)
	}
	v.agentAudit.previewRun(run)
}

func (c *agentAuditController) Refresh() {
	v := c.view
	if v.metadataStore == nil {
		v.agentAuditStatus.SetText("Open a workspace before inspecting agent audit history.")
		v.agentAuditResults.Objects = []fyne.CanvasObject{widget.NewLabel("No workspace metadata store is active.")}
		v.agentAuditResults.Refresh()
		v.agentAuditDetail.SetText("")
		return
	}
	runs, err := v.metadataStore.ListAgentRuns(50)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.agentAuditStatus.SetText(fmt.Sprintf("%d persisted agent run(s)", len(runs)))
	v.agentAuditResults.Objects = agentAuditRows(runs, c.previewRun)
	v.agentAuditResults.Refresh()
}

func (c *agentAuditController) previewRun(run metadataSvc.AgentRunRecord) {
	v := c.view
	if v.metadataStore == nil {
		return
	}
	tools, err := v.metadataStore.ListToolRuns(run.ID)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	v.agentAuditDetail.SetText(formatAgentAuditDetail(run, tools))
	v.agentAuditStatus.SetText(fmt.Sprintf("Previewing %s with %d tool run(s)", run.ID, len(tools)))
}

func agentAuditRows(runs []metadataSvc.AgentRunRecord, onPreview func(metadataSvc.AgentRunRecord)) []fyne.CanvasObject {
	if len(runs) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No persisted agent runs yet. Use Agent mode to create one.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(runs))
	for _, run := range runs {
		run := run
		preview := widget.NewButtonWithIcon("", theme.VisibilityIcon(), func() {
			onPreview(run)
		})
		preview.Importance = widget.LowImportance
		title := widget.NewLabel(agentAuditTitle(run))
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Truncation = fyne.TextTruncateEllipsis
		meta := widget.NewLabel(agentAuditMeta(run))
		meta.Truncation = fyne.TextTruncateEllipsis
		message := widget.NewLabel(compactAgentAuditMessage(run.Message))
		message.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, preview, nil, container.NewVBox(title, meta, message)))
	}
	return rows
}
