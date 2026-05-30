package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const artifactPreviewMaxBytes = 256 * 1024

func (s *Store) ListTaskRunReports() ([]Artifact, error) {
	return s.ListArtifacts(ListOptions{Query: "kind:task-report"})
}

func (s *Store) ListArtifacts(options ListOptions) ([]Artifact, error) {
	artifactsDir := s.absPath(artifactsDirRelPath)
	if _, err := os.Stat(artifactsDir); os.IsNotExist(err) {
		return []Artifact{}, nil
	} else if err != nil {
		return nil, err
	}
	artifacts := []Artifact{}
	err := filepath.WalkDir(artifactsDir, func(absPath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == "rollback" {
				return filepath.SkipDir
			}
			if !options.IncludeArchived && entry.Name() == "archive" {
				return filepath.SkipDir
			}
			return nil
		}
		if isMetadataSidecar(entry.Name()) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		relPath, err := filepath.Rel(s.root, absPath)
		if err != nil {
			return nil
		}
		artifact := s.artifactFromFile(filepath.ToSlash(relPath), absPath, info)
		if !artifactMatches(artifact, options.Query) {
			return nil
		}
		artifacts = append(artifacts, artifact)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.SliceStable(artifacts, func(i, j int) bool {
		left := firstTime(artifacts[i].GeneratedAt, artifacts[i].CreatedAt)
		right := firstTime(artifacts[j].GeneratedAt, artifacts[j].CreatedAt)
		if left.Equal(right) {
			return artifacts[i].RelPath > artifacts[j].RelPath
		}
		return left.After(right)
	})
	return artifacts, nil
}

func (s *Store) ReadArtifactText(relPath string) (string, error) {
	absPath, err := s.resolveArtifactPath(relPath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("artifact path must be a file")
	}
	if info.Size() > artifactPreviewMaxBytes {
		return "", errors.New("artifact is too large for inline preview")
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (s *Store) artifactByPath(relPath string) (Artifact, error) {
	absPath, err := s.resolveArtifactPath(relPath)
	if err != nil {
		return Artifact{}, err
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return Artifact{}, err
	}
	if info.IsDir() {
		return Artifact{}, errors.New("artifact path must be a file")
	}
	return s.artifactFromFile(filepath.ToSlash(relPath), absPath, info), nil
}

func (s *Store) artifactFromFile(relPath string, absPath string, info os.FileInfo) Artifact {
	metadata, metadataRelPath := s.readMetadata(relPath)
	kind := metadata.Kind
	if kind == "" {
		kind = inferKind(relPath)
	}
	title := metadata.Title
	if title == "" {
		title = strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath))
	}
	return Artifact{
		Kind:         kind,
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: metadataRelPath,
		Size:         info.Size(),
		CreatedAt:    info.ModTime().UTC(),
		GeneratedAt:  metadata.GeneratedAt,
		JobID:        metadata.JobID,
		TaskID:       metadata.TaskID,
		Source:       metadata.Source,
		SourcePaths:  append([]string{}, metadata.SourcePaths...),
		Fingerprints: append([]SourceFingerprint{}, metadata.SourceFingerprints...),
		Archived:     strings.HasPrefix(relPath, artifactsDirRelPath+"/archive/"),
	}
}

func (s *Store) resolveArtifactPath(relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	relPath = strings.Trim(relPath, `"'`)
	relPath = filepath.ToSlash(relPath)
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" || relPath == "." || relPath == ".." || strings.HasPrefix(relPath, "../") || strings.Contains(relPath, "/../") {
		return "", errors.New("artifact path must stay inside the artifact root")
	}
	if !strings.HasPrefix(relPath, artifactsDirRelPath+"/") {
		return "", errors.New("artifact path must be under .nexusdesk/artifacts")
	}
	absRoot := s.absPath(artifactsDirRelPath)
	absPath := filepath.Join(s.root, filepath.FromSlash(relPath))
	relToArtifacts, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", err
	}
	if relToArtifacts == ".." || strings.HasPrefix(relToArtifacts, ".."+string(filepath.Separator)) {
		return "", errors.New("artifact path must stay inside the artifact root")
	}
	return absPath, nil
}
