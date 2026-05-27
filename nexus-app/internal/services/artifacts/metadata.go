package artifacts

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) writeMetadata(metadata Metadata) error {
	metadata.RelPath = filepath.ToSlash(strings.TrimSpace(metadata.RelPath))
	if metadata.RelPath == "" {
		return errors.New("artifact metadata rel path is required")
	}
	if metadata.GeneratedAt.IsZero() {
		metadata.GeneratedAt = time.Now().UTC()
	}
	metadataPath := s.absPath(metadata.RelPath) + ".json"
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(metadataPath, append(data, '\n'), 0o644)
}

func (s *Store) readMetadata(relPath string) (Metadata, string) {
	metadataRelPath := filepath.ToSlash(relPath + ".json")
	data, err := os.ReadFile(s.absPath(metadataRelPath))
	if err != nil {
		return Metadata{}, ""
	}
	var metadata Metadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return Metadata{}, ""
	}
	return metadata, metadataRelPath
}

func isMetadataSidecar(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".json")
}
