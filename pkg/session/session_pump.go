// Package session manages WebSocket connections and player sessions.
package session

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"os"
	"runtime/debug"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/metrics"
)

// preAuthIdleTimeoutNanos is the DP-912 login idle timeout for an
// unauthenticated WebSocket socket, in nanoseconds. It mirrors the telnet
// listener's loginIdleTimeout on the WebSocket path: an unauthenticated socket
// that stops sending data is dropped, because writePump's pings would otherwise
// keep it alive forever (browsers answer pings automatically). Data frames
// refresh it; pong control frames do not — a live socket is not a live login
// attempt. The LOGIN_IDLE_TIMEOUT override (seconds) governs both transports.
// It is atomic because readPump loads it from its own goroutine while tests
// store a shorter value, mirroring authReadDeadlineNanos.
var preAuthIdleTimeoutNanos atomic.Int64

// preAuthIdleTimeout returns the current pre-auth (DP-912) login idle timeout.
func preAuthIdleTimeout() time.Duration {
	return time.Duration(preAuthIdleTimeoutNanos.Load())
}

func init() {
	preAuthIdleTimeoutNanos.Store(int64(120 * time.Second))
	authReadDeadlineNanos.Store(int64(defaultAuthReadDeadline))
	if v := os.Getenv("LOGIN_IDLE_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			preAuthIdleTimeoutNanos.Store(int64(time.Duration(n) * time.Second))
		} else {
			slog.Warn("LOGIN_IDLE_TIMEOUT invalid, using default", "value", v, "default", preAuthIdleTimeout().String())
		}
	}
}

// defaultAuthReadDeadline is the production liveness clock for an authenticated
// (playing) WebSocket session.
const defaultAuthReadDeadline = 60 * time.Second

// authReadDeadlineNanos is the read deadline for an authenticated (playing)
// WebSocket session, in nanoseconds. Unlike the telnet transport, whose playing
// descriptor carries no transport deadline at all (DP-1385), the browser pump
// keeps a liveness clock refreshed on every pong: writePump pings every 54 s
// and a browser answers automatically, so an idle-but-live player is never
// dropped — only a socket that stops answering pings is. It is atomic because
// the session's readPump loads it from its own goroutine while the DP-1385 test
// stores a shorter value.
var authReadDeadlineNanos atomic.Int64

// authReadDeadline returns the current authenticated-session read deadline.
func authReadDeadline() time.Duration {
	return time.Duration(authReadDeadlineNanos.Load())
}

// readDeadline is the read deadline for the WebSocket pump: the pong-refreshed
// liveness clock once authenticated, the DP-912 login idle timeout (refreshed
// only by data frames) before that.
func (s *Session) readDeadline() time.Duration {
	if s.authenticated {
		return authReadDeadline()
	}
	return preAuthIdleTimeout()
}

