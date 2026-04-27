package privacy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PII categories supported by OpenAI Privacy Filter
const (
	CategoryAccountNumber = "account_number"
	CategoryAddress       = "address"
	CategoryEmail         = "email"
	CategoryPerson        = "person"
	CategoryPhone         = "phone"
	CategoryURL           = "url"
	CategoryDate          = "date"
	CategorySecret        = "secret"
)

// FilterConfig defines what PII categories to filter
type FilterConfig struct {
	// Categories to filter (empty = filter all)
	Categories []string `json:"categories"`
	// Replacement string for filtered content
	Replacement string `json:"replacement"`
	// Whether to keep original length with asterisks
	KeepLength bool `json:"keep_length"`
}

// DefaultFilterConfig returns a config that filters all categories
func DefaultFilterConfig() FilterConfig {
	return FilterConfig{
		Categories: []string{
			CategoryAccountNumber,
			CategoryAddress,
			CategoryEmail,
			CategoryPerson,
			CategoryPhone,
			CategoryURL,
			CategoryDate,
			CategorySecret,
		},
		Replacement: "[REDACTED]",
		KeepLength:  false,
	}
}

// FilterRequest represents a request to the privacy filter
type FilterRequest struct {
	Text   string       `json:"text"`
	Config FilterConfig `json:"config"`
}

// FilterResponse represents a response from the privacy filter
type FilterResponse struct {
	FilteredText string   `json:"filtered_text"`
	Detected     []string `json:"detected_categories"`
	Error        string   `json:"error,omitempty"`
}

// Client for the OpenAI Privacy Filter service
type Client struct {
	baseURL    string
	httpClient *http.Client
	config     FilterConfig
}

// NewClient creates a new privacy filter client
func NewClient(baseURL string, config FilterConfig) *Client {
	if baseURL == "" {
		baseURL = "http://privacy-filter:8000"
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: config,
	}
}

// FilterText sends text to the privacy filter service
func (c *Client) FilterText(text string) (string, []string, error) {
	// If service is not available, use fallback
	if c.baseURL == "disabled" {
		return c.fallbackFilter(text), []string{"fallback"}, nil
	}

	reqBody := FilterRequest{
		Text:   text,
		Config: c.config,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.httpClient.Post(
		fmt.Sprintf("%s/filter", c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		// Fallback to simple filtering if service is unavailable
		return c.fallbackFilter(text), []string{"fallback"}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
// #nosec G104
		io.ReadAll(resp.Body) // Read and discard body
		// Return fallback without error
		return c.fallbackFilter(text), []string{"fallback"}, nil
	}

	var filterResp FilterResponse
	if err := json.NewDecoder(resp.Body).Decode(&filterResp); err != nil {
		return c.fallbackFilter(text), []string{"fallback"}, fmt.Errorf("failed to decode response: %w", err)
	}

	if filterResp.Error != "" {
		// Return fallback without error
		return c.fallbackFilter(text), []string{"fallback"}, nil
	}

	return filterResp.FilteredText, filterResp.Detected, nil
}

// fallbackFilter blocks content entirely when privacy service is unavailable
func (c *Client) fallbackFilter(text string) string {
	return "[FILTERED]"
}

// BatchFilter filters multiple texts at once
func (c *Client) BatchFilter(texts []string) ([]string, [][]string, error) {
	// For simplicity, process sequentially
	// In production, you might want to implement batch API if supported
	var filteredTexts []string
	var allDetected [][]string

	for _, text := range texts {
		filtered, detected, err := c.FilterText(text)
		if err != nil {
			return nil, nil, err
		}
		filteredTexts = append(filteredTexts, filtered)
		allDetected = append(allDetected, detected)
	}

	return filteredTexts, allDetected, nil
}
