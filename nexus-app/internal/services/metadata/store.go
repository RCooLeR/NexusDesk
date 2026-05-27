package metadata

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const metadataDirRelPath = ".nexusdesk/metadata"

type Store struct {
	root string
	path string
}

func NewStore(root string) (*Store, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return nil, err
	}
	return &Store{
		root: absRoot,
		path: filepath.Join(absRoot, filepath.FromSlash(metadataDirRelPath), "nexusdesk.sqlite"),
	}, nil
}

func (s *Store) Ensure() (Status, error) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return Status{}, err
	}
	schemaPath := filepath.Join(filepath.Dir(s.path), "schema.sql")
	if err := os.WriteFile(schemaPath, []byte(schemaSQL), 0o644); err != nil {
		return Status{}, err
	}
	db, err := sql.Open("sqlite", s.path)
	if err != nil {
		return Status{}, err
	}
	defer db.Close()
	if _, err := db.Exec(schemaSQL); err != nil {
		return Status{}, err
	}
	now := time.Now().UTC()
	if _, err := db.Exec(
		`INSERT INTO workspaces (id, root, name, opened_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(root) DO UPDATE SET name = excluded.name, opened_at = excluded.opened_at`,
		hashID(s.root), s.root, filepath.Base(s.root), formatTime(now),
	); err != nil {
		return Status{}, err
	}
	tables, err := listTables(db)
	if err != nil {
		return Status{}, err
	}
	status := Status{
		Path:          s.path,
		SchemaPath:    schemaPath,
		SchemaVersion: schemaVersion,
		SchemaHash:    schemaHash(),
		Tables:        tables,
		Message:       "SQLite metadata store is active.",
		UpdatedAt:     now,
	}
	if err := s.writeManifest(status); err != nil {
		return Status{}, err
	}
	return status, nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) open() (*sql.DB, error) {
	if _, err := s.Ensure(); err != nil {
		return nil, err
	}
	return sql.Open("sqlite", s.path)
}

func (s *Store) writeManifest(status Status) error {
	payload := struct {
		Path          string   `json:"path"`
		SchemaPath    string   `json:"schemaPath"`
		SchemaVersion int      `json:"schemaVersion"`
		SchemaHash    string   `json:"schemaHash"`
		Tables        []string `json:"tables"`
		Message       string   `json:"message"`
		UpdatedAt     string   `json:"updatedAt"`
	}{
		Path:          status.Path,
		SchemaPath:    status.SchemaPath,
		SchemaVersion: status.SchemaVersion,
		SchemaHash:    status.SchemaHash,
		Tables:        status.Tables,
		Message:       status.Message,
		UpdatedAt:     formatTime(status.UpdatedAt),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(filepath.Dir(s.path), "sqlite-manifest.json"), append(data, '\n'), 0o644)
}

func listTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tables := []string{}
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}

func cleanRoot(root string) (string, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return "", errors.New("metadata root is required")
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
		return "", errors.New("metadata root must be a directory")
	}
	return absRoot, nil
}

func schemaHash() string {
	hash := sha256.Sum256([]byte(schemaSQL))
	return hex.EncodeToString(hash[:])
}

func hashID(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:16])
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}
