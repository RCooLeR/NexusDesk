package artifact

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"NexusDesk/internal/workspace"
)

const reportContentLimit = 12 * 1024
const generatedArtifactContentLimit = 64 * 1024
const artifactDirRelPath = ".nexusdesk/artifacts"

type MarkdownReport struct {
	RelPath string `json:"relPath"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Size    int64  `json:"size"`
}

type MarkdownArtifactRequest struct {
	Title          string   `json:"title"`
	Content        string   `json:"content"`
	ContextRelPath string   `json:"contextRelPath"`
	Prompt         string   `json:"prompt"`
	Model          string   `json:"model"`
	Source         string   `json:"source"`
	SourcePaths    []string `json:"sourcePaths"`
}

type ArtifactMetadata struct {
	Kind           string   `json:"kind"`
	Title          string   `json:"title"`
	Source         string   `json:"source"`
	SourcePaths    []string `json:"sourcePaths"`
	ContextRelPath string   `json:"contextRelPath"`
	Prompt         string   `json:"prompt"`
	Model          string   `json:"model"`
	CreatedAt      string   `json:"createdAt"`
}

type WorkspaceArtifact struct {
	RelPath    string `json:"relPath"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
	Source     string `json:"source"`
	Summary    string `json:"summary"`
	Model      string `json:"model"`
}

func CreateMarkdownReport(root string, source workspace.FilePreview, now time.Time) (MarkdownReport, error) {
	if strings.TrimSpace(root) == "" {
		return MarkdownReport{}, errors.New("open a workspace before creating reports")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return MarkdownReport{}, err
	}

	reportDir := filepath.Join(absRoot, filepath.FromSlash(artifactDirRelPath))
	if err := os.MkdirAll(reportDir, 0o755); err != nil {
		return MarkdownReport{}, err
	}

	name := reportFileName(source, now)
	path := filepath.Join(reportDir, name)
	if err := ensureInsideRoot(absRoot, path); err != nil {
		return MarkdownReport{}, err
	}

	content := buildMarkdownReport(source, now)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return MarkdownReport{}, err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return MarkdownReport{}, err
	}

	if err := writeArtifactMetadata(absRoot, path, ArtifactMetadata{
		Kind:           "markdown-report",
		Title:          "Report: " + source.Name,
		Source:         "selected preview",
		SourcePaths:    cleanMetadataPaths([]string{source.RelPath}),
		ContextRelPath: source.RelPath,
		CreatedAt:      now.UTC().Format(time.RFC3339),
	}); err != nil {
		return MarkdownReport{}, err
	}

	info, err := file.Stat()
	if err != nil {
		return MarkdownReport{}, err
	}

	relPath, err := filepath.Rel(absRoot, path)
	if err != nil {
		return MarkdownReport{}, err
	}

	return MarkdownReport{
		RelPath: filepath.ToSlash(relPath),
		Name:    name,
		Path:    path,
		Message: "Markdown report artifact created inside the workspace.",
		Size:    info.Size(),
	}, nil
}

func CreateGeneratedMarkdown(root string, request MarkdownArtifactRequest, now time.Time) (MarkdownReport, error) {
	if strings.TrimSpace(root) == "" {
		return MarkdownReport{}, errors.New("open a workspace before creating artifacts")
	}
	if strings.TrimSpace(request.Content) == "" {
		return MarkdownReport{}, errors.New("assistant response is empty")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return MarkdownReport{}, err
	}

	artifactDir := filepath.Join(absRoot, filepath.FromSlash(artifactDirRelPath))
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return MarkdownReport{}, err
	}

	name := generatedArtifactFileName(request, now)
	path := filepath.Join(artifactDir, name)
	if err := ensureInsideRoot(absRoot, path); err != nil {
		return MarkdownReport{}, err
	}

	content := buildGeneratedMarkdown(request, now)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return MarkdownReport{}, err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return MarkdownReport{}, err
	}

	if err := writeArtifactMetadata(absRoot, path, ArtifactMetadata{
		Kind:           "chat-answer",
		Title:          generatedArtifactTitle(request),
		Source:         fallbackString(request.Source, "NexusDesk chat"),
		SourcePaths:    cleanMetadataPaths(request.SourcePaths),
		ContextRelPath: request.ContextRelPath,
		Prompt:         request.Prompt,
		Model:          request.Model,
		CreatedAt:      now.UTC().Format(time.RFC3339),
	}); err != nil {
		return MarkdownReport{}, err
	}

	info, err := file.Stat()
	if err != nil {
		return MarkdownReport{}, err
	}

	relPath, err := filepath.Rel(absRoot, path)
	if err != nil {
		return MarkdownReport{}, err
	}

	return MarkdownReport{
		RelPath: filepath.ToSlash(relPath),
		Name:    name,
		Path:    path,
		Message: "Assistant response artifact created inside the workspace.",
		Size:    info.Size(),
	}, nil
}

