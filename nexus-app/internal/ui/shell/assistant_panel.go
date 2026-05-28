package shell

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	agentSvc "nexusdesk/internal/services/agent"
	assistantSvc "nexusdesk/internal/services/assistant"
	jobsSvc "nexusdesk/internal/services/jobs"
	llmSvc "nexusdesk/internal/services/llm"
	metadataSvc "nexusdesk/internal/services/metadata"
	settingsSvc "nexusdesk/internal/services/settings"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

const assistantConversationLimit = 24
const assistantHistoryPreviewLimit = 6
const defaultAgentContextMaxBytes = 96 * 1024

func (v *View) newAssistantPanel() fyne.CanvasObject {
	prompt := widget.NewMultiLineEntry()
	prompt.SetPlaceHolder("Ask Nexus about this workspace")
	prompt.Wrapping = fyne.TextWrapWord
	prompt.SetMinRowsVisible(3)
	v.assistantPrompt = prompt
	response := widget.NewRichTextFromMarkdown("Assistant output will stream here.")
	v.assistantContextStatus = widget.NewLabel("")
	v.assistantContextStatus.Wrapping = fyne.TextWrapWord
	v.assistantContextList = container.NewVBox()
	v.assistantHistoryStatus = widget.NewLabel("")
	v.assistantHistoryStatus.Wrapping = fyne.TextWrapWord
	v.assistantHistoryList = container.NewVBox()
	pinSelection := widget.NewButton("Pin selection", v.pinSelectedAssistantContext)
	pinProject := widget.NewButton("Pin project", func() {
		v.pinAssistantContextPath(".")
	})
	clearPins := widget.NewButton("Clear", v.clearAssistantContextPins)
	contextBar := container.NewVBox(
		container.NewHBox(pinSelection, pinProject, clearPins),
		v.assistantContextStatus,
		v.assistantContextList,
	)
	historyBar := container.NewVBox(
		v.assistantHistoryStatus,
		v.assistantHistoryList,
	)
	mode := widget.NewSelect([]string{"Ask", "Agent"}, func(string) {})
	mode.SetSelected("Ask")
	v.assistantMode = mode
	agentTaskApproval := widget.NewCheck("Allow task tool this run", nil)
	v.assistantRunTaskApproval = agentTaskApproval
	send := widget.NewButtonWithIcon("", theme.MailSendIcon(), nil)
	send.OnTapped = func() {
		v.runAssistantRequest(prompt, response, send, mode.Selected)
	}
	composer := container.NewBorder(nil, nil, container.NewVBox(mode, agentTaskApproval), send, prompt)
	composer = container.NewPadded(composer)
	sidebar := container.NewVBox(contextBar, widget.NewSeparator(), historyBar)
	card := widget.NewCard("Assistant", "Local-first context and tool mediation", container.NewBorder(sidebar, composer, nil, nil, response))
	v.refreshAssistantContextPins()
	v.refreshAssistantHistory()
	return container.NewPadded(card)
}

func (v *View) runAssistantRequest(prompt *widget.Entry, response *widget.RichText, send *widget.Button, mode string) {
	text := strings.TrimSpace(prompt.Text)
	if text == "" {
		v.addActivity("Assistant prompt is empty.")
		return
	}
	if strings.EqualFold(strings.TrimSpace(mode), "Agent") {
		v.runAgentRequest(text, response, send)
		return
	}
	workspace := v.state.Workspace()
	request := assistantSvc.Request{
		Prompt:        text,
		WorkspaceRoot: workspace.Root,
		SelectedPath:  v.state.SelectedPath(),
		ContextPaths:  assistantContextPathsForRequest(v.state.AssistantContextPaths(), ""),
		Conversation:  v.state.AssistantConversation(),
	}
	startedAt := time.Now().UTC()
	send.Disable()
	response.ParseMarkdown("Receiving response...")
	v.addActivity("Assistant request started.")

	go func() {
		var builder strings.Builder
		result, err := v.assistantService.AskStream(context.Background(), request, func(delta string) error {
			builder.WriteString(delta)
			current := builder.String()
			fyne.Do(func() {
				response.ParseMarkdown(current)
			})
			return nil
		})
		fyne.Do(func() {
			defer send.Enable()
			if err != nil {
				response.ParseMarkdown("Assistant request failed: " + err.Error())
				v.addActivity("Assistant request failed: " + err.Error())
				return
			}
			response.ParseMarkdown(result.Message)
			if result.ContextWarning != "" {
				v.addActivity(result.ContextWarning)
			}
			v.persistAssistantExchange(text, result, startedAt)
			v.addActivity("Assistant response completed with " + result.Model + ".")
		})
	}()
}

