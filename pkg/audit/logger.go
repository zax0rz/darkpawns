package audit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// AuditEvent represents a single auditable action such as a login, security incident, or admin operation.
type AuditEvent struct {
	Timestamp time.Time `json:"timestamp"`
	EventType string    `json:"event_type"`
	User      string    `json:"user,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	Action    string    `json:"action"`
	Details   string    `json:"details,omitempty"`
	Success   bool      `json:"success"`
}

// AuditLogger writes structured audit events to an append-only file.
type AuditLogger struct {
	file *os.File
}

// NewAuditLogger opens (or creates) an audit log file with restrictive permissions.
func NewAuditLogger(filename string) (*AuditLogger, error) {
// #nosec G304
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}

	return &AuditLogger{file: file}, nil
}

// Log writes an AuditEvent to the log file, hashing the IP address for privacy.
func (a *AuditLogger) Log(event AuditEvent) {
	event.Timestamp = time.Now()

	// Hash IP address for privacy
	if event.IPAddress != "" {
		h := sha256.Sum256([]byte(event.IPAddress))
		event.IPAddress = fmt.Sprintf("%x", h[:8]) // first 8 bytes (16 hex chars)
	}

	data, err := json.Marshal(event)
	if err != nil {
		slog.Error("Failed to marshal audit event", "error", err)
		return
	}

// #nosec G104
	a.file.Write(append(data, '\n'))

	// Also log to console for important events
	if !event.Success || event.EventType == "security" {
		slog.Warn("audit event",
			"event_type", event.EventType,
			"action", event.Action,
			"user", event.User,
			"ip_address", event.IPAddress,
		)
	}
}

// Close flushes and closes the underlying audit log file.
func (a *AuditLogger) Close() {
// #nosec G104
	a.file.Close()
}

// Global audit logger instance
var globalLogger *AuditLogger

// Initialize the global audit logger
func Init(filename string) error {
	logger, err := NewAuditLogger(filename)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// LogEvent logs an event using the global logger
func LogEvent(event AuditEvent) {
	if globalLogger != nil {
		globalLogger.Log(event)
	}
}

// Convenience functions
// LogLoginAttempt records a login success or failure for auditing.
func LogLoginAttempt(user, ip string, success bool) {
	event := AuditEvent{
		EventType: "authentication",
		User:      user,
		IPAddress: ip,
		Action:    "login_attempt",
		Success:   success,
	}

	if !success {
		event.Details = "Failed login attempt"
	}

	LogEvent(event)
}

// LogSecurityEvent records a security-relevant event (e.g. rate limit exceeded, invalid token).
func LogSecurityEvent(action, details, user, ip string) {
	event := AuditEvent{
		EventType: "security",
		User:      user,
		IPAddress: ip,
		Action:    action,
		Details:   details,
		Success:   false, // Security events are typically about issues
	}

	LogEvent(event)
}

// LogAdminAction records an administrative action performed by a staff member.
func LogAdminAction(user, action, details string) {
	event := AuditEvent{
		EventType: "administration",
		User:      user,
		Action:    action,
		Details:   details,
		Success:   true,
	}

	LogEvent(event)
}
