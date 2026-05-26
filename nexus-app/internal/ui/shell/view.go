package shell

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	editorSvc "nexusdesk/internal/services/editor"
	workspaceSvc "nexusdesk/internal/services/workspace"
)

type View struct {
	window           fyne.Window
	state            *State
	workspaceService *workspaceSvc.Service
	editorSession    *editorSvc.Session
	status           *widget.Label
	navigator        *fyne.Container
	editorTabs       *container.DocTabs
	openTabs         map[string]*container.TabItem
	tabIDs           map[*container.TabItem]string
	activityLog      *widget.RichText
	activityText     string
	searchResults    *fyne.Container
	searchStatus     *widget.Label
}

func New(window fyne.Window) *View {
	editorSession := editorSvc.NewSession()
	welcome := editorSession.OpenWelcome("Welcome")
	editorTabs := newEditorTabs(welcome.Title)
	view := &View{
		window:           window,
		state:            NewState(),
		workspaceService: workspaceSvc.New(),
		editorSession:    editorSession,
		status:           widget.NewLabel("No workspace open"),
		navigator:        container.NewStack(widget.NewLabel("Open a workspace to browse files.")),
		editorTabs:       editorTabs,
		openTabs:         map[string]*container.TabItem{welcome.ID: editorTabs.Items[0]},
		tabIDs:           map[*container.TabItem]string{editorTabs.Items[0]: welcome.ID},
		activityLog:      widget.NewRichTextFromMarkdown("Ready."),
		activityText:     "Ready.",
		searchResults:    container.NewVBox(widget.NewLabel("Search results will appear here.")),
		searchStatus:     widget.NewLabel("No search yet."),
	}
	view.configureEditorTabs()
	return view
}

func (v *View) Canvas() fyne.CanvasObject {
	rail := v.newRail()
	workbench := container.NewBorder(v.newToolbar(), v.newBottomPanel(), v.navigator, v.newAssistantPanel(), v.editorTabs)
	return container.NewBorder(nil, v.status, rail, nil, workbench)
}
