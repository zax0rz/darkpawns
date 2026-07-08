package privacy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// maxLoggedBodyBytes caps how many bytes of a request/response body the
// logging middleware buffers for its truncated log output. Full bodies are
// always passed through untouched to downstream handlers and to the client
// respectively; only the copy captured for logging is bounded, so
// multi-megabyte uploads/responses don't cause unbounded memory growth.
const maxLoggedBodyBytes = 1000

// LoggingResponseWriter wraps http.ResponseWriter to capture status code
type LoggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
	truncated  bool
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

// Write captures up to maxLoggedBodyBytes of the response body for logging,
// then passes the full write through to the underlying ResponseWriter
// unmodified so the client always receives the complete response.
func (lrw *LoggingResponseWriter) Write(b []byte) (int, error) {
	if remaining := maxLoggedBodyBytes - lrw.body.Len(); remaining > 0 {
		if len(b) <= remaining {
			lrw.body.Write(b)
		} else {
			lrw.body.Write(b[:remaining])
			lrw.truncated = true
		}
	} else if len(b) > 0 {
		lrw.truncated = true
	}
	return lrw.ResponseWriter.Write(b)
}

// Hijack implements http.Hijacker for WebSocket connections
func (lrw *LoggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := lrw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("response writer does not implement http.Hijacker")
}

// bodyReadCloser stitches a small pre-read buffer back onto the still-open
// underlying body reader, so downstream handlers see the same byte stream
// the original body would have produced, while Close still closes the
// original reader (e.g. releasing the connection).
type bodyReadCloser struct {
	io.Reader
	closer io.Closer
}

func (b bodyReadCloser) Close() error {
	return b.closer.Close()
}

// captureRequestBody reads at most maxLoggedBodyBytes+1 bytes from r.Body
// for logging purposes, then reconstructs r.Body so downstream handlers
// still receive the complete, unaltered body. It returns the bytes to log
// (capped at maxLoggedBodyBytes) and whether the body was longer than that.
func captureRequestBody(r *http.Request) (captured []byte, truncated bool) {
	if r.Body == nil {
		return nil, false
	}

	peeked, err := io.ReadAll(io.LimitReader(r.Body, maxLoggedBodyBytes+1))
	if err != nil {
		slog.Warn(
			"failed to read request body in privacy middleware",
			"error", err,
			"path", r.URL.Path,
			"method", r.Method,
		)
		r.Body = io.NopCloser(bytes.NewReader(nil))
		return nil, false
	}

	r.Body = bodyReadCloser{
		Reader: io.MultiReader(bytes.NewReader(peeked), r.Body),
		closer: r.Body,
	}

	if len(peeked) > maxLoggedBodyBytes {
		return peeked[:maxLoggedBodyBytes], true
	}
	return peeked, false
}

// HTTPMiddleware provides PII-filtered HTTP request/response logging
func HTTPMiddleware(next http.Handler, client *Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create logging response writer
		lrw := NewLoggingResponseWriter(w)

		// Read request body, capturing only enough for logging while
		// leaving the full body available to downstream handlers.
		var requestBody bytes.Buffer
		capturedReq, reqTruncated := captureRequestBody(r)
		requestBody.Write(capturedReq)

		// Process request
		next.ServeHTTP(lrw, r)

		// Calculate duration
		duration := time.Since(start)

		// Filter sensitive data from logs. Use a request-local logger tied to
		// the supplied client so concurrent handlers cannot influence each
		// other's privacy filter configuration.
		var logger *PrivacyLogger
		if client != nil {
			logger = NewPrivacyLogger(client, "", log.LstdFlags)
		} else {
			logger = GetGlobalLogger()
		}

		// Log request (filtered)
		remoteAddr := r.RemoteAddr
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			remoteAddr = forwarded
		}

		// Filter request body if it contains sensitive data
		reqBodyStr := requestBody.String()
		if reqTruncated {
			reqBodyStr += "... [truncated]"
		}

		// Filter response body if it contains sensitive data
		respBodyStr := lrw.body.String()
		if lrw.truncated {
			respBodyStr += "... [truncated]"
		}

		logger.Printf(
			"HTTP %s %s %s %d %v\nRequest: %s\nResponse: %s",
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
	logger *PrivacyLogger
}

// NewWebSocketLogger creates a new WebSocket logger
func NewWebSocketLogger(client *Client, prefix string) *WebSocketLogger {
	var logger *PrivacyLogger
	if client != nil {
		logger = NewPrivacyLogger(client, prefix, log.LstdFlags)
	} else {
		logger = GetGlobalLogger()
	}
	return &WebSocketLogger{
		client: client,
		prefix: prefix,
		logger: logger,
	}
}

// LogIncoming logs an incoming WebSocket message with PII filtering
func (wsl *WebSocketLogger) LogIncoming(sessionID, message string) {
	wsl.logger.Printf("%s [WS IN] [%s] %s", wsl.prefix, sessionID, message)
}

// LogOutgoing logs an outgoing WebSocket message with PII filtering
func (wsl *WebSocketLogger) LogOutgoing(sessionID, message string) {
	wsl.logger.Printf("%s [WS OUT] [%s] %s", wsl.prefix, sessionID, message)
}

// LogEvent logs a WebSocket event with PII filtering
func (wsl *WebSocketLogger) LogEvent(sessionID, eventType, details string) {
	fullDetails := fmt.Sprintf("Event: %s, Details: %s", eventType, details)
	wsl.logger.Printf("%s [WS EVENT] [%s] %s", wsl.prefix, sessionID, fullDetails)
}
