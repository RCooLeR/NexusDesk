package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) ArchiveArtifact(relPath string) (Artifact, error) {
	artifact, err := s.artifactByPath(relPath)
	if err != nil {
		return Artifact{}, err
	}
	if _, err := s.SnapshotArtifact(artifact.RelPath, "archive"); err != nil {
		return Artifact{}, err
	}
	stamp := artifactTimestamp(time.Now().UTC())
	targetRel := s.relPath("archive", stamp+"-"+filepath.Base(artifact.RelPath))
	targetAbs := s.absPath(targetRel)
	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return Artifact{}, err
	}
	if err := os.Rename(artifact.AbsPath, targetAbs); err != nil {
		return Artifact{}, err
	}
	if artifact.MetadataPath != "" {
		targetMeta := targetAbs + ".json"
		_ = os.Rename(s.absPath(artifact.MetadataPath), targetMeta)
	}
	return s.artifactByPath(targetRel)
}

func (s *Store) RestoreArtifact(relPath string) (Artifact, error) {
	artifact, err := s.artifactByPath(relPath)
	if err != nil {
		return Artifact{}, err
	}
	if !artifact.Archived {
		return Artifact{}, errors.New("artifact is not archived")
	}
	if _, err := s.SnapshotArtifact(artifact.RelPath, "restore"); err != nil {
		return Artifact{}, err
	}
	targetRel := s.restoreTargetRelPath(artifact)
	targetAbs := s.absPath(targetRel)
	if err := os.MkdirAll(filepath.Dir(targetAbs), 0o755); err != nil {
		return Artifact{}, err
	}
	if err := os.Rename(artifact.AbsPath, targetAbs); err != nil {
		return Artifact{}, err
	}
	if artifact.MetadataPath != "" {
		targetMeta := targetAbs + ".json"
		_ = os.Rename(s.absPath(artifact.MetadataPath), targetMeta)
	}
	return s.artifactByPath(targetRel)
}

func (s *Store) DeleteArtifact(relPath string) error {
	artifact, err := s.artifactByPath(relPath)
	if err != nil {
		return err
	}
	if _, err := s.SnapshotArtifact(artifact.RelPath, "delete"); err != nil {
		return err
	}
	if err := os.Remove(artifact.AbsPath); err != nil {
		return err
	}
	if artifact.MetadataPath != "" {
		_ = os.Remove(s.absPath(artifact.MetadataPath))
	}
	return nil
}

func (s *Store) restoreTargetRelPath(artifact Artifact) string {
	metadata, _ := s.readMetadata(artifact.RelPath)
	targetRel := cleanRestoreRelPath(metadata.RelPath)
	if targetRel == "" {
		targetRel = s.relPath("restored", archiveBaseName(artifact.RelPath))
	}
	if _, err := os.Stat(s.absPath(targetRel)); os.IsNotExist(err) {
		return targetRel
	}
	stamp := artifactTimestamp(time.Now().UTC())
	dir := filepath.ToSlash(filepath.Dir(targetRel))
	name := filepath.Base(targetRel)
	return filepath.ToSlash(filepath.Join(dir, stamp+"-restored-"+name))
}

func cleanRestoreRelPath(relPath string) string {
	relPath = filepath.ToSlash(strings.TrimSpace(relPath))
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" || relPath == "." || relPath == ".." ||
		strings.HasPrefix(relPath, "../") ||
		strings.Contains(relPath, "/../") ||
		!strings.HasPrefix(relPath, artifactsDirRelPath+"/") ||
		strings.HasPrefix(relPath, artifactsDirRelPath+"/archive/") ||
		isMetadataSidecar(relPath) {
		return ""
	}
	return relPath
}

func archiveBaseName(relPath string) string {
	name := filepath.Base(filepath.ToSlash(relPath))
	parts := strings.SplitN(name, "-", 4)
	if len(parts) == 4 && len(parts[0]) == 8 && len(parts[1]) == 6 && len(parts[2]) == 9 {
		return parts[3]
	}
	return name
}
