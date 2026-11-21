package command

import (
	"strings"
	"testing"
)

// mockCommandSession satisfies common.CommandSession
type mockCommandSession struct {
	messages []string
}

func (m *mockCommandSession) Send(msg string)        { m.messages = append(m.messages, msg) }
func (m *mockCommandSession) Close()                 {}
func (m *mockCommandSession) GetPlayer() any         { return nil }
func (m *mockCommandSession) GetPlayerName() string  { return "TestUser" }
func (m *mockCommandSession) GetPlayerRoomVNum() int { return 1001 }
func (m *mockCommandSession) IsAuthenticated() bool  { return true }
func (m *mockCommandSession) HasPlayer() bool        { return true }
func (m *mockCommandSession) GetPlayerLevel() int    { return 1 }

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewRegistry()

	handlerCalled := false
	var handler Handler = func(ctx ExecutionContext, cmd string, args []string) error {
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
	h := func(ctx ExecutionContext, cmd string, args []string) error { return nil }

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
