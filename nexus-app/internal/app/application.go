package app

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	"nexusdesk/internal/ui/shell"
	nexustheme "nexusdesk/internal/ui/theme"
)

const (
	appID    = "com.rcooler.nexus"
	appTitle = "Nexus Augentic Studio"
)

func Run() {
	application := fyneapp.NewWithID(appID)
	application.Settings().SetTheme(nexustheme.NexusTheme{})

	window := application.NewWindow(appTitle)
	window.Resize(fyne.NewSize(1440, 920))
	window.SetMaster()

	view := shell.New(window)
	window.SetContent(view.Canvas())
	window.ShowAndRun()
}
