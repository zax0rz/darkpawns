package session

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/internal/dpclock"
)

// maxControlPumpPulses bounds one ~dpclock control line.
const maxControlPumpPulses = 100_000

// HandleClockControl consumes the DP_CLOCK harness control line
// "~dpclock pulse N" and advances the frozen heartbeat N pulses. It reports
// whether the line was the control (and so must not reach the command
// interpreter). Outside DP_CLOCK it never matches. Telnet and WebSocket
// sessions share it, so a scenario can drive the clock over either transport.
func (s *Session) HandleClockControl(line string) bool {
	if !dpclock.Frozen() {
		return false
	}
	fields := strings.Fields(line)
	if len(fields) != 3 || fields[0] != "~dpclock" || fields[1] != "pulse" {
		return false
	}
	n, err := strconv.Atoi(fields[2])
	if err != nil || n <= 0 || n > maxControlPumpPulses {
		return false
	}
	// The control line is input on this session's descriptor: C's input
	// processing clears has_prompt, so this session's next output flush
	// carries process_output's interruption CRLF (comm.c:607, 1620-1643).
	if err := s.manager.PumpPulsesFrom(s, n); err != nil {
		slog.Error("DP_CLOCK pulse pump failed", "pulses", n, "error", err)
	}
	return true
}

// isClockControlMessage handles a command message whose line is the clock
// control, whatever state the session is in.
func (s *Session) isClockControlMessage(data json.RawMessage) bool {
	if !dpclock.Frozen() {
		return false
	}
	var cmd CommandData
	if err := json.Unmarshal(data, &cmd); err != nil {
		return false
	}
	line := cmd.RawLine
	if line == "" {
		line = cmd.Command
		if len(cmd.Args) > 0 {
			line += " " + strings.Join(cmd.Args, " ")
		}
	}
	return s.HandleClockControl(line)
}
