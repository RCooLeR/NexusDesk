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
	reportsDir := s.absPath(s.relPath("task-runs"))
	entries, err := os.ReadDir(reportsDir)
	if os.IsNotExist(err) {
		return []Artifact{}, nil
	}
	if err != nil {
		return nil, err
	}
	reports := []Artifact{}
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		relPath := s.relPath("task-runs", entry.Name())
		reports = append(reports, Artifact{
			Kind:      "task-report",
			Title:     strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())),
			RelPath:   relPath,
			AbsPath:   s.absPath(relPath),
			Size:      info.Size(),
			CreatedAt: info.ModTime().UTC(),
		})
	}
	sort.SliceStable(reports, func(i, j int) bool {
		if reports[i].CreatedAt.Equal(reports[j].CreatedAt) {
			return reports[i].RelPath > reports[j].RelPath
		}
		return reports[i].CreatedAt.After(reports[j].CreatedAt)
	})
	return reports, nil
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
