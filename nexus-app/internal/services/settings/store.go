package settings

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	RedactedAPIKey             = "********"
	storedAPIKeyReference      = "__nexus_os_credential_store__"
	legacyStoredAPIKeyRef      = "__nexusdesk_os_credential_store__"
	settingsConfigRelativePath = "NexusDesk/settings.json"
)

type Store struct {
	path string
	mu   sync.Mutex
}

func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return NewFileStore(filepath.Join(dir, settingsConfigRelativePath)), nil
}

func NewFileStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

func (s *Store) Load() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.read()
	if err != nil {
		return Settings{}, err
	}
	if isStoredAPIKeyReference(settings.APIKey) {
		secret, err := s.readAPIKeySecret()
		if err != nil {
			return Settings{}, err
		}
		settings.APIKey = secret
	}
	return normalized(settings), nil
}

func (s *Store) LoadForDisplay() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.read()
	if err != nil {
		return Settings{}, err
	}
	if strings.TrimSpace(settings.APIKey) != "" {
		settings.APIKey = RedactedAPIKey
	}
	return normalized(settings), nil
}

func (s *Store) ResolveForUse(settings Settings) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings = normalized(settings)
	if settings.APIKey != RedactedAPIKey && !isStoredAPIKeyReference(settings.APIKey) {
		return settings, nil
	}
	secret, err := s.existingAPIKey()
	if err != nil {
		return Settings{}, err
	}
	settings.APIKey = secret
	return normalized(settings), nil
}

func (s *Store) Save(settings Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings = normalized(settings)
	apiKey := settings.APIKey
	if apiKey == RedactedAPIKey {
		existing, err := s.existingAPIKey()
		if err != nil {
			return err
		}
		apiKey = existing
	}
	if apiKey != "" {
		if err := s.writeAPIKeySecret(apiKey); err != nil {
			return err
		}
		settings.APIKey = storedAPIKeyReference
	} else {
		if err := s.deleteAPIKeySecret(); err != nil {
			return err
		}
		settings.APIKey = ""
	}
	return s.write(settings)
}

func (s *Store) read() (Settings, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return Defaults(), nil
	}
	if err != nil {
		return Settings{}, err
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return Settings{}, err
	}
	return normalized(settings), nil
}

func (s *Store) write(settings Settings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}

func (s *Store) apiKeySecretPath() string {
	return s.path + ".secret"
}

func (s *Store) existingAPIKey() (string, error) {
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
	if isStoredAPIKeyReference(existing.APIKey) || existing.APIKey == RedactedAPIKey {
		return "", nil
	}
	return strings.TrimSpace(existing.APIKey), nil
}

func (s *Store) readAPIKeySecret() (string, error) {
	data, err := os.ReadFile(s.apiKeySecretPath())
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	protected, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
	if err != nil {
		return "", err
	}
	plain, err := unprotectSecret(protected)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *Store) writeAPIKeySecret(apiKey string) error {
	protected, err := protectSecret([]byte(apiKey))
	if err != nil {
		return err
	}
	if err := s.deleteAPIKeySecret(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.apiKeySecretPath()), 0o755); err != nil {
		return err
	}
	encoded := base64.StdEncoding.EncodeToString(protected)
	return os.WriteFile(s.apiKeySecretPath(), []byte(encoded+"\n"), 0o600)
}

func (s *Store) deleteAPIKeySecret() error {
	data, readErr := os.ReadFile(s.apiKeySecretPath())
	if readErr == nil {
		if protected, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data))); err == nil {
			if err := deleteProtectedSecret(protected); err != nil {
				return err
			}
		}
	} else if !os.IsNotExist(readErr) {
		return readErr
	}
	err := os.Remove(s.apiKeySecretPath())
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func isStoredAPIKeyReference(value string) bool {
	value = strings.TrimSpace(value)
	return value == storedAPIKeyReference || value == legacyStoredAPIKeyRef
}

func normalized(settings Settings) Settings {
	defaults := Defaults()
	settings.Provider = strings.TrimSpace(settings.Provider)
	settings.Protocol = strings.TrimSpace(settings.Protocol)
	settings.BaseURL = strings.TrimSpace(settings.BaseURL)
	settings.APIKey = strings.TrimSpace(settings.APIKey)
	if settings.Provider == "" {
		settings.Provider = defaults.Provider
	}
	if settings.Protocol == "" {
		settings.Protocol = inferProtocol(settings.Provider, settings.BaseURL)
	}
	if settings.BaseURL == "" {
		if profile, ok := ProviderProfileByID(settings.Provider); ok && profile.DefaultBaseURL != "" {
			settings.BaseURL = profile.DefaultBaseURL
		} else {
			settings.BaseURL = defaults.BaseURL
		}
	}
	settings.Model = strings.TrimSpace(settings.Model)
	if settings.ContextTokens <= 0 {
		settings.ContextTokens = defaults.ContextTokens
	}
	if settings.ResponseReserveTokens <= 0 {
		settings.ResponseReserveTokens = defaults.ResponseReserveTokens
	}
	if settings.ResponseReserveTokens >= settings.ContextTokens {
		settings.ResponseReserveTokens = settings.ContextTokens / 4
	}
	settings.ModelRoutes = normalizedModelRoutes(settings.ModelRoutes)
	return settings
}

func inferProtocol(provider string, baseURL string) string {
	if profile, ok := ProviderProfileByID(strings.TrimSpace(provider)); ok {
		return profile.Protocol
	}
	lowerProvider := strings.ToLower(strings.TrimSpace(provider))
	lowerBaseURL := strings.ToLower(strings.TrimSpace(baseURL))
	if strings.Contains(lowerProvider, "ollama") ||
		strings.Contains(lowerBaseURL, "localhost:11434") ||
		strings.Contains(lowerBaseURL, "127.0.0.1:11434") {
		return ProtocolOllamaOpenAICompatible
	}
	return ProtocolOpenAICompatible
}
