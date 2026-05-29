//go:build windows

package protectedsecret

import (
	"strconv"
	"strings"
	"testing"
)

func TestWindowsDPAPIRoundTrip(t *testing.T) {
	protected, err := Protect("settings.api-key", []byte("windows-secret"))
	if err != nil {
		t.Fatalf("Protect returned error: %v", err)
	}
	if string(protected) == "windows-secret" {
		t.Fatal("expected protected secret bytes to differ from plaintext")
	}

	plain, err := Unprotect(protected)
	if err != nil {
		t.Fatalf("Unprotect returned error: %v", err)
	}
	if string(plain) != "windows-secret" {
		t.Fatalf("expected restored secret, got %q", string(plain))
	}
}

func TestWindowsDPAPIStressRoundTrip(t *testing.T) {
	for index := 0; index < 50; index++ {
		secret := []byte("windows-secret-" + strconv.Itoa(index) + "-" + strings.Repeat("x", index))
		protected, err := Protect("settings.api-key", secret)
		if err != nil {
			t.Fatalf("Protect(%d) returned error: %v", index, err)
		}
		plain, err := Unprotect(protected)
		if err != nil {
			t.Fatalf("Unprotect(%d) returned error: %v", index, err)
		}
		if string(plain) != string(secret) {
			t.Fatalf("round trip %d restored %q, want %q", index, string(plain), string(secret))
		}
	}
}
