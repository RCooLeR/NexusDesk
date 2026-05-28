package editor

import (
	"path/filepath"
	"regexp"
	"strings"
)

const (
	syntaxMaxLines  = 4000
	syntaxMaxTokens = 500
)

var syntaxTokenPattern = regexp.MustCompile(`(//[^\n]*|--[^\n]*|#[^\n]*|"(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|` + "`" + `(?:\\.|[^` + "`" + `\\])*` + "`" + `|\b\d+(?:\.\d+)?\b|\b[A-Za-z_][A-Za-z0-9_]*\b)`)

type SyntaxLanguage struct {
	ID          string
	Label       string
	NativeLight bool
	FutureLSP   bool
}

type SyntaxToken struct {
	Text string
	Kind string
	Line int
}

type SyntaxAnalysis struct {
	Language  SyntaxLanguage
	Tokens    []SyntaxToken
	Counts    map[string]int
	LineCount int
	Truncated bool
}

func AnalyzeSyntax(fileName string, content string) SyntaxAnalysis {
	language := DetectSyntaxLanguage(fileName)
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	keywords := syntaxKeywords(language.ID)
	analysis := SyntaxAnalysis{
		Language:  language,
		Tokens:    []SyntaxToken{},
		Counts:    map[string]int{},
		LineCount: len(lines),
	}
	for index, line := range lines {
		if index >= syntaxMaxLines || len(analysis.Tokens) >= syntaxMaxTokens {
			analysis.Truncated = true
			break
		}
		for _, match := range syntaxTokenPattern.FindAllString(line, -1) {
			kind := classifySyntaxToken(match, keywords)
			if kind == "plain" {
				continue
			}
			analysis.Tokens = append(analysis.Tokens, SyntaxToken{Text: match, Kind: kind, Line: index + 1})
			analysis.Counts[kind]++
			if len(analysis.Tokens) >= syntaxMaxTokens {
				analysis.Truncated = true
				break
			}
		}
	}
	return analysis
}

func DetectSyntaxLanguage(fileName string) SyntaxLanguage {
	lowerName := strings.ToLower(filepath.ToSlash(fileName))
	base := filepath.Base(lowerName)
	ext := strings.TrimPrefix(filepath.Ext(base), ".")
	switch {
	case base == "dockerfile" || strings.HasPrefix(base, "dockerfile."):
		return syntaxLanguage("dockerfile", "Dockerfile", true, false)
	case strings.HasSuffix(lowerName, ".code-workspace"):
		return syntaxLanguage("json", "JSON", true, false)
	}
	switch ext {
	case "go":
		return syntaxLanguage("go", "Go", true, true)
	case "js", "jsx", "mjs", "cjs":
		return syntaxLanguage("javascript", "JavaScript", true, true)
	case "ts", "tsx":
		return syntaxLanguage("typescript", "TypeScript", true, true)
	case "json", "jsonc":
		return syntaxLanguage("json", "JSON", true, false)
	case "md", "markdown", "mdx":
		return syntaxLanguage("markdown", "Markdown", true, false)
	case "sql":
		return syntaxLanguage("sql", "SQL", true, false)
	case "yaml", "yml":
		return syntaxLanguage("yaml", "YAML", true, false)
	case "xml", "svg", "html", "htm":
		return syntaxLanguage("markup", "Markup", true, false)
	case "css", "scss", "less":
		return syntaxLanguage("css", "CSS", true, false)
	case "py":
		return syntaxLanguage("python", "Python", true, true)
	case "rs":
		return syntaxLanguage("rust", "Rust", true, true)
	case "java":
		return syntaxLanguage("java", "Java", true, true)
	case "cs":
		return syntaxLanguage("csharp", "C#", true, true)
	case "ps1":
		return syntaxLanguage("powershell", "PowerShell", true, false)
	case "env", "log", "txt", "ini", "toml":
		return syntaxLanguage(ext, strings.ToUpper(ext), false, false)
	default:
		return syntaxLanguage("plaintext", "Plain text", false, false)
	}
}

func syntaxLanguage(id string, label string, nativeLight bool, futureLSP bool) SyntaxLanguage {
	return SyntaxLanguage{ID: id, Label: label, NativeLight: nativeLight, FutureLSP: futureLSP}
}

func classifySyntaxToken(text string, keywords map[string]bool) string {
	lower := strings.ToLower(text)
	switch {
	case strings.HasPrefix(text, "//"), strings.HasPrefix(text, "#"), strings.HasPrefix(text, "--"):
		return "comment"
	case strings.HasPrefix(text, `"`), strings.HasPrefix(text, `'`), strings.HasPrefix(text, "`"):
		return "string"
	case text[0] >= '0' && text[0] <= '9':
		return "number"
	case keywords[text] || keywords[lower]:
		return "keyword"
	default:
		return "plain"
	}
}

func syntaxKeywords(languageID string) map[string]bool {
	keywords := map[string][]string{
		"go":         {"break", "case", "chan", "const", "continue", "defer", "else", "fallthrough", "for", "func", "go", "if", "import", "interface", "map", "package", "range", "return", "select", "struct", "switch", "type", "var"},
		"javascript": {"async", "await", "break", "case", "catch", "class", "const", "continue", "default", "else", "export", "extends", "finally", "for", "from", "function", "if", "import", "let", "new", "return", "switch", "throw", "try", "var", "while"},
		"typescript": {"async", "await", "break", "case", "catch", "class", "const", "continue", "default", "else", "enum", "export", "extends", "finally", "for", "from", "function", "if", "implements", "import", "interface", "let", "new", "return", "switch", "throw", "try", "type", "var", "while"},
		"json":       {"false", "null", "true"},
		"sql":        {"and", "as", "by", "create", "delete", "from", "group", "insert", "into", "join", "left", "limit", "not", "null", "on", "or", "order", "right", "select", "table", "update", "values", "where"},
		"python":     {"and", "as", "async", "await", "break", "class", "continue", "def", "elif", "else", "except", "false", "finally", "for", "from", "if", "import", "in", "is", "lambda", "none", "not", "or", "pass", "return", "true", "try", "while", "with", "yield"},
		"rust":       {"as", "async", "await", "break", "const", "continue", "crate", "else", "enum", "fn", "for", "if", "impl", "let", "loop", "match", "mod", "move", "mut", "pub", "ref", "return", "self", "static", "struct", "trait", "type", "use", "where", "while"},
		"java":       {"abstract", "boolean", "break", "case", "catch", "class", "continue", "else", "enum", "extends", "final", "finally", "for", "if", "implements", "import", "interface", "new", "null", "package", "private", "protected", "public", "return", "static", "switch", "this", "throw", "try", "void", "while"},
		"csharp":     {"abstract", "as", "base", "bool", "break", "case", "catch", "class", "const", "continue", "else", "enum", "event", "false", "finally", "for", "if", "interface", "namespace", "new", "null", "private", "protected", "public", "return", "static", "string", "switch", "this", "throw", "true", "try", "using", "void", "while"},
		"dockerfile": {"add", "arg", "cmd", "copy", "entrypoint", "env", "expose", "from", "healthcheck", "label", "run", "user", "volume", "workdir"},
		"powershell": {"begin", "break", "catch", "class", "continue", "data", "do", "dynamicparam", "else", "elseif", "end", "filter", "finally", "for", "foreach", "from", "function", "if", "in", "param", "process", "return", "switch", "throw", "trap", "try", "until", "using", "while"},
	}
	values := keywords[languageID]
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}
