package agenttools

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const toolRunDirRelPath = ".nexusdesk/tool-runs"
const toolRunLogName = "log.json"
const maxToolRunRecords = 300

type RunRequest struct {
	ToolName   string            `json:"toolName"`
	Target     string            `json:"target"`
	Inputs     map[string]string `json:"inputs"`
	Approved   bool              `json:"approved"`
	ApprovalID string            `json:"approvalId"`
}

type RunRecord struct {
	ID               string            `json:"id"`
	ToolName         string            `json:"toolName"`
	Title            string            `json:"title"`
	Target           string            `json:"target"`
	Risk             string            `json:"risk"`
	RequiresApproval bool              `json:"requiresApproval"`
	Status           string            `json:"status"`
	Mode             string            `json:"mode"`
	Inputs           map[string]string `json:"inputs"`
	OutputSummary    string            `json:"outputSummary"`
	Error            string            `json:"error"`
	ApprovalID       string            `json:"approvalId"`
	StartedAt        string            `json:"startedAt"`
	CompletedAt      string            `json:"completedAt"`
	DurationMs       int64             `json:"durationMs"`
}

func Find(name string) (Descriptor, bool) {
	for _, descriptor := range Registry() {
		if descriptor.Name == name {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

func NewRecord(request RunRequest, descriptor Descriptor, mode string, status string, startedAt time.Time) RunRecord {
	return RunRecord{
		ID:               startedAt.UTC().Format("20060102T150405.000000000Z") + "-" + slug(request.ToolName),
		ToolName:         descriptor.Name,
		Title:            descriptor.Title,
		Target:           strings.TrimSpace(request.Target),
		Risk:             descriptor.Risk,
		RequiresApproval: descriptor.RequiresApproval,
		Status:           status,
		Mode:             mode,
		Inputs:           cleanInputs(request.Inputs),
		ApprovalID:       strings.TrimSpace(request.ApprovalID),
		StartedAt:        startedAt.UTC().Format(time.RFC3339Nano),
	}
}

func FinishRecord(record RunRecord, status string, summary string, runErr error, completedAt time.Time) RunRecord {
	record.Status = status
	record.OutputSummary = strings.TrimSpace(summary)
	if runErr != nil {
		record.Status = "failed"
		record.Error = runErr.Error()
	}
	record.CompletedAt = completedAt.UTC().Format(time.RFC3339Nano)
	if startedAt, err := time.Parse(time.RFC3339Nano, record.StartedAt); err == nil {
		record.DurationMs = completedAt.Sub(startedAt).Milliseconds()
	}
	return record
}

func Append(root string, record RunRecord) ([]RunRecord, error) {
	if strings.TrimSpace(root) == "" {
		return []RunRecord{}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	items, err := read(absRoot)
	if err != nil {
		return nil, err
	}
	items = append([]RunRecord{record}, items...)
	if len(items) > maxToolRunRecords {
		items = items[:maxToolRunRecords]
	}
	if err := write(absRoot, items); err != nil {
		return nil, err
	}
	return items, nil
}

func List(root string) ([]RunRecord, error) {
	if strings.TrimSpace(root) == "" {
		return []RunRecord{}, nil
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	return read(absRoot)
}

func RequireDescriptor(name string) (Descriptor, error) {
	descriptor, ok := Find(strings.TrimSpace(name))
	if !ok {
		return Descriptor{}, errors.New("agent tool is not registered")
	}
	return descriptor, nil
}

func read(absRoot string) ([]RunRecord, error) {
	path := filepath.Join(absRoot, filepath.FromSlash(toolRunDirRelPath), toolRunLogName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []RunRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []RunRecord{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func write(absRoot string, items []RunRecord) error {
	dir := filepath.Join(absRoot, filepath.FromSlash(toolRunDirRelPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, toolRunLogName), append(data, '\n'), 0o644)
}

func cleanInputs(inputs map[string]string) map[string]string {
	cleaned := map[string]string{}
	for key, value := range inputs {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key != "" && value != "" {
			lowerKey := strings.ToLower(key)
			if (strings.Contains(lowerKey, "content") || strings.Contains(lowerKey, "patch") || strings.Contains(lowerKey, "diff")) && len(value) > 500 {
				value = value[:500] + "... [truncated]"
			}
			cleaned[key] = value
		}
	}
	return cleaned
}

func slug(value string) string {
	value = strings.ToLower(value)
	value = strings.ReplaceAll(value, ".", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "tool"
	}
	return value
}
