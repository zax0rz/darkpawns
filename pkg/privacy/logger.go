package privacy

import (
	"fmt"
	"log"
	"os"
	"sync"
)

// PrivacyLogger wraps the standard logger with PII filtering
type PrivacyLogger struct {
	client  *Client
	stdLog  *log.Logger
	mu      sync.Mutex
	enabled bool
}

// NewPrivacyLogger creates a new privacy-aware logger
func NewPrivacyLogger(client *Client, prefix string, flag int) *PrivacyLogger {
	if client == nil {
		// Create a disabled client for fallback
		client = NewClient("disabled", DefaultFilterConfig())
	}

	return &PrivacyLogger{
		client:  client,
		stdLog:  log.New(os.Stdout, prefix, flag),
		enabled: true,
	}
}

// Print logs a message with PII filtering
func (pl *PrivacyLogger) Print(v ...interface{}) {
	pl.log(fmt.Sprint(v...))
}

// Printf logs a formatted message with PII filtering
func (pl *PrivacyLogger) Printf(format string, v ...interface{}) {
	pl.log(fmt.Sprintf(format, v...))
}

// Println logs a message with PII filtering and newline
func (pl *PrivacyLogger) Println(v ...interface{}) {
	pl.log(fmt.Sprintln(v...))
}

// log processes and logs a message with PII filtering
func (pl *PrivacyLogger) log(msg string) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if !pl.enabled {
		pl.stdLog.Print(msg)
		return
	}

	filtered, detected, err := pl.client.FilterText(msg)
	if err != nil {
		// Log the error but still output the original message
		pl.stdLog.Printf("PII filter error: %v", err)
		pl.stdLog.Print(msg)
		return
	}

	if len(detected) > 0 && detected[0] != "fallback" {
		// Add metadata about what was filtered
		filtered = fmt.Sprintf("[PII filtered: %v] %s", detected, filtered)
	}

	pl.stdLog.Print(filtered)
}

// Disable turns off PII filtering
func (pl *PrivacyLogger) Disable() {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.enabled = false
}

// Enable turns on PII filtering
func (pl *PrivacyLogger) Enable() {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.enabled = true
}

// SetClient updates the privacy filter client
func (pl *PrivacyLogger) SetClient(client *Client) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.client = client
}

// Global privacy logger instance
var (
	globalLogger *PrivacyLogger
	once         sync.Once
)

// GetGlobalLogger returns the global privacy logger instance
func GetGlobalLogger() *PrivacyLogger {
	once.Do(func() {
		// Initialize with default config
		config := DefaultFilterConfig()
		// Don't filter dates by default (often needed for game logs)
		config.Categories = []string{
			CategoryAccountNumber,
			CategoryAddress,
			CategoryEmail,
			CategoryPerson,
			CategoryPhone,
			CategoryURL,
			CategorySecret,
		}
		client := NewClient("", config)
		globalLogger = NewPrivacyLogger(client, "", log.LstdFlags)
	})
	return globalLogger
}

// Convenience functions using global logger
func Print(v ...interface{}) {
	GetGlobalLogger().Print(v...)
}

func Printf(format string, v ...interface{}) {
	GetGlobalLogger().Printf(format, v...)
}

func Println(v ...interface{}) {
	GetGlobalLogger().Println(v...)
}
