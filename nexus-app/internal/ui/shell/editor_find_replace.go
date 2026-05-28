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

func (v *View) activeEditorTabID() (string, bool) {
	item := v.editorTabs.Selected()
	if item == nil {
		return "", false
	}
	id := strings.TrimSpace(v.tabIDs[item])
	if id == "" {
		return "", false
	}
	tab, ok := v.editorSession.Tab(id)
	if !ok || tab.Kind != editorSvc.KindFile {
		return "", false
	}
	return id, true
}

func (v *View) activeTextEditor() (string, *textEditorBinding, bool) {
	id, ok := v.activeEditorTabID()
	if !ok {
		return "", nil, false
	}
	editor, ok := v.textEditor(id)
	if !ok {
		return "", nil, false
	}
	return id, editor, true
}

func (v *View) openFindReplaceDialog() {
	tabID, _, ok := v.activeTextEditor()
	if !ok {
		v.addActivity("Select a text editor tab to use Find / Replace.")
		return
	}
	findEntry := widget.NewEntry()
	findEntry.SetPlaceHolder("Find text")
	replaceEntry := widget.NewEntry()
	replaceEntry.SetPlaceHolder("Replace with")
	caseSensitive := widget.NewCheck("Case sensitive", nil)
	status := widget.NewLabel("Enter search text.")
	status.Wrapping = fyne.TextWrapWord
	findNext := func() {
		activeEditor, exists := v.textEditor(tabID)
		if !exists {
			status.SetText("Editor tab is no longer available.")
			return
		}
		query := strings.TrimSpace(findEntry.Text)
		if query == "" {
			status.SetText("Enter search text first.")
			return
		}
		text := activeEditor.source.Text
		start := editorCursorToOffset(text, activeEditor.source.CursorRow, activeEditor.source.CursorColumn)
		offset := editorFindNextOffset(text, query, start, caseSensitive.Checked)
		if offset < 0 && start > 0 {
			offset = editorFindNextOffset(text, query, 0, caseSensitive.Checked)
		}
		if offset < 0 {
			status.SetText("No matches found.")
			return
		}
		editorSetCursorOffset(activeEditor.source, text, offset+len(query))
		row, col := editorOffsetToCursor(text, offset)
		status.SetText(fmt.Sprintf("Match at line %d, column %d.", row+1, col+1))
	}
	replaceNext := func() {
		activeEditor, exists := v.textEditor(tabID)
		if !exists {
			status.SetText("Editor tab is no longer available.")
			return
		}
		query := strings.TrimSpace(findEntry.Text)
		if query == "" {
			status.SetText("Enter search text first.")
			return
		}
		text := activeEditor.source.Text
		start := editorCursorToOffset(text, activeEditor.source.CursorRow, activeEditor.source.CursorColumn)
		nextText, matchOffset, replaced := editorReplaceNext(text, query, replaceEntry.Text, start, caseSensitive.Checked)
		if !replaced && start > 0 {
			nextText, matchOffset, replaced = editorReplaceNext(text, query, replaceEntry.Text, 0, caseSensitive.Checked)
		}
		if !replaced {
			status.SetText("No matches found to replace.")
			return
		}
		activeEditor.source.SetText(nextText)
		editorSetCursorOffset(activeEditor.source, nextText, matchOffset+len(replaceEntry.Text))
		row, col := editorOffsetToCursor(nextText, matchOffset)
		status.SetText(fmt.Sprintf("Replaced match at line %d, column %d.", row+1, col+1))
	}
	replaceAll := func() {
		activeEditor, exists := v.textEditor(tabID)
		if !exists {
			status.SetText("Editor tab is no longer available.")
			return
		}
		query := strings.TrimSpace(findEntry.Text)
		if query == "" {
			status.SetText("Enter search text first.")
			return
		}
		nextText, count := editorReplaceAll(activeEditor.source.Text, query, replaceEntry.Text, caseSensitive.Checked)
		if count == 0 {
			status.SetText("No matches found to replace.")
			return
		}
		activeEditor.source.SetText(nextText)
		editorSetCursorOffset(activeEditor.source, nextText, 0)
		status.SetText(fmt.Sprintf("Replaced %d match(es).", count))
	}
	findEntry.OnSubmitted = func(string) { findNext() }
	replaceEntry.OnSubmitted = func(string) { replaceNext() }
	content := container.NewVBox(
		widget.NewForm(
			widget.NewFormItem("Find", findEntry),
			widget.NewFormItem("Replace", replaceEntry),
		),
		caseSensitive,
		status,
		container.NewHBox(
			widget.NewButton("Find next", findNext),
			widget.NewButton("Replace next", replaceNext),
			widget.NewButton("Replace all", replaceAll),
		),
	)
	pop := dialog.NewCustom("Find & Replace", "Close", content, v.window)
	pop.Resize(fyne.NewSize(560, 240))
	pop.Show()
}

func editorFindNextOffset(text string, query string, start int, caseSensitive bool) int {
	if query == "" {
		return -1
	}
	if start < 0 {
		start = 0
	}
	if start > len(text) {
		start = len(text)
	}
	if caseSensitive {
		index := strings.Index(text[start:], query)
		if index < 0 {
			return -1
		}
		return start + index
	}
	needle := strings.ToLower(query)
	haystack := strings.ToLower(text[start:])
	index := strings.Index(haystack, needle)
	if index < 0 {
		return -1
	}
	return start + index
}

func editorReplaceNext(text string, query string, replacement string, start int, caseSensitive bool) (string, int, bool) {
	index := editorFindNextOffset(text, query, start, caseSensitive)
	if index < 0 {
		return text, -1, false
	}
	return text[:index] + replacement + text[index+len(query):], index, true
}

func editorReplaceAll(text string, query string, replacement string, caseSensitive bool) (string, int) {
	if query == "" {
		return text, 0
	}
	if caseSensitive {
		count := strings.Count(text, query)
		if count == 0 {
			return text, 0
		}
		return strings.ReplaceAll(text, query, replacement), count
	}
	var builder strings.Builder
	current := text
	offset := 0
	replaced := 0
	for {
		index := editorFindNextOffset(current, query, offset, false)
		if index < 0 {
			break
		}
		builder.WriteString(current[:index])
		builder.WriteString(replacement)
		current = current[index+len(query):]
		offset = 0
		replaced++
	}
	builder.WriteString(current)
	if replaced == 0 {
		return text, 0
	}
	return builder.String(), replaced
}

func editorCursorToOffset(text string, row int, column int) int {
	if row < 0 {
		row = 0
	}
	if column < 0 {
		column = 0
	}
	currentRow := 0
	currentColumn := 0
	for index, r := range text {
		if currentRow == row && currentColumn == column {
			return index
		}
		if r == '\n' {
			if currentRow == row {
				return index
			}
			currentRow++
			currentColumn = 0
			continue
		}
		if currentRow == row {
			currentColumn++
		}
	}
	return len(text)
}

func editorOffsetToCursor(text string, offset int) (int, int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	row := 0
	column := 0
	for _, r := range text[:offset] {
		if r == '\n' {
			row++
			column = 0
			continue
		}
		column++
	}
	return row, column
}

func editorSetCursorOffset(entry *widget.Entry, text string, offset int) {
	row, column := editorOffsetToCursor(text, offset)
	entry.CursorRow = row
	entry.CursorColumn = column
	entry.Refresh()
}