func (v *View) loadAssistantChatHistory() {
	if v.metadataStore == nil {
		v.state.SetAssistantConversation(nil)
		v.refreshAssistantHistory()
		return
	}
	records, err := v.metadataStore.ListChatMessages(assistantConversationLimit)
	if err != nil {
		v.state.SetAssistantConversation(nil)
		v.refreshAssistantHistory()
		v.addActivity("Assistant chat history unavailable: " + err.Error())
		return
	}
	v.state.SetAssistantConversation(chatTurnsFromMetadata(records))
	v.refreshAssistantHistory()
	if len(records) > 0 {
		v.addActivity(fmt.Sprintf("Loaded %d assistant chat message(s).", len(records)))
	}
}

func (v *View) persistAssistantExchange(prompt string, result assistantSvc.Result, startedAt time.Time) {
	if v.metadataStore == nil {
		return
	}
	if err := v.metadataStore.SaveChatMessage(metadataSvc.ChatMessageRecord{
		Role:      "user",
		Content:   prompt,
		CreatedAt: startedAt,
	}); err != nil {
		v.addActivity("Could not persist user chat message: " + err.Error())
		return
	}
	if err := v.metadataStore.SaveChatMessage(metadataSvc.ChatMessageRecord{
		Role:        "assistant",
		Content:     result.Message,
		Model:       result.Model,
		SourcePaths: result.SourcePaths,
		CreatedAt:   time.Now().UTC(),
	}); err != nil {
		v.addActivity("Could not persist assistant chat message: " + err.Error())
		return
	}
	v.state.AppendAssistantExchange(prompt, result.Message)
	v.refreshAssistantHistory()
}

func (v *View) refreshAssistantHistory() {
	if v.assistantHistoryStatus == nil || v.assistantHistoryList == nil {
		return
	}
	turns := v.state.AssistantConversation()
	v.assistantHistoryList.Objects = nil
	if len(turns) == 0 {
		v.assistantHistoryStatus.SetText("Chat history: no persisted workspace turns yet.")
		v.assistantHistoryList.Add(widget.NewLabel("Ask a question to start history."))
		v.assistantHistoryList.Refresh()
		return
	}
	v.assistantHistoryStatus.SetText(fmt.Sprintf("Chat history: %d recent persisted turn(s).", len(turns)))
	start := len(turns) - assistantHistoryPreviewLimit
	if start < 0 {
		start = 0
	}
	for _, turn := range turns[start:] {
		label := widget.NewLabel(chatTurnPreview(turn))
		label.Truncation = fyne.TextTruncateEllipsis
		v.assistantHistoryList.Add(label)
	}
	v.assistantHistoryList.Refresh()
}

func chatTurnsFromMetadata(records []metadataSvc.ChatMessageRecord) []llmSvc.ChatTurn {
	turns := make([]llmSvc.ChatTurn, 0, len(records))
	for _, record := range records {
		role := strings.ToLower(strings.TrimSpace(record.Role))
		content := strings.TrimSpace(record.Content)
		if content == "" || (role != "user" && role != "assistant") {
			continue
		}
		turns = append(turns, llmSvc.ChatTurn{Role: role, Content: content})
	}
	return turns
}

func chatTurnPreview(turn llmSvc.ChatTurn) string {
	role := strings.ToLower(strings.TrimSpace(turn.Role))
	if role == "" {
		role = "turn"
	}
	content := strings.Join(strings.Fields(turn.Content), " ")
	if content == "" {
		content = "(empty)"
	}
	if len(content) > 90 {
		content = content[:87] + "..."
	}
	return strings.ToUpper(role[:1]) + role[1:] + ": " + content
}

func (v *View) pinSelectedAssistantContext() {
	selected := selectedPathOrEmpty(v)
	if selected == "" {
		v.addActivity("Select a file or folder before pinning assistant context.")
		return
	}
	v.pinAssistantContextPath(selected)
}

