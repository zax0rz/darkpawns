package privacy

import (
	"os"
	"strconv"
	"strings"
)

// Config holds privacy filter configuration
type Config struct {
	// Service URL
	URL string

	// Whether filtering is enabled
	Enabled bool

	// Categories to filter
	Categories []string

	// Replacement text
	Replacement string

	// Keep original length
	KeepLength bool

	// Batch size
	BatchSize int

	// Timeout in seconds
	Timeout int

	// Fallback behavior
	Fallback string

	// Log level
	LogLevel string

	// Game-specific settings
	FilterPlayerNames   bool
	FilterLocationNames bool
	FilterCommands      bool
	FilterCombatDetails bool
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		URL:                 "http://privacy-filter:8000",
		Enabled:             true,
		Categories:          []string{"account_number", "address", "email", "person", "phone", "url", "secret"},
		Replacement:         "[REDACTED]",
		KeepLength:          false,
		BatchSize:           10,
		Timeout:             10,
		Fallback:            "mask",
		LogLevel:            "info",
		FilterPlayerNames:   true,
		FilterLocationNames: false,
		FilterCommands:      false,
		FilterCombatDetails: false,
	}
}

// LoadConfig loads configuration from environment variables
func LoadConfig() Config {
	config := DefaultConfig()

	if url := os.Getenv("PRIVACY_FILTER_URL"); url != "" {
		config.URL = url
	}

	if enabled := os.Getenv("PRIVACY_FILTER_ENABLED"); enabled != "" {
		config.Enabled = strings.ToLower(enabled) == "true"
	}

	if categories := os.Getenv("PRIVACY_FILTER_CATEGORIES"); categories != "" {
		config.Categories = strings.Split(categories, ",")
	}

	if replacement := os.Getenv("PRIVACY_FILTER_REPLACEMENT"); replacement != "" {
		config.Replacement = replacement
	}

	if keepLength := os.Getenv("PRIVACY_FILTER_KEEP_LENGTH"); keepLength != "" {
		config.KeepLength = strings.ToLower(keepLength) == "true"
	}

	if batchSize := os.Getenv("PRIVACY_FILTER_BATCH_SIZE"); batchSize != "" {
		if val, err := strconv.Atoi(batchSize); err == nil && val > 0 {
			config.BatchSize = val
		}
	}

	if timeout := os.Getenv("PRIVACY_FILTER_TIMEOUT"); timeout != "" {
		if val, err := strconv.Atoi(timeout); err == nil && val > 0 {
			config.Timeout = val
		}
	}

	if fallback := os.Getenv("PRIVACY_FILTER_FALLBACK"); fallback != "" {
		config.Fallback = fallback
	}

	if logLevel := os.Getenv("PRIVACY_FILTER_LOG_LEVEL"); logLevel != "" {
		config.LogLevel = logLevel
	}

	if filterPlayerNames := os.Getenv("FILTER_PLAYER_NAMES"); filterPlayerNames != "" {
		config.FilterPlayerNames = strings.ToLower(filterPlayerNames) == "true"
	}

	if filterLocationNames := os.Getenv("FILTER_LOCATION_NAMES"); filterLocationNames != "" {
		config.FilterLocationNames = strings.ToLower(filterLocationNames) == "true"
	}

	if filterCommands := os.Getenv("FILTER_COMMANDS"); filterCommands != "" {
		config.FilterCommands = strings.ToLower(filterCommands) == "true"
	}

	if filterCombatDetails := os.Getenv("FILTER_COMBAT_DETAILS"); filterCombatDetails != "" {
		config.FilterCombatDetails = strings.ToLower(filterCombatDetails) == "true"
	}

	return config
}

// ToFilterConfig converts Config to FilterConfig
func (c Config) ToFilterConfig() FilterConfig {
	return FilterConfig{
		Categories:  c.Categories,
		Replacement: c.Replacement,
		KeepLength:  c.KeepLength,
	}
}
