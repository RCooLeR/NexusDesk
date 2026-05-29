//go:build darwin

package protectedsecret

import (
	"errors"
	"fmt"
)

const darwinBackend = "macos-keychain"

var darwinKeychainStore darwinKeychain = nativeDarwinKeychain{}

type darwinKeychain interface {
	Store(service string, account string, secret []byte) error
	Lookup(service string, account string) ([]byte, error)
	Delete(service string, account string) error
	Available() bool
}

func Protect(purpose string, data []byte) ([]byte, error) {
	token, err := newToken(darwinBackend, purpose)
	if err != nil {
		return nil, err
	}
	if err := darwinKeychainStore.Store(serviceName(token.Purpose), token.Account, data); err != nil {
		return nil, fmt.Errorf("store secret in macOS Keychain: %w", err)
	}
	return encodeToken(token), nil
}

func Unprotect(data []byte) ([]byte, error) {
	token, ok, err := decodeToken(data)
	if err != nil {
		return nil, err
	}
	if !ok || token.Backend != darwinBackend {
		return nil, errors.New("protected secret is not a macOS Keychain token")
	}
	out, err := darwinKeychainStore.Lookup(serviceName(token.Purpose), token.Account)
	if err != nil {
		return nil, fmt.Errorf("read secret from macOS Keychain: %w", err)
	}
	return out, nil
}

func Delete(data []byte) error {
	token, ok, err := decodeToken(data)
	if err != nil || !ok || token.Backend != darwinBackend {
		return err
	}
	if err := darwinKeychainStore.Delete(serviceName(token.Purpose), token.Account); err != nil {
		return fmt.Errorf("delete secret from macOS Keychain: %w", err)
	}
	return nil
}

func Available() bool {
	return darwinKeychainStore.Available()
}

func serviceName(purpose string) string {
	return "NexusDesk " + purpose
}
