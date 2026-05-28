package protectedsecret

import "testing"

func TestKeychainTokenRoundTrip(t *testing.T) {
	token := keychainToken{
		Backend: "test-backend",
		Purpose: "settings.api-key",
		Account: "account-123",
	}
	decoded, ok, err := decodeToken(encodeToken(token))
	if err != nil {
		t.Fatalf("decodeToken returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected encoded token to be recognized")
	}
	if decoded != token {
		t.Fatalf("token mismatch: got %#v want %#v", decoded, token)
	}
}

func TestDecodeTokenIgnoresLegacyProtectedBlob(t *testing.T) {
	_, ok, err := decodeToken([]byte("legacy-or-dpapi-bytes"))
	if err != nil {
		t.Fatalf("decodeToken returned error: %v", err)
	}
	if ok {
		t.Fatal("expected non-token data to be ignored")
	}
}
