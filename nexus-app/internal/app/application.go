package app

import (
	"fmt"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	"nexusdesk/internal/brand"
	"nexusdesk/internal/buildinfo"
	startupSvc "nexusdesk/internal/services/startup"
	"nexusdesk/internal/ui/shell"
	nexustheme "nexusdesk/internal/ui/theme"
)

func RunWithArgs(args []string) int {
	if len(args) > 0 {
		switch strings.TrimSpace(args[0]) {
		case "--version":
			fmt.Fprintln(os.Stdout, buildinfo.AboutText())
			return 0
		case "--smoke-check":
			if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
				fmt.Fprintln(os.Stderr, "--smoke-check requires a workspace directory")
				return 2
			}
			if err := RunSmokeCheck(args[1], os.Stdout); err != nil {
				fmt.Fprintln(os.Stderr, "NexusDesk smoke check failed:", err)
				return 1
			}
			return 0
		}
	}
	Run()
	return 0
}

func Run() {
	started := time.Now().UTC()
	startupStore, startupStatus := beginStartupSession()
	if startupStore != nil {
		defer func() { _ = startupStore.MarkClean(startupStatus.CurrentID, time.Time{}) }()
	}
	application := fyneapp.NewWithID(buildinfo.AppID)
	application.SetIcon(brand.AppIcon())
	application.Settings().SetTheme(nexustheme.NexusTheme{})

	window := application.NewWindow(buildinfo.AppName)
	window.SetIcon(brand.AppIcon())
	window.Resize(fyne.NewSize(1280, 820))
	window.CenterOnScreen()
	window.SetMaster()

	view := shell.NewWithStartupStatus(window, startupStatus)
	view.InstallWindowActions()
	window.SetContent(view.Canvas())
	view.RecordStartupReady(started, "native shell content is ready")
	window.ShowAndRun()
}

func beginStartupSession() (*startupSvc.Store, startupSvc.Status) {
	store, err := startupSvc.NewStore()
	if err != nil {
		return nil, startupSvc.Status{Message: "Startup recovery marker is unavailable: " + err.Error()}
	}
	info := buildinfo.Current()
	status, err := store.Begin(startupSvc.Options{
		AppName: info.AppName,
		Version: info.Version,
		Commit:  info.Commit,
	})
	if err != nil {
		return nil, startupSvc.Status{Path: store.Path(), Message: "Startup recovery marker could not be written: " + err.Error()}
	}
	return store, status
}
