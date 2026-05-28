package editor

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"go/parser"
	"go/scanner"
	"go/token"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

const (
	draftDiagnosticMaxMarkerLines = 4000
	draftDiagnosticMaxMarkers     = 8
	draftDiagnosticMaxGoErrors    = 4
)

var draftDiagnosticLinePattern = regexp.MustCompile(`(?i)\bline\s+(\d+)`)

type DraftDiagnostic struct {
	RelPath  string
	Severity string
	Source   string
	Message  string
	Line     int
}

func AnalyzeDraftDiagnostics(fileName string, content string) []DraftDiagnostic {
	fileName = normalizeDefinitionRelPath(fileName)
	content = strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n")
	diagnostics := append([]DraftDiagnostic{}, draftMarkerDiagnostics(fileName, content)...)
	diagnostics = append(diagnostics, draftSyntaxDiagnostics(fileName, content)...)
	sort.SliceStable(diagnostics, func(left int, right int) bool {
		if diagnostics[left].Severity != diagnostics[right].Severity {
			return draftDiagnosticSeverityRank(diagnostics[left].Severity) < draftDiagnosticSeverityRank(diagnostics[right].Severity)
		}
		if diagnostics[left].Line == diagnostics[right].Line {
			return diagnostics[left].Source < diagnostics[right].Source
		}
		return diagnostics[left].Line < diagnostics[right].Line
	})
	return diagnostics
}

func draftMarkerDiagnostics(fileName string, content string) []DraftDiagnostic {
	lines := strings.Split(content, "\n")
	diagnostics := []DraftDiagnostic{}
	for index, line := range lines {
		if index >= draftDiagnosticMaxMarkerLines || len(diagnostics) >= draftDiagnosticMaxMarkers {
			break
		}
		upper := strings.ToUpper(line)
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "<<<<<<<") || strings.HasPrefix(trimmed, "=======") || strings.HasPrefix(trimmed, ">>>>>>>"):
			diagnostics = append(diagnostics, draftDiagnosticFromLine(fileName, "error", "merge-conflict", "Merge conflict marker", index+1, line))
		case strings.Contains(upper, "FIXME") || strings.Contains(upper, "BUG"):
			diagnostics = append(diagnostics, draftDiagnosticFromLine(fileName, "warning", "marker", "Fix marker", index+1, line))
		case strings.Contains(upper, "TODO") || strings.Contains(upper, "HACK"):
			diagnostics = append(diagnostics, draftDiagnosticFromLine(fileName, "info", "marker", "Task marker", index+1, line))
		}
	}
	return diagnostics
}

func draftSyntaxDiagnostics(fileName string, content string) []DraftDiagnostic {
	extension := strings.ToLower(filepath.Ext(fileName))
	switch extension {
	case ".json", ".code-workspace":
		if strings.TrimSpace(content) != "" && !json.Valid([]byte(content)) {
			return []DraftDiagnostic{draftJSONDiagnostic(fileName, content)}
		}
	case ".go":
		return draftGoDiagnostics(fileName, content)
	case ".yaml", ".yml":
		if diagnostic, ok := draftYAMLDiagnostic(fileName, content); ok {
			return []DraftDiagnostic{diagnostic}
		}
	case ".toml":
		if diagnostic, ok := draftTOMLDiagnostic(fileName, content); ok {
			return []DraftDiagnostic{diagnostic}
		}
	case ".xml":
		if diagnostic, ok := draftXMLDiagnostic(fileName, content); ok {
			return []DraftDiagnostic{diagnostic}
		}
	}
	return nil
}

func draftJSONDiagnostic(fileName string, content string) DraftDiagnostic {
	var value any
	err := json.Unmarshal([]byte(content), &value)
	line := 1
	message := "Invalid JSON"
	if err != nil {
		message = err.Error()
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			line = draftLineForOffset(content, syntaxErr.Offset)
		}
	}
	return DraftDiagnostic{RelPath: fileName, Severity: "error", Source: "json", Message: message, Line: line}
}

