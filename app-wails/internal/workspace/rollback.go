package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const rollbackDirRelPath = ".nexusdesk/rollbacks"
const rollbackLogName = "log.json"
const rollbackMaxRecords = 120
const rollbackMaxSnapshotBytes = 32 * 1024 * 1024

type RollbackEntry struct {
	RelPath       string `json:"relPath"`
	Existed       bool   `json:"existed"`
	BackupRelPath string `json:"backupRelPath"`
	Mode          uint32 `json:"mode"`
	Size          int64  `json:"size"`
	SHA256        string `json:"sha256"`
}

type RollbackRecord struct {
	ID        string          `json:"id"`
	Action    string          `json:"action"`
	Target    string          `json:"target"`
	Status    string          `json:"status"`
	Message   string          `json:"message"`
	Entries   []RollbackEntry `json:"entries"`
	CreatedAt string          `json:"createdAt"`
	AppliedAt string          `json:"appliedAt"`
}

type RollbackApplyResult struct {
	ID        string   `json:"id"`
	Restored  []string `json:"restored"`
	Removed   []string `json:"removed"`
	Message   string   `json:"message"`
	AppliedAt string   `json:"appliedAt"`
}

func PrepareRollback(root string, action string, target string, relPaths []string) (RollbackRecord, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return RollbackRecord{}, err
	}
	if strings.TrimSpace(absRoot) == "" {
		return RollbackRecord{}, errors.New("workspace root is required")
	}
	id := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + slugRollback(action)
	record := RollbackRecord{
		ID:        id,
		Action:    strings.TrimSpace(action),
		Target:    strings.TrimSpace(target),
		Status:    "active",
		Entries:   []RollbackEntry{},
		CreatedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	seen := map[string]bool{}
	for _, relPath := range relPaths {
		cleanRel, err := cleanRollbackRelPath(relPath)
		if err != nil {
			return RollbackRecord{}, err
		}
		if seen[cleanRel] {
			continue
		}
		seen[cleanRel] = true
		entry, err := snapshotRollbackPath(absRoot, id, cleanRel)
		if err != nil {
			return RollbackRecord{}, err
		}
		record.Entries = append(record.Entries, entry)
	}
	if len(record.Entries) == 0 {
		return RollbackRecord{}, errors.New("rollback snapshot requires at least one workspace path")
	}
	record.Message = fmt.Sprintf("Rollback snapshot %s prepared for %d path(s).", record.ID, len(record.Entries))
	return record, nil
}

func CommitRollback(root string, record RollbackRecord) (RollbackRecord, error) {
	record.Status = fallbackRollback(record.Status, "active")
	record.Message = fallbackRollback(record.Message, fmt.Sprintf("Rollback snapshot %s is available.", record.ID))
	items, err := ListRollbacks(root)
	if err != nil {
		return RollbackRecord{}, err
	}
	items = append([]RollbackRecord{record}, items...)
	if len(items) > rollbackMaxRecords {
		items = items[:rollbackMaxRecords]
	}
	if err := writeRollbackLog(root, items); err != nil {
		return RollbackRecord{}, err
	}
	return record, nil
}

