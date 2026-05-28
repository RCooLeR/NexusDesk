package settings

import "strings"

type Settings struct {
	Provider              string
	Protocol              string
	BaseURL               string
	Model                 string
	APIKey                string
	ContextTokens         int
	ResponseReserveTokens int
	ModelRoutes           []ModelRoute
}

type ProviderProfile struct {
	ID             string
	Label          string
	Protocol       string
	DefaultBaseURL string
	RequiresAPIKey bool
	RuntimeProbe   bool
}

type ModelOption struct {
	ID               string
	Label            string
	ChatLabel        string
	MaxContextTokens int
}

type ModelRoute struct {
	ID                    string
	Label                 string
	Provider              string
	Protocol              string
	BaseURL               string
	Model                 string
	AlternativeModel      string
	ContextTokens         int
	ResponseReserveTokens int
	CapabilityProfile     string
}

const (
	ProtocolOpenAICompatible       = "openai-compatible"
	ProtocolOllamaOpenAICompatible = "ollama-openai-compatible"
	FallbackModelContextTokens     = 32768

	RouteMainCoding        = "main-coding"
	RouteReactTypeScript   = "react-typescript-javascript"
	RouteGoBackend         = "go-backend"
	RoutePythonCoding      = "python-coding"
	RoutePHPLaravel        = "php-laravel"
	RouteSQL               = "sql"
	RouteNeo4jCypher       = "neo4j-cypher"
	RouteCSVExcelScripts   = "csv-excel-scripts"
	RouteAnalytics         = "analytics-explanations"
	RouteResearchSummaries = "research-summaries"
	RouteVisionScreenshot  = "vision-screenshot"
	RouteBalancedVision    = "balanced-reasoning-vision"
	RouteFastCoding30B     = "fast-30b-coding"
)

func Defaults() Settings {
	return Settings{
		Provider:              "ollama",
		Protocol:              ProtocolOllamaOpenAICompatible,
		BaseURL:               "http://localhost:11434/v1",
		Model:                 "",
		ContextTokens:         32768,
		ResponseReserveTokens: 4096,
	}
}

func ModelOptions() []string {
	recommended := RecommendedModelOptions()
	options := make([]string, 0, len(recommended))
	for _, option := range recommended {
		options = append(options, option.ID)
	}
	return options
}

func RecommendedModelOptions() []ModelOption {
	return []ModelOption{
		{ID: "qwen3:4b-instruct", Label: "Qwen3 4B Instruct - fast local", ChatLabel: "Qwen3 4B", MaxContextTokens: 32768},
		{ID: "qwen3:8b", Label: "Qwen3 8B - balanced", ChatLabel: "Qwen3 8B", MaxContextTokens: 40960},
		{ID: "qwen3.5:9b", Label: "Qwen3.5 9B - workspace chat", ChatLabel: "Qwen3.5 9B", MaxContextTokens: 131072},
		{ID: "qwen3-coder:30b", Label: "Qwen3 Coder 30B - production coding", ChatLabel: "Qwen3 Coder 30B", MaxContextTokens: 131072},
		{ID: "qwen3.6:27b", Label: "Qwen3.6 27B - balanced reasoning and vision", ChatLabel: "Qwen3.6 27B", MaxContextTokens: 131072},
		{ID: "phi4:14b", Label: "Phi-4 14B - reasoning", ChatLabel: "Phi-4 14B", MaxContextTokens: 16384},
		{ID: "phi4-reasoning:14b", Label: "Phi-4 Reasoning 14B - deep reasoning", ChatLabel: "Phi-4 Reasoning", MaxContextTokens: 32768},
		{ID: "gpt-oss:20b", Label: "GPT-OSS 20B - strong general", ChatLabel: "GPT-OSS 20B", MaxContextTokens: 131072},
		{ID: "mistral-small3.2:latest", Label: "Mistral Small 3.2 - long context", ChatLabel: "Mistral Small", MaxContextTokens: 131072},
		{ID: "gemma4:26b", Label: "Gemma 4 26B - max local", ChatLabel: "Gemma 4 26B", MaxContextTokens: 131072},
		{ID: "gemma4:31b", Label: "Gemma 4 31B - analytics and research", ChatLabel: "Gemma 4 31B", MaxContextTokens: 131072},
	}
}

