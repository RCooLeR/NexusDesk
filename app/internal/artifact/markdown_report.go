package artifact

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
		if strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
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

func generatedArtifactTitle(request MarkdownArtifactRequest) string {
	base := strings.TrimSpace(request.Title)
	if base == "" {
		base = "Assistant Response"
	}
	return base
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
