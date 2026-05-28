//go:build !windows

package dbconnector

import "errors"

func protectSecret(data []byte) ([]byte, error) {
	_ = data
	return nil, errors.New("protected secret storage is not implemented on this platform; configure an OS keychain backend before saving connector credentials")
}

func unprotectSecret(data []byte) ([]byte, error) {
	_ = data
	return nil, errors.New("protected secret storage is not implemented on this platform")
}
