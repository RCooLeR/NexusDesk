//go:build !windows

package settings

import "errors"

func protectSecret(data []byte) ([]byte, error) {
	_ = data
	return nil, errors.New("protected secret storage is not implemented on this platform; configure an OS keychain backend before saving API keys")
}

func unprotectSecret(data []byte) ([]byte, error) {
	_ = data
	return nil, errors.New("protected secret storage is not implemented on this platform")
}
