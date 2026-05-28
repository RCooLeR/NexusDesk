package artifacts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) WriteWorkspaceScanReport(report WorkspaceScanReport) (Artifact, error) {
	if strings.TrimSpace(report.WorkspaceName) == "" {
		return Artifact{}, errors.New("workspace scan report requires a workspace name")
	}
	createdAt := time.Now().UTC()
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = "Workspace Scan Report - " + strings.TrimSpace(report.WorkspaceName)
	}
	markdown := workspaceScanMarkdown(report, title, createdAt)
	relPath := s.relPath("scan-reports", fmt.Sprintf("%s-%s.md", createdAt.Format("20060102-150405-000000000"), safeName(title)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.WriteString(markdown); err != nil {
		return Artifact{}, err
	}
	metadata := Metadata{
		Kind:           "scan-report",
		Title:          title,
		RelPath:        relPath,
		Source:         "workspace scan",
		ContextRelPath: ".",
		SourcePaths:    []string{"."},
		GeneratedAt:    createdAt,
	}
	if err := s.writeMetadata(metadata); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:         metadata.Kind,
		Title:        title,
		RelPath:      relPath,
		AbsPath:      absPath,
		MetadataPath: relPath + ".json",
		Message:      "Workspace scan report artifact created at " + relPath + ".",
		Size:         int64(len(markdown)),
		CreatedAt:    createdAt,
		GeneratedAt:  createdAt,
		Source:       metadata.Source,
		SourcePaths:  []string{"."},
	}, nil
}

func workspaceScanMarkdown(report WorkspaceScanReport, title string, createdAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	writeKV(&builder, "Generated", formatArtifactTime(createdAt))
	writeKV(&builder, "Workspace", report.WorkspaceName)
	writeKV(&builder, "Indexed entries", fmt.Sprintf("%d", report.Included))
	writeKV(&builder, "Skipped entries", fmt.Sprintf("%d", workspaceScanSkipped(report)))
	writeKV(&builder, "Max depth", fmt.Sprintf("%d", report.MaxDepth))
	writeKV(&builder, "Entry cap", fmt.Sprintf("%d", report.MaxEntries))
	writeKV(&builder, "Truncated", fmt.Sprintf("%t", report.Truncated))
	writeKV(&builder, "Message", report.Message)

	builder.WriteString("\n## Counters\n\n")
	writeMarkdownTable(&builder, []string{"Counter", "Value"}, [][]string{
		{"Included", fmt.Sprintf("%d", report.Included)},
		{"Ignored", fmt.Sprintf("%d", report.Ignored)},
		{"Depth skipped", fmt.Sprintf("%d", report.DepthSkipped)},
		{"Entry cap skipped", fmt.Sprintf("%d", report.EntrySkipped)},
		{"Unreadable", fmt.Sprintf("%d", report.Unreadable)},
	})

	builder.WriteString("\n## Samples\n\n")
	writeWorkspaceScanSamples(&builder, "Ignored", report.IgnoredSamples)
	writeWorkspaceScanSamples(&builder, "Skipped", report.SkippedSamples)
	builder.WriteString("## Next Actions\n\n")
	builder.WriteString("- Expand the workspace tree where relevant.\n")
	builder.WriteString("- Search for target files before adding large folders to context.\n")
	builder.WriteString("- Review ignored or skipped samples if expected files are missing.\n")
	return builder.String()
}

func workspaceScanSkipped(report WorkspaceScanReport) int {
	return report.Ignored + report.DepthSkipped + report.EntrySkipped + report.Unreadable
}

func writeWorkspaceScanSamples(builder *strings.Builder, title string, samples []string) {
	builder.WriteString("### ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	if len(samples) == 0 {
		builder.WriteString("_No samples recorded._\n\n")
		return
	}
	for _, sample := range samples {
		builder.WriteString("- `")
		builder.WriteString(strings.ReplaceAll(sample, "`", "'"))
		builder.WriteString("`\n")
	}
	builder.WriteString("\n")
}
