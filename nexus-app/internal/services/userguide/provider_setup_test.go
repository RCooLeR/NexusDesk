package userguide

import (
	"strings"
	"testing"
)

func TestProviderSetupWizardGuideCoversProviderModelAndVerification(t *testing.T) {
	markdown := ProviderSetupWizardMarkdown()
	for _, expected := range []string{
		"Provider Setup Wizard",
		"Choose Provider And Endpoint",
		"OpenAI-compatible",
		"http://localhost:11434/v1",
		"Select Or Detect Model",
		"suggests the first detected provider model",
		"Save Credentials Safely",
		"protected OS storage",
		"Test connection",
		"Diagnostics",
		"Task Model Routes",
	} {
		if !strings.Contains(markdown, expected) {
			t.Fatalf("expected %q in provider setup wizard markdown:\n%s", expected, markdown)
		}
	}
}
