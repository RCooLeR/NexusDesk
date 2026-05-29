//go:build windows

package protectedsecret

import (
	"errors"
	"runtime"
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

func Protect(purpose string, data []byte) (protected []byte, err error) {
	_ = purpose
	in := bytesToBlob(data)
	var out dataBlob
	defer func() {
		if freeErr := freeLocalData(out.pbData); err == nil && freeErr != nil {
			err = freeErr
		}
	}()
	result, _, callErr := procCryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	runtime.KeepAlive(data)
	if result == 0 {
		return nil, windowsSecretError("CryptProtectData", callErr)
	}
	return blobToBytes(out), nil
}

func Unprotect(data []byte) (plain []byte, err error) {
	in := bytesToBlob(data)
	var out dataBlob
	defer func() {
		if freeErr := freeLocalData(out.pbData); err == nil && freeErr != nil {
			err = freeErr
		}
	}()
	result, _, callErr := procCryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	runtime.KeepAlive(data)
	if result == 0 {
		return nil, windowsSecretError("CryptUnprotectData", callErr)
	}
	return blobToBytes(out), nil
}

func Delete(data []byte) error {
	_ = data
	return nil
}

func Available() bool {
	return true
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

func freeLocalData(pointer *byte) error {
	if pointer == nil {
		return nil
	}
	result, _, callErr := procLocalFree.Call(uintptr(unsafe.Pointer(pointer)))
	if result != 0 {
		return windowsSecretError("LocalFree", callErr)
	}
	return nil
}

func windowsSecretError(operation string, err error) error {
	if err != nil && !errors.Is(err, syscall.Errno(0)) {
		return err
	}
	return errors.New(operation + " failed")
}
