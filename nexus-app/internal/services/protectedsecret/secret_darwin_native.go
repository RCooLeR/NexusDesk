//go:build darwin && cgo

package protectedsecret

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>
#include <stdlib.h>
#include <string.h>

static void secure_free(void *p, size_t n) {
	if (p == NULL) {
		return;
	}
	volatile unsigned char *bytes = (volatile unsigned char *)p;
	for (size_t i = 0; i < n; i++) {
		bytes[i] = 0;
	}
	free(p);
}

static SecKeychainRef default_keychain_ref(void) {
	return NULL;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type nativeDarwinKeychain struct{}

func (nativeDarwinKeychain) Store(service string, account string, secret []byte) error {
	serviceCString := C.CString(service)
	defer C.free(unsafe.Pointer(serviceCString))
	accountCString := C.CString(account)
	defer C.free(unsafe.Pointer(accountCString))
	secretPointer, secretLength := cSecretBytes(secret)
	defer secureFree(secretPointer, secretLength)

	status := C.SecKeychainAddGenericPassword(
		C.default_keychain_ref(),
		C.UInt32(len(service)),
		serviceCString,
		C.UInt32(len(account)),
		accountCString,
		C.UInt32(secretLength),
		secretPointer,
		nil,
	)
	if status == C.errSecDuplicateItem {
		return updateGenericPassword(serviceCString, len(service), accountCString, len(account), secretPointer, secretLength)
	}
	return keychainStatusError("SecKeychainAddGenericPassword", status)
}

func (nativeDarwinKeychain) Lookup(service string, account string) ([]byte, error) {
	serviceCString := C.CString(service)
	defer C.free(unsafe.Pointer(serviceCString))
	accountCString := C.CString(account)
	defer C.free(unsafe.Pointer(accountCString))

	var passwordLength C.UInt32
	var passwordData unsafe.Pointer
	status := C.SecKeychainFindGenericPassword(
		C.default_keychain_ref(),
		C.UInt32(len(service)),
		serviceCString,
		C.UInt32(len(account)),
		accountCString,
		&passwordLength,
		&passwordData,
		nil,
	)
	if err := keychainStatusError("SecKeychainFindGenericPassword", status); err != nil {
		return nil, err
	}
	defer C.SecKeychainItemFreeContent(nil, passwordData)

	if passwordData == nil || passwordLength == 0 {
		return nil, nil
	}
	return C.GoBytes(passwordData, C.int(passwordLength)), nil
}

func (nativeDarwinKeychain) Delete(service string, account string) error {
	serviceCString := C.CString(service)
	defer C.free(unsafe.Pointer(serviceCString))
	accountCString := C.CString(account)
	defer C.free(unsafe.Pointer(accountCString))

	var item C.SecKeychainItemRef
	status := C.SecKeychainFindGenericPassword(
		C.default_keychain_ref(),
		C.UInt32(len(service)),
		serviceCString,
		C.UInt32(len(account)),
		accountCString,
		nil,
		nil,
		&item,
	)
	if err := keychainStatusError("SecKeychainFindGenericPassword", status); err != nil {
		return err
	}
	defer C.CFRelease(C.CFTypeRef(item))
	return keychainStatusError("SecKeychainItemDelete", C.SecKeychainItemDelete(item))
}

func (nativeDarwinKeychain) Available() bool {
	return true
}

func updateGenericPassword(service *C.char, serviceLength int, account *C.char, accountLength int, secret unsafe.Pointer, secretLength uintptr) error {
	var item C.SecKeychainItemRef
	status := C.SecKeychainFindGenericPassword(
		C.default_keychain_ref(),
		C.UInt32(serviceLength),
		service,
		C.UInt32(accountLength),
		account,
		nil,
		nil,
		&item,
	)
	if err := keychainStatusError("SecKeychainFindGenericPassword", status); err != nil {
		return err
	}
	defer C.CFRelease(C.CFTypeRef(item))

	status = C.SecKeychainItemModifyAttributesAndData(item, nil, C.UInt32(secretLength), secret)
	return keychainStatusError("SecKeychainItemModifyAttributesAndData", status)
}

func cSecretBytes(secret []byte) (unsafe.Pointer, uintptr) {
	if len(secret) == 0 {
		return nil, 0
	}
	return C.CBytes(secret), uintptr(len(secret))
}

func secureFree(pointer unsafe.Pointer, length uintptr) {
	C.secure_free(pointer, C.size_t(length))
}

func keychainStatusError(operation string, status C.OSStatus) error {
	if status == C.errSecSuccess {
		return nil
	}
	return fmt.Errorf("%s failed with OSStatus %d", operation, int32(status))
}
