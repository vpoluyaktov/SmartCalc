package nettools

import (
	"strings"
	"testing"
)

func TestIsNetToolsExpression(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		{"ping google.com", true},
		{"ping google.com 80", true},
		{"ping google.com 443", true},
		{"trace google.com", true},
		{"trace google.com 80", true},
		{"traceroute google.com", true},
		{"http ping google.com", true},
		{"http trace google.com", true},
		{"PING google.com", true},
		{"TRACE google.com", true},
		{"not a ping command", false},
		{"", false},
		{"http google.com", false},
		{"curl google.com", false},
	}

	for _, tt := range tests {
		result := IsNetToolsExpression(tt.expr)
		if result != tt.expected {
			t.Errorf("IsNetToolsExpression(%q) = %v, want %v", tt.expr, result, tt.expected)
		}
	}
}

func TestParseHostPort(t *testing.T) {
	tests := []struct {
		expr         string
		expectedHost string
		expectedPort int
		expectError  bool
	}{
		{"ping google.com", "google.com", 443, false},
		{"ping google.com 80", "google.com", 80, false},
		{"ping google.com 8080", "google.com", 8080, false},
		{"trace example.com", "example.com", 443, false},
		{"trace example.com 443", "example.com", 443, false},
		{"traceroute example.com 22", "example.com", 22, false},
		{"http ping localhost", "localhost", 443, false},
		{"http trace localhost 3000", "localhost", 3000, false},
		{"ping", "", 0, true},
		{"ping google.com invalid", "google.com", 0, true},
		{"ping google.com 99999", "google.com", 0, true},
		{"ping google.com 0", "google.com", 0, true},
	}

	for _, tt := range tests {
		host, port, err := parseHostPort(tt.expr)
		if tt.expectError {
			if err == nil {
				t.Errorf("parseHostPort(%q) expected error, got none", tt.expr)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseHostPort(%q) unexpected error: %v", tt.expr, err)
			continue
		}
		if host != tt.expectedHost {
			t.Errorf("parseHostPort(%q) host = %q, want %q", tt.expr, host, tt.expectedHost)
		}
		if port != tt.expectedPort {
			t.Errorf("parseHostPort(%q) port = %d, want %d", tt.expr, port, tt.expectedPort)
		}
	}
}

func TestEvalNetToolsPing(t *testing.T) {
	// Test with a reliable host
	result, err := EvalNetTools("ping google.com 80")
	if err != nil {
		t.Skipf("Skipping ping test (network required): %v", err)
	}

	if !strings.Contains(result, "HTTP Ping to google.com:80") {
		t.Errorf("Expected ping result to contain 'HTTP Ping to google.com:80', got: %s", result)
	}

	if !strings.Contains(result, "Round trip times") {
		t.Errorf("Expected ping result to contain timing info, got: %s", result)
	}
}

func TestEvalNetToolsTrace(t *testing.T) {
	// Test with a reliable host
	result, err := EvalNetTools("trace google.com 80")
	if err != nil {
		t.Skipf("Skipping trace test (network required): %v", err)
	}

	if !strings.Contains(result, "Traceroute to google.com:80") {
		t.Errorf("Expected trace result to contain 'Traceroute to google.com:80', got: %s", result)
	}

	if !strings.Contains(result, "DNS Resolution") {
		t.Errorf("Expected trace result to contain DNS Resolution, got: %s", result)
	}
}

func TestEvalNetToolsNetstat(t *testing.T) {
	// Test with a reliable host
	result, err := EvalNetTools("netstat google.com 80")
	if err != nil {
		t.Skipf("Skipping netstat test (network required): %v", err)
	}

	if !strings.Contains(result, "HTTP Netstat to google.com:80") {
		t.Errorf("Expected netstat result to contain 'HTTP Netstat to google.com:80', got: %s", result)
	}

	if !strings.Contains(result, "Timing breakdown") {
		t.Errorf("Expected netstat result to contain timing breakdown, got: %s", result)
	}
}

func TestEvalNetToolsInvalidCommand(t *testing.T) {
	_, err := EvalNetTools("invalid command")
	if err == nil {
		t.Error("Expected error for invalid command, got nil")
	}
}
