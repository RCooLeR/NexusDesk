//go:build !windows && !darwin && !linux

package protectedsecret

import "errors"

func Protect(purpose string, data []byte) ([]byte, error) {
	_, _ = purpose, data
	return nil, errors.New("protected secret storage is not implemented on this platform")
}

func Unprotect(data []byte) ([]byte, error) {
	_ = data
	return nil, errors.New("protected secret storage is not implemented on this platform")
}

func Delete(data []byte) error {
	_ = data
	return nil
}

func Available() bool {
	return false
}
