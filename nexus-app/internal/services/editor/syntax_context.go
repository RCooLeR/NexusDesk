package editor

import (
	"fmt"
	"strings"
)

type SyntaxCursorContext struct {
	Line       int
	Column     int
	LineText   string
	Symbol     string
	Token      SyntaxToken
	LineTokens []SyntaxToken
	Message    string
}

func SyntaxContextAtCursor(fileName string, content string, cursorRow int, cursorColumn int) SyntaxCursorContext {
	analysis := AnalyzeSyntax(fileName, content)
	return SyntaxContextFromAnalysis(content, analysis, cursorRow, cursorColumn)
}

func SyntaxContextFromAnalysis(content string, analysis SyntaxAnalysis, cursorRow int, cursorColumn int) SyntaxCursorContext {
	if cursorRow < 0 {
		cursorRow = 0
	}
	if cursorColumn < 0 {
		cursorColumn = 0
	}
	lines := strings.Split(strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n"), "\n")
	lineNumber := cursorRow + 1
	lineText := ""
	if cursorRow < len(lines) {
		lineText = lines[cursorRow]
	}
	context := SyntaxCursorContext{
		Line:       lineNumber,
		Column:     cursorColumn + 1,
		LineText:   lineText,
		Symbol:     SymbolAtCursor(content, cursorRow, cursorColumn),
		LineTokens: syntaxTokensOnLine(analysis.Tokens, lineNumber),
	}
	context.Token = syntaxTokenAtColumn(context.LineTokens, cursorColumn)
	context.Message = syntaxCursorMessage(context, analysis)
	return context
}

func syntaxTokensOnLine(tokens []SyntaxToken, line int) []SyntaxToken {
	out := make([]SyntaxToken, 0)
	for _, token := range tokens {
		if token.Line == line {
			out = append(out, token)
		}
	}
	return out
}

func syntaxTokenAtColumn(tokens []SyntaxToken, column int) SyntaxToken {
	for _, token := range tokens {
		if column >= token.StartColumn && column < token.EndColumn {
			return token
		}
		if column == token.EndColumn && token.EndColumn > token.StartColumn {
			return token
		}
	}
	return SyntaxToken{}
}

func syntaxCursorMessage(context SyntaxCursorContext, analysis SyntaxAnalysis) string {
	language := strings.TrimSpace(analysis.Language.Label)
	if language == "" {
		language = "Plain text"
	}
	parts := []string{fmt.Sprintf("Cursor: L%d:C%d", context.Line, context.Column), "language " + language}
	if strings.TrimSpace(context.Token.Kind) != "" {
		parts = append(parts, fmt.Sprintf("%s token %q", context.Token.Kind, compactSyntaxContextText(context.Token.Text, 48)))
	} else if len(context.LineTokens) > 0 {
		parts = append(parts, fmt.Sprintf("%d token(s) on this line", len(context.LineTokens)))
	} else {
		parts = append(parts, "no token under cursor")
	}
	if strings.TrimSpace(context.Symbol) != "" {
		parts = append(parts, "symbol "+context.Symbol)
	}
	return strings.Join(parts, "; ") + "."
}

func compactSyntaxContextText(value string, limit int) string {
	value = strings.Join(strings.Fields(value), " ")
	if limit <= 3 || len(value) <= limit {
		return value
	}
	return value[:limit-3] + "..."
}