func CreateDatasetChartSVG(root string, chart workspace.DatasetChartResult, now time.Time) (MarkdownReport, error) {
	if strings.TrimSpace(root) == "" {
		return MarkdownReport{}, errors.New("open a workspace before creating chart artifacts")
	}
	if strings.TrimSpace(chart.RelPath) == "" || len(chart.Points) == 0 {
		return MarkdownReport{}, errors.New("chart artifact needs dataset points")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return MarkdownReport{}, err
	}

	artifactDir := filepath.Join(absRoot, filepath.FromSlash(artifactDirRelPath))
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return MarkdownReport{}, err
	}

	name := datasetChartFileName(chart, now)
	path := filepath.Join(artifactDir, name)
	if err := ensureInsideRoot(absRoot, path); err != nil {
		return MarkdownReport{}, err
	}

	content := buildDatasetChartSVG(chart, now)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return MarkdownReport{}, err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return MarkdownReport{}, err
	}

	title := datasetChartTitle(chart)
	if err := writeArtifactMetadata(absRoot, path, ArtifactMetadata{
		Kind:           "chart-svg",
		Title:          title,
		Source:         "dataset chart",
		SourcePaths:    cleanMetadataPaths([]string{chart.RelPath}),
		ContextRelPath: chart.RelPath,
		Prompt:         datasetChartPrompt(chart),
		CreatedAt:      now.UTC().Format(time.RFC3339),
	}); err != nil {
		return MarkdownReport{}, err
	}

	info, err := file.Stat()
	if err != nil {
		return MarkdownReport{}, err
	}

	relPath, err := filepath.Rel(absRoot, path)
	if err != nil {
		return MarkdownReport{}, err
	}

	return MarkdownReport{
		RelPath: filepath.ToSlash(relPath),
		Name:    name,
		Path:    path,
		Message: "SVG chart artifact created inside the workspace.",
		Size:    info.Size(),
	}, nil
}

func CreateDatasetQueryCSV(root string, result workspace.DatasetQueryResult, now time.Time) (MarkdownReport, error) {
	if strings.TrimSpace(root) == "" {
		return MarkdownReport{}, errors.New("open a workspace before exporting dataset queries")
	}
	if strings.TrimSpace(result.RelPath) == "" || len(result.Columns) == 0 {
		return MarkdownReport{}, errors.New("dataset query export needs columns")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return MarkdownReport{}, err
	}

	artifactDir := filepath.Join(absRoot, filepath.FromSlash(artifactDirRelPath))
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return MarkdownReport{}, err
	}

	name := datasetQueryFileName(result, now)
	path := filepath.Join(artifactDir, name)
	if err := ensureInsideRoot(absRoot, path); err != nil {
		return MarkdownReport{}, err
	}

	content, err := buildDatasetQueryCSV(result)
	if err != nil {
		return MarkdownReport{}, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return MarkdownReport{}, err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return MarkdownReport{}, err
	}

	if err := writeArtifactMetadata(absRoot, path, ArtifactMetadata{
		Kind:           "dataset-query-csv",
		Title:          datasetQueryTitle(result),
		Source:         "dataset query",
		SourcePaths:    cleanMetadataPaths([]string{result.RelPath}),
		ContextRelPath: result.RelPath,
		Prompt:         datasetQueryPrompt(result),
		CreatedAt:      now.UTC().Format(time.RFC3339),
	}); err != nil {
		return MarkdownReport{}, err
	}

	info, err := file.Stat()
	if err != nil {
		return MarkdownReport{}, err
	}

	relPath, err := filepath.Rel(absRoot, path)
	if err != nil {
		return MarkdownReport{}, err
	}

	return MarkdownReport{
		RelPath: filepath.ToSlash(relPath),
		Name:    name,
		Path:    path,
		Message: "CSV dataset query artifact created inside the workspace.",
		Size:    info.Size(),
	}, nil
}

