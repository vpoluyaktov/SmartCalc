package nettools

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"
)

// HTTPTrace performs an HTTP-based traceroute to the specified host and port
// It shows the timing breakdown of the HTTP request
func HTTPTrace(host string, port int) (string, error) {
	// Build URL
	scheme := "https"
	if port == 80 {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%d/", scheme, host, port)
	
	// Timing variables
	var dnsStart, dnsEnd time.Time
	var connectStart, connectEnd time.Time
	var tlsStart, tlsEnd time.Time
	var reqStart, reqEnd time.Time
	var firstByteTime time.Time
	
	var remoteAddr string
	
	// Create trace
	trace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			dnsEnd = time.Now()
			if len(info.Addrs) > 0 {
				remoteAddr = info.Addrs[0].IP.String()
			}
		},
		ConnectStart: func(network, addr string) {
			connectStart = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			connectEnd = time.Now()
			if remoteAddr == "" {
				host, _, _ := net.SplitHostPort(addr)
				remoteAddr = host
			}
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tlsEnd = time.Now()
		},
		GotFirstResponseByte: func() {
			firstByteTime = time.Now()
		},
	}
	
	// Create request with trace
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %v", err)
	}
	req.Header.Set("User-Agent", "SmartCalc/1.0")
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	
	// Create client
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
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	
	// Perform request
	reqStart = time.Now()
	resp, err := client.Do(req)
	if err != nil {
		// If HTTPS fails, try HTTP
		if scheme == "https" {
			httpURL := fmt.Sprintf("http://%s:%d/", host, port)
			req2, err2 := http.NewRequest("HEAD", httpURL, nil)
			if err2 != nil {
				return "", fmt.Errorf("connection failed: %v", err)
			}
			req2.Header.Set("User-Agent", "SmartCalc/1.0")
			
			// Reset trace for HTTP attempt
			dnsStart, dnsEnd = time.Time{}, time.Time{}
			connectStart, connectEnd = time.Time{}, time.Time{}
			tlsStart, tlsEnd = time.Time{}, time.Time{}
			remoteAddr = ""
			
			req2 = req2.WithContext(httptrace.WithClientTrace(req2.Context(), trace))
			reqStart = time.Now()
			resp2, err2 := client.Do(req2)
			if err2 != nil {
				return "", fmt.Errorf("connection failed: %v", err)
			}
			defer resp2.Body.Close()
			reqEnd = time.Now()
			scheme = "http"
			resp = resp2
		} else {
			return "", fmt.Errorf("connection failed: %v", err)
		}
	} else {
		defer resp.Body.Close()
		reqEnd = time.Now()
	}
	
	// Calculate durations
	var dnsTime, connectTime, tlsTime, waitTime, totalTime time.Duration
	
	if !dnsStart.IsZero() && !dnsEnd.IsZero() {
		dnsTime = dnsEnd.Sub(dnsStart)
	}
	
	if !connectStart.IsZero() && !connectEnd.IsZero() {
		connectTime = connectEnd.Sub(connectStart)
	}
	
	if !tlsStart.IsZero() && !tlsEnd.IsZero() {
		tlsTime = tlsEnd.Sub(tlsStart)
	}
	
	if !firstByteTime.IsZero() {
		if !tlsEnd.IsZero() {
			waitTime = firstByteTime.Sub(tlsEnd)
		} else if !connectEnd.IsZero() {
			waitTime = firstByteTime.Sub(connectEnd)
		}
	}
	
	totalTime = reqEnd.Sub(reqStart)
	
	// Format output
	output := fmt.Sprintf("\n> HTTP Trace to %s:%d (%s)", host, port, scheme)
	if remoteAddr != "" {
		output += fmt.Sprintf("\n> Remote Address: %s", remoteAddr)
	}
	output += fmt.Sprintf("\n> HTTP Status: %s", resp.Status)
	output += fmt.Sprintf("\n> Timing breakdown:")
	
	if dnsTime > 0 {
		output += fmt.Sprintf("\n>   DNS Lookup:      %6dms", dnsTime.Milliseconds())
	}
	if connectTime > 0 {
		output += fmt.Sprintf("\n>   TCP Connect:     %6dms", connectTime.Milliseconds())
	}
	if tlsTime > 0 {
		output += fmt.Sprintf("\n>   TLS Handshake:   %6dms", tlsTime.Milliseconds())
	}
	if waitTime > 0 {
		output += fmt.Sprintf("\n>   Server Response: %6dms", waitTime.Milliseconds())
	}
	output += fmt.Sprintf("\n>   Total Time:      %6dms", totalTime.Milliseconds())
	
	return output, nil
}
