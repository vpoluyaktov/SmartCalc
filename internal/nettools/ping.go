package nettools

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

// HTTPPing performs an HTTP-based ping to the specified host and port
// It measures the response time and returns formatted results
func HTTPPing(host string, port int) (string, error) {
	// Build URL
	scheme := "https"
	if port == 80 {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%d/", scheme, host, port)

	// Perform multiple pings (10 attempts)
	const attempts = 10
	var results []time.Duration
	var failed int

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

	for i := 0; i < attempts; i++ {
		start := time.Now()

		req, err := http.NewRequest("HEAD", url, nil)
		if err != nil {
			failed++
			continue
		}
		req.Header.Set("User-Agent", "SmartCalc/1.0")

		resp, err := client.Do(req)
		if err != nil {
			// If HTTPS fails on first attempt, try HTTP
			if i == 0 && scheme == "https" {
				httpURL := fmt.Sprintf("http://%s:%d/", host, port)
				req2, err2 := http.NewRequest("HEAD", httpURL, nil)
				if err2 == nil {
					req2.Header.Set("User-Agent", "SmartCalc/1.0")
					resp2, err2 := client.Do(req2)
					if err2 == nil {
						defer resp2.Body.Close()
						duration := time.Since(start)
						results = append(results, duration)
						scheme = "http"
						url = httpURL
						continue
					}
				}
			}
			failed++
			continue
		}
		defer resp.Body.Close()

		duration := time.Since(start)
		results = append(results, duration)

		// Small delay between pings
		if i < attempts-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if len(results) == 0 {
		return "", fmt.Errorf("all ping attempts failed")
	}

	// Calculate statistics
	var min, max, sum time.Duration
	min = results[0]
	max = results[0]

	for _, d := range results {
		sum += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	avg := sum / time.Duration(len(results))
	successRate := float64(len(results)) / float64(attempts) * 100

	// Format output
	output := fmt.Sprintf("\n> HTTP Ping to %s:%d (%s)", host, port, scheme)
	output += fmt.Sprintf("\n> Packets: Sent = %d, Received = %d, Lost = %d (%.0f%% loss)",
		attempts, len(results), failed, 100-successRate)
	output += fmt.Sprintf("\n> Round trip times:")
	output += fmt.Sprintf("\n>   Minimum = %dms, Maximum = %dms, Average = %dms",
		min.Milliseconds(), max.Milliseconds(), avg.Milliseconds())

	return output, nil
}
