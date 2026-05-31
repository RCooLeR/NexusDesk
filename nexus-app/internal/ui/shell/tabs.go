package shell

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"nexusdesk/internal/domain"
	editorSvc "nexusdesk/internal/services/editor"
	readinessSvc "nexusdesk/internal/services/readiness"
	recentWorkspacesSvc "nexusdesk/internal/services/recentworkspaces"
	settingsSvc "nexusdesk/internal/services/settings"
	startupSvc "nexusdesk/internal/services/startup"
)

func newEditorTabs(welcomeItem *container.TabItem) *container.DocTabs {
	tabs := container.NewDocTabs(welcomeItem)
	tabs.SetTabLocation(container.TabLocationTop)
	return tabs
}

type editorController struct {
	view             *View
	tabs             *container.DocTabs
	openTabs         map[string]*container.TabItem
	tabIDs           map[*container.TabItem]string
	previews         map[string]domain.FilePreview
	textEditors      map[string]*textEditorBinding
	savingTabs       map[string]bool
	splitEnabled     bool
	splitPane        *container.Split
	splitOffset      float64
	secondaryRelPath string
}

func newEditorController(view *View, initialTabID string, welcomeItem *container.TabItem) *editorController {
	tabs := newEditorTabs(welcomeItem)
	return &editorController{
		view:        view,
		tabs:        tabs,
		openTabs:    map[string]*container.TabItem{initialTabID: tabs.Items[0]},
		tabIDs:      map[*container.TabItem]string{tabs.Items[0]: initialTabID},
		previews:    map[string]domain.FilePreview{},
		textEditors: map[string]*textEditorBinding{},
		savingTabs:  map[string]bool{},
	}
}

func (v *View) newWelcomePanel() fyne.CanvasObject {
	recentItems, recentErr := v.listRecentWorkspaces()
	if showEditorEmptyWelcome(v.state.Workspace(), recentItems, recentErr) {
		return v.newEditorEmptyWelcomePanel()
	}
	title := widget.NewRichTextFromMarkdown("# NexusDesk\n\nNative local-first workbench for code, data, agents, and artifacts.")
	title.Wrapping = fyne.TextWrapWord
	openWorkspaceButton := widget.NewButtonWithIcon("Open Workspace", theme.FolderOpenIcon(), v.openWorkspaceDialog)
	openFileButton := widget.NewButtonWithIcon("Open File", theme.FileTextIcon(), v.openFileDialog)
	providerSetupButton := widget.NewButtonWithIcon("Provider Setup", theme.SettingsIcon(), v.openProviderSetupWizardTab)
	sampleWorkflowButton := widget.NewButtonWithIcon("Sample Workflow", theme.MediaPlayIcon(), v.openSampleWorkflowGuideTab)
	diagnosticsButton := widget.NewButtonWithIcon("Diagnostics", theme.SearchIcon(), func() {
		if !v.selectBottomTab("Diagnostics") {
			v.addActivity("Diagnostics panel is unavailable.")
		}
	})
	onboarding := widget.NewRichTextFromMarkdown(v.welcomeOnboardingMarkdown())
	onboarding.Wrapping = fyne.TextWrapWord
	readiness := widget.NewRichTextFromMarkdown(v.welcomeReadinessMarkdown())
	readiness.Wrapping = fyne.TextWrapWord
	recent := recentWorkspaceRowsFrom(recentItems, recentErr, v.openWorkspace, v.removeRecentWorkspace, v.removeMissingRecentWorkspaces, v.clearRecentWorkspaces)
	contentObjects := []fyne.CanvasObject{
		title,
		container.NewHBox(openWorkspaceButton, openFileButton, providerSetupButton, sampleWorkflowButton, diagnosticsButton),
		widget.NewSeparator(),
	}
	if recovery := v.welcomeCrashRecoveryBanner(); recovery != nil {
		contentObjects = append(contentObjects, recovery, widget.NewSeparator())
	}
	contentObjects = append(contentObjects,
		widget.NewCard("First Run", "", onboarding),
		widget.NewSeparator(),
		widget.NewCard("Setup", "", readiness),
		widget.NewSeparator(),
		widget.NewCard("Recent Workspaces", "", recent),
	)
	content := container.NewVBox(contentObjects...)
	return container.NewPadded(container.NewVScroll(content))
}

