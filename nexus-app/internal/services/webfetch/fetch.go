// Package webfetch fetches bounded text-like HTTP(S) content for approved agent use.
package webfetch

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	defaultMaxBytes = 128 * 1024
	maxBytesLimit   = 512 * 1024
	defaultTimeout  = 12 * time.Second
	maxRedirects    = 3
)

var resolveHostIPs = defaultResolveHostIPs

type Request struct {
	URL            string
	AllowedDomains []string
	AllowLocal     bool
	MaxBytes       int
	Timeout        time.Duration
}

type Result struct {
	URL         string
	FinalURL    string
	Status      int
	ContentType string
	Title       string
	Text        string
	BytesRead   int
	Truncated   bool
	Redirects   int
	Message     string
}

func Fetch(ctx context.Context, request Request) (Result, error) {
	targetURL, err := validateURL(request.URL, request.AllowedDomains, request.AllowLocal)
	if err != nil {
		return Result{}, err
	}
	maxBytes := request.MaxBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxBytes
	}
	if maxBytes > maxBytesLimit {
		maxBytes = maxBytesLimit
	}
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	fetchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	redirects := 0
	client := &http.Client{
		Timeout:   timeout,
		Transport: guardedTransport(request.AllowLocal),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			redirects = len(via)
			if len(via) >= maxRedirects {
				return errors.New("web fetch redirect limit reached")
			}
			if _, err := validateURL(req.URL.String(), request.AllowedDomains, request.AllowLocal); err != nil {
				return err
			}
			return nil
		},
	}
	httpRequest, err := http.NewRequestWithContext(fetchCtx, http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return Result{}, err
	}
	httpRequest.Header.Set("User-Agent", "NexusDesk/agent-web-fetch")
	httpRequest.Header.Set("Accept", "text/html,text/plain,application/json,application/xml,text/xml,text/csv,text/markdown,text/yaml,application/yaml;q=0.9,*/*;q=0.1")

	response, err := client.Do(httpRequest)
	if err != nil {
		return Result{}, err
	}
	defer response.Body.Close()

	contentType := response.Header.Get("Content-Type")
	if !isTextLikeContentType(contentType) {
		return Result{}, fmt.Errorf("web fetch only accepts text-like responses, got %q", contentType)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(maxBytes)+1))
	if err != nil {
		return Result{}, err
	}
	truncated := len(body) > maxBytes
	if truncated {
		body = body[:maxBytes]
	}
	text := string(body)
	if !utf8.ValidString(text) {
		text = strings.ToValidUTF8(text, "")
	}
	text = normalizeText(contentType, text)
	title := extractTitle(text)
	result := Result{
		URL:         targetURL.String(),
		FinalURL:    response.Request.URL.String(),
		Status:      response.StatusCode,
		ContentType: contentType,
		Title:       title,
		Text:        strings.TrimSpace(text),
		BytesRead:   len(body),
		Truncated:   truncated,
		Redirects:   redirects,
	}
	result.Message = fmt.Sprintf("Fetched %s with status %d (%d bytes).", result.FinalURL, result.Status, result.BytesRead)
	if result.Truncated {
		result.Message += " Response was truncated."
	}
	return result, nil
}

func validateURL(rawURL string, allowedDomains []string, allowLocal bool) (*url.URL, error) {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil, errors.New("URL is required")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("web fetch URL must use http or https")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return nil, errors.New("web fetch URL must include a host")
	}
	if len(allowedDomains) > 0 && !hostAllowed(host, allowedDomains) {
		return nil, errors.New("web fetch host is outside the allowed domains")
	}
	if !allowLocal {
		if err := rejectPrivateHost(host); err != nil {
			return nil, err
		}
	}
	return parsed, nil
}

func hostAllowed(host string, allowedDomains []string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	for _, domain := range allowedDomains {
		domain = strings.ToLower(strings.Trim(strings.TrimSpace(domain), "."))
		if domain == "" {
			continue
		}
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func rejectPrivateHost(host string) error {
	if strings.EqualFold(host, "localhost") {
		return errors.New("web fetch blocks localhost unless allowLocal is enabled")
	}
	ips, err := resolveHostIPs(context.Background(), host)
	if err != nil {
		return err
	}
	for _, ip := range ips {
		if blockedLocalIP(ip) {
			return errors.New("web fetch blocks private, loopback, link-local, and unspecified hosts unless allowLocal is enabled")
		}
	}
	return nil
}

func guardedTransport(allowLocal bool) *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	dialer := &net.Dialer{Timeout: defaultTimeout, KeepAlive: 30 * time.Second}
	transport.DialContext = func(ctx context.Context, network string, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		ips, err := resolveHostIPs(ctx, host)
		if err != nil {
			return nil, err
		}
		var blocked []string
		for _, ip := range ips {
			if ip == nil {
				continue
			}
			if !allowLocal && blockedLocalIP(ip) {
				blocked = append(blocked, ip.String())
				continue
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		}
		if len(blocked) > 0 {
			return nil, fmt.Errorf("web fetch blocks private, loopback, link-local, multicast, and unspecified dial targets unless allowLocal is enabled: %s", strings.Join(blocked, ", "))
		}
		return nil, fmt.Errorf("web fetch could not resolve a usable address for %s", net.JoinHostPort(host, port))
	}
	return transport
}

func defaultResolveHostIPs(ctx context.Context, host string) ([]net.IP, error) {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "" {
		return nil, errors.New("host is required")
	}
	if ip := net.ParseIP(host); ip != nil {
		return []net.IP{ip}, nil
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	ips := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if address.IP != nil {
			ips = append(ips, address.IP)
		}
	}
	return ips, nil
}

func blockedLocalIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}

func isTextLikeContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		mediaType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}
	mediaType = strings.ToLower(mediaType)
	if strings.HasPrefix(mediaType, "text/") {
		return true
	}
	switch mediaType {
	case "application/json", "application/xml", "application/xhtml+xml", "application/rss+xml", "application/atom+xml", "application/yaml", "application/x-yaml":
		return true
	default:
		return strings.HasSuffix(mediaType, "+json") || strings.HasSuffix(mediaType, "+xml")
	}
}

func normalizeText(contentType string, text string) string {
	mediaType, _, _ := mime.ParseMediaType(contentType)
	mediaType = strings.ToLower(mediaType)
	if mediaType == "text/html" || mediaType == "application/xhtml+xml" {
		text = stripHTML(text)
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

func stripHTML(value string) string {
	value = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?i)</(p|div|section|article|header|footer|li|h[1-6])>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(value, " ")
	return html.UnescapeString(value)
}

func extractTitle(text string) string {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if len(line) > 120 {
				return line[:120] + "..."
			}
			return line
		}
	}
	return ""
}
