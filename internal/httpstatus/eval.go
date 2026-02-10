package httpstatus

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// statusInfo holds the short reason phrase and a longer description
type statusInfo struct {
	Reason      string
	Description string
}

// HTTP status codes database
var statusCodes = map[int]statusInfo{
	// 1xx Informational
	100: {"Continue", "The server has received the request headers and the client should proceed to send the request body"},
	101: {"Switching Protocols", "The server is switching protocols as requested by the client"},
	102: {"Processing", "The server has received and is processing the request, but no response is available yet"},
	103: {"Early Hints", "Used to return some response headers before final HTTP message"},

	// 2xx Success
	200: {"OK", "The request has succeeded"},
	201: {"Created", "The request has been fulfilled and a new resource has been created"},
	202: {"Accepted", "The request has been accepted for processing, but the processing has not been completed"},
	203: {"Non-Authoritative Information", "The returned information is from a local or third-party copy, not the original server"},
	204: {"No Content", "The server successfully processed the request but is not returning any content"},
	205: {"Reset Content", "The server successfully processed the request and is asking the client to reset the document view"},
	206: {"Partial Content", "The server is delivering only part of the resource due to a range header sent by the client"},
	207: {"Multi-Status", "The message body contains multiple status codes for multiple independent operations (WebDAV)"},
	208: {"Already Reported", "The members of a DAV binding have already been enumerated and are not included again (WebDAV)"},
	226: {"IM Used", "The server has fulfilled a request for the resource with instance-manipulations applied"},

	// 3xx Redirection
	300: {"Multiple Choices", "The request has more than one possible response; the user or user agent should choose one"},
	301: {"Moved Permanently", "The resource has been permanently moved to a new URL"},
	302: {"Found", "The resource has been temporarily moved to a different URL"},
	303: {"See Other", "The response can be found at a different URL using a GET request"},
	304: {"Not Modified", "The resource has not been modified since the last request"},
	305: {"Use Proxy", "The requested resource must be accessed through the specified proxy (deprecated)"},
	307: {"Temporary Redirect", "The request should be repeated with the same method at a different URL"},
	308: {"Permanent Redirect", "The request and all future requests should be repeated at a different URL using the same method"},

	// 4xx Client Error
	400: {"Bad Request", "The server cannot process the request due to a client error (e.g., malformed syntax)"},
	401: {"Unauthorized", "Authentication is required and has failed or has not been provided"},
	402: {"Payment Required", "Reserved for future use; some APIs use this for rate limiting or payment walls"},
	403: {"Forbidden", "The server understood the request but refuses to authorize it"},
	404: {"Not Found", "The requested resource could not be found on the server"},
	405: {"Method Not Allowed", "The request method is not supported for the requested resource"},
	406: {"Not Acceptable", "The server cannot produce a response matching the list of acceptable values in the request"},
	407: {"Proxy Authentication Required", "The client must first authenticate itself with the proxy"},
	408: {"Request Timeout", "The server timed out waiting for the request"},
	409: {"Conflict", "The request conflicts with the current state of the server"},
	410: {"Gone", "The resource is no longer available and will not be available again"},
	411: {"Length Required", "The request did not specify the length of its content, which is required"},
	412: {"Precondition Failed", "The server does not meet one of the preconditions specified in the request"},
	413: {"Payload Too Large", "The request is larger than the server is willing or able to process"},
	414: {"URI Too Long", "The URI provided was too long for the server to process"},
	415: {"Unsupported Media Type", "The request entity has a media type which the server does not support"},
	416: {"Range Not Satisfiable", "The client has asked for a portion of the file that the server cannot supply"},
	417: {"Expectation Failed", "The server cannot meet the requirements of the Expect request-header field"},
	418: {"I'm a Teapot", "The server refuses to brew coffee because it is, permanently, a teapot (RFC 2324)"},
	421: {"Misdirected Request", "The request was directed at a server that is not able to produce a response"},
	422: {"Unprocessable Entity", "The request was well-formed but could not be followed due to semantic errors (WebDAV)"},
	423: {"Locked", "The resource that is being accessed is locked (WebDAV)"},
	424: {"Failed Dependency", "The request failed because it depended on another request that failed (WebDAV)"},
	425: {"Too Early", "The server is unwilling to risk processing a request that might be replayed"},
	426: {"Upgrade Required", "The client should switch to a different protocol"},
	428: {"Precondition Required", "The origin server requires the request to be conditional"},
	429: {"Too Many Requests", "The user has sent too many requests in a given amount of time (rate limiting)"},
	431: {"Request Header Fields Too Large", "The server is unwilling to process the request because its header fields are too large"},
	451: {"Unavailable For Legal Reasons", "The resource is unavailable due to legal demands (e.g., censorship or government-mandated block)"},

	// 5xx Server Error
	500: {"Internal Server Error", "The server encountered an unexpected condition that prevented it from fulfilling the request"},
	501: {"Not Implemented", "The server does not support the functionality required to fulfill the request"},
	502: {"Bad Gateway", "The server, while acting as a gateway or proxy, received an invalid response from the upstream server"},
	503: {"Service Unavailable", "The server is currently unable to handle the request due to overload or maintenance"},
	504: {"Gateway Timeout", "The server, while acting as a gateway or proxy, did not receive a timely response from the upstream server"},
	505: {"HTTP Version Not Supported", "The server does not support the HTTP protocol version used in the request"},
	506: {"Variant Also Negotiates", "Transparent content negotiation results in a circular reference"},
	507: {"Insufficient Storage", "The server is unable to store the representation needed to complete the request (WebDAV)"},
	508: {"Loop Detected", "The server detected an infinite loop while processing the request (WebDAV)"},
	510: {"Not Extended", "Further extensions to the request are required for the server to fulfill it"},
	511: {"Network Authentication Required", "The client needs to authenticate to gain network access"},
}

// Pattern: "http code 404", "http status 404", "status code 404"
var httpPattern = regexp.MustCompile(`(?i)^(?:http\s+(?:code|status)|status\s+code)\s+(\d{3})$`)

// IsHTTPStatusExpression checks if an expression looks like an HTTP status code query
func IsHTTPStatusExpression(expr string) bool {
	return httpPattern.MatchString(strings.TrimSpace(expr))
}

// EvalHTTPStatus evaluates an HTTP status code expression and returns a human-readable explanation
func EvalHTTPStatus(expr string) (string, error) {
	expr = strings.TrimSpace(expr)

	matches := httpPattern.FindStringSubmatch(expr)
	if matches == nil {
		return "", fmt.Errorf("not an HTTP status expression")
	}

	code, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", fmt.Errorf("invalid status code: %s", matches[1])
	}

	info, ok := statusCodes[code]
	if !ok {
		return "", fmt.Errorf("unknown HTTP status code: %d", code)
	}

	category := statusCategory(code)
	return fmt.Sprintf("%d %s (%s) — %s", code, info.Reason, category, info.Description), nil
}

// statusCategory returns the category name for a status code
func statusCategory(code int) string {
	switch {
	case code >= 100 && code < 200:
		return "Informational"
	case code >= 200 && code < 300:
		return "Success"
	case code >= 300 && code < 400:
		return "Redirection"
	case code >= 400 && code < 500:
		return "Client Error"
	case code >= 500 && code < 600:
		return "Server Error"
	default:
		return "Unknown"
	}
}
