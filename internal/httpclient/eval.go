package httpclient

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Pattern: "http google.com", "http https://google.com", "curl google.com", "curl https://example.com"
var httpClientPattern = regexp.MustCompile(`(?i)^(?:http|curl)\s+(.+)$`)

// IsHTTPClientExpression checks if an expression is an HTTP HEAD request expression
func IsHTTPClientExpression(expr string) bool {
	expr = strings.TrimSpace(expr)
	if !httpClientPattern.MatchString(expr) {
		return false
	}

	// Extract the URL part and validate it looks like a URL/hostname
	matches := httpClientPattern.FindStringSubmatch(expr)
	if matches == nil {
		return false
	}

	target := strings.TrimSpace(matches[1])

	// Reject if it looks like an HTTP status code expression (e.g., "http code 404")
	if matched, _ := regexp.MatchString(`(?i)^(?:code|status)\s+\d{3}$`, target); matched {
		return false
	}

	// Must look like a hostname or URL
	if isValidTarget(target) {
		return true
	}

	return false
}

// isValidTarget checks if the target looks like a valid hostname or URL
func isValidTarget(target string) bool {
	// If it has a scheme, parse as URL
	if strings.HasPrefix(strings.ToLower(target), "http://") || strings.HasPrefix(strings.ToLower(target), "https://") {
		u, err := url.Parse(target)
		if err != nil {
			return false
		}
		return u.Host != ""
	}

	// Otherwise treat as hostname - must contain at least one dot or be localhost
	if target == "localhost" || strings.HasPrefix(target, "localhost:") {
		return true
	}

	// Must have a dot (domain.tld) and no spaces
	if strings.Contains(target, " ") {
		return false
	}
	if strings.Contains(target, ".") {
		return true
	}

	return false
}

// EvalHTTPClient performs an HTTP HEAD request and returns the response headers
func EvalHTTPClient(expr string) (string, error) {
	expr = strings.TrimSpace(expr)

	matches := httpClientPattern.FindStringSubmatch(expr)
	if matches == nil {
		return "", fmt.Errorf("not an HTTP client expression")
	}

	target := strings.TrimSpace(matches[1])

	// Normalize URL
	urlStr := normalizeURL(target)

	// Perform HEAD request
	return doHeadRequest(urlStr)
}

// normalizeURL ensures the target has a scheme
func normalizeURL(target string) string {
	lower := strings.ToLower(target)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return target
	}
	// Default to https
	return "https://" + target
}

// doHeadRequest performs an HTTP HEAD request and formats the response
func doHeadRequest(urlStr string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 5 * time.Second,
		},
		// Don't follow redirects - show the actual response
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("HEAD", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}
	req.Header.Set("User-Agent", "SmartCalc/1.0")

	resp, err := client.Do(req)
	if err != nil {
		// If HTTPS fails, try HTTP
		if strings.HasPrefix(urlStr, "https://") {
			httpURL := "http://" + urlStr[8:]
			req2, err2 := http.NewRequest("HEAD", httpURL, nil)
			if err2 != nil {
				return "", fmt.Errorf("connection failed: %v", err)
			}
			req2.Header.Set("User-Agent", "SmartCalc/1.0")
			resp2, err2 := client.Do(req2)
			if err2 != nil {
				return "", fmt.Errorf("connection failed: %v", err)
			}
			defer resp2.Body.Close()
			return formatResponse(resp2), nil
		}
		return "", fmt.Errorf("connection failed: %v", err)
	}
	defer resp.Body.Close()

	return formatResponse(resp), nil
}

// formatResponse formats the HTTP response headers like curl -I output
func formatResponse(resp *http.Response) string {
	var lines []string

	// Status line
	lines = append(lines, fmt.Sprintf("> %s %s", resp.Proto, resp.Status))

	// Collect and sort header names for consistent output
	// But put important headers first
	priorityHeaders := []string{
		"Location",
		"Content-Type",
		"Content-Length",
		"Server",
		"Date",
		"Cache-Control",
		"Expires",
		"Last-Modified",
		"ETag",
	}

	seen := make(map[string]bool)

	// Output priority headers first (if present)
	for _, name := range priorityHeaders {
		values := resp.Header.Values(name)
		if len(values) > 0 {
			seen[strings.ToLower(name)] = true
			for _, v := range values {
				lines = append(lines, fmt.Sprintf("> %s: %s", name, v))
			}
		}
	}

	// Output remaining headers alphabetically
	var remaining []string
	for name := range resp.Header {
		if !seen[strings.ToLower(name)] {
			remaining = append(remaining, name)
		}
	}
	sort.Strings(remaining)

	for _, name := range remaining {
		for _, v := range resp.Header.Values(name) {
			lines = append(lines, fmt.Sprintf("> %s: %s", name, v))
		}
	}

	return strings.Join(lines, "\n")
}
