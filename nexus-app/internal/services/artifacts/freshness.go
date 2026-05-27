package artifacts

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (s *Store) SourceFreshness(relPath string) (SourceFreshness, error) {
	artifact, err := s.artifactByPath(relPath)
	if err != nil {
		return SourceFreshness{}, err
	}
	generatedAt := firstTime(artifact.GeneratedAt, artifact.CreatedAt)
	report := SourceFreshness{
		ArtifactRelPath: artifact.RelPath,
		GeneratedAt:     generatedAt,
	}
	for _, source := range artifactFreshnessSources(artifact) {
		status := s.sourceFreshnessStatus(source, generatedAt)
		if status.Changed {
			report.ChangedCount++
		}
		if status.Unknown {
			report.UnknownCount++
		} else if !status.Exists {
			report.MissingCount++
		}
		report.Sources = append(report.Sources, status)
	}
	report.Stale = report.ChangedCount > 0 || report.MissingCount > 0
	report.Message = sourceFreshnessMessage(report)
	return report, nil
}

func (s *Store) sourceFreshnessStatus(source string, generatedAt time.Time) SourceFreshnessStatus {
	status := SourceFreshnessStatus{RelPath: filepath.ToSlash(strings.TrimSpace(source))}
	absPath, err := s.resolveWorkspaceSourcePath(status.RelPath)
	if err != nil {
		status.Unknown = true
		status.Message = err.Error()
		return status
	}
	info, err := os.Stat(absPath)
	if errors.Is(err, os.ErrNotExist) {
		status.Message = "Source is missing."
		return status
	}
	if err != nil {
		status.Unknown = true
		status.Message = err.Error()
		return status
	}
	status.Exists = true
	status.ModifiedAt = info.ModTime().UTC()
	if sourceChangedAfterGenerated(status.ModifiedAt, generatedAt) {
		status.Changed = true
		status.Message = "Source changed after artifact generation."
		return status
	}
	status.Message = "Source is current."
	return status
}

func (s *Store) resolveWorkspaceSourcePath(relPath string) (string, error) {
	relPath = strings.TrimSpace(relPath)
	relPath = strings.Trim(relPath, `"'`)
	relPath = filepath.ToSlash(relPath)
	relPath = strings.TrimPrefix(relPath, "/")
	if relPath == "" || relPath == "." {
		return "", errors.New("source path is empty")
	}
	if relPath == ".." || strings.HasPrefix(relPath, "../") || strings.Contains(relPath, "/../") || filepath.IsAbs(relPath) {
		return "", errors.New("source path must stay inside the workspace")
	}
	target := filepath.Join(s.root, filepath.FromSlash(relPath))
	relToRoot, err := filepath.Rel(s.root, target)
	if err != nil {
		return "", err
	}
	if relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", errors.New("source path must stay inside the workspace")
	}
	return target, nil
}

func sourceChangedAfterGenerated(modifiedAt time.Time, generatedAt time.Time) bool {
	return !generatedAt.IsZero() && modifiedAt.After(generatedAt)
}

func artifactFreshnessSources(artifact Artifact) []string {
	seen := map[string]bool{}
	sources := []string{}
	candidates := append([]string{}, artifact.SourcePaths...)
	if len(candidates) == 0 {
		candidates = append(candidates, artifact.Source)
	}
	for _, source := range candidates {
		source = filepath.ToSlash(strings.TrimSpace(source))
		if source == "" || seen[source] {
			continue
		}
		seen[source] = true
		sources = append(sources, source)
	}
	return sources
}

func sourceFreshnessMessage(report SourceFreshness) string {
	if len(report.Sources) == 0 {
		return "No source paths recorded for this artifact."
	}
	if report.MissingCount > 0 || report.ChangedCount > 0 {
		parts := []string{}
		if report.ChangedCount > 0 {
			parts = append(parts, pluralize(report.ChangedCount, "changed source"))
		}
		if report.MissingCount > 0 {
			parts = append(parts, pluralize(report.MissingCount, "missing source"))
		}
		return "Artifact may be stale: " + strings.Join(parts, ", ") + "."
	}
	if report.UnknownCount > 0 {
		return "Some source paths could not be checked."
	}
	return "All recorded sources are current."
}

func pluralize(count int, label string) string {
	if count == 1 {
		return "1 " + label
	}
	return strconv.Itoa(count) + " " + label + "s"
}
