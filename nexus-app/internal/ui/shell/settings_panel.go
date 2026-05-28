package shell

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

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
	current, err := v.settingsStore.Load()
	if err != nil {
		current = settingsSvc.Defaults()
		v.addActivity("Could not load settings: " + err.Error())
	}
	provider := widget.NewSelect(settingsSvc.ProviderOptions(), nil)
	provider.SetSelected(current.Provider)
	baseURL := widget.NewEntry()
	baseURL.SetText(current.BaseURL)
	model := widget.NewSelect(settingsSvc.ModelOptions(), nil)
	model.SetSelected(current.Model)
	apiKey := widget.NewPasswordEntry()
	apiKey.SetText(current.APIKey)
	contextTokens := widget.NewEntry()
	contextTokens.SetText(strconv.Itoa(current.ContextTokens))
	responseReserve := widget.NewEntry()
	responseReserve.SetText(strconv.Itoa(current.ResponseReserveTokens))

	form := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem("Provider", provider),
			widget.NewFormItem("Base URL", baseURL),
			widget.NewFormItem("Model", model),
			widget.NewFormItem("API key", apiKey),
			widget.NewFormItem("Context tokens", contextTokens),
			widget.NewFormItem("Response reserve", responseReserve),
		},
		OnSubmit: func() {
			next, err := settingsFromForm(provider.Selected, baseURL.Text, model.Selected, apiKey.Text, contextTokens.Text, responseReserve.Text)
			if err != nil {
				dialog.ShowError(err, v.window)
				return
			}
			if err := v.settingsStore.Save(next); err != nil {
				dialog.ShowError(err, v.window)
				return
			}
			v.addActivity("Settings saved.")
		},
		SubmitText: "Save",
	}
	return container.NewPadded(container.NewBorder(widget.NewLabel("LLM Provider Settings"), nil, nil, nil, form))
}

func settingsFromForm(provider string, baseURL string, model string, apiKey string, contextTokensValue string, responseReserveValue string) (settingsSvc.Settings, error) {
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
		BaseURL:               baseURL,
		Model:                 model,
		APIKey:                apiKey,
		ContextTokens:         contextTokens,
		ResponseReserveTokens: responseReserve,
	}, nil
}