func CreateDatasetSummaryMarkdown(root string, source workspace.FilePreview, now time.Time) (MarkdownReport, error) {
	if strings.TrimSpace(root) == "" {
		return MarkdownReport{}, errors.New("open a workspace before creating dataset summaries")
	}
	if source.Table == nil {
		return MarkdownReport{}, errors.New("dataset summary requires a CSV table preview")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return MarkdownReport{}, err
	}

	artifactDir := filepath.Join(absRoot, filepath.FromSlash(artifactDirRelPath))
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return MarkdownReport{}, err
	}

	name := datasetSummaryFileName(source, now)
	path := filepath.Join(artifactDir, name)
	if err := ensureInsideRoot(absRoot, path); err != nil {
		return MarkdownReport{}, err
	}

	content := buildDatasetSummaryMarkdown(source, now)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return MarkdownReport{}, err
	}
	defer file.Close()

	if _, err := file.WriteString(content); err != nil {
		return MarkdownReport{}, err
	}

	if err := writeArtifactMetadata(absRoot, path, ArtifactMetadata{
		Kind:           "dataset-summary",
		Title:          "Dataset summary: " + source.Name,
		Source:         "dataset summary",
		SourcePaths:    cleanMetadataPaths([]string{source.RelPath}),
		ContextRelPath: source.RelPath,
		CreatedAt:      now.UTC().Format(time.RFC3339),
	}); err != nil {
		return MarkdownReport{}, err
	}

	info, err := file.Stat()
	if err != nil {
		return MarkdownReport{}, err
	}
	relPath, err := filepath.Rel(absRoot, path)
	if err != nil {
		return MarkdownReport{}, err
	}
	return MarkdownReport{
		RelPath: filepath.ToSlash(relPath),
		Name:    name,
		Path:    path,
		Message: "Dataset summary artifact created inside the workspace.",
		Size:    info.Size(),
	}, nil
}

func List(root string) ([]WorkspaceArtifact, error) {
	if strings.TrimSpace(root) == "" {
		return []WorkspaceArtifact{}, nil
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	artifactDir := filepath.Join(absRoot, filepath.FromSlash(artifactDirRelPath))
	entries, err := os.ReadDir(artifactDir)
	if errors.Is(err, os.ErrNotExist) {
		return []WorkspaceArtifact{}, nil
	}
	if err != nil {
		return nil, err
	}

	artifacts := make([]WorkspaceArtifact, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		if extension != ".md" && extension != ".svg" && extension != ".csv" {
			continue
		}

		path := filepath.Join(artifactDir, entry.Name())
		if err := ensureInsideRoot(absRoot, path); err != nil {
			return nil, err
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			return nil, err
		}
		metadata := readArtifactMetadata(path)

		artifacts = append(artifacts, WorkspaceArtifact{
			RelPath:    filepath.ToSlash(relPath),
			Name:       entry.Name(),
			Path:       path,
			Kind:       fallbackString(metadata.Kind, "markdown-report"),
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
			Source:     metadata.Source,
			Summary:    artifactSummary(metadata),
			Model:      metadata.Model,
		})
	}

	sort.SliceStable(artifacts, func(i, j int) bool {
		if artifacts[i].ModifiedAt == artifacts[j].ModifiedAt {
			return artifacts[i].Name < artifacts[j].Name
		}
		return artifacts[i].ModifiedAt > artifacts[j].ModifiedAt
	})

	return artifacts, nil
}

func Metadata(root string, relPath string) (ArtifactMetadata, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ArtifactMetadata{}, err
	}
	cleanRel := filepath.Clean(filepath.FromSlash(relPath))
	if cleanRel == "." || filepath.IsAbs(cleanRel) {
		return ArtifactMetadata{}, errors.New("artifact path must be relative")
	}
	path := filepath.Join(absRoot, cleanRel)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ArtifactMetadata{}, err
	}
	if err := ensureInsideRoot(absRoot, absPath); err != nil {
		return ArtifactMetadata{}, err
	}
	metadata := readArtifactMetadata(absPath)
	if metadata.Kind == "" {
		return ArtifactMetadata{}, errors.New("artifact metadata is unavailable")
	}
	return metadata, nil
}

