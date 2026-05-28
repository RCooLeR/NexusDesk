package shell

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	agentSvc "nexusdesk/internal/services/agent"
	approvalsSvc "nexusdesk/internal/services/approvals"
	artifactsSvc "nexusdesk/internal/services/artifacts"
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
const assistantCitationSnippetLimit = 8
const assistantCitationSnippetLineLimit = 4
const assistantCitationSnippetLineMaxChars = 180

var assistantCitationPattern = regexp.MustCompile(`(?i)([\w./\\-]+\.[A-Za-z0-9]{1,12})(?:(?:#L|:)(\d+)(?:[-:L]+(\d+))?)`)
var assistantCitationRefPattern = regexp.MustCompile(`^(.+):L(\d+)(?:-L(\d+))?$`)

type assistantCitationPreviewer interface {
	PreviewFile(root string, relPath string) (domain.FilePreview, error)
}

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
	profileSelect := widget.NewSelect(nil, func(profileID string) {
		v.updateAssistantProfileSelection(profileID)
	})
	v.assistantProfileSelect = profileSelect
	memory := widget.NewMultiLineEntry()
	memory.SetPlaceHolder("Assistant memory and preferences")
	memory.SetMinRowsVisible(2)
	v.assistantMemory = memory
	saveProfile := widget.NewButtonWithIcon("Save memory", theme.DocumentSaveIcon(), v.saveAssistantProfile)
	profileBar := container.NewVBox(
		widget.NewLabel("Prompt profile"),
		profileSelect,
		memory,
		saveProfile,
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
	retry := widget.NewButton("Retry", func() {
		v.retryLatestAssistantAnswer(prompt, response, send)
	})
	compare := widget.NewButton("Compare", func() {
		v.compareLatestAssistantAnswer(prompt, response, send)
	})
	saveAnswer := widget.NewButtonWithIcon("Save answer", theme.DocumentSaveIcon(), func() {
		v.saveLatestAssistantAnswer()
	})
	assistantActions := container.NewHBox(retry, compare, saveAnswer)
	composer := container.NewBorder(assistantActions, nil, container.NewVBox(mode, agentTaskApproval), send, prompt)
	composer = container.NewPadded(composer)
	sidebar := container.NewVBox(profileBar, widget.NewSeparator(), contextBar, widget.NewSeparator(), historyBar)
	card := widget.NewCard("Assistant", "Local-first context and tool mediation", container.NewBorder(sidebar, composer, nil, nil, response))
	v.loadAssistantProfile()
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
			response.ParseMarkdown(assistantResponseMarkdown(result))
			if result.ContextWarning != "" {
				v.addActivity(result.ContextWarning)
			}
			if len(assistantEffectiveSourcePaths(result)) == 0 {
				v.addActivity("Assistant answer has no explicit source context attached.")
			}
			v.assistantLastPrompt = text
			v.assistantLastResult = result
			v.persistAssistantExchange(text, result, startedAt)
			v.addActivity("Assistant response completed with " + result.Model + ".")
		})
	}()
}

func (v *View) loadAssistantProfile() {
	if v.assistantProfileStore == nil {
		v.assistantProfile = assistantSvc.DefaultProfile()
		v.refreshAssistantProfileControls()
		return
	}
	profile, err := v.assistantProfileStore.Get()
	if err != nil {
		v.assistantProfile = assistantSvc.DefaultProfile()
		v.addActivity("Assistant profile defaults loaded: " + err.Error())
	} else {
		v.assistantProfile = profile
	}
	v.refreshAssistantProfileControls()
}

func (v *View) refreshAssistantProfileControls() {
	if v.assistantProfileSelect == nil || v.assistantMemory == nil {
		return
	}
	profile := assistantSvc.NormalizeProfile(v.assistantProfile)
	options := make([]string, 0, len(profile.PromptProfiles))
	for _, item := range profile.PromptProfiles {
		options = append(options, assistantProfileOption(item))
	}
	v.assistantProfileSelect.Options = options
	if active := assistantSvc.ActivePromptProfile(profile); active.ID != "" {
		v.assistantProfileSelect.SetSelected(assistantProfileOption(active))
	}
	v.assistantMemory.SetText(profile.Memory)
	v.assistantProfile = profile
}

