package readiness

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	settingsSvc "nexusdesk/internal/services/settings"
	startupSvc "nexusdesk/internal/services/startup"
)

const (
	StatusOK      = "ok"
	StatusWarning = "warning"
	StatusAction  = "action"
)

type Item struct {
	ID     string
	Label  string
	Status string
	Detail string
	Action string
}

type ToolchainStatus struct {
	OS             string
	Arch           string
	CGOEnabled     string
	GCCPath        string
	MSYS2UCRT64Bin string
	Status         string
	Detail         string
	Action         string
}

type Snapshot struct {
	CollectedAt      time.Time
	WorkspaceOpen    bool
	WorkspaceName    string
	WorkspaceRoot    string
	SettingsLoaded   bool
	SettingsError    string
	Provider         string
	Protocol         string
	BaseURL          string
	Model            string
	ModelConfigured  bool
	APIKeyRequired   bool
	APIKeyConfigured bool
	Toolchain        ToolchainStatus
	StartupRecovery  startupSvc.Status
	Items            []Item
}

type Options struct {
	WorkspaceRoot   string
	WorkspaceName   string
	Settings        settingsSvc.Settings
	SettingsError   string
	Now             time.Time
	GOOS            string
	GOARCH          string
	MSYS2UCRT64Bin  string
	LookupPath      func(string) (string, error)
	Getenv          func(string) string
	Stat            func(string) (os.FileInfo, error)
	StartupRecovery startupSvc.Status
}

func Collect(options Options) Snapshot {
	now := options.Now
	if now.IsZero() {
		now = time.Now()
	}
	config := normalizedSettings(options.Settings)
	profile, hasProfile := settingsSvc.ProviderProfileByID(config.Provider)
	apiKeyRequired := hasProfile && profile.RequiresAPIKey
	apiKeyConfigured := strings.TrimSpace(config.APIKey) != ""

	snapshot := Snapshot{
		CollectedAt:      now,
		WorkspaceOpen:    strings.TrimSpace(options.WorkspaceRoot) != "",
		WorkspaceName:    strings.TrimSpace(options.WorkspaceName),
		WorkspaceRoot:    strings.TrimSpace(options.WorkspaceRoot),
		SettingsLoaded:   strings.TrimSpace(options.SettingsError) == "",
		SettingsError:    strings.TrimSpace(options.SettingsError),
		Provider:         config.Provider,
		Protocol:         config.Protocol,
		BaseURL:          config.BaseURL,
		Model:            config.Model,
		ModelConfigured:  strings.TrimSpace(config.Model) != "",
		APIKeyRequired:   apiKeyRequired,
		APIKeyConfigured: apiKeyConfigured,
		Toolchain:        inspectToolchain(options),
		StartupRecovery:  options.StartupRecovery,
	}
	snapshot.Items = append(snapshot.Items,
		workspaceItem(snapshot),
		settingsItem(snapshot),
		modelItem(snapshot),
		credentialsItem(snapshot),
		toolchainItem(snapshot.Toolchain),
		startupRecoveryItem(snapshot.StartupRecovery),
		safetyItem(),
	)
	return snapshot
}

func FormatMarkdown(snapshot Snapshot) string {
	var builder strings.Builder
	builder.WriteString("## First-run readiness\n\n")
	builder.WriteString("This native workspace cockpit keeps setup gaps visible before long-running agent work starts.\n\n")
	for _, item := range snapshot.Items {
		builder.WriteString(fmt.Sprintf("- **[%s] %s:** %s", statusLabel(item.Status), item.Label, item.Detail))
		if strings.TrimSpace(item.Action) != "" {
			builder.WriteString(" Next: ")
			builder.WriteString(item.Action)
		}
		builder.WriteString("\n")
	}
	return builder.String()
}