func Search(root string, query string) ([]workspace.SearchResult, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, nil
	}
	artifacts, err := List(root)
	if err != nil {
		return nil, err
	}
	results := []workspace.SearchResult{}
	for _, item := range artifacts {
		metadata, _ := Metadata(root, item.RelPath)
		haystack := strings.ToLower(strings.Join([]string{
			item.RelPath,
			item.Name,
			item.Kind,
			item.Source,
			item.Summary,
			metadata.Title,
			metadata.Prompt,
			strings.Join(metadata.SourcePaths, " "),
		}, "\n"))
		if !strings.Contains(haystack, query) {
			continue
		}
		results = append(results, workspace.SearchResult{
			RelPath:   item.RelPath,
			Name:      item.Name,
			Kind:      "file",
			FileType:  artifactFileType(item),
			MatchType: "artifact",
			Snippet:   artifactSearchSnippet(item, metadata),
		})
	}
	return results, nil
}

func reportFileName(source workspace.FilePreview, now time.Time) string {
	base := strings.TrimSuffix(source.Name, filepath.Ext(source.Name))
	if base == "" {
		base = "workspace-report"
	}

	slug := slugify(base)
	if slug == "" {
		slug = "workspace-report"
	}

	return fmt.Sprintf("%s-%s.md", slug, now.UTC().Format("20060102-150405"))
}

func generatedArtifactFileName(request MarkdownArtifactRequest, now time.Time) string {
	base := generatedArtifactTitle(request)

	slug := slugify(base)
	if slug == "" {
		slug = "assistant-response"
	}

	return fmt.Sprintf("%s-%s.md", slug, now.UTC().Format("20060102-150405"))
}

func datasetChartFileName(chart workspace.DatasetChartResult, now time.Time) string {
	slug := slugify(datasetChartTitle(chart))
	if slug == "" {
		slug = "dataset-chart"
	}
	return fmt.Sprintf("%s-%s.svg", slug, now.UTC().Format("20060102-150405"))
}

func datasetQueryFileName(result workspace.DatasetQueryResult, now time.Time) string {
	base := strings.TrimSuffix(filepath.Base(result.RelPath), filepath.Ext(result.RelPath))
	if base == "" {
		base = "dataset"
	}
	suffix := "rows"
	if strings.TrimSpace(result.Query) != "" {
		suffix = "query"
	}
	slug := slugify(base + "-" + suffix)
	if slug == "" {
		slug = "dataset-query"
	}
	return fmt.Sprintf("%s-%s.csv", slug, now.UTC().Format("20060102-150405"))
}

func datasetSummaryFileName(source workspace.FilePreview, now time.Time) string {
	base := strings.TrimSuffix(source.Name, filepath.Ext(source.Name))
	if base == "" {
		base = "dataset"
	}
	slug := slugify(base + "-summary")
	if slug == "" {
		slug = "dataset-summary"
	}
	return fmt.Sprintf("%s-%s.md", slug, now.UTC().Format("20060102-150405"))
}