func (v *View) updateAssistantProfileSelection(option string) {
	profileID := assistantProfileIDFromOption(option, v.assistantProfile)
	if profileID == "" {
		return
	}
	v.assistantProfile.ActiveProfileID = profileID
}

func (v *View) saveAssistantProfile() {
	if v.assistantProfileStore == nil {
		v.addActivity("Assistant profile store is unavailable.")
		return
	}
	profile := v.assistantProfile
	if v.assistantProfileSelect != nil && strings.TrimSpace(v.assistantProfileSelect.Selected) != "" {
		profile.ActiveProfileID = assistantProfileIDFromOption(v.assistantProfileSelect.Selected, profile)
	}
	if v.assistantMemory != nil {
		profile.Memory = v.assistantMemory.Text
	}
	if len(profile.PromptProfiles) == 0 {
		profile.PromptProfiles = assistantSvc.DefaultProfile().PromptProfiles
	}
	saved, err := v.assistantProfileStore.Save(profile)
	if err != nil {
		v.addActivity("Assistant profile save failed: " + err.Error())
		return
	}
	v.assistantProfile = saved
	v.refreshAssistantProfileControls()
	v.addActivity("Assistant profile saved: " + assistantSvc.ActivePromptProfile(saved).Name + ".")
}

func (v *View) retryLatestAssistantAnswer(prompt *widget.Entry, response *widget.RichText, send *widget.Button) {
	if strings.TrimSpace(v.assistantLastPrompt) == "" {
		v.addActivity("No assistant answer is available to retry yet.")
		return
	}
	prompt.SetText(v.assistantLastPrompt)
	v.runAssistantRequest(prompt, response, send, "Ask")
}

func (v *View) compareLatestAssistantAnswer(prompt *widget.Entry, response *widget.RichText, send *widget.Button) {
	if strings.TrimSpace(v.assistantLastPrompt) == "" || strings.TrimSpace(v.assistantLastResult.Message) == "" {
		v.addActivity("No assistant answer is available to compare yet.")
		return
	}
	comparePrompt := compareLatestAssistantPrompt(v.assistantLastPrompt, v.assistantLastResult.Message)
	prompt.SetText(comparePrompt)
	v.runAssistantRequest(prompt, response, send, "Ask")
}

func (v *View) saveLatestAssistantAnswer() {
	workspace := v.state.Workspace()
	if strings.TrimSpace(workspace.Root) == "" {
		v.addActivity("Open a workspace before saving an assistant answer artifact.")
		return
	}
	if strings.TrimSpace(v.assistantLastResult.Message) == "" {
		v.addActivity("No assistant answer is available to save yet.")
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		v.addActivity("Assistant answer artifact failed: " + err.Error())
		return
	}
	diagnostic := assistantEvidenceDiagnosticForResult(v.assistantLastResult)
	citationSnippets := assistantCitationSnippets(workspace.Root, v.assistantLastResult, v.workspaceService)
	artifact, err := store.WriteChatAnswer(artifactsSvc.ChatAnswerReport{
		Prompt:                 v.assistantLastPrompt,
		Content:                v.assistantLastResult.Message,
		Model:                  v.assistantLastResult.Model,
		ContextRelPath:         v.assistantLastResult.ContextRelPath,
		Source:                 "Nexus assistant",
		SourcePaths:            assistantEffectiveSourcePaths(v.assistantLastResult),
		CitationRefs:           assistantCitationRefs(v.assistantLastResult),
		UnverifiedCitationRefs: assistantUnverifiedCitationRefs(v.assistantLastResult),
		CitationSnippets:       citationSnippets,
		CitedSourcePaths:       diagnostic.CitedSourcePaths,
		UncitedSourcePaths:     diagnostic.UncitedSourcePaths,
		EvidenceQuality:        diagnostic.Quality,
		EvidenceSummary:        diagnostic.Summary,
	})
	if err != nil {
		v.addActivity("Assistant answer artifact failed: " + err.Error())
		return
	}
	v.persistArtifactRecord(artifact)
	v.refreshArtifactsWithQuery("kind:chat-answer")
	v.addActivity(artifact.Message)
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
		Role:           "assistant",
		Content:        result.Message,
		Model:          result.Model,
		ContextRelPath: strings.TrimSpace(result.ContextRelPath),
		SourcePaths:    assistantEffectiveSourcePaths(result),
		CreatedAt:      time.Now().UTC(),
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

func assistantResponseMarkdown(result assistantSvc.Result) string {
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = "Assistant completed without a final message."
	}
	footer := assistantDiagnosticFooter(result)
	if footer != "" {
		message += "\n\n" + footer
	}
	return message
}

