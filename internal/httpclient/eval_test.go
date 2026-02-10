package httpclient

import (
	"testing"
)

func TestIsHTTPClientExpression(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		// Valid expressions
		{"http google.com", true},
		{"http https://google.com", true},
		{"http http://example.com", true},
		{"HTTP google.com", true},
		{"Http example.org", true},
		{"curl google.com", true},
		{"curl https://example.com", true},
		{"curl http://example.com", true},
		{"CURL google.com", true},
		{"Curl example.org", true},
		{"http www.google.com", true},
		{"curl api.github.com", true},
		{"http localhost", true},
		{"http localhost:8080", true},

		// Not HTTP client expressions
		{"http code 200", false},
		{"http code 404", false},
		{"http status 500", false},
		{"http", false},
		{"curl", false},
		{"200", false},
		{"hello world", false},
		{"2 + 2", false},
		{"chmod 755", false},
		{"http abc", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result := IsHTTPClientExpression(tt.expr)
			if result != tt.expected {
				t.Errorf("IsHTTPClientExpression(%q) = %v, want %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestIsValidTarget(t *testing.T) {
	tests := []struct {
		target   string
		expected bool
	}{
		{"google.com", true},
		{"https://google.com", true},
		{"http://example.com", true},
		{"www.google.com", true},
		{"api.github.com", true},
		{"localhost", true},
		{"localhost:8080", true},
		{"example.com:443", true},

		// Invalid
		{"", false},
		{"abc", false},
		{"hello world", false},
		{"code 404", false},
		{"status 500", false},
	}

	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			result := isValidTarget(tt.target)
			if result != tt.expected {
				t.Errorf("isValidTarget(%q) = %v, want %v", tt.target, result, tt.expected)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"google.com", "https://google.com"},
		{"https://google.com", "https://google.com"},
		{"http://example.com", "http://example.com"},
		{"HTTP://EXAMPLE.COM", "HTTP://EXAMPLE.COM"},
		{"www.google.com", "https://www.google.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatResponse(t *testing.T) {
	// We can't easily test the full HTTP request without a server,
	// but we can test the format function with a mock response
	// This is tested indirectly through integration tests
}

// Integration test - requires network access
func TestEvalHTTPClientIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tests := []struct {
		expr        string
		shouldWork  bool
		containsAny []string
	}{
		{"http google.com", true, []string{"HTTP/"}},
		{"curl google.com", true, []string{"HTTP/"}},
		{"http https://google.com", true, []string{"HTTP/"}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalHTTPClient(tt.expr)
			if tt.shouldWork {
				if err != nil {
					t.Errorf("EvalHTTPClient(%q) returned error: %v", tt.expr, err)
					return
				}
				for _, s := range tt.containsAny {
					if !contains(result, s) {
						t.Errorf("EvalHTTPClient(%q) = %q, want to contain %q", tt.expr, result, s)
					}
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
