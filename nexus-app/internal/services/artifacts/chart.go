package artifacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) WriteChartArtifact(report ChartArtifactReport) (Artifact, error) {
	svg := strings.TrimSpace(report.SVG)
	if svg == "" {
		return Artifact{}, errors.New("chart SVG content is required")
	}
	if strings.TrimSpace(report.SourcePath) == "" {
		return Artifact{}, errors.New("chart source path is required")
	}
	createdAt := time.Now().UTC()
	artifactKind := "chart"
	artifactFolder := "charts"
	if strings.EqualFold(strings.TrimSpace(report.Mode), "dashboard") {
		artifactKind = "dashboard"
		artifactFolder = "dashboards"
	}
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = chartArtifactTitle(report)
	}
	relPath := s.relPath(artifactFolder, fmt.Sprintf("%s-%s.svg", createdAt.Format("20060102-150405-000000000"), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.WriteString(svg + "\n"); err != nil {
		return Artifact{}, err
	}
	metadata := Metadata{
		Kind:        artifactKind,
		Title:       title,
		RelPath:     relPath,
		Source:      chartSourceSummary(report),
		SourcePaths: []string{report.SourcePath},
		GeneratedAt: createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         artifactKind,
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      artifactMessage(artifactKind, relPath),
		Size:         int64(len(svg) + 1),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  []string{report.SourcePath},
	}, nil
}

func chartArtifactTitle(report ChartArtifactReport) string {
	if strings.EqualFold(strings.TrimSpace(report.Mode), "dashboard") {
		if report.ValueColumn != "" {
			return fmt.Sprintf("Dashboard - %s by %s", report.ValueColumn, report.CategoryColumn)
		}
		return fmt.Sprintf("Dashboard - rows by %s", report.CategoryColumn)
	}
	if report.Mode == "line" && report.ValueColumn != "" {
		return fmt.Sprintf("Chart - %s over %s", report.ValueColumn, report.CategoryColumn)
	}
	if report.Mode == "sum" && report.ValueColumn != "" {
		return fmt.Sprintf("Chart - %s by %s", report.ValueColumn, report.CategoryColumn)
	}
	return fmt.Sprintf("Chart - rows by %s", report.CategoryColumn)
}

func chartSourceSummary(report ChartArtifactReport) string {
	parts := []string{report.SourcePath}
	if strings.TrimSpace(report.Query) != "" {
		parts = append(parts, "query: "+strings.TrimSpace(report.Query))
	}
	if report.Format != "" {
		parts = append(parts, "format: "+report.Format)
	}
	if strings.TrimSpace(report.Mode) != "" {
		parts = append(parts, "mode: "+strings.TrimSpace(report.Mode))
	}
	if report.PointCount > 0 {
		parts = append(parts, fmt.Sprintf("points: %d", report.PointCount))
	}
	if report.Truncated {
		parts = append(parts, "bounded sample")
	}
	return strings.Join(parts, " | ")
}

func artifactMessage(kind string, relPath string) string {
	if kind == "dashboard" {
		return "Dashboard artifact created at " + relPath + "."
	}
	return "Chart artifact created at " + relPath + "."
}
