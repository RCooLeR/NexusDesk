package metadata

import (
	"database/sql"
	"encoding/json"

	jobssvc "nexusdesk/internal/services/jobs"
)

func (s *Store) SaveJob(job jobssvc.Job) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	logTail, _ := json.Marshal(job.LogTail)
	_, err = db.Exec(
		`INSERT INTO jobs (id, workspace_root, kind, label, status, message, error, log_tail_json, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    kind = excluded.kind,
		    label = excluded.label,
		    status = excluded.status,
		    message = excluded.message,
		    error = excluded.error,
		    log_tail_json = excluded.log_tail_json,
		    completed_at = excluded.completed_at`,
		job.ID,
		s.root,
		job.Kind,
		job.Label,
		string(job.Status),
		job.Message,
		job.Error,
		string(logTail),
		formatTime(job.StartedAt),
		formatTime(job.CompletedAt),
	)
	return err
}

func (s *Store) ListJobs() ([]jobssvc.Job, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT id, kind, label, status, message, error, log_tail_json, started_at, completed_at
		 FROM jobs WHERE workspace_root = ? ORDER BY started_at DESC, id DESC LIMIT 200`,
		s.root,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []jobssvc.Job{}
	for rows.Next() {
		var job jobssvc.Job
		var status string
		var logTail string
		var started string
		var completed string
		if err := rows.Scan(&job.ID, &job.Kind, &job.Label, &status, &job.Message, &job.Error, &logTail, &started, &completed); err != nil {
			return nil, err
		}
		job.Status = jobssvc.Status(status)
		job.StartedAt = parseTime(started)
		job.CompletedAt = parseTime(completed)
		_ = json.Unmarshal([]byte(logTail), &job.LogTail)
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) DeleteJobs(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM task_runs WHERE workspace_root = ? AND job_id = ?`, s.root, id); err != nil {
			_ = tx.Rollback()
			return err
		}
		if _, err := tx.Exec(`DELETE FROM jobs WHERE workspace_root = ? AND id = ?`, s.root, id); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) SaveTaskRun(record TaskRunRecord) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	record = s.NormalizeTaskRunRecord(record)
	_, err = db.Exec(
		`INSERT INTO task_runs (id, workspace_root, job_id, task_id, kind, label, command, cwd, source, status, exit_code, stdout, stderr, message, artifact_path, started_at, completed_at, duration_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    status = excluded.status,
		    exit_code = excluded.exit_code,
		    stdout = excluded.stdout,
		    stderr = excluded.stderr,
		    message = excluded.message,
		    artifact_path = excluded.artifact_path,
		    completed_at = excluded.completed_at,
		    duration_ms = excluded.duration_ms`,
		record.ID,
		s.root,
		record.JobID,
		record.TaskID,
		record.Kind,
		record.Label,
		record.Command,
		record.Cwd,
		record.Source,
		record.Status,
		record.ExitCode,
		record.Stdout,
		record.Stderr,
		record.Message,
		record.ArtifactPath,
		formatTime(record.StartedAt),
		formatTime(record.CompletedAt),
		record.DurationMs,
	)
	return err
}

func (s *Store) ListTaskRuns(limit int) ([]TaskRunRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT id, job_id, task_id, kind, label, command, cwd, source, status, exit_code, stdout, stderr, message, artifact_path, started_at, completed_at, duration_ms
		 FROM task_runs WHERE workspace_root = ? ORDER BY started_at DESC, id DESC LIMIT ?`,
		s.root,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []TaskRunRecord{}
	for rows.Next() {
		var record TaskRunRecord
		var started string
		var completed string
		if err := rows.Scan(
			&record.ID,
			&record.JobID,
			&record.TaskID,
			&record.Kind,
			&record.Label,
			&record.Command,
			&record.Cwd,
			&record.Source,
			&record.Status,
			&record.ExitCode,
			&record.Stdout,
			&record.Stderr,
			&record.Message,
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

func (s *Store) LatestTaskRunForJob(jobID string) (TaskRunRecord, bool, error) {
	db, err := s.open()
	if err != nil {
		return TaskRunRecord{}, false, err
	}
	row := db.QueryRow(
		`SELECT id, job_id, task_id, kind, label, command, cwd, source, status, exit_code, stdout, stderr, message, artifact_path, started_at, completed_at, duration_ms
		 FROM task_runs WHERE workspace_root = ? AND job_id = ? ORDER BY started_at DESC, id DESC LIMIT 1`,
		s.root,
		jobID,
	)
	var record TaskRunRecord
	var started string
	var completed string
	if err := row.Scan(
		&record.ID,
		&record.JobID,
		&record.TaskID,
		&record.Kind,
		&record.Label,
		&record.Command,
		&record.Cwd,
		&record.Source,
		&record.Status,
		&record.ExitCode,
		&record.Stdout,
		&record.Stderr,
		&record.Message,
		&record.ArtifactPath,
		&started,
		&completed,
		&record.DurationMs,
	); err != nil {
		if err == sql.ErrNoRows {
			return TaskRunRecord{}, false, nil
		}
		return TaskRunRecord{}, false, err
	}
	record.StartedAt = parseTime(started)
	record.CompletedAt = parseTime(completed)
	return record, true, nil
}
