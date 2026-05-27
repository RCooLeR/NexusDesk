package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	artifactsSvc "nexusdesk/internal/services/artifacts"
	historySvc "nexusdesk/internal/services/history"
)

const historySearchLimit = 100

func (v *View) newHistoryPanel() fyne.CanvasObject {
	query := widget.NewEntry()
	query.SetPlaceHolder("Search chat, artifacts, jobs, and agent runs")
	kind := widget.NewSelect([]string{"All", "Chat", "Artifacts", "Jobs", "Agent"}, func(string) {})
	kind.SetSelected("All")
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		v.refreshHistory(query.Text, historyKindFromLabel(kind.Selected))
	})
	search := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		v.refreshHistory(query.Text, historyKindFromLabel(kind.Selected))
	})
	query.OnSubmitted = func(string) {
		v.refreshHistory(query.Text, historyKindFromLabel(kind.Selected))
	}
	toolbar := container.NewBorder(nil, nil, container.NewHBox(kind, refresh), search, query)
	results := container.NewVScroll(v.historyResults)
	results.SetMinSize(fyne.NewSize(280, 130))
	detail := container.NewVScroll(v.historyDetail)
	detail.SetMinSize(fyne.NewSize(320, 130))
	content := container.NewHSplit(results, detail)
	content.Offset = 0.45
	return container.NewBorder(container.NewVBox(toolbar, v.historyStatus), nil, nil, nil, content)
}

func (v *View) refreshHistory(query string, kind historySvc.Kind) {
	if v.historyStatus == nil || v.historyResults == nil || v.historyDetail == nil {
		return
	}
	if v.metadataStore == nil {
		v.historyStatus.SetText("Open a workspace before inspecting history.")
		v.historyResults.Objects = []fyne.CanvasObject{widget.NewLabel("No workspace metadata store is active.")}
		v.historyDetail.SetText("")
		v.historyResults.Refresh()
		return
	}
	service, err := v.historyService()
	if err != nil {
		v.historyStatus.SetText("History unavailable: " + err.Error())
		dialog.ShowError(err, v.window)
		return
	}
	items, err := service.List(historySvc.Options{Query: query, Kind: kind, Limit: historySearchLimit})
	if err != nil {
		v.historyStatus.SetText("History unavailable: " + err.Error())
		v.historyResults.Objects = []fyne.CanvasObject{widget.NewLabel("Could not read history.")}
		v.historyDetail.SetText("")
		v.historyResults.Refresh()
		return
	}
	v.historyStatus.SetText(historyStatusText(query, kind, len(items)))
	v.historyResults.Objects = historyRows(items, v.previewHistoryItem, v.openHistoryItem)
	if len(items) == 0 {
		v.historyDetail.SetText("")
	} else {
		v.historyDetail.SetText(formatHistoryItem(items[0]))
	}
	v.historyResults.Refresh()
}

func (v *View) previewHistoryItem(item historySvc.Item) {
	v.historyDetail.SetText(formatHistoryItem(item))
}

func (v *View) openHistoryItem(item historySvc.Item) {
	switch item.Kind {
	case historySvc.KindArtifact:
		v.openHistoryArtifact(item.Ref)
	case historySvc.KindChat:
		v.historyDetail.SetText(formatHistoryItem(item))
		v.refreshChatHistory("")
		v.chatHistoryDetail.SetText(strings.TrimSpace(item.Detail))
		v.addActivity("Opened chat history item " + item.Ref + ".")
	case historySvc.KindJob:
		v.historyDetail.SetText(formatHistoryItem(item))
		v.refreshJobs()
		v.addActivity("Opened job history item " + item.Ref + ".")
	case historySvc.KindAgent:
		v.historyDetail.SetText(formatHistoryItem(item))
		v.openHistoryAgentRun(item.Ref)
	default:
		v.historyDetail.SetText(formatHistoryItem(item))
	}
}

