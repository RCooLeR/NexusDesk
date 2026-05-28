package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Store struct {
	path string
}

func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return NewFileStore(filepath.Join(dir, "NexusDesk", "settings.json")), nil
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

func (s *Store) Save(settings Settings) error {
	settings = normalized(settings)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
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
