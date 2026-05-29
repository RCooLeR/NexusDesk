//go:build linux

package protectedsecret

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
)

const linuxBackend = "linux-secret-service"

var secretToolCommand = "secret-tool"

func Protect(purpose string, data []byte) ([]byte, error) {
	if _, err := exec.LookPath(secretToolCommand); err != nil {
		return nil, fmt.Errorf("protected secret storage is unavailable: %s was not found in PATH", secretToolCommand)
	}
	token, err := newToken(linuxBackend, purpose)
	if err != nil {
		return nil, err
	}
	if err := runSecretToolWithInput(data, "store", "--label", "NexusDesk "+token.Purpose, "application", "NexusDesk", "purpose", token.Purpose, "account", token.Account); err != nil {
		return nil, fmt.Errorf("store secret in Linux Secret Service: %w", err)
	}
	return encodeToken(token), nil
}

func Unprotect(data []byte) ([]byte, error) {
	if _, err := exec.LookPath(secretToolCommand); err != nil {
		return nil, fmt.Errorf("protected secret storage is unavailable: %s was not found in PATH", secretToolCommand)
	}
	token, ok, err := decodeToken(data)
	if err != nil {
		return nil, err
	}
	if !ok || token.Backend != linuxBackend {
		return nil, errors.New("protected secret is not a Linux Secret Service token")
	}
	out, err := exec.Command(secretToolCommand, "lookup", "application", "NexusDesk", "purpose", token.Purpose, "account", token.Account).Output()
	if err != nil {
		return nil, fmt.Errorf("read secret from Linux Secret Service: %w", err)
	}
	return bytes.TrimRight(out, "\r\n"), nil
}

func Delete(data []byte) error {
	if _, err := exec.LookPath(secretToolCommand); err != nil {
		if len(bytes.TrimSpace(data)) == 0 || errors.Is(err, os.ErrNotExist) || errors.Is(err, exec.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("protected secret storage is unavailable: %s was not found in PATH", secretToolCommand)
	}
	token, ok, err := decodeToken(data)
	if err != nil || !ok || token.Backend != linuxBackend {
		return err
	}
	if err := runSecretToolWithInput(nil, "clear", "application", "NexusDesk", "purpose", token.Purpose, "account", token.Account); err != nil {
		return fmt.Errorf("delete secret from Linux Secret Service: %w", err)
	}
	return nil
}

func Available() bool {
	_, err := exec.LookPath(secretToolCommand)
	return err == nil
}

func runSecretToolWithInput(input []byte, args ...string) error {
	cmd := exec.Command(secretToolCommand, args...)
	if input != nil {
		cmd.Stdin = bytes.NewReader(input)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, bytes.TrimSpace(out))
	}
	return nil
}
