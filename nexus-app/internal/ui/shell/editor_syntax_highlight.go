package shell

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	editorSvc "nexusdesk/internal/services/editor"
)

var syntaxHighlightStyles = map[string]widget.TextGridStyle{
	"comment": &widget.CustomTextGridStyle{
		TextStyle: fyne.TextStyle{Italic: true},
		FGColor:   color.NRGBA{R: 0x64, G: 0x74, B: 0x8b, A: 0xff},
	},
	"keyword": &widget.CustomTextGridStyle{
		TextStyle: fyne.TextStyle{Bold: true},
		FGColor:   color.NRGBA{R: 0x1d, G: 0x4e, B: 0xd8, A: 0xff},
	},
	"number": &widget.CustomTextGridStyle{
		FGColor: color.NRGBA{R: 0xb4, G: 0x53, B: 0x09, A: 0xff},
	},
	"string": &widget.CustomTextGridStyle{
		FGColor: color.NRGBA{R: 0x15, G: 0x80, B: 0x3d, A: 0xff},
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
	if grid == nil {
		return
	}
	grid.SetText(normalizeSyntaxHighlightText(text))
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
