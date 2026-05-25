package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLLMSettingsStoreReturnsDefaultsWhenMissing(t *testing.T) {
	store := NewLLMSettingsStore(filepath.Join(t.TempDir(), "settings.json"))

	settings, err := store.Get()
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if settings.ProviderName == "" {
		t.Fatal("expected default provider name")
	}
	if settings.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("unexpected default base URL: %s", settings.BaseURL)
	}
	if settings.Model != "qwen3:8b" {
		t.Fatalf("unexpected default model: %s", settings.Model)
	}
	if settings.MaxContextTokens != 32768 {
		t.Fatalf("unexpected default context window: %d", settings.MaxContextTokens)
	}
	if settings.ResponseReserveTokens != 4096 {
		t.Fatalf("unexpected default response reserve: %d", settings.ResponseReserveTokens)
	}
}

func TestLLMSettingsStoreSavesAndReadsSettings(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := NewLLMSettingsStore(path)

	saved, err := store.Save(LLMSettings{
		ProviderName:          "Test Provider",
		BaseURL:               "https://example.test/v1",
		Model:                 "test-model",
		APIKey:                "secret",
		MaxContextTokens:      65536,
		ResponseReserveTokens: 8192,
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if saved.UpdatedAt == "" {
		t.Fatal("expected UpdatedAt to be set")
	}

	read, err := store.Get()
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if read.ProviderName != "Test Provider" {
		t.Fatalf("unexpected provider: %s", read.ProviderName)
	}
	if read.BaseURL != "https://example.test/v1" {
		t.Fatalf("unexpected base URL: %s", read.BaseURL)
	}
	if read.Model != "test-model" {
		t.Fatalf("unexpected model: %s", read.Model)
	}
	if read.APIKey != RedactedAPIKey {
		t.Fatal("expected API key to be redacted")
	}
	if read.MaxContextTokens != 65536 || read.ResponseReserveTokens != 8192 {
		t.Fatalf("unexpected context settings: %+v", read)
	}

	rawData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if strings.Contains(string(rawData), "secret") {
		t.Fatal("expected raw settings file to avoid storing the API key")
	}
	raw := readRawLLMSettings(t, path)
	if raw.APIKey != storedAPIKeyReference {
		t.Fatalf("expected settings to reference OS credential storage, got %q", raw.APIKey)
	}
}

func TestLLMSettingsStoreNormalizesContextReserve(t *testing.T) {
	store := NewLLMSettingsStore(filepath.Join(t.TempDir(), "settings.json"))

	saved, err := store.Save(LLMSettings{
		ProviderName:          "Test Provider",
		BaseURL:               "https://example.test/v1",
		MaxContextTokens:      12000,
		ResponseReserveTokens: 12000,
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved.MaxContextTokens != 12000 {
		t.Fatalf("unexpected context tokens: %d", saved.MaxContextTokens)
	}
	if saved.ResponseReserveTokens != 3000 {
		t.Fatalf("expected reserve to be clamped to quarter window, got %d", saved.ResponseReserveTokens)
	}
}

func TestLLMSettingsStorePreservesRedactedAPIKeyOnSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := NewLLMSettingsStore(path)

	if _, err := store.Save(LLMSettings{
		ProviderName: "Test Provider",
		BaseURL:      "https://example.test/v1",
		Model:        "first-model",
		APIKey:       "secret",
	}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	saved, err := store.Save(LLMSettings{
		ProviderName: "Updated Provider",
		BaseURL:      "https://example.test/v1",
		Model:        "second-model",
		APIKey:       RedactedAPIKey,
	})
	if err != nil {
		t.Fatalf("Save with redacted key failed: %v", err)
	}
	if saved.APIKey != RedactedAPIKey {
		t.Fatal("expected saved API key to remain redacted")
	}

	raw := readRawLLMSettings(t, path)
	if raw.APIKey != storedAPIKeyReference {
		t.Fatalf("expected settings to keep secret reference, got %q", raw.APIKey)
	}
	secret, err := store.readAPIKeySecret()
	if err != nil {
		t.Fatalf("readAPIKeySecret failed: %v", err)
	}
	if secret != "secret" {
		t.Fatalf("expected stored secret to be preserved, got %q", secret)
	}
}

func TestLLMSettingsStoreResolvesRedactedAPIKeyForUse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := NewLLMSettingsStore(path)

	if _, err := store.Save(LLMSettings{
		ProviderName: "Test Provider",
		BaseURL:      "https://example.test/v1",
		Model:        "test-model",
		APIKey:       "secret",
	}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	resolved, err := store.ResolveForUse(LLMSettings{
		ProviderName: "Test Provider",
		BaseURL:      "https://example.test/v1",
		Model:        "test-model",
		APIKey:       RedactedAPIKey,
	})
	if err != nil {
		t.Fatalf("ResolveForUse failed: %v", err)
	}
	if resolved.APIKey != "secret" {
		t.Fatalf("expected resolved secret, got %q", resolved.APIKey)
	}
}

func TestLLMSettingsStoreClearsAPIKeyWhenBlankIsSaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store := NewLLMSettingsStore(path)

	if _, err := store.Save(LLMSettings{
		ProviderName: "Test Provider",
		BaseURL:      "https://example.test/v1",
		APIKey:       "secret",
	}); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	saved, err := store.Save(LLMSettings{
		ProviderName: "Test Provider",
		BaseURL:      "https://example.test/v1",
		APIKey:       "",
	})
	if err != nil {
		t.Fatalf("Save blank key failed: %v", err)
	}
	if saved.APIKey != "" {
		t.Fatal("expected blank saved key")
	}

	raw := readRawLLMSettings(t, path)
	if raw.APIKey != "" {
		t.Fatalf("expected stored key to be cleared, got %q", raw.APIKey)
	}
	secret, err := store.readAPIKeySecret()
	if err != nil {
		t.Fatalf("readAPIKeySecret failed: %v", err)
	}
	if secret != "" {
		t.Fatalf("expected credential secret to be cleared, got %q", secret)
	}
}

func TestLLMSettingsStoreRejectsInvalidURL(t *testing.T) {
	store := NewLLMSettingsStore(filepath.Join(t.TempDir(), "settings.json"))

	_, err := store.Save(LLMSettings{
		ProviderName: "Test Provider",
		BaseURL:      "not a url",
	})

	if err == nil {
		t.Fatal("expected invalid URL error")
	}
}

func readRawLLMSettings(t *testing.T, path string) LLMSettings {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	var settings LLMSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	return settings
}
