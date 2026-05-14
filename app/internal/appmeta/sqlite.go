package appmeta

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
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

type ChatMirror struct {
	ID             string   `json:"id"`
	Role           string   `json:"role"`
	Content        string   `json:"content"`
	ContextRelPath string   `json:"contextRelPath"`
	SourcePaths    []string `json:"sourcePaths"`
	CreatedAt      string   `json:"createdAt"`
}

type ApprovalMirror struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	Risk      string `json:"risk"`
	Decision  string `json:"decision"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

type ArtifactMirror struct {
	ID             string          `json:"id"`
	RelPath        string          `json:"relPath"`
	Kind           string          `json:"kind"`
	Title          string          `json:"title"`
	Source         string          `json:"source"`
	ContextRelPath string          `json:"contextRelPath"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      string          `json:"createdAt"`
}

type ToolRunMirror struct {
	ID            string          `json:"id"`
	ToolName      string          `json:"toolName"`
	Target        string          `json:"target"`
	Risk          string          `json:"risk"`
	Status        string          `json:"status"`
	Mode          string          `json:"mode"`
	ApprovalID    string          `json:"approvalId"`
	Inputs        json.RawMessage `json:"inputs"`
	OutputSummary string          `json:"outputSummary"`
	Error         string          `json:"error"`
	StartedAt     string          `json:"startedAt"`
	CompletedAt   string          `json:"completedAt"`
	DurationMs    int64           `json:"durationMs"`
}

type MirrorData struct {
	Chats     []ChatMirror     `json:"chats"`
	Approvals []ApprovalMirror `json:"approvals"`
	Artifacts []ArtifactMirror `json:"artifacts"`
	ToolRuns  []ToolRunMirror  `json:"toolRuns"`
}

type LineageStats struct {
	NodeCount         int            `json:"nodeCount"`
	EdgeCount         int            `json:"edgeCount"`
	RelationshipCount map[string]int `json:"relationshipCount"`
}

type MetadataColumn struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type MetadataTable struct {
	Name       string           `json:"name"`
	RowCount   int              `json:"rowCount"`
	Columns    []MetadataColumn `json:"columns"`
	SampleRows [][]string       `json:"sampleRows"`
}

type DatasetView struct {
	Name    string   `json:"name"`
	RelPath string   `json:"relPath"`
	Engine  string   `json:"engine"`
	Columns []string `json:"columns"`
	Rows    int      `json:"rows"`
	Message string   `json:"message"`
}

type MetadataBrowser struct {
	Path         string          `json:"path"`
	Tables       []MetadataTable `json:"tables"`
	DatasetViews []DatasetView   `json:"datasetViews"`
	Message      string          `json:"message"`
	UpdatedAt    string          `json:"updatedAt"`
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
	if err := writeManifest(absRoot, status); err != nil {
		return SQLiteStatus{}, err
	}
	return status, nil
}

func Exists(root string) bool {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	_, err = os.Stat(filepath.Join(absRoot, filepath.FromSlash(metadataDirRelPath), "nexusdesk.sqlite"))
	return err == nil
}

func Mirror(root string, data MirrorData) (SQLiteStatus, error) {
	status, err := Ensure(root)
	if err != nil {
		return SQLiteStatus{}, err
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return SQLiteStatus{}, err
	}
	db, err := sql.Open("sqlite", status.Path)
	if err != nil {
		return SQLiteStatus{}, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return SQLiteStatus{}, err
	}
	if err := replaceMirrorData(tx, absRoot, data); err != nil {
		_ = tx.Rollback()
		return SQLiteStatus{}, err
	}
	if err := tx.Commit(); err != nil {
		return SQLiteStatus{}, err
	}

	status.Message = "SQLite metadata store mirrored from current JSON compatibility stores."
	status.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return status, writeManifest(absRoot, status)
}