func DefaultModelRoutes() []ModelRoute {
	return []ModelRoute{
		defaultRoute(RouteMainCoding, "Main coding model", "qwen3-coder:30b", "", "coding"),
		defaultRoute(RouteReactTypeScript, "React / TypeScript / JavaScript", "qwen3-coder:30b", "", "coding"),
		defaultRoute(RouteGoBackend, "Golang backend", "qwen3-coder:30b", "", "coding"),
		defaultRoute(RoutePythonCoding, "Python coding", "qwen3-coder:30b", "", "coding"),
		defaultRoute(RoutePHPLaravel, "PHP / Laravel", "qwen3-coder:30b", "", "coding"),
		defaultRoute(RouteSQL, "MySQL / PostgreSQL", "qwen3-coder:30b", "", "database"),
		defaultRoute(RouteNeo4jCypher, "Neo4j / Cypher", "qwen3-coder:30b", "", "database"),
		defaultRoute(RouteCSVExcelScripts, "CSV / Excel data scripts", "qwen3-coder:30b", "", "data-coding"),
		defaultRoute(RouteAnalytics, "Analytics explanations", "gemma4:31b", "", "analytics"),
		defaultRoute(RouteResearchSummaries, "Research / summaries", "gemma4:31b", "", "research"),
		defaultRoute(RouteVisionScreenshot, "Image / screenshot understanding", "gemma4:31b", "qwen3.6:27b", "vision"),
		defaultRoute(RouteBalancedVision, "Balanced coding + reasoning + vision", "qwen3.6:27b", "", "balanced-vision"),
		defaultRoute(RouteFastCoding30B, "Fastest practical 30B-class coding model", "qwen3-coder:30b", "", "coding"),
	}
}

func ModelRouteByID(settings Settings, id string) (ModelRoute, bool) {
	id = strings.TrimSpace(id)
	for _, route := range normalizedModelRoutes(settings.ModelRoutes) {
		if route.ID == id {
			return route, true
		}
	}
	return ModelRoute{}, false
}

func SettingsForModelRoute(settings Settings, id string) (Settings, bool) {
	route, ok := ModelRouteByID(settings, id)
	if !ok {
		return normalized(settings), false
	}
	next := normalized(settings)
	if strings.TrimSpace(route.Provider) != "" {
		next.Provider = route.Provider
	}
	if strings.TrimSpace(route.Protocol) != "" {
		next.Protocol = route.Protocol
	}
	if strings.TrimSpace(route.BaseURL) != "" {
		next.BaseURL = route.BaseURL
	}
	if strings.TrimSpace(route.Model) != "" {
		next.Model = route.Model
	}
	if route.ContextTokens > 0 {
		next.ContextTokens = route.ContextTokens
	}
	if route.ResponseReserveTokens > 0 {
		next.ResponseReserveTokens = route.ResponseReserveTokens
	}
	return normalized(next), true
}

func SettingsWithModelRoute(settings Settings, routeID string, model string) Settings {
	settings = normalized(settings)
	routeID = strings.TrimSpace(routeID)
	if routeID == "" {
		return settings
	}
	model = strings.TrimSpace(model)
	routes := settings.ModelRoutes
	for index, route := range routes {
		if route.ID != routeID {
			continue
		}
		route.Model = model
		if model != "" {
			route.ContextTokens = ModelContextWindow(model)
			route.ResponseReserveTokens = ResponseReserveForContext(route.ContextTokens)
		}
		routes[index] = route
		settings.ModelRoutes = normalizedModelRoutes(routes)
		return settings
	}
	routes = append(routes, ModelRoute{ID: routeID, Model: model})
	settings.ModelRoutes = normalizedModelRoutes(routes)
	return settings
}

func LegacyModelOptions() []string {
	return []string{
		"qwen2.5-coder:7b",
		"qwen2.5-coder:14b",
		"deepseek-coder-v2:16b",
		"mistral-small:24b",
		"phi4:14b",
		"llama3.1:8b",
		"gemma3:12b",
	}
}

func ModelContextWindow(model string) int {
	for _, option := range RecommendedModelOptions() {
		if modelMatches(option.ID, model) {
			return option.MaxContextTokens
		}
	}
	return FallbackModelContextTokens
}

func ResponseReserveForContext(maxContextTokens int) int {
	if maxContextTokens <= 0 {
		return 4096
	}
	reserve := maxContextTokens / 8
	if reserve < 2048 {
		return 2048
	}
	if reserve > 32768 {
		return 32768
	}
	return reserve
}

func SettingsForSelectedModel(settings Settings, model string) Settings {
	settings.Model = model
	settings.ContextTokens = ModelContextWindow(model)
	settings.ResponseReserveTokens = ResponseReserveForContext(settings.ContextTokens)
	return settings
}

