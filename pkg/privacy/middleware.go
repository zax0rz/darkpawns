package privacy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// LoggingResponseWriter wraps http.ResponseWriter to capture status code
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// NewLoggingResponseWriter creates a new logging response writer
func NewLoggingResponseWriter(w http.ResponseWriter) *LoggingResponseWriter {
	return &LoggingResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           bytes.NewBuffer(nil),
	}
}

// WriteHeader captures the status code
func (lrw *LoggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write captures the response body
func (lrw *LoggingResponseWriter) Write(b []byte) (int, error) {
	lrw.body.Write(b)
	return lrw.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker for WebSocket connections
func (lrw *LoggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
}

// HTTPMiddleware provides PII-filtered HTTP request/response logging
func HTTPMiddleware(next http.Handler, client *Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create logging response writer
		lrw := NewLoggingResponseWriter(w)

		// Read request body
		var requestBody bytes.Buffer
		if r.Body != nil {
			bodyBytes, _ := io.ReadAll(r.Body)
			requestBody.Write(bodyBytes)
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Process request
		next.ServeHTTP(lrw, r)

		// Calculate duration
		duration := time.Since(start)

		// Filter sensitive data from logs
		logger := GetGlobalLogger()
		if client != nil {
			logger.SetClient(client)
		}

		// Log request (filtered)
		remoteAddr := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			remoteAddr = forwarded
		}

		// Filter request body if it contains sensitive data
		reqBodyStr := requestBody.String()
		if len(reqBodyStr) > 1000 {
			reqBodyStr = reqBodyStr[:1000] + "... [truncated]"
		}

		// Filter response body if it contains sensitive data
		respBodyStr := lrw.body.String()
		if len(respBodyStr) > 1000 {
			respBodyStr = respBodyStr[:1000] + "... [truncated]"
		}

		logger.Printf("HTTP %s %s %s %d %v\nRequest: %s\nResponse: %s",
			r.Method,
			r.URL.Path,
			remoteAddr,
			lrw.statusCode,
			duration,
			reqBodyStr,
			respBodyStr,
		)
	})
}

// WebSocketLogger provides PII-filtered WebSocket message logging
type WebSocketLogger struct {
	client *Client
	prefix string
}

// NewWebSocketLogger creates a new WebSocket logger
func NewWebSocketLogger(client *Client, prefix string) *WebSocketLogger {
	return &WebSocketLogger{
		client: client,
		prefix: prefix,
	}
}

// LogIncoming logs an incoming WebSocket message with PII filtering
func (wsl *WebSocketLogger) LogIncoming(sessionID, message string) {
	logger := GetGlobalLogger()
	if wsl.client != nil {
		logger.SetClient(wsl.client)
	}

	logger.Printf("%s [WS IN] [%s] %s", wsl.prefix, sessionID, message)
}

// LogOutgoing logs an outgoing WebSocket message with PII filtering
func (wsl *WebSocketLogger) LogOutgoing(sessionID, message string) {
	logger := GetGlobalLogger()
	if wsl.client != nil {
		logger.SetClient(wsl.client)
	}

	logger.Printf("%s [WS OUT] [%s] %s", wsl.prefix, sessionID, message)
}

// LogEvent logs a WebSocket event with PII filtering
func (wsl *WebSocketLogger) LogEvent(sessionID, eventType, details string) {
	logger := GetGlobalLogger()
	if wsl.client != nil {
		logger.SetClient(wsl.client)
	}

	fullDetails := fmt.Sprintf("Event: %s, Details: %s", eventType, details)
	logger.Printf("%s [WS EVENT] [%s] %s", wsl.prefix, sessionID, fullDetails)
}
