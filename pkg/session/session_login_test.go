package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// loginMsg marshals a ClientMessage with type "login" and the given LoginData.
func loginMsg(name, password string) json.RawMessage {
	login := LoginData{
		PlayerName: name,
		Password:   password,
	}
	b, _ := json.Marshal(login)
	return b
}

// callHandleLogin runs handleLogin in a goroutine with panic recovery.
// Needed because error paths call s.conn.Close() on nil conn.
// Returns (error, panicked).
func callHandleLogin(s *Session, data json.RawMessage) (error, bool) {
	type result struct {
		err      error
		panicked bool
	}
	ch := make(chan result, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- result{panicked: true}
			}
		}()
		err := s.handleLogin(data)
		ch <- result{err: err}
	}()
	select {
	case r := <-ch:
		return r.err, r.panicked
	case <-time.After(5 * time.Second):
		panic("callHandleLogin: timeout (bcrypt may be slow, or deadlock)")
	}
}

// drainSend reads one message from s.send without blocking.
func drainSend(s *Session) ([]byte, bool) {
	select {
	case msg := <-s.send:
		return msg, true
	default:
		return nil, false
	}
}

// drainSendWait reads one message with a short wait.
func drainSendWait(s *Session, d time.Duration) ([]byte, bool) {
	select {
	case msg := <-s.send:
		return msg, true
	case <-time.After(d):
		return nil, false
	}
}

// unmarshalServerMsg parses a raw message into ServerMessage.
func unmarshalServerMsg(t *testing.T, raw []byte) ServerMessage {
	t.Helper()
	var srv ServerMessage
	if err := json.Unmarshal(raw, &srv); err != nil {
		t.Fatalf("unmarshalServerMsg: %v", err)
	}
	return srv
}

// ---------------------------------------------------------------------------
// TestHandleLogin_EmptyFields — empty player_name returns ErrInvalidPlayerName
// ---------------------------------------------------------------------------

func TestHandleLogin_EmptyFields(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	data := loginMsg("", "")
	err := s.handleLogin(data)

	if err != ErrInvalidPlayerName {
		t.Errorf("expected ErrInvalidPlayerName, got %v", err)
	}
	// No message should have been sent
	if msg, ok := drainSend(s); ok {
		t.Errorf("unexpected message on send channel: %s", msg)
	}
}

// ---------------------------------------------------------------------------
// TestHandleLogin_PasswordRequired — valid name, missing password, no DB
// ---------------------------------------------------------------------------

func TestHandleLogin_PasswordRequired(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	data := loginMsg("Tester", "")
	_, panicked := callHandleLogin(s, data)

	// Error path calls conn.Close() on nil conn → panic after sendError
	if !panicked {
		// If the implementation changed and no longer panics, verify an error was sent
		msg, ok := drainSend(s)
		if !ok {
			t.Fatal("expected either panic (nil conn.Close) or error message on send channel")
		}
		srv := unmarshalServerMsg(t, msg)
		if srv.Type != MsgError {
			t.Errorf("expected error message type, got %q", srv.Type)
		}
		return
	}

	// The sendError call writes to send BEFORE conn.Close() panics
	msg, ok := drainSendWait(s, 100*time.Millisecond)
	if !ok {
		t.Fatal("expected error message on send channel before conn.Close() panic")
	}
	srv := unmarshalServerMsg(t, msg)
	if srv.Type != MsgError {
		t.Errorf("message type = %q, want %q", srv.Type, MsgError)
	}
	b, _ := json.Marshal(srv.Data)
	var errData ErrorData
	if err := json.Unmarshal(b, &errData); err != nil {
		t.Fatalf("unmarshal ErrorData: %v", err)
	}
	if !strings.Contains(errData.Message, "Password") {
		t.Errorf("expected password error, got %q", errData.Message)
	}
}

// ---------------------------------------------------------------------------
// TestHandleLogin_NewCharacter — valid name + password → char creation starts
// ---------------------------------------------------------------------------

func TestHandleLogin_NewCharacter(t *testing.T) {
	m := makeTestManager(t)
	s := makeCharSession(t, m)

	data := loginMsg("NewChar", "test123")
	err, panicked := callHandleLogin(s, data)

	if panicked {
		t.Fatal("handleLogin panicked unexpectedly on new-char creation path")
	}
	if err != nil {
		t.Fatalf("handleLogin returned error: %v", err)
	}

	// char creation should have started
	if !s.charCreating {
		t.Error("charCreating should be true after new character login")
	}
	if s.charName != "NewChar" {
		t.Errorf("charName = %q, want NewChar", s.charName)
	}
	if s.charPassword == "" {
		t.Error("charPassword should be set after new character login")
	}

	// A char_create prompt should be on the send channel
	msg, ok := drainSendWait(s, 500*time.Millisecond)
	if !ok {
		t.Fatal("expected char_create message on send channel")
	}
	srv := unmarshalServerMsg(t, msg)
	if srv.Type != MsgCharCreate {
		t.Errorf("message type = %q, want %q", srv.Type, MsgCharCreate)
	}
}

