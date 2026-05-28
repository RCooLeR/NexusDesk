package protectedsecret

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"strings"
)

const tokenPrefix = "nexusdesk-secret-v1:"

type keychainToken struct {
	Backend string
	Purpose string
	Account string
}

func newToken(backend string, purpose string) (keychainToken, error) {
	purpose = strings.TrimSpace(purpose)
	if purpose == "" {
		return keychainToken{}, errors.New("secret purpose is required")
	}
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return keychainToken{}, err
	}
	return keychainToken{
		Backend: backend,
		Purpose: purpose,
		Account: base64.RawURLEncoding.EncodeToString(random),
	}, nil
}

func encodeToken(token keychainToken) []byte {
	parts := []string{
		tokenPrefix,
		base64.RawURLEncoding.EncodeToString([]byte(token.Backend)),
		":",
		base64.RawURLEncoding.EncodeToString([]byte(token.Purpose)),
		":",
		base64.RawURLEncoding.EncodeToString([]byte(token.Account)),
	}
	return []byte(strings.Join(parts, ""))
}

func decodeToken(data []byte) (keychainToken, bool, error) {
	value := strings.TrimSpace(string(data))
	if !strings.HasPrefix(value, tokenPrefix) {
		return keychainToken{}, false, nil
	}
	fields := strings.Split(strings.TrimPrefix(value, tokenPrefix), ":")
	if len(fields) != 3 {
		return keychainToken{}, true, errors.New("invalid protected secret token")
	}
	backend, err := decodeTokenField(fields[0])
	if err != nil {
		return keychainToken{}, true, err
	}
	purpose, err := decodeTokenField(fields[1])
	if err != nil {
		return keychainToken{}, true, err
	}
	account, err := decodeTokenField(fields[2])
	if err != nil {
		return keychainToken{}, true, err
	}
	if backend == "" || purpose == "" || account == "" {
		return keychainToken{}, true, errors.New("invalid protected secret token")
	}
	return keychainToken{Backend: backend, Purpose: purpose, Account: account}, true, nil
}

func decodeTokenField(value string) (string, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
