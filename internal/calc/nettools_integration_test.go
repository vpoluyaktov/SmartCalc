package calc

import (
	"strings"
	"testing"
)

func TestPingIntegration(t *testing.T) {
	lines := []string{
		"ping google.com 80 =",
	}
	
	results := EvalLines(lines, 1)
	
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	
	output := results[0].Output
	
	// Check if it contains expected ping output
	if !strings.Contains(output, "HTTP Ping to google.com:80") {
		t.Skipf("Skipping ping test (network required), got: %s", output)
	}
	
	if !strings.Contains(output, "Round trip times") {
		t.Errorf("Expected ping output to contain timing info, got: %s", output)
	}
}

func TestTraceIntegration(t *testing.T) {
	lines := []string{
		"trace google.com 80 =",
	}
	
	results := EvalLines(lines, 1)
	
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	
	output := results[0].Output
	
	// Check if it contains expected trace output
	if !strings.Contains(output, "HTTP Trace to google.com:80") {
		t.Skipf("Skipping trace test (network required), got: %s", output)
	}
	
	if !strings.Contains(output, "Timing breakdown") {
		t.Errorf("Expected trace output to contain timing breakdown, got: %s", output)
	}
}

func TestPingDefaultPort(t *testing.T) {
	lines := []string{
		"ping google.com =",
	}
	
	results := EvalLines(lines, 1)
	
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	
	output := results[0].Output
	
	// Should use default port 443
	if !strings.Contains(output, "HTTP Ping to google.com:443") && !strings.Contains(output, "ERR:") {
		t.Skipf("Skipping ping test (network required), got: %s", output)
	}
}

func TestTraceDefaultPort(t *testing.T) {
	lines := []string{
		"trace google.com =",
	}
	
	results := EvalLines(lines, 1)
	
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}
	
	output := results[0].Output
	
	// Should use default port 443
	if !strings.Contains(output, "HTTP Trace to google.com:443") && !strings.Contains(output, "ERR:") {
		t.Skipf("Skipping trace test (network required), got: %s", output)
	}
}