func Inspect(root string, datasetViews []DatasetView) (MetadataBrowser, error) {
	status, err := Ensure(root)
	if err != nil {
		return MetadataBrowser{}, err
	}
	db, err := sql.Open("sqlite", status.Path)
	if err != nil {
		return MetadataBrowser{}, err
	}
	defer db.Close()

	tables := []MetadataTable{}
	for _, tableName := range status.Tables {
		table, err := inspectTable(db, tableName)
		if err != nil {
			return MetadataBrowser{}, err
		}
		tables = append(tables, table)
	}
	return MetadataBrowser{
		Path:         status.Path,
		Tables:       tables,
		DatasetViews: datasetViews,
		Message:      "SQLite metadata tables and dataset SQL views are available for inspection.",
		UpdatedAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func ListChats(root string) ([]ChatMirror, error) {
	status, db, err := openExisting(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, role, content, context_rel_path, source_paths_json, created_at
		FROM chats WHERE workspace_root = ? ORDER BY created_at ASC, id ASC`, workspaceRoot)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	_ = status
	items := []ChatMirror{}
	for rows.Next() {
		var item ChatMirror
		var sourcePaths string
		if err := rows.Scan(&item.ID, &item.Role, &item.Content, &item.ContextRelPath, &sourcePaths, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(sourcePaths), &item.SourcePaths)
		items = append(items, item)
	}
	return items, rows.Err()
}

func ListApprovals(root string) ([]ApprovalMirror, error) {
	_, db, err := openExisting(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, action, target, risk, decision, message, created_at
		FROM approvals WHERE workspace_root = ? ORDER BY created_at DESC, id DESC`, workspaceRoot)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ApprovalMirror{}
	for rows.Next() {
		var item ApprovalMirror
		if err := rows.Scan(&item.ID, &item.Action, &item.Target, &item.Risk, &item.Decision, &item.Message, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func ListArtifacts(root string) ([]ArtifactMirror, error) {
	_, db, err := openExisting(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, rel_path, kind, title, source, context_rel_path, metadata_json, created_at
		FROM artifacts WHERE workspace_root = ? ORDER BY created_at DESC, rel_path ASC`, workspaceRoot)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ArtifactMirror{}
	for rows.Next() {
		var item ArtifactMirror
		var metadata string
		if err := rows.Scan(&item.ID, &item.RelPath, &item.Kind, &item.Title, &item.Source, &item.ContextRelPath, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Metadata = json.RawMessage(metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func ListToolRuns(root string) ([]ToolRunMirror, error) {
	_, db, err := openExisting(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, tool_name, target, risk, status, mode, approval_id, inputs_json, output_summary, error, started_at, completed_at, duration_ms
		FROM tool_runs WHERE workspace_root = ? ORDER BY started_at DESC, id DESC`, workspaceRoot)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ToolRunMirror{}
	for rows.Next() {
		var item ToolRunMirror
		var inputs string
		if err := rows.Scan(
			&item.ID,
			&item.ToolName,
			&item.Target,
			&item.Risk,
			&item.Status,
			&item.Mode,
			&item.ApprovalID,
			&inputs,
			&item.OutputSummary,
			&item.Error,
			&item.StartedAt,
			&item.CompletedAt,
			&item.DurationMs,
		); err != nil {
			return nil, err
		}
		item.Inputs = json.RawMessage(inputs)
		items = append(items, item)
	}
	return items, rows.Err()
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

func openExisting(root string) (SQLiteStatus, *sql.DB, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return SQLiteStatus{}, nil, err
	}
	dbPath := filepath.Join(absRoot, filepath.FromSlash(metadataDirRelPath), "nexusdesk.sqlite")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return SQLiteStatus{}, nil, err
	}
	tables, err := listTables(db)
	if err != nil {
		db.Close()
		return SQLiteStatus{}, nil, err
	}
	hash := sha256.Sum256([]byte(schemaSQL))
	return SQLiteStatus{
		Path:          dbPath,
		SchemaPath:    filepath.Join(absRoot, filepath.FromSlash(metadataDirRelPath), schemaFileName),
		SchemaVersion: 1,
		SchemaHash:    hex.EncodeToString(hash[:]),
		Tables:        tables,
	}, db, nil
}

func hashID(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:16])
}

func replaceMirrorData(tx *sql.Tx, workspaceRoot string, data MirrorData) error {
	for _, table := range []string{"chats", "approvals", "artifacts", "tool_runs"} {
		if _, err := tx.Exec("DELETE FROM "+table+" WHERE workspace_root = ?", workspaceRoot); err != nil {
			return err
		}
	}
	for index, item := range data.Chats {
		sourcePaths, _ := json.Marshal(item.SourcePaths)
		if _, err := tx.Exec(
			`INSERT INTO chats (id, workspace_root, role, content, context_rel_path, source_paths_json, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			fallbackID(item.ID, workspaceRoot, "chat", index, item.Role+item.CreatedAt+item.Content),
			workspaceRoot,
			item.Role,
			item.Content,
			item.ContextRelPath,
			string(sourcePaths),
			fallbackTime(item.CreatedAt),
		); err != nil {
			return err
		}
	}
	for index, item := range data.Approvals {
		if _, err := tx.Exec(
			`INSERT INTO approvals (id, workspace_root, action, target, risk, decision, message, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			fallbackID(item.ID, workspaceRoot, "approval", index, item.Action+item.Target+item.CreatedAt),
			workspaceRoot,
			item.Action,
			item.Target,
			fallbackString(item.Risk, "medium"),
			fallbackString(item.Decision, "applied"),
			item.Message,
			fallbackTime(item.CreatedAt),
		); err != nil {
			return err
		}
	}
	for index, item := range data.Artifacts {
		if _, err := tx.Exec(
			`INSERT INTO artifacts (id, workspace_root, rel_path, kind, title, source, context_rel_path, metadata_json, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			fallbackID(item.ID, workspaceRoot, "artifact", index, item.RelPath+item.CreatedAt),
			workspaceRoot,
			item.RelPath,
			fallbackString(item.Kind, "artifact"),
			item.Title,
			item.Source,
			item.ContextRelPath,
			string(item.Metadata),
			fallbackTime(item.CreatedAt),
		); err != nil {
			return err
		}
	}
	for index, item := range data.ToolRuns {
		if _, err := tx.Exec(
			`INSERT INTO tool_runs (id, workspace_root, tool_name, target, risk, status, mode, approval_id, inputs_json, output_summary, error, started_at, completed_at, duration_ms)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			fallbackID(item.ID, workspaceRoot, "tool", index, item.ToolName+item.Target+item.StartedAt),
			workspaceRoot,
			item.ToolName,
			item.Target,
			fallbackString(item.Risk, "low"),
			fallbackString(item.Status, "completed"),
			fallbackString(item.Mode, "dry-run"),
			item.ApprovalID,
			string(item.Inputs),
			item.OutputSummary,
			item.Error,
			fallbackTime(item.StartedAt),
			item.CompletedAt,
			item.DurationMs,
		); err != nil {
			return err
		}
	}
	return nil
}

func inspectTable(db *sql.DB, tableName string) (MetadataTable, error) {
	columns, err := tableColumns(db, tableName)
	if err != nil {
		return MetadataTable{}, err
	}
	rowCount := 0
	if err := db.QueryRow("SELECT COUNT(*) FROM " + tableName).Scan(&rowCount); err != nil {
		return MetadataTable{}, err
	}
	rows, err := db.Query("SELECT * FROM " + tableName + " LIMIT 5")
	if err != nil {
		return MetadataTable{}, err
	}
	defer rows.Close()
	sampleRows := [][]string{}
	values := make([]sql.NullString, len(columns))
	dest := make([]any, len(columns))
	for index := range values {
		dest[index] = &values[index]
	}
	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return MetadataTable{}, err
		}
		row := make([]string, len(columns))
		for index, value := range values {
			if value.Valid {
				row[index] = value.String
			}
		}
		sampleRows = append(sampleRows, row)
	}
	return MetadataTable{Name: tableName, RowCount: rowCount, Columns: columns, SampleRows: sampleRows}, rows.Err()
}

func tableColumns(db *sql.DB, tableName string) ([]MetadataColumn, error) {
	rows, err := db.Query("PRAGMA table_info(" + tableName + ")")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	columns := []MetadataColumn{}
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, MetadataColumn{Name: name, Type: columnType})
	}
	return columns, rows.Err()
}

func writeManifest(absRoot string, status SQLiteStatus) error {
	dir := filepath.Join(absRoot, filepath.FromSlash(metadataDirRelPath))
	payload, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, manifestFileName), append(payload, '\n'), 0o644)
}

func fallbackID(id string, workspaceRoot string, kind string, index int, value string) string {
	id = strings.TrimSpace(id)
	if id != "" {
		return id
	}
	return hashID(strings.Join([]string{workspaceRoot, kind, strconv.Itoa(index), value}, "|"))
}

func fallbackTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return value
}

func fallbackString(value string, next string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return next
	}
	return value
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
