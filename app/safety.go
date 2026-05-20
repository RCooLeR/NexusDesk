package main

import (
	"log"

	"NexusDesk/internal/safety"
)

const maxProviderAuditSnippet = 140

var providerFailureAuditf = func(component string, inputBytes int, redacted bool, truncated bool, snippet string) {
	log.Printf("provider_failure component=%q inputBytes=%d redacted=%t truncated=%t snippet=%q", component, inputBytes, redacted, truncated, snippet)
}

func sanitizeProviderMessage(value string) string {
	return safety.SanitizeProviderMessage(value)
}

func sanitizeProviderMessageWithAudit(value string, component string) string {
	result := safety.SanitizeProviderMessageResult(value)
	if result.Redacted || result.Truncated {
		snippet := result.Value
		if len(snippet) > maxProviderAuditSnippet {
			snippet = snippet[:maxProviderAuditSnippet]
		}
		providerFailureAuditf(component, len(value), result.Redacted, result.Truncated, snippet)
	}
	return result.Value
}

func sanitizeQueryForMetadata(sqlText string) string {
	return safety.SanitizeMetadataQuery(sqlText)
}
