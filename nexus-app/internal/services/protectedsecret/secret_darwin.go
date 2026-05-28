//go:build darwin

package protectedsecret

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

const darwinBackend = "macos-keychain"

var securityCommand = "security"

func Protect(purpose string, data []byte) ([]byte, error) {
	token, err := newToken(darwinBackend, purpose)
	if err != nil {
		return nil, err
	}
	if err := runSecurity("add-generic-password", "-a", token.Account, "-s", serviceName(token.Purpose), "-w", string(data), "-U"); err != nil {
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
	out, err := exec.Command(securityCommand, "find-generic-password", "-a", token.Account, "-s", serviceName(token.Purpose), "-w").Output()
	if err != nil {
		return nil, fmt.Errorf("read secret from macOS Keychain: %w", err)
	}
	return bytes.TrimRight(out, "\r\n"), nil
}

func Delete(data []byte) error {
	token, ok, err := decodeToken(data)
	if err != nil || !ok || token.Backend != darwinBackend {
		return err
	}
	if err := runSecurity("delete-generic-password", "-a", token.Account, "-s", serviceName(token.Purpose)); err != nil {
		return fmt.Errorf("delete secret from macOS Keychain: %w", err)
	}
	return nil
}

func Available() bool {
	_, err := exec.LookPath(securityCommand)
	return err == nil
}

func runSecurity(args ...string) error {
	cmd := exec.Command(securityCommand, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}

func serviceName(purpose string) string {
	return "NexusDesk " + purpose
}
