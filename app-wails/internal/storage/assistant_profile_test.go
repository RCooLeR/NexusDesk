package storage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAssistantProfileStoreReturnsDefaultsWhenMissing(t *testing.T) {
	store := NewAssistantProfileStore(filepath.Join(t.TempDir(), "assistant-profile.json"))

	profile, err := store.Get()
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if profile.ActiveProfileID != "balanced" {
		t.Fatalf("expected balanced default profile, got %q", profile.ActiveProfileID)
	}
	if len(profile.PromptProfiles) < 2 {
		t.Fatalf("expected default prompt profiles, got %#v", profile.PromptProfiles)
	}
}

func TestAssistantProfileStoreSavesMemoryAndActiveProfile(t *testing.T) {
	store := NewAssistantProfileStore(filepath.Join(t.TempDir(), "assistant-profile.json"))

	saved, err := store.Save(AssistantProfile{
		Memory:          "Prefer compact answers.",
		ActiveProfileID: "reviewer",
		PromptProfiles:  DefaultAssistantProfile().PromptProfiles,
	})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if saved.Memory != "Prefer compact answers." || saved.ActiveProfileID != "reviewer" {
		t.Fatalf("unexpected saved profile: %#v", saved)
	}

	loaded, err := store.Get()
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if loaded.Memory != saved.Memory || loaded.ActiveProfileID != saved.ActiveProfileID {
		t.Fatalf("expected persisted profile, got %#v", loaded)
	}
}

func TestAssistantProfileStoreNormalizesInvalidProfiles(t *testing.T) {
	store := NewAssistantProfileStore(filepath.Join(t.TempDir(), "assistant-profile.json"))
	longMemory := strings.Repeat("x", maxAssistantMemoryLength+20)

	saved, err := store.Save(AssistantProfile{
		Memory:          longMemory,
		ActiveProfileID: "missing",
		PromptProfiles: []PromptProfile{
			{ID: "My Profile!", Name: "Custom", Instructions: "Do it well."},
			{ID: "My Profile!", Name: "Duplicate", Instructions: "Ignored."},
			{ID: "", Name: "No ID", Instructions: "Ignored."},
		},
	})
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if len(saved.Memory) != maxAssistantMemoryLength {
		t.Fatalf("expected memory truncation, got %d", len(saved.Memory))
	}
	if saved.ActiveProfileID != "myprofile" {
		t.Fatalf("expected active profile fallback, got %q", saved.ActiveProfileID)
	}
	if len(saved.PromptProfiles) != 1 || saved.PromptProfiles[0].ID != "myprofile" {
		t.Fatalf("unexpected normalized profiles: %#v", saved.PromptProfiles)
	}
}
