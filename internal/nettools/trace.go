package nettools

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// HTTPTrace performs an HTTP-based connection trace showing the path to the target
// Since true TTL-based traceroute requires raw sockets, this shows DNS resolution
// and connection details instead
func HTTPTrace(host string, port int) (string, error) {
	var output strings.Builder

	// Step 1: DNS Resolution
	output.WriteString(fmt.Sprintf("\n> Traceroute to %s:%d", host, port))
	output.WriteString("\n> ")

	start := time.Now()
	ips, err := net.LookupIP(host)
	dnsTime := time.Since(start)

	if err != nil {
		return "", fmt.Errorf("DNS resolution failed: %v", err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses found for %s", host)
	}

	targetIP := ips[0].String()
	output.WriteString(fmt.Sprintf("\n> 1  DNS Resolution: %s -> %s (%dms)", host, targetIP, dnsTime.Milliseconds()))

	// Step 2: Reverse DNS for target
	start = time.Now()
	names, err := net.LookupAddr(targetIP)
	rdnsTime := time.Since(start)
	hostname := targetIP
	if err == nil && len(names) > 0 {
		hostname = strings.TrimSuffix(names[0], ".")
		output.WriteString(fmt.Sprintf("\n> 2  Reverse DNS: %s -> %s (%dms)", targetIP, hostname, rdnsTime.Milliseconds()))
	} else {
		output.WriteString(fmt.Sprintf("\n> 2  Reverse DNS: %s (no PTR record)", targetIP))
	}

	// Step 3: TCP Connection
	scheme := "https"
	if port == 80 {
		scheme = "http"
	}

	output.WriteString(fmt.Sprintf("\n> 3  TCP Connect to %s:%d", targetIP, port))

	start = time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetIP, port), 5*time.Second)
	tcpTime := time.Since(start)

	if err != nil {
		output.WriteString(fmt.Sprintf(" - Failed (%dms): %v", tcpTime.Milliseconds(), err))
		return output.String(), nil
	}
	defer conn.Close()

	// Get local address
	localAddr := conn.LocalAddr().String()
	output.WriteString(fmt.Sprintf("\n>     Local: %s -> Remote: %s (%dms)", localAddr, conn.RemoteAddr(), tcpTime.Milliseconds()))

	// Step 4: HTTP Request
	url := fmt.Sprintf("%s://%s:%d/", scheme, host, port)
	output.WriteString(fmt.Sprintf("\n> 4  HTTP %s Request", strings.ToUpper(scheme)))

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	start = time.Now()
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		output.WriteString(fmt.Sprintf(" - Failed: %v", err))
		return output.String(), nil
	}
	req.Header.Set("User-Agent", "SmartCalc/1.0")

	resp, err := client.Do(req)
	httpTime := time.Since(start)

	if err != nil {
		// Try HTTP if HTTPS failed
		if scheme == "https" {
			httpURL := fmt.Sprintf("http://%s:%d/", host, port)
			req2, err2 := http.NewRequest("HEAD", httpURL, nil)
			if err2 == nil {
				req2.Header.Set("User-Agent", "SmartCalc/1.0")
				start = time.Now()
				resp2, err2 := client.Do(req2)
				httpTime = time.Since(start)
				if err2 == nil {
					defer resp2.Body.Close()
					output.WriteString(fmt.Sprintf("\n>     Status: %s (%dms)", resp2.Status, httpTime.Milliseconds()))
					output.WriteString("\n>     Note: Fell back to HTTP")
					return output.String(), nil
				}
			}
		}
		output.WriteString(fmt.Sprintf(" - Failed (%dms): %v", httpTime.Milliseconds(), err))
		return output.String(), nil
	}
	defer resp.Body.Close()

	output.WriteString(fmt.Sprintf("\n>     Status: %s (%dms)", resp.Status, httpTime.Milliseconds()))

	// Show any redirects
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		if location := resp.Header.Get("Location"); location != "" {
			output.WriteString(fmt.Sprintf("\n>     Redirect: %s", location))
		}
	}

	// Summary
	totalTime := dnsTime + rdnsTime + tcpTime + httpTime
	output.WriteString(fmt.Sprintf("\n> "))
	output.WriteString(fmt.Sprintf("\n> Total time: %dms", totalTime.Milliseconds()))

	return output.String(), nil
}