func normalizedSettings(settings settingsSvc.Settings) settingsSvc.Settings {
	defaults := settingsSvc.Defaults()
	if strings.TrimSpace(settings.Provider) == "" {
		settings.Provider = defaults.Provider
	}
	if strings.TrimSpace(settings.Protocol) == "" {
		if profile, ok := settingsSvc.ProviderProfileByID(strings.TrimSpace(settings.Provider)); ok {
			settings.Protocol = profile.Protocol
		} else {
			settings.Protocol = defaults.Protocol
		}
	}
	if strings.TrimSpace(settings.BaseURL) == "" {
		if profile, ok := settingsSvc.ProviderProfileByID(strings.TrimSpace(settings.Provider)); ok && profile.DefaultBaseURL != "" {
			settings.BaseURL = profile.DefaultBaseURL
		} else {
			settings.BaseURL = defaults.BaseURL
		}
	}
	settings.Provider = strings.TrimSpace(settings.Provider)
	settings.Protocol = strings.TrimSpace(settings.Protocol)
	settings.BaseURL = strings.TrimSpace(settings.BaseURL)
	settings.Model = strings.TrimSpace(settings.Model)
	settings.APIKey = strings.TrimSpace(settings.APIKey)
	return settings
}

func workspaceItem(snapshot Snapshot) Item {
	if snapshot.WorkspaceOpen {
		name := snapshot.WorkspaceName
		if name == "" {
			name = filepath.Base(snapshot.WorkspaceRoot)
		}
		return Item{
			ID:     "workspace",
			Label:  "Workspace",
			Status: StatusOK,
			Detail: fmt.Sprintf("%s is open and indexed from %s.", name, snapshot.WorkspaceRoot),
		}
	}
	return Item{
		ID:     "workspace",
		Label:  "Workspace",
		Status: StatusAction,
		Detail: "No workspace is open yet.",
		Action: "Open a trusted project folder to enable search, assistant context, tasks, metadata, and rollback records.",
	}
}

func settingsItem(snapshot Snapshot) Item {
	if _, ok := settingsSvc.ProviderProfileByID(snapshot.Provider); !ok {
		return Item{
			ID:     "settings",
			Label:  "Provider settings",
			Status: StatusWarning,
			Detail: "Provider profile " + snapshot.Provider + " is not one of the built-in native profiles.",
			Action: "Open Settings and choose Ollama, OpenAI-compatible, or Custom OpenAI-compatible.",
		}
	}
	if snapshot.SettingsLoaded {
		return Item{
			ID:     "settings",
			Label:  "Provider settings",
			Status: StatusOK,
			Detail: fmt.Sprintf("%s via %s at %s.", snapshot.Provider, snapshot.Protocol, snapshot.BaseURL),
		}
	}
	return Item{
		ID:     "settings",
		Label:  "Provider settings",
		Status: StatusWarning,
		Detail: "Settings could not be loaded: " + snapshot.SettingsError,
		Action: "Open Settings and save the provider configuration again.",
	}
}

func modelItem(snapshot Snapshot) Item {
	if snapshot.ModelConfigured {
		return Item{
			ID:     "model",
			Label:  "Model",
			Status: StatusOK,
			Detail: snapshot.Model + " is selected for assistant and agent work.",
		}
	}
	return Item{
		ID:     "model",
		Label:  "Model",
		Status: StatusAction,
		Detail: "No chat model is selected.",
		Action: "Open Settings, choose a model, and run Test connection before using agent workflows.",
	}
}

func credentialsItem(snapshot Snapshot) Item {
	if !snapshot.APIKeyRequired {
		return Item{
			ID:     "credentials",
			Label:  "Credentials",
			Status: StatusOK,
			Detail: "The selected provider profile does not require an API key.",
		}
	}
	if snapshot.APIKeyConfigured {
		return Item{
			ID:     "credentials",
			Label:  "Credentials",
			Status: StatusOK,
			Detail: "An API key is configured and displayed redacted.",
		}
	}
	return Item{
		ID:     "credentials",
		Label:  "Credentials",
		Status: StatusAction,
		Detail: "This provider requires an API key, but none is configured.",
		Action: "Save the key in Settings so protected storage can handle it.",
	}
}

func toolchainItem(toolchain ToolchainStatus) Item {
	return Item{
		ID:     "toolchain",
		Label:  "Native build toolchain",
		Status: toolchain.Status,
		Detail: toolchain.Detail,
		Action: toolchain.Action,
	}
}

func safetyItem() Item {
	return Item{
		ID:     "safety",
		Label:  "Local safety",
		Status: StatusOK,
		Detail: "Approvals, rollback records, job history, redacted issue reports, and local metadata are available for risky actions.",
	}
}

