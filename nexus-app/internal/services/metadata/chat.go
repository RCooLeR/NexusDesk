package metadata

import (
	"encoding/json"
	"errors"
	"strings"
	"time"
)

func (s *Store) SaveChatMessage(record ChatMessageRecord) error {
	record.Role = normalizeChatRole(record.Role)
	record.Content = strings.TrimSpace(record.Content)
	if record.Role == "" {
		return errors.New("chat role must be user or assistant")
	}
	if record.Content == "" {
		return errors.New("chat content is required")
	}
	if record.CreatedAt.IsZero() {
		record.CreatedAt = time.Now().UTC()
	}
	db, err := s.open()
	if err != nil {
		return err
	}
	defer db.Close()
	record = s.NormalizeChatMessageRecord(record)
	sourcePathsJSON, _ := json.Marshal(record.SourcePaths)
	_, err = db.Exec(
		`INSERT INTO chat_messages (id, workspace_root, role, content, model, source_paths_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		    role = excluded.role,
		    content = excluded.content,
		    model = excluded.model,
		    source_paths_json = excluded.source_paths_json,
		    created_at = excluded.created_at`,
		record.ID,
		s.root,
		record.Role,
		record.Content,
		record.Model,
		string(sourcePathsJSON),
		formatTime(record.CreatedAt),
	)
	return err
}

func (s *Store) ListChatMessages(limit int) ([]ChatMessageRecord, error) {
	if limit <= 0 || limit > 100 {
		limit = 24
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.Query(
		`SELECT id, role, content, model, source_paths_json, created_at
		 FROM (
		    SELECT id, role, content, model, source_paths_json, created_at
		    FROM chat_messages
		    WHERE workspace_root = ?
		    ORDER BY created_at DESC, id DESC
		    LIMIT ?
		 )
		 ORDER BY created_at ASC, id ASC`,
		s.root,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []ChatMessageRecord{}
	for rows.Next() {
		var record ChatMessageRecord
		var sourcePathsJSON string
		var created string
		if err := rows.Scan(&record.ID, &record.Role, &record.Content, &record.Model, &sourcePathsJSON, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(sourcePathsJSON), &record.SourcePaths)
		record.CreatedAt = parseTime(created)
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) NormalizeChatMessageRecord(record ChatMessageRecord) ChatMessageRecord {
	if record.ID == "" {
		record.ID = hashID(s.root + "|chat|" + record.Role + "|" + formatTime(record.CreatedAt) + "|" + record.Content)
	}
	return record
}

func normalizeChatRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "user":
		return "user"
	case "assistant":
		return "assistant"
	default:
		return ""
	}
}
