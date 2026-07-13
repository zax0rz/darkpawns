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

// Scenario is a split differential script: per-server setup (not diffed) plus
// a shared probe (diffed block-by-block).
type Scenario struct {
	Name        string
	SetupOracle []string
	SetupPort   []string
	Probe       []string
}

// ProbeBlock is one probe command and the raw output it produced.
type ProbeBlock struct {
	Command string
	Output  string
}

// ParseScenario reads a sectioned scenario file:
//
//	[setup:oracle]      # sent only to the C oracle; not diffed
//	<creation keystrokes…>
//	[setup:port]        # sent only to the Go port; not diffed
//	<creation keystrokes…>
//	[probe]             # sent to BOTH; this is the only diffed section
//	look
//	look sign
//	quit
//
// Blank lines and lines beginning with # are comments; <ENTER> represents an
// intentional empty command.
func ParseScenario(name string, r io.Reader) (Scenario, error) {
	sc := Scenario{Name: name}
	scanner := bufio.NewScanner(r)
	var section *[]string
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			switch strings.ToLower(line) {
			case "[setup:oracle]":
				section = &sc.SetupOracle
			case "[setup:port]", "[setup:go]":
				section = &sc.SetupPort
			case "[probe]":
				section = &sc.Probe
			default:
				return Scenario{}, fmt.Errorf("scenario %q line %d: unknown section %q", name, lineNo, line)
			}
			continue
		}
		if section == nil {
			return Scenario{}, fmt.Errorf("scenario %q line %d: command %q before any [section]", name, lineNo, line)
		}
		if line == enterStep {
			line = ""
		}
		*section = append(*section, line)
	}
	if err := scanner.Err(); err != nil {
		return Scenario{}, fmt.Errorf("read scenario: %w", err)
	}
	if len(sc.Probe) == 0 {
		return Scenario{}, fmt.Errorf("scenario %q has no [probe] steps", name)
	}
	return sc, nil
}

// RunSetup plays one server's setup lines and returns the captured transcript.
// It reads and discards the initial greeting before the first scripted line.
func RunSetup(conn Conn, setup []string, quiescence time.Duration) (string, error) {
	var transcript strings.Builder
	initial, err := conn.ReadUntilQuiescent(quiescence)
	if err != nil {
		return "", fmt.Errorf("read greeting: %w", err)
	}
	transcript.WriteString(initial)
	for i, step := range setup {
		if err := conn.Send(step); err != nil {
			return transcript.String(), fmt.Errorf("setup step %d send %q: %w\ntranscript so far:\n%s", i+1, step, err, transcript.String())
		}
		output, err := conn.ReadUntilQuiescent(quiescence)
		if err != nil {
			transcript.WriteString(output)
			return transcript.String(), fmt.Errorf("setup step %d read after %q: %w\ntranscript so far:\n%s", i+1, step, err, transcript.String())
		}
		transcript.WriteString(output)
	}
	return transcript.String(), nil
}

// RunProbe plays the shared probe commands and returns a block per command.
// Each block contains only the output produced by that command.
func RunProbe(conn Conn, probe []string, quiescence time.Duration) ([]ProbeBlock, error) {
	blocks := make([]ProbeBlock, 0, len(probe))
	for i, step := range probe {
		if err := conn.Send(step); err != nil {
			return blocks, fmt.Errorf("probe step %d send %q: %w", i+1, step, err)
		}
		output, err := conn.ReadUntilQuiescent(quiescence)
		if err != nil {
			// A final quit may close the connection without emitting a goodbye
			// block. EOF at that exact boundary is a completed scenario.
			if i == len(probe)-1 && errors.Is(err, io.EOF) {
				blocks = append(blocks, ProbeBlock{Command: step, Output: output})
				break
			}
			return blocks, fmt.Errorf("probe step %d read after %q: %w\noutput so far:\n%s", i+1, step, err, output)
		}
		blocks = append(blocks, ProbeBlock{Command: step, Output: output})
	}
	return blocks, nil
}
