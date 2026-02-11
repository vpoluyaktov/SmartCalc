package nettools

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IsNetToolsExpression checks if an expression is a ping or trace command
func IsNetToolsExpression(expr string) bool {
	expr = strings.TrimSpace(expr)
	exprLower := strings.ToLower(expr)
	
	// Check for ping command
	pingPattern := regexp.MustCompile(`^(?:ping|http\s+ping)\s+\S+`)
	if pingPattern.MatchString(exprLower) {
		return true
	}
	
	// Check for trace command
	tracePattern := regexp.MustCompile(`^(?:trace|traceroute|http\s+trace)\s+\S+`)
	if tracePattern.MatchString(exprLower) {
		return true
	}
	
	return false
}

// EvalNetTools evaluates ping or trace expressions
func EvalNetTools(expr string) (string, error) {
	expr = strings.TrimSpace(expr)
	exprLower := strings.ToLower(expr)
	
	// Try ping first
	if strings.HasPrefix(exprLower, "ping ") || strings.HasPrefix(exprLower, "http ping ") {
		return evalPing(expr)
	}
	
	// Try trace
	if strings.HasPrefix(exprLower, "trace ") || 
	   strings.HasPrefix(exprLower, "traceroute ") || 
	   strings.HasPrefix(exprLower, "http trace ") {
		return evalTrace(expr)
	}
	
	return "", fmt.Errorf("not a nettools expression")
}

// parseHostPort extracts host and port from expressions like:
// "ping google.com" -> ("google.com", 443)
// "ping google.com 80" -> ("google.com", 80)
func parseHostPort(expr string) (string, int, error) {
	// Remove command prefix
	exprLower := strings.ToLower(expr)
	var rest string
	
	if strings.HasPrefix(exprLower, "http ping ") {
		rest = strings.TrimSpace(expr[10:])
	} else if strings.HasPrefix(exprLower, "http trace ") {
		rest = strings.TrimSpace(expr[11:])
	} else if strings.HasPrefix(exprLower, "traceroute ") {
		rest = strings.TrimSpace(expr[11:])
	} else if strings.HasPrefix(exprLower, "trace ") {
		rest = strings.TrimSpace(expr[6:])
	} else if strings.HasPrefix(exprLower, "ping ") {
		rest = strings.TrimSpace(expr[5:])
	} else {
		return "", 0, fmt.Errorf("invalid command")
	}
	
	// Split by space to get host and optional port
	parts := strings.Fields(rest)
	if len(parts) == 0 {
		return "", 0, fmt.Errorf("missing host")
	}
	
	host := parts[0]
	port := 443 // default port
	
	if len(parts) >= 2 {
		// Try to parse port
		p, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", 0, fmt.Errorf("invalid port: %s", parts[1])
		}
		if p < 1 || p > 65535 {
			return "", 0, fmt.Errorf("port out of range: %d", p)
		}
		port = p
	}
	
	return host, port, nil
}

func evalPing(expr string) (string, error) {
	host, port, err := parseHostPort(expr)
	if err != nil {
		return "", err
	}
	
	result, err := HTTPPing(host, port)
	if err != nil {
		return "", err
	}
	
	return result, nil
}

func evalTrace(expr string) (string, error) {
	host, port, err := parseHostPort(expr)
	if err != nil {
		return "", err
	}
	
	result, err := HTTPTrace(host, port)
	if err != nil {
		return "", err
	}
	
	return result, nil
}
