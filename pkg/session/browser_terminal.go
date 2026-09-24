package session

import (
	"encoding/json"
	"log/slog"
)

// The browser client drives its WebSocket session as a terminal: it sends raw
// lines and receives the bytes telnet would write, so /play shows what a
// telnet player sees (DP-1320). Structured vars and state still reach it
// alongside, for the sidebar, the way GMCP rides beside telnet text.

// startBrowserTerminal switches the session into terminal mode and queues the
// greeting and name prompt.
func (s *Session) startBrowserTerminal() {
	if s.browserTerminal.Swap(true) {
		return
	}
	s.sendRawEvent(TerminalGreeting())
}

// handleTerminalLine routes one line from the browser through the shared
// terminal, closing the connection when the terminal ends it.
func (s *Session) handleTerminalLine(data json.RawMessage) error {
	if !s.browserTerminal.Load() {
		return ErrUnknownMessageType
	}
	var in struct {
		Line string `json:"line"`
	}
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	if !s.TerminalLine(in.Line) && !s.SendClosed() {
		s.CloseSend()
	}
	return nil
}

// terminalOut is the browser's frame for rendered terminal bytes.
type terminalOut struct {
	Text string `json:"text"`
	// Prompt marks the command prompt; Entry marks a login or creation
	// prompt, and Secret asks the client not to echo the answer.
	Prompt bool `json:"prompt,omitempty"`
	Entry  bool `json:"entry,omitempty"`
	Secret bool `json:"secret,omitempty"`
}

// renderForBrowserTerminal turns one queued message into what a terminal-mode
// browser receives: an "out" frame with telnet's bytes, the message itself if
// it is sidebar data, or nothing.
func renderForBrowserTerminal(msg []byte) ([]byte, bool) {
	f, ok := RenderTerminalFrame(msg)
	if !ok {
		var sm struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(msg, &sm); err == nil && (sm.Type == MsgVars || sm.Type == MsgState) {
			return msg, true
		}
		return nil, false
	}
	var out terminalOut
	switch f.Kind {
	case FrameText:
		out = terminalOut{Text: f.Text}
	case FramePrompt:
		out = terminalOut{Text: f.Text, Prompt: true}
	case FrameEntryPrompt:
		out = terminalOut{Text: f.Text, Entry: true, Secret: f.Secret}
	default:
		// GMCP is negotiated by telnet clients only.
		return nil, false
	}
	b, err := json.Marshal(ServerMessage{Type: MsgOut, Data: out})
	if err != nil {
		slog.Error("render browser terminal frame", "error", err)
		return nil, false
	}
	return b, true
}