func ListRollbacks(root string) ([]RollbackRecord, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(absRoot, filepath.FromSlash(rollbackDirRelPath), rollbackLogName)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []RollbackRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []RollbackRecord{}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func ApplyRollback(root string, id string) (RollbackApplyResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RollbackApplyResult{}, errors.New("rollback id is required")
	}
	items, err := ListRollbacks(root)
	if err != nil {
		return RollbackApplyResult{}, err
	}
	index := -1
	for candidateIndex, item := range items {
		if item.ID == id {
			index = candidateIndex
			break
		}
	}
	if index < 0 {
		return RollbackApplyResult{}, errors.New("rollback record was not found")
	}
	record := items[index]
	if record.Status == "applied" {
		return RollbackApplyResult{}, errors.New("rollback record was already applied")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return RollbackApplyResult{}, err
	}

	result := RollbackApplyResult{ID: record.ID, Restored: []string{}, Removed: []string{}, AppliedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	for entryIndex := len(record.Entries) - 1; entryIndex >= 0; entryIndex-- {
		entry := record.Entries[entryIndex]
		relPath, err := cleanRollbackRelPath(entry.RelPath)
		if err != nil {
			return RollbackApplyResult{}, err
		}
		absTarget, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(relPath)))
		if err != nil {
			return RollbackApplyResult{}, err
		}
		if err := ensureInsideRoot(absRoot, absTarget); err != nil {
			return RollbackApplyResult{}, err
		}
		if info, err := os.Lstat(absTarget); err == nil {
			if info.IsDir() {
				return RollbackApplyResult{}, fmt.Errorf("rollback target %s is now a directory", relPath)
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return RollbackApplyResult{}, fmt.Errorf("rollback target %s is now a symlink", relPath)
			}
		} else if !os.IsNotExist(err) {
			return RollbackApplyResult{}, err
		}

		if !entry.Existed {
			if err := os.Remove(absTarget); err != nil && !os.IsNotExist(err) {
				return RollbackApplyResult{}, err
			}
			result.Removed = append(result.Removed, relPath)
			continue
		}
		backupPath := filepath.Join(absRoot, filepath.FromSlash(entry.BackupRelPath))
		content, err := os.ReadFile(backupPath)
		if err != nil {
			return RollbackApplyResult{}, err
		}
		if entry.SHA256 != "" {
			sum := sha256.Sum256(content)
			if hex.EncodeToString(sum[:]) != entry.SHA256 {
				return RollbackApplyResult{}, fmt.Errorf("rollback backup checksum mismatch for %s", relPath)
			}
		}
		if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
			return RollbackApplyResult{}, err
		}
		mode := os.FileMode(entry.Mode)
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(absTarget, content, mode.Perm()); err != nil {
			return RollbackApplyResult{}, err
		}
		result.Restored = append(result.Restored, relPath)
	}
	record.Status = "applied"
	record.AppliedAt = result.AppliedAt
	record.Message = fmt.Sprintf("Rollback %s applied: %d restored, %d removed.", record.ID, len(result.Restored), len(result.Removed))
	items[index] = record
	if err := writeRollbackLog(root, items); err != nil {
		return RollbackApplyResult{}, err
	}
	result.Message = record.Message
	return result, nil
}

func snapshotRollbackPath(absRoot string, id string, relPath string) (RollbackEntry, error) {
	absTarget, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(relPath)))
	if err != nil {
		return RollbackEntry{}, err
	}
	if err := ensureInsideRoot(absRoot, absTarget); err != nil {
		return RollbackEntry{}, err
	}
	entry := RollbackEntry{RelPath: relPath}
	info, err := os.Lstat(absTarget)
	if os.IsNotExist(err) {
		return entry, nil
	}
	if err != nil {
		return RollbackEntry{}, err
	}
	if info.IsDir() {
		return RollbackEntry{}, fmt.Errorf("rollback path %s is a directory", relPath)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return RollbackEntry{}, fmt.Errorf("rollback path %s is a symlink", relPath)
	}
	if info.Size() > rollbackMaxSnapshotBytes {
		return RollbackEntry{}, fmt.Errorf("rollback snapshot for %s is too large", relPath)
	}
	content, err := os.ReadFile(absTarget)
	if err != nil {
		return RollbackEntry{}, err
	}
	sum := sha256.Sum256(content)
	backupRelPath := filepath.ToSlash(filepath.Join(rollbackDirRelPath, id, encodedRollbackPath(relPath)+".bin"))
	backupAbsPath := filepath.Join(absRoot, filepath.FromSlash(backupRelPath))
	if err := os.MkdirAll(filepath.Dir(backupAbsPath), 0o755); err != nil {
		return RollbackEntry{}, err
	}
	if err := os.WriteFile(backupAbsPath, content, info.Mode().Perm()); err != nil {
		return RollbackEntry{}, err
	}
	entry.Existed = true
	entry.BackupRelPath = backupRelPath
	entry.Mode = uint32(info.Mode().Perm())
	entry.Size = info.Size()
	entry.SHA256 = hex.EncodeToString(sum[:])
	return entry, nil
}

func writeRollbackLog(root string, items []RollbackRecord) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	dir := filepath.Join(absRoot, filepath.FromSlash(rollbackDirRelPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, rollbackLogName), append(data, '\n'), 0o644)
}

func cleanRollbackRelPath(relPath string) (string, error) {
	cleanRel, err := cleanPreviewRelPath(relPath)
	if err != nil {
		return "", err
	}
	cleanRel = filepath.ToSlash(cleanRel)
	if cleanRel == "." || cleanRel == "" {
		return "", errors.New("rollback path must name a file")
	}
	if strings.HasPrefix(cleanRel, ".nexusdesk/") {
		return "", errors.New("rollback snapshots cannot target Nexus metadata")
	}
	return cleanRel, nil
}

func encodedRollbackPath(relPath string) string {
	sum := sha256.Sum256([]byte(filepath.ToSlash(relPath)))
	return hex.EncodeToString(sum[:])
}

func slugRollback(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(".", "-", "_", "-", " ", "-")
	value = replacer.Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "rollback"
	}
	return value
}

func fallbackRollback(value string, next string) string {
	if strings.TrimSpace(value) == "" {
		return next
	}
	return strings.TrimSpace(value)
}
