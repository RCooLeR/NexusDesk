package safety

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestSanitizeProviderMessageTruncatesAndRedacts(t *testing.T) {
	input := "call failed with api_key=super-secret-key because secret=abcd1234; message too long: " + strings.Repeat("x", 5000)
	result := SanitizeProviderMessage(input)

	if strings.Contains(result, "super-secret-key") || strings.Contains(result, "abcd1234") {
		t.Fatalf("expected secret material to be redacted, got %q", result)
	}
	if !strings.Contains(result, "[redacted]") {
		t.Fatalf("expected redacted marker, got %q", result)
	}
	if len(result) > 1024+3 {
		t.Fatalf("expected max-length enforcement with truncation, got len=%d", len(result))
	}
}

func TestSanitizeMetadataQueryPreservesUTF8AndPreservesLength(t *testing.T) {
	query := "SELECT '" + string('\u03C0') + string('\u03C0') + string('\u03C0') + "' FROM logs WHERE api_key=abc123; " + strings.Repeat("data ", 2500)
	got := SanitizeMetadataQuery(query)

	if !utf8.ValidString(got) {
		t.Fatalf("expected UTF-8 output, got %q", got)
	}
	if strings.Contains(got, "abc123") {
		t.Fatalf("expected sensitive value to be redacted, got %q", got)
	}
	if len(got) > 4096+3 {
		t.Fatalf("expected metadata sanitization cap, got %d", len(got))
	}
}

func TestParseProviderJSONErrorHandlesAlternatePayload(t *testing.T) {
	body := []byte(`{"error": {"message":"permission denied: api_key=xyz"}, "other": true}`)
	got := ParseProviderJSONError(body)
	if !strings.Contains(got, "api_key") {
		t.Fatalf("expected parsed JSON message, got %q", got)
	}
}

func TestSanitizeProviderMessageResultCapturesFlags(t *testing.T) {
	input := "api_key=abcdef"
	got := SanitizeProviderMessageResult(input)
	if !got.Redacted {
		t.Fatalf("expected redacted=true for token-bearing string")
	}
	if got.Truncated {
		t.Fatalf("expected no truncation for short input")
	}
	if got.Value != SanitizeProviderMessage(input) {
		t.Fatalf("mismatched sanitization result, got %q vs %q", got.Value, SanitizeProviderMessage(input))
	}
}
