package metadata

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"time"
)

type compatibilitySQLRun struct {
	ID        string
	RelPath   string
	SQL       string
	Engine    string
	Rows      int
	Artifact  string
	Status    string
	Message   string
	CreatedAt string
}

type compatibilityDatasetDependency struct {
	ID          string
	RelPath     string
	Kind        string
	Target      string
	Query       string
	Artifact    string
	CreatedAt   string
	LastRefresh string
}

func (s *Store) importCompatibilitySQLiteDatasets(ctx context.Context) (int, int, int, error) {
	if err := compatibilityContextErr(ctx); err != nil {
		return 0, 0, 0, err
	}
	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		return 0, 0, 0, nil
	} else if err != nil {
		return 0, 0, 0, err
	}
	if err := s.closeCachedDB(); err != nil {
		return 0, 0, 0, err
	}
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return 0, 0, 0, err
	}
	if err := backupLegacyCompatibilityTables(db); err != nil {
		db.Close()
		return 0, 0, 0, err
	}
	sqlRuns, sqlLegacy, sqlSkipped, err := s.readCompatibilitySQLRuns(db)
	if err != nil {
		db.Close()
		return 0, 0, 0, err
	}
	dependencies, dependencyLegacy, dependencySkipped, err := s.readCompatibilityDatasetDependencies(db)
	if err != nil {
		db.Close()
		return 0, 0, 0, err
	}
	if !sqlLegacy && !dependencyLegacy {
		db.Close()
		return 0, 0, sqlSkipped + dependencySkipped, nil
	}
	if err := compatibilityContextErr(ctx); err != nil {
		db.Close()
		return 0, 0, sqlSkipped + dependencySkipped, err
	}
	if sqlLegacy {
		if err := renameCompatibilityTable(db, "sql_runs", "legacy_sql_runs"); err != nil {
			db.Close()
			return 0, 0, 0, err
		}
	}
	if dependencyLegacy {
		if err := renameCompatibilityTable(db, "dataset_dependencies", "legacy_dataset_dependencies"); err != nil {
			db.Close()
			return 0, 0, 0, err
		}
	}
	db.Close()
	if _, err := s.Ensure(); err != nil {
		return 0, 0, 0, err
	}
	importedSQL := 0
	importedDependencies := 0
	skipped := sqlSkipped + dependencySkipped
	for index, item := range sqlRuns {
		if index%64 == 0 {
			if err := compatibilityContextErr(ctx); err != nil {
				return importedSQL, importedDependencies, skipped, err
			}
		}
		record := s.compatibilitySQLRunRecord(item)
		if err := s.SaveSQLRun(record); err != nil {
			skipped++
			continue
		}
		importedSQL++
	}
	for index, item := range dependencies {
		if index%64 == 0 {
			if err := compatibilityContextErr(ctx); err != nil {
				return importedSQL, importedDependencies, skipped, err
			}
		}
		record := s.compatibilityDatasetDependencyRecord(item)
		if record.SourcePath == "" || record.DependentRef == "" {
			skipped++
			continue
		}
		if err := s.SaveDatasetDependency(record); err != nil {
			skipped++
			continue
		}
		importedDependencies++
	}
	return importedSQL, importedDependencies, skipped, nil
}

func (s *Store) readCompatibilitySQLRuns(db *sql.DB) ([]compatibilitySQLRun, bool, int, error) {
	columns, ok, err := tableColumns(db, "sql_runs")
	if err != nil || !ok {
		return nil, false, 0, err
	}
	if !columns["rows_returned"] || columns["row_count"] {
		return nil, false, 0, nil
	}
	rows, err := db.Query(`SELECT id, rel_path, sql_text, engine, rows_returned, artifact, status, message, created_at
		FROM sql_runs WHERE workspace_root = ? ORDER BY created_at DESC`, s.root)
	if err != nil {
		return nil, true, 0, err
	}
	defer rows.Close()
	items := []compatibilitySQLRun{}
	skipped := 0
	for rows.Next() {
		var item compatibilitySQLRun
		if err := rows.Scan(&item.ID, &item.RelPath, &item.SQL, &item.Engine, &item.Rows, &item.Artifact, &item.Status, &item.Message, &item.CreatedAt); err != nil {
			skipped++
			continue
		}
		if strings.TrimSpace(item.RelPath) == "" || strings.TrimSpace(item.SQL) == "" {
			skipped++
			continue
		}
		items = append(items, item)
	}
	return items, true, skipped, rows.Err()
}

