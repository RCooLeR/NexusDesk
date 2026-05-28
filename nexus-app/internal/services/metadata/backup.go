package metadata

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportBackup writes a timestamped ZIP backup of core metadata files.
func (s *Store) ExportBackup() (BackupResult, error) {
	status, err := s.Ensure()
	if err != nil {
		return BackupResult{}, err
	}
	now := time.Now().UTC()
	backupDir := filepath.Join(filepath.Dir(s.path), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return BackupResult{}, err
	}
	backupPath := filepath.Join(backupDir, fmt.Sprintf("metadata-backup-%s.zip", now.Format("20060102-150405")))
	file, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return BackupResult{}, err
	}
	removeOnError := true
	defer func() {
		_ = file.Close()
		if removeOnError {
			_ = os.Remove(backupPath)
		}
	}()
	writer := zip.NewWriter(file)

	candidates := []string{
		s.path,
		status.SchemaPath,
		filepath.Join(filepath.Dir(s.path), "sqlite-manifest.json"),
		s.path + "-wal",
		s.path + "-shm",
	}
	exported := []string{}
	for _, candidate := range candidates {
		baseName := filepath.Base(candidate)
		ok, err := addFileToBackupZip(writer, candidate, baseName)
		if err != nil {
			_ = writer.Close()
			return BackupResult{}, err
		}
		if ok {
			exported = append(exported, baseName)
		}
	}
	ok, err := addBackupSummaryToZip(writer, now, s.root, status.Path, exported)
	if err != nil {
		_ = writer.Close()
		return BackupResult{}, err
	}
	if ok {
		exported = append(exported, "backup-summary.json")
	}
	if err := writer.Close(); err != nil {
		return BackupResult{}, err
	}
	if err := file.Close(); err != nil {
		return BackupResult{}, err
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		return BackupResult{}, err
	}
	removeOnError = false
	return BackupResult{
		Path:      backupPath,
		Files:     exported,
		SizeBytes: info.Size(),
		CreatedAt: now,
	}, nil
}

func addFileToBackupZip(writer *zip.Writer, sourcePath string, zipName string) (bool, error) {
	info, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}
	src, err := os.Open(sourcePath)
	if err != nil {
		return false, err
	}
	defer src.Close()
	entry, err := writer.Create(zipName)
	if err != nil {
		return false, err
	}
	if _, err := io.Copy(entry, src); err != nil {
		return false, err
	}
	return true, nil
}

