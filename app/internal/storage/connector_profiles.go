package storage

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const connectorCredentialReferencePrefix = "nexus:connector-profile:"
const defaultConnectorResultLimit = 1000
const maxConnectorResultLimit = 10000
const defaultConnectorTimeoutSeconds = 30
const maxConnectorTimeoutSeconds = 300

type ConnectorProfile struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	Driver         string `json:"driver"`
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Database       string `json:"database"`
	Username       string `json:"username"`
	Password       string `json:"password,omitempty"`
	CredentialRef  string `json:"credentialRef"`
	SSLMode        string `json:"sslMode"`
	WorkspaceScope string `json:"workspaceScope"`
	ReadOnly       bool   `json:"readOnly"`
	ResultLimit    int    `json:"resultLimit"`
	TimeoutSeconds int    `json:"timeoutSeconds"`
	UpdatedAt      string `json:"updatedAt"`
}

type ConnectorProfileStore struct {
	path string
	mu   sync.Mutex
}

func NewDefaultConnectorProfileStore() *ConnectorProfileStore {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		configDir = os.TempDir()
	}
	return NewConnectorProfileStore(filepath.Join(configDir, "NexusAugenticStudio", "connector-profiles.json"))
}

func NewConnectorProfileStore(path string) *ConnectorProfileStore {
	return &ConnectorProfileStore{path: path}
}

func (s *ConnectorProfileStore) List() ([]ConnectorProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profiles, err := s.readProfiles()
	if err != nil {
		return nil, err
	}
	for index := range profiles {
		profiles[index] = s.redactedProfile(profiles[index])
	}
	sort.SliceStable(profiles, func(left, right int) bool {
		return strings.ToLower(profiles[left].Name) < strings.ToLower(profiles[right].Name)
	})
	return profiles, nil
}

func (s *ConnectorProfileStore) Save(profile ConnectorProfile) (ConnectorProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profiles, err := s.readProfiles()
	if err != nil {
		return ConnectorProfile{}, err
	}

	profile = normalizeConnectorProfile(profile)
	if profile.ID == "" {
		profile.ID = newConnectorProfileID(profile.Kind, profile.Name)
	}
	if err := validateConnectorProfile(profile); err != nil {
		return ConnectorProfile{}, err
	}

	existing := findConnectorProfile(profiles, profile.ID)
	password := strings.TrimSpace(profile.Password)
	if password == RedactedAPIKey {
		password = ""
		if existing != nil {
			secret, err := s.readCredentialSecret(existing.ID)
			if err != nil {
				return ConnectorProfile{}, err
			}
			password = secret
		}
	}

	if password != "" {
		if err := s.writeCredentialSecret(profile.ID, password); err != nil {
			return ConnectorProfile{}, err
		}
		profile.CredentialRef = connectorCredentialReference(profile.ID)
	} else if profile.CredentialRef == "" {
		if err := s.deleteCredentialSecret(profile.ID); err != nil {
			return ConnectorProfile{}, err
		}
	} else if existing != nil && existing.CredentialRef != "" {
		profile.CredentialRef = existing.CredentialRef
	} else {
		profile.CredentialRef = ""
	}

	profile.Password = ""
	profile.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	profiles = upsertConnectorProfile(profiles, profile)
	if err := s.writeProfiles(profiles); err != nil {
		return ConnectorProfile{}, err
	}
	return s.redactedProfile(profile), nil
}

func (s *ConnectorProfileStore) Delete(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("connector profile id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	profiles, err := s.readProfiles()
	if err != nil {
		return err
	}

	next := profiles[:0]
	found := false
	for _, profile := range profiles {
		if profile.ID == id {
			found = true
			continue
		}
		next = append(next, profile)
	}
	if !found {
		return fmt.Errorf("connector profile %q was not found", id)
	}
	if err := s.deleteCredentialSecret(id); err != nil {
		return err
	}
	return s.writeProfiles(next)
}

func (s *ConnectorProfileStore) ResolveForUse(profile ConnectorProfile) (ConnectorProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile = normalizeConnectorProfile(profile)
	if profile.Password != RedactedAPIKey && profile.Password != "" {
		return profile, nil
	}

	secret, err := s.readCredentialSecret(profile.ID)
	if err != nil {
		return ConnectorProfile{}, err
	}
	profile.Password = secret
	return profile, nil
}

func (s *ConnectorProfileStore) readProfiles() ([]ConnectorProfile, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []ConnectorProfile{}, nil
	}
	if err != nil {
		return nil, err
	}

	var profiles []ConnectorProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		return nil, err
	}
	for index := range profiles {
		profiles[index] = normalizeConnectorProfile(profiles[index])
		profiles[index].Password = ""
	}
	return profiles, nil
}

func (s *ConnectorProfileStore) writeProfiles(profiles []ConnectorProfile) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	cleaned := make([]ConnectorProfile, 0, len(profiles))
	for _, profile := range profiles {
		profile = normalizeConnectorProfile(profile)
		profile.Password = ""
		cleaned = append(cleaned, profile)
	}
	data, err := json.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, append(data, '\n'), 0o600)
}

func (s *ConnectorProfileStore) credentialsPath() string {
	return s.path + ".secrets"
}

func (s *ConnectorProfileStore) readCredentialSecret(id string) (string, error) {
	secrets, err := s.readCredentialSecrets()
	if err != nil {
		return "", err
	}
	return secrets[id], nil
}

func (s *ConnectorProfileStore) writeCredentialSecret(id string, secret string) error {
	secrets, err := s.readCredentialSecrets()
	if err != nil {
		return err
	}
	secrets[id] = secret
	return s.writeCredentialSecrets(secrets)
}

