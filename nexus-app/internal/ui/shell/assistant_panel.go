package shell

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
const assistantSourceActionLimit = 8
const assistantSourceDigestListLimit = 24
const assistantStreamRefreshInterval = 80 * time.Millisecond
const agentEventRefreshInterval = 120 * time.Millisecond

var assistantCitationPattern = regexp.MustCompile(`(?i)([\w./\\-]+\.[A-Za-z0-9]{1,12})(?:(?:#L|:)(\d+)(?:[-:L]+(\d+))?)`)
var assistantCitationRefPattern = regexp.MustCompile(`^(.+):L(\d+)(?:-L(\d+))?$`)

type assistantCitationPreviewer interface {
	PreviewFile(root string, relPath string) (domain.FilePreview, error)
}

type assistantController struct {
	view            *View
	contextStatus   *widget.Label
	contextList     *fyne.Container
	sourcesStatus   *widget.Label
	sourcesList     *fyne.Container
	lineageStatus   *widget.Label
	lineageList     *fyne.Container
	inspectorStatus *widget.Label
	inspectorList   *fyne.Container
	historyStatus   *widget.Label
	historyList     *fyne.Container
	prompt          *widget.Entry
	mode            *widget.Select
	runTaskApproval *widget.Check
	profile         assistantSvc.Profile
	profileSelect   *widget.Select
	modelRoute      *widget.Select
	runStatus       *widget.Label
	sourceDigest    *widget.Label
	stopButton      *widget.Button
	activeJobID     string
	memory          *widget.Entry
	lastPrompt      string
	lastResult      assistantSvc.Result
}

func newAssistantController(view *View) *assistantController {
	return &assistantController{view: view}
}

func (v *View) newAssistantPanel() fyne.CanvasObject {
	prompt := widget.NewMultiLineEntry()
	prompt.SetPlaceHolder("Ask Nexus about this workspace")
	prompt.Wrapping = fyne.TextWrapWord
	prompt.SetMinRowsVisible(3)
	v.assistant.prompt = prompt
	response := widget.NewRichTextFromMarkdown("Assistant output will stream here.")
	v.assistant.sourceDigest = widget.NewLabel("")
	v.assistant.sourceDigest.Wrapping = fyne.TextWrapWord
	v.assistant.contextStatus = widget.NewLabel("")
	v.assistant.contextStatus.Wrapping = fyne.TextWrapWord
	v.assistant.contextList = container.NewVBox()
	v.assistant.sourcesStatus = widget.NewLabel("")
	v.assistant.sourcesStatus.Wrapping = fyne.TextWrapWord
	v.assistant.sourcesList = container.NewVBox()
	v.assistant.lineageStatus = widget.NewLabel("")
	v.assistant.lineageStatus.Wrapping = fyne.TextWrapWord
	v.assistant.lineageList = container.NewVBox()
	v.assistant.inspectorStatus = widget.NewLabel("")
	v.assistant.inspectorStatus.Wrapping = fyne.TextWrapWord
	v.assistant.inspectorList = container.NewVBox()
	v.assistant.historyStatus = widget.NewLabel("")
	v.assistant.historyStatus.Wrapping = fyne.TextWrapWord
	v.assistant.historyList = container.NewVBox()
	pinSelection := widget.NewButton("Pin selection", v.pinSelectedAssistantContext)
	pinProject := widget.NewButton("Pin project", func() {
		v.pinAssistantContextPath(".")
	})
	clearPins := widget.NewButton("Clear", v.clearAssistantContextPins)
	contextBar := container.NewVBox(
		container.NewHBox(pinSelection, pinProject, clearPins),
		v.assistant.contextStatus,
		v.assistant.contextList,
	)
	sourcesBar := container.NewVBox(
		v.assistant.sourcesStatus,
		v.assistant.sourcesList,
	)
	lineageBar := container.NewVBox(
		v.assistant.lineageStatus,
		v.assistant.lineageList,
	)
	inspectorBar := container.NewVBox(
		v.assistant.inspectorStatus,
		v.assistant.inspectorList,
	)
	historyBar := container.NewVBox(
		v.assistant.historyStatus,
		v.assistant.historyList,
	)
	profileSelect := widget.NewSelect(nil, func(profileID string) {
		v.updateAssistantProfileSelection(profileID)
	})
	v.assistant.profileSelect = profileSelect
	memory := widget.NewMultiLineEntry()
	memory.SetPlaceHolder("Assistant memory and preferences")
	memory.SetMinRowsVisible(2)
	v.assistant.memory = memory
	saveProfile := widget.NewButtonWithIcon("Save memory", theme.DocumentSaveIcon(), v.saveAssistantProfile)
	profileBar := container.NewVBox(
		widget.NewLabel("Prompt profile"),
		profileSelect,
		memory,
		saveProfile,
	)
	modelRoute := widget.NewSelect(assistantModelRouteOptions(v.settingsStore), func(string) {
		v.refreshAssistantContextPins()
		v.refreshAssistantRunStatus()
	})
	modelRoute.SetSelected(assistantAutoModelRouteLabel)
	v.assistant.modelRoute = modelRoute
	mode := widget.NewSelect([]string{"Ask", "Agent"}, func(string) {
		v.refreshAssistantRunStatus()
	})
	mode.SetSelected("Ask")
	v.assistant.mode = mode
	agentTaskApproval := widget.NewCheck("Allow task tool this run", nil)
	v.assistant.runTaskApproval = agentTaskApproval
	v.assistant.runStatus = widget.NewLabel("")
	v.assistant.runStatus.Wrapping = fyne.TextWrapWord
	header := newAssistantHeader(v.assistant.runStatus)
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
	stop := widget.NewButtonWithIcon("Stop", theme.CancelIcon(), v.cancelActiveAssistantRun)
	stop.Importance = widget.HighImportance
	stop.Disable()
	v.assistant.stopButton = stop
	openSources := widget.NewButtonWithIcon("Open sources", theme.FolderOpenIcon(), v.openLatestAssistantSources)
	pinSources := widget.NewButtonWithIcon("Pin sources", theme.ContentAddIcon(), v.pinLatestAssistantSources)
	sourceDigest := widget.NewButtonWithIcon("Source digest", theme.SearchIcon(), v.showLatestAssistantSourceDigest)
	assistantActions := container.NewHBox(stop, retry, compare, saveAnswer, openSources, pinSources, sourceDigest)
	composerControls := container.NewVBox(mode, modelRoute, agentTaskApproval)
	composer := container.NewBorder(assistantActions, nil, composerControls, send, prompt)
	composer = container.NewPadded(composer)
	sidebar := container.NewVBox(profileBar, widget.NewSeparator(), contextBar, widget.NewSeparator(), sourcesBar, widget.NewSeparator(), lineageBar, widget.NewSeparator(), inspectorBar, widget.NewSeparator(), historyBar)
	messageArea := container.NewBorder(v.assistant.sourceDigest, nil, nil, nil, response)
	panel := newAssistantPanelLayout(header, composer, sidebar, messageArea)
	v.loadAssistantProfile()
	v.refreshAssistantContextPins()
	v.refreshAssistantRunStatus()
	v.refreshAssistantSourceDigest()
	v.refreshAssistantSourcesPane()
	v.refreshAssistantLineagePane()
	v.refreshAssistantInspectorPane()
	v.refreshAssistantHistory()
	return container.NewPadded(panel)
}

