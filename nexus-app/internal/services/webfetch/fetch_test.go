package webfetch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchRejectsNonHTTP(t *testing.T) {
	_, err := Fetch(context.Background(), Request{URL: "file:///etc/passwd", AllowLocal: true})
	if err == nil || !strings.Contains(err.Error(), "http or https") {
		t.Fatalf("expected scheme rejection, got %v", err)
	}
}

func TestFetchBlocksLocalByDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		_, _ = response.Write([]byte("hello"))
	}))
	defer server.Close()

	_, err := Fetch(context.Background(), Request{URL: server.URL})
	if err == nil || !strings.Contains(err.Error(), "blocks private") {
		t.Fatalf("expected local host rejection, got %v", err)
	}
}

func TestFetchReadsTextWhenLocalAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = response.Write([]byte("<html><body><h1>Hello</h1><script>secret()</script><p>World</p></body></html>"))
	}))
	defer server.Close()

	result, err := Fetch(context.Background(), Request{URL: server.URL, AllowLocal: true})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if !strings.Contains(result.Text, "Hello") || !strings.Contains(result.Text, "World") || strings.Contains(result.Text, "secret()") {
		t.Fatalf("unexpected normalized text: %q", result.Text)
	}
}

func TestFetchEnforcesAllowedDomains(t *testing.T) {
	_, err := Fetch(context.Background(), Request{
		URL:            "https://example.com",
		AllowedDomains: []string{"openai.com"},
		AllowLocal:     true,
	})
	if err == nil || !strings.Contains(err.Error(), "allowed domains") {
		t.Fatalf("expected allowed-domain rejection, got %v", err)
	}
}

func TestFetchRejectsBinaryContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.Header().Set("Content-Type", "application/octet-stream")
		_, _ = response.Write([]byte{0, 1, 2})
	}))
	defer server.Close()

	_, err := Fetch(context.Background(), Request{URL: server.URL, AllowLocal: true})
	if err == nil || !strings.Contains(err.Error(), "text-like") {
		t.Fatalf("expected content type rejection, got %v", err)
	}
}
