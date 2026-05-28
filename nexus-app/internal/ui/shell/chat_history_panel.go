package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	metadataSvc "nexusdesk/internal/services/metadata"
)

const chatHistorySearchLimit = 80

func (v *View) newChatHistoryPanel() fyne.CanvasObject {
	query := widget.NewEntry()
	query.SetPlaceHolder("Search chat history")
	query.OnSubmitted = v.refreshChatHistory
	refresh := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		v.refreshChatHistory(query.Text)
	})
	search := widget.NewButtonWithIcon("", theme.SearchIcon(), func() {
		v.refreshChatHistory(query.Text)
	})
	toolbar := container.NewBorder(nil, nil, refresh, search, query)
	results := container.NewVScroll(v.chatHistoryResults)
	results.SetMinSize(fyne.NewSize(260, 130))
	detail := container.NewVScroll(v.chatHistoryDetail)
	detail.SetMinSize(fyne.NewSize(320, 130))
	content := container.NewHSplit(results, detail)
	content.Offset = 0.42
	return container.NewBorder(container.NewVBox(toolbar, v.chatHistoryStatus), nil, nil, nil, content)
}

func (v *View) refreshChatHistory(query string) {
	if v.chatHistoryStatus == nil || v.chatHistoryResults == nil || v.chatHistoryDetail == nil {
		return
	}
	if v.metadataStore == nil {
		v.chatHistoryStatus.SetText("Open a workspace before searching chat history.")
		v.chatHistoryResults.Objects = []fyne.CanvasObject{widget.NewLabel("No workspace metadata store is active.")}
		v.chatHistoryDetail.SetText("")
		v.chatHistoryResults.Refresh()
		return
	}
	records, err := v.metadataStore.SearchChatMessages(query, chatHistorySearchLimit)
	if err != nil {
		v.chatHistoryStatus.SetText("Chat history unavailable: " + err.Error())
		v.chatHistoryResults.Objects = []fyne.CanvasObject{widget.NewLabel("Could not read chat history.")}
		v.chatHistoryDetail.SetText("")
		v.chatHistoryResults.Refresh()
		return
	}
	v.chatHistoryStatus.SetText(chatHistoryStatusText(query, len(records)))
	freshness := chatHistoryFreshness(v.metadataStore.Root(), records)
	v.chatHistoryResults.Objects = chatHistoryRows(records, freshness, v.openChatHistoryRecord, v.useChatHistoryRecordForAssistant)
	if len(records) == 0 {
		v.chatHistoryDetail.SetText("")
	} else {
		v.chatHistoryDetail.SetText(formatChatHistoryRecord(records[0], freshness[records[0].ID]))
	}
	v.chatHistoryResults.Refresh()
}

func (v *View) openChatHistoryRecord(record metadataSvc.ChatMessageRecord) {
	if v.chatHistoryDetail == nil {
		return
	}
	stale := []string{}
	if v.metadataStore != nil {
		stale = chatHistoryFreshness(v.metadataStore.Root(), []metadataSvc.ChatMessageRecord{record})[record.ID]
	}
	v.chatHistoryDetail.SetText(formatChatHistoryRecord(record, stale))
}

func (v *View) useChatHistoryRecordForAssistant(record metadataSvc.ChatMessageRecord) {
	if v.assistantPrompt == nil {
		v.addActivity("Assistant composer is not ready yet.")
		return
	}
	v.assistantPrompt.SetText(chatHistorySeedPrompt(record))
	if v.assistantMode != nil {
		v.assistantMode.SetSelected("Agent")
	}
	pinned := 0
	for _, source := range record.SourcePaths {
		if v.state.AddAssistantContextPath(source) {
			pinned++
		}
	}
	v.refreshAssistantContextPins()
	if pinned > 0 {
		v.addActivity(fmt.Sprintf("Seeded Agent prompt from chat history and pinned %d source(s).", pinned))
		return
	}
	v.addActivity("Seeded Agent prompt from chat history.")
}

func chatHistoryRows(
	records []metadataSvc.ChatMessageRecord,
	freshness map[string][]string,
	onOpen func(metadataSvc.ChatMessageRecord),
	onUse func(metadataSvc.ChatMessageRecord),
) []fyne.CanvasObject {
	if len(records) == 0 {
		return []fyne.CanvasObject{widget.NewLabel("No chat messages found.")}
	}
	rows := make([]fyne.CanvasObject, 0, len(records))
	for _, record := range records {
		item := record
		title := widget.NewLabel(chatHistoryRowTitle(item))
		title.TextStyle = fyne.TextStyle{Bold: true}
		title.Truncation = fyne.TextTruncateEllipsis
		preview := widget.NewLabel(compactChatHistoryContent(item.Content, 120))
		preview.Truncation = fyne.TextTruncateEllipsis
		meta := widget.NewLabel(chatHistoryRowMeta(item, freshness[item.ID]))
		meta.Truncation = fyne.TextTruncateEllipsis
		open := widget.NewButtonWithIcon("", theme.VisibilityIcon(), func() {
			onOpen(item)
		})
		open.Importance = widget.LowImportance
		use := widget.NewButtonWithIcon("", theme.MailForwardIcon(), func() {
			onUse(item)
		})
		use.Importance = widget.LowImportance
		rows = append(rows, container.NewBorder(nil, nil, container.NewHBox(open, use), nil, container.NewVBox(title, meta, preview)))
	}
	return rows
}