func (v *View) newEditorEmptyWelcomePanel() fyne.CanvasObject {
	title := widget.NewLabel("No file open")
	title.TextStyle = fyne.TextStyle{Bold: true}
	subtitle := widget.NewLabel("Open a workspace or file to start.")
	subtitle.Wrapping = fyne.TextWrapWord
	openWorkspaceButton := widget.NewButtonWithIcon("Open Workspace", theme.FolderOpenIcon(), v.openWorkspaceDialog)
	openFileButton := widget.NewButtonWithIcon("Open File", theme.FileTextIcon(), v.openFileDialog)
	providerSetupButton := widget.NewButtonWithIcon("Provider Setup", theme.SettingsIcon(), v.openProviderSetupWizardTab)
	sampleWorkflowButton := widget.NewButtonWithIcon("Sample Workflow", theme.MediaPlayIcon(), v.openSampleWorkflowGuideTab)
	diagnosticsButton := widget.NewButtonWithIcon("Diagnostics", theme.SearchIcon(), func() {
		if !v.selectBottomTab("Diagnostics") {
			v.addActivity("Diagnostics panel is unavailable.")
		}
	})
	onboarding := widget.NewRichTextFromMarkdown(v.welcomeOnboardingMarkdown())
	onboarding.Wrapping = fyne.TextWrapWord
	contentObjects := []fyne.CanvasObject{
		title,
		subtitle,
		widget.NewSeparator(),
	}
	if recovery := v.welcomeCrashRecoveryBanner(); recovery != nil {
		contentObjects = append(contentObjects, recovery, widget.NewSeparator())
	}
	contentObjects = append(contentObjects,
		widget.NewCard("First Run", "", onboarding),
		widget.NewSeparator(),
		welcomeEmptyCommandRows(),
		widget.NewSeparator(),
		container.NewHBox(openWorkspaceButton, openFileButton, providerSetupButton, sampleWorkflowButton, diagnosticsButton),
	)
	content := container.NewVBox(contentObjects...)
	return container.NewPadded(container.NewCenter(content))
}

type welcomeEmptyCommand struct {
	Label    string
	Shortcut string
}

func welcomeEmptyCommands() []welcomeEmptyCommand {
	return []welcomeEmptyCommand{
		{Label: "Project View", Shortcut: "Alt+1"},
		{Label: "Go to File", Shortcut: "Ctrl+P"},
		{Label: "Search Everywhere", Shortcut: "Ctrl+Shift+P"},
		{Label: "Recent Files", Shortcut: "Ctrl+E"},
		{Label: "Command Palette", Shortcut: "Ctrl+Shift+P"},
		{Label: "Drop files here", Shortcut: ""},
	}
}

func welcomeEmptyCommandRows() fyne.CanvasObject {
	rows := container.NewVBox()
	for _, command := range welcomeEmptyCommands() {
		label := widget.NewLabel(command.Label)
		label.TextStyle = fyne.TextStyle{Monospace: true}
		shortcut := widget.NewLabel(command.Shortcut)
		shortcut.TextStyle = fyne.TextStyle{Monospace: true}
		rows.Add(container.NewBorder(nil, nil, label, nil, shortcut))
	}
	return rows
}

func showEditorEmptyWelcome(workspace domain.Workspace, recentItems []recentWorkspacesSvc.Workspace, recentErr error) bool {
	return strings.TrimSpace(workspace.Root) == "" && recentErr == nil && len(recentItems) == 0
}

func (v *View) welcomeCrashRecoveryBanner() fyne.CanvasObject {
	if v == nil || !v.startupStatus.PreviousUnclean {
		return nil
	}
	text := widget.NewRichTextFromMarkdown(formatWelcomeCrashRecoveryMarkdown(v.startupStatus))
	text.Wrapping = fyne.TextWrapWord
	return widget.NewCard("Crash Recovery", "", container.NewVBox(
		text,
		container.NewHBox(
			welcomeBottomTabButton("Diagnostics", theme.SearchIcon(), v),
			welcomeBottomTabButton("Jobs", theme.ListIcon(), v),
			welcomeBottomTabButton("Agent Audit", theme.HistoryIcon(), v),
			welcomeBottomTabButton("History", theme.DocumentIcon(), v),
		),
	))
}

