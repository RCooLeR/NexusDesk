package artifacts

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const presentationPackageFormat = "nexus-presentation-package-v1"

type presentationPackageManifest struct {
	Kind         string   `json:"kind"`
	Format       string   `json:"format"`
	Title        string   `json:"title"`
	GeneratedAt  string   `json:"generatedAt"`
	GeneratedBy  string   `json:"generatedBy"`
	SourcePath   string   `json:"sourcePath"`
	SourceTitle  string   `json:"sourceTitle,omitempty"`
	SourceKind   string   `json:"sourceKind,omitempty"`
	SourcePaths  []string `json:"sourcePaths,omitempty"`
	SlideCount   int      `json:"slideCount"`
	PackageFiles []string `json:"packageFiles"`
}

func BuildPresentationPackageReport(title string, sourcePath string, sourceTitle string, sourceKind string, outlineText string, sourcePaths []string) PresentationPackageReport {
	sourceTitle = strings.TrimSpace(sourceTitle)
	title = strings.TrimSpace(title)
	if title == "" {
		title = presentationPackageTitle(sourceTitle, sourcePath)
	}
	outline := presentationOutlineSlideSection(outlineText)
	slides := presentationSlidesFromMarkdown(outline)
	return PresentationPackageReport{
		Title:       title,
		SourcePath:  filepath.ToSlash(strings.TrimSpace(sourcePath)),
		SourceTitle: sourceTitle,
		SourceKind:  strings.TrimSpace(sourceKind),
		SourcePaths: append([]string{}, sourcePaths...),
		Outline:     outline,
		SlideCount:  len(slides),
		GeneratedBy: "Nexus native presentation package writer",
	}
}

func (s *Store) WritePresentationPackageReport(report PresentationPackageReport) (Artifact, error) {
	outline := strings.TrimSpace(report.Outline)
	if outline == "" {
		return Artifact{}, errors.New("presentation package outline is required")
	}
	createdAt := time.Now().UTC()
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = presentationPackageTitle(report.SourceTitle, report.SourcePath)
	}
	relPath := s.relPath("presentation-packages", fmt.Sprintf("%s-%s.zip", artifactTimestamp(createdAt), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	sourcePaths := presentationOutlineSourcePaths(report.SourcePath, report.SourcePaths)
	slides := presentationSlidesFromMarkdown(outline)
	files := []string{"manifest.json", "outline.md", "slides.json", "slides.md", "README.md"}
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	zipWriter := zip.NewWriter(file)
	writeErr := writePresentationPackageZip(zipWriter, report, title, outline, sourcePaths, slides, files, createdAt)
	closeZipErr := zipWriter.Close()
	closeFileErr := file.Close()
	if writeErr != nil {
		_ = os.Remove(absPath)
		return Artifact{}, writeErr
	}
	if closeZipErr != nil {
		_ = os.Remove(absPath)
		return Artifact{}, closeZipErr
	}
	if closeFileErr != nil {
		_ = os.Remove(absPath)
		return Artifact{}, closeFileErr
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return Artifact{}, err
	}
	metadata := Metadata{
		Kind:         "presentation-package",
		Title:        title,
		RelPath:      relPath,
		Source:       filepath.ToSlash(strings.TrimSpace(report.SourcePath)),
		SourcePaths:  sourcePaths,
		GeneratedAt:  createdAt,
		ExportFormat: "zip",
		PackageFiles: append([]string{}, files...),
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         "presentation-package",
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Presentation package artifact created at " + relPath + ".",
		Size:         info.Size(),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  append([]string{}, sourcePaths...),
	}, nil
}

func writePresentationPackageZip(zipWriter *zip.Writer, report PresentationPackageReport, title string, outline string, sourcePaths []string, slides []presentationSlide, files []string, createdAt time.Time) error {
	manifest := presentationPackageManifest{
		Kind:         "presentation-package",
		Format:       presentationPackageFormat,
		Title:        title,
		GeneratedAt:  createdAt.Format(time.RFC3339),
		GeneratedBy:  firstNonEmptyArtifact(report.GeneratedBy, "Nexus native presentation package writer"),
		SourcePath:   filepath.ToSlash(strings.TrimSpace(report.SourcePath)),
		SourceTitle:  strings.TrimSpace(report.SourceTitle),
		SourceKind:   strings.TrimSpace(report.SourceKind),
		SourcePaths:  append([]string{}, sourcePaths...),
		SlideCount:   firstNonZeroInt(report.SlideCount, len(slides)),
		PackageFiles: append([]string{}, files...),
	}
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := addZipText(zipWriter, "manifest.json", string(append(manifestJSON, '\n'))); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "outline.md", strings.TrimSpace(outline)+"\n"); err != nil {
		return err
	}
	slidesJSON, err := json.MarshalIndent(slides, "", "  ")
	if err != nil {
		return err
	}
	if err := addZipText(zipWriter, "slides.json", string(append(slidesJSON, '\n'))); err != nil {
		return err
	}
	if err := addZipText(zipWriter, "slides.md", presentationOutlineContent(slides)+"\n"); err != nil {
		return err
	}
	return addZipText(zipWriter, "README.md", presentationPackageReadme(manifest))
}

func addZipText(zipWriter *zip.Writer, name string, content string) error {
	writer, err := zipWriter.Create(name)
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(content))
	return err
}

func presentationPackageReadme(manifest presentationPackageManifest) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(manifest.Title)
	builder.WriteString("\n\n")
	builder.WriteString("This NexusDesk presentation package contains a normalized outline, slide JSON, slide Markdown, and source lineage manifest.\n\n")
	builder.WriteString("- Format: ")
	builder.WriteString(manifest.Format)
	builder.WriteString("\n")
	builder.WriteString("- Source artifact: ")
	builder.WriteString(manifest.SourcePath)
	builder.WriteString("\n")
	builder.WriteString("- Slides: ")
	builder.WriteString(fmt.Sprintf("%d", manifest.SlideCount))
	builder.WriteString("\n\n")
	builder.WriteString("Use `slides.md` for human review, `slides.json` for future deck generation, and `manifest.json` for provenance.\n")
	return builder.String()
}

func presentationPackageTitle(sourceTitle string, sourcePath string) string {
	sourceTitle = strings.TrimSpace(sourceTitle)
	sourceTitle = strings.TrimPrefix(sourceTitle, "Presentation Outline - ")
	if sourceTitle != "" {
		return "Presentation Package - " + sourceTitle
	}
	sourcePath = filepath.ToSlash(strings.TrimSpace(sourcePath))
	if sourcePath != "" {
		name := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
		if strings.TrimSpace(name) != "" {
			return "Presentation Package - " + name
		}
	}
	return "Presentation Package"
}

func presentationOutlineSlideSection(markdown string) string {
	markdown = strings.TrimSpace(strings.ReplaceAll(markdown, "\r\n", "\n"))
	if markdown == "" {
		return ""
	}
	marker := "\n## Slide Outline"
	index := strings.Index(markdown, marker)
	if index == -1 {
		return markdown
	}
	section := strings.TrimSpace(markdown[index+len(marker):])
	if section == "" {
		return markdown
	}
	return section
}
