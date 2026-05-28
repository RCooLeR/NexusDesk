package metadata

import "encoding/json"

func (s *Store) SaveAgentRun(record AgentRunRecord) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	record = s.NormalizeAgentRunRecord(record)
	planJSON, _ := json.Marshal(record.Plan)
	sourcePathsJSON, _ := json.Marshal(record.SourcePaths)
	_, err = db.Exec(
		`INSERT INTO agent_runs (id, workspace_root, job_id, prompt, status, message, iterations, stop_reason, plan_json, source_paths_json, started_at, completed_at, duration_ms)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    job_id = excluded.job_id,
		    status = excluded.status,
		    message = excluded.message,
		    iterations = excluded.iterations,
		    stop_reason = excluded.stop_reason,
		    plan_json = excluded.plan_json,
		    source_paths_json = excluded.source_paths_json,
		    completed_at = excluded.completed_at,
		    duration_ms = excluded.duration_ms`,
		record.ID,
		s.root,
		record.JobID,
		record.Prompt,
		record.Status,
		record.Message,
		record.Iterations,
		record.StopReason,
		string(planJSON),
		string(sourcePathsJSON),
		formatTime(record.StartedAt),
		formatTime(record.CompletedAt),
		record.DurationMs,
	)
	return err
}

func (s *Store) SaveToolRun(record ToolRunRecord) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	record = s.NormalizeToolRunRecord(record)
	argsJSON, _ := json.Marshal(record.Args)
	_, err = db.Exec(
		`INSERT INTO tool_runs (id, workspace_root, agent_run_id, job_id, sequence, tool_name, risk, mutated, args_json, observation, error, started_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    sequence = excluded.sequence,
		    risk = excluded.risk,
		    mutated = excluded.mutated,
		    args_json = excluded.args_json,
		    observation = excluded.observation,
		    error = excluded.error,
		    completed_at = excluded.completed_at`,
		record.ID,
		s.root,
		record.AgentRunID,
		record.JobID,
		record.Sequence,
		record.ToolName,
		record.Risk,
		boolInt(record.Mutated),
		string(argsJSON),
		record.Observation,
		record.Error,
		formatTime(record.StartedAt),
		formatTime(record.CompletedAt),
	)
	return err
}

func (s *Store) ListAgentRuns(limit int) ([]AgentRunRecord, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT id, job_id, prompt, status, message, iterations, stop_reason, plan_json, source_paths_json, started_at, completed_at, duration_ms
		 FROM agent_runs WHERE workspace_root = ? ORDER BY started_at DESC, id DESC LIMIT ?`,
		s.root,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []AgentRunRecord{}
	for rows.Next() {
		var record AgentRunRecord
		var planJSON string
		var sourcePathsJSON string
		var started string
		var completed string
		if err := rows.Scan(&record.ID, &record.JobID, &record.Prompt, &record.Status, &record.Message, &record.Iterations, &record.StopReason, &planJSON, &sourcePathsJSON, &started, &completed, &record.DurationMs); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(planJSON), &record.Plan)
		_ = json.Unmarshal([]byte(sourcePathsJSON), &record.SourcePaths)
		record.StartedAt = parseTime(started)
		record.CompletedAt = parseTime(completed)
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) ListToolRuns(agentRunID string) ([]ToolRunRecord, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT id, agent_run_id, job_id, sequence, tool_name, risk, mutated, args_json, observation, error, started_at, completed_at
		 FROM tool_runs WHERE workspace_root = ? AND agent_run_id = ? ORDER BY sequence ASC, started_at ASC, id ASC`,
		s.root,
		agentRunID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []ToolRunRecord{}
	for rows.Next() {
		var record ToolRunRecord
		var mutated int
		var argsJSON string
		var started string
		var completed string
		if err := rows.Scan(&record.ID, &record.AgentRunID, &record.JobID, &record.Sequence, &record.ToolName, &record.Risk, &mutated, &argsJSON, &record.Observation, &record.Error, &started, &completed); err != nil {
			return nil, err
		}
		record.Mutated = mutated != 0
		_ = json.Unmarshal([]byte(argsJSON), &record.Args)
		record.StartedAt = parseTime(started)
		record.CompletedAt = parseTime(completed)
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) NormalizeAgentRunRecord(record AgentRunRecord) AgentRunRecord {
	if record.ID == "" {
		record.ID = hashID(s.root + "|agent|" + record.JobID + "|" + record.Prompt + "|" + formatTime(record.StartedAt))
	}
	return record
}

func (s *Store) NormalizeToolRunRecord(record ToolRunRecord) ToolRunRecord {
	if record.ID == "" {
		record.ID = hashID(s.root + "|tool|" + record.AgentRunID + "|" + record.ToolName + "|" + formatTime(record.StartedAt) + "|" + record.Observation)
	}
	return record
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
