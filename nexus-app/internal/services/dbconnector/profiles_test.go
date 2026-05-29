package dbconnector

import (
	"encoding/json"
	"nexusdesk/internal/services/protectedsecret"
	"os"
	"path/filepath"
	"runtime"
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
	requireProtectedConnectorSecretStorage(t)
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
	if saved.Password != RedactedSecret {
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
	if len(listed) != 1 || listed[0].Password != RedactedSecret {
		t.Fatalf("expected listed profile to be redacted, got %+v", listed)
	}
	resolved, err := store.ResolveByIDForUse(saved.ID)
	if err != nil {
		t.Fatalf("ResolveByIDForUse failed: %v", err)
	}
	if resolved.Password != "super-secret" {
		t.Fatalf("expected resolved secret, got %q", resolved.Password)
	}
}

func TestConnectorProfileStorePreservesRedactedCredentialOnSave(t *testing.T) {
	requireProtectedConnectorSecretStorage(t)
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
		Password:      RedactedSecret,
		CredentialRef: saved.CredentialRef,
	})
	if err != nil {
		t.Fatalf("Save with redacted credential failed: %v", err)
	}
	if updated.Password != RedactedSecret {
		t.Fatalf("expected redacted password, got %q", updated.Password)
	}
	resolved, err := store.ResolveByIDForUse(updated.ID)
	if err != nil {
		t.Fatalf("ResolveByIDForUse failed: %v", err)
	}
	if resolved.Password != "secret-one" {
		t.Fatalf("expected preserved secret, got %q", resolved.Password)
	}
}

