package shell

import (
	"context"
	"strings"

	"fyne.io/fyne/v2"

	assistantSvc "nexusdesk/internal/services/assistant"
	gitSvc "nexusdesk/internal/services/git"
)

const maxGitAIDiffChars = 48000

func (v *View) summarizeSelectedGitDiff() {
	v.runGitAI("AI diff summary", gitSummaryPrompt(v.gitLastDiff))
}

func (v *View) draftSelectedGitCommitMessage() {
	v.runGitAI("AI commit draft", gitCommitDraftPrompt(v.gitLastDiff))
}

func (v *View) runGitAI(label string, prompt string) {
	if !gitDiffHasContent(v.gitLastDiff) {
		v.addActivity("Select a changed file before asking Nexus to review a diff.")
		return
	}
	v.gitDiffStatus.SetText(label + " running...")
	v.gitDiffText.SetText(label + " running through the native assistant service...")
	v.addActivity(label + " started for " + v.gitLastDiff.Path + ".")

	go func() {
		var builder strings.Builder
		result, err := v.assistantService.AskStream(context.Background(), assistantSvc.Request{
			Prompt:        prompt,
			WorkspaceRoot: v.state.Workspace().Root,
		}, func(delta string) error {
			builder.WriteString(delta)
			current := builder.String()
			fyne.Do(func() {
				v.gitDiffText.SetText(current)
			})
			return nil
		})
		fyne.Do(func() {
			if err != nil {
				message := label + " failed: " + err.Error()
				v.gitDiffStatus.SetText(message)
				v.gitDiffText.SetText(message)
				v.addActivity(message)
				return
			}
			v.gitDiffStatus.SetText(label + " completed with " + result.Model + ".")
			v.gitDiffText.SetText(result.Message)
			v.addActivity(label + " completed for " + v.gitLastDiff.Path + ".")
		})
	}()
}

func gitSummaryPrompt(diff gitSvc.FileDiff) string {
	return "Review this selected Git diff for Nexus Aegrail Studio. Summarize the behavior change, risk, test/docs impact, and any follow-up needed. Use concise bullets and do not invent files.\n\n" + gitAIDiffBlock(diff)
}

func gitCommitDraftPrompt(diff gitSvc.FileDiff) string {
	return "Draft a commit message for this selected Git diff. Return a concise subject line, a blank line, and optional bullet details. Prefer conventional-commit style when a clear type/scope is obvious.\n\n" + gitAIDiffBlock(diff)
}

func gitAIDiffBlock(diff gitSvc.FileDiff) string {
	payload := formatGitDiff(diff, gitDiffModeUnified)
	truncated := false
	if len(payload) > maxGitAIDiffChars {
		payload = payload[:maxGitAIDiffChars]
		truncated = true
	}
	var builder strings.Builder
	builder.WriteString("Path: ")
	builder.WriteString(diff.Path)
	builder.WriteString("\n\n```diff\n")
	builder.WriteString(payload)
	if !strings.HasSuffix(payload, "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString("```\n")
	if truncated {
		builder.WriteString("\nThe diff text was truncated before being sent to the model.\n")
	}
	return builder.String()
}

func gitDiffHasContent(diff gitSvc.FileDiff) bool {
	return strings.TrimSpace(diff.StagedDiff) != "" || strings.TrimSpace(diff.UnstagedDiff) != ""
}
