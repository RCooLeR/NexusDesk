package shell

import (
	"context"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

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
		v.runAssistantRequest(prompt, response, send)
	}
	composer := container.NewBorder(nil, nil, mode, send, prompt)
	card := widget.NewCard("Assistant", "Local-first context and tool mediation", container.NewBorder(nil, composer, nil, nil, response))
	return container.NewPadded(card)
}

func (v *View) runAssistantRequest(prompt *widget.Entry, response *widget.RichText, send *widget.Button) {
	text := strings.TrimSpace(prompt.Text)
	if text == "" {
		v.addActivity("Assistant prompt is empty.")
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
