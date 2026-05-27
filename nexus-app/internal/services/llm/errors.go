package llm

import (
	"encoding/json"
	"regexp"
	"strings"
)

const maxProviderErrorBodyBytes = 2048

func providerErrorDetail(body []byte) string {
	raw := strings.TrimSpace(string(body))
	if raw == "" {
		return ""
	}
	detail := parseProviderJSONError(body)
	if detail == "" {
		detail = raw
	}
	return sanitizeProviderError(detail)
}

func parseProviderJSONError(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return firstString(payload["error"], payload["message"], payload["detail"])
}

func firstString(values ...any) string {
	for _, value := range values {
		switch typed := value.(type) {
		case string:
			if trimmed := strings.TrimSpace(typed); trimmed != "" {
				return trimmed
			}
		case map[string]any:
			if nested := firstString(typed["message"], typed["detail"], typed["type"]); nested != "" {
				return nested
			}
		}
	}
	return ""
}

func sanitizeProviderError(value string) string {
	result := strings.TrimSpace(value)
	result = bearerTokenPattern.ReplaceAllString(result, "Bearer [redacted]")
	result = apiKeyPattern.ReplaceAllString(result, "${1}[redacted]")
	result = openAIKeyPattern.ReplaceAllString(result, "sk-[redacted]")
	if len(result) > maxProviderErrorBodyBytes {
		result = result[:maxProviderErrorBodyBytes] + "..."
	}
	return result
}

var (
	bearerTokenPattern = regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]+`)
	apiKeyPattern      = regexp.MustCompile(`(?i)(api[_-]?key\s*[:=]\s*)[^\s,"']+`)
	openAIKeyPattern   = regexp.MustCompile(`sk-[A-Za-z0-9._-]+`)
)
