package shell

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	llmSvc "nexusdesk/internal/services/llm"
	settingsSvc "nexusdesk/internal/services/settings"
)

func (v *View) openSettingsTab() {
	tabState := v.editorSession.OpenPlaceholder("Settings")
	content := v.newSettingsPanel()
	if existing := v.editor.openTabs[tabState.ID]; existing != nil {
		existing.Content = content
		v.editor.tabs.Select(existing)
		return
	}
	tab := container.NewTabItemWithIcon(editorTabTitle(tabState), theme.SettingsIcon(), content)
	v.editor.openTabs[tabState.ID] = tab
	v.editor.tabIDs[tab] = tabState.ID
	v.editor.tabs.Append(tab)
	v.editor.tabs.Select(tab)
}

type settingsPanelSection struct {
	Title    string
	Summary  string
	Keywords []string
	Content  fyne.CanvasObject
}

func (v *View) newSettingsPanel() fyne.CanvasObject {
	current, err := v.settingsStore.LoadForDisplay()
	if err != nil {
		current = settingsSvc.Defaults()
		v.addActivity("Could not load settings: " + err.Error())
	}
	if len(current.ModelRoutes) == 0 {
		current.ModelRoutes = settingsSvc.DefaultModelRoutes()
	}
	provider := widget.NewSelect(settingsSvc.ProviderOptions(), nil)
	provider.SetSelected(current.Provider)
	protocol := widget.NewSelect(settingsSvc.ProtocolOptions(), nil)
	protocol.SetSelected(current.Protocol)
	baseURL := widget.NewEntry()
	baseURL.SetText(current.BaseURL)
	var refreshValidation func()
	provider.OnChanged = func(id string) {
		profile, ok := settingsSvc.ProviderProfileByID(id)
		if !ok {
			return
		}
		protocol.SetSelected(profile.Protocol)
		if strings.TrimSpace(baseURL.Text) == "" || baseURL.Text == settingsSvc.Defaults().BaseURL {
			baseURL.SetText(profile.DefaultBaseURL)
		}
		if refreshValidation != nil {
			refreshValidation()
		}
	}
	model := widget.NewEntry()
	model.SetText(current.Model)
	model.SetPlaceHolder("Choose from Test connection or type a model ID")
	recommendedModel := widget.NewSelect(settingsModelOptionLabels(), nil)
	apiKey := widget.NewPasswordEntry()
	apiKey.SetText(current.APIKey)
	contextTokens := widget.NewEntry()
	contextTokens.SetText(strconv.Itoa(current.ContextTokens))
	responseReserve := widget.NewEntry()
	responseReserve.SetText(strconv.Itoa(current.ResponseReserveTokens))
	modelRoutes := current.ModelRoutes
	selectedRouteID := ""
	routeSelect := widget.NewSelect(settingsRouteOptionLabels(modelRoutes), nil)
	routeModel := widget.NewEntry()
	routeModel.SetPlaceHolder("Model ID for the selected task route")
	routeRecommendedModel := widget.NewSelect(settingsModelOptionLabels(), nil)
	routeDetail := widget.NewLabel("Task model defaults are saved for future route-aware workflows. The global chat model above remains the current fallback.")
	routeDetail.Wrapping = fyne.TextWrapWord
	routeModel.OnChanged = func(value string) {
		if selectedRouteID == "" {
			return
		}
		modelRoutes = settingsModelRoutesWithModel(modelRoutes, selectedRouteID, value)
		routeDetail.SetText(settingsRouteDetail(modelRoutes, selectedRouteID))
		if refreshValidation != nil {
			refreshValidation()
		}
	}
	routeRecommendedModel.OnChanged = func(label string) {
		option, ok := settingsModelOptionByLabel(label)
		if !ok {
			return
		}
		routeModel.SetText(option.ID)
	}
	routeSelect.OnChanged = func(label string) {
		route, ok := settingsRouteByLabel(modelRoutes, label)
		if !ok {
			return
		}
		selectedRouteID = route.ID
		routeModel.SetText(route.Model)
		if recommendedLabel, ok := settingsModelLabelForID(route.Model); ok {
			routeRecommendedModel.SetSelected(recommendedLabel)
		} else {
			routeRecommendedModel.ClearSelected()
		}
		routeDetail.SetText(settingsRouteDetail(modelRoutes, selectedRouteID))
	}
	recommendedModel.OnChanged = func(label string) {
		option, ok := settingsModelOptionByLabel(label)
		if !ok {
			return
		}
		model.SetText(option.ID)
		contextTokens.SetText(strconv.Itoa(option.MaxContextTokens))
		responseReserve.SetText(strconv.Itoa(settingsSvc.ResponseReserveForContext(option.MaxContextTokens)))
		if refreshValidation != nil {
			refreshValidation()
		}
	}
	if label, ok := settingsModelLabelForID(current.Model); ok {
		recommendedModel.SetSelected(label)
	}
	if len(routeSelect.Options) > 0 {
		routeSelect.SetSelected(routeSelect.Options[0])
	}
	for _, entry := range []*widget.Entry{baseURL, model, contextTokens, responseReserve} {
		entry.OnChanged = func(string) {
			if refreshValidation != nil {
				refreshValidation()
			}
		}
	}
	protocol.OnChanged = func(string) {
		if refreshValidation != nil {
			refreshValidation()
		}
	}
	probeStatus := widget.NewLabel("Connection test has not run.")
	probeStatus.Wrapping = fyne.TextWrapWord
	testConnection := widget.NewButtonWithIcon("Test connection", theme.SearchIcon(), nil)
	saveSettings := widget.NewButtonWithIcon("Save settings", theme.DocumentSaveIcon(), nil)
	saveSettings.Importance = widget.HighImportance
	validationStatus := widget.NewLabel("")
	validationStatus.Wrapping = fyne.TextWrapWord
	refreshValidation = func() {
		validationStatus.SetText(settingsValidationText(settingsValidationIssues(
			provider.Selected,
			protocol.Selected,
			baseURL.Text,
			model.Text,
			contextTokens.Text,
			responseReserve.Text,
			modelRoutes,
		)))
	}

	providerForm := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("Provider", provider),
			widget.NewFormItem("Protocol", protocol),
			widget.NewFormItem("Base URL", baseURL),
			widget.NewFormItem("Model for chat", model),
			widget.NewFormItem("Recommended model", recommendedModel),
			widget.NewFormItem("Context tokens", contextTokens),
			widget.NewFormItem("Response reserve", responseReserve),
		},
	}
	securityForm := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("API key", apiKey),
		},
	}
	routeForm := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("Task route", routeSelect),
			widget.NewFormItem("Task route model", routeModel),
			widget.NewFormItem("Task route recommended model", routeRecommendedModel),
			widget.NewFormItem("Task route detail", routeDetail),
		},
	}
	saveSettings.OnTapped = func() {
		if err := settingsBlockingValidationError(settingsValidationIssues(provider.Selected, protocol.Selected, baseURL.Text, model.Text, contextTokens.Text, responseReserve.Text, modelRoutes)); err != nil {
			dialog.ShowError(err, v.window)
			refreshValidation()
			return
		}
		next, err := settingsFromFormWithRoutes(provider.Selected, protocol.Selected, baseURL.Text, model.Text, apiKey.Text, contextTokens.Text, responseReserve.Text, modelRoutes)
		if err != nil {
			dialog.ShowError(err, v.window)
			refreshValidation()
			return
		}
		if err := v.settingsStore.Save(next); err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		if display, err := v.settingsStore.LoadForDisplay(); err == nil {
			apiKey.SetText(display.APIKey)
		}
		refreshValidation()
		v.refreshStatusBar()
		v.addActivity("Settings saved.")
	}
	testConnection.OnTapped = func() {
		if err := settingsBlockingValidationError(settingsValidationIssues(provider.Selected, protocol.Selected, baseURL.Text, model.Text, contextTokens.Text, responseReserve.Text, modelRoutes)); err != nil {
			dialog.ShowError(err, v.window)
			refreshValidation()
			return
		}
		next, err := settingsFromForm(provider.Selected, protocol.Selected, baseURL.Text, model.Text, apiKey.Text, contextTokens.Text, responseReserve.Text)
		if err != nil {
			dialog.ShowError(err, v.window)
			refreshValidation()
			return
		}
		next, err = v.settingsStore.ResolveForUse(next)
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		testConnection.Disable()
		probeStatus.SetText("Testing provider connection...")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			prober := v.diagnosticsProber
			if prober == nil {
				prober = llmSvc.NewClient()
			}
			probeConfig := llmSvc.ConfigFromSettings(next)
			result, probeErr := prober.Probe(ctx, probeConfig)
			cancel()
			message := formatSettingsProbeResultWithConfig(probeConfig, result, probeErr)
			fyne.Do(func() {
				if probeErr == nil {
					tuned := llmSvc.SettingsWithRuntimeContext(next, result.Runtime)
					if tuned.ContextTokens != next.ContextTokens {
						contextTokens.SetText(strconv.Itoa(tuned.ContextTokens))
						responseReserve.SetText(strconv.Itoa(tuned.ResponseReserveTokens))
						message += "\nUpdated context tokens from loaded model runtime."
					}
				}
				probeStatus.SetText(message)
				testConnection.Enable()
			})
		}()
	}
	actions := container.NewHBox(saveSettings, testConnection)
	secretNote := widget.NewLabel("API keys are stored in protected OS storage where available (Windows DPAPI, macOS Keychain, Linux Secret Service) and displayed redacted after save.")
	secretNote.Wrapping = fyne.TextWrapWord
	sectionContainer := container.NewVBox()
	search := widget.NewEntry()
	search.SetPlaceHolder("Search settings (provider, API key, route, context...)")
	sections := []settingsPanelSection{
		{
			Title:    "Provider & Runtime",
			Summary:  "Configure the OpenAI-compatible or Ollama endpoint, fallback chat model, and context budget.",
			Keywords: []string{"provider", "protocol", "base url", "model", "recommended", "context", "tokens", "reserve", "ollama", "openai"},
			Content:  providerForm,
		},
		{
			Title:    "Secrets & Credentials",
			Summary:  "Store provider API keys through protected OS storage and keep display values redacted.",
			Keywords: []string{"api key", "credential", "secret", "dpapi", "keychain", "linux", "security"},
			Content:  container.NewVBox(securityForm, secretNote),
		},
		{
			Title:    "Task Model Routes",
			Summary:  "Choose default models for coding, data, research, vision, and balanced reasoning workflows.",
			Keywords: []string{"route", "task", "coding", "data", "database", "research", "vision", "screenshot", "analytics", "balanced"},
			Content:  routeForm,
		},
	}
	var applySearch func(string)
	applySearch = func(query string) {
		sectionContainer.Objects = nil
		matches := 0
		for _, section := range sections {
			if !settingsSectionMatches(section, query) {
				continue
			}
			matches++
			sectionContainer.Add(widget.NewCard(section.Title, section.Summary, section.Content))
		}
		if matches == 0 {
			sectionContainer.Add(widget.NewLabel("No matching settings sections. Try provider, model, API key, route, context, or vision."))
		}
		sectionContainer.Refresh()
	}
	search.OnChanged = applySearch
	refreshValidation()
	applySearch("")
	header := container.NewVBox(
		widget.NewLabelWithStyle("Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		search,
		widget.NewCard("Readiness", "Inline validation for provider, model, token budget, and route defaults.", validationStatus),
	)
	footer := container.NewVBox(actions, probeStatus)
	return container.NewPadded(container.NewBorder(header, footer, nil, nil, container.NewVScroll(sectionContainer)))
}

