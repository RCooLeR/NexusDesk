package main

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeProviderMessageRedactsSensitiveData(t *testing.T) {
	input := "error api_key=top-secret-key; authorization=abc123xyz"
	got := sanitizeProviderMessage(input)

	if strings.Contains(got, "top-secret-key") {
		t.Fatalf("expected api key to be redacted, got %q", got)
	}
	if strings.Contains(got, "abc123xyz") {
		t.Fatalf("expected bearer token to be redacted, got %q", got)
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("expected redacted marker, got %q", got)
	}
}

func TestSanitizeQueryForMetadataTruncatesAndPreservesUTF8(t *testing.T) {
	query := "SELECT * FROM logs WHERE message LIKE '" + string('\u03C0') + string('\u03C0') + string('\u03C0') + "' UNION ALL SELECT 1; " + strings.Repeat("data ", 3000)
	got := sanitizeQueryForMetadata(query)

	if !utf8.ValidString(got) {
		t.Fatalf("expected valid UTF-8 after sanitize, got %q", got)
	}
	if len(got) > 4099 {
		t.Fatalf("expected truncated/sanitized query, got len=%d", len(got))
	}
}

func TestSanitizeQueryForMetadataHandlesWhitespaceAndEmptyInput(t *testing.T) {
	if got := sanitizeQueryForMetadata("   \t  "); got != "" {
		t.Fatalf("expected empty sanitize result, got %q", got)
	}
}

func TestSanitizeProviderMessageWithAudit(t *testing.T) {
	var logOutput bytes.Buffer
	originalAuditf := providerFailureAuditf
	providerFailureAuditf = func(component string, inputBytes int, redacted bool, truncated bool, snippet string) {
		logOutput.WriteString(component)
		logOutput.WriteString(" redacted=")
		logOutput.WriteString(strconv.FormatBool(redacted))
		logOutput.WriteString(" truncated=")
		logOutput.WriteString(strconv.FormatBool(truncated))
		logOutput.WriteString(" bytes=")
		logOutput.WriteString(strconv.Itoa(inputBytes))
		logOutput.WriteString(" snippet=")
		logOutput.WriteString(snippet)
	}
	defer func() {
		providerFailureAuditf = originalAuditf
	}()

	got := sanitizeProviderMessageWithAudit("normal message", "ask_llm_test")
	if got != "normal message" {
		t.Fatalf("expected unchanged message, got %q", got)
	}
	if logOutput.Len() != 0 {
		t.Fatalf("expected no audit for unchanged message, got %q", logOutput.String())
	}

	logOutput.Reset()
	got = sanitizeProviderMessageWithAudit("api_key=secret", "ask_llm_test")
	if got == "api_key=secret" {
		t.Fatalf("expected redacted message, got %q", got)
	}
	if !strings.Contains(logOutput.String(), "redacted=true") {
		t.Fatalf("expected redaction in audit log, got %q", logOutput.String())
	}
}
