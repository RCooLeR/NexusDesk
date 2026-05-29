package protectedsecret

import (
	"runtime"
	"strings"
	"testing"
)

func TestStatusNamesPlatformBackend(t *testing.T) {
	status := Status()
	if strings.TrimSpace(status.Backend) == "" {
		t.Fatal("expected backend name")
	}
	if strings.TrimSpace(status.Message) == "" {
		t.Fatal("expected status message")
	}
	if status.Available != Available() {
		t.Fatalf("expected status availability to match Available(), got %t want %t", status.Available, Available())
	}
	switch runtime.GOOS {
	case "windows":
		if !strings.Contains(status.Backend, "DPAPI") {
			t.Fatalf("expected Windows DPAPI backend, got %#v", status)
		}
	case "darwin":
		if !strings.Contains(status.Backend, "Keychain") {
			t.Fatalf("expected macOS Keychain backend, got %#v", status)
		}
	case "linux":
		if !strings.Contains(status.Backend, "Secret Service") {
			t.Fatalf("expected Linux Secret Service backend, got %#v", status)
		}
	}
}
