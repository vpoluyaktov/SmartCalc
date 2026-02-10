package httpstatus

import (
	"strings"
	"testing"
)

func TestIsHTTPStatusExpression(t *testing.T) {
	tests := []struct {
		expr     string
		expected bool
	}{
		// Valid expressions
		{"http code 200", true},
		{"http code 404", true},
		{"http code 500", true},
		{"HTTP CODE 200", true},
		{"Http Code 301", true},
		{"http code 418", true},
		{"http status 200", true},
		{"http status 404", true},
		{"status code 200", true},
		{"status code 404", true},
		{"STATUS CODE 500", true},

		// Not HTTP status expressions
		{"200", false},
		{"404", false},
		{"http", false},
		{"http 200", false},
		{"http 404", false},
		{"hello world", false},
		{"2 + 2", false},
		{"http abc", false},
		{"http 99", false},
		{"http 1000", false},
		{"chmod 755", false},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result := IsHTTPStatusExpression(tt.expr)
			if result != tt.expected {
				t.Errorf("IsHTTPStatusExpression(%q) = %v, want %v", tt.expr, result, tt.expected)
			}
		})
	}
}

func TestEvalHTTPStatus(t *testing.T) {
	tests := []struct {
		expr     string
		contains []string
	}{
		// 1xx
		{"http code 100", []string{"100", "Continue", "Informational"}},
		{"http code 101", []string{"101", "Switching Protocols", "Informational"}},

		// 2xx
		{"http code 200", []string{"200", "OK", "Success"}},
		{"http code 201", []string{"201", "Created", "Success"}},
		{"http code 204", []string{"204", "No Content", "Success"}},

		// 3xx
		{"http code 301", []string{"301", "Moved Permanently", "Redirection"}},
		{"http code 302", []string{"302", "Found", "Redirection"}},
		{"http code 304", []string{"304", "Not Modified", "Redirection"}},
		{"http code 307", []string{"307", "Temporary Redirect", "Redirection"}},
		{"http code 308", []string{"308", "Permanent Redirect", "Redirection"}},

		// 4xx
		{"http code 400", []string{"400", "Bad Request", "Client Error"}},
		{"http code 401", []string{"401", "Unauthorized", "Client Error"}},
		{"http code 403", []string{"403", "Forbidden", "Client Error"}},
		{"http code 404", []string{"404", "Not Found", "Client Error"}},
		{"http code 405", []string{"405", "Method Not Allowed", "Client Error"}},
		{"http code 408", []string{"408", "Request Timeout", "Client Error"}},
		{"http code 409", []string{"409", "Conflict", "Client Error"}},
		{"http code 418", []string{"418", "Teapot", "Client Error"}},
		{"http code 429", []string{"429", "Too Many Requests", "Client Error"}},
		{"http code 451", []string{"451", "Unavailable For Legal Reasons", "Client Error"}},

		// 5xx
		{"http code 500", []string{"500", "Internal Server Error", "Server Error"}},
		{"http code 502", []string{"502", "Bad Gateway", "Server Error"}},
		{"http code 503", []string{"503", "Service Unavailable", "Server Error"}},
		{"http code 504", []string{"504", "Gateway Timeout", "Server Error"}},

		// Alternative prefixes
		{"HTTP CODE 200", []string{"200", "OK"}},
		{"http status 404", []string{"404", "Not Found"}},
		{"status code 404", []string{"404", "Not Found"}},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			result, err := EvalHTTPStatus(tt.expr)
			if err != nil {
				t.Errorf("EvalHTTPStatus(%q) returned error: %v", tt.expr, err)
				return
			}
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("EvalHTTPStatus(%q) = %q, want to contain %q", tt.expr, result, s)
				}
			}
		})
	}
}

func TestEvalHTTPStatusInvalid(t *testing.T) {
	tests := []string{
		"http code 999",
		"http code 600",
		"http code 099",
	}

	for _, expr := range tests {
		t.Run(expr, func(t *testing.T) {
			_, err := EvalHTTPStatus(expr)
			if err == nil {
				t.Errorf("EvalHTTPStatus(%q) should return error", expr)
			}
		})
	}
}

func TestStatusCategory(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{100, "Informational"},
		{200, "Success"},
		{301, "Redirection"},
		{404, "Client Error"},
		{500, "Server Error"},
		{600, "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := statusCategory(tt.code)
			if result != tt.expected {
				t.Errorf("statusCategory(%d) = %q, want %q", tt.code, result, tt.expected)
			}
		})
	}
}
