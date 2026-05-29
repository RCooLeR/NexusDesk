//go:build windows

package protectedsecret

import "testing"

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
