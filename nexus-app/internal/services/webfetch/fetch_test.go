package webfetch

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

func TestFetchBlocksDNSRebindingAtDialTime(t *testing.T) {
	var calls atomic.Int32
	oldResolver := resolveHostIPs
	resolveHostIPs = func(ctx context.Context, host string) ([]net.IP, error) {
		if host != "rebind.test" {
			return oldResolver(ctx, host)
		}
		if calls.Add(1) == 1 {
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		}
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}
	t.Cleanup(func() {
		resolveHostIPs = oldResolver
	})

	_, err := Fetch(context.Background(), Request{URL: "http://rebind.test/"})
	if err == nil || !strings.Contains(err.Error(), "blocks private") {
		t.Fatalf("expected dial-time private target rejection, got %v", err)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected resolver to be called during validation and dial, got %d call(s)", calls.Load())
	}
}

func TestFetchBlocksMulticastAtDialTime(t *testing.T) {
	var calls atomic.Int32
	oldResolver := resolveHostIPs
	resolveHostIPs = func(ctx context.Context, host string) ([]net.IP, error) {
		if host != "multicast.test" {
			return oldResolver(ctx, host)
		}
		if calls.Add(1) == 1 {
			return []net.IP{net.ParseIP("93.184.216.34")}, nil
		}
		return []net.IP{net.ParseIP("224.0.0.1")}, nil
	}
	t.Cleanup(func() {
		resolveHostIPs = oldResolver
	})

	_, err := Fetch(context.Background(), Request{URL: "http://multicast.test/"})
	if err == nil || !strings.Contains(err.Error(), "blocks private") {
		t.Fatalf("expected dial-time multicast target rejection, got %v", err)
	}
}

func TestRedirectValidationBlocksPrivateTargets(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/private", nil)
	err := validateRedirectURL(request, nil, false)
	if err == nil || !strings.Contains(err.Error(), "blocks private") {
		t.Fatalf("expected private redirect target rejection, got %v", err)
	}
}

func TestBlockedLocalIPCoversPrivateAndSpecialRanges(t *testing.T) {
	for _, rawIP := range []string{
		"0.0.0.0",
		"10.1.2.3",
		"127.0.0.1",
		"169.254.10.20",
		"172.16.1.1",
		"192.168.1.10",
		"224.0.0.1",
		"::",
		"::1",
		"fc00::1",
		"fe80::1",
		"ff02::1",
	} {
		if !blockedLocalIP(net.ParseIP(rawIP)) {
			t.Fatalf("expected %s to be blocked", rawIP)
		}
	}
	if blockedLocalIP(net.ParseIP("93.184.216.34")) {
		t.Fatal("expected public IPv4 address to be allowed")
	}
	if blockedLocalIP(net.ParseIP("2606:2800:220:1:248:1893:25c8:1946")) {
		t.Fatal("expected public IPv6 address to be allowed")
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
