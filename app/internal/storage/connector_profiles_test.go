package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConnectorProfileStoreReturnsEmptyListWhenMissing(t *testing.T) {
	store := NewConnectorProfileStore(filepath.Join(t.TempDir(), "connector-profiles.json"))

	profiles, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(profiles) != 0 {
		t.Fatalf("expected no profiles, got %d", len(profiles))
	}
}

func TestConnectorProfileStoreSavesRedactedCredentialReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connector-profiles.json")
	store := NewConnectorProfileStore(path)

	saved, err := store.Save(ConnectorProfile{
		Name:           "Marketing warehouse",
		Kind:           "postgres",
		Host:           "db.example.test",
		Port:           5432,
		Database:       "analytics",
		Username:       "analyst",
		Password:       "super-secret",
		ResultLimit:    500,
		TimeoutSeconds: 20,
		WorkspaceScope: "E:/workspace",
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved.ID == "" {
		t.Fatal("expected generated id")
	}
	if saved.Password != RedactedAPIKey {
		t.Fatalf("expected redacted password, got %q", saved.Password)
	}
	if saved.CredentialRef == "" {
		t.Fatal("expected credential reference")
	}
	if !saved.ReadOnly {
		t.Fatal("expected connector profiles to be read-only")
	}

	rawData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if strings.Contains(string(rawData), "super-secret") {
		t.Fatal("public profile file must not contain the secret")
	}
	raw := readRawConnectorProfiles(t, path)
	if len(raw) != 1 {
		t.Fatalf("expected one raw profile, got %d", len(raw))
	}
	if raw[0].Password != "" {
		t.Fatalf("expected raw profile password to be blank, got %q", raw[0].Password)
	}
	if raw[0].CredentialRef != saved.CredentialRef {
		t.Fatalf("expected credential ref %q, got %q", saved.CredentialRef, raw[0].CredentialRef)
	}

	listed, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(listed) != 1 || listed[0].Password != RedactedAPIKey {
		t.Fatalf("expected listed profile to be redacted, got %+v", listed)
	}
	resolved, err := store.ResolveForUse(listed[0])
	if err != nil {
		t.Fatalf("ResolveForUse failed: %v", err)
	}
	if resolved.Password != "super-secret" {
		t.Fatalf("expected resolved secret, got %q", resolved.Password)
	}
}

func TestConnectorProfileStorePreservesRedactedCredentialOnSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connector-profiles.json")
	store := NewConnectorProfileStore(path)

	saved, err := store.Save(ConnectorProfile{
		Name:     "First",
		Kind:     "mysql",
		Host:     "db.example.test",
		Database: "app",
		Password: "secret-one",
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	updated, err := store.Save(ConnectorProfile{
		ID:            saved.ID,
		Name:          "Renamed",
		Kind:          "mysql",
		Host:          "db.example.test",
		Database:      "app",
		Password:      RedactedAPIKey,
		CredentialRef: saved.CredentialRef,
	})
	if err != nil {
		t.Fatalf("Save with redacted credential failed: %v", err)
	}
	if updated.Password != RedactedAPIKey {
		t.Fatalf("expected redacted password, got %q", updated.Password)
	}

	resolved, err := store.ResolveForUse(updated)
	if err != nil {
		t.Fatalf("ResolveForUse failed: %v", err)
	}
	if resolved.Password != "secret-one" {
		t.Fatalf("expected preserved secret, got %q", resolved.Password)
	}
}

func TestConnectorProfileStoreClearsCredentialWhenBlankReferenceIsSaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connector-profiles.json")
	store := NewConnectorProfileStore(path)

	saved, err := store.Save(ConnectorProfile{
		Name:     "SQL Server",
		Kind:     "sqlserver",
		Host:     "db.example.test",
		Database: "warehouse",
		Password: "secret-two",
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	cleared, err := store.Save(ConnectorProfile{
		ID:       saved.ID,
		Name:     "SQL Server",
		Kind:     "sqlserver",
		Host:     "db.example.test",
		Database: "warehouse",
	})
	if err != nil {
		t.Fatalf("Save cleared credential failed: %v", err)
	}
	if cleared.Password != "" || cleared.CredentialRef != "" {
		t.Fatalf("expected credential to be cleared, got %+v", cleared)
	}
	resolved, err := store.ResolveForUse(cleared)
	if err != nil {
		t.Fatalf("ResolveForUse failed: %v", err)
	}
	if resolved.Password != "" {
		t.Fatalf("expected no resolved secret, got %q", resolved.Password)
	}
}

func TestConnectorProfileStoreNormalizesLimitsAndRejectsUnsupportedKind(t *testing.T) {
	store := NewConnectorProfileStore(filepath.Join(t.TempDir(), "connector-profiles.json"))

	saved, err := store.Save(ConnectorProfile{
		Name:           "DuckDB local",
		Kind:           "duckdb",
		ResultLimit:    999999,
		TimeoutSeconds: 999999,
	})
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if saved.ResultLimit != maxConnectorResultLimit {
		t.Fatalf("expected max result cap, got %d", saved.ResultLimit)
	}
	if saved.TimeoutSeconds != maxConnectorTimeoutSeconds {
		t.Fatalf("expected max timeout cap, got %d", saved.TimeoutSeconds)
	}

	if _, err := store.Save(ConnectorProfile{Name: "Bad", Kind: "redis"}); err == nil {
		t.Fatal("expected unsupported kind error")
	}
}

func readRawConnectorProfiles(t *testing.T, path string) []ConnectorProfile {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	var profiles []ConnectorProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	return profiles
}
