package metadata

import (
	"errors"
	"strings"
	"time"
)

func (s *Store) SaveApprovalRecord(record ApprovalRecord) error {
	record = s.NormalizeApprovalRecord(record)
	if record.Action == "" {
		return errors.New("approval action is required")
	}
	if record.Risk == "" {
		return errors.New("approval risk is required")
	}
	if record.Decision == "" {
		return errors.New("approval decision is required")
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	_, err = db.Exec(
		`INSERT INTO approval_records (id, workspace_root, action, target, risk, decision, message, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    action = excluded.action,
		    target = excluded.target,
		    risk = excluded.risk,
		    decision = excluded.decision,
		    message = excluded.message,
		    created_at = excluded.created_at`,
		record.ID,
		s.root,
		record.Action,
		record.Target,
		record.Risk,
		record.Decision,
		record.Message,
		formatTime(record.CreatedAt),
	)
	return err
}

func (s *Store) ListApprovalRecords(limit int) ([]ApprovalRecord, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(
		`SELECT id, action, target, risk, decision, message, created_at
		 FROM approval_records WHERE workspace_root = ? ORDER BY created_at DESC, id DESC LIMIT ?`,
		s.root,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []ApprovalRecord{}
	for rows.Next() {
		var record ApprovalRecord
		var created string
		if err := rows.Scan(&record.ID, &record.Action, &record.Target, &record.Risk, &record.Decision, &record.Message, &created); err != nil {
			return nil, err
		}
		record.CreatedAt = parseTime(created)
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) NormalizeApprovalRecord(record ApprovalRecord) ApprovalRecord {
	record.Action = strings.TrimSpace(record.Action)
	record.Target = strings.TrimSpace(record.Target)
	record.Risk = strings.TrimSpace(record.Risk)
	record.Decision = strings.TrimSpace(record.Decision)
	record.Message = strings.TrimSpace(record.Message)
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	if record.ID == "" {
		record.ID = hashID(strings.Join([]string{s.root, "approval", record.Action, record.Target, record.Decision, formatTime(record.CreatedAt)}, "|"))
	}
	return record
}