func welcomeBottomTabButton(label string, icon fyne.Resource, view *View) *widget.Button {
	return widget.NewButtonWithIcon(label, icon, func() {
		if view == nil || !view.selectBottomTab(label) {
			if view != nil {
				view.addActivity(label + " panel is unavailable.")
			}
			return
		}
		view.addActivity(label + " selected for crash recovery review.")
	})
}

func formatWelcomeCrashRecoveryMarkdown(status startupSvc.Status) string {
	detail := firstNonEmptyString(status.Message, "Previous NexusDesk run did not record a clean exit.")
	if !status.PreviousStartedAt.IsZero() {
		detail += " Previous start: " + status.PreviousStartedAt.Local().Format("2006-01-02 15:04:05") + "."
	}
	return strings.Join([]string{
		"- **[ACTION] Previous session:** " + compactWelcomeReadinessText(detail, 220),
		"- Review Diagnostics, Jobs, Agent Audit, and History before repeating long workflows.",
		"- Export a redacted issue report from Diagnostics if metadata, provider, or job state looks suspicious.",
	}, "\n")
}

func (v *View) welcomeOnboardingMarkdown() string {
	current := settingsSvc.Defaults()
	settingsError := ""
	if v.settingsStore != nil {
		loaded, err := v.settingsStore.LoadForDisplay()
		if err != nil {
			settingsError = err.Error()
		} else {
			current = loaded
		}
	}
	return formatWelcomeOnboardingMarkdown(v.state.Workspace(), current, settingsError)
}

func formatWelcomeOnboardingMarkdown(workspace domain.Workspace, settings settingsSvc.Settings, settingsError string) string {
	providerStatus := "ACTION"
	providerDetail := "Open Model Settings, choose provider/base URL/model, then run Test connection."
	if strings.TrimSpace(settingsError) != "" {
		providerDetail = "Settings could not be loaded: " + compactWelcomeReadinessText(settingsError, 120)
	} else if strings.TrimSpace(settings.Provider) != "" && strings.TrimSpace(settings.BaseURL) != "" && strings.TrimSpace(settings.Model) != "" {
		providerStatus = "OK"
		providerDetail = fmt.Sprintf("%s/%s configured; run Test connection before long Ask or Agent workflows.", settings.Provider, settings.Model)
	}
	workspaceStatus := "ACTION"
	workspaceDetail := "Open a trusted sample workspace before running assistant, data, or artifact workflows."
	if strings.TrimSpace(workspace.Root) != "" {
		workspaceStatus = "OK"
		workspaceDetail = fmt.Sprintf("%s is open; keep heavy work explicit and use Jobs/Diagnostics for long runs.", firstNonEmptyString(workspace.Name, filepath.Base(workspace.Root)))
	}
	return strings.Join([]string{
		fmt.Sprintf("- **[%s] Provider setup:** %s", providerStatus, providerDetail),
		fmt.Sprintf("- **[%s] Workspace:** %s", workspaceStatus, workspaceDetail),
		"- **[NEXT] Sample workflow:** Open the Sample Workflow guide for a safe edit, Ask, Agent, Data, Artifacts, and Diagnostics path.",
		"- **[VERIFY] Diagnostics:** Run Diagnostics after setup changes and export a redacted issue report if anything fails.",
	}, "\n")
}

func (v *View) welcomeReadinessMarkdown() string {
	current := settingsSvc.Defaults()
	settingsError := ""
	if v.settingsStore != nil {
		loaded, err := v.settingsStore.LoadForDisplay()
		if err != nil {
			settingsError = err.Error()
		} else {
			current = loaded
		}
	}
	workspace := v.state.Workspace()
	return formatWelcomeReadinessMarkdown(readinessSvc.Collect(readinessSvc.Options{
		WorkspaceRoot:   workspace.Root,
		WorkspaceName:   workspace.Name,
		Settings:        current,
		SettingsError:   settingsError,
		StartupRecovery: v.startupStatus,
	}))
}