func assistantDiagnosticFooter(result assistantSvc.Result) string {
	lines := []string{}
	if model := strings.TrimSpace(result.Model); model != "" {
		lines = append(lines, "Model: `"+model+"`")
	}
	if contextPath := strings.TrimSpace(result.ContextRelPath); contextPath != "" && contextPath != "agent" {
		lines = append(lines, "Context: `"+contextPath+"`")
	}
	sources := assistantEffectiveSourcePaths(result)
	if len(sources) > 0 {
		lines = append(lines, "Sources: `"+strings.Join(sources, "`, `")+"`")
	} else {
		lines = append(lines, "> No explicit source context is attached to this answer.")
	}
	citations := assistantCitationRefs(result)
	if len(citations) > 0 {
		lines = append(lines, "Citations: `"+strings.Join(citations, "`, `")+"`")
	}
	if unverified := assistantUnverifiedCitationRefs(result); len(unverified) > 0 {
		lines = append(lines, "Unverified citations: `"+strings.Join(unverified, "`, `")+"`")
	}
	if diagnostic := assistantEvidenceDiagnosticForResult(result); diagnostic.Summary != "" {
		lines = append(lines, "Evidence: "+diagnostic.Summary)
	}
	return strings.Join(lines, "\n")
}

type assistantEvidenceDiagnostic struct {
	Quality                 string
	Summary                 string
	SourceCount             int
	CitationCount           int
	UnverifiedCitationCount int
	CitedSourceCount        int
	CitedSourcePaths        []string
	UncitedSourcePaths      []string
}

func assistantEvidenceDiagnosticForResult(result assistantSvc.Result) assistantEvidenceDiagnostic {
	sources := assistantEffectiveSourcePaths(result)
	citations := assistantCitationRefs(result)
	unverified := assistantUnverifiedCitationRefs(result)
	citedSources, uncitedSources := assistantCitationSourceCoverage(sources, citations)
	diagnostic := assistantEvidenceDiagnostic{
		SourceCount:             len(sources),
		CitationCount:           len(citations),
		UnverifiedCitationCount: len(unverified),
		CitedSourceCount:        len(citedSources),
		CitedSourcePaths:        citedSources,
		UncitedSourcePaths:      uncitedSources,
	}
	switch {
	case len(sources) == 0:
		diagnostic.Quality = "weak"
		diagnostic.Summary = "weak (no explicit source context" + assistantUnverifiedSummarySuffix(len(unverified)) + ")."
	case len(citations) == 0:
		diagnostic.Quality = "source-backed"
		if len(unverified) > 0 {
			diagnostic.Summary = fmt.Sprintf("source-backed (%d source(s), no verified line citations, cited 0/%d source(s); %s outside selected sources).", len(sources), len(sources), assistantPlural(len(unverified), "citation", "citations"))
		} else {
			diagnostic.Summary = fmt.Sprintf("source-backed (%d source(s), no line citations detected, cited 0/%d source(s)).", len(sources), len(sources))
		}
	default:
		diagnostic.Quality = "line-cited"
		coverage := assistantCitationCoverageSummary(len(citedSources), sources, uncitedSources)
		if len(unverified) > 0 {
			diagnostic.Summary = fmt.Sprintf("line-cited (%d source(s), %d line ref(s), %s; %s outside selected sources).", len(sources), len(citations), coverage, assistantPlural(len(unverified), "citation", "citations"))
		} else {
			diagnostic.Summary = fmt.Sprintf("line-cited (%d source(s), %d line ref(s), %s).", len(sources), len(citations), coverage)
		}
	}
	return diagnostic
}

