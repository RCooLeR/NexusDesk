package shell

import "testing"

func TestSettingsFromFormParsesTokens(t *testing.T) {
	settings, err := settingsFromForm("ollama", "http://localhost:11434/v1", "qwen2.5-coder:14b", "32768", "4096")
	if err != nil {
		t.Fatalf("settingsFromForm returned error: %v", err)
	}
	if settings.ContextTokens != 32768 || settings.ResponseReserveTokens != 4096 {
		t.Fatalf("unexpected settings: %#v", settings)
	}
}

func TestSettingsFromFormRejectsInvalidTokens(t *testing.T) {
	if _, err := settingsFromForm("ollama", "http://localhost:11434/v1", "qwen2.5-coder:14b", "bad", "4096"); err == nil {
		t.Fatal("expected invalid context tokens to fail")
	}
}