type settingsValidationIssue struct {
	Severity string
	Message  string
}

func settingsSectionMatches(section settingsPanelSection, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	haystack := []string{section.Title, section.Summary}
	haystack = append(haystack, section.Keywords...)
	for _, value := range haystack {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}

func settingsVisibleSectionTitles(sections []settingsPanelSection, query string) []string {
	titles := make([]string, 0, len(sections))
	for _, section := range sections {
		if settingsSectionMatches(section, query) {
			titles = append(titles, section.Title)
		}
	}
	return titles
}

func settingsValidationIssues(provider string, protocol string, baseURL string, model string, contextTokensValue string, responseReserveValue string, modelRoutes []settingsSvc.ModelRoute) []settingsValidationIssue {
	issues := []settingsValidationIssue{}
	if strings.TrimSpace(provider) == "" {
		issues = append(issues, settingsValidationIssue{Severity: "error", Message: "Provider is required."})
	}
	if strings.TrimSpace(protocol) == "" {
		issues = append(issues, settingsValidationIssue{Severity: "error", Message: "Protocol is required."})
	}
	if strings.TrimSpace(baseURL) == "" {
		issues = append(issues, settingsValidationIssue{Severity: "error", Message: "Base URL is required."})
	}
	if strings.TrimSpace(model) == "" {
		issues = append(issues, settingsValidationIssue{Severity: "warning", Message: "Global chat model is not selected; Ask/Agent will need a route or manual model before provider calls."})
	}
	contextTokens, contextErr := strconv.Atoi(strings.TrimSpace(contextTokensValue))
	if contextErr != nil || contextTokens <= 0 {
		issues = append(issues, settingsValidationIssue{Severity: "error", Message: "Context tokens must be a positive integer."})
	}
	responseReserve, reserveErr := strconv.Atoi(strings.TrimSpace(responseReserveValue))
	if reserveErr != nil || responseReserve <= 0 {
		issues = append(issues, settingsValidationIssue{Severity: "error", Message: "Response reserve must be a positive integer."})
	}
	if contextErr == nil && reserveErr == nil && contextTokens > 0 && responseReserve > 0 && responseReserve >= contextTokens {
		issues = append(issues, settingsValidationIssue{Severity: "error", Message: "Response reserve must be smaller than the context window."})
	}
	missingRoutes := 0
	for _, route := range modelRoutes {
		if strings.TrimSpace(route.Model) == "" {
			missingRoutes++
		}
	}
	if missingRoutes > 0 {
		issues = append(issues, settingsValidationIssue{Severity: "warning", Message: fmt.Sprintf("%d task model route(s) have no default model.", missingRoutes)})
	}
	return issues
}

func settingsValidationText(issues []settingsValidationIssue) string {
	if len(issues) == 0 {
		return "Settings look ready. Save changes, then run Test connection to verify the provider runtime."
	}
	lines := make([]string, 0, len(issues)+1)
	lines = append(lines, "Settings need attention:")
	for _, issue := range issues {
		severity := strings.ToUpper(strings.TrimSpace(issue.Severity))
		if severity == "" {
			severity = "INFO"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", severity, issue.Message))
	}
	return strings.Join(lines, "\n")
}

func settingsBlockingValidationError(issues []settingsValidationIssue) error {
	messages := []string{}
	for _, issue := range issues {
		if strings.EqualFold(strings.TrimSpace(issue.Severity), "error") {
			messages = append(messages, issue.Message)
		}
	}
	if len(messages) == 0 {
		return nil
	}
	return errors.New(strings.Join(messages, " "))
}

func settingsModelOptionLabels() []string {
	options := settingsSvc.RecommendedModelOptions()
	labels := make([]string, 0, len(options))
	for _, option := range options {
		labels = append(labels, option.Label)
	}
	return labels
}

func settingsModelOptionByLabel(label string) (settingsSvc.ModelOption, bool) {
	for _, option := range settingsSvc.RecommendedModelOptions() {
		if option.Label == label {
			return option, true
		}
	}
	return settingsSvc.ModelOption{}, false
}

func settingsModelLabelForID(model string) (string, bool) {
	for _, option := range settingsSvc.RecommendedModelOptions() {
		if settingsModelIDMatches(option.ID, model) {
			return option.Label, true
		}
	}
	return "", false
}

func settingsRouteOptionLabels(routes []settingsSvc.ModelRoute) []string {
	labels := make([]string, 0, len(routes))
	for _, route := range routes {
		labels = append(labels, route.Label)
	}
	return labels
}

func settingsRouteByLabel(routes []settingsSvc.ModelRoute, label string) (settingsSvc.ModelRoute, bool) {
	for _, route := range routes {
		if route.Label == label {
			return route, true
		}
	}
	return settingsSvc.ModelRoute{}, false
}

func settingsModelRoutesWithModel(routes []settingsSvc.ModelRoute, routeID string, model string) []settingsSvc.ModelRoute {
	settings := settingsSvc.Settings{ModelRoutes: append([]settingsSvc.ModelRoute(nil), routes...)}
	settings = settingsSvc.SettingsWithModelRoute(settings, routeID, model)
	return settings.ModelRoutes
}

func settingsRouteDetail(routes []settingsSvc.ModelRoute, routeID string) string {
	for _, route := range routes {
		if route.ID != routeID {
			continue
		}
		parts := []string{
			"Capability: " + firstNonEmptyString(route.CapabilityProfile, "custom"),
			fmt.Sprintf("Context: %d tokens", route.ContextTokens),
			fmt.Sprintf("Reserve: %d tokens", route.ResponseReserveTokens),
			"Provider: " + firstNonEmptyString(route.Provider, "global fallback"),
		}
		if strings.TrimSpace(route.AlternativeModel) != "" {
			parts = append(parts, "Alternative: "+route.AlternativeModel)
		}
		return strings.Join(parts, "\n")
	}
	return "Select a task route to edit its default model."
}

func settingsModelIDMatches(left string, right string) bool {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(left)), ":latest") == strings.TrimSuffix(strings.ToLower(strings.TrimSpace(right)), ":latest")
}

