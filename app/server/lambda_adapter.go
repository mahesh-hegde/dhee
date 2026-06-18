package server

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// LambdaEventToHTTPRequest converts a Lambda Function URL event to an http.Request.
// The request is constructed to be compatible with echo routing and middleware.
func LambdaEventToHTTPRequest(ctx context.Context, event events.LambdaFunctionURLRequest) (*http.Request, error) {
	// Determine if body is base64 encoded
	bodyReader := io.Reader(strings.NewReader(event.Body))
	if event.IsBase64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(event.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(decoded)
	}

	// Determine scheme from headers or default to https
	scheme := "https"
	if event.Headers != nil {
		if proto := event.Headers["x-forwarded-proto"]; proto != "" {
			scheme = proto
		}
	}

	// Get host from headers or request context
	host := event.Headers["host"]
	if host == "" && event.RequestContext.DomainName != "" {
		host = event.RequestContext.DomainName
	}
	if host == "" {
		host = "localhost"
	}

	// Build path with query string
	requestPath := event.RawPath
	if event.RawQueryString != "" {
		requestPath = requestPath + "?" + event.RawQueryString
	}

	urlStr := scheme + "://" + host + requestPath
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	// Create request with the HTTP method from request context
	method := event.RequestContext.HTTP.Method
	if method == "" {
		method = "GET"
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
	if err != nil {
		return nil, err
	}

	// Copy headers from the event
	if event.Headers != nil {
		for k, v := range event.Headers {
			// Skip pseudo-headers
			if k == "host" {
				continue
			}
			req.Header.Add(k, v)
		}
	}

	// Set RemoteAddr for logging and rate limiting
	// Extract source IP from headers first, then fall back to request context
	if xff := event.Headers["x-forwarded-for"]; xff != "" {
		// Take the first IP if multiple are present
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			req.RemoteAddr = strings.TrimSpace(ips[0]) + ":0"
		}
	} else if sourceIP := event.RequestContext.HTTP.SourceIP; sourceIP != "" {
		req.RemoteAddr = sourceIP + ":0"
	} else {
		req.RemoteAddr = "127.0.0.1:0"
	}

	// Ensure URL and Host are correctly set
	req.URL = u
	req.RequestURI = ""
	req.Host = host

	return req, nil
}

// HTTPResponseToLambdaResponse converts a buffered HTTP response to a Lambda Function URL response.
func HTTPResponseToLambdaResponse(rw *BufferedResponseWriter) events.LambdaFunctionURLResponse {
	statusCode := rw.StatusCode
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	body := rw.Body.String()
	isBase64 := false

	// Check if response should be base64 encoded (for binary content)
	contentType := rw.Headers.Get("content-type")
	if contentType == "" {
		// Default to text/html for Lambda functions if not set
		rw.Headers.Set("content-type", "text/html; charset=utf-8")
		contentType = "text/html; charset=utf-8"
	}

	// Check if we should base64 encode the response
	// Gzipped responses appear as binary
	if strings.Contains(contentType, "application/") || rw.Headers.Get("content-encoding") == "gzip" {
		body = base64.StdEncoding.EncodeToString([]byte(body))
		isBase64 = true
	}

	// Convert http.Header to map[string]string (Lambda Function URL expects single values)
	headers := make(map[string]string)
	for k, vals := range rw.Headers {
		if len(vals) > 0 {
			// For set-cookie and other multi-value headers, join with comma (standard HTTP behavior)
			headers[k] = strings.Join(vals, ",")
		}
	}

	return events.LambdaFunctionURLResponse{
		StatusCode:      statusCode,
		Headers:         headers,
		Body:            body,
		Cookies:         []string{}, // Lambda will extract cookies from Set-Cookie header if present
		IsBase64Encoded: isBase64,
	}
}