func defaultRoute(id string, label string, model string, alternativeModel string, capabilityProfile string) ModelRoute {
	contextTokens := ModelContextWindow(model)
	return ModelRoute{
		ID:                    id,
		Label:                 label,
		Provider:              Defaults().Provider,
		Protocol:              Defaults().Protocol,
		BaseURL:               Defaults().BaseURL,
		Model:                 model,
		AlternativeModel:      alternativeModel,
		ContextTokens:         contextTokens,
		ResponseReserveTokens: ResponseReserveForContext(contextTokens),
		CapabilityProfile:     capabilityProfile,
	}
}

func ProviderOptions() []string {
	profiles := ProviderProfiles()
	options := make([]string, 0, len(profiles))
	for _, profile := range profiles {
		options = append(options, profile.ID)
	}
	return options
}

func ProtocolOptions() []string {
	return []string{ProtocolOllamaOpenAICompatible, ProtocolOpenAICompatible}
}

func ProviderProfiles() []ProviderProfile {
	return []ProviderProfile{
		{
			ID:             "ollama",
			Label:          "Ollama",
			Protocol:       ProtocolOllamaOpenAICompatible,
			DefaultBaseURL: "http://localhost:11434/v1",
			RuntimeProbe:   true,
		},
		{
			ID:             "openai-compatible",
			Label:          "OpenAI-compatible",
			Protocol:       ProtocolOpenAICompatible,
			DefaultBaseURL: "http://localhost:1234/v1",
		},
		{
			ID:             "custom-openai-compatible",
			Label:          "Custom OpenAI-compatible",
			Protocol:       ProtocolOpenAICompatible,
			DefaultBaseURL: "https://api.openai.com/v1",
			RequiresAPIKey: true,
		},
	}
}

func ProviderProfileByID(id string) (ProviderProfile, bool) {
	for _, profile := range ProviderProfiles() {
		if profile.ID == id {
			return profile, true
		}
	}
	return ProviderProfile{}, false
}

func normalizedModelRoutes(routes []ModelRoute) []ModelRoute {
	defaults := DefaultModelRoutes()
	byID := make(map[string]ModelRoute, len(routes))
	for _, route := range routes {
		route.ID = strings.TrimSpace(route.ID)
		if route.ID == "" {
			continue
		}
		byID[route.ID] = route
	}
	normalized := make([]ModelRoute, 0, len(defaults)+len(byID))
	seen := map[string]bool{}
	for _, fallback := range defaults {
		route := fallback
		if override, ok := byID[fallback.ID]; ok {
			route = mergeModelRoute(fallback, override)
		}
		normalized = append(normalized, route)
		seen[route.ID] = true
	}
	for _, route := range routes {
		route.ID = strings.TrimSpace(route.ID)
		if route.ID == "" || seen[route.ID] {
			continue
		}
		normalized = append(normalized, mergeModelRoute(ModelRoute{
			ID:                    route.ID,
			Label:                 route.ID,
			Provider:              Defaults().Provider,
			Protocol:              Defaults().Protocol,
			BaseURL:               Defaults().BaseURL,
			ContextTokens:         FallbackModelContextTokens,
			ResponseReserveTokens: ResponseReserveForContext(FallbackModelContextTokens),
			CapabilityProfile:     "custom",
		}, route))
	}
	return normalized
}

func mergeModelRoute(fallback ModelRoute, override ModelRoute) ModelRoute {
	route := fallback
	if value := strings.TrimSpace(override.Label); value != "" {
		route.Label = value
	}
	if value := strings.TrimSpace(override.Provider); value != "" {
		route.Provider = value
	}
	if value := strings.TrimSpace(override.Protocol); value != "" {
		route.Protocol = value
	}
	if value := strings.TrimSpace(override.BaseURL); value != "" {
		route.BaseURL = value
	}
	if value := strings.TrimSpace(override.Model); value != "" {
		route.Model = value
	}
	if value := strings.TrimSpace(override.AlternativeModel); value != "" {
		route.AlternativeModel = value
	}
	if override.ContextTokens > 0 {
		route.ContextTokens = override.ContextTokens
	} else {
		route.ContextTokens = ModelContextWindow(route.Model)
	}
	if override.ResponseReserveTokens > 0 {
		route.ResponseReserveTokens = override.ResponseReserveTokens
	} else {
		route.ResponseReserveTokens = ResponseReserveForContext(route.ContextTokens)
	}
	if value := strings.TrimSpace(override.CapabilityProfile); value != "" {
		route.CapabilityProfile = value
	}
	if route.ResponseReserveTokens >= route.ContextTokens {
		route.ResponseReserveTokens = route.ContextTokens / 4
	}
	return route
}

func modelMatches(left string, right string) bool {
	return normalizeModelName(left) == normalizeModelName(right)
}

func normalizeModelName(value string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(value)), ":latest")
}
