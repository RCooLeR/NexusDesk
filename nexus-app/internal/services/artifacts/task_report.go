package artifacts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) WriteTaskRunReport(record TaskRunReport) (Artifact, error) {
	createdAt := time.Now().UTC()
	stamp := createdAt.Format("20060102-150405-000000000")
	identifier := strings.TrimSpace(record.ID)
	if identifier == "" {
		identifier = strings.TrimSpace(record.JobID)
	}
	if identifier == "" {
		identifier = "task-run"
	}
	if len(identifier) > 16 {
		identifier = identifier[:16]
	}
	relPath := s.relPath("task-runs", fmt.Sprintf("%s-%s.md", stamp, safeName(identifier)))
	absPath := s.absPath(relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return Artifact{}, err
	}
	content := taskRunMarkdown(record, createdAt)
	file, err := os.OpenFile(absPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return Artifact{}, err
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		return Artifact{}, err
	}
	return Artifact{
		Kind:      "task-report",
		Title:     record.Label,
		RelPath:   relPath,
		AbsPath:   absPath,
		Message:   "Task report artifact created at " + relPath + ".",
		CreatedAt: createdAt,
	}, nil
}

func taskRunMarkdown(record TaskRunReport, createdAt time.Time) string {
	var builder strings.Builder
	builder.WriteString("# Task Run Report\n\n")
	writeKV(&builder, "Task", record.Label)
	writeKV(&builder, "Status", record.Status)
	writeKV(&builder, "Exit code", fmt.Sprintf("%d", record.ExitCode))
	writeKV(&builder, "Job", record.JobID)
	writeKV(&builder, "Task ID", record.TaskID)
	writeKV(&builder, "Kind", record.Kind)
	writeKV(&builder, "Command", record.Command)
	writeKV(&builder, "Working directory", record.Cwd)
	writeKV(&builder, "Source", record.Source)
	writeKV(&builder, "Started", formatArtifactTime(record.StartedAt))
	writeKV(&builder, "Completed", formatArtifactTime(record.CompletedAt))
	writeKV(&builder, "Duration", fmt.Sprintf("%d ms", record.DurationMs))
	writeKV(&builder, "Generated", formatArtifactTime(createdAt))
	builder.WriteString("\n## Summary\n\n")
	builder.WriteString(strings.TrimSpace(record.Message))
	builder.WriteString("\n\n## Stdout\n\n")
	builder.WriteString(indentedLog(record.Stdout))
	builder.WriteString("\n## Stderr\n\n")
	builder.WriteString(indentedLog(record.Stderr))
	return builder.String()
}

func writeKV(builder *strings.Builder, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "-"
	}
	builder.WriteString("- **")
	builder.WriteString(key)
	builder.WriteString(":** ")
	builder.WriteString(value)
	builder.WriteString("\n")
}

func indentedLog(value string) string {
	value = strings.TrimRight(value, "\r\n")
	if value == "" {
		return "    (empty)\n\n"
	}
	lines := strings.Split(value, "\n")
	var builder strings.Builder
	for _, line := range lines {
		builder.WriteString("    ")
		builder.WriteString(strings.TrimRight(line, "\r"))
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
	return builder.String()
}

func formatArtifactTime(value time.Time) string {
	if value.IsZero() {
		return "-"
	}
	return value.UTC().Format(time.RFC3339)
}

func safeName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if allowed {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteRune('-')
			lastDash = true
		}
	}
	name := strings.Trim(builder.String(), "-")
	if name == "" {
		return "task-run"
	}
	return name
}