func newAssistantPanelLayout(header fyne.CanvasObject, composer fyne.CanvasObject, sidebar fyne.CanvasObject, messages fyne.CanvasObject) *fyne.Container {
	return container.NewBorder(header, composer, sidebar, nil, messages)
}

func newAssistantHeader(runStatus *widget.Label) fyne.CanvasObject {
	title := widget.NewLabel("Assistant")
	title.TextStyle = fyne.TextStyle{Bold: true}
	if runStatus == nil {
		runStatus = widget.NewLabel("")
	}
	runStatus.Wrapping = fyne.TextWrapWord
	return container.NewBorder(nil, nil, title, nil, runStatus)
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
		ModelRouteID:  v.selectedAssistantModelRouteID(text),
	}
	startedAt := time.Now().UTC()
	send.Disable()
	v.setAssistantRunStatus(assistantPreRunStatusLine(v.settingsStore, "Ask", selectedAssistantModelRouteOption(v), text, request.ContextPaths, request.SelectedPath))
	response.ParseMarkdown("Receiving response...")
	v.addActivity("Assistant request started.")

	go func() {
		stream := newAssistantStreamRenderer(response, assistantStreamRefreshInterval)
		result, err := v.assistantService.AskStream(context.Background(), request, func(delta string) error {
			stream.Append(delta)
			return nil
		})
		stream.Flush()
		stream.Stop()
		fyne.Do(func() {
			defer send.Enable()
			if err != nil {
				v.setAssistantRunStatus("Assistant failed: " + err.Error())
				response.ParseMarkdown("Assistant request failed: " + err.Error())
				v.addActivity("Assistant request failed: " + err.Error())
				return
			}
			response.ParseMarkdown(assistantResponseMarkdown(result))
			v.setAssistantRunStatus(assistantResultStatusLine(result))
			if result.ContextWarning != "" {
				v.addActivity(result.ContextWarning)
			}
			if result.RouteWarning != "" {
				v.addActivity(result.RouteWarning)
			}
			if len(assistantEffectiveSourcePaths(result)) == 0 {
				v.addActivity("Assistant answer has no explicit source context attached.")
			}
			v.assistant.lastPrompt = text
			v.assistant.lastResult = result
			v.refreshAssistantSourceDigest()
			v.refreshAssistantSourcesPane()
			v.refreshAssistantLineagePane()
			v.refreshAssistantInspectorPane()
			v.persistAssistantExchange(text, result, startedAt)
			v.addActivity("Assistant response completed with " + result.Model + ".")
		})
	}()
}

type assistantMarkdownRenderer interface {
	ParseMarkdown(string)
}

type assistantStreamRenderer struct {
	mu       sync.Mutex
	builder  strings.Builder
	dirty    bool
	render   func(string)
	stop     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
}

func newAssistantStreamRenderer(response assistantMarkdownRenderer, interval time.Duration) *assistantStreamRenderer {
	return newAssistantStreamRendererWithRender(func(text string) {
		if response == nil {
			return
		}
		fyne.Do(func() {
			response.ParseMarkdown(text)
		})
	}, interval)
}

func newAssistantStreamRendererWithRender(render func(string), interval time.Duration) *assistantStreamRenderer {
	if interval <= 0 {
		interval = assistantStreamRefreshInterval
	}
	stream := &assistantStreamRenderer{
		render:  render,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go stream.run(interval)
	return stream
}

func (r *assistantStreamRenderer) Append(delta string) {
	if r == nil || delta == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.builder.WriteString(delta)
	r.dirty = true
}

func (r *assistantStreamRenderer) Flush() {
	if r == nil {
		return
	}
	if text, ok := r.consume(); ok && r.render != nil {
		r.render(text)
	}
}

func (r *assistantStreamRenderer) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		close(r.stop)
		<-r.stopped
	})
}

func (r *assistantStreamRenderer) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(r.stopped)
	for {
		select {
		case <-ticker.C:
			r.Flush()
		case <-r.stop:
			r.Flush()
			return
		}
	}
}

func (r *assistantStreamRenderer) consume() (string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.dirty {
		return "", false
	}
	r.dirty = false
	return r.builder.String(), true
}

const assistantAutoModelRouteLabel = "Auto by selected context"
const assistantGlobalModelRouteLabel = "Global model fallback"

func (v *View) selectedAssistantModelRouteID(prompt string) string {
	if v == nil || v.assistant == nil || v.assistant.modelRoute == nil {
		return ""
	}
	routes := assistantModelRoutesForStore(v.settingsStore)
	option := strings.TrimSpace(v.assistant.modelRoute.Selected)
	if option == assistantAutoModelRouteLabel {
		return inferAssistantModelRouteID(prompt, v.state.AssistantContextPaths(), v.state.SelectedPath())
	}
	return assistantModelRouteIDFromOption(option, routes)
}

func assistantModelRouteOptions(store interface {
	LoadForDisplay() (settingsSvc.Settings, error)
}) []string {
	routes := assistantModelRoutesForStore(store)
	options := []string{assistantAutoModelRouteLabel, assistantGlobalModelRouteLabel}
	for _, route := range routes {
		options = append(options, route.Label)
	}
	return options
}

func assistantModelRoutesForStore(store interface {
	LoadForDisplay() (settingsSvc.Settings, error)
}) []settingsSvc.ModelRoute {
	if store == nil {
		return settingsSvc.DefaultModelRoutes()
	}
	settings, err := store.LoadForDisplay()
	if err != nil {
		return settingsSvc.DefaultModelRoutes()
	}
	return settings.ModelRoutes
}