func (v *View) openHistoryArtifact(relPath string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	artifacts, err := store.ListArtifacts(artifactsSvc.ListOptions{Query: relPath, IncludeArchived: true})
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	for _, artifact := range artifacts {
		if artifact.RelPath == relPath {
			v.previewArtifact(artifact)
			v.refreshArtifactsWithQuery(relPath)
			return
		}
	}
	v.addActivity("Artifact no longer exists: " + relPath + ".")
}

func (v *View) openHistoryAgentRun(id string) {
	if v.metadataStore == nil {
		return
	}
	runs, err := v.metadataStore.ListAgentRuns(100)
	if err != nil {
		dialog.ShowError(err, v.window)
		return
	}
	for _, run := range runs {
		if run.ID == id {
			v.previewAgentAuditRun(run)
			v.refreshAgentAudit()
			return
		}
	}
	v.addActivity("Agent run no longer exists: " + id + ".")
}

func historyRows(
	items []historySvc.Item,
	onPreview func(historySvc.Item),
	onOpen func(historySvc.Item),
) []fyne.CanvasObject {
	if len(items) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No history records found.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(items))
	for _, item := range items {
		item := item
		preview := widget.NewButtonWithIcon("", theme.VisibilityIcon(), func() {
			onPreview(item)
		})
		preview.Importance = widget.LowImportance
		open := widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() {
			onOpen(item)
		})
		open.Importance = widget.LowImportance
		title := widget.NewLabel(historyRowTitle(item))
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Truncation = fyne.TextTruncateEllipsis
		meta := widget.NewLabel(historyRowMeta(item))
		meta.Truncation = fyne.TextTruncateEllipsis
		summary := widget.NewLabel(item.Summary)
		summary.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, container.NewHBox(preview, open), nil, container.NewVBox(title, meta, summary)))
	}
	return rows
}

func historyKindFromLabel(label string) historySvc.Kind {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "chat":
		return historySvc.KindChat
	case "artifacts":
		return historySvc.KindArtifact
	case "jobs":
		return historySvc.KindJob
	case "agent":
		return historySvc.KindAgent
	default:
		return ""
	}
}

func historyStatusText(query string, kind historySvc.Kind, count int) string {
	parts := []string{"History"}
	if kind != "" {
		parts = append(parts, string(kind))
	}
	if strings.TrimSpace(query) != "" {
		parts = append(parts, fmt.Sprintf("%q", strings.TrimSpace(query)))
	}
	return fmt.Sprintf("%s: %d record(s).", strings.Join(parts, " / "), count)
}

func historyRowTitle(item historySvc.Item) string {
	title := strings.TrimSpace(item.Title)
	if title == "" {
		title = item.Ref
	}
	return strings.ToUpper(string(item.Kind)) + " - " + title
}

func historyRowMeta(item historySvc.Item) string {
	parts := []string{}
	if !item.When.IsZero() {
		parts = append(parts, item.When.Local().Format("2006-01-02 15:04"))
	}
	if item.Ref != "" {
		parts = append(parts, item.Ref)
	}
	if len(item.SourcePaths) > 0 {
		parts = append(parts, fmt.Sprintf("%d source(s)", len(item.SourcePaths)))
	}
	return strings.Join(parts, " | ")
}

func formatHistoryItem(item historySvc.Item) string {
	var builder strings.Builder
	builder.WriteString(historyRowTitle(item))
	if meta := historyRowMeta(item); meta != "" {
		builder.WriteString("\n")
		builder.WriteString(meta)
	}
	if strings.TrimSpace(item.Summary) != "" {
		builder.WriteString("\n\n")
		builder.WriteString(strings.TrimSpace(item.Summary))
	}
	if strings.TrimSpace(item.Detail) != "" {
		builder.WriteString("\n\n---\n\n")
		builder.WriteString(strings.TrimSpace(item.Detail))
	}
	return builder.String()
}