func generatedArtifactTitle(request MarkdownArtifactRequest) string {
	base := strings.TrimSpace(request.Title)
	if base == "" {
		base = "Assistant Response"
	}
	return base
}

func datasetChartTitle(chart workspace.DatasetChartResult) string {
	metric := "Rows"
	if chart.ValueColumn != "" {
		metric = chart.ValueColumn
	}
	category := strings.TrimSpace(chart.CategoryColumn)
	if category == "" {
		category = "Category"
	}
	return fmt.Sprintf("%s by %s", metric, category)
}

func datasetChartPrompt(chart workspace.DatasetChartResult) string {
	if chart.ValueColumn == "" {
		return fmt.Sprintf("Chart row count by %s from %s", chart.CategoryColumn, chart.RelPath)
	}
	return fmt.Sprintf("Chart sum of %s by %s from %s", chart.ValueColumn, chart.CategoryColumn, chart.RelPath)
}

func datasetQueryTitle(result workspace.DatasetQueryResult) string {
	if strings.TrimSpace(result.Query) == "" {
		return fmt.Sprintf("Rows from %s", result.RelPath)
	}
	return fmt.Sprintf("Query %q from %s", result.Query, result.RelPath)
}

func datasetQueryPrompt(result workspace.DatasetQueryResult) string {
	if strings.TrimSpace(result.Query) == "" {
		return fmt.Sprintf("Export first %d rows from %s", len(result.Rows), result.RelPath)
	}
	return fmt.Sprintf("Export query %q from %s", result.Query, result.RelPath)
}

func buildMarkdownReport(source workspace.FilePreview, now time.Time) string {
	var builder strings.Builder

	title := source.Name
	if title == "" {
		title = "Workspace Report"
	}

	builder.WriteString("# Report: ")
	builder.WriteString(escapeMarkdownLine(title))
	builder.WriteString("\n\n")
	builder.WriteString("- Generated: ")
	builder.WriteString(now.UTC().Format(time.RFC3339))
	builder.WriteString("\n")
	if source.RelPath != "" {
		builder.WriteString("- Source: `")
		builder.WriteString(strings.ReplaceAll(source.RelPath, "`", "'"))
		builder.WriteString("`\n")
	}
	if source.FileType != "" {
		builder.WriteString("- Type: ")
		builder.WriteString(source.FileType)
		builder.WriteString("\n")
	}
	if source.Encoding != "" {
		builder.WriteString("- Encoding: ")
		builder.WriteString(source.Encoding)
		builder.WriteString("\n")
	}
	if source.Size > 0 {
		builder.WriteString("- Source bytes: ")
		builder.WriteString(fmt.Sprintf("%d", source.Size))
		builder.WriteString("\n")
	}
	builder.WriteString("\n## Summary\n\n")
	builder.WriteString("Draft the key findings here.\n\n")
	builder.WriteString("## Source Excerpt\n\n")

	if source.Kind != "file" || strings.TrimSpace(source.Content) == "" {
		builder.WriteString("_No text excerpt was available for this source._\n")
	} else {
		excerpt := source.Content
		if len(excerpt) > reportContentLimit {
			excerpt = excerpt[:reportContentLimit]
		}
		builder.WriteString("````text\n")
		builder.WriteString(excerpt)
		if !strings.HasSuffix(excerpt, "\n") {
			builder.WriteString("\n")
		}
		builder.WriteString("````\n")
		if source.Truncated || len(source.Content) > reportContentLimit {
			builder.WriteString("\n_Source excerpt was truncated._\n")
		}
	}

	builder.WriteString("\n## Next Actions\n\n")
	builder.WriteString("- Review source context.\n")
	builder.WriteString("- Add conclusions and owner notes.\n")
	builder.WriteString("- Attach supporting artifacts where needed.\n")

	return builder.String()
}