func (s *Session) readPump() {
	// The WebSocket connection's life is this function, not HandleWebSocket,
	// which returns as soon as it spawns this goroutine. Counting at the
	// upgrade would leak whenever a ban check rejected the socket in between.
	metrics.ConnectionOpened()
	defer func() {
		metrics.ConnectionClosed()
		if r := recover(); r != nil {
			slog.Error(
				"CRITICAL PANIC RECOVERED in readPump",
				"player", s.playerName,
				"recover", r,
				"stack", string(debug.Stack()),
			)
			s.sendError("An internal server error occurred. Your connection has been reset.")
			s.manager.UnregisterSession(s)
		}
		// Always decrement IP connection count (C5 leak fix)
		if !s.connCountDecremented && (s.request != nil || s.remoteIP != "") {
			s.connCountDecremented = true
			ip := s.RemoteIP()
			if ip != "" {
				s.manager.ipConnMu.Lock()
				s.manager.ipConnCount[ip]--
				if s.manager.ipConnCount[ip] <= 0 {
					delete(s.manager.ipConnCount, ip)
				}
				s.manager.ipConnMu.Unlock()
			}
		}
		// A playing character whose connection drops stays in the world,
		// linkdead, as C's close_socket leaves it and as telnet does
		// (DP-1323); the linkdead reaper extracts it later. Anything else is
		// cleaned up now.
		// An orderly close (the server closed the send channel: goodbye, a
		// refused login, quit) leaves the writer flushing the last message.
		// Closing the socket now would drop it and end the connection with
		// an abnormal close, so the reader waits for the writer first.
		if s.SendClosed() && s.writerDone != nil {
			select {
			case <-s.writerDone:
			case <-time.After(5 * time.Second):
			}
		}
		s.finishWebSocketTransport()
		s.Close()
	}()

	s.conn.SetReadLimit(16384) // 16KB max message size (C4)
	_ = s.conn.SetReadDeadline(time.Now().Add(s.readDeadline()))
	s.conn.SetPongHandler(func(string) error {
		// Pongs prove a live socket, not a live login attempt: before
		// authentication they must not extend the deadline, or a parked
		// client holds a connection slot forever by answering pings.
		if s.authenticated {
			_ = s.conn.SetReadDeadline(time.Now().Add(authReadDeadline()))
		}
		return nil
	})

	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() && !s.authenticated {
				slog.Info("WebSocket pre-auth idle timeout, disconnecting", "ip", s.RemoteIP())
			}
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Error("WebSocket error", "error", err)
			}
			break
		}

		// DP-902 + DP-928: record inbound activity (WebSocket path). The telnet
		// path calls the same helper from its input loop.
		s.OnInboundActivity()

		if err := s.handleMessage(message); err != nil {
			slog.Error("handle message error", "error", err)
			s.sendErrorWithState(err)
		}

		// A data frame is real client activity: refresh the deadline (this is
		// what keeps an interactive login or char-creation flow alive past the
		// pre-auth idle timeout), at the clock the post-handleMessage
		// authentication state calls for.
		_ = s.conn.SetReadDeadline(time.Now().Add(s.readDeadline()))

		// If handleLogin rejected the client (wrong password, invalid name,
		// banned, rate-limited, etc.), it calls CloseSend instead of Close so
		// writePump can flush the error message. Detect that here and break
		// the read loop so the deferred cleanup (Unregister + Close) runs.
		if s.SendClosed() {
			break
		}
	}
}

