package dbconnector

import (
	"regexp"
	"strings"
)

var (
	connectorURLCredentialPattern   = regexp.MustCompile(`([a-zA-Z][a-zA-Z0-9+.-]*://[^/\s:@]*:)([^@\s/]*)(@)`)
	connectorMySQLCredentialPattern = regexp.MustCompile(`\b([A-Za-z0-9._%+-]+:)([^@\s]+)(@(?:tcp|unix)\()`)
	connectorSensitiveQueryPattern  = regexp.MustCompile(`(?i)([?&](?:password|passwd|pwd|token|access_token|auth_token|api_key|apikey|secret)=)([^&#\s]+)`)
	connectorSensitiveKVPattern     = regexp.MustCompile(`(?i)\b(password|passwd|pwd|token|access_token|auth_token|api_key|apikey|secret)\s*=\s*([^;,\s]+)`)
	connectorSensitiveColonPattern  = regexp.MustCompile(`(?i)\b(password|passwd|pwd|token|access_token|auth_token|api_key|apikey|secret)\s*:\s*('[^']*'|"[^"]*"|[^;,\s]+)`)
	connectorSensitiveJSONPattern   = regexp.MustCompile(`(?i)("(?:password|passwd|pwd|token|access_token|auth_token|api_key|apikey|secret)"\s*:\s*)"[^"]*"`)
	connectorAuthHeaderPattern      = regexp.MustCompile(`(?i)\b(authorization\s*:\s*bearer)\s+[^\s,;]+`)
)

func sanitizeConnectorErrorMessage(message string) string {
	if strings.TrimSpace(message) == "" {
		return message
	}
	redacted := connectorURLCredentialPattern.ReplaceAllString(message, `$1[redacted]$3`)
	redacted = connectorMySQLCredentialPattern.ReplaceAllString(redacted, `$1[redacted]$3`)
	redacted = connectorSensitiveQueryPattern.ReplaceAllString(redacted, `$1[redacted]`)
	redacted = connectorSensitiveKVPattern.ReplaceAllString(redacted, `$1=[redacted]`)
	redacted = connectorSensitiveColonPattern.ReplaceAllString(redacted, `$1: [redacted]`)
	redacted = connectorSensitiveJSONPattern.ReplaceAllString(redacted, `$1"[redacted]"`)
	redacted = connectorAuthHeaderPattern.ReplaceAllString(redacted, `$1 [redacted]`)
	return redacted
}
