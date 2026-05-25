package appmeta

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
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

type MetadataSearchResult struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Title     string `json:"title"`
	Target    string `json:"target"`
	Snippet   string `json:"snippet"`
	CreatedAt string `json:"createdAt"`
}

type DatasetDependency struct {
	ID          string `json:"id"`
	RelPath     string `json:"relPath"`
	Kind        string `json:"kind"`
	Target      string `json:"target"`
	Query       string `json:"query"`
	Artifact    string `json:"artifact"`
	CreatedAt   string `json:"createdAt"`
	LastRefresh string `json:"lastRefresh"`
}

type SQLRun struct {
	ID        string `json:"id"`
	RelPath   string `json:"relPath"`
	SQL       string `json:"sql"`
	Engine    string `json:"engine"`
	Rows      int    `json:"rows"`
	Artifact  string `json:"artifact"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
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

func AppendChats(root string, chats []ChatMirror) error {
	if len(chats) == 0 {
		return nil
	}
	workspaceRoot, db, err := writableDB(root)
	if err != nil {
		return err
	}
	defer db.Close()
	for index, item := range chats {
		sourcePaths, _ := json.Marshal(item.SourcePaths)
		if _, err := db.Exec(
			`INSERT OR REPLACE INTO chats (id, workspace_root, role, content, context_rel_path, source_paths_json, created_at)
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
	return nil
}

func ClearChats(root string) error {
	workspaceRoot, db, err := writableDB(root)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(`DELETE FROM chats WHERE workspace_root = ?`, workspaceRoot)
	return err
}

