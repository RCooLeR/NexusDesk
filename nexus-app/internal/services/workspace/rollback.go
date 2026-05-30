package workspace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	rollbackDirRelPath      = ".nexusdesk/rollbacks"
	rollbackLogName         = "log.json"
	rollbackMaxRecords      = 120
	rollbackMaxSnapshotSize = 32 * 1024 * 1024
)

type RollbackEntry struct {
	RelPath       string
	Existed       bool
	BackupRelPath string
	Mode          uint32
	Size          int64
	SHA256        string
}

type RollbackRecord struct {
	ID        string
	Action    string
	Target    string
	Status    string
	Message   string
	Entries   []RollbackEntry
	CreatedAt time.Time
	AppliedAt *time.Time
}

type RollbackApplyResult struct {
	ID        string
	Restored  []string
	Removed   []string
	Message   string
	AppliedAt time.Time
}

type RollbackStorageUsage struct {
	Records        int
	ActiveRecords  int
	AppliedRecords int
	Entries        int
	SnapshotBytes  int64
	StoredBytes    int64
}

func (s *Service) ListRollbacks(root string) ([]RollbackRecord, error) {
	absRoot, err := cleanRoot(root)
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
	records := []RollbackRecord{}
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func (s *Service) RollbackStorageUsage(root string) (RollbackStorageUsage, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return RollbackStorageUsage{}, err
	}
	records, err := s.ListRollbacks(root)
	if err != nil {
		return RollbackStorageUsage{}, err
	}
	usage := RollbackStorageUsage{Records: len(records)}
	for _, record := range records {
		switch record.Status {
		case "applied":
			usage.AppliedRecords++
		default:
			usage.ActiveRecords++
		}
		usage.Entries += len(record.Entries)
		for _, entry := range record.Entries {
			usage.SnapshotBytes += entry.Size
		}
	}
	rollbackDir := filepath.Join(absRoot, filepath.FromSlash(rollbackDirRelPath))
	if err := filepath.WalkDir(rollbackDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		usage.StoredBytes += info.Size()
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return RollbackStorageUsage{}, err
	}
	return usage, nil
}

func (s *Service) ApplyRollback(root string, id string) (RollbackApplyResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RollbackApplyResult{}, errors.New("rollback id is required")
	}
	records, err := s.ListRollbacks(root)
	if err != nil {
		return RollbackApplyResult{}, err
	}
	index := -1
	for candidateIndex, record := range records {
		if record.ID == id {
			index = candidateIndex
			break
		}
	}
	if index < 0 {
		return RollbackApplyResult{}, errors.New("rollback record was not found")
	}
	record := records[index]
	if record.Status == "applied" {
		return RollbackApplyResult{}, errors.New("rollback record was already applied")
	}
	absRoot, err := cleanRoot(root)
	if err != nil {
		return RollbackApplyResult{}, err
	}

	appliedAt := time.Now().UTC()
	result := RollbackApplyResult{ID: record.ID, Restored: []string{}, Removed: []string{}, AppliedAt: appliedAt}
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
		if !isInside(absRoot, absTarget) {
			return RollbackApplyResult{}, errors.New("rollback path must stay inside the root")
		}
		if info, err := os.Lstat(absTarget); err == nil {
			if !entry.Existed {
				if info.Mode()&os.ModeSymlink != 0 {
					return RollbackApplyResult{}, fmt.Errorf("rollback target %s is now a symlink", relPath)
				}
				if info.IsDir() {
					if err := os.Remove(absTarget); err != nil && !os.IsNotExist(err) {
						return RollbackApplyResult{}, err
					}
					result.Removed = append(result.Removed, relPath)
					continue
				}
				if err := os.Remove(absTarget); err != nil && !os.IsNotExist(err) {
					return RollbackApplyResult{}, err
				}
				result.Removed = append(result.Removed, relPath)
				continue
			}
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
			result.Removed = append(result.Removed, relPath)
			continue
		}
		content, err := os.ReadFile(filepath.Join(absRoot, filepath.FromSlash(entry.BackupRelPath)))
		if err != nil {
			return RollbackApplyResult{}, err
		}
		if err := verifyRollbackChecksum(entry, content); err != nil {
			return RollbackApplyResult{}, err
		}
		mode := os.FileMode(entry.Mode)
		if mode == 0 {
			mode = 0o644
		}
		if err := os.MkdirAll(filepath.Dir(absTarget), 0o755); err != nil {
			return RollbackApplyResult{}, err
		}
		if err := os.WriteFile(absTarget, content, mode.Perm()); err != nil {
			return RollbackApplyResult{}, err
		}
		result.Restored = append(result.Restored, relPath)
	}

	record.Status = "applied"
	record.AppliedAt = &appliedAt
	record.Message = fmt.Sprintf("Rollback %s applied: %d restored, %d removed.", record.ID, len(result.Restored), len(result.Removed))
	records[index] = record
	if err := writeRollbackLog(root, records); err != nil {
		return RollbackApplyResult{}, err
	}
	result.Message = record.Message
	return result, nil
}

