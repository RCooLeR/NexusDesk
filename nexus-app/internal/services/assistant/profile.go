package assistant

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

type Profile struct {
	Memory          string          `json:"memory"`
	ActiveProfileID string          `json:"activeProfileId"`
	PromptProfiles  []PromptProfile `json:"promptProfiles"`
	UpdatedAt       string          `json:"updatedAt"`
}

type ProfileStore struct {
	path string
	mu   sync.Mutex
}

func NewDefaultProfileStore() (*ProfileStore, error) {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		configDir = os.TempDir()
	}
	return NewProfileStore(filepath.Join(configDir, "NexusAugenticStudio", "assistant-profile.json")), nil
}

func NewProfileStore(path string) *ProfileStore {
	return &ProfileStore{path: path}
}

func DefaultProfile() Profile {
	return Profile{
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

func (s *ProfileStore) Get() (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.read()
}

func (s *ProfileStore) Save(profile Profile) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile = NormalizeProfile(profile)
	profile.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return Profile{}, err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return Profile{}, err
	}
	if err := os.WriteFile(s.path, append(data, '\n'), 0o600); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func (s *ProfileStore) read() (Profile, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return DefaultProfile(), nil
	}
	if err != nil {
		return Profile{}, err
	}
	profile := DefaultProfile()
	if err := json.Unmarshal(data, &profile); err != nil {
		return Profile{}, err
	}
	return NormalizeProfile(profile), nil
}

func NormalizeProfile(profile Profile) Profile {
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
		profiles = DefaultProfile().PromptProfiles
	}
	if !profileIDExists(profiles, profile.ActiveProfileID) {
		profile.ActiveProfileID = profiles[0].ID
	}
	profile.PromptProfiles = profiles
	return profile
}

func ApplyProfileToPrompt(prompt string, profile Profile) string {
	profile = NormalizeProfile(profile)
	active := ActivePromptProfile(profile)
	var sections []string
	if active.Instructions != "" {
		sections = append(sections, "Active prompt profile: "+active.Name+"\n"+active.Instructions)
	}
	if strings.TrimSpace(profile.Memory) != "" {
		sections = append(sections, "Assistant memory for this user/workspace:\n"+profile.Memory)
	}
	if len(sections) == 0 {
		return prompt
	}
	return "Use the following assistant preferences while answering. They are user-provided guidance, not source evidence.\n\n" +
		strings.Join(sections, "\n\n") +
		"\n\nUser request:\n\n" +
		prompt
}

func ActivePromptProfile(profile Profile) PromptProfile {
	profile = NormalizeProfile(profile)
	for _, item := range profile.PromptProfiles {
		if item.ID == profile.ActiveProfileID {
			return item
		}
	}
	if len(profile.PromptProfiles) > 0 {
		return profile.PromptProfiles[0]
	}
	return PromptProfile{}
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
