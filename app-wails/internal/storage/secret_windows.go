//go:build windows

package storage

import (
	"errors"
	"syscall"
	"unsafe"
)

var (
	crypt32                = syscall.NewLazyDLL("crypt32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	procCryptProtectData   = crypt32.NewProc("CryptProtectData")
	procCryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
	procLocalFree          = kernel32.NewProc("LocalFree")
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func protectSecret(data []byte) ([]byte, error) {
	in := bytesToBlob(data)
	var out dataBlob
	result, _, callErr := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if result == 0 {
		return nil, windowsSecretError("CryptProtectData", callErr)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return blobToBytes(out), nil
}

func unprotectSecret(data []byte) ([]byte, error) {
	in := bytesToBlob(data)
	var out dataBlob
	result, _, callErr := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if result == 0 {
		return nil, windowsSecretError("CryptUnprotectData", callErr)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	return blobToBytes(out), nil
}

func bytesToBlob(data []byte) dataBlob {
	if len(data) == 0 {
		return dataBlob{}
	}
	return dataBlob{cbData: uint32(len(data)), pbData: &data[0]}
}

func blobToBytes(blob dataBlob) []byte {
	if blob.pbData == nil || blob.cbData == 0 {
		return nil
	}
	data := unsafe.Slice(blob.pbData, int(blob.cbData))
	out := make([]byte, len(data))
	copy(out, data)
	return out
}

func windowsSecretError(operation string, err error) error {
	if err != nil && !errors.Is(err, syscall.Errno(0)) {
		return err
	}
	return errors.New(operation + " failed")
}
