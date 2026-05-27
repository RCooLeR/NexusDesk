package artifacts

import (
	"os"
	"path/filepath"
	"time"
)

func (s *Store) ArchiveArtifact(relPath string) (Artifact, error) {
	artifact, err := s.artifactByPath(relPath)
	if err != nil {
		return Artifact{}, err
	}
	stamp := time.Now().UTC().Format("20060102-150405-000000000")
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

func (s *Store) DeleteArtifact(relPath string) error {
	artifact, err := s.artifactByPath(relPath)
	if err != nil {
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