func (s *Store) readCompatibilityDatasetDependencies(db *sql.DB) ([]compatibilityDatasetDependency, bool, int, error) {
	columns, ok, err := tableColumns(db, "dataset_dependencies")
	if err != nil || !ok {
		return nil, false, 0, err
	}
	if !columns["rel_path"] || columns["source_path"] {
		return nil, false, 0, nil
	}
	rows, err := db.Query(`SELECT id, rel_path, kind, target, query, artifact, created_at, last_refresh
		FROM dataset_dependencies WHERE workspace_root = ? ORDER BY created_at DESC`, s.root)
	if err != nil {
		return nil, true, 0, err
	}
	defer rows.Close()
	items := []compatibilityDatasetDependency{}
	skipped := 0
	for rows.Next() {
		var item compatibilityDatasetDependency
		if err := rows.Scan(&item.ID, &item.RelPath, &item.Kind, &item.Target, &item.Query, &item.Artifact, &item.CreatedAt, &item.LastRefresh); err != nil {
			skipped++
			continue
		}
		if strings.TrimSpace(item.RelPath) == "" {
			skipped++
			continue
		}
		items = append(items, item)
	}
	return items, true, skipped, rows.Err()
}

func (s *Store) compatibilitySQLRunRecord(item compatibilitySQLRun) SQLRunRecord {
	started := parseCompatibilityTime(item.CreatedAt)
	return SQLRunRecord{
		ID:           strings.TrimSpace(item.ID),
		RelPath:      strings.TrimSpace(item.RelPath),
		SQL:          strings.TrimSpace(item.SQL),
		Engine:       firstNonEmptyString(item.Engine, "legacy-dataset-sql"),
		Status:       compatibilitySQLStatus(item.Status),
		RowCount:     item.Rows,
		MatchedRows:  item.Rows,
		ShownRows:    item.Rows,
		Message:      item.Message,
		ArtifactPath: item.Artifact,
		StartedAt:    started,
		CompletedAt:  started,
	}
}

func (s *Store) compatibilityDatasetDependencyRecord(item compatibilityDatasetDependency) DatasetDependencyRecord {
	created := parseCompatibilityTime(item.CreatedAt)
	updated := parseCompatibilityTime(item.LastRefresh)
	if updated.IsZero() {
		updated = created
	}
	ref := firstNonEmptyString(item.Target, item.Artifact, item.ID)
	if ref == "" {
		ref = hashID(strings.Join([]string{s.root, item.RelPath, item.Kind, item.Query}, "|"))
	}
	metadata := map[string]string{}
	addMetadata(metadata, "legacyKind", item.Kind)
	addMetadata(metadata, "target", item.Target)
	addMetadata(metadata, "query", item.Query)
	addMetadata(metadata, "artifact", item.Artifact)
	return DatasetDependencyRecord{
		ID:            strings.TrimSpace(item.ID),
		SourcePath:    strings.TrimSpace(item.RelPath),
		DependentKind: firstNonEmptyString(item.Kind, "legacy-dataset-dependency"),
		DependentRef:  ref,
		Relation:      compatibilityDatasetRelation(item),
		Metadata:      metadata,
		CreatedAt:     created,
		UpdatedAt:     updated,
	}
}

func tableColumns(db *sql.DB, table string) (map[string]bool, bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return nil, false, err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return columns, len(columns) > 0, nil
}

func renameCompatibilityTable(db *sql.DB, table string, backup string) error {
	exists, err := tableExists(db, backup)
	if err != nil {
		return err
	}
	if exists {
		backup = backup + "_" + time.Now().UTC().Format("20060102150405")
	}
	_, err = db.Exec("ALTER TABLE " + table + " RENAME TO " + backup)
	return err
}

func backupLegacyCompatibilityTables(db *sql.DB) error {
	for _, item := range []struct {
		table     string
		backup    string
		nativeKey string
	}{
		{table: "artifacts", backup: "legacy_artifacts", nativeKey: "metadata_path"},
		{table: "tool_runs", backup: "legacy_tool_runs", nativeKey: "agent_run_id"},
	} {
		columns, ok, err := tableColumns(db, item.table)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if columns[item.nativeKey] {
			continue
		}
		if err := renameCompatibilityTable(db, item.table, item.backup); err != nil {
			return err
		}
	}
	return nil
}

func tableExists(db *sql.DB, table string) (bool, error) {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

func compatibilitySQLStatus(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "completed", "complete", "ok":
		return "success"
	case "":
		return "success"
	default:
		return value
	}
}

func compatibilityDatasetRelation(item compatibilityDatasetDependency) string {
	kind := strings.ToLower(item.Kind)
	switch {
	case strings.TrimSpace(item.Artifact) != "":
		return "generates"
	case strings.Contains(kind, "sql") || strings.TrimSpace(item.Query) != "":
		return "saves"
	default:
		return "links"
	}
}

func addMetadata(metadata map[string]string, key string, value string) {
	value = strings.TrimSpace(value)
	if value != "" {
		metadata[key] = value
	}
}
