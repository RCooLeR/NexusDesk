//go:build linux

package protectedsecret

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxSecretServiceCommandRoundTrip(t *testing.T) {
	fake := filepath.Join(t.TempDir(), "secret-tool")
	store := filepath.Join(t.TempDir(), "secret")
	script := `#!/bin/sh
set -eu
store="${NEXUSDESK_FAKE_SECRET_STORE}"
cmd="$1"
shift
purpose=""
account=""
case "$cmd" in
  store)
    while [ "$#" -gt 0 ]; do
      case "$1" in
        --label) shift 2 ;;
        application) shift 2 ;;
        purpose) purpose="$2"; shift 2 ;;
        account) account="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
    cat > "${store}.${purpose}.${account}"
    ;;
  lookup)
    while [ "$#" -gt 0 ]; do
      case "$1" in
        application) shift 2 ;;
        purpose) purpose="$2"; shift 2 ;;
        account) account="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
    cat "${store}.${purpose}.${account}"
    ;;
  clear)
    while [ "$#" -gt 0 ]; do
      case "$1" in
        application) shift 2 ;;
        purpose) purpose="$2"; shift 2 ;;
        account) account="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
    rm -f "${store}.${purpose}.${account}"
    ;;
  *) exit 2 ;;
esac
`
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile fake secret-tool failed: %v", err)
	}
	oldCommand := secretToolCommand
	oldStore := os.Getenv("NEXUSDESK_FAKE_SECRET_STORE")
	secretToolCommand = fake
	t.Setenv("NEXUSDESK_FAKE_SECRET_STORE", store)
	t.Cleanup(func() {
		secretToolCommand = oldCommand
		_ = os.Setenv("NEXUSDESK_FAKE_SECRET_STORE", oldStore)
	})

	token, err := Protect("settings.api-key", []byte("linux-secret"))
	if err != nil {
		t.Fatalf("Protect returned error: %v", err)
	}
	plain, err := Unprotect(token)
	if err != nil {
		t.Fatalf("Unprotect returned error: %v", err)
	}
	if string(plain) != "linux-secret" {
		t.Fatalf("expected restored secret, got %q", string(plain))
	}
	if err := Delete(token); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := Unprotect(token); err == nil {
		t.Fatal("expected lookup to fail after Delete")
	}
}
