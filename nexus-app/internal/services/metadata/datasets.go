package metadata

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func (s *Store) SaveSQLRun(record SQLRunRecord) error {
	record = s.NormalizeSQLRunRecord(record)
	if record.RelPath == "" {
		return errors.New("sql run source path is required")
	}
	if record.SQL == "" {
		return errors.New("sql text is required")
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(
		`INSERT INTO sql_runs (id, workspace_root, rel_path, sql_text, engine, status, row_count, matched_rows, shown_rows, message, error, artifact_path, started_at, completed_at, duration_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    status = excluded.status,
		    row_count = excluded.row_count,
		    matched_rows = excluded.matched_rows,
		    shown_rows = excluded.shown_rows,
		    message = excluded.message,
		    error = excluded.error,
		    artifact_path = excluded.artifact_path,
		    completed_at = excluded.completed_at,
		    duration_ms = excluded.duration_ms`,
		record.ID,
		s.root,
		record.RelPath,
		record.SQL,
		record.Engine,
		record.Status,
		record.RowCount,
		record.MatchedRows,
		record.ShownRows,
		record.Message,
		record.Error,
		record.ArtifactPath,
		formatTime(record.StartedAt),
		formatTime(record.CompletedAt),
		record.DurationMs,
	)
	return err
}

func (s *Store) ListSQLRuns(limit int) ([]SQLRunRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(
		`SELECT id, rel_path, sql_text, engine, status, row_count, matched_rows, shown_rows, message, error, artifact_path, started_at, completed_at, duration_ms
		 FROM sql_runs WHERE workspace_root = ? ORDER BY started_at DESC, id DESC LIMIT ?`,
		s.root,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []SQLRunRecord{}
	for rows.Next() {
		var record SQLRunRecord
		var started string
		var completed string
		if err := rows.Scan(
			&record.ID,
			&record.RelPath,
			&record.SQL,
			&record.Engine,
			&record.Status,
			&record.RowCount,
			&record.MatchedRows,
			&record.ShownRows,
			&record.Message,
			&record.Error,
			&record.ArtifactPath,
			&started,
			&completed,
			&record.DurationMs,
		); err != nil {
			return nil, err
		}
		record.StartedAt = parseTime(started)
		record.CompletedAt = parseTime(completed)
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) SaveDatasetDependency(record DatasetDependencyRecord) error {
	record = s.NormalizeDatasetDependencyRecord(record)
	if record.SourcePath == "" {
		return errors.New("dataset dependency source path is required")
	}
	if record.DependentKind == "" || record.DependentRef == "" {
		return errors.New("dataset dependency target is required")
	}
	if record.Relation == "" {
		return errors.New("dataset dependency relation is required")
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()
	metadataJSON, _ := json.Marshal(record.Metadata)
	_, err = db.Exec(
		`INSERT INTO dataset_dependencies (id, workspace_root, source_path, dependent_kind, dependent_ref, relation, metadata_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(workspace_root, source_path, dependent_kind, dependent_ref, relation) DO UPDATE SET
		    metadata_json = excluded.metadata_json,
		    updated_at = excluded.updated_at`,
		record.ID,
		s.root,
		record.SourcePath,
		record.DependentKind,
		record.DependentRef,
		record.Relation,
		string(metadataJSON),
		formatTime(record.CreatedAt),
		formatTime(record.UpdatedAt),
	)
	return err
}

func (s *Store) ListDatasetDependencies(sourcePath string, limit int) ([]DatasetDependencyRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT id, source_path, dependent_kind, dependent_ref, relation, metadata_json, created_at, updated_at
		FROM dataset_dependencies WHERE workspace_root = ?`
	args := []any{s.root}
	if strings.TrimSpace(sourcePath) != "" {
		query += ` AND source_path = ?`
		args = append(args, strings.TrimSpace(sourcePath))
	}
	query += ` ORDER BY updated_at DESC, id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []DatasetDependencyRecord{}
	for rows.Next() {
		var record DatasetDependencyRecord
		var metadataJSON string
		var created string
		var updated string
		if err := rows.Scan(
			&record.ID,
			&record.SourcePath,
			&record.DependentKind,
			&record.DependentRef,
			&record.Relation,
			&metadataJSON,
			&created,
			&updated,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(metadataJSON), &record.Metadata)
		record.CreatedAt = parseTime(created)
		record.UpdatedAt = parseTime(updated)
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) NormalizeSQLRunRecord(record SQLRunRecord) SQLRunRecord {
	record.RelPath = strings.TrimSpace(record.RelPath)
	record.SQL = strings.TrimSpace(record.SQL)
	if record.Engine == "" {
		record.Engine = "native-dataset-sql"
	}
	if record.Status == "" {
		record.Status = "success"
	}
	if record.StartedAt.IsZero() {
		record.StartedAt = time.Now().UTC()
	}
	if record.CompletedAt.IsZero() {
		record.CompletedAt = record.StartedAt
	}
	if record.ID == "" {
		record.ID = hashID(strings.Join([]string{s.root, "sql-run", record.RelPath, record.SQL, formatTime(record.StartedAt)}, "|"))
	}
	return record
}

func (s *Store) NormalizeDatasetDependencyRecord(record DatasetDependencyRecord) DatasetDependencyRecord {
	record.SourcePath = strings.TrimSpace(record.SourcePath)
	record.DependentKind = strings.TrimSpace(record.DependentKind)
	record.DependentRef = strings.TrimSpace(record.DependentRef)
	record.Relation = strings.TrimSpace(record.Relation)
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = record.CreatedAt
	}
	if record.ID == "" {
		record.ID = hashID(strings.Join([]string{s.root, "dataset-dependency", record.SourcePath, record.DependentKind, record.DependentRef, record.Relation}, "|"))
	}
	if record.Metadata == nil {
		record.Metadata = map[string]string{}
	}
	return record
}