func buildDatasetQueryCSV(result workspace.DatasetQueryResult) (string, error) {
	var builder strings.Builder
	writer := csv.NewWriter(&builder)
	if err := writer.Write(result.Columns); err != nil {
		return "", err
	}
	for _, row := range result.Rows {
		if err := writer.Write(row); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return builder.String(), nil
}

func buildDatasetSummaryMarkdown(source workspace.FilePreview, now time.Time) string {
	var builder strings.Builder
	builder.WriteString("# Dataset Summary: ")
	builder.WriteString(escapeMarkdownLine(source.Name))
	builder.WriteString("\n\n")
	builder.WriteString("- Generated: ")
	builder.WriteString(now.UTC().Format(time.RFC3339))
	builder.WriteString("\n")
	builder.WriteString("- Source: `")
	builder.WriteString(strings.ReplaceAll(source.RelPath, "`", "'"))
	builder.WriteString("`\n")
	builder.WriteString(fmt.Sprintf("- Rows: %d\n", source.Table.TotalRows))
	builder.WriteString(fmt.Sprintf("- Columns: %d\n", len(source.Table.Columns)))
	if source.Table.Truncated || source.Truncated {
		builder.WriteString("- Note: Source preview/profile was bounded for responsiveness.\n")
	}

	builder.WriteString("\n## Columns\n\n")
	builder.WriteString("| Column | Type | Missing | Distinct | Range |\n")
	builder.WriteString("|---|---:|---:|---:|---|\n")
	for _, profile := range source.Table.Profiles {
		valueRange := ""
		if profile.Min != "" || profile.Max != "" {
			valueRange = profile.Min + "..." + profile.Max
		}
		builder.WriteString("| ")
		builder.WriteString(escapeMarkdownCell(profile.Name))
		builder.WriteString(" | ")
		builder.WriteString(profile.Type)
		builder.WriteString(fmt.Sprintf(" | %d | %d | ", profile.Missing, profile.Distinct))
		builder.WriteString(escapeMarkdownCell(valueRange))
		builder.WriteString(" |\n")
	}

	builder.WriteString("\n## Suggested Questions\n\n")
	for _, profile := range source.Table.Profiles {
		if profile.Type == "number" || profile.Type == "integer" {
			builder.WriteString("- Which segments explain the largest values in `")
			builder.WriteString(strings.ReplaceAll(profile.Name, "`", "'"))
			builder.WriteString("`?\n")
			continue
		}
		if profile.Distinct > 1 && profile.Distinct <= 30 {
			builder.WriteString("- How do rows break down by `")
			builder.WriteString(strings.ReplaceAll(profile.Name, "`", "'"))
			builder.WriteString("`?\n")
		}
	}
	builder.WriteString("- Which rows are missing important values?\n")
	builder.WriteString("- What chart best communicates the top categories or trends?\n")
	return builder.String()
}

func buildDatasetChartSVG(chart workspace.DatasetChartResult, now time.Time) string {
	const width = 960.0
	const rowHeight = 42.0
	const top = 116.0
	const left = 220.0
	const right = 52.0
	const barHeight = 20.0

	height := top + float64(len(chart.Points))*rowHeight + 70
	maxValue := 0.0
	for _, point := range chart.Points {
		if point.Value > maxValue {
			maxValue = point.Value
		}
	}
	if maxValue <= 0 {
		maxValue = 1
	}

	title := datasetChartTitle(chart)
	subtitle := chart.Message
	barMaxWidth := width - left - right

	var builder strings.Builder
	builder.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	builder.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%.0f" height="%.0f" viewBox="0 0 %.0f %.0f" role="img" aria-labelledby="title desc">`+"\n", width, height, width, height))
	builder.WriteString("<title>")
	builder.WriteString(html.EscapeString(title))
	builder.WriteString("</title>\n<desc>")
	builder.WriteString(html.EscapeString(subtitle))
	builder.WriteString("</desc>\n")
	builder.WriteString(`<rect width="100%" height="100%" fill="#f7f5f0"/>` + "\n")
	builder.WriteString(`<text x="40" y="48" fill="#1f2933" font-family="Inter, Segoe UI, Arial, sans-serif" font-size="28" font-weight="700">`)
	builder.WriteString(html.EscapeString(title))
	builder.WriteString("</text>\n")
	builder.WriteString(`<text x="40" y="78" fill="#65717f" font-family="Inter, Segoe UI, Arial, sans-serif" font-size="14">`)
	builder.WriteString(html.EscapeString(subtitle))
	builder.WriteString("</text>\n")
	builder.WriteString(`<line x1="40" y1="96" x2="920" y2="96" stroke="#d8d4ca" stroke-width="1"/>` + "\n")

	if chart.ChartType == "line" {
		writeLineChartSVG(&builder, chart, left, top, barMaxWidth, float64(len(chart.Points))*rowHeight, maxValue)
	} else {
		for index, point := range chart.Points {
			y := top + float64(index)*rowHeight
			barWidth := math.Max(2, (point.Value/maxValue)*barMaxWidth)
			builder.WriteString(fmt.Sprintf(`<text x="40" y="%.1f" fill="#30363d" font-family="Inter, Segoe UI, Arial, sans-serif" font-size="14">%s</text>`+"\n", y+15, html.EscapeString(truncateChartLabel(point.Label))))
			builder.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="4" fill="#2f7d7e"/>`+"\n", left, y, barWidth, barHeight))
			builder.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" fill="#1f2933" font-family="Inter, Segoe UI, Arial, sans-serif" font-size="13">%s</text>`+"\n", left+barWidth+10, y+15, html.EscapeString(formatChartValue(point.Value))))
		}
	}

	builder.WriteString(fmt.Sprintf(`<text x="40" y="%.1f" fill="#65717f" font-family="Inter, Segoe UI, Arial, sans-serif" font-size="12">Generated %s from %s. Rows used: %d of %d.</text>`+"\n", height-28, html.EscapeString(now.UTC().Format(time.RFC3339)), html.EscapeString(chart.RelPath), chart.UsedRows, chart.TotalRows))
	builder.WriteString("</svg>\n")
	return builder.String()
}

