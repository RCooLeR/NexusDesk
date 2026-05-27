package artifacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) WriteArtifactComparisonReport(comparison ArtifactComparison) (Artifact, error) {
	if strings.TrimSpace(comparison.LeftPath) == "" || strings.TrimSpace(comparison.RightPath) == "" {
		return Artifact{}, errors.New("comparison report requires left and right artifact paths")
	}
	if strings.TrimSpace(comparison.Diff) == "" {
		return Artifact{}, errors.New("comparison report requires diff content")
	}
	createdAt := time.Now().UTC()
	title := artifactComparisonTitle(comparison)
	relPath := s.relPath("comparisons", fmt.Sprintf("%s-%s.md", createdAt.Format("20060102-150405-000000000"), comparisonFileSlug(comparison)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	content := artifactComparisonMarkdown(comparison, title, createdAt)
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return Artifact{}, err
	}
	sourcePaths := []string{comparison.LeftPath, comparison.RightPath}
	metadata := Metadata{
		Kind:        "artifact-comparison",
		Title:       title,
		RelPath:     relPath,
		Source:      strings.Join(sourcePaths, ", "),
		SourcePaths: sourcePaths,
		GeneratedAt: createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         "artifact-comparison",
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Artifact comparison report created at " + relPath + ".",
		Size:         int64(len(content)),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  sourcePaths,
	}, nil
}

func artifactComparisonMarkdown(comparison ArtifactComparison, title string, createdAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	writeKV(&builder, "Generated", formatArtifactTime(createdAt))
	writeKV(&builder, "Compared kind", comparison.Kind)
	writeKV(&builder, "Left artifact", comparison.LeftPath)
	writeKV(&builder, "Right artifact", comparison.RightPath)
	writeKV(&builder, "Same", fmt.Sprintf("%t", comparison.Same))
	builder.WriteString("\n## Summary\n\n")
	builder.WriteString(strings.TrimSpace(comparison.Message))
	builder.WriteString("\n\n## Diff\n\n```diff\n")
	builder.WriteString(strings.TrimSpace(comparison.Diff))
	builder.WriteString("\n```\n")
	return builder.String()
}

func artifactComparisonTitle(comparison ArtifactComparison) string {
	left := strings.TrimSpace(comparison.LeftTitle)
	if left == "" {
		left = filepath.Base(comparison.LeftPath)
	}
	right := strings.TrimSpace(comparison.RightTitle)
	if right == "" {
		right = filepath.Base(comparison.RightPath)
	}
	return "Artifact Comparison - " + left + " vs " + right
}

func comparisonFileSlug(comparison ArtifactComparison) string {
	left := safeName(filepath.Base(comparison.LeftPath))
	right := safeName(filepath.Base(comparison.RightPath))
	slug := left + "-vs-" + right
	if len(slug) > 80 {
		slug = slug[:80]
	}
	return strings.Trim(slug, "-")
}