func settingsFromForm(provider string, protocol string, baseURL string, model string, apiKey string, contextTokensValue string, responseReserveValue string) (settingsSvc.Settings, error) {
	return settingsFromFormWithRoutes(provider, protocol, baseURL, model, apiKey, contextTokensValue, responseReserveValue, nil)
}

func settingsFromFormWithRoutes(provider string, protocol string, baseURL string, model string, apiKey string, contextTokensValue string, responseReserveValue string, modelRoutes []settingsSvc.ModelRoute) (settingsSvc.Settings, error) {
	contextTokens, err := strconv.Atoi(contextTokensValue)
	if err != nil {
		return settingsSvc.Settings{}, err
	}
	responseReserve, err := strconv.Atoi(responseReserveValue)
	if err != nil {
		return settingsSvc.Settings{}, err
	}
	return settingsSvc.Settings{
		Provider:              provider,
		Protocol:              protocol,
		BaseURL:               baseURL,
		Model:                 model,
		APIKey:                apiKey,
		ContextTokens:         contextTokens,
		ResponseReserveTokens: responseReserve,
		ModelRoutes:           modelRoutes,
	}, nil
}

func formatSettingsProbeResult(result llmSvc.ProbeResult, err error) string {
	return formatSettingsProbeResultWithConfig(llmSvc.Config{}, result, err)
}