func (v *View) pinAssistantContextPath(relPath string) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before pinning assistant context.")
		return
	}
	if v.state.AddAssistantContextPath(relPath) {
		v.addActivity("Pinned assistant context " + relPath + ".")
	} else {
		v.addActivity("Assistant context already includes " + relPath + ".")
	}
	v.refreshAssistantContextPins()
}

func (v *View) removeAssistantContextPin(relPath string) {
	if v.state.RemoveAssistantContextPath(relPath) {
		v.addActivity("Removed assistant context " + relPath + ".")
	}
	v.refreshAssistantContextPins()
}

func (v *View) clearAssistantContextPins() {
	if len(v.state.AssistantContextPaths()) == 0 {
		v.addActivity("No assistant context pins to clear.")
		return
	}
	v.state.ClearAssistantContextPaths()
	v.addActivity("Cleared assistant context pins.")
	v.refreshAssistantContextPins()
}

func (v *View) refreshAssistantContextPins() {
	if v.assistantContextStatus == nil || v.assistantContextList == nil {
		return
	}
	paths := v.state.AssistantContextPaths()
	v.assistantContextList.Objects = nil
	if len(paths) == 0 {
		selected := selectedPathOrEmpty(v)
		if selected == "" {
			v.assistantContextStatus.SetText("Context: pin files, folders, or the project root before sending.")
		} else {
			v.assistantContextStatus.SetText("Context: selected item will be used unless pins are added: " + selected)
		}
		v.assistantContextList.Add(widget.NewLabel("No pinned context."))
		v.assistantContextList.Refresh()
		return
	}
	v.assistantContextStatus.SetText(fmt.Sprintf("Context pack: %d pinned root(s).", len(paths)))
	for _, relPath := range paths {
		pinnedPath := relPath
		label := widget.NewLabel(pinnedPath)
		label.Truncation = fyne.TextTruncateEllipsis
		remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			v.removeAssistantContextPin(pinnedPath)
		})
		v.assistantContextList.Add(container.NewBorder(nil, nil, nil, remove, label))
	}
	v.assistantContextList.Refresh()
}

func (v *View) runAgentRequest(text string, response *widget.RichText, send *widget.Button) {
	workspace := v.state.Workspace()
	if workspace.Root == "" {
		v.addActivity("Open a workspace before running Agent mode.")
		response.ParseMarkdown("Open a workspace before running Agent mode.")
		return
	}
	request := agentSvc.Request{
		ID:            fmt.Sprintf("agent-%d", time.Now().UTC().UnixNano()),
		Prompt:        text,
		WorkspaceRoot: workspace.Root,
		ApproveWrites: v.approvalService.HasFullProjectAccess(workspace.Root),
		ApproveShell:  v.assistantRunTaskApprovalChecked(),
	}
	v.attachAgentContext(&request)
	job, ctx := v.jobService.Start("agent", agentJobLabel(text))
	v.jobService.AppendLog(job.ID, "Prompt: "+agentJobLabel(text))
	send.Disable()
	response.ParseMarkdown("Agent starting...")
	v.addActivity("Agent request started as " + job.ID + ".")
	v.refreshJobs()
	go func() {
		tail := agentActivityTail{}
		result, err := v.agentService.Run(ctx, request, func(event agentSvc.Event) {
			line := agentEventLine(event)
			if line == "" {
				return
			}
			tail.Add(line)
			current := tail.Markdown()
			fyne.Do(func() {
				response.ParseMarkdown(current)
				v.addActivity(line)
				v.jobService.AppendLog(job.ID, line)
				v.refreshJobs()
			})
		})
		fyne.Do(func() {
			defer send.Enable()
			if v.assistantRunTaskApproval != nil && !v.approvalService.HasFullProjectAccess(workspace.Root) {
				v.assistantRunTaskApproval.SetChecked(false)
			}
			if err != nil {
				message := "Agent request failed: " + err.Error()
				response.ParseMarkdown(message)
				v.addActivity(message)
				v.jobService.Finish(job.ID, jobsSvc.StatusFailed, message, err)
				v.persistAgentRun(job.ID, request, result, "failed", message, job.StartedAt)
				v.refreshJobs()
				return
			}
			response.ParseMarkdown(agentFinalMarkdown(result))
			message := fmt.Sprintf("Agent response completed after %d iteration(s).", result.Iterations)
			v.addActivity(message)
			v.jobService.Finish(job.ID, jobsSvc.StatusSuccess, message, nil)
			v.persistAgentRun(job.ID, request, result, "success", result.Message, job.StartedAt)
			v.refreshJobs()
		})
	}()
}

