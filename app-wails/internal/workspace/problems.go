package workspace

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultProblemMaxResults = 80
	problemPreviewMaxBytes   = 64 * 1024
)

type ProblemSummary struct {
	Problems    []WorkspaceProblem `json:"problems"`
	Message     string             `json:"message"`
	GeneratedAt string             `json:"generatedAt"`
	Truncated   bool               `json:"truncated"`
}

type WorkspaceProblem struct {
	RelPath  string `json:"relPath"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
}

func ScanProblems(root string, maxResults int) (ProblemSummary, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return ProblemSummary{}, err
	}
	if maxResults <= 0 {
		maxResults = defaultProblemMaxResults
	}

	problems := []WorkspaceProblem{}
	truncated := false
	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || path == absRoot {
			return nil
		}

		relPath, err := filepath.Rel(absRoot, path)
		if err != nil {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		depth := strings.Count(relPath, "/") + 1
		if shouldIgnore(relPath, entry) || depth > defaultMaxDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if len(problems) >= maxResults {
			truncated = true
			return nil
		}

		problems = append(problems, scanFileProblems(absRoot, relPath)...)
		if len(problems) > maxResults {
			problems = problems[:maxResults]
			truncated = true
		}
		return nil
	})
	if err != nil {
		return ProblemSummary{}, err
	}

	sort.SliceStable(problems, func(i, j int) bool {
		if problems[i].Severity != problems[j].Severity {
			return problemSeverityRank(problems[i].Severity) < problemSeverityRank(problems[j].Severity)
		}
		if problems[i].RelPath == problems[j].RelPath {
			return problems[i].Line < problems[j].Line
		}
		return compareSearchPaths(problems[i].RelPath, problems[j].RelPath)
	})

	message := "No workspace problems detected by lightweight scanners."
	if len(problems) > 0 {
		message = fmt.Sprintf("%d workspace problems detected by lightweight scanners.", len(problems))
	}
	if truncated {
		message += " Results truncated."
	}
	return ProblemSummary{
		Problems:    problems,
		Message:     message,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Truncated:   truncated,
	}, nil
}

func scanFileProblems(root string, relPath string) []WorkspaceProblem {
	preview, err := Preview(root, relPath, PreviewOptions{MaxBytes: problemPreviewMaxBytes})
	if err != nil || strings.TrimSpace(preview.Content) == "" {
		return nil
	}

	problems := markerProblems(preview)
	if strings.EqualFold(filepath.Ext(preview.Name), ".json") && !json.Valid([]byte(preview.Content)) {
		problems = append(problems, jsonProblem(preview))
	}
	return problems
}

func markerProblems(preview FilePreview) []WorkspaceProblem {
	lines := strings.Split(strings.ReplaceAll(preview.Content, "\r\n", "\n"), "\n")
	problems := []WorkspaceProblem{}
	for index, line := range lines {
		if index >= 4000 || len(problems) >= 8 {
			break
		}
		upper := strings.ToUpper(line)
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "<<<<<<<") || strings.HasPrefix(trimmed, "=======") || strings.HasPrefix(trimmed, ">>>>>>>"):
			problems = append(problems, problemFromLine(preview, "error", "merge-conflict", "Merge conflict marker", index+1, line))
		case strings.Contains(upper, "FIXME") || strings.Contains(upper, "BUG"):
			problems = append(problems, problemFromLine(preview, "warning", "marker", "Fix marker", index+1, line))
		case strings.Contains(upper, "TODO") || strings.Contains(upper, "HACK"):
			problems = append(problems, problemFromLine(preview, "info", "marker", "Task marker", index+1, line))
		}
	}
	return problems
}

func jsonProblem(preview FilePreview) WorkspaceProblem {
	var value any
	err := json.Unmarshal([]byte(preview.Content), &value)
	line := 1
	message := "Invalid JSON"
	if err != nil {
		message = err.Error()
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			line = lineForOffset(preview.Content, syntaxErr.Offset)
		}
	}
	return WorkspaceProblem{
		RelPath:  preview.RelPath,
		Name:     preview.Name,
		Severity: "error",
		Source:   "json",
		Message:  message,
		Line:     line,
	}
}

func problemFromLine(preview FilePreview, severity string, source string, message string, lineNumber int, line string) WorkspaceProblem {
	snippet := strings.TrimSpace(line)
	if len(snippet) > 120 {
		snippet = snippet[:117] + "..."
	}
	if snippet != "" {
		message += ": " + snippet
	}
	return WorkspaceProblem{
		RelPath:  preview.RelPath,
		Name:     preview.Name,
		Severity: severity,
		Source:   source,
		Message:  message,
		Line:     lineNumber,
	}
}

func lineForOffset(content string, offset int64) int {
	if offset <= 0 {
		return 1
	}
	line := 1
	for index, char := range content {
		if int64(index) >= offset {
			break
		}
		if char == '\n' {
			line++
		}
	}
	return line
}

func problemSeverityRank(severity string) int {
	switch severity {
	case "error":
		return 0
	case "warning":
		return 1
	default:
		return 2
	}
}
