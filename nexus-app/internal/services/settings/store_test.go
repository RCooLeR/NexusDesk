package settings

import (
	"nexusdesk/internal/services/protectedsecret"
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

func TestStoreSavesAPIKeyInProtectedSidecar(t *testing.T) {
	requireProtectedSettingsSecretStorage(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	store := NewFileStore(path)

	if err := store.Save(Settings{
		Provider:              "custom-openai-compatible",
		Protocol:              ProtocolOpenAICompatible,
		BaseURL:               "https://api.example.test/v1",
		Model:                 "model-a",
		APIKey:                "test-api-key",
		ContextTokens:         8192,
		ResponseReserveTokens: 1024,
	}); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "test-api-key") {
		t.Fatalf("public settings file must not contain the API key: %s", string(raw))
	}
	if !strings.Contains(string(raw), storedAPIKeyReference) {
		t.Fatalf("expected stored API key reference in settings file: %s", string(raw))
	}
	display, err := store.LoadForDisplay()
	if err != nil {
		t.Fatalf("LoadForDisplay returned error: %v", err)
	}
	if display.APIKey != RedactedAPIKey {
		t.Fatalf("expected redacted display API key, got %#v", display)
	}
	resolved, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if resolved.APIKey != "test-api-key" {
		t.Fatalf("expected resolved API key, got %#v", resolved)
	}
}

func TestStorePreservesRedactedAPIKeyOnSave(t *testing.T) {
	requireProtectedSettingsSecretStorage(t)
	store := NewFileStore(filepath.Join(t.TempDir(), "settings.json"))
	if err := store.Save(Settings{BaseURL: "https://api.example.test/v1", APIKey: "secret-one"}); err != nil {
		t.Fatalf("initial Save returned error: %v", err)
	}
	if err := store.Save(Settings{BaseURL: "https://api.example.test/v2", APIKey: RedactedAPIKey}); err != nil {
		t.Fatalf("redacted Save returned error: %v", err)
	}
	resolved, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if resolved.APIKey != "secret-one" || resolved.BaseURL != "https://api.example.test/v2" {
		t.Fatalf("expected preserved secret and updated base URL, got %#v", resolved)
	}
}

func TestStoreRejectsAPIKeyWhenProtectedStorageUnsupported(t *testing.T) {
	if protectedsecret.Available() {
		t.Skip("unsupported-platform refusal requires a platform without protected secret storage")
	}
	store := NewFileStore(filepath.Join(t.TempDir(), "settings.json"))
	err := store.Save(Settings{BaseURL: "https://api.example.test/v1", APIKey: "secret"})
	if err == nil || !strings.Contains(err.Error(), "protected secret storage is not implemented") {
		t.Fatalf("expected protected storage refusal, got %v", err)
	}
}

func TestStoreNormalizesInvalidTokenReserve(t *testing.T) {
	settings := normalized(Settings{ContextTokens: 1000, ResponseReserveTokens: 1000})
	if settings.ResponseReserveTokens != 250 {
		t.Fatalf("expected reserve to be reduced, got %#v", settings)
	}
}

func requireProtectedSettingsSecretStorage(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" && os.Getenv("NEXUSDESK_RUN_OS_SECRET_TESTS") != "1" {
		t.Skip("set NEXUSDESK_RUN_OS_SECRET_TESTS=1 to exercise the real OS secret backend on " + runtime.GOOS)
	}
	if !protectedsecret.Available() {
		t.Skip("protected settings secret storage backend is unavailable on " + runtime.GOOS)
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

func TestRecommendedModelCatalogPortsWailsContextRules(t *testing.T) {
	options := RecommendedModelOptions()
	if len(options) == 0 || options[0].ID != "qwen3:4b-instruct" {
		t.Fatalf("expected Wails recommended model catalog, got %#v", options)
	}
	if got := ModelContextWindow("mistral-small3.2"); got != 131072 {
		t.Fatalf("expected :latest-insensitive context lookup, got %d", got)
	}
	if got := ModelContextWindow("unknown-model"); got != FallbackModelContextTokens {
		t.Fatalf("expected fallback context, got %d", got)
	}
}

func TestSettingsForSelectedModelUpdatesContextAndReserve(t *testing.T) {
	settings := SettingsForSelectedModel(Defaults(), "qwen3:8b")
	if settings.Model != "qwen3:8b" || settings.ContextTokens != 40960 {
		t.Fatalf("unexpected selected model settings: %#v", settings)
	}
	if settings.ResponseReserveTokens != 5120 {
		t.Fatalf("expected 1/8 response reserve, got %d", settings.ResponseReserveTokens)
	}
	if got := ResponseReserveForContext(999999); got != 32768 {
		t.Fatalf("expected reserve cap, got %d", got)
	}
	if got := ResponseReserveForContext(1000); got != 2048 {
		t.Fatalf("expected reserve floor, got %d", got)
	}
}