func formatWelcomeReadinessMarkdown(snapshot readinessSvc.Snapshot) string {
	var builder strings.Builder
	builder.WriteString("## First-run readiness\n\n")
	builder.WriteString("This native workspace keeps setup gaps visible before long-running agent work starts.\n\n")
	for _, item := range snapshot.Items {
		builder.WriteString(fmt.Sprintf("- **[%s] %s:** %s", welcomeReadinessStatusLabel(item.Status), item.Label, compactWelcomeReadinessText(item.Detail, 180)))
		if strings.TrimSpace(item.Action) != "" {
			builder.WriteString(" Next: ")
			builder.WriteString(compactWelcomeReadinessText(item.Action, 160))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("\n## Production failure gates\n\n")
	if err := readinessSvc.ValidateProductionFailureScenarios(snapshot.FailureScenarios); err != nil {
		builder.WriteString("- **[ACTION] Production failure scenarios:** ")
		builder.WriteString(compactWelcomeReadinessText(err.Error(), 180))
		builder.WriteString(" Open Diagnostics for the detailed matrix.\n")
		return builder.String()
	}
	builder.WriteString(fmt.Sprintf("- **[OK] Production failure scenarios:** %d scenario(s) cover crash/hang/provider/metadata/cancel release gates. Open Diagnostics for owners, automated checks, and manual smoke details.\n", len(snapshot.FailureScenarios)))
	return builder.String()
}

func welcomeReadinessStatusLabel(status string) string {
	switch status {
	case readinessSvc.StatusOK:
		return "OK"
	case readinessSvc.StatusWarning:
		return "WARN"
	default:
		return "ACTION"
	}
}

func compactWelcomeReadinessText(text string, limit int) string {
	compact := strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
	if limit <= 0 || len(compact) <= limit {
		return compact
	}
	if limit <= 3 {
		return compact[:limit]
	}
	return strings.TrimSpace(compact[:limit-3]) + "..."
}

func (v *View) recentWorkspaceRows() fyne.CanvasObject {
	items, err := v.listRecentWorkspaces()
	return recentWorkspaceRowsFrom(items, err, v.openWorkspace, v.removeRecentWorkspace, v.removeMissingRecentWorkspaces, v.clearRecentWorkspaces)
}

func recentWorkspaceRowsFrom(items []recentWorkspacesSvc.Workspace, err error, open func(string), remove func(string), removeMissing func(), clear func()) fyne.CanvasObject {
	if err != nil {
		return widget.NewLabel("Recent workspaces are unavailable: " + err.Error())
	}
	if len(items) == 0 {
		return widget.NewLabel("No recent workspaces yet.")
	}
	rows := []fyne.CanvasObject{widget.NewLabel("Recent workspaces")}
	hasMissing := false
	for _, item := range items {
		item := item
		if item.Missing {
			hasMissing = true
		}
		openButton := widget.NewButtonWithIcon(recentWorkspaceButtonLabel(item), theme.FolderOpenIcon(), func() {
			open(item.Path)
		})
		if item.Missing {
			openButton.Disable()
		}
		removeButton := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
			remove(item.Path)
		})
		removeButton.Importance = widget.LowImportance
		pathLabel := widget.NewLabel(recentWorkspacePathLabel(item))
		pathLabel.Truncation = fyne.TextTruncateEllipsis
		rows = append(rows, container.NewBorder(nil, nil, openButton, removeButton, pathLabel))
	}
	if hasMissing {
		removeMissingButton := widget.NewButtonWithIcon("Remove missing", theme.DeleteIcon(), removeMissing)
		removeMissingButton.Importance = widget.LowImportance
		rows = append(rows, container.NewHBox(layout.NewSpacer(), removeMissingButton))
	}
	clearButton := widget.NewButtonWithIcon("Clear recent workspaces", theme.DeleteIcon(), clear)
	clearButton.Importance = widget.LowImportance
	rows = append(rows, container.NewHBox(layout.NewSpacer(), clearButton))
	return container.NewVBox(rows...)
}

func recentWorkspaceButtonLabel(item recentWorkspacesSvc.Workspace) string {
	name := strings.TrimSpace(item.Name)
	if name == "" {
		name = filepath.Base(item.Path)
	}
	if item.Missing {
		return name + " (missing)"
	}
	return name
}

func recentWorkspacePathLabel(item recentWorkspacesSvc.Workspace) string {
	if item.Missing {
		return "Missing: " + item.Path
	}
	return item.Path
}

func (v *View) listRecentWorkspaces() ([]recentWorkspacesSvc.Workspace, error) {
	if v.recentWorkspaceStore == nil {
		return nil, fmt.Errorf("recent workspace store is unavailable")
	}
	return v.recentWorkspaceStore.List()
}

