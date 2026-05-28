package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	editorSvc "nexusdesk/internal/services/editor"
)

const editorSymbolPickerLimit = 80

type editorSymbolCandidate struct {
	Label      string
	SearchText string
	Line       int
}

func (v *View) openEditorSymbolDialog(tabID string) {
	tab, ok := v.editorSession.Tab(tabID)
	if !ok {
		v.addActivity("Editor tab is no longer available.")
		return
	}
	activeEditor, ok := v.textEditor(tabID)
	if !ok {
		v.addActivity("Select a text editor tab to use Symbols.")
		return
	}
	candidates := editorSymbolCandidates(tab.RelPath, activeEditor.source.Text)
	if len(candidates) == 0 {
		v.addActivity("No symbols detected for " + tab.RelPath + ".")
		return
	}

	query := widget.NewEntry()
	query.SetPlaceHolder("Filter symbols")
	status := widget.NewLabel(editorSymbolStatus(len(candidates), ""))
	status.Wrapping = fyne.TextWrapWord
	list := container.NewVBox()
	var pop dialog.Dialog

	render := func(filter string) {
		items := filterEditorSymbolCandidates(candidates, filter, editorSymbolPickerLimit)
		list.Objects = list.Objects[:0]
		status.SetText(editorSymbolStatus(len(items), filter))
		if len(items) == 0 {
			list.Add(widget.NewLabel("No matching symbols."))
			list.Refresh()
			return
		}
		for _, item := range items {
			current := item
			button := widget.NewButton(current.Label, func() {
				if latestEditor, exists := v.textEditor(tabID); exists {
					editorSetCursorLine(latestEditor.source, current.Line)
					if latestEditor.outlineStatus != nil {
						latestEditor.outlineStatus.SetText(fmt.Sprintf("Moved cursor to %s.", current.Label))
					}
				}
				if pop != nil {
					pop.Hide()
				}
			})
			button.Alignment = widget.ButtonAlignLeading
			button.Importance = widget.LowImportance
			list.Add(button)
		}
		list.Refresh()
	}

	query.OnChanged = render
	query.OnSubmitted = func(value string) {
		items := filterEditorSymbolCandidates(candidates, value, 1)
		if len(items) == 0 {
			status.SetText("No matching symbols.")
			return
		}
		if latestEditor, exists := v.textEditor(tabID); exists {
			editorSetCursorLine(latestEditor.source, items[0].Line)
			if latestEditor.outlineStatus != nil {
				latestEditor.outlineStatus.SetText(fmt.Sprintf("Moved cursor to %s.", items[0].Label))
			}
		}
		if pop != nil {
			pop.Hide()
		}
	}
	render("")
	content := container.NewBorder(
		container.NewVBox(widget.NewLabel(tab.RelPath), query, status),
		nil,
		nil,
		nil,
		container.NewVScroll(list),
	)
	pop = dialog.NewCustom("Go to Symbol", "Close", content, v.window)
	pop.Resize(fyne.NewSize(600, 440))
	pop.Show()
}

func editorSymbolCandidates(relPath string, content string) []editorSymbolCandidate {
	outline := editorSvc.BuildOutline(relPath, content)
	items := make([]editorSymbolCandidate, 0, len(outline))
	for _, item := range outline {
		label := outlineItemText(item)
		items = append(items, editorSymbolCandidate{
			Label:      label,
			SearchText: strings.ToLower(strings.Join([]string{item.Kind, item.Label, fmt.Sprintf("line %d", item.Line)}, " ")),
			Line:       item.Line,
		})
	}
	return items
}

func filterEditorSymbolCandidates(items []editorSymbolCandidate, query string, limit int) []editorSymbolCandidate {
	if limit <= 0 {
		limit = len(items)
	}
	terms := strings.Fields(strings.ToLower(strings.TrimSpace(query)))
	filtered := make([]editorSymbolCandidate, 0, min(len(items), limit))
	for _, item := range items {
		if editorSymbolMatches(item.SearchText, terms) {
			filtered = append(filtered, item)
			if len(filtered) >= limit {
				break
			}
		}
	}
	return filtered
}

func editorSymbolMatches(searchText string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(searchText, term) {
			return false
		}
	}
	return true
}

func editorSymbolStatus(count int, query string) string {
	if strings.TrimSpace(query) == "" {
		return fmt.Sprintf("%d symbol(s). Select one or type to filter.", count)
	}
	return fmt.Sprintf("%d matching symbol(s). Press Enter to jump to the first match.", count)
}
