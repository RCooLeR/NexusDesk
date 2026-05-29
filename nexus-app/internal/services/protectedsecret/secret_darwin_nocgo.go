//go:build darwin && !cgo

package protectedsecret

import "errors"

type nativeDarwinKeychain struct{}

func (nativeDarwinKeychain) Store(service string, account string, secret []byte) error {
	return errors.New("macOS Keychain storage requires cgo")
}

func (nativeDarwinKeychain) Lookup(service string, account string) ([]byte, error) {
	return nil, errors.New("macOS Keychain storage requires cgo")
}

func (nativeDarwinKeychain) Delete(service string, account string) error {
	return errors.New("macOS Keychain storage requires cgo")
}

func (nativeDarwinKeychain) Available() bool {
	return false
}
