package shell

import (
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

const workspaceDefinitionSearchLimit = 60

type textEditorBinding struct {
	source               *widget.Entry
	status               *widget.Label
	tabState             *widget.Label
	saveButton           *widget.Button
	rendered             *previewPane
	outlineStatus        *widget.Label
	outlineList          *fyne.Container
	mapStatus            *widget.Label
	mapList              *fyne.Container
	diagnosticStatus     *widget.Label
	diagnosticList       *fyne.Container
	syntaxStatus         *widget.Label
	syntaxPreview        *widget.Label
	languageActions      *widget.Label
	syntaxGrid           *widget.TextGrid
	syntaxText           string
	syntaxAnalysis       editorSvc.SyntaxAnalysis
	relPath              string
	sourceEncoding       string
	saveEncoding         string
	encodingExplicit     bool
	initializingEncoding bool
	saving               bool
	onEncoding           func()
	onState              func(editorSvc.Tab, bool, bool)
}

func (b *textEditorBinding) applyTabState(tab editorSvc.Tab) {
	if b == nil {
		return
	}
	b.status.SetText(draftStatusText(tab))
	b.rendered.SetText(tab.DraftText)
	b.setOutline(tab.DraftText)
	b.setDocumentMap(tab.DraftText)
	b.setSyntax(tab.DraftText)
	if b.onState != nil {
		b.onState(tab, b.encodingDirty(), b.encodingExplicit)
	}
}

func (v *View) newTextEditor(tab editorSvc.Tab, preview domain.FilePreview, onState func(editorSvc.Tab, bool, bool)) fyne.CanvasObject {
	source := widget.NewMultiLineEntry()
	source.SetText(tab.DraftText)
	source.Wrapping = fyne.TextWrapOff
	source.TextStyle = fyne.TextStyle{Monospace: true}
	status := widget.NewLabel(draftStatusText(tab))
	rendered := newPreviewPane(preview, tab.DraftText)
	initialEncoding := editorWriteEncoding(preview.Encoding)
	outlineStatus := widget.NewLabel("")
	outlineStatus.Wrapping = fyne.TextWrapWord
	outlineList := container.NewVBox()
	mapStatus := widget.NewLabel("")
	mapStatus.Wrapping = fyne.TextWrapWord
	mapList := container.NewVBox()
	diagnosticStatus := widget.NewLabel("")
	diagnosticStatus.Wrapping = fyne.TextWrapWord
	diagnosticList := container.NewVBox()
	syntaxStatus := widget.NewLabel("")
	syntaxStatus.Wrapping = fyne.TextWrapWord
	syntaxPreview := widget.NewLabel("")
	syntaxPreview.Wrapping = fyne.TextWrapWord
	syntaxPreview.TextStyle = fyne.TextStyle{Monospace: true}
	languageActions := widget.NewLabel("")
	languageActions.Wrapping = fyne.TextWrapWord
	syntaxGrid := newSyntaxHighlightGrid(tab.RelPath, tab.DraftText)
	binding := &textEditorBinding{
		source:           source,
		status:           status,
		rendered:         rendered,
		outlineStatus:    outlineStatus,
		outlineList:      outlineList,
		mapStatus:        mapStatus,
		mapList:          mapList,
		diagnosticStatus: diagnosticStatus,
		diagnosticList:   diagnosticList,
		syntaxStatus:     syntaxStatus,
		syntaxPreview:    syntaxPreview,
		languageActions:  languageActions,
		syntaxGrid:       syntaxGrid,
		relPath:          tab.RelPath,
		sourceEncoding:   initialEncoding,
		saveEncoding:     initialEncoding,
		encodingExplicit: !preview.EncodingAmbiguous,
		onState:          onState,
	}
	encodingSelect := widget.NewSelect(editorEncodingOptions(), func(value string) {
		binding.saveEncoding = editorWriteEncoding(value)
		if !binding.initializingEncoding {
			binding.encodingExplicit = true
		}
		status.SetText(draftStatusTextWithEncoding(tab, binding.encodingDirty(), !binding.encodingExplicit))
		if binding.onEncoding != nil {
			binding.onEncoding()
		}
	})
	binding.initializingEncoding = true
	encodingSelect.SetSelected(initialEncoding)
	binding.initializingEncoding = false
	binding.onEncoding = func() {
		if next, ok := v.editorSession.Tab(tab.ID); ok {
			onState(next, binding.encodingDirty(), binding.encodingExplicit)
		}
	}
	if preview.Truncated {
		source.Disable()
		encodingSelect.Disable()
		status.SetText("Read-only partial preview. Inline editing is disabled to protect the full file.")
	} else if preview.EncodingAmbiguous {
		status.SetText(ambiguousEncodingStatusText())
	}
	binding.setOutline(tab.DraftText)
	binding.setDocumentMap(tab.DraftText)
	binding.setDiagnostics(tab.DraftText)
	binding.setSyntax(tab.DraftText)
	source.OnChanged = func(text string) {
		if !v.editorSession.UpdateDraft(tab.ID, text) {
			return
		}
		if next, ok := v.editorSession.Tab(tab.ID); ok {
			status.SetText(draftStatusTextWithEncoding(next, binding.encodingDirty(), !binding.encodingExplicit))
			rendered.SetText(next.DraftText)
			binding.setOutline(next.DraftText)
			binding.setDocumentMap(next.DraftText)
			binding.setDiagnostics(next.DraftText)
			binding.setSyntax(next.DraftText)
			onState(next, binding.encodingDirty(), binding.encodingExplicit)
		}
	}
	source.OnCursorChanged = func() {
		binding.updateSyntaxCursor()
	}
	revert := widget.NewButtonWithIcon("Revert draft", theme.ContentUndoIcon(), func() {
		v.revertEditorDraft(tab.ID)
	})
	revert.Importance = widget.LowImportance
	format := widget.NewButtonWithIcon("Format", theme.DocumentCreateIcon(), func() {
		result, err := editorSvc.FormatDocument(tab.RelPath, source.Text)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		if result.Changed {
			source.SetText(result.Content)
		}
		status.SetText(result.Message)
	})
	format.Importance = widget.LowImportance
	if preview.Truncated {
		format.Disable()
	}
	symbols := widget.NewButtonWithIcon("Symbols", theme.SearchIcon(), func() {
		v.openEditorSymbolDialog(tab.ID)
	})
	symbols.Importance = widget.LowImportance
	references := widget.NewButtonWithIcon("References", theme.SearchIcon(), func() {
		v.openEditorReferencesDialog(tab.ID)
	})
	references.Importance = widget.LowImportance
	definition := widget.NewButtonWithIcon("Definition", theme.NavigateNextIcon(), func() {
		result, ok := editorSvc.ResolveDefinition(tab.RelPath, source.Text, source.CursorRow, source.CursorColumn)
		if ok {
			editorSetCursorLine(source, result.Item.Line)
			status.SetText(definitionStatusText(result, true))
			return
		}
		if strings.TrimSpace(result.Query) == "" {
			status.SetText(definitionStatusText(result, false))
			return
		}
		workspaceResult, found, err := v.resolveWorkspaceDefinition(tab.RelPath, source.Text, result.Query)
		if err != nil {
			status.SetText("Workspace definition lookup failed: " + err.Error())
			return
		}
		if !found {
			status.SetText(definitionStatusText(result, false))
			return
		}
		if workspaceResult.RelPath == tab.RelPath {
			editorSetCursorLine(source, workspaceResult.Item.Line)
			status.SetText(definitionStatusText(workspaceResult, true))
			return
		}
		workspace := v.state.Workspace()
		preview, err := v.workspaceService.PreviewFile(workspace.Root, workspaceResult.RelPath)
		if err != nil {
			status.SetText("Workspace definition target could not be opened: " + err.Error())
			return
		}
		v.openPreviewTab(preview)
		if activeEditor, exists := v.textEditor(v.editorSession.ActiveID()); exists {
			editorSetCursorLine(activeEditor.source, workspaceResult.Item.Line)
			activeEditor.status.SetText(definitionStatusText(workspaceResult, true))
		}
	})
	definition.Importance = widget.LowImportance

	v.bindTextEditor(tab.ID, binding)

	if preview.Truncated {
		revert.Disable()
	}
	encodingActions := []fyne.CanvasObject{widget.NewLabel("Save as"), encodingSelect, definition, references, symbols, format, revert}
	if preview.Truncated {
		loadFull := widget.NewButtonWithIcon("Load full", theme.DownloadIcon(), func() {
			v.loadFullTextPreview(tab.RelPath, status)
		})
		loadFull.Importance = widget.MediumImportance
		encodingActions = append(encodingActions, loadFull)
	}
	encodingControl := container.NewHBox(encodingActions...)
	sourcePanel := container.NewBorder(container.NewBorder(nil, nil, status, encodingControl), nil, nil, nil, source)
	previewPanel := container.NewBorder(widget.NewLabel(previewHeader(preview)), nil, nil, nil, rendered.Canvas())
	outlinePanel := container.NewBorder(outlineStatus, nil, nil, nil, container.NewVScroll(outlineList))
	mapPanel := container.NewBorder(mapStatus, nil, nil, nil, container.NewVScroll(mapList))
	diagnosticsPanel := container.NewBorder(diagnosticStatus, nil, nil, nil, container.NewVScroll(diagnosticList))
	syntaxDetails := container.NewVScroll(container.NewVBox(languageActions, syntaxPreview))
	syntaxPanel := container.NewBorder(syntaxStatus, nil, nil, nil, container.NewHSplit(container.NewVScroll(syntaxGrid), syntaxDetails))
	tabs := container.NewAppTabs(
		container.NewTabItem("Source", sourcePanel),
		container.NewTabItem("Preview", previewPanel),
		container.NewTabItem("Highlight", syntaxPanel),
		container.NewTabItem("Diagnostics", diagnosticsPanel),
		container.NewTabItem("Outline", outlinePanel),
		container.NewTabItem("Map", mapPanel),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

func (v *View) loadFullTextPreview(relPath string, status *widget.Label) {
	workspace := v.state.Workspace()
	if strings.TrimSpace(workspace.Root) == "" {
		status.SetText("Open a workspace before loading the full file.")
		return
	}
	preview, err := v.workspaceService.FullTextPreviewFile(workspace.Root, relPath)
	if err != nil {
		status.SetText("Full load unavailable: " + err.Error())
		return
	}
	v.addActivity("Loaded full text preview for " + relPath + ".")
	v.openPreviewTab(preview)
}

func (b *textEditorBinding) writeEncoding() string {
	if b == nil {
		return "utf-8"
	}
	return editorWriteEncoding(b.saveEncoding)
}

func (b *textEditorBinding) encodingDirty() bool {
	if b == nil {
		return false
	}
	return editorWriteEncoding(b.sourceEncoding) != editorWriteEncoding(b.saveEncoding)
}

func (b *textEditorBinding) hasExplicitEncoding() bool {
	if b == nil {
		return true
	}
	return b.encodingExplicit
}

func (b *textEditorBinding) markEncodingSaved(encoding string) {
	if b == nil {
		return
	}
	next := editorWriteEncoding(encoding)
	b.sourceEncoding = next
	b.saveEncoding = next
	b.encodingExplicit = true
}

func editorEncodingOptions() []string {
	return []string{"utf-8", "utf-8-bom", "utf-16le", "utf-16be", "iso-8859-1", "windows-1251", "windows-1252"}
}

func editorWriteEncoding(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "utf8", "utf-8":
		return "utf-8"
	case "utf8-bom", "utf-8-bom", "utf-8 bom":
		return "utf-8-bom"
	case "utf16le", "utf-16le", "utf-16 le":
		return "utf-16le"
	case "utf16be", "utf-16be", "utf-16 be":
		return "utf-16be"
	case "latin1", "latin-1", "iso8859-1", "iso-8859-1":
		return "iso-8859-1"
	case "cp1251", "windows1251", "windows-1251":
		return "windows-1251"
	case "cp1252", "windows1252", "windows-1252":
		return "windows-1252"
	default:
		return value
	}
}

func (b *textEditorBinding) setOutline(text string) {
	if b == nil || b.outlineList == nil || b.outlineStatus == nil {
		return
	}
	items := editorSvc.BuildOutline(b.relPath, text)
	b.outlineList.Objects = b.outlineList.Objects[:0]
	if len(items) == 0 {
		b.outlineStatus.SetText("Outline: no symbols detected for this file.")
		b.outlineList.Add(widget.NewLabel("No outline symbols detected."))
		b.outlineList.Refresh()
		return
	}
	b.outlineStatus.SetText(fmt.Sprintf("Outline: %d symbol(s). Select one to move the editor cursor.", len(items)))
	for _, item := range items {
		current := item
		button := widget.NewButton(outlineItemText(current), func() {
			editorSetCursorLine(b.source, current.Line)
			b.outlineStatus.SetText(fmt.Sprintf("Moved cursor to %s on line %d.", current.Label, current.Line))
		})
		button.Alignment = widget.ButtonAlignLeading
		button.Importance = widget.LowImportance
		b.outlineList.Add(button)
	}
	b.outlineList.Refresh()
}

func (b *textEditorBinding) setDocumentMap(text string) {
	if b == nil || b.mapList == nil || b.mapStatus == nil {
		return
	}
	items := editorSvc.BuildDocumentMap(b.relPath, text)
	b.mapList.Objects = b.mapList.Objects[:0]
	if len(items) == 0 {
		b.mapStatus.SetText("Map: no landmarks detected for this file.")
		b.mapList.Add(widget.NewLabel("No document map landmarks detected."))
		b.mapList.Refresh()
		return
	}
	b.mapStatus.SetText(fmt.Sprintf("Map: %d landmark(s). This native overview replaces Monaco's minimap with jumpable structure.", len(items)))
	for _, item := range items {
		current := item
		button := widget.NewButton(documentMapItemText(current), func() {
			editorSetCursorLine(b.source, current.Line)
			b.mapStatus.SetText(fmt.Sprintf("Moved cursor to %s on line %d.", current.Label, current.Line))
		})
		button.Alignment = widget.ButtonAlignLeading
		button.Importance = widget.LowImportance
		b.mapList.Add(button)
	}
	b.mapList.Refresh()
}

func (b *textEditorBinding) setDiagnostics(text string) {
	if b == nil || b.diagnosticList == nil || b.diagnosticStatus == nil {
		return
	}
	diagnostics := editorSvc.AnalyzeDraftDiagnostics(b.relPath, text)
	b.diagnosticList.Objects = b.diagnosticList.Objects[:0]
	if len(diagnostics) == 0 {
		b.diagnosticStatus.SetText("Diagnostics: no draft problems detected.")
		b.diagnosticList.Add(widget.NewLabel("No live draft diagnostics for this file."))
		b.diagnosticList.Refresh()
		return
	}
	b.diagnosticStatus.SetText(draftDiagnosticsStatusText(diagnostics))
	for _, diagnostic := range diagnostics {
		current := diagnostic
		button := widget.NewButton(draftDiagnosticItemText(current), func() {
			editorSetCursorLine(b.source, current.Line)
			b.diagnosticStatus.SetText(fmt.Sprintf("Moved cursor to %s on line %d.", current.Source, current.Line))
		})
		button.Alignment = widget.ButtonAlignLeading
		button.Importance = widget.LowImportance
		b.diagnosticList.Add(button)
	}
	b.diagnosticList.Refresh()
}

func (b *textEditorBinding) setSyntax(text string) {
	if b == nil || b.syntaxStatus == nil || b.syntaxPreview == nil {
		return
	}
	analysis := editorSvc.AnalyzeSyntax(b.relPath, text)
	b.syntaxText = text
	b.syntaxAnalysis = analysis
	context := editorSvc.SyntaxContextFromAnalysis(text, analysis, b.source.CursorRow, b.source.CursorColumn)
	b.syntaxStatus.SetText(syntaxStatusTextWithCursor(analysis, context))
	if b.languageActions != nil {
		b.languageActions.SetText(formatLanguageActionPlan(editorSvc.BuildLanguageActionPlan(b.relPath, text)))
		b.languageActions.Refresh()
	}
	b.syntaxPreview.SetText(formatSyntaxAnalysis(analysis))
	b.syntaxPreview.Refresh()
	applySyntaxHighlightGridWithCursor(b.syntaxGrid, text, analysis, b.source.CursorRow)
}

func (b *textEditorBinding) updateSyntaxCursor() {
	if b == nil || b.source == nil || b.syntaxStatus == nil || b.syntaxGrid == nil {
		return
	}
	text := b.syntaxText
	if text == "" {
		text = b.source.Text
	}
	context := editorSvc.SyntaxContextFromAnalysis(text, b.syntaxAnalysis, b.source.CursorRow, b.source.CursorColumn)
	b.syntaxStatus.SetText(syntaxStatusTextWithCursor(b.syntaxAnalysis, context))
	applySyntaxHighlightGridWithCursor(b.syntaxGrid, text, b.syntaxAnalysis, b.source.CursorRow)
}

func outlineItemText(item editorSvc.OutlineItem) string {
	indent := strings.Repeat("  ", item.Level)
	return fmt.Sprintf("%s%s  %s  L%d", indent, item.Kind, item.Label, item.Line)
}

func documentMapItemText(item editorSvc.DocumentMapItem) string {
	return fmt.Sprintf("%s  %s  L%d", item.Kind, item.Label, item.Line)
}

func draftDiagnosticItemText(diagnostic editorSvc.DraftDiagnostic) string {
	return fmt.Sprintf("%s  %s/%s  L%d  %s", diagnostic.RelPath, diagnostic.Severity, diagnostic.Source, diagnostic.Line, diagnostic.Message)
}

func draftDiagnosticsStatusText(diagnostics []editorSvc.DraftDiagnostic) string {
	if len(diagnostics) == 0 {
		return "Diagnostics: no draft problems detected."
	}
	counts := map[string]int{}
	for _, diagnostic := range diagnostics {
		severity := strings.TrimSpace(diagnostic.Severity)
		if severity == "" {
			severity = "info"
		}
		counts[severity]++
	}
	parts := []string{}
	for _, severity := range []string{"error", "warning", "info"} {
		if counts[severity] > 0 {
			parts = append(parts, fmt.Sprintf("%s=%d", severity, counts[severity]))
		}
	}
	return fmt.Sprintf("Diagnostics: %d draft problem(s): %s.", len(diagnostics), strings.Join(parts, ", "))
}

func definitionStatusText(result editorSvc.DefinitionResult, resolved bool) string {
	query := strings.TrimSpace(result.Query)
	if query == "" {
		return "Place the cursor on a symbol name before using Definition."
	}
	if !resolved {
		return fmt.Sprintf("No local definition found for %s.", query)
	}
	if strings.TrimSpace(result.RelPath) != "" {
		return fmt.Sprintf("Moved to %s %s in %s on line %d.", result.Item.Kind, result.Item.Label, result.RelPath, result.Item.Line)
	}
	return fmt.Sprintf("Moved to %s %s on line %d.", result.Item.Kind, result.Item.Label, result.Item.Line)
}

func (v *View) resolveWorkspaceDefinition(currentRelPath string, currentContent string, query string) (editorSvc.DefinitionResult, bool, error) {
	workspace := v.state.Workspace()
	if strings.TrimSpace(workspace.Root) == "" {
		return editorSvc.DefinitionResult{Query: query}, false, nil
	}
	files := []editorSvc.DefinitionFile{{
		RelPath: currentRelPath,
		Content: currentContent,
	}}
	seen := map[string]bool{strings.Trim(strings.ReplaceAll(currentRelPath, "\\", "/"), "/"): true}
	results, err := v.workspaceService.Search(workspace.Root, query, workspaceSvc.SearchOptions{MaxResults: workspaceDefinitionSearchLimit})
	if err != nil {
		return editorSvc.DefinitionResult{}, false, err
	}
	for _, result := range results {
		relPath := strings.Trim(strings.ReplaceAll(result.RelPath, "\\", "/"), "/")
		if relPath == "" || seen[relPath] || result.Kind == "directory" {
			continue
		}
		seen[relPath] = true
		preview, err := v.workspaceService.PreviewFile(workspace.Root, relPath)
		if err != nil || strings.TrimSpace(preview.Text) == "" {
			continue
		}
		files = append(files, editorSvc.DefinitionFile{
			RelPath: preview.RelPath,
			Content: preview.Text,
		})
	}
	definition, ok := editorSvc.ResolveWorkspaceDefinition(query, currentRelPath, files)
	return definition, ok, nil
}

func syntaxStatusText(analysis editorSvc.SyntaxAnalysis) string {
	return syntaxStatusTextWithCursor(analysis, editorSvc.SyntaxCursorContext{})
}

func syntaxStatusTextWithCursor(analysis editorSvc.SyntaxAnalysis, context editorSvc.SyntaxCursorContext) string {
	language := strings.TrimSpace(analysis.Language.Label)
	if language == "" {
		language = "Plain text"
	}
	strategy := "native plain-text editing"
	if analysis.Language.NativeLight {
		strategy = "native lightweight tokenizer"
	}
	if analysis.Language.FutureLSP {
		strategy += "; LSP candidate"
	}
	base := ""
	if analysis.Truncated {
		base = fmt.Sprintf("Syntax: %s via %s. Showing first %d token(s); analysis capped for responsiveness.", language, strategy, len(analysis.Tokens))
	} else {
		base = fmt.Sprintf("Syntax: %s via %s. %d token(s) across %d line(s).", language, strategy, len(analysis.Tokens), analysis.LineCount)
	}
	if strings.TrimSpace(context.Message) == "" {
		return base
	}
	return base + " " + context.Message
}

func formatSyntaxAnalysis(analysis editorSvc.SyntaxAnalysis) string {
	var builder strings.Builder
	builder.WriteString("Language: ")
	builder.WriteString(firstNonEmptyString(analysis.Language.Label, "Plain text"))
	builder.WriteString("\nMode: ")
	if analysis.Language.NativeLight {
		builder.WriteString("native lightweight syntax")
	} else {
		builder.WriteString("plain text")
	}
	if analysis.Language.FutureLSP {
		builder.WriteString(" + future LSP candidate")
	}
	builder.WriteString("\n")
	if len(analysis.Counts) > 0 {
		builder.WriteString("Token counts: ")
		builder.WriteString(syntaxCountsText(analysis.Counts))
		builder.WriteString("\n")
	}
	if analysis.Truncated {
		builder.WriteString("Note: analysis capped to keep editing responsive.\n")
	}
	if len(analysis.Tokens) == 0 {
		builder.WriteString("\nNo syntax tokens detected for this file yet.\n")
		return builder.String()
	}
	builder.WriteString("\nTokens\n")
	limit := len(analysis.Tokens)
	if limit > 80 {
		limit = 80
	}
	for index := 0; index < limit; index++ {
		token := analysis.Tokens[index]
		builder.WriteString(fmt.Sprintf("L%d  %-7s  %s\n", token.Line, token.Kind, compactSyntaxToken(token.Text)))
	}
	if len(analysis.Tokens) > limit {
		builder.WriteString(fmt.Sprintf("... %d more token(s)\n", len(analysis.Tokens)-limit))
	}
	return builder.String()
}

func formatLanguageActionPlan(plan editorSvc.LanguageActionPlan) string {
	var builder strings.Builder
	builder.WriteString("Language Actions\n")
	builder.WriteString("Summary: ")
	builder.WriteString(firstNonEmptyString(plan.Summary, "no language actions"))
	builder.WriteString("\n")
	if strings.TrimSpace(plan.BetaStrategy.Decision) != "" {
		builder.WriteString("Native Parity Beta: ")
		builder.WriteString(plan.BetaStrategy.Status)
		builder.WriteString("\n")
		builder.WriteString(plan.BetaStrategy.Decision)
		builder.WriteString("\n")
		builder.WriteString(plan.BetaStrategy.InlineSyntax)
		builder.WriteString("\n")
		builder.WriteString(plan.BetaStrategy.LSP)
		builder.WriteString("\n")
	}
	for _, action := range plan.Actions {
		builder.WriteString("- ")
		builder.WriteString(action.Name)
		builder.WriteString(" [")
		builder.WriteString(action.Status)
		builder.WriteString("]: ")
		builder.WriteString(action.Detail)
		builder.WriteString("\n")
	}
	return builder.String()
}

func syntaxCountsText(counts map[string]int) string {
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}

func compactSyntaxToken(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) <= 80 {
		return value
	}
	return value[:77] + "..."
}

func draftStatusText(tab editorSvc.Tab) string {
	return draftStatusTextWithEncoding(tab, false, false)
}

func draftStatusTextWithEncoding(tab editorSvc.Tab, encodingDirty bool, encodingBlocked bool) string {
	if encodingBlocked {
		return ambiguousEncodingStatusText()
	}
	if encodingDirty && tab.Dirty {
		return "Draft modified and save encoding changed. Save applies through the safe write service and creates a rollback snapshot."
	}
	if encodingDirty {
		return "Save encoding changed. Save applies through the safe write service and creates a rollback snapshot."
	}
	if tab.Dirty {
		return "Draft modified. Save applies through the safe write service and creates a rollback snapshot."
	}
	return "Draft matches source."
}

func ambiguousEncodingStatusText() string {
	return "Encoding is ambiguous. Choose a save encoding before saving changes."
}

func (v *View) bindTextEditor(tabID string, binding *textEditorBinding) {
	if v.editor.textEditors == nil {
		v.editor.textEditors = map[string]*textEditorBinding{}
	}
	v.editor.textEditors[tabID] = binding
}

func (v *View) removeTextEditor(tabID string) {
	if len(v.editor.textEditors) == 0 {
		return
	}
	delete(v.editor.textEditors, tabID)
}

func (v *View) textEditor(tabID string) (*textEditorBinding, bool) {
	binding := v.editor.textEditors[tabID]
	return binding, binding != nil
}
