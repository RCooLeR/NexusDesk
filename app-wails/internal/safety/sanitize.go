package safety

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	maxMetadataTextLen = 4096
	maxErrorTextLen    = 1024
	redactedValue      = "[redacted]"
)

type redactionRule struct {
	pattern *regexp.Regexp
	repl    string
}

var redactionRules = []redactionRule{
	{
		pattern: regexp.MustCompile(`(?i)\b(api[_-]?key|access[_-]?token|refresh[_-]?token|secret|password)\s*[=:]\s*([^\s"',;]+)`),
		repl:    `$1=` + redactedValue,
	},
	{
		pattern: regexp.MustCompile(`(?i)\b(authorization)\s*[:=]\s*"?[A-Za-z0-9._~+/-]+"?`),
		repl:    `$1=` + redactedValue,
	},
	{
		pattern: regexp.MustCompile(`(?i)\b(Bearer)\s+[A-Za-z0-9._~+/-]+`),
		repl:    `$1 ` + redactedValue,
	},
	{
		pattern: regexp.MustCompile(`(?i)([?&](?:api[_-]?key|access[_-]?token|refresh[_-]?token|password|secret|authorization)=)([^&\s#]+)`),
		repl:    `$1` + redactedValue,
	},
	{
		pattern: regexp.MustCompile(`(?i)(\b(?:api[_-]?key|access[_-]?token|refresh[_-]?token|password|secret|authorization)\b\s*[:=]\s*)"[^"]*"`),
		repl:    `$1"` + redactedValue + `"`,
	},
}

func SanitizeProviderMessage(value string) string {
	return sanitizeProviderText(value).Value
}

func SanitizeMetadataQuery(value string) string {
	return sanitizeMetadataText(value).Value
}

func SanitizeLLMError(value string) string {
	return sanitizeProviderText(value).Value
}

func SanitizeLLMErrorResult(value string) SanitizationResult {
	return sanitizeProviderText(value)
}

func ParseProviderJSONError(body []byte) string {
	bodyText := strings.TrimSpace(string(body))
	if bodyText == "" {
		return ""
	}

	var payload struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && strings.TrimSpace(payload.Error.Message) != "" {
		return payload.Error.Message
	}

	return bodyText
}

type SanitizationResult struct {
	Value     string
	Redacted  bool
	Truncated bool
}

func sanitizeText(value string, maxLen int) SanitizationResult {
	sanitized := strings.TrimSpace(value)
	if sanitized == "" {
		return SanitizationResult{}
	}

	original := sanitized
	sanitized = strings.ReplaceAll(sanitized, "\r\n", "\n")
	for _, rule := range redactionRules {
		sanitized = rule.pattern.ReplaceAllString(sanitized, rule.repl)
	}

	if maxLen <= 0 || len(sanitized) <= maxLen {
		return SanitizationResult{
			Value:     sanitized,
			Redacted:  sanitized != original,
			Truncated: false,
		}
	}

	truncated := sanitized[:maxLen]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	if len(truncated) == 0 {
		return SanitizationResult{
			Value:     redactedValue,
			Redacted:  true,
			Truncated: true,
		}
	}
	return SanitizationResult{
		Value:     truncated + "...",
		Redacted:  sanitized != original,
		Truncated: true,
	}
}

func sanitizeProviderText(value string) SanitizationResult {
	return sanitizeText(value, maxErrorTextLen)
}

func sanitizeMetadataText(value string) SanitizationResult {
	return sanitizeText(value, maxMetadataTextLen)
}

func SanitizeProviderMessageResult(value string) SanitizationResult {
	return sanitizeProviderText(value)
}

func SanitizeMetadataQueryResult(value string) SanitizationResult {
	return sanitizeMetadataText(value)
}
