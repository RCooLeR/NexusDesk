package artifacts

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const sourceFingerprintMaxBytes int64 = 16 * 1024 * 1024

func (s *Store) sourceFingerprints(sourcePaths []string) []SourceFingerprint {
	fingerprints := make([]SourceFingerprint, 0, len(sourcePaths))
	seen := map[string]bool{}
	for _, sourcePath := range sourcePaths {
		sourcePath = filepath.ToSlash(strings.TrimSpace(sourcePath))
		if sourcePath == "" || seen[sourcePath] {
			continue
		}
		seen[sourcePath] = true
		fingerprints = append(fingerprints, s.sourceFingerprint(sourcePath))
	}
	return fingerprints
}

func (s *Store) sourceFingerprint(sourcePath string) SourceFingerprint {
	fingerprint := SourceFingerprint{RelPath: filepath.ToSlash(strings.TrimSpace(sourcePath))}
	absPath, err := s.resolveWorkspaceSourcePath(fingerprint.RelPath)
	if err != nil {
		fingerprint.Error = err.Error()
		return fingerprint
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fingerprint.Error = "source is missing"
		} else {
			fingerprint.Error = err.Error()
		}
		return fingerprint
	}
	if info.IsDir() {
		fingerprint.Error = "source path is a directory"
		return fingerprint
	}
	fingerprint.Size = info.Size()
	fingerprint.ModifiedAt = info.ModTime().UTC()
	if info.Size() > sourceFingerprintMaxBytes {
		fingerprint.Error = "source is too large for content fingerprint"
		return fingerprint
	}
	file, err := os.Open(absPath)
	if err != nil {
		fingerprint.Error = err.Error()
		return fingerprint
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		fingerprint.Error = err.Error()
		return fingerprint
	}
	fingerprint.SHA256 = hex.EncodeToString(hash.Sum(nil))
	return fingerprint
}

func metadataSources(metadata Metadata) []string {
	candidates := append([]string{}, metadata.SourcePaths...)
	if len(candidates) == 0 {
		candidates = append(candidates, metadata.Source)
	}
	return candidates
}
