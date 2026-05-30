package shell

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	editorSvc "nexusdesk/internal/services/editor"
	nexusTheme "nexusdesk/internal/ui/theme"
)

var syntaxPalette = nexusTheme.JetBrainsDarkPalette()

var syntaxHighlightStyles = map[string]widget.TextGridStyle{
	"active-line": &widget.CustomTextGridStyle{
		BGColor: syntaxPalette.SyntaxActiveLine,
	},
	"comment": &widget.CustomTextGridStyle{
		TextStyle: fyne.TextStyle{Italic: true},
		FGColor:   syntaxPalette.SyntaxComment,
	},
	"keyword": &widget.CustomTextGridStyle{
		TextStyle: fyne.TextStyle{Bold: true},
		FGColor:   syntaxPalette.SyntaxKeyword,
	},
	"number": &widget.CustomTextGridStyle{
		FGColor: syntaxPalette.SyntaxNumber,
	},
	"string": &widget.CustomTextGridStyle{
		FGColor: syntaxPalette.SyntaxString,
	},
}

func newSyntaxHighlightGrid(relPath string, text string) *widget.TextGrid {
	grid := widget.NewTextGrid()
	grid.ShowLineNumbers = true
	grid.Scroll = fyne.ScrollBoth
	analysis := editorSvc.AnalyzeSyntax(relPath, text)
	applySyntaxHighlightGrid(grid, text, analysis)
	return grid
}

func applySyntaxHighlightGrid(grid *widget.TextGrid, text string, analysis editorSvc.SyntaxAnalysis) {
	applySyntaxHighlightGridWithCursor(grid, text, analysis, -1)
}

func applySyntaxHighlightGridWithCursor(grid *widget.TextGrid, text string, analysis editorSvc.SyntaxAnalysis, cursorRow int) {
	if grid == nil {
		return
	}
	grid.SetText(normalizeSyntaxHighlightText(text))
	if cursorRow >= 0 && cursorRow < len(grid.Rows) {
		grid.SetRowStyle(cursorRow, syntaxStyleForKind("active-line"))
	}
	for _, token := range analysis.Tokens {
		style := syntaxStyleForKind(token.Kind)
		if style == nil || token.Line <= 0 || token.StartColumn < 0 || token.EndColumn <= token.StartColumn {
			continue
		}
		grid.SetStyleRange(token.Line-1, token.StartColumn, token.Line-1, token.EndColumn-1, style)
	}
	grid.Refresh()
}

func normalizeSyntaxHighlightText(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

func syntaxStyleForKind(kind string) widget.TextGridStyle {
	return syntaxHighlightStyles[strings.ToLower(strings.TrimSpace(kind))]
}
