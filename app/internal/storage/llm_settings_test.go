package storage

import (
	"path/filepath"
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
}

func TestLLMSettingsStoreSavesAndReadsSettings(t *testing.T) {
	store := NewLLMSettingsStore(filepath.Join(t.TempDir(), "settings.json"))

	saved, err := store.Save(LLMSettings{
		ProviderName: "Test Provider",
		BaseURL:      "https://example.test/v1",
		Model:        "test-model",
		APIKey:       "secret",
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
	if read.APIKey != "secret" {
		t.Fatal("expected API key to round-trip")
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
