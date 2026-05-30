package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type toolPanelWidgets struct {
	problemResults     *fyne.Container
	problemStatus      *widget.Label
	taskResults        *fyne.Container
	taskStatus         *widget.Label
	taskOutput         *widget.Entry
	chatHistoryResults *fyne.Container
	chatHistoryStatus  *widget.Label
	chatHistoryDetail  *widget.Entry
	historyResults     *fyne.Container
	historyStatus      *widget.Label
	historyDetail      *widget.Entry
	agentAuditResults  *fyne.Container
	agentAuditStatus   *widget.Label
	agentAuditDetail   *widget.Entry
	approvalResults    *fyne.Container
	approvalStatus     *widget.Label
	accessStatus       *widget.Label
}

func newToolPanelWidgets() toolPanelWidgets {
	return toolPanelWidgets{
		problemResults:     container.NewVBox(widget.NewLabel("Run a scan to inspect lightweight workspace problems.")),
		problemStatus:      widget.NewLabel("No problem scan yet."),
		taskResults:        container.NewVBox(widget.NewLabel("Discover workspace tasks to run tests, scripts, or Compose checks.")),
		taskStatus:         widget.NewLabel("No tasks discovered."),
		taskOutput:         newReadOnlyMonospaceEntry(fyne.TextWrapOff),
		chatHistoryResults: container.NewVBox(widget.NewLabel("Open a workspace to search persisted chat messages.")),
		chatHistoryStatus:  widget.NewLabel("Chat history has not been loaded."),
		chatHistoryDetail:  newReadOnlyMonospaceEntry(fyne.TextWrapWord),
		historyResults:     container.NewVBox(widget.NewLabel("Open a workspace to inspect unified history.")),
		historyStatus:      widget.NewLabel("History has not been loaded."),
		historyDetail:      newReadOnlyMonospaceEntry(fyne.TextWrapWord),
		agentAuditResults:  container.NewVBox(widget.NewLabel("Open a workspace to inspect persisted agent runs.")),
		agentAuditStatus:   widget.NewLabel("Agent audit has not been loaded."),
		agentAuditDetail:   newReadOnlyMonospaceEntry(fyne.TextWrapWord),
		approvalResults:    container.NewVBox(widget.NewLabel("Open a workspace to inspect approval records.")),
		approvalStatus:     widget.NewLabel("Approval records have not been loaded."),
		accessStatus:       widget.NewLabel("Full project access: inactive"),
	}
}

func newReadOnlyMonospaceEntry(wrap fyne.TextWrap) *widget.Entry {
	entry := widget.NewMultiLineEntry()
	entry.TextStyle = fyne.TextStyle{Monospace: true}
	entry.Wrapping = wrap
	entry.Disable()
	return entry
}
