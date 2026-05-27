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
)

func (v *View) newAssistantPanel() fyne.CanvasObject {
	prompt := widget.NewMultiLineEntry()
	prompt.SetPlaceHolder("Ask Nexus about this workspace")
	prompt.Wrapping = fyne.TextWrapWord
	response := widget.NewRichTextFromMarkdown("Assistant output will stream here.")
	mode := widget.NewSelect([]string{"Ask", "Agent"}, func(string) {})
	mode.SetSelected("Ask")
	send := widget.NewButtonWithIcon("", theme.MailSendIcon(), nil)
	send.OnTapped = func() {
		v.runAssistantRequest(prompt, response, send, mode.Selected)
	}
	composer := container.NewBorder(nil, nil, mode, send, prompt)
	card := widget.NewCard("Assistant", "Local-first context and tool mediation", container.NewBorder(nil, composer, nil, nil, response))
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
	}
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
			v.addActivity("Assistant response completed with " + result.Model + ".")
		})
	}()
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
		ApproveShell:  false,
	}
	send.Disable()
	response.ParseMarkdown("Agent starting...")
	v.addActivity("Agent request started.")
	go func() {
		tail := agentActivityTail{}
		result, err := v.agentService.Run(context.Background(), request, func(event agentSvc.Event) {
			line := agentEventLine(event)
			if line == "" {
				return
			}
			tail.Add(line)
			current := tail.Markdown()
			fyne.Do(func() {
				response.ParseMarkdown(current)
				v.addActivity(line)
			})
		})
		fyne.Do(func() {
			defer send.Enable()
			if err != nil {
				message := "Agent request failed: " + err.Error()
				response.ParseMarkdown(message)
				v.addActivity(message)
				return
			}
			response.ParseMarkdown(agentFinalMarkdown(result))
			v.addActivity(fmt.Sprintf("Agent response completed after %d iteration(s).", result.Iterations))
		})
	}()
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
		return "Tool requested: " + event.ToolName
	case "tool_done":
		return "Tool completed: " + event.ToolName
	case "tool_error":
		return "Tool failed: " + firstNonEmpty(event.ToolName, event.Error)
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