func assistantModelRouteIDFromOption(option string, routes []settingsSvc.ModelRoute) string {
	option = strings.TrimSpace(option)
	if option == "" || option == assistantAutoModelRouteLabel || option == assistantGlobalModelRouteLabel {
		return ""
	}
	for _, route := range routes {
		if option == route.ID || option == route.Label {
			return route.ID
		}
	}
	return ""
}

func inferAssistantModelRouteID(prompt string, pinned []string, selected string) string {
	for _, candidate := range assistantRouteContextCandidates(pinned, selected) {
		if routeID := routeIDForAssistantPath(candidate); routeID != "" {
			return routeID
		}
	}
	return routeIDForAssistantPrompt(prompt)
}

func assistantRouteContextCandidates(pinned []string, selected string) []string {
	seen := map[string]bool{}
	candidates := make([]string, 0, len(pinned)+1)
	for _, relPath := range pinned {
		relPath = strings.TrimSpace(relPath)
		if relPath == "" || seen[relPath] {
			continue
		}
		seen[relPath] = true
		candidates = append(candidates, relPath)
	}
	selected = strings.TrimSpace(selected)
	if selected != "" && !seen[selected] {
		candidates = append(candidates, selected)
	}
	return candidates
}

func routeIDForAssistantPath(relPath string) string {
	normalized := strings.ToLower(filepath.ToSlash(strings.TrimSpace(relPath)))
	ext := strings.ToLower(filepath.Ext(normalized))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".svg":
		return settingsSvc.RouteVisionScreenshot
	case ".csv", ".tsv", ".xlsx", ".xls":
		return settingsSvc.RouteCSVExcelScripts
	case ".sql", ".sqlite", ".sqlite3", ".db", ".duckdb":
		return settingsSvc.RouteSQL
	case ".cypher", ".cql":
		return settingsSvc.RouteNeo4jCypher
	case ".go":
		return settingsSvc.RouteGoBackend
	case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
		return settingsSvc.RouteReactTypeScript
	case ".py", ".ipynb":
		return settingsSvc.RoutePythonCoding
	case ".php", ".blade.php":
		return settingsSvc.RoutePHPLaravel
	case ".md", ".markdown", ".txt", ".pdf", ".docx", ".html", ".htm", ".xml", ".rtf", ".odt":
		return settingsSvc.RouteResearchSummaries
	case ".json", ".jsonl", ".ndjson", ".parquet", ".log":
		if strings.Contains(normalized, "/data/") || strings.Contains(normalized, "/datasets/") {
			return settingsSvc.RouteCSVExcelScripts
		}
		return settingsSvc.RouteAnalytics
	default:
		if strings.Contains(normalized, "/data/") || strings.Contains(normalized, "/datasets/") || strings.Contains(normalized, "/analytics/") {
			return settingsSvc.RouteAnalytics
		}
	}
	return ""
}

func routeIDForAssistantPrompt(prompt string) string {
	normalized := strings.ToLower(strings.TrimSpace(prompt))
	switch {
	case assistantContainsAny(normalized, "screenshot", "image", "vision", "picture", "photo", "ui reference"):
		return settingsSvc.RouteVisionScreenshot
	case assistantContainsAny(normalized, "postgres", "mysql", "sqlite", "sql server", "duckdb", "query", "database"):
		return settingsSvc.RouteSQL
	case assistantContainsAny(normalized, "neo4j", "cypher"):
		return settingsSvc.RouteNeo4jCypher
	case assistantContainsAny(normalized, "csv", "excel", "xlsx", "spreadsheet"):
		return settingsSvc.RouteCSVExcelScripts
	case assistantContainsAny(normalized, "analytics", "dashboard", "kpi", "metric"):
		return settingsSvc.RouteAnalytics
	case assistantContainsAny(normalized, "research", "summarize", "summary", "document", "report"):
		return settingsSvc.RouteResearchSummaries
	case assistantContainsAny(normalized, "golang", " go ", "go backend"):
		return settingsSvc.RouteGoBackend
	case assistantContainsAny(normalized, "react", "typescript", "javascript", "frontend"):
		return settingsSvc.RouteReactTypeScript
	case assistantContainsAny(normalized, "python"):
		return settingsSvc.RoutePythonCoding
	case assistantContainsAny(normalized, "php", "laravel"):
		return settingsSvc.RoutePHPLaravel
	}
	return ""
}

func assistantContainsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func assistantRouteBudgetLine(store interface {
	LoadForDisplay() (settingsSvc.Settings, error)
}, option string, prompt string, pinned []string, selected string) string {
	settings := settingsSvc.Defaults()
	if store != nil {
		if loaded, err := store.LoadForDisplay(); err == nil {
			settings = loaded
		}
	}
	routeLabel := "Global fallback"
	option = strings.TrimSpace(option)
	switch option {
	case "", assistantAutoModelRouteLabel:
		routeLabel = "Auto -> global fallback"
		if routeID := inferAssistantModelRouteID(prompt, pinned, selected); routeID != "" {
			if route, ok := settingsSvc.ModelRouteByID(settings, routeID); ok {
				routeLabel = "Auto -> " + firstNonEmpty(route.Label, route.ID)
				if routed, ok := settingsSvc.SettingsForModelRoute(settings, routeID); ok {
					settings = routed
				}
			}
		}
	case assistantGlobalModelRouteLabel:
		routeLabel = "Global fallback"
	default:
		if route, ok := settingsRouteByLabelOrID(settings.ModelRoutes, option); ok {
			routeLabel = firstNonEmpty(route.Label, route.ID)
			if routed, ok := settingsSvc.SettingsForModelRoute(settings, route.ID); ok {
				settings = routed
			}
		} else {
			routeLabel = option + " -> global fallback"
		}
	}
	return fmt.Sprintf("Model route: %s. Context budget: ~%s.", routeLabel, formatDiagnosticsBytes(int64(assistantContextBudgetBytes(settings))))
}

func assistantPreRunStatusLine(store interface {
	LoadForDisplay() (settingsSvc.Settings, error)
}, mode string, option string, prompt string, pinned []string, selected string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "Ask"
	}
	context := "no explicit context"
	switch {
	case len(pinned) > 0:
		context = fmt.Sprintf("%d pinned context root(s)", len(pinned))
	case strings.TrimSpace(selected) != "":
		context = "selected context: " + strings.TrimSpace(selected)
	}
	return fmt.Sprintf("Ready: %s. %s Context: %s.", mode, assistantRouteBudgetLine(store, option, prompt, pinned, selected), context)
}