func (v *View) assistantRunTaskApprovalChecked() bool {
	if v.assistantRunTaskApproval == nil {
		return false
	}
	return v.assistantRunTaskApproval.Checked || v.approvalService.HasFullProjectAccess(v.state.Workspace().Root)
}

func (v *View) attachAgentContext(request *agentSvc.Request) {
	contextPaths := assistantContextPathsForRequest(v.state.AssistantContextPaths(), v.state.SelectedPath())
	if strings.TrimSpace(request.WorkspaceRoot) == "" || len(contextPaths) == 0 {
		return
	}
	pack, err := v.workspaceService.BuildContextPack(request.WorkspaceRoot, contextPaths, workspaceSvc.ContextPackOptions{
		MaxBytes: agentContextBudgetBytes(v.settingsStore),
	})
	if err != nil {
		v.addActivity("Agent context was not included: " + err.Error())
		return
	}
	request.ContextRelPath = pack.Label
	request.ContextContent = pack.Content
	request.SourcePaths = append([]string{}, pack.SourcePaths...)
	v.addActivity("Attached agent context " + pack.Label + ".")
	if pack.Truncated {
		v.addActivity("Agent context pack was capped to fit the model budget.")
	}
}

func assistantContextPathsForRequest(pinned []string, selected string) []string {
	seen := map[string]bool{}
	paths := make([]string, 0, len(pinned)+1)
	for _, relPath := range pinned {
		relPath = strings.TrimSpace(relPath)
		if relPath == "" || seen[relPath] {
			continue
		}
		seen[relPath] = true
		paths = append(paths, relPath)
	}
	selected = strings.TrimSpace(selected)
	if len(paths) == 0 && selected != "" {
		paths = append(paths, selected)
	}
	return paths
}

func agentContextBudgetBytes(store interface {
	Load() (settingsSvc.Settings, error)
}) int {
	if store == nil {
		return defaultAgentContextMaxBytes
	}
	settings, err := store.Load()
	if err != nil {
		return defaultAgentContextMaxBytes
	}
	config := llmSvc.ConfigFromSettings(settings)
	budgetTokens := config.ContextTokens - config.ResponseReserveTokens
	if budgetTokens <= 0 {
		return defaultAgentContextMaxBytes / 4
	}
	return budgetTokens * 4
}

func agentJobLabel(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "Agent run"
	}
	prompt = strings.Join(strings.Fields(prompt), " ")
	if len(prompt) > 80 {
		return prompt[:77] + "..."
	}
	return prompt
}

type agentActivityTail struct {
	items []string
}

func (t *agentActivityTail) Add(message string) {
	message = strings.TrimSpace(message)
	if message == "" {
		return
	}
	t.items = append(t.items, message)
	if len(t.items) > 2 {
		t.items = t.items[len(t.items)-2:]
	}
}

func (t agentActivityTail) Markdown() string {
	if len(t.items) == 0 {
		return "Agent starting..."
	}
	return strings.Join(t.items, "\n\n")
}

func agentEventLine(event agentSvc.Event) string {
	switch event.Type {
	case "start":
		return "Agent started."
	case "model_request":
		return fmt.Sprintf("Thinking, step %d...", event.Iteration)
	case "tool_start":
		return "Tool requested: " + firstNonEmpty(event.ToolName, "unknown tool")
	case "tool_done":
		return "Tool completed: " + firstNonEmpty(event.ToolName, "unknown tool")
	case "tool_error":
		return "Tool failed: " + firstNonEmpty(event.ToolName, event.Error, "unknown tool")
	case "plan_update":
		return "Plan updated."
	case "finalizing":
		return "Wrapping up agent run..."
	case "stopped", "error":
		return firstNonEmpty(event.Message, event.Error)
	default:
		return ""
	}
}

func agentFinalMarkdown(result agentSvc.Result) string {
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = "Agent completed without a final message."
	}
	if result.StopReason != "" {
		message += "\n\nStop reason: `" + result.StopReason + "`"
	}
	return message
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