func addFileToBackupZipAtPath(writer *zip.Writer, sourcePath string, zipPath string) (bool, error) {
	info, err := os.Stat(sourcePath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.IsDir() {
		return false, nil
	}
	src, err := os.Open(sourcePath)
	if err != nil {
		return false, err
	}
	defer src.Close()
	entry, err := writer.Create(zipPath)
	if err != nil {
		return false, err
	}
	if _, err := io.Copy(entry, src); err != nil {
		return false, err
	}
	return true, nil
}

func addBackupSummaryToZip(writer *zip.Writer, createdAt time.Time, workspaceRoot string, storePath string, files []string) (bool, error) {
	entry, err := writer.Create("backup-summary.json")
	if err != nil {
		return false, err
	}
	payload := struct {
		CreatedAt     string   `json:"createdAt"`
		WorkspaceRoot string   `json:"workspaceRoot"`
		StorePath     string   `json:"storePath"`
		Files         []string `json:"files"`
	}{
		CreatedAt:     formatTime(createdAt),
		WorkspaceRoot: workspaceRoot,
		StorePath:     storePath,
		Files:         append([]string{}, files...),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return false, err
	}
	if _, err := entry.Write(append(data, '\n')); err != nil {
		return false, err
	}
	return true, nil
}

// ExportWorkspaceStateBackup writes a timestamped ZIP backup of local workspace
// state rooted in .nexusdesk plus optional app-level config snapshots.
func (s *Store) ExportWorkspaceStateBackup(options WorkspaceStateBackupOptions) (WorkspaceStateBackupResult, error) {
	if _, err := s.Ensure(); err != nil {
		return WorkspaceStateBackupResult{}, err
	}
	now := time.Now().UTC()
	stateRoot := filepath.Join(s.root, ".nexusdesk")
	stateInfo, err := os.Stat(stateRoot)
	if err != nil {
		return WorkspaceStateBackupResult{}, err
	}
	if !stateInfo.IsDir() {
		return WorkspaceStateBackupResult{}, errors.New(".nexusdesk workspace state directory is unavailable")
	}
	backupDir := filepath.Join(stateRoot, "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return WorkspaceStateBackupResult{}, err
	}
	backupPath := filepath.Join(backupDir, fmt.Sprintf("workspace-state-backup-%s.zip", now.Format("20060102-150405")))
	file, err := os.OpenFile(backupPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return WorkspaceStateBackupResult{}, err
	}
	removeOnError := true
	defer func() {
		_ = file.Close()
		if removeOnError {
			_ = os.Remove(backupPath)
		}
	}()
	writer := zip.NewWriter(file)

	exported := []string{}
	backupDirWithSep := strings.ToLower(filepath.Clean(backupDir) + string(os.PathSeparator))
	err = filepath.WalkDir(stateRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		cleanPath := filepath.Clean(path)
		if entry.IsDir() {
			cleanDirWithSep := strings.ToLower(cleanPath + string(os.PathSeparator))
			if strings.HasPrefix(cleanDirWithSep, backupDirWithSep) {
				return filepath.SkipDir
			}
			return nil
		}
		relToRoot, err := filepath.Rel(s.root, cleanPath)
		if err != nil {
			return err
		}
		zipPath := filepath.ToSlash(relToRoot)
		if zipPath == "" {
			return nil
		}
		ok, err := addFileToBackupZipAtPath(writer, cleanPath, zipPath)
		if err != nil {
			return err
		}
		if ok {
			exported = append(exported, zipPath)
		}
		return nil
	})
	if err != nil {
		_ = writer.Close()
		return WorkspaceStateBackupResult{}, err
	}
	configFiles := []struct {
		path   string
		zipDir string
	}{
		{path: strings.TrimSpace(options.SettingsPath), zipDir: "app-config/settings.json"},
		{path: strings.TrimSpace(options.ConnectorProfilesPath), zipDir: "app-config/connector-profiles.json"},
	}
	connectorProfilesPath := strings.TrimSpace(options.ConnectorProfilesPath)
	if connectorProfilesPath != "" {
		configFiles = append(configFiles, struct {
			path   string
			zipDir string
		}{
			path:   connectorProfilesPath + ".secrets",
			zipDir: "app-config/connector-profiles.secrets.json",
		})
	}
	for _, candidate := range configFiles {
		sourcePath := strings.TrimSpace(candidate.path)
		if sourcePath == "" {
			continue
		}
		ok, err := addFileToBackupZipAtPath(writer, sourcePath, candidate.zipDir)
		if err != nil {
			_ = writer.Close()
			return WorkspaceStateBackupResult{}, err
		}
		if ok {
			exported = append(exported, candidate.zipDir)
		}
	}
	if _, err := addWorkspaceStateSummaryToZip(writer, now, s.root, exported); err != nil {
		_ = writer.Close()
		return WorkspaceStateBackupResult{}, err
	}
	exported = append(exported, "workspace-state-summary.json")
	if err := writer.Close(); err != nil {
		return WorkspaceStateBackupResult{}, err
	}
	if err := file.Close(); err != nil {
		return WorkspaceStateBackupResult{}, err
	}
	info, err := os.Stat(backupPath)
	if err != nil {
		return WorkspaceStateBackupResult{}, err
	}
	removeOnError = false
	return WorkspaceStateBackupResult{
		Path:      backupPath,
		Files:     exported,
		SizeBytes: info.Size(),
		CreatedAt: now,
	}, nil
}

func addWorkspaceStateSummaryToZip(writer *zip.Writer, createdAt time.Time, workspaceRoot string, files []string) (bool, error) {
	entry, err := writer.Create("workspace-state-summary.json")
	if err != nil {
		return false, err
	}
	payload := struct {
		CreatedAt     string   `json:"createdAt"`
		WorkspaceRoot string   `json:"workspaceRoot"`
		Files         []string `json:"files"`
	}{
		CreatedAt:     formatTime(createdAt),
		WorkspaceRoot: workspaceRoot,
		Files:         append([]string{}, files...),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return false, err
	}
	if _, err := entry.Write(append(data, '\n')); err != nil {
		return false, err
	}
	return true, nil
}
