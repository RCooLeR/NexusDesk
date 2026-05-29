//go:build darwin

package protectedsecret

import "testing"

func TestDarwinKeychainRoundTripUsesNativeSecretBytes(t *testing.T) {
	store := newFakeDarwinKeychain()
	oldStore := darwinKeychainStore
	darwinKeychainStore = store
	t.Cleanup(func() {
		darwinKeychainStore = oldStore
	})

	token, err := Protect("settings.api-key", []byte("darwin-secret\n"))
	if err != nil {
		t.Fatalf("Protect returned error: %v", err)
	}
	if store.lastStoredSecret != "darwin-secret\n" {
		t.Fatalf("expected secret to be passed as private bytes, got %q", store.lastStoredSecret)
	}
	plain, err := Unprotect(token)
	if err != nil {
		t.Fatalf("Unprotect returned error: %v", err)
	}
	if string(plain) != "darwin-secret\n" {
		t.Fatalf("expected restored secret, got %q", string(plain))
	}
	if err := Delete(token); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if _, err := Unprotect(token); err == nil {
		t.Fatal("expected lookup to fail after Delete")
	}
}

type fakeDarwinKeychain struct {
	values           map[string][]byte
	lastStoredSecret string
}

func newFakeDarwinKeychain() *fakeDarwinKeychain {
	return &fakeDarwinKeychain{values: map[string][]byte{}}
}

func (f *fakeDarwinKeychain) Store(service string, account string, secret []byte) error {
	key := service + "\x00" + account
	f.lastStoredSecret = string(secret)
	f.values[key] = append([]byte{}, secret...)
	return nil
}

func (f *fakeDarwinKeychain) Lookup(service string, account string) ([]byte, error) {
	key := service + "\x00" + account
	value, ok := f.values[key]
	if !ok {
		return nil, errFakeDarwinKeychainMissing
	}
	return append([]byte{}, value...), nil
}

func (f *fakeDarwinKeychain) Delete(service string, account string) error {
	key := service + "\x00" + account
	delete(f.values, key)
	return nil
}

func (f *fakeDarwinKeychain) Available() bool {
	return true
}

type fakeDarwinKeychainMissingError struct{}

func (fakeDarwinKeychainMissingError) Error() string {
	return "secret not found"
}

var errFakeDarwinKeychainMissing error = fakeDarwinKeychainMissingError{}
