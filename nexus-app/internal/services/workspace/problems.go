package workspace

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"

	"nexusdesk/internal/domain"
)

const (
	defaultProblemMaxResults = 80
	problemPreviewMaxBytes   = 64 * 1024
)

var diagnosticLinePattern = regexp.MustCompile(`(?i)\bline\s+(\d+)`)

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
	problems = append(problems, syntaxProblems(preview)...)
	return problems
}

func syntaxProblems(preview domain.FilePreview) []WorkspaceProblem {
	ext := strings.ToLower(filepath.Ext(preview.Name))
	switch ext {
	case ".json":
		if !json.Valid([]byte(preview.Text)) {
			return []WorkspaceProblem{jsonProblem(preview)}
		}
	case ".go":
		return goProblems(preview)
	case ".yaml", ".yml":
		if problem, ok := yamlProblem(preview); ok {
			return []WorkspaceProblem{problem}
		}
	case ".toml":
		if problem, ok := tomlProblem(preview); ok {
			return []WorkspaceProblem{problem}
		}
	case ".xml":
		if problem, ok := xmlProblem(preview); ok {
			return []WorkspaceProblem{problem}
		}
	}
	return nil
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

func goProblems(preview domain.FilePreview) []WorkspaceProblem {
	fileSet := token.NewFileSet()
	_, err := parser.ParseFile(fileSet, preview.RelPath, preview.Text, parser.AllErrors)
	if err == nil {
		return nil
	}
	if list, ok := err.(scanner.ErrorList); ok {
		problems := make([]WorkspaceProblem, 0, minInt(len(list), 4))
		for index, item := range list {
			if index >= 4 {
				break
			}
			problems = append(problems, WorkspaceProblem{
				RelPath:  preview.RelPath,
				Name:     preview.Name,
				Severity: "error",
				Source:   "go",
				Message:  "Go syntax: " + item.Msg,
				Line:     maxInt(item.Pos.Line, 1),
			})
		}
		return problems
	}
	position := fileSet.Position(errPosition(fileSet, err))
	return []WorkspaceProblem{{
		RelPath:  preview.RelPath,
		Name:     preview.Name,
		Severity: "error",
		Source:   "go",
		Message:  "Go syntax: " + err.Error(),
		Line:     maxInt(position.Line, 1),
	}}
}

func yamlProblem(preview domain.FilePreview) (WorkspaceProblem, bool) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(preview.Text), &node); err != nil {
		return WorkspaceProblem{
			RelPath:  preview.RelPath,
			Name:     preview.Name,
			Severity: "error",
			Source:   "yaml",
			Message:  "YAML syntax: " + strings.TrimPrefix(err.Error(), "yaml: "),
			Line:     diagnosticLine(err.Error()),
		}, true
	}
	return WorkspaceProblem{}, false
}

func tomlProblem(preview domain.FilePreview) (WorkspaceProblem, bool) {
	if _, err := toml.Decode(preview.Text, &map[string]any{}); err != nil {
		line := diagnosticLine(err.Error())
		var parseErr toml.ParseError
		if errors.As(err, &parseErr) && parseErr.Position.Line > 0 {
			line = parseErr.Position.Line
		}
		return WorkspaceProblem{
			RelPath:  preview.RelPath,
			Name:     preview.Name,
			Severity: "error",
			Source:   "toml",
			Message:  "TOML syntax: " + err.Error(),
			Line:     line,
		}, true
	}
	return WorkspaceProblem{}, false
}

func xmlProblem(preview domain.FilePreview) (WorkspaceProblem, bool) {
	decoder := xml.NewDecoder(strings.NewReader(preview.Text))
	for {
		if _, err := decoder.Token(); err != nil {
			if errors.Is(err, io.EOF) {
				return WorkspaceProblem{}, false
			}
			line := 1
			if syntaxErr, ok := err.(*xml.SyntaxError); ok && syntaxErr.Line > 0 {
				line = syntaxErr.Line
			}
			return WorkspaceProblem{
				RelPath:  preview.RelPath,
				Name:     preview.Name,
				Severity: "error",
				Source:   "xml",
				Message:  "XML syntax: " + err.Error(),
				Line:     line,
			}, true
		}
	}
}

func problemFromLine(preview domain.FilePreview, severity string, source string, message string, lineNumber int, line string) WorkspaceProblem {
	snippet := trimSearchSnippet(line, 0)
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

func diagnosticLine(message string) int {
	match := diagnosticLinePattern.FindStringSubmatch(message)
	if len(match) < 2 {
		return 1
	}
	var line int
	if _, err := fmt.Sscanf(match[1], "%d", &line); err != nil || line <= 0 {
		return 1
	}
	return line
}

func errPosition(fileSet *token.FileSet, err error) token.Pos {
	if positioned, ok := err.(interface{ Pos() token.Pos }); ok {
		return positioned.Pos()
	}
	return token.NoPos
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
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
