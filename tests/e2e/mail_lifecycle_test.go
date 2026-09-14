package e2e

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

// TestMailProductionBootBoundary reaches the real server, session, command,
// special-procedure, and postmaster dispatch paths. It intentionally stops at
// the first production precondition: a fresh character has no coins, so the
// postmaster rejects the send before recipient lookup or composition. This is
// a passing characterization of the blocked path, not a production mail
// success claim. The initialization boundary is proved separately by the
// source trace and helper-level hook test.
func TestMailProductionBootBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}
	t.Setenv("DP_FRESH_MUD", "1")

	conn, reader := launchAndDial(t)
	defer conn.Close()

	name := fmt.Sprintf("MailBoot%d", time.Now().UnixNano()%100000)
	createCrownedWarrior(t, conn, reader, name, "mailbootpw")

	// The fresh crowned character enters room 1204, which the native zone
	// fixture populates with VNum 3010 (the assigned postmaster). The room
	// renderer may leave the postmaster and prompt in separate writes, so
	// drain until either the room boundary or the prompt is visible.
	room := readUntil(t, conn, reader, "postman", 10*time.Second)
	room += readFor(t, conn, reader, time.Second)
	t.Logf("postmaster-room-output=%q", room)
	if room == "" {
		t.Fatal("production vehicle did not reach the room containing the assigned postmaster")
	}

	mustWrite(t, conn, "look\r\n")
	look := readFor(t, conn, reader, 3*time.Second)
	t.Logf("production-look-output=%q", look)
	if look == "" {
		t.Fatal("production vehicle did not process a post-login command")
	}

	mustWrite(t, conn, "mail Recipient body\r\n")
	blocked := readFor(t, conn, reader, 3*time.Second)
	t.Logf("production-mail-output=%q", blocked)
	if !strings.Contains(blocked, "$n tells you, 'A stamp costs 50 coins.'") ||
		!strings.Contains(blocked, "$n tells you, '...which I see you can't afford.'") {
		t.Fatalf("production mail path did not expose the expected affordability boundary: %q", blocked)
	}
	t.Logf("production_mail_blocked_stage=stamp_affordability output=%q", strings.TrimSpace(blocked))

	// Keep the connection lifecycle ordinary; this is not a restart vehicle.
	mustWrite(t, conn, "quit\r\n")
}

// readFor drains a bounded interval without assuming a prompt or room marker.
// It is used only after the production character-creation path, where output
// from the initial look and prompt can arrive in separate writes.
func readFor(t *testing.T, conn net.Conn, reader *bufio.Reader, duration time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(duration)
	var out strings.Builder
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
		b, err := reader.ReadByte()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			break
		}
		if b == 255 {
			consumeIAC(reader)
			continue
		}
		out.WriteByte(b)
	}
	return out.String()
}

func createCrownedWarrior(t *testing.T, conn net.Conn, reader *bufio.Reader, name, password string) string {
	t.Helper()
	if got := readUntil(t, conn, reader, "By what name", 10*time.Second); got == "" {
		t.Fatal("never received the name prompt")
	}
	mustWrite(t, conn, name+"\r\n")
	for _, st := range []struct {
		awaitPrompt string
		send        string
	}{
		{"Did I get that right", "Y\r\n"},
		{"Give me a password", password + "\r\n"},
		{"Please retype password", password + "\r\n"},
		{"ANSI color", "N\r\n"},
		{"sex", "M\r\n"},
		{"Race:", "H\r\n"},
		{"Class:", "W\r\n"},
		{"home town", "K\r\n"},
		{"keep these stats", "Y\r\n"},
		{"PRESS RETURN", "\r\n"},
	} {
		if got := readUntil(t, conn, reader, st.awaitPrompt, 10*time.Second); got == "" {
			t.Fatalf("character creation stalled at prompt %q", st.awaitPrompt)
		}
		mustWrite(t, conn, st.send)
	}
	if got := readUntil(t, conn, reader, "Make your choice", 10*time.Second); got == "" {
		t.Fatal("never received the main menu after character creation")
	}
	mustWrite(t, conn, "1\r\n")
	entered := readUntil(t, conn, reader, "The Board Room Of The Immortals", 10*time.Second)
	if entered == "" {
		t.Fatal("crowned character never entered the world")
	}
	return entered
}
