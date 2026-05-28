package settings

import "nexusdesk/internal/services/protectedsecret"

const apiKeySecretPurpose = "settings.api-key"

func protectSecret(data []byte) ([]byte, error) {
	return protectedsecret.Protect(apiKeySecretPurpose, data)
}

func unprotectSecret(data []byte) ([]byte, error) {
	return protectedsecret.Unprotect(data)
}

func deleteProtectedSecret(data []byte) error {
	return protectedsecret.Delete(data)
}
