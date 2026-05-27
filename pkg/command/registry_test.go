package command

import (
	"errors"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/common"
)

// mockCommandSession satisfies common.CommandSession
type mockCommandSession struct {
	messages []string
}

func (m *mockCommandSession) Send(msg string)               { m.messages = append(m.messages, msg) }
func (m *mockCommandSession) Close()                        {}
func (m *mockCommandSession) GetPlayer() interface{}        { return nil }
func (m *mockCommandSession) GetPlayerName() string         { return "TestUser" }
func (m *mockCommandSession) GetPlayerRoomVNum() int        { return 1001 }
func (m *mockCommandSession) IsAuthenticated() bool         { return true }
func (m *mockCommandSession) HasPlayer() bool               { return true }
func (m *mockCommandSession) GetPlayerLevel() int           { return 1 }

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewRegistry()

	handlerCalled := false
	var handler Handler = func(s common.CommandSession, args []string) error {
		handlerCalled = true
		return nil
	}

	r.Register("look", handler, "Look around", 1, 0, "l", "examine")

	// Lookup primary name
	entry, ok := r.Lookup("look")
	if !ok {
		t.Fatal("expected to find primary command 'look'")
	}
	if entry.HelpText != "Look around" {
		t.Errorf("expected HelpText 'Look around', got %q", entry.HelpText)
	}

	// Lookup case-insensitively
	_, ok = r.Lookup("LOOK")
	if !ok {
		t.Error("expected to find case-insensitive command 'LOOK'")
	}

	// Lookup aliases
	for _, alias := range []string{"l", "examine"} {
		entry, ok = r.Lookup(alias)
		if !ok {
			t.Fatalf("expected to find alias %q", alias)
		}
		if entry.Name != "look" {
			t.Errorf("expected alias %q to map to 'look', got %q", alias, entry.Name)
		}
	}

	// Execute command
	sess := &mockCommandSession{}
	err := r.Execute(sess, "examine", []string{"north"})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !handlerCalled {
		t.Error("expected command handler to be executed")
	}
}

func TestRegistry_GetAll(t *testing.T) {
	r := NewRegistry()
	h := func(s common.CommandSession, args []string) error { return nil }

	r.Register("look", h, "look help", 1, 0, "l")
	r.Register("score", h, "score help", 1, 0, "sc")

	all := r.GetAll()
	// Should have exactly 2 unique entries (not counting aliases)
	if len(all) != 2 {
		t.Errorf("expected 2 unique entries, got %d", len(all))
	}

	foundLook := false
	foundScore := false
	for _, entry := range all {
		if entry.Name == "look" {
			foundLook = true
		}
		if entry.Name == "score" {
			foundScore = true
		}
	}

	if !foundLook || !foundScore {
		t.Error("expected to find both 'look' and 'score' in GetAll")
	}
}

func TestRegistry_ExecuteUnknownCommand(t *testing.T) {
	r := NewRegistry()
	sess := &mockCommandSession{}
	err := r.Execute(sess, "nonexistent", nil)
	if err == nil {
		t.Error("expected error when executing unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRegistry_Middleware(t *testing.T) {
	r := NewRegistry()

	// Add a test middleware that increments a counter
	mw1Called := 0
	var mw1 Middleware = func(next Handler) Handler {
		return func(s common.CommandSession, args []string) error {
			mw1Called++
			return next(s, args)
		}
	}

	// Add another middleware that intercepts and returns error
	var mwIntercept Middleware = func(next Handler) Handler {
		return func(s common.CommandSession, args []string) error {
			if len(args) > 0 && args[0] == "intercept" {
				return errors.New("intercepted")
			}
			return next(s, args)
		}
	}

	r.Use(mw1)
	r.Use(mwIntercept)

	handlerCalled := false
	r.Register("say", func(s common.CommandSession, args []string) error {
		handlerCalled = true
		return nil
	}, "Say something", 1, 0)

	sess := &mockCommandSession{}

	// Execution without intercept
	err := r.Execute(sess, "say", []string{"hello"})
	if err != nil {
		t.Fatalf("say failed: %v", err)
	}
	if mw1Called != 1 {
		t.Errorf("expected mw1 to be called once, got %d", mw1Called)
	}
	if !handlerCalled {
		t.Error("handler should be called")
	}

	// Reset state
	handlerCalled = false

	// Execution with intercept trigger
	err = r.Execute(sess, "say", []string{"intercept"})
	if err == nil {
		t.Error("expected intercepted error")
	}
	if err.Error() != "intercepted" {
		t.Errorf("expected 'intercepted', got %v", err)
	}
	if mw1Called != 2 {
		t.Errorf("expected mw1 to be called twice, got %d", mw1Called)
	}
	if handlerCalled {
		t.Error("handler should NOT be called under intercept")
	}
}

func TestMiddleware_Whitelist(t *testing.T) {
	r := NewRegistry()
	r.Use(WhitelistMiddleware("look", "score"))

	r.Register("look", func(s common.CommandSession, args []string) error { return nil }, "", 1, 0)
	r.Register("shout", func(s common.CommandSession, args []string) error { return nil }, "", 1, 0)

	sess := &mockCommandSession{}

	// Allowed command
	err := r.Execute(sess, "look", []string{"look"})
	if err != nil {
		t.Errorf("allowed command failed: %v", err)
	}

	// Blocked command
	err = r.Execute(sess, "shout", []string{"shout"})
	if err == nil {
		t.Error("expected blocked command to return error")
	}
}

func TestMiddleware_Logging(t *testing.T) {
	r := NewRegistry()
	r.Use(LoggingMiddleware())

	r.Register("look", func(s common.CommandSession, args []string) error { return nil }, "", 1, 0)
	r.Register("errorcmd", func(s common.CommandSession, args []string) error { return errors.New("err") }, "", 1, 0)

	sess := &mockCommandSession{}

	// Test successful log
	_ = r.Execute(sess, "look", []string{"look"})

	// Test error log
	_ = r.Execute(sess, "errorcmd", []string{"errorcmd"})
}
