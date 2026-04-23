package audit

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type AuditEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	EventType   string    `json:"event_type"`
	User        string    `json:"user,omitempty"`
	IPAddress   string    `json:"ip_address,omitempty"`
	Action      string    `json:"action"`
	Details     string    `json:"details,omitempty"`
	Success     bool      `json:"success"`
}

type AuditLogger struct {
	file *os.File
}

func NewAuditLogger(filename string) (*AuditLogger, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	
	return &AuditLogger{file: file}, nil
}

func (a *AuditLogger) Log(event AuditEvent) {
	event.Timestamp = time.Now()
	
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal audit event: %v", err)
		return
	}
	
	a.file.Write(append(data, '\n'))
	
	// Also log to console for important events
	if !event.Success || event.EventType == "security" {
		log.Printf("[AUDIT] %s: %s (User: %s, IP: %s)", 
			event.EventType, event.Action, event.User, event.IPAddress)
	}
}

func (a *AuditLogger) Close() {
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