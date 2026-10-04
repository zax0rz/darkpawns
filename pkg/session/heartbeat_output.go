package session

import "sync"

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
	b.frames = append(b.frames, heartbeatFrame{session: s, message: append([]byte(nil), message...), text: text, note: note})
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
				frame.session.closeSendNow()
			} else {
				frame.deliver()
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

func (f heartbeatFrame) deliver() {
	s := f.session
	delivered := false
	s.sendMu.RLock()
	if !s.sendClosed && !s.outputDiscarded.Load() {
		select {
		case s.send <- f.message:
			delivered = true
		default:
		}
	}
	s.sendMu.RUnlock()
	if delivered {
		if f.note {
			s.notePlayerOutput()
		}
		if f.text != "" {
			s.forwardSnoopOutput(f.text)
		}
	}
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
