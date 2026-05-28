package artifacts

import (
	"path/filepath"
	"strings"
	"time"
)

func inferKind(relPath string) string {
	normalized := strings.ToLower(filepath.ToSlash(relPath))
	switch {
	case strings.Contains(normalized, "/task-runs/"):
		return "task-report"
	case strings.Contains(normalized, "/document-sets/"):
		return "document-report"
	case strings.Contains(normalized, "/document-extracts/"):
		return "document-extract"
	case strings.Contains(normalized, "/comparisons/"):
		return "artifact-comparison"
	case strings.Contains(normalized, "/chat-answers/"):
		return "chat-answer"
	case strings.Contains(normalized, "/dashboards/"):
		return "dashboard"
	case strings.HasSuffix(normalized, ".md"):
		return "markdown"
	case strings.HasSuffix(normalized, ".csv"):
		return "csv"
	case strings.HasSuffix(normalized, ".svg"):
		return "chart"
	default:
		return strings.TrimPrefix(strings.ToLower(filepath.Ext(normalized)), ".")
	}
}

func artifactMatches(artifact Artifact, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	haystack := strings.ToLower(strings.Join([]string{
		artifact.Kind,
		artifact.Title,
		artifact.RelPath,
		artifact.JobID,
		artifact.TaskID,
		artifact.Source,
		strings.Join(artifact.SourcePaths, " "),
	}, " "))
	if strings.HasPrefix(query, "kind:") {
		return strings.Contains(strings.ToLower(artifact.Kind), strings.TrimSpace(strings.TrimPrefix(query, "kind:")))
	}
	return strings.Contains(haystack, query)
}

func firstTime(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value
		}
	}
	return time.Time{}
}

func artifactTitle(artifact Artifact) string {
	if artifact.Title != "" {
		return artifact.Title
	}
	return filepath.Base(artifact.RelPath)
}