func (s *ConnectorProfileStore) deleteCredentialSecret(id string) error {
	secrets, err := s.readCredentialSecrets()
	if err != nil {
		return err
	}
	delete(secrets, id)
	return s.writeCredentialSecrets(secrets)
}

func (s *ConnectorProfileStore) readCredentialSecrets() (map[string]string, error) {
	data, err := os.ReadFile(s.credentialsPath())
	if os.IsNotExist(err) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}

	var encoded map[string]string
	if err := json.Unmarshal(data, &encoded); err != nil {
		return nil, err
	}

	secrets := map[string]string{}
	for id, value := range encoded {
		protected, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		plain, err := unprotectSecret(protected)
		if err != nil {
			return nil, err
		}
		secrets[id] = string(plain)
	}
	return secrets, nil
}

func (s *ConnectorProfileStore) writeCredentialSecrets(secrets map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(s.credentialsPath()), 0o755); err != nil {
		return err
	}
	encoded := map[string]string{}
	for id, secret := range secrets {
		if strings.TrimSpace(secret) == "" {
			continue
		}
		protected, err := protectSecret([]byte(secret))
		if err != nil {
			return err
		}
		encoded[id] = base64.StdEncoding.EncodeToString(protected)
	}
	if len(encoded) == 0 {
		err := os.Remove(s.credentialsPath())
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	data, err := json.MarshalIndent(encoded, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.credentialsPath(), append(data, '\n'), 0o600)
}

func (s *ConnectorProfileStore) redactedProfile(profile ConnectorProfile) ConnectorProfile {
	profile = normalizeConnectorProfile(profile)
	if profile.CredentialRef != "" {
		if secret, err := s.readCredentialSecret(profile.ID); err == nil && secret != "" {
			profile.Password = RedactedAPIKey
		}
	}
	return profile
}

func normalizeConnectorProfile(profile ConnectorProfile) ConnectorProfile {
	profile.ID = strings.TrimSpace(profile.ID)
	profile.Name = strings.TrimSpace(profile.Name)
	profile.Kind = strings.ToLower(strings.TrimSpace(profile.Kind))
	profile.Driver = strings.TrimSpace(profile.Driver)
	profile.Host = strings.TrimSpace(profile.Host)
	profile.Database = strings.TrimSpace(profile.Database)
	profile.Username = strings.TrimSpace(profile.Username)
	profile.Password = strings.TrimSpace(profile.Password)
	profile.CredentialRef = strings.TrimSpace(profile.CredentialRef)
	profile.SSLMode = strings.TrimSpace(profile.SSLMode)
	profile.WorkspaceScope = strings.TrimSpace(profile.WorkspaceScope)
	if profile.Kind == "" {
		profile.Kind = "postgres"
	}
	if profile.Driver == "" {
		profile.Driver = profile.Kind
	}
	if profile.Name == "" {
		profile.Name = defaultConnectorProfileName(profile)
	}
	if profile.SSLMode == "" {
		profile.SSLMode = "prefer"
	}
	profile.ReadOnly = true
	if profile.ResultLimit <= 0 {
		profile.ResultLimit = defaultConnectorResultLimit
	}
	if profile.ResultLimit > maxConnectorResultLimit {
		profile.ResultLimit = maxConnectorResultLimit
	}
	if profile.TimeoutSeconds <= 0 {
		profile.TimeoutSeconds = defaultConnectorTimeoutSeconds
	}
	if profile.TimeoutSeconds > maxConnectorTimeoutSeconds {
		profile.TimeoutSeconds = maxConnectorTimeoutSeconds
	}
	return profile
}

func validateConnectorProfile(profile ConnectorProfile) error {
	if profile.Name == "" {
		return errors.New("connector profile name is required")
	}
	switch profile.Kind {
	case "sqlite", "postgres", "mysql", "mariadb", "sqlserver", "duckdb":
	default:
		return fmt.Errorf("unsupported connector kind %q", profile.Kind)
	}
	if profile.Port < 0 || profile.Port > 65535 {
		return errors.New("connector profile port must be between 0 and 65535")
	}
	if profile.ResultLimit <= 0 {
		return errors.New("connector profile result limit must be positive")
	}
	if profile.TimeoutSeconds <= 0 {
		return errors.New("connector profile timeout must be positive")
	}
	return nil
}

func defaultConnectorProfileName(profile ConnectorProfile) string {
	if profile.Host != "" && profile.Database != "" {
		return fmt.Sprintf("%s / %s", profile.Host, profile.Database)
	}
	if profile.Database != "" {
		return profile.Database
	}
	if profile.Host != "" {
		return profile.Host
	}
	return strings.ToUpper(profile.Kind[:1]) + profile.Kind[1:] + " connector"
}

func newConnectorProfileID(kind string, name string) string {
	slug := slugifyConnectorProfileID(kind + "-" + name)
	if slug == "" {
		slug = "connector"
	}
	return fmt.Sprintf("%s-%d", slug, time.Now().UTC().UnixNano())
}

var connectorProfileIDPattern = regexp.MustCompile(`[^a-z0-9]+`)

func slugifyConnectorProfileID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = connectorProfileIDPattern.ReplaceAllString(value, "-")
	return strings.Trim(value, "-")
}

func connectorCredentialReference(id string) string {
	return connectorCredentialReferencePrefix + id + ":password"
}

func findConnectorProfile(profiles []ConnectorProfile, id string) *ConnectorProfile {
	for index := range profiles {
		if profiles[index].ID == id {
			return &profiles[index]
		}
	}
	return nil
}

func upsertConnectorProfile(profiles []ConnectorProfile, profile ConnectorProfile) []ConnectorProfile {
	for index := range profiles {
		if profiles[index].ID == profile.ID {
			profiles[index] = profile
			return profiles
		}
	}
	return append(profiles, profile)
}