func assistantResultStatusLine(result assistantSvc.Result) string {
	diagnostic := assistantEvidenceDiagnosticForResult(result)
	model := strings.TrimSpace(result.Model)
	if model == "" {
		model = "model not reported"
	}
	route := strings.TrimSpace(result.ModelRoute)
	if route == "" {
		route = "global fallback"
	}
	line := fmt.Sprintf(
		"Completed: %s via %s. Evidence: %s Sources: %d, verified refs: %d, unverified refs: %d.",
		model,
		route,
		firstNonEmpty(diagnostic.Summary, "not classified."),
		diagnostic.SourceCount,
		diagnostic.CitationCount,
		diagnostic.UnverifiedCitationCount,
	)
	if warning := strings.TrimSpace(result.RouteWarning); warning != "" {
		line += " Route warning: " + warning
	}
	return line
}

func settingsRouteByLabelOrID(routes []settingsSvc.ModelRoute, option string) (settingsSvc.ModelRoute, bool) {
	option = strings.TrimSpace(option)
	for _, route := range routes {
		if option == route.ID || option == route.Label {
			return route, true
		}
	}
	return settingsSvc.ModelRoute{}, false
}

func assistantContextBudgetBytes(settings settingsSvc.Settings) int {
	config := llmSvc.ConfigFromSettings(settings)
	budgetTokens := config.ContextTokens - config.ResponseReserveTokens
	if budgetTokens <= 0 {
		return defaultAgentContextMaxBytes
	}
	return budgetTokens * 4
}

func (v *View) loadAssistantProfile() {
	if v.assistantProfileStore == nil {
		v.assistant.profile = assistantSvc.DefaultProfile()
		v.refreshAssistantProfileControls()
		return
	}
	profile, err := v.assistantProfileStore.Get()
	if err != nil {
		v.assistant.profile = assistantSvc.DefaultProfile()
		v.addActivity("Assistant profile defaults loaded: " + err.Error())
	} else {
		v.assistant.profile = profile
	}
	v.refreshAssistantProfileControls()
}

func (v *View) refreshAssistantProfileControls() {
	if v.assistant.profileSelect == nil || v.assistant.memory == nil {
		return
	}
	profile := assistantSvc.NormalizeProfile(v.assistant.profile)
	options := make([]string, 0, len(profile.PromptProfiles))
	for _, item := range profile.PromptProfiles {
		options = append(options, assistantProfileOption(item))
	}
	v.assistant.profileSelect.Options = options
	if active := assistantSvc.ActivePromptProfile(profile); active.ID != "" {
		v.assistant.profileSelect.SetSelected(assistantProfileOption(active))
	}
	v.assistant.memory.SetText(profile.Memory)
	v.assistant.profile = profile
}

func (v *View) updateAssistantProfileSelection(option string) {
	profileID := assistantProfileIDFromOption(option, v.assistant.profile)
	if profileID == "" {
		return
	}
	v.assistant.profile.ActiveProfileID = profileID
}

func (v *View) saveAssistantProfile() {
	if v.assistantProfileStore == nil {
		v.addActivity("Assistant profile store is unavailable.")
		return
	}
	profile := v.assistant.profile
	if v.assistant.profileSelect != nil && strings.TrimSpace(v.assistant.profileSelect.Selected) != "" {
		profile.ActiveProfileID = assistantProfileIDFromOption(v.assistant.profileSelect.Selected, profile)
	}
	if v.assistant.memory != nil {
		profile.Memory = v.assistant.memory.Text
	}
	if len(profile.PromptProfiles) == 0 {
		profile.PromptProfiles = assistantSvc.DefaultProfile().PromptProfiles
	}
	saved, err := v.assistantProfileStore.Save(profile)
	if err != nil {
		v.addActivity("Assistant profile save failed: " + err.Error())
		return
	}
	v.assistant.profile = saved
	v.refreshAssistantProfileControls()
	v.addActivity("Assistant profile saved: " + assistantSvc.ActivePromptProfile(saved).Name + ".")
}

func (v *View) retryLatestAssistantAnswer(prompt *widget.Entry, response *widget.RichText, send *widget.Button) {
	if strings.TrimSpace(v.assistant.lastPrompt) == "" {
		v.addActivity("No assistant answer is available to retry yet.")
		return
	}
	prompt.SetText(v.assistant.lastPrompt)
	v.runAssistantRequest(prompt, response, send, "Ask")
}

func (v *View) compareLatestAssistantAnswer(prompt *widget.Entry, response *widget.RichText, send *widget.Button) {
	if strings.TrimSpace(v.assistant.lastPrompt) == "" || strings.TrimSpace(v.assistant.lastResult.Message) == "" {
		v.addActivity("No assistant answer is available to compare yet.")
		return
	}
	comparePrompt := compareLatestAssistantPrompt(v.assistant.lastPrompt, v.assistant.lastResult.Message)
	prompt.SetText(comparePrompt)
	v.runAssistantRequest(prompt, response, send, "Ask")
}