func chatHistoryStatusText(query string, count int) string {
	query = strings.TrimSpace(query)
	if query == "" {
		return fmt.Sprintf("Chat history: %d recent message(s).", count)
	}
	return fmt.Sprintf("Chat history: %d result(s) for %q.", count, query)
}

func chatHistoryRowTitle(record metadataSvc.ChatMessageRecord) string {
	role := strings.TrimSpace(record.Role)
	if role == "" {
		role = "message"
	}
	return strings.ToUpper(role[:1]) + role[1:]
}

func chatHistoryRowMeta(record metadataSvc.ChatMessageRecord, staleSources []string) string {
	parts := []string{}
	if !record.CreatedAt.IsZero() {
		parts = append(parts, record.CreatedAt.Local().Format("2006-01-02 15:04"))
	}
	if strings.TrimSpace(record.Model) != "" {
		parts = append(parts, record.Model)
	}
	if len(record.SourcePaths) > 0 {
		parts = append(parts, fmt.Sprintf("%d source(s)", len(record.SourcePaths)))
	}
	if len(staleSources) > 0 {
		parts = append(parts, fmt.Sprintf("%d stale source(s)", len(staleSources)))
	}
	return strings.Join(parts, " | ")
}

func compactChatHistoryContent(content string, limit int) string {
	content = strings.Join(strings.Fields(content), " ")
	if limit <= 0 || len(content) <= limit {
		return content
	}
	if limit <= 3 {
		return content[:limit]
	}
	return content[:limit-3] + "..."
}

func formatChatHistoryRecord(record metadataSvc.ChatMessageRecord, staleSources []string) string {
	var builder strings.Builder
	builder.WriteString(chatHistoryRowTitle(record))
	if meta := chatHistoryRowMeta(record, staleSources); meta != "" {
		builder.WriteString("\n")
		builder.WriteString(meta)
	}
	if strings.TrimSpace(record.ContextRelPath) != "" {
		builder.WriteString("\nContext: ")
		builder.WriteString(record.ContextRelPath)
	}
	if len(record.SourcePaths) > 0 {
		builder.WriteString("\nSources: ")
		builder.WriteString(strings.Join(record.SourcePaths, ", "))
	}
	if len(staleSources) > 0 {
		builder.WriteString("\nWarning: context changed since this answer was created: ")
		builder.WriteString(strings.Join(staleSources, ", "))
	}
	builder.WriteString("\n\n")
	builder.WriteString(strings.TrimSpace(record.Content))
	return builder.String()
}

func chatHistoryFreshness(root string, records []metadataSvc.ChatMessageRecord) map[string][]string {
	out := map[string][]string{}
	root = strings.TrimSpace(root)
	if root == "" {
		return out
	}
	for _, record := range records {
		for _, source := range chatHistorySourceCandidates(record) {
			if chatSourceStale(root, source, record.CreatedAt) {
				out[record.ID] = append(out[record.ID], source)
			}
		}
	}
	return out
}

func chatHistorySourceCandidates(record metadataSvc.ChatMessageRecord) []string {
	seen := map[string]bool{}
	sources := []string{}
	candidates := append([]string{}, record.SourcePaths...)
	context := strings.TrimSpace(record.ContextRelPath)
	if context != "" && !strings.Contains(context, ":") {
		candidates = append(candidates, context)
	}
	for _, source := range candidates {
		source = filepath.ToSlash(strings.TrimSpace(source))
		if source == "" || source == "." || seen[source] {
			continue
		}
		seen[source] = true
		sources = append(sources, source)
	}
	return sources
}

func chatSourceStale(root string, source string, createdAt time.Time) bool {
	absPath, ok := safeChatSourcePath(root, source)
	if !ok {
		return false
	}
	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		return false
	}
	return !createdAt.IsZero() && info.ModTime().UTC().After(createdAt.UTC())
}

func safeChatSourcePath(root string, relPath string) (string, bool) {
	relPath = filepath.ToSlash(strings.Trim(strings.TrimSpace(relPath), `"'`))
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" || relPath == "." || relPath == ".." || strings.HasPrefix(relPath, "../") || strings.Contains(relPath, "/../") || filepath.IsAbs(relPath) {
		return "", false
	}
	target := filepath.Join(root, filepath.FromSlash(relPath))
	relToRoot, err := filepath.Rel(root, target)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", false
	}
	return target, true
}

func chatHistorySeedPrompt(record metadataSvc.ChatMessageRecord) string {
	role := strings.ToLower(strings.TrimSpace(record.Role))
	if role == "" {
		role = "message"
	}
	content := compactChatHistoryContent(record.Content, 4000)
	var builder strings.Builder
	builder.WriteString("Use this prior ")
	builder.WriteString(role)
	builder.WriteString(" message as context and continue from it.\n\n")
	if len(record.SourcePaths) > 0 {
		builder.WriteString("Original source paths are pinned in the assistant context.\n\n")
	}
	builder.WriteString("Prior message:\n")
	builder.WriteString(content)
	builder.WriteString("\n\nNext task:\n")
	return builder.String()
}
