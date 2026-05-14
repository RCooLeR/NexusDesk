package approval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const approvalDirRelPath = ".nexusdesk/approvals"
const approvalLogName = "log.json"
const maxApprovalRecords = 200

type Record struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Target    string `json:"target"`
	Risk      string `json:"risk"`
	Decision  string `json:"decision"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

func Append(root string, record Record) ([]Record, error) {
	if strings.TrimSpace(root) == "" {
		return []Record{}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	items, err := read(absRoot)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	record.Action = strings.TrimSpace(record.Action)
	record.Target = strings.TrimSpace(record.Target)
	record.Risk = fallback(record.Risk, "medium")
	record.Decision = fallback(record.Decision, "applied")
	record.CreatedAt = fallback(record.CreatedAt, now.Format(time.RFC3339))
	record.ID = fallback(record.ID, now.Format("20060102T150405.000000000Z")+"-"+slug(record.Action))
	items = append([]Record{record}, items...)
	if len(items) > maxApprovalRecords {
		items = items[:maxApprovalRecords]
	}
	if err := write(absRoot, items); err != nil {
		return nil, err
	}
	return items, nil
}

func List(root string) ([]Record, error) {
	if strings.TrimSpace(root) == "" {
		return []Record{}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return read(absRoot)
}

func read(absRoot string) ([]Record, error) {
	path := filepath.Join(absRoot, filepath.FromSlash(approvalDirRelPath), approvalLogName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []Record{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []Record{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func write(absRoot string, items []Record) error {
	dir := filepath.Join(absRoot, filepath.FromSlash(approvalDirRelPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, approvalLogName), append(data, '\n'), 0o644)
}

func fallback(value string, next string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return next
	}
	return value
}

func slug(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "approval"
	}
	return value
}