// ---------------------------------------------------------------------------
// TestHandleLogin_ExistingPlayer — pre-registered session name; login starts
// char creation (no-DB path — no session takeover without DB)
// ---------------------------------------------------------------------------

func TestHandleLogin_ExistingPlayer(t *testing.T) {
	m := makeTestManager(t)

	// Pre-register a session for "existing"
	existing := makeTestSession(t, m, "Existing", 1001, true)
	m.mu.Lock()
	m.sessions["existing"] = existing
	m.mu.Unlock()

	// New login attempt for the same name
	s := makeCharSession(t, m)
	data := loginMsg("Existing", "somepassword")
	err, panicked := callHandleLogin(s, data)

	if panicked {
		t.Fatal("handleLogin panicked unexpectedly")
	}
	if err != nil {
		t.Fatalf("handleLogin returned error: %v", err)
	}

	// Without DB, always enters char creation regardless of existing session
	if !s.charCreating {
		t.Error("charCreating should be true; no-DB path always starts char creation")
	}
	// A char_create prompt should arrive
	msg, ok := drainSendWait(s, 500*time.Millisecond)
	if !ok {
		t.Fatal("expected char_create message on send channel")
	}
	srv := unmarshalServerMsg(t, msg)
	if srv.Type != MsgCharCreate {
		t.Errorf("message type = %q, want %q", srv.Type, MsgCharCreate)
	}
}

// ---------------------------------------------------------------------------
// TestHandleCommand_NotAuthenticated — command rejected before auth
// ---------------------------------------------------------------------------

func TestHandleCommand_NotAuthenticated(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Ghost", 1001, false) // authenticated=false

	msg, _ := json.Marshal(ClientMessage{
		Type: MsgCommand,
		Data: json.RawMessage(`{"command":"look"}`),
	})
	err := s.handleMessage(msg)
	if err != ErrNotAuthenticated {
		t.Errorf("expected ErrNotAuthenticated, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// TestHandleCommand_Authenticated — "look" dispatches and sends room state
// ---------------------------------------------------------------------------

func TestHandleCommand_Authenticated(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.limiter = rate.NewLimiter(rate.Inf, 1000)

	data, _ := json.Marshal(CommandData{Command: "look"})
	err := s.handleCommand(data)
	if err != nil {
		t.Fatalf("handleCommand(look) returned error: %v", err)
	}

	msg, ok := drainSendWait(s, 500*time.Millisecond)
	if !ok {
		t.Fatal("expected message on send channel after look")
	}
	srv := unmarshalServerMsg(t, msg)
	if srv.Type != MsgState && srv.Type != MsgText && srv.Type != MsgEvent {
		t.Errorf("unexpected message type %q from look; want state/text/event", srv.Type)
	}
}

// ---------------------------------------------------------------------------
// TestHandleCommand_UnknownCommand — "xyzzy" sends "Unknown command" text
// ---------------------------------------------------------------------------

func TestHandleCommand_UnknownCommand(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.limiter = rate.NewLimiter(rate.Inf, 1000)

	data, _ := json.Marshal(CommandData{Command: "xyzzy"})
	err := s.handleCommand(data)
	if err != nil {
		t.Fatalf("handleCommand(xyzzy) returned error: %v", err)
	}

	msg, ok := drainSendWait(s, 500*time.Millisecond)
	if !ok {
		t.Fatal("expected 'unknown command' message on send channel")
	}
	srv := unmarshalServerMsg(t, msg)
	if srv.Type != MsgText {
		t.Errorf("message type = %q, want %q", srv.Type, MsgText)
	}
	b, _ := json.Marshal(srv.Data)
	var td TextData
	if err := json.Unmarshal(b, &td); err != nil {
		t.Fatalf("unmarshal TextData: %v", err)
	}
	// C interpreter.c:916 answers any unmatched command with "Huh?!?".
	if !strings.Contains(td.Text, "Huh?!?") {
		t.Errorf("expected 'Huh?!?' in response, got %q", td.Text)
	}
}