func writeLineChartSVG(builder *strings.Builder, chart workspace.DatasetChartResult, left float64, top float64, width float64, height float64, maxValue float64) {
	if len(chart.Points) == 0 {
		return
	}
	bottom := top + math.Max(80, height-20)
	step := width
	if len(chart.Points) > 1 {
		step = width / float64(len(chart.Points)-1)
	}

	points := []string{}
	for index, point := range chart.Points {
		x := left + float64(index)*step
		y := bottom - (point.Value/maxValue)*math.Max(40, height-50)
		points = append(points, fmt.Sprintf("%.1f,%.1f", x, y))
		builder.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="4" fill="#2f7d7e"/>`+"\n", x, y))
		builder.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" fill="#30363d" font-family="Inter, Segoe UI, Arial, sans-serif" font-size="11" text-anchor="middle">%s</text>`+"\n", x, bottom+20, html.EscapeString(truncateChartLabel(point.Label))))
		builder.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" fill="#1f2933" font-family="Inter, Segoe UI, Arial, sans-serif" font-size="12" text-anchor="middle">%s</text>`+"\n", x, y-10, html.EscapeString(formatChartValue(point.Value))))
	}
	builder.WriteString(fmt.Sprintf(`<polyline points="%s" fill="none" stroke="#2f7d7e" stroke-width="3"/>`+"\n", strings.Join(points, " ")))
}

