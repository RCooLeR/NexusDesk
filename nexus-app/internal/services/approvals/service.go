package approvals

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const approvalsDirRelPath = ".nexusdesk/approvals"
const approvalLogName = "log.json"
const policyName = "policy.json"
const maxApprovalRecords = 200

type Repository interface {
	SaveApprovalRecord(record Record) error
	ListApprovalRecords(limit int) ([]Record, error)
}

type Service struct {
	repository Repository
}

func New() *Service {
	return &Service{}
}

func (s *Service) SetRepository(repository Repository) {
	s.repository = repository
}

func (s *Service) Append(root string, record Record) ([]Record, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return nil, err
	}
	records, err := s.List(absRoot)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	record.Action = fallback(record.Action, "approval")
	record.Target = strings.TrimSpace(record.Target)
	record.Risk = fallback(record.Risk, "medium")
	record.Decision = fallback(record.Decision, "applied")
	record.Message = strings.TrimSpace(record.Message)
	if record.CreatedAt.IsZero() {
		record.CreatedAt = now
	}
	record.CreatedAt = record.CreatedAt.UTC()
	record.ID = fallback(record.ID, record.CreatedAt.Format("20060102T150405.000000000Z")+"-"+slug(record.Action))
	records = append([]Record{record}, records...)
	if len(records) > maxApprovalRecords {
		records = records[:maxApprovalRecords]
	}
	if err := writeJSON(logPath(absRoot), records); err != nil {
		return nil, err
	}
	if s.repository != nil {
		for _, record := range records {
			_ = s.repository.SaveApprovalRecord(record)
		}
	}
	return records, nil
}

func (s *Service) List(root string) ([]Record, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return nil, err
	}
	records := []Record{}
	if err := readJSON(logPath(absRoot), &records); err != nil {
		return nil, err
	}
	if s.repository != nil {
		metadataRecords, err := s.repository.ListApprovalRecords(maxApprovalRecords)
		if err == nil && len(metadataRecords) > 0 {
			return metadataRecords, nil
		}
	}
	return records, nil
}

func (s *Service) LoadPolicy(root string) (Policy, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return Policy{}, err
	}
	policy := Policy{}
	if err := readJSON(policyPath(absRoot), &policy); err != nil {
		return Policy{}, err
	}
	if policy.WorkspaceRoot == "" {
		policy.WorkspaceRoot = absRoot
	}
	if policy.FullProjectAccess && !policy.Active(time.Now().UTC()) {
		policy.FullProjectAccess = false
		policy.Message = "Full project access expired."
	}
	return policy, nil
}

func (s *Service) GrantFullProjectAccess(root string, duration time.Duration) (Policy, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return Policy{}, err
	}
	if duration <= 0 {
		duration = time.Hour
	}
	now := time.Now().UTC()
	policy := Policy{
		WorkspaceRoot:     absRoot,
		FullProjectAccess: true,
		GrantedAt:         now,
		ExpiresAt:         now.Add(duration),
		Message:           "Full project access granted for this workspace.",
	}
	if err := writeJSON(policyPath(absRoot), policy); err != nil {
		return Policy{}, err
	}
	_, err = s.Append(absRoot, Record{
		Action:   "access.full_project.grant",
		Target:   ".",
		Risk:     "high",
		Decision: "granted",
		Message:  policy.Message,
	})
	return policy, err
}

func (s *Service) RevokeFullProjectAccess(root string) (Policy, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return Policy{}, err
	}
	now := time.Now().UTC()
	policy := Policy{
		WorkspaceRoot:     absRoot,
		FullProjectAccess: false,
		GrantedAt:         now,
		ExpiresAt:         now,
		Message:           "Full project access revoked.",
	}
	if err := writeJSON(policyPath(absRoot), policy); err != nil {
		return Policy{}, err
	}
	_, err = s.Append(absRoot, Record{
		Action:   "access.full_project.revoke",
		Target:   ".",
		Risk:     "high",
		Decision: "revoked",
		Message:  policy.Message,
	})
	return policy, err
}

func (s *Service) HasFullProjectAccess(root string) bool {
	policy, err := s.LoadPolicy(root)
	if err != nil {
		return false
	}
	return policy.Active(time.Now().UTC())
}

func cleanRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.New("workspace root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("workspace root must be a directory")
	}
	return absRoot, nil
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func logPath(root string) string {
	return filepath.Join(root, filepath.FromSlash(approvalsDirRelPath), approvalLogName)
}

func policyPath(root string) string {
	return filepath.Join(root, filepath.FromSlash(approvalsDirRelPath), policyName)
}

func fallback(value string, next string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return next
	}
	return value
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.Trim(value, "-")
	if value == "" {
		return "approval"
	}
	return value
}
