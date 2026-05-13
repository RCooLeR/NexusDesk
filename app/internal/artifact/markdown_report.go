package artifact

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"NexusDesk/internal/workspace"
)

const reportContentLimit = 12 * 1024
const artifactDirRelPath = ".nexusdesk/artifacts"

type MarkdownReport struct {
	RelPath string `json:"relPath"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Message string `json:"message"`
	Size    int64  `json:"size"`
}

type WorkspaceArtifact struct {
	RelPath    string `json:"relPath"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modifiedAt"`
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

		artifacts = append(artifacts, WorkspaceArtifact{
			RelPath:    filepath.ToSlash(relPath),
			Name:       entry.Name(),
			Path:       path,
			Kind:       "markdown-report",
			Size:       info.Size(),
			ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
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
