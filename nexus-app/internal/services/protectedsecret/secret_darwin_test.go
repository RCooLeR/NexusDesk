//go:build darwin

package protectedsecret

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDarwinKeychainCommandRoundTrip(t *testing.T) {
	fake := filepath.Join(t.TempDir(), "security")
	store := filepath.Join(t.TempDir(), "secret")
	script := `#!/bin/sh
set -eu
store="${NEXUSDESK_FAKE_KEYCHAIN_STORE}"
cmd="$1"
shift
account=""
service=""
password=""
case "$cmd" in
  add-generic-password)
    while [ "$#" -gt 0 ]; do
      case "$1" in
        -a) account="$2"; shift 2 ;;
        -s) service="$2"; shift 2 ;;
        -w) password="$2"; shift 2 ;;
        -U) shift ;;
        *) shift ;;
      esac
    done
    printf "%s" "$password" > "${store}.${service}.${account}"
    ;;
  find-generic-password)
    while [ "$#" -gt 0 ]; do
      case "$1" in
        -a) account="$2"; shift 2 ;;
        -s) service="$2"; shift 2 ;;
        -w) shift ;;
        *) shift ;;
      esac
    done
    cat "${store}.${service}.${account}"
    ;;
  delete-generic-password)
    while [ "$#" -gt 0 ]; do
      case "$1" in
        -a) account="$2"; shift 2 ;;
        -s) service="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
    rm -f "${store}.${service}.${account}"
    ;;
  *) exit 2 ;;
esac
`
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile fake security failed: %v", err)
	}
	oldCommand := securityCommand
	oldStore := os.Getenv("NEXUSDESK_FAKE_KEYCHAIN_STORE")
	securityCommand = fake
	t.Setenv("NEXUSDESK_FAKE_KEYCHAIN_STORE", store)
	t.Cleanup(func() {
		securityCommand = oldCommand
		_ = os.Setenv("NEXUSDESK_FAKE_KEYCHAIN_STORE", oldStore)
	})

	token, err := Protect("settings.api-key", []byte("darwin-secret"))
	if err != nil {
		t.Fatalf("Protect returned error: %v", err)
	}
	plain, err := Unprotect(token)
	if err != nil {
		t.Fatalf("Unprotect returned error: %v", err)
	}
	if string(plain) != "darwin-secret" {
		t.Fatalf("expected restored secret, got %q", string(plain))
	}
	if err := Delete(token); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := Unprotect(token); err == nil {
		t.Fatal("expected lookup to fail after Delete")
	}
}
