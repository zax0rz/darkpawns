package session

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// C's WHOD timing constants (src/whod.c).
const (
	// whodDelayTime is WHOD_DELAY_TIME: C holds the socket open after the last
	// write before closing it (src/whod.c:42,427-441).
	whodDelayTime = 3 * time.Second
	// whodPollInterval is how often the supervisor reconciles C's on/off bits.
	// C polls whod_loop from the game loop with a 100µs select
	// (src/whod.c:303-304, src/comm.c:555); only the reconcile latency differs.
	whodPollInterval = 100 * time.Millisecond
)

// whodListener ports C's WHOD daemon (src/whod.c). C polls a six-state machine
// (OPENING/OPEN/DELAY/END/CLOSING/CLOSED) from the game loop and serves one
// connection at a time: accept, write, hold WHOD_DELAY_TIME, close, then accept
// the next. This port keeps C's bytes and its four producers but serves each
// accepted connection on its own goroutine, so requests overlap instead of
// queueing. Per-client bytes are identical; the concurrency model is not.
// DP-1415 approved the goroutine-per-request shape (Claude/Zach, 2026-10-10);
// see docs/fidelity/depth/handoff/2026-10-11-dp-1415-whod-listener.md.
type whodListener struct {
	manager *Manager
	world   *game.World
	port    int
	delay   time.Duration

	mu      sync.Mutex
	ln      net.Listener // nil while C's SHOW_OFF has the daemon closed
	stopped bool
	done    chan struct{}
	stopOne sync.Once
}

// StartWhodListener opens C's WHOD port and starts the supervisor. It ports
// init_whod (src/whod.c:246-253), including C's order: log the open, then bind.
func (m *Manager) StartWhodListener(port int) error {
	return m.startWhodListener(port, whodDelayTime)
}

// startWhodListener is StartWhodListener with the post-write hold injected
// (WHOD_DELAY_TIME). Tests bind an ephemeral port and hold for zero.
func (m *Manager) startWhodListener(port int, delay time.Duration) error {
	l := &whodListener{
		manager: m,
		world:   m.world,
		port:    port,
		delay:   delay,
		done:    make(chan struct{}),
	}
	m.mu.Lock()
	m.whod = l
	m.mu.Unlock()

	// src/whod.c:250: LOG("WHOD port opened.") precedes init_socket.
	game.MudLog("WHOD port opened.", game.MudlogBrief, game.LVL_GOD, true)
	if err := l.listen(); err != nil {
		return err
	}
	go l.supervise()
	return nil
}

// StopWhodListener is close_whod (src/whod.c:260-269) on the shutdown path: it
// closes the port if it is open and logs the close exactly once, mirroring C's
// state != WHOD_CLOSED guard.
func (m *Manager) StopWhodListener() {
	m.mu.Lock()
	l := m.whod
	m.mu.Unlock()
	if l == nil {
		return
	}
	if l.stop() {
		game.MudLog("WHOD port closed.", game.MudlogBrief, game.LVL_GOD, true)
	}
}

// WhodListenerAddr returns the bound address, or "" when the daemon is closed
// (SHOW_OFF) or was never started. It exists for ops and for the listener's
// tests, which bind an ephemeral port.
func (m *Manager) WhodListenerAddr() string {
	m.mu.Lock()
	l := m.whod
	m.mu.Unlock()
	if l == nil {
		return ""
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ln == nil {
		return ""
	}
	return l.ln.Addr().String()
}

// listen binds the port and starts the accept loop.
func (l *whodListener) listen() error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", l.port))
	if err != nil {
		slog.Error("WHOD listener failed", "port", l.port, "error", err)
		return err
	}
	l.mu.Lock()
	l.ln = ln
	l.mu.Unlock()
	go l.accept(ln)
	return nil
}

// accept serves every incoming connection on its own goroutine. The listener's
// close (SHOW_OFF or shutdown) ends the loop with net.ErrClosed.
func (l *whodListener) accept(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				slog.Error("WHOD accept failed", "error", err)
			}
			return
		}
		go l.serve(conn)
	}
}

// serve answers one WHO request: log the served line, then write C's greeting
// followed by the who list, then hold the socket for WHOD_DELAY_TIME.
func (l *whodListener) serve(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	host := whodRequestHost(conn.RemoteAddr(), net.LookupAddr)
	// src/whod.c:322-335: the served line is logged before any byte is written.
	game.MudLog(fmt.Sprintf("WHO request from %s served.", host), game.MudlogBrief, game.LVL_GOD, true)

	// src/whod.c:337-339: SHORT_GREETINGS goes out first, then the list.
	payload := game.WhodGreeting() + l.world.WhodDisplay.BuildWhoList(l.entries())
	if _, err := conn.Write([]byte(payload)); err != nil {
		slog.Error("WHOD write failed", "remote", host, "error", err)
	}
	time.Sleep(l.delay)
}