func buildGeneratedMarkdown(request MarkdownArtifactRequest, now time.Time) string {
	var builder strings.Builder

	title := strings.TrimSpace(request.Title)
	if title == "" {
		title = "Assistant Response"
	}
	source := strings.TrimSpace(request.Source)
	if source == "" {
		source = "Assistant response"
	}

	content := truncateValidUTF8(request.Content, generatedArtifactContentLimit)

	builder.WriteString("# ")
	builder.WriteString(escapeMarkdownLine(title))
	builder.WriteString("\n\n")
	builder.WriteString("- Generated: ")
	builder.WriteString(now.UTC().Format(time.RFC3339))
	builder.WriteString("\n")
	builder.WriteString("- Source: ")
	builder.WriteString(escapeMarkdownLine(source))
	builder.WriteString("\n")
	if strings.TrimSpace(request.ContextRelPath) != "" {
		builder.WriteString("- Context: `")
		builder.WriteString(strings.ReplaceAll(request.ContextRelPath, "`", "'"))
		builder.WriteString("`\n")
	}
	builder.WriteString("\n")
	builder.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		builder.WriteString("\n")
	}
	if len(request.Content) > len(content) {
		builder.WriteString("\n_Response content was truncated._\n")
	}

	return builder.String()
}

func truncateChartLabel(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "(blank)"
	}
	const maxRunes = 24
	runes := []rune(label)
	if len(runes) <= maxRunes {
		return label
	}
	return string(runes[:maxRunes-1]) + "..."
}

func formatChartValue(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func artifactFileType(item WorkspaceArtifact) string {
	switch strings.ToLower(filepath.Ext(item.Name)) {
	case ".csv":
		return "data"
	case ".svg":
		return "image"
	default:
		return "document"
	}
}

func artifactSearchSnippet(item WorkspaceArtifact, metadata ArtifactMetadata) string {
	if metadata.Title != "" {
		return metadata.Title
	}
	if metadata.Prompt != "" {
		return metadata.Prompt
	}
	if item.Summary != "" {
		return item.Summary
	}
	return item.Source
}

func escapeMarkdownCell(value string) string {
	return strings.ReplaceAll(escapeMarkdownLine(value), "|", "\\|")
}

func writeArtifactMetadata(root string, artifactPath string, metadata ArtifactMetadata) error {
	metadataPath := artifactMetadataPath(artifactPath)
	if err := ensureInsideRoot(root, metadataPath); err != nil {
		return err
	}

	payload, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	file, err := os.OpenFile(metadataPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(payload)
	return err
}

func readArtifactMetadata(artifactPath string) ArtifactMetadata {
	content, err := os.ReadFile(artifactMetadataPath(artifactPath))
	if err != nil {
		return ArtifactMetadata{}
	}

	var metadata ArtifactMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		return ArtifactMetadata{}
	}
	return metadata
}

func artifactMetadataPath(artifactPath string) string {
	extension := filepath.Ext(artifactPath)
	return strings.TrimSuffix(artifactPath, extension) + ".meta.json"
}

func artifactSummary(metadata ArtifactMetadata) string {
	if len(metadata.SourcePaths) > 0 {
		if len(metadata.SourcePaths) == 1 {
			return metadata.SourcePaths[0]
		}
		return fmt.Sprintf("%d source paths", len(metadata.SourcePaths))
	}
	if metadata.ContextRelPath != "" {
		return metadata.ContextRelPath
	}
	if metadata.Prompt != "" {
		return strings.TrimSpace(escapeMarkdownLine(metadata.Prompt))
	}
	return ""
}

func cleanMetadataPaths(paths []string) []string {
	cleaned := []string{}
	seen := map[string]bool{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		cleaned = append(cleaned, path)
	}
	return cleaned
}

func fallbackString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func slugify(value string) string {
	value = strings.ToLower(value)
	value = nonSlugCharacters.ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	if len(value) > 48 {
		value = strings.Trim(value[:48], "-")
	}
	return value
}

func escapeMarkdownLine(value string) string {
	return strings.ReplaceAll(value, "\n", " ")
}

func truncateValidUTF8(content string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(content) <= maxBytes {
		return content
	}

	truncated := content[:maxBytes]
	for !utf8.ValidString(truncated) && len(truncated) > 0 {
		truncated = truncated[:len(truncated)-1]
	}
	return truncated
}

func ensureInsideRoot(root string, target string) error {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return errors.New("artifact path must stay inside the workspace")
	}
	return nil
}

var nonSlugCharacters = regexp.MustCompile(`[^a-z0-9]+`)
