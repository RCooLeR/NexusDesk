package settings

import (
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