func (s *Service) prepareRollback(root string, action string, target string, relPaths []string) (RollbackRecord, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return RollbackRecord{}, err
	}
	id := time.Now().UTC().Format("20060102T150405.000000000Z") + "-" + slugRollback(action)
	record := RollbackRecord{
		ID:        id,
		Action:    strings.TrimSpace(action),
		Target:    strings.TrimSpace(target),
		Status:    "active",
		Entries:   []RollbackEntry{},
		CreatedAt: time.Now().UTC(),
	}
	seen := map[string]bool{}
	for _, relPath := range relPaths {
		cleanRelPath, err := cleanRollbackRelPath(relPath)
		if err != nil {
			return RollbackRecord{}, err
		}
		if seen[cleanRelPath] {
			continue
		}
		seen[cleanRelPath] = true
		entry, err := snapshotRollbackPath(absRoot, id, cleanRelPath)
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

func (s *Service) commitRollback(root string, record RollbackRecord) (RollbackRecord, error) {
	if record.Status == "" {
		record.Status = "active"
	}
	if record.Message == "" {
		record.Message = fmt.Sprintf("Rollback snapshot %s is available.", record.ID)
	}
	records, err := s.ListRollbacks(root)
	if err != nil {
		return RollbackRecord{}, err
	}
	records = append([]RollbackRecord{record}, records...)
	if len(records) > rollbackMaxRecords {
		records = records[:rollbackMaxRecords]
	}
	if err := writeRollbackLog(root, records); err != nil {
		return RollbackRecord{}, err
	}
	return record, nil
}

func (s *Service) discardPreparedRollback(root string, record RollbackRecord) error {
	if strings.TrimSpace(record.ID) == "" {
		return nil
	}
	absRoot, err := cleanRoot(root)
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(absRoot, filepath.FromSlash(rollbackDirRelPath), record.ID))
}

func snapshotRollbackPath(absRoot string, id string, relPath string) (RollbackEntry, error) {
	absTarget, err := filepath.Abs(filepath.Join(absRoot, filepath.FromSlash(relPath)))
	if err != nil {
		return RollbackEntry{}, err
	}
	if !isInside(absRoot, absTarget) {
		return RollbackEntry{}, errors.New("rollback path must stay inside the root")
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
	if info.Size() > rollbackMaxSnapshotSize {
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

func writeRollbackLog(root string, records []RollbackRecord) error {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return err
	}
	dir := filepath.Join(absRoot, filepath.FromSlash(rollbackDirRelPath))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, rollbackLogName), append(data, '\n'), 0o644)
}

func cleanRollbackRelPath(relPath string) (string, error) {
	cleanRelPath, err := cleanRel(relPath)
	if err != nil {
		return "", err
	}
	if cleanRelPath == "" {
		return "", errors.New("rollback path must name a file")
	}
	if isInternalMetadataPath(cleanRelPath) {
		return "", errors.New("rollback snapshots cannot target Nexus metadata")
	}
	return filepath.ToSlash(cleanRelPath), nil
}

func verifyRollbackChecksum(entry RollbackEntry, content []byte) error {
	if entry.SHA256 == "" {
		return nil
	}
	sum := sha256.Sum256(content)
	if hex.EncodeToString(sum[:]) != entry.SHA256 {
		return fmt.Errorf("rollback backup checksum mismatch for %s", entry.RelPath)
	}
	return nil
}

func encodedRollbackPath(relPath string) string {
	sum := sha256.Sum256([]byte(filepath.ToSlash(relPath)))
	return hex.EncodeToString(sum[:])
}

func slugRollback(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.NewReplacer(".", "-", "_", "-", " ", "-").Replace(value)
	value = strings.Trim(value, "-")
	if value == "" {
		return "rollback"
	}
	return value
}
