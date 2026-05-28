package dbconnector

import "nexusdesk/internal/services/protectedsecret"

const connectorCredentialSecretPurpose = "connector.profile-credential"

func protectSecret(data []byte) ([]byte, error) {
	return protectedsecret.Protect(connectorCredentialSecretPurpose, data)
}

func unprotectSecret(data []byte) ([]byte, error) {
	return protectedsecret.Unprotect(data)
}

func deleteProtectedSecret(data []byte) error {
	return protectedsecret.Delete(data)
}
