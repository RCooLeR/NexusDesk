package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLoadsDefaultsWhenMissing(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "settings.json"))

	settings, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if settings.Provider != Defaults().Provider || settings.Model != "" {
		t.Fatalf("expected defaults, got %#v", settings)
	}
}

func TestStoreSavesAndLoadsSettings(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "settings.json"))
	want := Settings{
		Provider:              "openai-compatible",
		Protocol:              ProtocolOpenAICompatible,
		BaseURL:               "http://localhost:1234/v1",
		Model:                 "mistral-small:24b",
		APIKey:                "test-api-key",
		ContextTokens:         8192,
		ResponseReserveTokens: 1024,
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if got != want {
		t.Fatalf("settings mismatch: got %#v want %#v", got, want)
	}
}

func TestStoreNormalizesInvalidTokenReserve(t *testing.T) {
	settings := normalized(Settings{ContextTokens: 1000, ResponseReserveTokens: 1000})
	if settings.ResponseReserveTokens != 250 {
		t.Fatalf("expected reserve to be reduced, got %#v", settings)
	}
}

func TestStoreNormalizesAPIKeyWhitespace(t *testing.T) {
	settings := normalized(Settings{APIKey: "  abc123  "})
	if settings.APIKey != "abc123" {
		t.Fatalf("expected trimmed API key, got %#v", settings)
	}
}

func TestStoreDoesNotInventDefaultModel(t *testing.T) {
	settings := normalized(Settings{Model: "  "})
	if settings.Model != "" {
		t.Fatalf("expected empty model to remain explicit, got %#v", settings)
	}
}

func TestStoreInfersLegacyProviderProtocol(t *testing.T) {
	settings := normalized(Settings{Provider: "ollama", BaseURL: "http://localhost:11434/v1"})
	if settings.Protocol != ProtocolOllamaOpenAICompatible {
		t.Fatalf("expected Ollama protocol inference, got %#v", settings)
	}
	settings = normalized(Settings{Provider: "my-provider", BaseURL: "https://example.com/v1"})
	if settings.Protocol != ProtocolOpenAICompatible {
		t.Fatalf("expected OpenAI-compatible fallback, got %#v", settings)
	}
}

func TestStoreLoadsLegacySettingsWithoutProtocol(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"Provider":"openai-compatible","BaseURL":"https://example.com/v1","Model":"model"}`), 0o600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	settings, err := NewFileStore(path).Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if settings.Protocol != ProtocolOpenAICompatible {
		t.Fatalf("expected legacy settings to infer OpenAI-compatible protocol, got %#v", settings)
	}
}

func TestProviderProfilesExposeExtensibleOptions(t *testing.T) {
	profile, ok := ProviderProfileByID("custom-openai-compatible")
	if !ok || profile.Protocol != ProtocolOpenAICompatible || !profile.RequiresAPIKey {
		t.Fatalf("expected custom OpenAI-compatible profile, got %#v ok=%v", profile, ok)
	}
	if len(ProtocolOptions()) < 2 {
		t.Fatalf("expected explicit protocol options")
	}
}