func draftGoDiagnostics(fileName string, content string) []DraftDiagnostic {
	fileSet := token.NewFileSet()
	_, err := parser.ParseFile(fileSet, fileName, content, parser.AllErrors)
	if err == nil {
		return nil
	}
	if list, ok := err.(scanner.ErrorList); ok {
		diagnostics := make([]DraftDiagnostic, 0, minEditorInt(len(list), draftDiagnosticMaxGoErrors))
		for index, item := range list {
			if index >= draftDiagnosticMaxGoErrors {
				break
			}
			diagnostics = append(diagnostics, DraftDiagnostic{
				RelPath:  fileName,
				Severity: "error",
				Source:   "go",
				Message:  "Go syntax: " + item.Msg,
				Line:     maxEditorInt(item.Pos.Line, 1),
			})
		}
		return diagnostics
	}
	position := fileSet.Position(draftErrPosition(fileSet, err))
	return []DraftDiagnostic{{RelPath: fileName, Severity: "error", Source: "go", Message: "Go syntax: " + err.Error(), Line: maxEditorInt(position.Line, 1)}}
}

func draftYAMLDiagnostic(fileName string, content string) (DraftDiagnostic, bool) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		return DraftDiagnostic{
			RelPath:  fileName,
			Severity: "error",
			Source:   "yaml",
			Message:  "YAML syntax: " + strings.TrimPrefix(err.Error(), "yaml: "),
			Line:     draftDiagnosticLine(err.Error()),
		}, true
	}
	return DraftDiagnostic{}, false
}

func draftTOMLDiagnostic(fileName string, content string) (DraftDiagnostic, bool) {
	if _, err := toml.Decode(content, &map[string]any{}); err != nil {
		line := draftDiagnosticLine(err.Error())
		var parseErr toml.ParseError
		if errors.As(err, &parseErr) && parseErr.Position.Line > 0 {
			line = parseErr.Position.Line
		}
		return DraftDiagnostic{RelPath: fileName, Severity: "error", Source: "toml", Message: "TOML syntax: " + err.Error(), Line: line}, true
	}
	return DraftDiagnostic{}, false
}

func draftXMLDiagnostic(fileName string, content string) (DraftDiagnostic, bool) {
	decoder := xml.NewDecoder(strings.NewReader(content))
	for {
		if _, err := decoder.Token(); err != nil {
			if errors.Is(err, io.EOF) {
				return DraftDiagnostic{}, false
			}
			line := 1
			if syntaxErr, ok := err.(*xml.SyntaxError); ok && syntaxErr.Line > 0 {
				line = syntaxErr.Line
			}
			return DraftDiagnostic{RelPath: fileName, Severity: "error", Source: "xml", Message: "XML syntax: " + err.Error(), Line: line}, true
		}
	}
}

func draftDiagnosticFromLine(fileName string, severity string, source string, message string, lineNumber int, line string) DraftDiagnostic {
	snippet := compactSyntaxContextText(line, 120)
	if snippet != "" {
		message += ": " + snippet
	}
	return DraftDiagnostic{RelPath: fileName, Severity: severity, Source: source, Message: message, Line: lineNumber}
}

func draftLineForOffset(content string, offset int64) int {
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

func draftDiagnosticLine(message string) int {
	match := draftDiagnosticLinePattern.FindStringSubmatch(message)
	if len(match) < 2 {
		return 1
	}
	var line int
	if _, err := fmt.Sscanf(match[1], "%d", &line); err != nil || line <= 0 {
		return 1
	}
	return line
}

func draftErrPosition(fileSet *token.FileSet, err error) token.Pos {
	if positioned, ok := err.(interface{ Pos() token.Pos }); ok {
		return positioned.Pos()
	}
	return token.NoPos
}

func draftDiagnosticSeverityRank(severity string) int {
	switch severity {
	case "error":
		return 0
	case "warning":
		return 1
	default:
		return 2
	}
}

func minEditorInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxEditorInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