func startupRecoveryItem(status startupSvc.Status) Item {
	if status.PreviousUnclean {
		return Item{
			ID:     "startup",
			Label:  "Startup recovery",
			Status: StatusWarning,
			Detail: status.Message,
			Action: "Open Diagnostics, inspect recent Jobs and Agent Audit, then export an issue report if the previous run lost work.",
		}
	}
	if strings.TrimSpace(status.Message) != "" && strings.Contains(strings.ToLower(status.Message), "unavailable") {
		return Item{
			ID:     "startup",
			Label:  "Startup recovery",
			Status: StatusWarning,
			Detail: status.Message,
			Action: "Verify app data permissions if startup recovery markers keep failing.",
		}
	}
	return Item{
		ID:     "startup",
		Label:  "Startup recovery",
		Status: StatusOK,
		Detail: "Clean-exit markers are active for crash and hang triage.",
	}
}

func inspectToolchain(options Options) ToolchainStatus {
	goos := firstNonEmpty(strings.TrimSpace(options.GOOS), runtime.GOOS)
	goarch := firstNonEmpty(strings.TrimSpace(options.GOARCH), runtime.GOARCH)
	lookupPath := options.LookupPath
	if lookupPath == nil {
		lookupPath = exec.LookPath
	}
	getenv := options.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	stat := options.Stat
	if stat == nil {
		stat = os.Stat
	}
	cgoEnabled := strings.TrimSpace(getenv("CGO_ENABLED"))
	status := ToolchainStatus{
		OS:         goos,
		Arch:       goarch,
		CGOEnabled: cgoEnabled,
		Status:     StatusOK,
	}
	if cgoEnabled == "0" {
		status.Status = StatusWarning
		status.Detail = "CGO_ENABLED=0 is set; Fyne desktop builds need CGO enabled."
		status.Action = "Unset CGO_ENABLED or set CGO_ENABLED=1 before building installers."
		return status
	}
	if goos == "windows" {
		return inspectWindowsToolchain(status, options, lookupPath, stat)
	}
	return inspectPOSIXToolchain(status, lookupPath)
}

func inspectWindowsToolchain(status ToolchainStatus, options Options, lookupPath func(string) (string, error), stat func(string) (os.FileInfo, error)) ToolchainStatus {
	if gccPath, err := lookupPath("gcc.exe"); err == nil && strings.TrimSpace(gccPath) != "" {
		status.GCCPath = gccPath
		status.Detail = "gcc.exe is available on PATH for CGO/Fyne builds."
		return status
	}
	ucrtBin := firstNonEmpty(strings.TrimSpace(options.MSYS2UCRT64Bin), `C:\msys64\ucrt64\bin`)
	status.MSYS2UCRT64Bin = ucrtBin
	ucrtGCC := filepath.Join(ucrtBin, "gcc.exe")
	if _, err := stat(ucrtGCC); err == nil {
		status.GCCPath = ucrtGCC
		status.Status = StatusWarning
		status.Detail = "MSYS2 UCRT64 gcc.exe exists but is not on PATH."
		status.Action = "Run scripts/dev-env.ps1 or add " + ucrtBin + " to PATH before building."
		return status
	}
	status.Status = StatusAction
	status.Detail = "No Windows CGO compiler was found for Fyne builds."
	status.Action = "Install MSYS2 UCRT64 mingw-w64-ucrt-x86_64-gcc or run the Windows CI setup script."
	return status
}

func inspectPOSIXToolchain(status ToolchainStatus, lookupPath func(string) (string, error)) ToolchainStatus {
	for _, name := range []string{"gcc", "cc", "clang"} {
		path, err := lookupPath(name)
		if err == nil && strings.TrimSpace(path) != "" {
			status.GCCPath = path
			status.Detail = name + " is available on PATH for CGO/Fyne builds."
			return status
		}
	}
	status.Status = StatusWarning
	status.Detail = "No C compiler was found on PATH for CGO/Fyne builds."
	status.Action = "Install the platform compiler toolchain before packaging desktop releases."
	return status
}

func statusLabel(status string) string {
	switch status {
	case StatusOK:
		return "OK"
	case StatusWarning:
		return "WARN"
	case StatusAction:
		return "ACTION"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
