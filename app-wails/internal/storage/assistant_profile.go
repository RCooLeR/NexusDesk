package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxAssistantMemoryLength = 4000
const maxPromptProfileInstructionsLength = 2400
const maxPromptProfiles = 12

type PromptProfile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Instructions string `json:"instructions"`
}

type AssistantProfile struct {
	Memory          string          `json:"memory"`
	ActiveProfileID string          `json:"activeProfileId"`
	PromptProfiles  []PromptProfile `json:"promptProfiles"`
	UpdatedAt       string          `json:"updatedAt"`
}

type AssistantProfileStore struct {
	path string
	mu   sync.Mutex
}

func NewDefaultAssistantProfileStore() *AssistantProfileStore {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		configDir = os.TempDir()
	}

	return NewAssistantProfileStore(filepath.Join(configDir, "NexusAugenticStudio", "assistant-profile.json"))
}

func NewAssistantProfileStore(path string) *AssistantProfileStore {
	return &AssistantProfileStore{path: path}
}

func DefaultAssistantProfile() AssistantProfile {
	return AssistantProfile{
		ActiveProfileID: "balanced",
		PromptProfiles: []PromptProfile{
			{
				ID:           "balanced",
				Name:         "Balanced",
				Instructions: "Answer clearly, stay grounded in attached sources, call out uncertainty, and suggest practical next steps.",
			},
			{
				ID:           "reviewer",
				Name:         "Reviewer",
				Instructions: "Review for bugs, risks, regressions, missing tests, unclear assumptions, and source-backed fixes. Lead with findings.",
			},
			{
				ID:           "architect",
				Name:         "Architect",
				Instructions: "Focus on architecture, module boundaries, data flow, safety boundaries, maintainability, and phased implementation.",
			},
			{
				ID:           "report",
				Name:         "Report Writer",
				Instructions: "Produce polished Markdown with concise headings, source-grounded claims, decisions, risks, and action items.",
			},
		},
	}
}

func (s *AssistantProfileStore) Get() (AssistantProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.read()
}

func (s *AssistantProfileStore) Save(profile AssistantProfile) (AssistantProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile = normalizeAssistantProfile(profile)
	profile.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return AssistantProfile{}, err
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return AssistantProfile{}, err
	}
	if err := os.WriteFile(s.path, append(data, '\n'), 0o600); err != nil {
		return AssistantProfile{}, err
	}
	return profile, nil
}

func (s *AssistantProfileStore) read() (AssistantProfile, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return DefaultAssistantProfile(), nil
	}
	if err != nil {
		return AssistantProfile{}, err
	}

	profile := DefaultAssistantProfile()
	if err := json.Unmarshal(data, &profile); err != nil {
		return AssistantProfile{}, err
	}
	return normalizeAssistantProfile(profile), nil
}

func normalizeAssistantProfile(profile AssistantProfile) AssistantProfile {
	profile.Memory = truncateString(strings.TrimSpace(profile.Memory), maxAssistantMemoryLength)
	profiles := make([]PromptProfile, 0, len(profile.PromptProfiles))
	seen := map[string]bool{}
	for _, item := range profile.PromptProfiles {
		normalized := PromptProfile{
			ID:           normalizeProfileID(item.ID),
			Name:         truncateString(strings.TrimSpace(item.Name), 80),
			Instructions: truncateString(strings.TrimSpace(item.Instructions), maxPromptProfileInstructionsLength),
		}
		if normalized.ID == "" || normalized.Name == "" || seen[normalized.ID] {
			continue
		}
		seen[normalized.ID] = true
		profiles = append(profiles, normalized)
		if len(profiles) >= maxPromptProfiles {
			break
		}
	}
	if len(profiles) == 0 {
		profiles = DefaultAssistantProfile().PromptProfiles
	}
	if !profileIDExists(profiles, profile.ActiveProfileID) {
		profile.ActiveProfileID = profiles[0].ID
	}
	profile.PromptProfiles = profiles
	return profile
}

func normalizeProfileID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' || char == '_' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}

func profileIDExists(profiles []PromptProfile, id string) bool {
	for _, profile := range profiles {
		if profile.ID == id {
			return true
		}
	}
	return false
}

func truncateString(value string, maxLength int) string {
	if maxLength <= 0 || len(value) <= maxLength {
		return value
	}
	return value[:maxLength]
}
