package shell

import (
	"context"
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
	if existing := v.openTabs[tabState.ID]; existing != nil {
		existing.Content = content
		v.editorTabs.Select(existing)
		return
	}
	tab := container.NewTabItemWithIcon(editorTabTitle(tabState), theme.SettingsIcon(), content)
	v.openTabs[tabState.ID] = tab
	v.tabIDs[tab] = tabState.ID
	v.editorTabs.Append(tab)
	v.editorTabs.Select(tab)
}

func (v *View) newSettingsPanel() fyne.CanvasObject {
	current, err := v.settingsStore.LoadForDisplay()
	if err != nil {
		current = settingsSvc.Defaults()
		v.addActivity("Could not load settings: " + err.Error())
	}
	provider := widget.NewSelect(settingsSvc.ProviderOptions(), nil)
	provider.SetSelected(current.Provider)
	protocol := widget.NewSelect(settingsSvc.ProtocolOptions(), nil)
	protocol.SetSelected(current.Protocol)
	baseURL := widget.NewEntry()
	baseURL.SetText(current.BaseURL)
	provider.OnChanged = func(id string) {
		profile, ok := settingsSvc.ProviderProfileByID(id)
		if !ok {
			return
		}
		protocol.SetSelected(profile.Protocol)
		if strings.TrimSpace(baseURL.Text) == "" || baseURL.Text == settingsSvc.Defaults().BaseURL {
			baseURL.SetText(profile.DefaultBaseURL)
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
	}
	if label, ok := settingsModelLabelForID(current.Model); ok {
		recommendedModel.SetSelected(label)
	}
	if len(routeSelect.Options) > 0 {
		routeSelect.SetSelected(routeSelect.Options[0])
	}
	probeStatus := widget.NewLabel("Connection test has not run.")
	probeStatus.Wrapping = fyne.TextWrapWord
	testConnection := widget.NewButtonWithIcon("Test connection", theme.SearchIcon(), nil)

	form := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("Provider", provider),
			widget.NewFormItem("Protocol", protocol),
			widget.NewFormItem("Base URL", baseURL),
			widget.NewFormItem("Model for chat", model),
			widget.NewFormItem("Recommended model", recommendedModel),
			widget.NewFormItem("API key", apiKey),
			widget.NewFormItem("Context tokens", contextTokens),
			widget.NewFormItem("Response reserve", responseReserve),
			widget.NewFormItem("Task route", routeSelect),
			widget.NewFormItem("Task route model", routeModel),
			widget.NewFormItem("Task route recommended model", routeRecommendedModel),
			widget.NewFormItem("Task route detail", routeDetail),
		},
		OnSubmit: func() {
			next, err := settingsFromFormWithRoutes(provider.Selected, protocol.Selected, baseURL.Text, model.Text, apiKey.Text, contextTokens.Text, responseReserve.Text, modelRoutes)
			if err != nil {
				dialog.ShowError(err, v.window)
				return
			}
			if err := v.settingsStore.Save(next); err != nil {
				dialog.ShowError(err, v.window)
				return
			}
			if display, err := v.settingsStore.LoadForDisplay(); err == nil {
				apiKey.SetText(display.APIKey)
			}
			v.addActivity("Settings saved.")
		},
		SubmitText: "Save",
	}
	testConnection.OnTapped = func() {
		next, err := settingsFromForm(provider.Selected, protocol.Selected, baseURL.Text, model.Text, apiKey.Text, contextTokens.Text, responseReserve.Text)
		if err != nil {
			dialog.ShowError(err, v.window)
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
			result, probeErr := prober.Probe(ctx, llmSvc.ConfigFromSettings(next))
			cancel()
			message := formatSettingsProbeResult(result, probeErr)
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
	actions := container.NewHBox(testConnection)
	secretNote := widget.NewLabel("API keys are stored in protected OS storage where available (Windows DPAPI, macOS Keychain, Linux Secret Service) and displayed redacted after save.")
	secretNote.Wrapping = fyne.TextWrapWord
	return container.NewPadded(container.NewBorder(
		widget.NewLabel("LLM Provider Settings"),
		container.NewVBox(actions, probeStatus, secretNote),
		nil,
		nil,
		form,
	))
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
	if err != nil {
		return "Connection test failed: " + err.Error()
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
	return strings.Join(parts, "\n")
}
