package dbconnector

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeConnectorQueryRequest(t *testing.T) {
	request := NormalizeConnectorQueryRequest(ConnectorQueryRequest{
		ProfileID:      "  profile  ",
		SQL:            "  select 1  ",
		ResultLimit:    999999,
		TimeoutSeconds: 999999,
		RequestID:      "  run  ",
	})
	if request.ProfileID != "profile" {
		t.Fatalf("expected trimmed profile id, got %q", request.ProfileID)
	}
	if request.SQL != "select 1" {
		t.Fatalf("expected trimmed SQL, got %q", request.SQL)
	}
	if request.ResultLimit != maxConnectorResultLimit {
		t.Fatalf("expected capped result limit %d, got %d", maxConnectorResultLimit, request.ResultLimit)
	}
	if request.TimeoutSeconds != maxConnectorTimeoutSeconds {
		t.Fatalf("expected capped timeout %d, got %d", maxConnectorTimeoutSeconds, request.TimeoutSeconds)
	}
	if request.RequestID != "run" {
		t.Fatalf("expected trimmed request id, got %q", request.RequestID)
	}
}

func TestProfileValidation(t *testing.T) {
	if err := requirePostgresProfile(ConnectorProfile{ID: "p", Kind: "postgres", Host: "db.local", Database: "app"}); err != nil {
		t.Fatalf("expected valid postgres profile, got %v", err)
	}
	if err := requireMySQLProfile(ConnectorProfile{ID: "m", Kind: "mysql", Host: "db.local", Database: "app"}); err != nil {
		t.Fatalf("expected valid mysql profile, got %v", err)
	}
	if err := requireSQLServerProfile(ConnectorProfile{ID: "s", Kind: "sqlserver", Host: "db.local", Database: "app"}); err != nil {
		t.Fatalf("expected valid sqlserver profile, got %v", err)
	}
	if err := requirePostgresProfile(ConnectorProfile{ID: "p", Kind: "postgres"}); err == nil {
		t.Fatal("expected missing host/database postgres profile to fail")
	}
	if err := requireMySQLProfile(ConnectorProfile{ID: "m", Kind: "mysql"}); err == nil {
		t.Fatal("expected missing host/database mysql profile to fail")
	}
	if err := requireSQLServerProfile(ConnectorProfile{ID: "s", Kind: "sqlserver"}); err == nil {
		t.Fatal("expected missing host/database sqlserver profile to fail")
	}
}

func TestSQLiteProfilePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "warehouse.sqlite")
	if err := writeSQLiteFixture(path); err != nil {
		t.Fatal(err)
	}
	resolved, err := sqliteProfilePath(ConnectorProfile{
		ID:       "sqlite-profile",
		Kind:     "sqlite",
		Database: path,
	})
	if err != nil {
		t.Fatalf("expected sqlite profile path to resolve, got %v", err)
	}
	if !strings.HasSuffix(strings.ToLower(resolved), "warehouse.sqlite") {
		t.Fatalf("unexpected resolved sqlite path: %q", resolved)
	}
	if _, err := sqliteProfilePath(ConnectorProfile{ID: "bad", Kind: "sqlite", Database: filepath.Join(dir, "missing.sqlite")}); err == nil {
		t.Fatal("expected missing sqlite file to fail")
	}
}

func TestDuckDBProfilePathAndDSN(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "warehouse.duckdb")
	if err := os.WriteFile(path, []byte("DUCK"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err := duckDBProfilePath(ConnectorProfile{
		ID:       "duckdb-profile",
		Kind:     "duckdb",
		Database: path,
	})
	if err != nil {
		t.Fatalf("expected duckdb profile path to resolve, got %v", err)
	}
	if !strings.HasSuffix(strings.ToLower(resolved), "warehouse.duckdb") {
		t.Fatalf("unexpected resolved duckdb path: %q", resolved)
	}
	dsn := duckDBDSN(path)
	if !strings.Contains(strings.ToLower(dsn), "access_mode=read_only") {
		t.Fatalf("expected readonly duckdb dsn, got %q", dsn)
	}
	if _, err := duckDBProfilePath(ConnectorProfile{ID: "bad", Kind: "duckdb", Database: filepath.Join(dir, "missing.duckdb")}); err == nil {
		t.Fatal("expected missing duckdb file to fail")
	}
	if _, err := duckDBProfilePath(ConnectorProfile{ID: "bad", Kind: "duckdb", Database: filepath.Join(dir, "bad.txt")}); err == nil {
		t.Fatal("expected non-duckdb extension to fail")
	}
}

func writeSQLiteFixture(path string) error {
	if err := ensureFixtureDir(path); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("SQLite format 3\x00"), 0o644)
}

func TestConnectorErrorRedactsCredentials(t *testing.T) {
	err := connectorError(errors.New(`dial failed: postgres://analyst:super-secret@db.example.test/app?sslmode=disable&password=second-secret`))
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	text := err.Error()
	for _, secret := range []string{"super-secret", "second-secret"} {
		if strings.Contains(text, secret) {
			t.Fatalf("expected secret %q to be redacted from %q", secret, text)
		}
	}
	if !strings.Contains(text, "[redacted]") {
		t.Fatalf("expected redaction marker in %q", text)
	}
}

func TestConnectorErrorRedactsMySQLAndJSONSecrets(t *testing.T) {
	err := connectorError(errors.New(`dsn analyst:s3cr3t@tcp(localhost:3306)/app refused; details={"password":"token123"}`))
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	text := err.Error()
	for _, secret := range []string{"s3cr3t", "token123"} {
		if strings.Contains(text, secret) {
			t.Fatalf("expected secret %q to be redacted from %q", secret, text)
		}
	}
	if !strings.Contains(text, "[redacted]") {
		t.Fatalf("expected redaction marker in %q", text)
	}
}

func TestConnectorErrorRedactsTokenFieldsAndAuthHeader(t *testing.T) {
	err := connectorError(errors.New(`request failed: Authorization: Bearer abc.def.ghi, url=https://api.example.test?q=1&api_key=super-key&token=run-token details={"access_token":"secret-token"} auth_token=inline-token`))
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	text := err.Error()
	for _, secret := range []string{"abc.def.ghi", "super-key", "run-token", "secret-token", "inline-token"} {
		if strings.Contains(text, secret) {
			t.Fatalf("expected secret %q to be redacted from %q", secret, text)
		}
	}
	if !strings.Contains(text, "[redacted]") {
		t.Fatalf("expected redaction marker in %q", text)
	}
}

func TestConnectorErrorRedactsURLPasswordWithoutUsername(t *testing.T) {
	err := connectorError(errors.New(`connect failed: postgres://:fallback-secret@db.example.test/app`))
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	text := err.Error()
	if strings.Contains(text, "fallback-secret") {
		t.Fatalf("expected URL password to be redacted from %q", text)
	}
	if !strings.Contains(text, "[redacted]") {
		t.Fatalf("expected redaction marker in %q", text)
	}
}
