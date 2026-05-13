package storage

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const RedactedAPIKey = "********"
const storedAPIKeyReference = "__nexusdesk_os_credential_store__"

type LLMSettings struct {
	ProviderName string `json:"providerName"`
	BaseURL      string `json:"baseUrl"`
	Model        string `json:"model"`
	APIKey       string `json:"apiKey"`
	UpdatedAt    string `json:"updatedAt"`
}

type LLMSettingsStore struct {
	path string
	mu   sync.Mutex
}

func NewDefaultLLMSettingsStore() *LLMSettingsStore {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		configDir = os.TempDir()
	}

	return NewLLMSettingsStore(filepath.Join(configDir, "NexusDesk", "llm-settings.json"))
}

func NewLLMSettingsStore(path string) *LLMSettingsStore {
	return &LLMSettingsStore{path: path}
}

func DefaultLLMSettings() LLMSettings {
	return LLMSettings{
		ProviderName: "Local OpenAI-compatible",
		BaseURL:      "http://localhost:11434/v1",
		Model:        "",
		APIKey:       "",
	}
}

func (s *LLMSettingsStore) Get() (LLMSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.read()
	if err != nil {
		return LLMSettings{}, err
	}
	if settings.APIKey == storedAPIKeyReference {
		secret, err := s.readAPIKeySecret()
		if err != nil {
			return LLMSettings{}, err
		}
		if secret == "" {
			settings.APIKey = ""
		}
	}

	return redactLLMSettings(settings), nil
}

func (s *LLMSettingsStore) Save(settings LLMSettings) (LLMSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings = normalizeLLMSettings(settings)
	apiKey := settings.APIKey
	if settings.APIKey == RedactedAPIKey {
		existingAPIKey, err := s.existingAPIKey()
		if err != nil {
			return LLMSettings{}, err
		}
		apiKey = existingAPIKey
	}

	if err := validateLLMSettings(settings); err != nil {
		return LLMSettings{}, err
	}

	if apiKey != "" {
		if err := s.writeAPIKeySecret(apiKey); err != nil {
			return LLMSettings{}, err
		}
		settings.APIKey = storedAPIKeyReference
	} else {
		if err := s.deleteAPIKeySecret(); err != nil {
			return LLMSettings{}, err
		}
		settings.APIKey = ""
	}

	settings.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return LLMSettings{}, err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return LLMSettings{}, err
	}

	if err := os.WriteFile(s.path, append(data, '\n'), 0o600); err != nil {
		return LLMSettings{}, err
	}

	return redactLLMSettings(settings), nil
}

func (s *LLMSettingsStore) ResolveForUse(settings LLMSettings) (LLMSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings = normalizeLLMSettings(settings)
	if settings.APIKey != RedactedAPIKey && settings.APIKey != storedAPIKeyReference {
		return settings, nil
	}

	apiKey, err := s.existingAPIKey()
	if err != nil {
		return LLMSettings{}, err
	}

	settings.APIKey = apiKey
	return settings, nil
}

func (s *LLMSettingsStore) read() (LLMSettings, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return DefaultLLMSettings(), nil
	}
	if err != nil {
		return LLMSettings{}, err
	}

	settings := DefaultLLMSettings()
	if err := json.Unmarshal(data, &settings); err != nil {
		return LLMSettings{}, err
	}

	return normalizeLLMSettings(settings), nil
}

func (s *LLMSettingsStore) existingAPIKey() (string, error) {
	secret, err := s.readAPIKeySecret()
	if err != nil {
		return "", err
	}
	if secret != "" {
		return secret, nil
	}

	existing, err := s.read()
	if err != nil {
		return "", err
	}
	if existing.APIKey == storedAPIKeyReference || existing.APIKey == RedactedAPIKey {
		return "", nil
	}
	return existing.APIKey, nil
}

func (s *LLMSettingsStore) apiKeySecretPath() string {
	return s.path + ".secret"
}

func (s *LLMSettingsStore) readAPIKeySecret() (string, error) {
	data, err := os.ReadFile(s.apiKeySecretPath())
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	encrypted, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return "", err
	}

	plain, err := unprotectSecret(encrypted)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *LLMSettingsStore) writeAPIKeySecret(apiKey string) error {
	protected, err := protectSecret([]byte(apiKey))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.apiKeySecretPath()), 0o755); err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(protected)
	return os.WriteFile(s.apiKeySecretPath(), []byte(encoded+"\n"), 0o600)
}

func (s *LLMSettingsStore) deleteAPIKeySecret() error {
	err := os.Remove(s.apiKeySecretPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func normalizeLLMSettings(settings LLMSettings) LLMSettings {
	settings.ProviderName = strings.TrimSpace(settings.ProviderName)
	settings.BaseURL = strings.TrimSpace(settings.BaseURL)
	settings.Model = strings.TrimSpace(settings.Model)
	settings.APIKey = strings.TrimSpace(settings.APIKey)

	if settings.ProviderName == "" {
		settings.ProviderName = "OpenAI-compatible"
	}

	return settings
}

func redactLLMSettings(settings LLMSettings) LLMSettings {
	settings = normalizeLLMSettings(settings)
	if settings.APIKey != "" {
		settings.APIKey = RedactedAPIKey
	}
	return settings
}

func validateLLMSettings(settings LLMSettings) error {
	if settings.BaseURL == "" {
		return errors.New("LLM base URL is required")
	}

	parsed, err := url.ParseRequestURI(settings.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("LLM base URL must be a valid HTTP URL")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("LLM base URL must use http or https")
	}

	return nil
}
