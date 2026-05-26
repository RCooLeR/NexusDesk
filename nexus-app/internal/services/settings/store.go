package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func (s *Store) Load() (Settings, error) {
	settings := Defaults()
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return Settings{}, err
	}
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
	if settings.Provider == "" {
		settings.Provider = defaults.Provider
	}
	if settings.BaseURL == "" {
		settings.BaseURL = defaults.BaseURL
	}
	if settings.Model == "" {
		settings.Model = defaults.Model
	}
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
