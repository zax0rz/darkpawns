package oraclediff

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const enterStep = "<ENTER>"

type Scenario struct {
	Name  string
	Steps []string
}

// ParseScenario reads one command per line. Blank lines and lines beginning
// with # are comments; <ENTER> represents an intentional empty command.
func ParseScenario(name string, r io.Reader) (Scenario, error) {
	scenario := Scenario{Name: name}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if line == enterStep {
			line = ""
		}
		scenario.Steps = append(scenario.Steps, line)
	}
	if err := scanner.Err(); err != nil {
		return Scenario{}, fmt.Errorf("read scenario: %w", err)
	}
	if len(scenario.Steps) == 0 {
		return Scenario{}, fmt.Errorf("scenario %q has no steps", name)
	}
	return scenario, nil
}

// RunScenario sends one shared input stream to a server and captures everything
// the server emits between prompts. The initial greeting is captured before the
// first scripted line.
func RunScenario(conn Conn, scenario Scenario, quiescence time.Duration) (string, error) {
	var transcript strings.Builder
	initial, err := conn.ReadUntilQuiescent(quiescence)
	if err != nil {
		return "", fmt.Errorf("read greeting: %w", err)
	}
	transcript.WriteString(initial)
	for i, step := range scenario.Steps {
		if err := conn.Send(step); err != nil {
			return transcript.String(), fmt.Errorf("step %d send %q: %w\ntranscript so far:\n%s", i+1, step, err, transcript.String())
		}
		output, err := conn.ReadUntilQuiescent(quiescence)
		if err != nil {
			transcript.WriteString(output)
			// A final quit may close the Go telnet connection without emitting a
			// goodbye block. EOF at that exact boundary is a completed scenario.
			if i == len(scenario.Steps)-1 && errors.Is(err, io.EOF) {
				break
			}
			return transcript.String(), fmt.Errorf("step %d read after %q: %w\ntranscript so far:\n%s", i+1, step, err, transcript.String())
		}
		transcript.WriteString(output)
	}
	return transcript.String(), nil
}
