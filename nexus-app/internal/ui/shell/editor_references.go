package shell

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	editorSvc "nexusdesk/internal/services/editor"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

const workspaceReferenceSearchLimit = 120

type editorReferenceCandidate struct {
	RelPath string
	Line    int
	Snippet string
}

func (v *View) openEditorReferencesDialog(tabID string) {
	tab, ok := v.editorSession.Tab(tabID)
	if !ok {
		v.addActivity("Editor tab is no longer available.")
		return
	}
	activeEditor, ok := v.textEditor(tabID)
	if !ok {
		v.addActivity("Select a text editor tab to find references.")
		return
	}
	query := editorSvc.SymbolAtCursor(activeEditor.source.Text, activeEditor.source.CursorRow, activeEditor.source.CursorColumn)
	if strings.TrimSpace(query) == "" {
		activeEditor.status.SetText("Place the cursor on a symbol name before using References.")
		return
	}
	candidates, err := v.editorReferenceCandidates(query)
	if err != nil {
		activeEditor.status.SetText("Reference lookup failed: " + err.Error())
		return
	}
	status := widget.NewLabel(editorReferencesStatus(query, len(candidates)))
	status.Wrapping = fyne.TextWrapWord
	list := container.NewVBox()
	var pop dialog.Dialog
	openCandidate := func(candidate editorReferenceCandidate) {
		workspace := v.state.Workspace()
		if workspace.Root == "" {
			return
		}
		preview, err := v.workspaceService.PreviewFile(workspace.Root, candidate.RelPath)
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		v.openPreviewTab(preview)
		if latestEditor, exists := v.textEditor(v.editorSession.ActiveID()); exists {
			editorSetCursorLine(latestEditor.source, candidate.Line)
			latestEditor.status.SetText(fmt.Sprintf("Moved to reference for %s in %s on line %d.", query, candidate.RelPath, candidate.Line))
		}
		if pop != nil {
			pop.Hide()
		}
	}
	if len(candidates) == 0 {
		list.Add(widget.NewLabel("No references found in previewable workspace files."))
	} else {
		for _, candidate := range candidates {
			current := candidate
			button := widget.NewButton(editorReferenceLabel(current), func() {
				openCandidate(current)
			})
			button.Alignment = widget.ButtonAlignLeading
			button.Importance = widget.LowImportance
			list.Add(button)
		}
	}
	content := container.NewBorder(
		container.NewVBox(widget.NewLabel(tab.RelPath), status),
		nil,
		nil,
		nil,
		container.NewVScroll(list),
	)
	pop = dialog.NewCustom("Find References", "Close", content, v.window)
	pop.Resize(fyne.NewSize(680, 460))
	pop.Show()
}

func (v *View) editorReferenceCandidates(query string) ([]editorReferenceCandidate, error) {
	workspace := v.state.Workspace()
	if strings.TrimSpace(workspace.Root) == "" {
		return nil, nil
	}
	results, err := v.workspaceService.Search(workspace.Root, query, workspaceSvc.SearchOptions{MaxResults: workspaceReferenceSearchLimit})
	if err != nil {
		return nil, err
	}
	return editorReferenceCandidatesFromSearch(query, results), nil
}

func editorReferenceCandidatesFromSearch(query string, results []workspaceSvc.SearchResult) []editorReferenceCandidate {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil
	}
	candidates := make([]editorReferenceCandidate, 0, len(results))
	seen := map[string]bool{}
	for _, result := range results {
		if result.Kind == "directory" || result.Line <= 0 || !strings.HasPrefix(result.MatchType, "content") {
			continue
		}
		if !referenceSnippetMatches(query, result.Snippet) {
			continue
		}
		key := fmt.Sprintf("%s:%d:%s", result.RelPath, result.Line, result.Snippet)
		if seen[key] {
			continue
		}
		seen[key] = true
		candidates = append(candidates, editorReferenceCandidate{
			RelPath: result.RelPath,
			Line:    result.Line,
			Snippet: result.Snippet,
		})
	}
	return candidates
}

func referenceSnippetMatches(query string, snippet string) bool {
	return strings.Contains(strings.ToLower(snippet), strings.ToLower(strings.TrimSpace(query)))
}

func editorReferenceLabel(candidate editorReferenceCandidate) string {
	location := candidate.RelPath
	if candidate.Line > 0 {
		location = fmt.Sprintf("%s:%d", candidate.RelPath, candidate.Line)
	}
	snippet := strings.TrimSpace(candidate.Snippet)
	if snippet == "" {
		return location
	}
	return location + "  " + snippet
}

func editorReferencesStatus(query string, count int) string {
	if count == 0 {
		return fmt.Sprintf("No references found for %s.", query)
	}
	return fmt.Sprintf("%d reference(s) for %s. Select one to open and jump.", count, query)
}
