package protectedsecret

import "runtime"

type BackendStatus struct {
	Backend   string
	Available bool
	Message   string
	Action    string
}

func Status() BackendStatus {
	switch runtime.GOOS {
	case "windows":
		return BackendStatus{
			Backend:   "Windows DPAPI",
			Available: Available(),
			Message:   "Windows DPAPI protected storage is available for provider API keys and connector credentials.",
		}
	case "darwin":
		available := Available()
		status := BackendStatus{
			Backend:   "macOS Keychain",
			Available: available,
		}
		if available {
			status.Message = "macOS Keychain protected storage is available for provider API keys and connector credentials."
			return status
		}
		status.Message = "macOS Keychain protected storage is unavailable in this build."
		status.Action = "Use a cgo-enabled signed macOS build so NexusDesk can store secrets in Keychain."
		return status
	case "linux":
		available := Available()
		status := BackendStatus{
			Backend:   "Linux Secret Service",
			Available: available,
		}
		if available {
			status.Message = "Linux Secret Service protected storage is available through secret-tool."
			return status
		}
		status.Message = "Linux Secret Service protected storage is unavailable because secret-tool was not found in PATH."
		status.Action = "Install libsecret secret-tool and ensure a Secret Service provider is running before saving credentials."
		return status
	default:
		return BackendStatus{
			Backend:   runtime.GOOS,
			Available: false,
			Message:   "Protected secret storage is not implemented on this platform.",
			Action:    "Do not save provider keys or connector credentials until this platform has an OS credential backend.",
		}
	}
}