// writePump writes messages to the WebSocket.
func (s *Session) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		if s.writerDone != nil {
			defer close(s.writerDone)
		}
		if r := recover(); r != nil {
			slog.Error(
				"CRITICAL PANIC RECOVERED in writePump",
				"player", s.playerName,
				"recover", r,
				"stack", string(debug.Stack()),
			)
		}
		ticker.Stop()
		s.Close()
		// A failed ping can make the writer exit before the reader notices
		// EOF. Decide linkdead retention here too, through the same once-only
		// path, so the writer cannot remove a playing character first.
		s.finishWebSocketTransport()
	}()

	for {
		select {
		case message, ok := <-s.send:
			_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = s.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if IsInputMarkFrame(message) {
				if s.browserTerminal.Load() {
					if f, ok := RenderTerminalFrame(message); ok {
						s.TrackPrompt(f)
					}
				}
				continue
			}
			if s.browserTerminal.Load() {
				var send bool
				if message, send = renderForBrowserTerminalTracked(s, message); !send {
					continue
				}
			}

			// Stamp a sequence number on every outbound message. Unmarshal into
			// a generic map, add seq, re-marshal: raw JSON string injection was
			// fragile.
			var raw map[string]interface{}
			if err := json.Unmarshal(message, &raw); err == nil {
				// P0-1: stamp sequence number on every outbound message
				s.msgSeq++
				raw["seq"] = s.msgSeq

				if marshaled, err := json.Marshal(raw); err == nil {
					message = marshaled
				}
			}

			_ = s.conn.WriteMessage(websocket.TextMessage, message)

		case <-s.TransportDone():
			// The connection went linkdead; the session keeps its send channel
			// for when the character is reattached or extracted.
			return

		case <-ticker.C:
			_ = s.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := s.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (s *Session) finishWebSocketTransport() {
	s.transportCleanupOnce.Do(func() {
		// C close_socket frees the descriptor, so a name held during entry is
		// free again (ban.c:266-268). A pre-login session is not registered,
		// so UnregisterSession below does not release it.
		s.releaseEntryName()
		if !s.manager.HandleTransportDisconnect(s) {
			s.manager.UnregisterSession(s)
		}
	})
}

// handleMessage processes incoming WebSocket messages.
func (s *Session) handleMessage(data []byte) error {
	if s.SendClosed() {
		return ErrNotAuthenticated
	}
	var msg ClientMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	switch msg.Type {
	case MsgLine, MsgLogin, MsgCommand, MsgCharInput, MsgPagerInput:
		// Structured clients get the same process_input character pass as a
		// terminal line (src/comm.c:1965-1982).
		filtered, err := cInputJSON(msg.Data)
		if err != nil {
			return err
		}
		msg.Data = filtered
	}

	switch msg.Type {
	case MsgTerminal:
		s.startBrowserTerminal()
		return nil
	case MsgLine:
		return s.handleTerminalLine(msg.Data)
	case MsgLogin:
		return s.handleLogin(msg.Data)
	case MsgCommand:
		// The DP_CLOCK harness control reaches a WebSocket session as an
		// ordinary command line (the browser client sends every line that
		// way); telnet consumes it before building the message.
		if s.isClockControlMessage(msg.Data) {
			return nil
		}
		if !s.authenticated || s.menuActive || s.charCreating || s.creationSaved {
			return ErrNotAuthenticated
		}
		return s.handleCommand(msg.Data)
	case MsgSubscribe:
		if !s.authenticated {
			return ErrNotAuthenticated
		}
		return s.handleSubscribe(msg.Data)
	case MsgCharInput:
		if s.menuActive {
			return s.handleMenuInput(msg.Data)
		}
		if s.charCreating {
			return s.handleCharInput(msg.Data)
		}
		return ErrNotInCharCreation
	case MsgPagerInput:
		// Pager navigation (DP-1195). While paging, input lines route here
		// instead of the command interpreter — mirrors C's showstr_count
		// routing (comm.c:617). The transport forks only send pager_input
		// while IsPaging(); guard anyway in case a client mis-sends it.
		if !s.IsPaging() {
			return ErrNotInCharCreation
		}
		return s.handlePagerInputMsg(msg.Data)
	default:
		return ErrUnknownMessageType
	}
}

// OnInboundActivity records that the session received inbound traffic. Called
// by both the WebSocket readPump and the telnet input loop so both protocols
// share the same transport-liveness bookkeeping (DP-902, DP-928). It performs
// no idle-lifecycle work: C resets char_specials.timer only when a command
// line is dequeued for dispatch (comm.c:600-601), never for raw transport
// chatter — see resetIdleOnCommand.
func (s *Session) OnInboundActivity() {
	s.lastActive.Store(time.Now().UnixNano())
}

// SetLastActiveForTest allows tests in other packages to manipulate the
// lastActive timestamp without exporting the field.
func (s *Session) SetLastActiveForTest(ts int64) {
	s.lastActive.Store(ts)
}

// resetIdleOnCommand mirrors comm.c:600-608: a command line dequeued for
// dispatch resets the character's idle timer and returns a voided character
// to their previous room BEFORE routing/dispatch. This is the shared
// command-consumption seam: Telnet reaches it via TerminalLine →
// handleCommand and WebSocket via handleMessage(MsgCommand) → handleCommand
// on the immediate path, and via Manager.DrainInputQueues when the command
// was deferred behind wait state — matching C's game-loop dequeue point in
// both cases.
func (s *Session) resetIdleOnCommand() {
	if !s.authenticated || s.player == nil {
		return
	}
	s.player.SetIdleTimer(0)
	s.maybeReturnFromVoid()
}

// maybeReturnFromVoid returns a player from the void room to their previous
// room before their command dispatches, exactly the state/room half of
// comm.c:602-608. The idle-timer reset is resetIdleOnCommand's job and no
// longer depends on WasInRoom being set.
func (s *Session) maybeReturnFromVoid() {
	p := s.player
	if p == nil {
		return
	}

	wasIn := p.GetWasInRoom()
	roomVNum := p.GetRoom()

	if wasIn == 0 {
		return
	}

	p.SetWasInRoom(0)

	if roomVNum == wasIn {
		return
	}

	if err := s.manager.world.PlayerTransfer(p, wasIn); err != nil {
		slog.Warn("return from void failed", "player", s.playerName, "error", err)
		return
	}
	// act("$n has returned.", TRUE, ch, 0, 0, TO_ROOM) — comm.c:607. Act,
	// not SendToRoom: the returnee is back in wasIn and must not receive
	// their own return line, and hidden viewers follow C's can-see gate.
	game.Act(s.manager.world, true, p, nil, nil, nil, "$n has returned.", "", game.ToRoom)
}

// handleLogin authenticates a player.
