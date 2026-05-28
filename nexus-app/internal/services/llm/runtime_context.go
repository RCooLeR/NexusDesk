package llm

import (
	"strings"

	settingssvc "nexusdesk/internal/services/settings"
)

func RuntimeContextWindow(model string, runtime *RuntimeStatus) int {
	if runtime == nil {
		return 0
	}
	model = normalizeRuntimeModelName(model)
	if model == "" {
		model = normalizeRuntimeModelName(runtime.SelectedModel)
	}
	if model == "" {
		return 0
	}
	for _, candidate := range runtime.LoadedModels {
		if runtimeModelMatches(candidate.Name, model) || runtimeModelMatches(candidate.Model, model) {
			return candidate.ContextLength
		}
	}
	return 0
}

func SettingsWithRuntimeContext(settings settingssvc.Settings, runtime *RuntimeStatus) settingssvc.Settings {
	runtimeContext := RuntimeContextWindow(settings.Model, runtime)
	if runtimeContext <= 0 || runtimeContext == settings.ContextTokens {
		return settings
	}
	settings.ContextTokens = runtimeContext
	settings.ResponseReserveTokens = settingssvc.ResponseReserveForContext(runtimeContext)
	return settings
}

func runtimeModelMatches(left string, right string) bool {
	return normalizeRuntimeModelName(left) == normalizeRuntimeModelName(right)
}

func normalizeRuntimeModelName(value string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ":latest")
}
