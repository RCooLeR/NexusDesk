package appmeta

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	hash := sha256.Sum256([]byte(schemaSQL))
	status := SQLiteStatus{
		Path:          filepath.Join(dir, "nexusdesk.sqlite"),
		SchemaPath:    schemaPath,
		SchemaVersion: 1,
		SchemaHash:    hex.EncodeToString(hash[:]),
		Tables:        []string{"workspaces", "chats", "approvals", "artifacts", "tool_runs"},
		Message:       "SQLite metadata schema prepared; JSON stores remain the active compatibility layer until the SQLite driver migration lands.",
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
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
