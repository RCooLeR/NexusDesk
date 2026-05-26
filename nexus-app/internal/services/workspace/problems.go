package workspace

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"nexusdesk/internal/domain"
)

const (
	defaultProblemMaxResults = 80
	problemPreviewMaxBytes   = 64 * 1024
)

type ProblemSummary struct {
	Problems    []WorkspaceProblem
	Message     string
	GeneratedAt time.Time
	Truncated   bool
}

type WorkspaceProblem struct {
	RelPath  string
	Name     string
	Severity string
	Source   string
	Message  string
	Line     int
}

func (s *Service) ScanProblems(root string, maxResults int) (ProblemSummary, error) {
	absRoot, err := cleanRoot(root)
	if err != nil {
		return ProblemSummary{}, err
	}
	if maxResults <= 0 {
		maxResults = defaultProblemMaxResults
	}

	problemService := *s
	problemService.previewByteLimit = problemPreviewMaxBytes
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
		if shouldSkipSearchPath(relPath, entry) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if depthOf(relPath) > defaultSearchMaxDepth {
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
		problems = append(problems, problemService.scanFileProblems(absRoot, relPath)...)
		if len(problems) > maxResults {
			problems = problems[:maxResults]
			truncated = true
		}
		return nil
	})
	if err != nil {
		return ProblemSummary{}, err
	}

	sort.SliceStable(problems, func(left int, right int) bool {
		if problems[left].Severity != problems[right].Severity {
			return problemSeverityRank(problems[left].Severity) < problemSeverityRank(problems[right].Severity)
		}
		if problems[left].RelPath == problems[right].RelPath {
			return problems[left].Line < problems[right].Line
		}
		return compareSearchPaths(problems[left].RelPath, problems[right].RelPath)
	})

	message := "No workspace problems detected by lightweight scanners."
	if len(problems) > 0 {
		message = fmt.Sprintf("%d workspace problem(s) detected by lightweight scanners.", len(problems))
	}
	if truncated {
		message += " Results truncated."
	}
	return ProblemSummary{
		Problems:    problems,
		Message:     message,
		GeneratedAt: time.Now().UTC(),
		Truncated:   truncated,
	}, nil
}

func (s *Service) scanFileProblems(root string, relPath string) []WorkspaceProblem {
	preview, err := s.PreviewFile(root, relPath)
	if err != nil || strings.TrimSpace(preview.Text) == "" {
		return nil
	}
	problems := markerProblems(preview)
	if strings.EqualFold(filepath.Ext(preview.Name), ".json") && !json.Valid([]byte(preview.Text)) {
		problems = append(problems, jsonProblem(preview))
	}
	return problems
}

func markerProblems(preview domain.FilePreview) []WorkspaceProblem {
	lines := strings.Split(strings.ReplaceAll(preview.Text, "\r\n", "\n"), "\n")
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

func jsonProblem(preview domain.FilePreview) WorkspaceProblem {
	var value any
	err := json.Unmarshal([]byte(preview.Text), &value)
	line := 1
	message := "Invalid JSON"
	if err != nil {
		message = err.Error()
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			line = lineForOffset(preview.Text, syntaxErr.Offset)
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

func problemFromLine(preview domain.FilePreview, severity string, source string, message string, lineNumber int, line string) WorkspaceProblem {
	snippet := trimSearchSnippet(line)
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