func assistantCitationCoverageSummary(citedSourceCount int, sources []string, uncitedSources []string) string {
	summary := fmt.Sprintf("cited %d/%d source(s)", citedSourceCount, len(sources))
	if len(uncitedSources) > 0 {
		summary += "; uncited: " + strings.Join(uncitedSources, ", ")
	}
	return summary
}

func assistantCitationSourceCoverage(sources []string, citations []string) ([]string, []string) {
	if len(sources) == 0 {
		return nil, nil
	}
	citationPaths := make([]string, 0, len(citations))
	for _, ref := range citations {
		path, _, _, ok := parseAssistantCitationRef(ref)
		if ok {
			citationPaths = append(citationPaths, path)
		}
	}
	cited := []string{}
	uncited := []string{}
	for _, source := range sources {
		source = normalizeAssistantCitationPath(source)
		if source == "" {
			continue
		}
		if assistantSourceCoveredByCitation(source, citationPaths) {
			cited = append(cited, source)
		} else {
			uncited = append(uncited, source)
		}
	}
	return cited, uncited
}

func assistantSourceCoveredByCitation(source string, citationPaths []string) bool {
	for _, citationPath := range citationPaths {
		if source == "." || source == citationPath || strings.HasPrefix(citationPath, strings.TrimSuffix(source, "/")+"/") {
			return true
		}
	}
	return false
}

func assistantUnverifiedSummarySuffix(count int) string {
	if count == 0 {
		return ""
	}
	return "; " + assistantPlural(count, "unverified line ref", "unverified line refs")
}

func assistantPlural(count int, singular string, plural string) string {
	word := plural
	if count == 1 {
		word = singular
	}
	return fmt.Sprintf("%d %s", count, word)
}

func assistantEffectiveSourcePaths(result assistantSvc.Result) []string {
	paths := result.SourcePaths
	if len(paths) == 0 {
		paths = assistantSourcePathsFromContext(result.ContextRelPath)
	}
	return dedupeAssistantSourcePaths(paths)
}

func assistantSourcePathsFromContext(contextRelPath string) []string {
	contextRelPath = strings.TrimSpace(contextRelPath)
	if contextRelPath == "" || contextRelPath == "agent" {
		return nil
	}
	if strings.HasPrefix(contextRelPath, "pack: ") {
		return dedupeAssistantSourcePaths(strings.Split(strings.TrimPrefix(contextRelPath, "pack: "), ","))
	}
	if strings.HasPrefix(contextRelPath, "dir: ") {
		value := strings.TrimPrefix(contextRelPath, "dir: ")
		if marker := strings.LastIndex(value, " ("); marker > 0 && strings.HasSuffix(value, " files)") {
			value = value[:marker]
		}
		return []string{value}
	}
	if strings.HasPrefix(contextRelPath, "context: ") {
		value := strings.TrimSpace(strings.TrimPrefix(contextRelPath, "context: "))
		if strings.HasSuffix(value, " roots") {
			return nil
		}
		return []string{value}
	}
	if strings.HasPrefix(contextRelPath, "project: ") {
		return []string{strings.TrimSpace(strings.TrimPrefix(contextRelPath, "project: "))}
	}
	return []string{contextRelPath}
}

func dedupeAssistantSourcePaths(paths []string) []string {
	seen := map[string]bool{}
	cleaned := make([]string, 0, len(paths))
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || path == "agent" || seen[path] {
			continue
		}
		seen[path] = true
		cleaned = append(cleaned, path)
	}
	return cleaned
}

func assistantCitationRefs(result assistantSvc.Result) []string {
	sources := assistantEffectiveSourcePaths(result)
	if len(sources) == 0 {
		return nil
	}
	citations := []string{}
	for _, ref := range assistantCitationRefsFromMessage(result.Message) {
		path, _, _, ok := parseAssistantCitationRef(ref)
		if ok && assistantCitationAllowed(path, sources) {
			citations = append(citations, ref)
		}
	}
	return citations
}

