package app

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	"nexusdesk/internal/brand"
	"nexusdesk/internal/buildinfo"
	"nexusdesk/internal/ui/shell"
	nexustheme "nexusdesk/internal/ui/theme"
)

func Run() {
	application := fyneapp.NewWithID(buildinfo.AppID)
	application.SetIcon(brand.AppIcon())
	application.Settings().SetTheme(nexustheme.NexusTheme{})

	window := application.NewWindow(buildinfo.AppName)
	window.SetIcon(brand.AppIcon())
	window.Resize(fyne.NewSize(1440, 920))
	window.SetMaster()

	view := shell.New(window)
	view.InstallWindowActions()
	window.SetContent(view.Canvas())
	window.ShowAndRun()
}