func (v *View) saveLatestAssistantAnswer() {
	workspace := v.state.Workspace()
	if strings.TrimSpace(workspace.Root) == "" {
		v.addActivity("Open a workspace before saving an assistant answer artifact.")
		return
	}
	if strings.TrimSpace(v.assistant.lastResult.Message) == "" {
		v.addActivity("No assistant answer is available to save yet.")
		return
	}
	store, err := artifactsSvc.NewStore(workspace.Root)
	if err != nil {
		v.addActivity("Assistant answer artifact failed: " + err.Error())
		return
	}
	diagnostic := assistantEvidenceDiagnosticForResult(v.assistant.lastResult)
	citationSnippets := assistantCitationSnippets(workspace.Root, v.assistant.lastResult, v.workspaceService)
	artifact, err := store.WriteChatAnswer(artifactsSvc.ChatAnswerReport{
		Prompt:                 v.assistant.lastPrompt,
		Content:                v.assistant.lastResult.Message,
		Model:                  v.assistant.lastResult.Model,
		ModelRouteID:           v.assistant.lastResult.ModelRouteID,
		ModelRoute:             v.assistant.lastResult.ModelRoute,
		ContextRelPath:         v.assistant.lastResult.ContextRelPath,
		Source:                 "Nexus assistant",
		SourcePaths:            assistantEffectiveSourcePaths(v.assistant.lastResult),
		CitationRefs:           assistantCitationRefs(v.assistant.lastResult),
		UnverifiedCitationRefs: assistantUnverifiedCitationRefs(v.assistant.lastResult),
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

func (v *View) openLatestAssistantSources() {
	workspace := v.state.Workspace()
	if strings.TrimSpace(workspace.Root) == "" {
		v.addActivity("Open a workspace before opening assistant sources.")
		return
	}
	paths := assistantActionableSourcePaths(v.assistant.lastResult, assistantSourceActionLimit)
	if len(paths) == 0 {
		v.addActivity("No assistant sources are available to open.")
		return
	}
	opened := 0
	failed := 0
	for _, relPath := range paths {
		preview, err := v.workspaceService.PreviewFile(workspace.Root, relPath)
		if err != nil {
			failed++
			v.addActivity("Could not open assistant source " + relPath + ": " + err.Error())
			continue
		}
		v.openPreviewTab(preview)
		opened++
	}
	v.addActivity(assistantSourceActionSummary("Opened", opened, len(paths), failed))
}

func (v *View) pinLatestAssistantSources() {
	workspace := v.state.Workspace()
	if strings.TrimSpace(workspace.Root) == "" {
		v.addActivity("Open a workspace before pinning assistant sources.")
		return
	}
	paths := assistantActionableSourcePaths(v.assistant.lastResult, assistantSourceActionLimit)
	if len(paths) == 0 {
		v.addActivity("No assistant sources are available to pin.")
		return
	}
	added := 0
	for _, relPath := range paths {
		if v.state.AddAssistantContextPath(relPath) {
			added++
		}
	}
	v.refreshAssistantContextPins()
	v.addActivity(assistantSourceActionSummary("Pinned", added, len(paths), 0))
}

func (v *View) showLatestAssistantSourceDigest() {
	if strings.TrimSpace(v.assistant.lastResult.Message) == "" {
		v.addActivity("No assistant answer is available for source digest yet.")
		return
	}
	markdown := assistantSourceDigestMarkdown(v.assistant.lastResult)
	if v.window == nil {
		v.addActivity("Assistant source digest is unavailable without a window.")
		return
	}
	content := widget.NewRichTextFromMarkdown(markdown)
	content.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(720, 520))
	dialog.ShowCustom("Assistant source digest", "Close", scroll, v.window)
	v.addActivity("Opened assistant source digest.")
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
	if v.assistant.historyStatus == nil || v.assistant.historyList == nil {
		return
	}
	turns := v.state.AssistantConversation()
	v.assistant.historyList.Objects = nil
	if len(turns) == 0 {
		v.assistant.historyStatus.SetText("Chat history: no persisted workspace turns yet.")
		v.assistant.historyList.Add(widget.NewLabel("Ask a question to start history."))
		v.assistant.historyList.Refresh()
		return
	}
	v.assistant.historyStatus.SetText(fmt.Sprintf("Chat history: %d recent persisted turn(s).", len(turns)))
	start := len(turns) - assistantHistoryPreviewLimit
	if start < 0 {
		start = 0
	}
	for _, turn := range turns[start:] {
		label := widget.NewLabel(chatTurnPreview(turn))
		label.Truncation = fyne.TextTruncateEllipsis
		v.assistant.historyList.Add(label)
	}
	v.assistant.historyList.Refresh()
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
	if route := strings.TrimSpace(result.ModelRoute); route != "" {
		lines = append(lines, "Model route: `"+route+"`")
	}
	if warning := strings.TrimSpace(result.RouteWarning); warning != "" {
		lines = append(lines, "Model route warning: "+warning)
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

func assistantActionableSourcePaths(result assistantSvc.Result, limit int) []string {
	paths := assistantEffectiveSourcePaths(result)
	if limit <= 0 || len(paths) <= limit {
		return paths
	}
	clipped := make([]string, limit)
	copy(clipped, paths[:limit])
	return clipped
}

func assistantSourceActionSummary(action string, count int, total int, failed int) string {
	action = firstNonEmpty(action, "Processed")
	summary := fmt.Sprintf("%s %d/%d assistant source(s).", action, count, total)
	if failed > 0 {
		summary += fmt.Sprintf(" %d failed.", failed)
	}
	return summary
}

func assistantSourceDigestMarkdown(result assistantSvc.Result) string {
	diagnostic := assistantEvidenceDiagnosticForResult(result)
	lines := []string{
		"# Assistant Source Digest",
		"",
		"Evidence: " + firstNonEmpty(diagnostic.Summary, "not classified."),
		fmt.Sprintf("Sources: %d. Verified refs: %d. Unverified refs: %d.", diagnostic.SourceCount, diagnostic.CitationCount, diagnostic.UnverifiedCitationCount),
	}
	if model := strings.TrimSpace(result.Model); model != "" {
		lines = append(lines, "Model: `"+model+"`")
	}
	if route := strings.TrimSpace(result.ModelRoute); route != "" {
		lines = append(lines, "Route: `"+route+"`")
	}
	if warning := strings.TrimSpace(result.RouteWarning); warning != "" {
		lines = append(lines, "Route warning: "+warning)
	}
	lines = append(lines,
		"",
		assistantMarkdownList("Sources", assistantEffectiveSourcePaths(result), "No explicit source context is attached.", assistantSourceDigestListLimit),
		"",
		assistantMarkdownList("Verified citations", assistantCitationRefs(result), "No verified line citations were detected.", assistantSourceDigestListLimit),
		"",
		assistantMarkdownList("Unverified citations", assistantUnverifiedCitationRefs(result), "No out-of-context line citations were detected.", assistantSourceDigestListLimit),
		"",
		assistantMarkdownList("Cited sources", diagnostic.CitedSourcePaths, "No sources were covered by verified citations.", assistantSourceDigestListLimit),
		"",
		assistantMarkdownList("Uncited sources", diagnostic.UncitedSourcePaths, "No uncited sources remain.", assistantSourceDigestListLimit),
	)
	return strings.Join(lines, "\n")
}

func assistantMarkdownList(title string, values []string, empty string, limit int) string {
	title = firstNonEmpty(strings.TrimSpace(title), "Items")
	if len(values) == 0 {
		return "## " + title + "\n\n" + firstNonEmpty(strings.TrimSpace(empty), "None.")
	}
	visible := values
	if limit > 0 && len(values) > limit {
		visible = values[:limit]
	}
	lines := []string{"## " + title}
	for _, value := range visible {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		lines = append(lines, "- `"+value+"`")
	}
	if len(visible) < len(values) {
		lines = append(lines, fmt.Sprintf("- ... %d more item(s) hidden", len(values)-len(visible)))
	}
	return strings.Join(lines, "\n")
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
	if v.assistant.contextStatus == nil || v.assistant.contextList == nil {
		return
	}
	paths := v.state.AssistantContextPaths()
	selected := selectedPathOrEmpty(v)
	budgetLine := assistantRouteBudgetLine(v.settingsStore, selectedAssistantModelRouteOption(v), "", paths, selected)
	v.assistant.contextList.Objects = nil
	if len(paths) == 0 {
		if selected == "" {
			v.assistant.contextStatus.SetText("Context: pin files, folders, or the project root before sending. " + budgetLine)
		} else {
			v.assistant.contextStatus.SetText("Context: selected item will be used unless pins are added: " + selected + ". " + budgetLine)
		}
		v.assistant.contextList.Add(widget.NewLabel("No pinned context."))
		v.assistant.contextList.Refresh()
		v.refreshAssistantInspectorPane()
		v.refreshAssistantRunStatus()
		return
	}
	v.assistant.contextStatus.SetText(fmt.Sprintf("Context pack: %d pinned root(s). %s", len(paths), budgetLine))
	for _, relPath := range paths {
		pinnedPath := relPath
		label := widget.NewLabel(pinnedPath)
		label.Truncation = fyne.TextTruncateEllipsis
		remove := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			v.removeAssistantContextPin(pinnedPath)
		})
		v.assistant.contextList.Add(container.NewBorder(nil, nil, nil, remove, label))
	}
	v.assistant.contextList.Refresh()
	v.refreshAssistantInspectorPane()
	v.refreshAssistantRunStatus()
}

func selectedAssistantModelRouteOption(v *View) string {
	if v == nil || v.assistant == nil || v.assistant.modelRoute == nil {
		return assistantAutoModelRouteLabel
	}
	return strings.TrimSpace(v.assistant.modelRoute.Selected)
}

func (v *View) refreshAssistantRunStatus() {
	if v == nil || v.assistant == nil || v.assistant.runStatus == nil {
		return
	}
	mode := "Ask"
	if v.assistant.mode != nil && strings.TrimSpace(v.assistant.mode.Selected) != "" {
		mode = strings.TrimSpace(v.assistant.mode.Selected)
	}
	v.assistant.runStatus.SetText(assistantPreRunStatusLine(
		v.settingsStore,
		mode,
		selectedAssistantModelRouteOption(v),
		"",
		v.state.AssistantContextPaths(),
		selectedPathOrEmpty(v),
	))
}

func (v *View) setAssistantRunStatus(status string) {
	if v == nil || v.assistant == nil || v.assistant.runStatus == nil {
		return
	}
	v.assistant.runStatus.SetText(strings.TrimSpace(status))
}

func (v *View) refreshAssistantSourceDigest() {
	if v == nil || v.assistant == nil || v.assistant.sourceDigest == nil {
		return
	}
	v.assistant.sourceDigest.SetText(assistantVisibleSourceDigest(v.assistant.lastResult))
}

func assistantVisibleSourceDigest(result assistantSvc.Result) string {
	if strings.TrimSpace(result.Message) == "" {
		return "Source digest: no answer yet."
	}
	diagnostic := assistantEvidenceDiagnosticForResult(result)
	return fmt.Sprintf(
		"Source digest: %s Sources: %d. Verified refs: %d. Unverified refs: %d.",
		firstNonEmpty(diagnostic.Summary, "not classified."),
		diagnostic.SourceCount,
		diagnostic.CitationCount,
		diagnostic.UnverifiedCitationCount,
	)
}

func (v *View) refreshAssistantSourcesPane() {
	if v == nil || v.assistant == nil || v.assistant.sourcesStatus == nil || v.assistant.sourcesList == nil {
		return
	}
	result := v.assistant.lastResult
	v.assistant.sourcesStatus.SetText(assistantSourcesPaneStatus(result))
	v.assistant.sourcesList.Objects = nil
	labels := assistantSourcesPaneLabels(result, assistantSourceActionLimit)
	if len(labels) == 0 {
		v.assistant.sourcesList.Add(widget.NewLabel("No assistant sources yet."))
		v.assistant.sourcesList.Refresh()
		return
	}
	for _, value := range labels {
		label := widget.NewLabel(value)
		label.Truncation = fyne.TextTruncateEllipsis
		v.assistant.sourcesList.Add(label)
	}
	v.assistant.sourcesList.Refresh()
}

func assistantSourcesPaneStatus(result assistantSvc.Result) string {
	if strings.TrimSpace(result.Message) == "" {
		return "Sources: no answer yet."
	}
	diagnostic := assistantEvidenceDiagnosticForResult(result)
	if diagnostic.SourceCount == 0 {
		return "Sources: no explicit source context attached."
	}
	return fmt.Sprintf("Sources: %d source(s). Evidence: %s", diagnostic.SourceCount, firstNonEmpty(diagnostic.Summary, "not classified."))
}

func assistantSourcesPaneLabels(result assistantSvc.Result, limit int) []string {
	return assistantActionableSourcePaths(result, limit)
}

func (v *View) refreshAssistantLineagePane() {
	if v == nil || v.assistant == nil || v.assistant.lineageStatus == nil || v.assistant.lineageList == nil {
		return
	}
	result := v.assistant.lastResult
	v.assistant.lineageStatus.SetText(assistantLineagePaneStatus(result))
	v.assistant.lineageList.Objects = nil
	labels := assistantLineagePaneLabels(result)
	if len(labels) == 0 {
		v.assistant.lineageList.Add(widget.NewLabel("No assistant lineage yet."))
		v.assistant.lineageList.Refresh()
		return
	}
	for _, value := range labels {
		label := widget.NewLabel(value)
		label.Truncation = fyne.TextTruncateEllipsis
		v.assistant.lineageList.Add(label)
	}
	v.assistant.lineageList.Refresh()
}

func assistantLineagePaneStatus(result assistantSvc.Result) string {
	if strings.TrimSpace(result.Message) == "" {
		return "Lineage: no answer yet."
	}
	model := firstNonEmpty(result.Model, "model not reported")
	route := firstNonEmpty(result.ModelRoute, "global fallback")
	return fmt.Sprintf("Lineage: answer from %s via %s.", model, route)
}

func assistantLineagePaneLabels(result assistantSvc.Result) []string {
	if strings.TrimSpace(result.Message) == "" {
		return nil
	}
	diagnostic := assistantEvidenceDiagnosticForResult(result)
	labels := []string{
		fmt.Sprintf("Sources: %d", diagnostic.SourceCount),
		fmt.Sprintf("Verified refs: %d", diagnostic.CitationCount),
		fmt.Sprintf("Unverified refs: %d", diagnostic.UnverifiedCitationCount),
	}
	if contextPath := strings.TrimSpace(result.ContextRelPath); contextPath != "" && contextPath != "agent" {
		labels = append([]string{"Context: " + contextPath}, labels...)
	}
	if warning := strings.TrimSpace(result.RouteWarning); warning != "" {
		labels = append(labels, "Route warning: "+warning)
	}
	return labels
}

func (v *View) refreshAssistantInspectorPane() {
	if v == nil || v.assistant == nil || v.assistant.inspectorStatus == nil || v.assistant.inspectorList == nil {
		return
	}
	selected := selectedPathOrEmpty(v)
	pins := v.state.AssistantContextPaths()
	result := v.assistant.lastResult
	v.assistant.inspectorStatus.SetText(assistantInspectorPaneStatus(result, selected, pins))
	v.assistant.inspectorList.Objects = nil
	labels := assistantInspectorPaneLabels(result, selected, pins)
	for _, value := range labels {
		label := widget.NewLabel(value)
		label.Truncation = fyne.TextTruncateEllipsis
		v.assistant.inspectorList.Add(label)
	}
	v.assistant.inspectorList.Refresh()
}

func assistantInspectorPaneStatus(result assistantSvc.Result, selected string, pins []string) string {
	if strings.TrimSpace(result.Message) == "" {
		if strings.TrimSpace(selected) != "" {
			return "Inspector: selected " + strings.TrimSpace(selected) + "."
		}
		if len(pins) > 0 {
			return fmt.Sprintf("Inspector: %d pinned context root(s).", len(pins))
		}
		return "Inspector: no active assistant answer."
	}
	return "Inspector: latest assistant answer."
}

func assistantInspectorPaneLabels(result assistantSvc.Result, selected string, pins []string) []string {
	labels := []string{}
	if selected = strings.TrimSpace(selected); selected != "" {
		labels = append(labels, "Selected: "+selected)
	}
	if len(pins) > 0 {
		labels = append(labels, fmt.Sprintf("Pinned roots: %d", len(pins)))
	}
	if strings.TrimSpace(result.Message) != "" {
		labels = append(labels,
			fmt.Sprintf("Answer chars: %d", len([]rune(result.Message))),
			"Model: "+firstNonEmpty(result.Model, "model not reported"),
			"Route: "+firstNonEmpty(result.ModelRoute, "global fallback"),
		)
	}
	if len(labels) == 0 {
		return []string{"No selection, pins, or answer yet."}
	}
	return labels
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
		ModelRouteID:  v.selectedAssistantModelRouteID(text),
		ApproveWrites: v.approvalService.HasFullProjectAccess(workspace.Root),
		ApproveShell:  v.assistantRunTaskApprovalChecked(),
		ApproveTool:   v.confirmAgentToolApproval,
	}
	v.attachAgentContext(&request)
	job, ctx := v.jobService.Start("agent", agentJobLabel(text))
	v.jobService.AppendLog(job.ID, "Prompt: "+agentJobLabel(text))
	send.Disable()
	v.setAssistantStopState(job.ID, true)
	v.setAssistantRunStatus(assistantAgentRunningStatusLine(job.ID, request))
	response.ParseMarkdown("Agent starting...")
	v.addActivity("Agent request started as " + job.ID + ".")
	v.refreshJobs()
	go func() {
		events := newAgentEventRenderer(func(markdown string, lines []string) {
			fyne.Do(func() {
				response.ParseMarkdown(markdown)
				for _, line := range lines {
					v.addActivity(line)
					v.jobService.AppendLog(job.ID, line)
				}
				v.refreshJobs()
			})
		}, agentEventRefreshInterval)
		result, err := v.agentService.Run(ctx, request, func(event agentSvc.Event) {
			line := agentEventLine(event)
			if line == "" {
				return
			}
			events.Append(line)
		})
		events.Flush()
		events.Stop()
		fyne.Do(func() {
			defer send.Enable()
			defer v.setAssistantStopState("", false)
			if v.assistant.runTaskApproval != nil && !v.approvalService.HasFullProjectAccess(workspace.Root) {
				v.assistant.runTaskApproval.SetChecked(false)
			}
			if err != nil {
				message := "Agent request failed: " + err.Error()
				v.setAssistantRunStatus(message)
				response.ParseMarkdown(message)
				v.addActivity(message)
				v.jobService.Finish(job.ID, jobsSvc.StatusFailed, message, err)
				v.persistAgentRun(job.ID, request, result, "failed", message, job.StartedAt)
				v.refreshJobs()
				return
			}
			response.ParseMarkdown(agentFinalMarkdown(result))
			v.setAssistantRunStatus(assistantAgentResultStatusLine(job.ID, result))
			if result.RouteWarning != "" {
				v.addActivity(result.RouteWarning)
			}
			status := jobsSvc.StatusSuccess
			message := fmt.Sprintf("Agent response completed after %d iteration(s).", result.Iterations)
			if result.StopReason == agentSvc.StopReasonTimeout {
				status = jobsSvc.StatusTimedOut
				message = fmt.Sprintf("Agent request timed out after %s.", formatAgentRunTimeout(agentSvc.EffectiveRunTimeout(request)))
			}
			v.addActivity(message)
			v.jobService.Finish(job.ID, status, message, nil)
			v.persistAgentRun(job.ID, request, result, string(status), result.Message, job.StartedAt)
			v.refreshJobs()
		})
	}()
}

func (v *View) setAssistantStopState(jobID string, enabled bool) {
	if v == nil || v.assistant == nil {
		return
	}
	v.assistant.activeJobID = strings.TrimSpace(jobID)
	if v.assistant.stopButton == nil {
		return
	}
	if enabled && v.assistant.activeJobID != "" {
		v.assistant.stopButton.Enable()
		return
	}
	v.assistant.stopButton.Disable()
}

func (v *View) cancelActiveAssistantRun() {
	if v == nil || v.assistant == nil {
		return
	}
	jobID := strings.TrimSpace(v.assistant.activeJobID)
	if jobID == "" || v.jobService == nil {
		v.setAssistantRunStatus("No cancellable assistant run is active.")
		return
	}
	if !v.jobService.Cancel(jobID) {
		v.setAssistantRunStatus("Assistant run is no longer cancellable.")
		v.setAssistantStopState("", false)
		return
	}
	v.setAssistantRunStatus("Cancel requested for " + jobID + ".")
	v.setAssistantStopState(jobID, false)
	v.addActivity("Cancel requested for assistant job " + jobID + ".")
	if v.jobs != nil {
		v.refreshJobs()
	}
}

func (v *View) assistantRunTaskApprovalChecked() bool {
	if v.assistant == nil || v.assistant.runTaskApproval == nil {
		return false
	}
	if v.approvalService == nil {
		return v.assistant.runTaskApproval.Checked
	}
	return v.assistant.runTaskApproval.Checked || v.approvalService.HasFullProjectAccess(v.state.Workspace().Root)
}

func (v *View) confirmAgentToolApproval(ctx context.Context, request agentSvc.ToolApprovalRequest) bool {
	result := make(chan bool, 1)
	fyne.Do(func() {
		dialog.ShowCustomConfirm("Approve agent tool", "Approve once", "Deny", container.NewPadded(agentToolApprovalCard(request)), func(confirm bool) {
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

func agentToolApprovalCard(request agentSvc.ToolApprovalRequest) fyne.CanvasObject {
	message := widget.NewLabel(agentToolApprovalMessage(request))
	message.Wrapping = fyne.TextWrapWord
	return widget.NewCard("Agent tool approval", agentToolApprovalSubtitle(request), message)
}

func agentToolApprovalSubtitle(request agentSvc.ToolApprovalRequest) string {
	parts := []string{firstNonEmpty(request.Name, "unknown tool")}
	if risk := strings.TrimSpace(request.Risk); risk != "" {
		parts = append(parts, "risk: "+risk)
	}
	if target := agentToolApprovalTarget(request); target != "" {
		parts = append(parts, "target: "+target)
	}
	return strings.Join(parts, " - ")
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
		MaxBytes: agentContextBudgetBytes(v.settingsStore, request.ModelRouteID),
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
}, modelRouteID string) int {
	if store == nil {
		return defaultAgentContextMaxBytes
	}
	settings, err := store.Load()
	if err != nil {
		return defaultAgentContextMaxBytes
	}
	if routed, ok := settingsSvc.SettingsForModelRoute(settings, modelRouteID); ok {
		settings = routed
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

type agentEventRenderer struct {
	mu       sync.Mutex
	tail     agentActivityTail
	pending  []string
	render   func(string, []string)
	stop     chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
}

func newAgentEventRenderer(render func(string, []string), interval time.Duration) *agentEventRenderer {
	if interval <= 0 {
		interval = agentEventRefreshInterval
	}
	renderer := &agentEventRenderer{
		render:  render,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
	}
	go renderer.run(interval)
	return renderer
}

func (r *agentEventRenderer) Append(line string) {
	line = strings.TrimSpace(line)
	if r == nil || line == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tail.Add(line)
	r.pending = append(r.pending, line)
}

func (r *agentEventRenderer) Flush() {
	if r == nil {
		return
	}
	markdown, lines, ok := r.consume()
	if ok && r.render != nil {
		r.render(markdown, lines)
	}
}

func (r *agentEventRenderer) Stop() {
	if r == nil {
		return
	}
	r.stopOnce.Do(func() {
		close(r.stop)
		<-r.stopped
	})
}

func (r *agentEventRenderer) run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	defer close(r.stopped)
	for {
		select {
		case <-ticker.C:
			r.Flush()
		case <-r.stop:
			r.Flush()
			return
		}
	}
}

func (r *agentEventRenderer) consume() (string, []string, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.pending) == 0 {
		return "", nil, false
	}
	lines := append([]string{}, r.pending...)
	r.pending = nil
	return r.tail.Markdown(), lines, true
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
		return "Timeline: started - " + firstNonEmpty(event.Message, "Agent started.")
	case "model_request":
		return fmt.Sprintf("Timeline: step %d - asking model for the next action.", event.Iteration)
	case "tool_start":
		return fmt.Sprintf("Timeline: step %d - tool requested: %s.", event.Iteration, firstNonEmpty(event.ToolName, "unknown tool"))
	case "tool_done":
		return fmt.Sprintf("Timeline: step %d - tool completed: %s.", event.Iteration, firstNonEmpty(event.ToolName, "unknown tool"))
	case "tool_error":
		return fmt.Sprintf("Timeline: step %d - tool failed: %s. %s", event.Iteration, firstNonEmpty(event.ToolName, "unknown tool"), firstNonEmpty(event.Error, "No error detail reported."))
	case "plan_update":
		return fmt.Sprintf("Timeline: step %d - plan updated.", event.Iteration)
	case "finalizing":
		return "Timeline: finalizing - wrapping up agent run."
	case "stopped", "error":
		return "Timeline: stopped - " + firstNonEmpty(event.Message, event.Error)
	default:
		return ""
	}
}

func agentFinalMarkdown(result agentSvc.Result) string {
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = "Agent completed without a final message."
	}
	if result.Model != "" {
		message += "\n\nModel: `" + result.Model + "`"
	}
	if result.ModelRoute != "" {
		message += "\n\nModel route: `" + result.ModelRoute + "`"
	}
	if result.RouteWarning != "" {
		message += "\n\nModel route warning: " + result.RouteWarning
	}
	if result.StopReason != "" {
		message += "\n\nStop reason: `" + result.StopReason + "`"
	}
	return message
}

func assistantAgentRunningStatusLine(jobID string, request agentSvc.Request) string {
	route := strings.TrimSpace(request.ModelRouteID)
	if route == "" {
		route = "global fallback"
	}
	sourceCount := len(request.SourcePaths)
	return fmt.Sprintf("Running: Agent job %s. Route: %s. Sources: %d. Writes: %t. Task tool: %t. Timeout: %s.", jobID, route, sourceCount, request.ApproveWrites, request.ApproveShell, formatAgentRunTimeout(agentSvc.EffectiveRunTimeout(request)))
}

func assistantAgentResultStatusLine(jobID string, result agentSvc.Result) string {
	model := strings.TrimSpace(result.Model)
	if model == "" {
		model = "model not reported"
	}
	route := strings.TrimSpace(result.ModelRoute)
	if route == "" {
		route = "global fallback"
	}
	line := fmt.Sprintf("Completed: Agent job %s with %s via %s after %d iteration(s), %d tool call(s).", jobID, model, route, result.Iterations, len(result.ToolCalls))
	if result.StopReason != "" {
		line += " Stop reason: " + result.StopReason + "."
	}
	if result.RouteWarning != "" {
		line += " Route warning: " + result.RouteWarning
	}
	return line
}

func formatAgentRunTimeout(timeout time.Duration) string {
	if timeout <= 0 {
		return "not set"
	}
	if timeout%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(timeout/time.Minute))
	}
	if timeout%time.Second == 0 {
		return fmt.Sprintf("%ds", int(timeout/time.Second))
	}
	return timeout.String()
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