func assistantUnverifiedCitationRefs(result assistantSvc.Result) []string {
	all := assistantCitationRefsFromMessage(result.Message)
	if len(all) == 0 {
		return nil
	}
	sources := assistantEffectiveSourcePaths(result)
	if len(sources) == 0 {
		return all
	}
	unverified := []string{}
	for _, ref := range all {
		path, _, _, ok := parseAssistantCitationRef(ref)
		if !ok || !assistantCitationAllowed(path, sources) {
			unverified = append(unverified, ref)
		}
	}
	return unverified
}

func assistantCitationRefsFromMessage(message string) []string {
	matches := assistantCitationPattern.FindAllStringSubmatch(message, -1)
	seen := map[string]bool{}
	citations := []string{}
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		path := normalizeAssistantCitationPath(match[1])
		start := strings.TrimSpace(match[2])
		if path == "" || start == "" {
			continue
		}
		ref := path + ":L" + start
		if len(match) > 3 {
			if end := strings.TrimSpace(match[3]); end != "" && end != start {
				ref += "-L" + end
			}
		}
		if !seen[ref] {
			seen[ref] = true
			citations = append(citations, ref)
		}
	}
	return citations
}

func assistantCitationSnippets(root string, result assistantSvc.Result, previewer assistantCitationPreviewer) []string {
	root = strings.TrimSpace(root)
	if root == "" || previewer == nil {
		return nil
	}
	refs := assistantCitationRefs(result)
	if len(refs) == 0 {
		return nil
	}
	snippets := []string{}
	for _, ref := range refs {
		if len(snippets) >= assistantCitationSnippetLimit {
			break
		}
		path, start, end, ok := parseAssistantCitationRef(ref)
		if !ok {
			continue
		}
		preview, err := previewer.PreviewFile(root, path)
		if err != nil || strings.TrimSpace(preview.Text) == "" {
			continue
		}
		snippet, ok := assistantSnippetFromText(ref, preview.Text, start, end)
		if ok {
			snippets = append(snippets, snippet)
		}
	}
	return snippets
}

func parseAssistantCitationRef(ref string) (string, int, int, bool) {
	matches := assistantCitationRefPattern.FindStringSubmatch(strings.TrimSpace(ref))
	if len(matches) < 3 {
		return "", 0, 0, false
	}
	path := normalizeAssistantCitationPath(matches[1])
	start, err := strconv.Atoi(matches[2])
	if path == "" || err != nil || start <= 0 {
		return "", 0, 0, false
	}
	end := start
	if len(matches) > 3 && strings.TrimSpace(matches[3]) != "" {
		if parsedEnd, err := strconv.Atoi(matches[3]); err == nil && parsedEnd >= start {
			end = parsedEnd
		}
	}
	if end-start+1 > assistantCitationSnippetLineLimit {
		end = start + assistantCitationSnippetLineLimit - 1
	}
	return path, start, end, true
}

func assistantSnippetFromText(ref string, text string, start int, end int) (string, bool) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	if start <= 0 || start > len(lines) {
		return "", false
	}
	if end < start {
		end = start
	}
	if end > len(lines) {
		end = len(lines)
	}
	parts := make([]string, 0, end-start+1)
	for lineNumber := start; lineNumber <= end; lineNumber++ {
		line := strings.TrimSpace(lines[lineNumber-1])
		if len(line) > assistantCitationSnippetLineMaxChars {
			line = line[:assistantCitationSnippetLineMaxChars-3] + "..."
		}
		parts = append(parts, fmt.Sprintf("L%d: %s", lineNumber, line))
	}
	return ref + " - " + strings.Join(parts, " | "), true
}

func normalizeAssistantCitationPath(path string) string {
	path = strings.TrimSpace(strings.ReplaceAll(path, "\\", "/"))
	path = strings.Trim(path, "`'\"()[]{}<>,.;")
	if path == "" || strings.Contains(path, "://") {
		return ""
	}
	return path
}

