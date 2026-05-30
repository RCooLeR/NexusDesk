package artifacts

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Store) SnapshotArtifact(relPath string, action string) (RollbackSnapshot, error) {
	artifact, err := s.artifactByPath(relPath)
	if err != nil {
		return RollbackSnapshot{}, err
	}
	action = strings.TrimSpace(action)
	if action == "" {
		action = "artifact-action"
	}
	createdAt := time.Now().UTC()
	id := artifactTimestamp(createdAt) + "-" + safeName(action) + "-" + safeName(filepath.Base(artifact.RelPath))
	dirRel := s.relPath("rollback", id)
	dirAbs := s.absPath(dirRel)
	if err := os.MkdirAll(dirAbs, 0o755); err != nil {
		return RollbackSnapshot{}, err
	}
	artifactSnapshotRel := filepath.ToSlash(filepath.Join(dirRel, filepath.Base(artifact.RelPath)))
	if err := copyFile(artifact.AbsPath, s.absPath(artifactSnapshotRel), 0o600); err != nil {
		return RollbackSnapshot{}, err
	}
	metadataSnapshotRel := ""
	if artifact.MetadataPath != "" {
		sourceMeta := s.absPath(artifact.MetadataPath)
		if _, err := os.Stat(sourceMeta); err == nil {
			metadataSnapshotRel = artifactSnapshotRel + ".json"
			if err := copyFile(sourceMeta, s.absPath(metadataSnapshotRel), 0o600); err != nil {
				return RollbackSnapshot{}, err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return RollbackSnapshot{}, err
		}
	}
	snapshot := RollbackSnapshot{
		ID:                  id,
		Action:              action,
		OriginalRelPath:     artifact.RelPath,
		ArtifactSnapshotRel: artifactSnapshotRel,
		MetadataSnapshotRel: metadataSnapshotRel,
		ManifestRelPath:     filepath.ToSlash(filepath.Join(dirRel, "rollback.json")),
		CreatedAt:           createdAt,
	}
	if err := s.writeRollbackSnapshot(snapshot); err != nil {
		return RollbackSnapshot{}, err
	}
	return snapshot, nil
}

func (s *Store) ListRollbackSnapshots() ([]RollbackSnapshot, error) {
	root := s.absPath(s.relPath("rollback"))
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return []RollbackSnapshot{}, nil
	} else if err != nil {
		return nil, err
	}
	snapshots := []RollbackSnapshot{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "rollback.json" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var snapshot RollbackSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return err
		}
		snapshots = append(snapshots, snapshot)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})
	return snapshots, nil
}

func (s *Store) writeRollbackSnapshot(snapshot RollbackSnapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.absPath(snapshot.ManifestRelPath), append(data, '\n'), 0o600)
}

func copyFile(source string, target string, mode os.FileMode) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	targetFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer targetFile.Close()
	_, err = io.Copy(targetFile, sourceFile)
	return err
}
