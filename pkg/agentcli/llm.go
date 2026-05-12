package agentcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LLMResponse represents the parsed LiteLLM response.
type LLMResponse struct {
	ActionType string // e.g. "hit", "north", "flee", "say"
	Args       []string
	SayLine    string // optional in-game speech
	Commentary string // optional terminal commentary (discarded by the server)
}

// CallLLM sends a prompt to the LiteLLM proxy and returns the parsed action.
func CallLLM(endpoint, apiKey, model string, messages []map[string]string, timeout time.Duration, temperature float64) (*LLMResponse, error) {
	body := map[string]any{
		"model":       model,
		"messages":    messages,
		"temperature": temperature,
	}

	raw, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", endpoint+"/v1/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("llm status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse llm response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("llm returned zero choices")
	}

	return parseLLMOutput(result.Choices[0].Message.Content)
}

// parseLLMOutput extracts action JSON and optional say/commentary from the LLM's output.
//
// Expected format:
//
//	{"command": "hit", "args": ["goblin"]}
//	say text here
//	commentary line
func parseLLMOutput(content string) (*LLMResponse, error) {
	lines := splitLines(content)
	if len(lines) == 0 {
		return nil, fmt.Errorf("empty llm output")
	}

	resp := &LLMResponse{}

	// Line 1: JSON action
	var action struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.Unmarshal([]byte(lines[0]), &action); err != nil {
		return nil, fmt.Errorf("parse action json: %w (line: %s)", err, lines[0])
	}
	resp.ActionType = action.Command
	resp.Args = action.Args

	// Line 2: optional say text
	if len(lines) > 1 && lines[1] != "" {
		resp.SayLine = lines[1]
	}

	// Lines 3+: optional terminal commentary
	if len(lines) > 2 {
		resp.Commentary = joinLines(lines[2:])
	}

	return resp, nil
}

func splitLines(s string) []string {
	var lines []string
	current := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, string(current))
			current = current[:0]
		} else {
			current = append(current, s[i])
		}
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	// Remove trailing empty lines.
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinLines(lines []string) string {
	var out string
	for i, l := range lines {
		if i > 0 {
			out += "\n"
		}
		out += l
	}
	return out
}
