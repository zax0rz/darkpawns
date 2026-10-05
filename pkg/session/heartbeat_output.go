package session

import (
	"encoding/json"
	"strings"
	"sync"
)

// heartbeatOutput owns immutable frames only. Never hold mu while calling
// world/session callbacks, acquiring sendMu, or doing transport/persistence work.
// Active remains true through commit so concurrent output cannot overtake it.
type heartbeatOutput struct {
	mu      sync.Mutex
	active  bool
	frames  []heartbeatFrame
	pending map[*Session]int
	closing map[*Session]bool
	prompts map[*Session]bool
}

type heartbeatFrame struct {
	session *Session
	message []byte
	text    string
	note    bool
	literal bool
	close   bool
}

// BeginHeartbeatOutput starts one live turn or the entire deterministic burst.
func (m *Manager) BeginHeartbeatOutput() {
	b := &m.outputBatch
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.active {
		panic("overlapping heartbeat output transactions")
	}
	b.active = true
	b.pending = make(map[*Session]int)
	b.closing = make(map[*Session]bool)
	b.prompts = make(map[*Session]bool)
}

// stageHeartbeat returns false outside a turn: callers then execute exactly
// their existing delivery path. The bounded channel's existing drop policy
// applies to the sum of queued and staged frames, without blocking a callback.
func (s *Session) stageHeartbeat(message []byte, text string, note bool) bool {
	return s.stageHeartbeatText(message, text, note, false)
}

func (s *Session) stageHeartbeatText(message []byte, text string, note, literal bool) bool {
	if s.manager == nil {
		return false
	}
	b := &s.manager.outputBatch
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.active {
		return s.outputDiscarded.Load()
	}
	if b.closing[s] || s.outputDiscarded.Load() || len(s.send)+b.pending[s] >= cap(s.send) {
		return true
	}
	// Preserve transport envelopes; recognize player text only for commit
	// bookkeeping/snoop. Raw controls never request an extra prompt.
	if text == "" {
		if f, ok := RenderTerminalFrame(message); ok && f.Kind == FrameText {
			text = f.Text
			var sm struct {
				Data struct {
					Type string `json:"type"`
				} `json:"data"`
			}
			if json.Unmarshal(message, &sm) == nil && sm.Data.Type != "raw" {
				note = true
			}
		}
	}
	b.frames = append(b.frames, heartbeatFrame{session: s, message: append([]byte(nil), message...), text: text, note: note, literal: literal})
	b.pending[s]++
	return true
}

func (s *Session) deferHeartbeatPrompt() bool {
	if s.manager == nil {
		return false
	}
	b := &s.manager.outputBatch
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.active {
		return false
	}
	if !b.closing[s] && !s.outputDiscarded.Load() {
		b.prompts[s] = true
	}
	return true
}

// EndHeartbeatOutput drains while still active, then appends requested and
// asynchronous prompts after all accepted output. No output lock encloses
// SendPrompt (which reads player/editor/GMCP state).
func (m *Manager) EndHeartbeatOutput() {
	b := &m.outputBatch
	type snoopText struct {
		strings.Builder
		actual  strings.Builder
		literal bool
	}
	snooped := make(map[*Session]*snoopText)
	flushSnoop := func(s *Session) {
		if text := snooped[s]; text != nil {
			delete(snooped, s)
			if !s.outputDiscarded.Load() {
				if text.literal {
					s.forwardSnoopText(text.actual.String(), true)
				} else {
					s.forwardSnoopOutput(text.String())
				}
			}
		}
	}
	for {
		b.mu.Lock()
		if len(b.frames) > 0 {
			frame := b.frames[0]
			b.frames[0] = heartbeatFrame{}
			b.frames = b.frames[1:]
			if !frame.close {
				b.pending[frame.session]--
			}
			b.mu.Unlock()
			if frame.close {
				flushSnoop(frame.session)
				frame.session.closeSendNow()
			} else {
				if frame.deliver() && frame.text != "" {
					text := snooped[frame.session]
					if text == nil {
						text = &snoopText{}
						snooped[frame.session] = text
					}
					text.WriteString(frame.text)
					if f, ok := RenderTerminalFrame(frame.message); ok {
						text.actual.WriteString(f.Text)
					}
					text.literal = text.literal || frame.literal
				}
			}
			continue
		}
		if len(snooped) > 0 {
			b.mu.Unlock()
			for s := range snooped {
				flushSnoop(s)
			}
			continue
		}
		{
			requested := b.prompts
			b.prompts = make(map[*Session]bool)
			b.mu.Unlock()
			m.mu.RLock()
			sessions := make([]*Session, 0, len(m.sessions))
			for _, s := range m.sessions {
				sessions = append(sessions, s)
			}
			m.mu.RUnlock()
			for _, s := range sessions {
				if !s.inputBusy.Load() && s.hasTransport() && !s.SendClosed() && s.outputSincePrompt.Load() > 0 && !s.IsCharCreating() && !s.IsMenuActive() && !s.IsPaging() {
					requested[s] = true
				}
			}
			for s := range requested {
				if !s.outputDiscarded.Load() && !s.SendClosed() {
					s.sendPromptNow()
				}
			}
			if len(requested) > 0 {
				continue
			}
		}
		b.mu.Lock()
		if len(b.frames) > 0 || len(b.prompts) > 0 {
			b.mu.Unlock()
			continue
		}
		b.active = false
		b.pending = nil
		b.closing = nil
		b.prompts = nil
		b.mu.Unlock()
		return
	}
}

func (f heartbeatFrame) deliver() bool {
	s := f.session
	delivered := false
	s.sendMu.RLock()
	if !s.sendClosed && !s.outputDiscarded.Load() {
		select {
		case s.send <- f.message:
			delivered = true
			if f.note {
				s.notePlayerOutput()
			}
		default:
		}
	}
	s.sendMu.RUnlock()
	return delivered
}

// queueHeartbeatClose is a FIFO barrier, distinct from immediate discard.
func (s *Session) queueHeartbeatClose() bool {
	if s.manager == nil {
		return false
	}
	b := &s.manager.outputBatch
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.active {
		return false
	}
	if !b.closing[s] {
		b.closing[s] = true
		b.frames = append(b.frames, heartbeatFrame{session: s, close: true})
	}
	return true
}

// discardHeartbeatOutput prevents even a detached commit frame from reaching
// a competing writer. The send lock never nests with the output batch lock.
func (s *Session) discardHeartbeatOutput() {
	s.sendMu.Lock()
	s.outputDiscarded.Store(true)
	s.sendMu.Unlock()
	s.outputSincePrompt.Store(0)
	s.closeSendNow()
}

// orderlyTransportDrain leaves transport shutdown to its writer after the
// accepted FIFO barrier closes send. Both live transports already own that
// drain/close path. Immediate idle discard never enters this branch.
func (s *Session) orderlyTransportDrain() bool {
	if s.manager == nil || s.outputDiscarded.Load() {
		return false
	}
	b := &s.manager.outputBatch
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.active && b.closing[s]
}

func (m *Manager) heartbeatOutputActive() bool {
	m.outputBatch.mu.Lock()
	defer m.outputBatch.mu.Unlock()
	return m.outputBatch.active
}