func AppendApproval(root string, item ApprovalMirror) error {
	workspaceRoot, db, err := writableDB(root)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(
		`INSERT OR REPLACE INTO approvals (id, workspace_root, action, target, risk, decision, message, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		fallbackID(item.ID, workspaceRoot, "approval", 0, item.Action+item.Target+item.CreatedAt),
		workspaceRoot,
		item.Action,
		item.Target,
		fallbackString(item.Risk, "medium"),
		fallbackString(item.Decision, "applied"),
		item.Message,
		fallbackTime(item.CreatedAt),
	)
	return err
}

func UpsertArtifact(root string, item ArtifactMirror) error {
	workspaceRoot, db, err := writableDB(root)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(
		`INSERT OR REPLACE INTO artifacts (id, workspace_root, rel_path, kind, title, source, context_rel_path, metadata_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fallbackID(item.ID, workspaceRoot, "artifact", 0, item.RelPath+item.CreatedAt),
		workspaceRoot,
		item.RelPath,
		fallbackString(item.Kind, "artifact"),
		item.Title,
		item.Source,
		item.ContextRelPath,
		string(item.Metadata),
		fallbackTime(item.CreatedAt),
	)
	return err
}

func AppendToolRun(root string, item ToolRunMirror) error {
	workspaceRoot, db, err := writableDB(root)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(
		`INSERT OR REPLACE INTO tool_runs (id, workspace_root, tool_name, target, risk, status, mode, approval_id, inputs_json, output_summary, error, started_at, completed_at, duration_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fallbackID(item.ID, workspaceRoot, "tool", 0, item.ToolName+item.Target+item.StartedAt),
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
	)
	return err
}

func RecordDatasetDependency(root string, item DatasetDependency) error {
	workspaceRoot, db, err := writableDB(root)
	if err != nil {
		return err
	}
	defer db.Close()
	createdAt := fallbackTime(item.CreatedAt)
	_, err = db.Exec(
		`INSERT OR REPLACE INTO dataset_dependencies (id, workspace_root, rel_path, kind, target, query, artifact, created_at, last_refresh)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fallbackID(item.ID, workspaceRoot, "dataset-dependency", 0, item.RelPath+item.Kind+item.Target+item.Query+item.Artifact),
		workspaceRoot,
		item.RelPath,
		fallbackString(item.Kind, "query"),
		item.Target,
		item.Query,
		item.Artifact,
		createdAt,
		fallbackString(item.LastRefresh, createdAt),
	)
	return err
}

func ListDatasetDependencies(root string, relPath string) ([]DatasetDependency, error) {
	_, db, err := openExisting(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, rel_path, kind, target, query, artifact, created_at, last_refresh
		FROM dataset_dependencies WHERE workspace_root = ? AND (? = '' OR rel_path = ?) ORDER BY created_at DESC`, workspaceRoot, strings.TrimSpace(relPath), strings.TrimSpace(relPath))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []DatasetDependency{}
	for rows.Next() {
		var item DatasetDependency
		if err := rows.Scan(&item.ID, &item.RelPath, &item.Kind, &item.Target, &item.Query, &item.Artifact, &item.CreatedAt, &item.LastRefresh); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func GetDatasetDependency(root string, id string) (DatasetDependency, error) {
	_, db, err := openExisting(root)
	if err != nil {
		return DatasetDependency{}, err
	}
	defer db.Close()
	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return DatasetDependency{}, err
	}

	var item DatasetDependency
	row := db.QueryRow(
		`SELECT id, rel_path, kind, target, query, artifact, created_at, last_refresh
		 FROM dataset_dependencies
		 WHERE workspace_root = ? AND id = ?`, workspaceRoot, strings.TrimSpace(id))
	if err := row.Scan(&item.ID, &item.RelPath, &item.Kind, &item.Target, &item.Query, &item.Artifact, &item.CreatedAt, &item.LastRefresh); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DatasetDependency{}, errors.New("dataset dependency not found")
		}
		return DatasetDependency{}, err
	}
	return item, nil
}

func UpdateDatasetDependencyRefresh(root string, id string, artifactRelPath string) (string, error) {
	_, db, err := writableDB(root)
	if err != nil {
		return "", err
	}
	defer db.Close()

	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := db.Exec(
		`UPDATE dataset_dependencies
			SET artifact = ?, last_refresh = ?
			WHERE workspace_root = ? AND id = ?`,
		artifactRelPath,
		now,
		workspaceRoot,
		strings.TrimSpace(id),
	)
	if err != nil {
		return "", err
	}
	changes, err := res.RowsAffected()
	if err != nil {
		return "", err
	}
	if changes == 0 {
		return "", errors.New("dataset dependency not found")
	}
	return now, nil
}

func AppendSQLRun(root string, item SQLRun) error {
	workspaceRoot, db, err := writableDB(root)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(
		`INSERT OR REPLACE INTO sql_runs (id, workspace_root, rel_path, sql_text, engine, rows_returned, artifact, status, message, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		fallbackID(item.ID, workspaceRoot, "sql-run", 0, item.RelPath+item.SQL+item.CreatedAt),
		workspaceRoot,
		item.RelPath,
		item.SQL,
		item.Engine,
		item.Rows,
		item.Artifact,
		fallbackString(item.Status, "completed"),
		item.Message,
		fallbackTime(item.CreatedAt),
	)
	return err
}

func ListSQLRuns(root string, relPath string) ([]SQLRun, error) {
	_, db, err := openExisting(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`SELECT id, rel_path, sql_text, engine, rows_returned, artifact, status, message, created_at
		FROM sql_runs WHERE workspace_root = ? AND (? = '' OR rel_path = ?) ORDER BY created_at DESC LIMIT 50`, workspaceRoot, strings.TrimSpace(relPath), strings.TrimSpace(relPath))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []SQLRun{}
	for rows.Next() {
		var item SQLRun
		if err := rows.Scan(&item.ID, &item.RelPath, &item.SQL, &item.Engine, &item.Rows, &item.Artifact, &item.Status, &item.Message, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func Search(root string, query string, limit int) ([]MetadataSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []MetadataSearchResult{}, nil
	}
	if limit <= 0 || limit > 100 {
		limit = 40
	}
	_, db, err := openExisting(root)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	like := "%" + strings.ToLower(query) + "%"
	items := []MetadataSearchResult{}
	appendRows := func(kind string, rows *sql.Rows, err error) error {
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item MetadataSearchResult
			item.Kind = kind
			if err := rows.Scan(&item.ID, &item.Title, &item.Target, &item.Snippet, &item.CreatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	}
	rows, err := db.Query(`SELECT id, role, context_rel_path, substr(content, 1, 220), created_at
		FROM chats WHERE workspace_root = ? AND lower(content || ' ' || context_rel_path || ' ' || source_paths_json) LIKE ? ORDER BY created_at DESC LIMIT ?`, workspaceRoot, like, limit)
	if err := appendRows("chat", rows, err); err != nil {
		return nil, err
	}
	rows, err = db.Query(`SELECT id, title, rel_path, substr(metadata_json, 1, 220), created_at
		FROM artifacts WHERE workspace_root = ? AND lower(rel_path || ' ' || title || ' ' || source || ' ' || metadata_json) LIKE ? ORDER BY created_at DESC LIMIT ?`, workspaceRoot, like, limit)
	if err := appendRows("artifact", rows, err); err != nil {
		return nil, err
	}
	rows, err = db.Query(`SELECT id, tool_name, target, substr(output_summary || ' ' || error || ' ' || inputs_json, 1, 220), started_at
		FROM tool_runs WHERE workspace_root = ? AND lower(tool_name || ' ' || target || ' ' || output_summary || ' ' || error || ' ' || inputs_json) LIKE ? ORDER BY started_at DESC LIMIT ?`, workspaceRoot, like, limit)
	if err := appendRows("tool", rows, err); err != nil {
		return nil, err
	}
	rows, err = db.Query(`SELECT id, rel_path, coalesce(status, ''), substr(coalesce(sql_text, '') || ' ' || coalesce(message, '') || ' ' || coalesce(engine, ''), 1, 220), created_at
		FROM sql_runs WHERE workspace_root = ? AND lower(rel_path || ' ' || coalesce(sql_text, '') || ' ' || coalesce(message, '') || ' ' || coalesce(status, '')) LIKE ? ORDER BY created_at DESC LIMIT ?`, workspaceRoot, like, limit)
	if err := appendRows("sql-run", rows, err); err != nil {
		return nil, err
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func ListChats(root string) ([]ChatMirror, error) {
	_, db, err := openExisting(root)
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
	hash := sha256.Sum256([]byte(schemaSQL))
	return SQLiteStatus{
		Path:          dbPath,
		SchemaPath:    filepath.Join(absRoot, filepath.FromSlash(metadataDirRelPath), schemaFileName),
		SchemaVersion: 1,
		SchemaHash:    hex.EncodeToString(hash[:]),
		Tables:        []string{},
	}, db, nil
}

func writableDB(root string) (string, *sql.DB, error) {
	workspaceRoot, err := filepath.Abs(root)
	if err != nil {
		return "", nil, err
	}

	dbPath := filepath.Join(workspaceRoot, filepath.FromSlash(metadataDirRelPath), "nexusdesk.sqlite")
	if !Exists(workspaceRoot) {
		status, err := Ensure(workspaceRoot)
		if err != nil {
			return "", nil, err
		}
		dbPath = status.Path
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", nil, err
	}
	return workspaceRoot, db, nil
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

const schemaSQL = `-- Nexus Augentic Studio SQLite metadata schema v1
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

CREATE TABLE IF NOT EXISTS dataset_dependencies (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    rel_path TEXT NOT NULL,
    kind TEXT NOT NULL,
    target TEXT,
    query TEXT,
    artifact TEXT,
    created_at TEXT NOT NULL,
    last_refresh TEXT
);

CREATE TABLE IF NOT EXISTS sql_runs (
    id TEXT PRIMARY KEY,
    workspace_root TEXT NOT NULL,
    rel_path TEXT NOT NULL,
    sql_text TEXT NOT NULL,
    engine TEXT NOT NULL,
    rows_returned INTEGER,
    artifact TEXT,
    status TEXT NOT NULL,
    message TEXT,
    created_at TEXT NOT NULL
);
`

func HasSchemaTable(schema string, table string) bool {
	return strings.Contains(schema, "CREATE TABLE IF NOT EXISTS "+table)
}