func formatSettingsProbeResultWithConfig(config llmSvc.Config, result llmSvc.ProbeResult, err error) string {
	if err != nil {
		parts := []string{"Connection test failed: " + err.Error()}
		if guidance := llmSvc.ProviderGuidance(config, result, err); len(guidance) > 0 {
			parts = append(parts, "Guidance: "+strings.Join(guidance, "; "))
		}
		return strings.Join(parts, "\n")
	}
	parts := []string{strings.TrimSpace(result.Message)}
	if parts[0] == "" {
		if result.OK {
			parts[0] = "Connected to provider."
		} else {
			parts[0] = "Provider test did not succeed."
		}
	}
	if result.Endpoint != "" {
		parts = append(parts, "Endpoint: "+result.Endpoint)
	}
	if protocol := strings.TrimSpace(result.Protocol); protocol != "" {
		parts = append(parts, "Protocol: "+protocol)
	}
	if result.ModelCount > 0 {
		line := fmt.Sprintf("Models: %d", result.ModelCount)
		if len(result.ModelSample) > 0 {
			line += " (" + strings.Join(result.ModelSample, ", ") + ")"
		}
		parts = append(parts, line)
	}
	if len(result.Capabilities) > 0 {
		parts = append(parts, "Capabilities: "+strings.Join(result.Capabilities, ", "))
	}
	if result.Runtime != nil && strings.TrimSpace(result.Runtime.Message) != "" {
		parts = append(parts, "Runtime: "+result.Runtime.Message)
	}
	if runtimeContext := llmSvc.RuntimeContextWindow("", result.Runtime); runtimeContext > 0 {
		parts = append(parts, fmt.Sprintf("Runtime context: %d tokens", runtimeContext))
	}
	if len(result.Warnings) > 0 {
		parts = append(parts, "Warnings: "+strings.Join(result.Warnings, "; "))
	}
	if guidance := llmSvc.ProviderGuidance(config, result, err); len(guidance) > 0 {
		parts = append(parts, "Guidance: "+strings.Join(guidance, "; "))
	}
	return strings.Join(parts, "\n")
}