func TestConnectorProfileStoreClearsCredentialWhenBlankReferenceIsSaved(t *testing.T) {
	requireProtectedConnectorSecretStorage(t)
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
	resolved, err := store.ResolveByIDForUse(saved.ID)
	if err != nil {
		t.Fatalf("ResolveByIDForUse failed: %v", err)
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

func TestNormalizeConnectorProfileDefaultsNetworkProfilesToEncryptedTransport(t *testing.T) {
	for _, kind := range []string{"postgres", "mysql", "mariadb", "sqlserver"} {
		profile := normalizeConnectorProfile(ConnectorProfile{Kind: kind})
		if profile.SSLMode != ConnectorSSLModeRequire {
			t.Fatalf("expected %s default SSL mode %q, got %q", kind, ConnectorSSLModeRequire, profile.SSLMode)
		}
	}
	for _, kind := range []string{"sqlite", "duckdb"} {
		profile := normalizeConnectorProfile(ConnectorProfile{Kind: kind})
		if profile.SSLMode != "" {
			t.Fatalf("expected local %s profile to have no SSL mode, got %q", kind, profile.SSLMode)
		}
	}
}

func TestNormalizeConnectorProfileMakesPlaintextModeExplicit(t *testing.T) {
	for _, input := range []string{"disable", "false", "off", "plaintext", "development-plaintext"} {
		profile := normalizeConnectorProfile(ConnectorProfile{Kind: "postgres", SSLMode: input})
		if profile.SSLMode != ConnectorSSLModeDevelopmentPlaintext {
			t.Fatalf("expected %q to normalize to explicit plaintext mode, got %q", input, profile.SSLMode)
		}
	}
}

func TestNormalizeExternalReadOnlySQL(t *testing.T) {
	sqlText, err := NormalizeExternalReadOnlySQL("with q as (select 1 as a) select * from q;")
	if err != nil {
		t.Fatalf("expected read-only SQL to be accepted: %v", err)
	}
	if sqlText != "with q as (select 1 as a) select * from q" {
		t.Fatalf("unexpected normalized SQL: %q", sqlText)
	}
	for _, blocked := range []string{
		"delete from users",
		"select * from users; select * from accounts",
		"pragma table_info(users)",
		"select id into backup_users from users",
		"select name into outfile '/tmp/users.csv' from users",
		";select * from users",
		"select * from users;;",
	} {
		if _, err := NormalizeExternalReadOnlySQL(blocked); err == nil {
			t.Fatalf("expected SQL to be rejected: %s", blocked)
		}
	}
	for _, allowed := range []string{
		"select \"into\" as keyword_name from report",
		"select * from [update]",
		"select * from `delete`",
		"select $$delete from not-a-command$$ as body",
		"/* delete from users */ select id from users",
	} {
		if _, err := NormalizeExternalReadOnlySQL(allowed); err != nil {
			t.Fatalf("expected SQL to be accepted: %s (%v)", allowed, err)
		}
	}
}

func TestNormalizeExternalReadOnlySQLKeepsSemicolonsInsideLiterals(t *testing.T) {
	for _, query := range []string{
		"select 'a;b;c' as note from logs",
		"select $$first;second$$ as body from logs",
		"select $tag$with;semi$tag$ as body from logs",
		"/* leading ; ; ; */ select id from logs",
	} {
		if _, err := NormalizeExternalReadOnlySQL(query); err != nil {
			t.Fatalf("expected SQL to be accepted: %s (%v)", query, err)
		}
	}
}

func TestNormalizeExternalReadOnlySQLForKindBlocksEngineSpecificTokens(t *testing.T) {
	blocked := []struct {
		kind  string
		query string
	}{
		{kind: "postgres", query: "select pg_read_file('/etc/passwd') as body"},
		{kind: "mysql", query: "select load_file('/etc/passwd') as body"},
		{kind: "mariadb", query: "select load_file('/etc/passwd') as body"},
		{kind: "sqlserver", query: "select * from openrowset('SQLNCLI', 'Server=.;Trusted_Connection=yes;', 'select 1')"},
		{kind: "sqlite", query: "select load_extension('mod_spatialite')"},
	}
	for _, testCase := range blocked {
		if _, err := NormalizeExternalReadOnlySQLForKind(testCase.kind, testCase.query); err == nil {
			t.Fatalf("expected %s query to be rejected: %s", testCase.kind, testCase.query)
		}
	}

	if _, err := NormalizeExternalReadOnlySQLForKind("", "select load_file('/etc/passwd') as body"); err != nil {
		t.Fatalf("expected generic guard to keep current compatibility behavior: %v", err)
	}

	if _, err := NormalizeExternalReadOnlySQLForKind("postgres", `select "pg_read_file" as keyword_name from report`); err != nil {
		t.Fatalf("expected quoted identifier to be accepted: %v", err)
	}
}

func TestConnectorProfileStoreListForWorkspaceFiltersByScope(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connector-profiles.json")
	store := NewConnectorProfileStore(path)
	if _, err := store.Save(ConnectorProfile{
		Name:           "Global",
		Kind:           "postgres",
		Host:           "global.db.test",
		Database:       "analytics",
		WorkspaceScope: "",
	}); err != nil {
		t.Fatalf("Save global profile failed: %v", err)
	}
	if _, err := store.Save(ConnectorProfile{
		Name:           "Scoped A",
		Kind:           "postgres",
		Host:           "a.db.test",
		Database:       "analytics",
		WorkspaceScope: `C:\Workspaces\A`,
	}); err != nil {
		t.Fatalf("Save scoped profile failed: %v", err)
	}
	profiles, err := store.ListForWorkspace(`C:\Workspaces\A`)
	if err != nil {
		t.Fatalf("ListForWorkspace failed: %v", err)
	}
	if len(profiles) != 2 {
		t.Fatalf("expected global + scoped profiles, got %d", len(profiles))
	}
	other, err := store.ListForWorkspace(`C:\Workspaces\B`)
	if err != nil {
		t.Fatalf("ListForWorkspace failed: %v", err)
	}
	if len(other) != 1 || other[0].Name != "Global" {
		t.Fatalf("expected only global profile in other workspace, got %#v", other)
	}
}

func TestWorkspaceScopeMatchesNormalizesWindowsPaths(t *testing.T) {
	if !workspaceScopeMatches(`C:\Workspaces\App`, `c:/workspaces/app`) {
		t.Fatal("expected mixed Windows separators/case to match")
	}
	if workspaceScopeMatches(`C:\Workspaces\App`, `c:/workspaces/other`) {
		t.Fatal("expected different paths to not match")
	}
	if !workspaceScopeMatches("", `c:/workspaces/any`) {
		t.Fatal("expected empty scope to match any workspace")
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

func requireProtectedConnectorSecretStorage(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" && os.Getenv("NEXUSDESK_RUN_OS_SECRET_TESTS") != "1" {
		t.Skip("set NEXUSDESK_RUN_OS_SECRET_TESTS=1 to exercise the real OS secret backend on " + runtime.GOOS)
	}
	if !protectedsecret.Available() {
		t.Skip("protected connector secret storage backend is unavailable on " + runtime.GOOS)
	}
}