func (v *View) recordRecentWorkspace(root string) {
	if v.recentWorkspaceStore == nil {
		return
	}
	if _, err := v.recentWorkspaceStore.Add(root); err != nil {
		v.addActivity("Could not update recent workspaces: " + err.Error())
	}
}

func (v *View) removeRecentWorkspace(root string) {
	if v.recentWorkspaceStore == nil {
		v.addActivity("Recent workspace store is unavailable.")
		return
	}
	if _, err := v.recentWorkspaceStore.Remove(root); err != nil {
		v.addActivity("Could not remove recent workspace: " + err.Error())
		return
	}
	v.refreshWelcomeTabs()
}

func (v *View) removeMissingRecentWorkspaces() {
	if v.recentWorkspaceStore == nil {
		v.addActivity("Recent workspace store is unavailable.")
		return
	}
	items, err := v.recentWorkspaceStore.RemoveMissing()
	if err != nil {
		v.addActivity("Could not remove missing recent workspaces: " + err.Error())
		return
	}
	v.addActivity(fmt.Sprintf("Removed missing recent workspaces; %d recent workspace(s) remain.", len(items)))
	v.refreshWelcomeTabs()
}

func (v *View) clearRecentWorkspaces() {
	if v.recentWorkspaceStore == nil {
		v.addActivity("Recent workspace store is unavailable.")
		return
	}
	if _, err := v.recentWorkspaceStore.Clear(); err != nil {
		v.addActivity("Could not clear recent workspaces: " + err.Error())
		return
	}
	v.refreshWelcomeTabs()
}

func (v *View) openHomeTab() {
	for item, id := range v.editor.tabIDs {
		tab, ok := v.editorSession.Tab(id)
		if !ok || tab.Kind != editorSvc.KindWelcome {
			continue
		}
		item.Content = v.newWelcomePanel()
		v.editor.tabs.Select(item)
		return
	}
	tabState := v.editorSession.OpenWelcome("Home")
	item := container.NewTabItemWithIcon(editorTabTitle(tabState), theme.HomeIcon(), v.newWelcomePanel())
	v.editor.openTabs[tabState.ID] = item
	v.editor.tabIDs[item] = tabState.ID
	v.editor.tabs.Append(item)
	v.editor.tabs.Select(item)
}

func (v *View) refreshWelcomeTabs() {
	for item, id := range v.editor.tabIDs {
		tab, ok := v.editorSession.Tab(id)
		if !ok || tab.Kind != editorSvc.KindWelcome {
			continue
		}
		item.Content = v.newWelcomePanel()
		item.Content.Refresh()
	}
}

func (v *View) configureEditorTabs() {
	v.editor.tabs.CloseIntercept = func(item *container.TabItem) {
		v.requestCloseTab(item)
	}
	v.editor.tabs.OnSelected = func(*container.TabItem) {
		v.refreshStatusBar()
	}
}

func (v *View) openPreviewTab(preview domain.FilePreview) {
	tabState := v.editorSession.OpenFileWithSource(preview.RelPath, filepath.Base(preview.RelPath), preview.Text)
	if existing := v.editor.openTabs[tabState.ID]; existing != nil {
		existing.Content = v.newEditorPanel(tabState, preview)
		v.updateEditorTabState(tabState)
		v.editor.tabs.Select(existing)
		return
	}
	tab := container.NewTabItemWithIcon(editorTabTitle(tabState), editorTabIcon(tabState), v.newEditorPanel(tabState, preview))
	v.editor.openTabs[tabState.ID] = tab
	v.editor.tabIDs[tab] = tabState.ID
	v.editor.tabs.Append(tab)
	v.editor.tabs.Select(tab)
}

func (v *View) addPlaceholderTab(title string, body string) {
	tabState := v.editorSession.OpenPlaceholder(title)
	tab := container.NewTabItemWithIcon(editorTabTitle(tabState), editorTabIcon(tabState), widget.NewRichTextFromMarkdown(body))
	v.editor.openTabs[tabState.ID] = tab
	v.editor.tabIDs[tab] = tabState.ID
	v.editor.tabs.Append(tab)
	v.editor.tabs.Select(tab)
}

func (v *View) closeWelcomeTabs() {
	for item, id := range v.editor.tabIDs {
		tab, ok := v.editorSession.Tab(id)
		if !ok || tab.Kind != editorSvc.KindWelcome {
			continue
		}
		v.closeEditorTabItem(item, id, true)
	}
}
