package appmeta

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const metadataDirRelPath = ".nexusdesk/metadata"
const schemaFileName = "schema.sql"
const manifestFileName = "sqlite-manifest.json"

type SQLiteStatus struct {
	Path          string   `json:"path"`
	SchemaPath    string   `json:"schemaPath"`
	SchemaVersion int      `json:"schemaVersion"`
	SchemaHash    string   `json:"schemaHash"`
	Tables        []string `json:"tables"`
	Message       string   `json:"message"`
	UpdatedAt     string   `json:"updatedAt"`
}

func Ensure(root string) (SQLiteStatus, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return SQLiteStatus{}, err
	}
	dir := filepath.Join(absRoot, filepath.FromSlash(metadataDirRelPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return SQLiteStatus{}, err
	}

	schemaPath := filepath.Join(dir, schemaFileName)
	if err := os.WriteFile(schemaPath, []byte(schemaSQL), 0o644); err != nil {
		return SQLiteStatus{}, err
	}
	dbPath := filepath.Join(dir, "nexusdesk.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return SQLiteStatus{}, err
	}
	defer db.Close()
	if _, err := db.Exec(schemaSQL); err != nil {
		return SQLiteStatus{}, err
	}
	now := time.Now().UTC()
	workspaceID := hashID(absRoot)
	if _, err := db.Exec(
		`INSERT INTO workspaces (id, root, name, opened_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(root) DO UPDATE SET name = excluded.name, opened_at = excluded.opened_at`,
		workspaceID,
		absRoot,
		filepath.Base(absRoot),
		now.Format(time.RFC3339),
	); err != nil {
		return SQLiteStatus{}, err
	}
	tables, err := listTables(db)
	if err != nil {
		return SQLiteStatus{}, err
	}
	hash := sha256.Sum256([]byte(schemaSQL))
	status := SQLiteStatus{
		Path:          dbPath,
		SchemaPath:    schemaPath,
		SchemaVersion: 1,
		SchemaHash:    hex.EncodeToString(hash[:]),
		Tables:        tables,
		Message:       "SQLite metadata store is active; JSON stores remain the compatibility layer while repositories migrate incrementally.",
		UpdatedAt:     now.Format(time.RFC3339),
	}
	payload, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return SQLiteStatus{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, manifestFileName), append(payload, '\n'), 0o644); err != nil {
		return SQLiteStatus{}, err
	}
	return status, nil
}

func listTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func hashID(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:16])
}

func SchemaSQL() string {
	return schemaSQL
}

const schemaSQL = `-- NexusDesk SQLite metadata schema v1
-- This schema mirrors the current JSON-backed stores so the migration can be replayed safely.
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS workspaces (
    id TEXT PRIMARY KEY,
    root TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    opened_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chats (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    context_rel_path TEXT,
    source_paths_json TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS approvals (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    action TEXT NOT NULL,
    target TEXT NOT NULL,
    risk TEXT NOT NULL,
    decision TEXT NOT NULL,
    message TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    rel_path TEXT NOT NULL,
    kind TEXT NOT NULL,
    title TEXT,
    source TEXT,
    context_rel_path TEXT,
    metadata_json TEXT,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tool_runs (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    target TEXT,
    risk TEXT NOT NULL,
    status TEXT NOT NULL,
    mode TEXT NOT NULL,
    approval_id TEXT,
    inputs_json TEXT,
    output_summary TEXT,
    error TEXT,
    started_at TEXT NOT NULL,
    completed_at TEXT,
    duration_ms INTEGER
);
`

func HasSchemaTable(schema string, table string) bool {
	return strings.Contains(schema, "CREATE TABLE IF NOT EXISTS "+table)
}