func assistantCitationAllowed(path string, sources []string) bool {
	if len(sources) == 0 {
		return true
	}
	for _, source := range sources {
		source = normalizeAssistantCitationPath(source)
		if source == "." || source == path || strings.HasPrefix(path, strings.TrimSuffix(source, "/")+"/") {
			return true
		}
	}
	return false
}

func compareLatestAssistantPrompt(prompt string, previousAnswer string) string {
	return strings.Join([]string{
		"Compare the previous assistant answer with a fresh answer using the currently selected model/settings.",
		"",
		"Original prompt:",
		strings.TrimSpace(prompt),
		"",
		"Previous assistant answer:",
		strings.TrimSpace(previousAnswer),
		"",
		"Return agreements, differences, corrections, and a recommended final answer. Stay grounded in attached source context and call out uncertainty.",
	}, "\n")
}

func assistantProfileOption(profile assistantSvc.PromptProfile) string {
	name := strings.TrimSpace(profile.Name)
	id := strings.TrimSpace(profile.ID)
	if name == "" {
		return id
	}
	if id == "" || strings.EqualFold(name, id) {
		return name
	}
	return name + " (" + id + ")"
}

func assistantProfileIDFromOption(option string, profile assistantSvc.Profile) string {
	option = strings.TrimSpace(option)
	if option == "" {
		return ""
	}
	profile = assistantSvc.NormalizeProfile(profile)
	for _, item := range profile.PromptProfiles {
		if option == item.ID || option == item.Name || option == assistantProfileOption(item) {
			return item.ID
		}
	}
	if strings.HasSuffix(option, ")") {
		start := strings.LastIndex(option, "(")
		if start >= 0 {
			return strings.TrimSpace(strings.TrimSuffix(option[start+1:], ")"))
		}
	}
	return option
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
		ApproveTool:   v.confirmAgentToolApproval,
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

func (v *View) confirmAgentToolApproval(ctx context.Context, request agentSvc.ToolApprovalRequest) bool {
	result := make(chan bool, 1)
	fyne.Do(func() {
		message := widget.NewLabel(agentToolApprovalMessage(request))
		message.Wrapping = fyne.TextWrapWord
		dialog.ShowCustomConfirm("Approve agent tool", "Approve once", "Deny", container.NewPadded(message), func(confirm bool) {
			v.recordAgentToolApproval(request, confirm)
			result <- confirm
		}, v.window)
	})
	select {
	case approved := <-result:
		return approved
	case <-ctx.Done():
		return false
	}
}

func (v *View) recordAgentToolApproval(request agentSvc.ToolApprovalRequest, approved bool) {
	workspace := v.state.Workspace()
	if workspace.Root == "" || v.approvalService == nil {
		return
	}
	decision := "denied"
	if approved {
		decision = "approved"
	}
	if _, err := v.approvalService.Append(workspace.Root, approvalsSvc.Record{
		Action:   "agent-tool:" + request.Name,
		Target:   agentToolApprovalTarget(request),
		Risk:     request.Risk,
		Decision: decision,
		Message:  "Per-call agent tool approval",
	}); err != nil {
		v.addActivity("Could not persist agent tool approval: " + err.Error())
		return
	}
	v.refreshApprovals()
}

func agentToolApprovalMessage(request agentSvc.ToolApprovalRequest) string {
	var builder strings.Builder
	builder.WriteString("Nexus Agent requested a high-risk tool.\n\n")
	builder.WriteString("Tool: ")
	builder.WriteString(request.Name)
	if request.Risk != "" {
		builder.WriteString("\nRisk: ")
		builder.WriteString(request.Risk)
	}
	if request.Description != "" {
		builder.WriteString("\n\n")
		builder.WriteString(request.Description)
	}
	if target := agentToolApprovalTarget(request); target != "" {
		builder.WriteString("\n\nTarget: ")
		builder.WriteString(target)
	}
	builder.WriteString("\n\nApprove only this single tool call?")
	return builder.String()
}

func agentToolApprovalTarget(request agentSvc.ToolApprovalRequest) string {
	for _, key := range []string{"relPath", "targetRelPath", "sourceRelPath", "taskId", "id"} {
		if value := strings.TrimSpace(request.Args[key]); value != "" {
			return value
		}
	}
	return ""
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
