package metadata

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func (s *Store) SaveArtifact(record ArtifactRecord) error {
	record = s.NormalizeArtifactRecord(record)
	if record.Kind == "" {
		return errors.New("artifact kind is required")
	}
	if record.RelPath == "" {
		return errors.New("artifact relative path is required")
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	sourcePathsJSON, _ := json.Marshal(record.SourcePaths)
	_, err = db.Exec(
		`INSERT INTO artifacts (id, workspace_root, kind, title, rel_path, metadata_path, size, job_id, task_id, source, source_paths_json, archived, created_at, generated_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(workspace_root, rel_path) DO UPDATE SET
		    kind = excluded.kind,
		    title = excluded.title,
		    metadata_path = excluded.metadata_path,
		    size = excluded.size,
		    job_id = excluded.job_id,
		    task_id = excluded.task_id,
		    source = excluded.source,
		    source_paths_json = excluded.source_paths_json,
		    archived = excluded.archived,
		    created_at = excluded.created_at,
		    generated_at = excluded.generated_at,
		    updated_at = excluded.updated_at`,
		record.ID,
		s.root,
		record.Kind,
		record.Title,
		record.RelPath,
		record.MetadataPath,
		record.Size,
		record.JobID,
		record.TaskID,
		record.Source,
		string(sourcePathsJSON),
		boolInt(record.Archived),
		formatTime(record.CreatedAt),
		formatTime(record.GeneratedAt),
		formatTime(record.UpdatedAt),
	)
	return err
}

func (s *Store) DeleteArtifact(relPath string) error {
	relPath = strings.TrimSpace(relPath)
	if relPath == "" {
		return errors.New("artifact relative path is required")
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	_, err = db.Exec(`DELETE FROM artifacts WHERE workspace_root = ? AND rel_path = ?`, s.root, relPath)
	return err
}

func (s *Store) ListArtifacts(query string, includeArchived bool, limit int) ([]ArtifactRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	records := []ArtifactRecord{}
	if strings.TrimSpace(query) == "" {
		rows, err := db.Query(
			`SELECT id, kind, title, rel_path, metadata_path, size, job_id, task_id, source, source_paths_json, archived, created_at, generated_at, updated_at
			 FROM artifacts
			 WHERE workspace_root = ? AND (? OR archived = 0)
			 ORDER BY COALESCE(generated_at, created_at, updated_at) DESC, rel_path DESC
			 LIMIT ?`,
			s.root,
			boolInt(includeArchived),
			limit,
		)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		return scanArtifactRecords(rows)
	}
	like := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := db.Query(
		`SELECT id, kind, title, rel_path, metadata_path, size, job_id, task_id, source, source_paths_json, archived, created_at, generated_at, updated_at
		 FROM artifacts
		 WHERE workspace_root = ?
		   AND (? OR archived = 0)
		   AND (
		     lower(kind) LIKE ?
		     OR lower(title) LIKE ?
		     OR lower(rel_path) LIKE ?
		     OR lower(job_id) LIKE ?
		     OR lower(task_id) LIKE ?
		     OR lower(source) LIKE ?
		     OR lower(source_paths_json) LIKE ?
		   )
		 ORDER BY COALESCE(generated_at, created_at, updated_at) DESC, rel_path DESC
		 LIMIT ?`,
		s.root,
		boolInt(includeArchived),
		like,
		like,
		like,
		like,
		like,
		like,
		like,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records, err = scanArtifactRecords(rows)
	if err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Store) NormalizeArtifactRecord(record ArtifactRecord) ArtifactRecord {
	record.Kind = strings.TrimSpace(record.Kind)
	record.RelPath = strings.TrimSpace(record.RelPath)
	if record.UpdatedAt.IsZero() {
		record.UpdatedAt = time.Now().UTC()
	}
	if record.ID == "" {
		record.ID = hashID(s.root + "|artifact|" + record.RelPath)
	}
	return record
}

type artifactRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanArtifactRecords(rows artifactRows) ([]ArtifactRecord, error) {
	records := []ArtifactRecord{}
	for rows.Next() {
		var record ArtifactRecord
		var sourcePathsJSON string
		var archived int
		var created string
		var generated string
		var updated string
		if err := rows.Scan(
			&record.ID,
			&record.Kind,
			&record.Title,
			&record.RelPath,
			&record.MetadataPath,
			&record.Size,
			&record.JobID,
			&record.TaskID,
			&record.Source,
			&sourcePathsJSON,
			&archived,
			&created,
			&generated,
			&updated,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(sourcePathsJSON), &record.SourcePaths)
		record.Archived = archived != 0
		record.CreatedAt = parseTime(created)
		record.GeneratedAt = parseTime(generated)
		record.UpdatedAt = parseTime(updated)
		records = append(records, record)
	}
	return records, rows.Err()
}