// entries builds the list C walks in whod_loop: every player with a live
// descriptor, newest descriptor first.
func (l *whodListener) entries() []game.WhodEntry {
	m := l.manager
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	// C walks character_list, which prepends each new character, so the newest
	// descriptor is first. Go has no equivalent list; connectionNumber
	// descending is the same order the users command models.
	sort.SliceStable(sessions, func(i, j int) bool {
		if sessions[i].connectionNumber != sessions[j].connectionNumber {
			return sessions[i].connectionNumber > sessions[j].connectionNumber
		}
		return sessions[i].playerName > sessions[j].playerName
	})

	entries := make([]game.WhodEntry, 0, len(sessions))
	for _, s := range sessions {
		p := s.player
		// DISPLAY_LINKDEAD is 0 (src/whod.c:45): C requires ch->desc, and a
		// linkless player is C's playing character with desc == NULL.
		if p == nil || p.IsLinkless() {
			continue
		}
		entries = append(entries, game.WhodEntry{
			Name:        p.Name,
			Class:       p.GetClass(),
			Level:       p.GetLevel(),
			Title:       p.GetTitle(),
			AFK:         p.GetFlags()&(1<<uint(game.PrfAFK)) != 0,
			Host:        s.MudHost(),
			IsInvisible: l.world.WhodInvisible(p),
		})
	}
	return entries
}

// supervise reconciles C's SHOW_ON/SHOW_OFF bits with the listener: `whod off`
// closes the port (CLOSING → CLOSED, src/whod.c:432-433,453-456) and `whod on`
// reopens it through the OPENING arm (src/whod.c:459-462,294-298).
func (l *whodListener) supervise() {
	ticker := time.NewTicker(whodPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-l.done:
			return
		case <-ticker.C:
			l.reconcile()
		}
	}
}

func (l *whodListener) reconcile() {
	if l.world.WhodDisplay == nil || l.isStopped() {
		return
	}
	mode := l.world.WhodDisplay.GetMode()
	if mode&game.WhodShowOff != 0 {
		if l.closeListener() {
			game.MudLog("WHOD port closed.", game.MudlogBrief, game.LVL_GOD, true)
		}
		return
	}
	if mode&game.WhodShowOn != 0 && l.listenerClosed() {
		if err := l.listen(); err == nil {
			// src/whod.c:296: the reopen arm logs the same payload.
			game.MudLog("WHOD port opened.", game.MudlogBrief, game.LVL_GOD, true)
		}
	}
}

// stop marks the listener stopped and closes it, reporting whether the port had
// still been open (C's close_whod logs only when state != WHOD_CLOSED).
func (l *whodListener) stop() bool {
	l.stopOne.Do(func() {
		close(l.done)
		l.mu.Lock()
		l.stopped = true
		l.mu.Unlock()
	})
	return l.closeListener()
}

func (l *whodListener) closeListener() bool {
	l.mu.Lock()
	ln := l.ln
	l.ln = nil
	l.mu.Unlock()
	if ln == nil {
		return false
	}
	_ = ln.Close()
	return true
}

func (l *whodListener) isStopped() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.stopped
}

func (l *whodListener) listenerClosed() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.ln == nil
}

// whodRequestHost mirrors whod_loop's gethostbyaddr branch (src/whod.c:322-333):
// the resolved name when reverse DNS answers, otherwise C's unpadded quad. C
// formats that quad with plain %d, unlike the padded descriptor host.
func whodRequestHost(addr net.Addr, resolve func(string) ([]string, error)) string {
	if addr == nil {
		return "0.0.0.0"
	}
	hostStr := addr.String()
	if ipOnly, _, err := net.SplitHostPort(hostStr); err == nil {
		hostStr = ipOnly
	}
	ip := net.ParseIP(hostStr)
	if ip == nil {
		return hostStr
	}
	if resolve != nil {
		if names, err := resolve(unpaddedIP(ip)); err == nil && len(names) > 0 && names[0] != "" {
			return names[0]
		}
	}
	return unpaddedIP(ip)
}

// unpaddedIP renders an address the way C's "%d.%d.%d.%d" does, never as an
// IPv4-mapped IPv6 form.
func unpaddedIP(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.%d", v4[0], v4[1], v4[2], v4[3])
	}
	return ip.String()
}
