package e2e

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"golang.org/x/crypto/bcrypt"
)

const productionMailBody = "proof-mail-body"

// TestMailProductionBootBoundary reaches the real no-DB server, session,
// command, special-procedure, and postmaster dispatch paths. No-DB mode has no
// persistent identity authority, so it remains a separate affordability
// boundary rather than a restart proof.
func TestMailProductionBootBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}
	t.Setenv("DP_FRESH_MUD", "1")

	conn, reader := launchAndDial(t)
	defer conn.Close()

	name := entryFixtureName("MailBoot", time.Now().UnixNano()%100000)
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
	if !strings.Contains(blocked, "The postman tells you, 'A stamp costs 25 coins.'") ||
		!strings.Contains(blocked, "the postman tells you, '...which I see you can't afford.'") {
		t.Fatalf("production mail path did not expose the expected affordability boundary: %q", blocked)
	}
	t.Logf("production_mail_blocked_stage=stamp_affordability output=%q", strings.TrimSpace(blocked))

	// Keep the connection lifecycle ordinary; this is not a restart vehicle.
	mustWrite(t, conn, "quit\r\n")
}

// TestMailProductionLifecycleAcrossRestart proves that a production send still
// completes but leaves no pending mail across a real server restart. On the
// reference host, C store_mail returns at its struct-size guard before writing
// or indexing the message (mail.c:285-289); modify.c still sends "Message sent!".
func TestMailProductionLifecycleAcrossRestart(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}
	// A disposable SQLite file, so the mail lifecycle runs against exactly the
	// database the shipped server runs: same path, same file, three restarts.
	dbURL := filepath.Join(t.TempDir(), "darkpawns.db")

	fixtureRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(fixtureRoot, "lib", "data"), 0o700); err != nil {
		t.Fatalf("create isolated mail storage: %v", err)
	}
	root := repoRoot(t)
	if err := os.Symlink(filepath.Join(root, "lib", "world"), filepath.Join(fixtureRoot, "lib", "world")); err != nil {
		t.Fatalf("link world fixture: %v", err)
	}

	database, err := db.New(dbURL)
	if err != nil {
		t.Fatalf("open test database through production initializer: %v", err)
	}
	var seededIDs []int
	t.Cleanup(func() {
		for _, id := range seededIDs {
			if err := database.DeletePlayer(id); err != nil {
				t.Logf("cleanup delete player %d: %v", id, err)
			}
		}
		if err := database.Close(); err != nil {
			t.Logf("cleanup close test database: %v", err)
		}
	})

	suffix := time.Now().UnixNano() % 1000000000
	senderName := entryFixtureName("MailSender", suffix)
	recipientName := entryFixtureName("MailRcpt", suffix)
	const password = "mailproof"
	sender := seedMailPlayer(t, database, senderName, password, 34)
	seededIDs = append(seededIDs, sender.ID)
	recipient := seedMailPlayer(t, database, recipientName, password, 1)
	seededIDs = append(seededIDs, recipient.ID)
	t.Logf("persistent_fixture sender=%s id=%d level=%d recipient=%s id=%d offline=true", sender.Name, sender.ID, 34, recipient.Name, recipient.ID)

	serverOne, senderConn, senderReader := launchMailServer(t, root, dbURL, fixtureRoot, "send")
	senderEntered := loginMailPlayer(t, senderConn, senderReader, senderName, password)
	if !strings.Contains(strings.ToLower(senderEntered), "postman") {
		t.Fatalf("sender did not enter the postmaster room: %q", senderEntered)
	}

	mustWrite(t, senderConn, "set "+senderName+" gold 51\r\n")
	if got := readUntil(t, senderConn, senderReader, "gold set to 51.", 5*time.Second); got == "" {
		t.Fatal("production sender could not be funded through the session set command")
	}
	// The production telnet transport supplies tokenized args to the existing
	// postmaster hook. Keep the same recipient-plus-remainder call shape as the
	// established production boundary vehicle; the hook consumes the recipient
	// token and then enters its line editor.
	mustWrite(t, senderConn, "mail "+recipientName+" body\r\n")
	mailPrompt := readFor(t, senderConn, senderReader, 3*time.Second)
	t.Logf("production-mail-prompt=%q", mailPrompt)
	if !strings.Contains(mailPrompt, "Write your message") {
		t.Fatalf("production sender did not reach mail composition: %q", mailPrompt)
	}
	mustWrite(t, senderConn, productionMailBody+"\r\n")
	mustWrite(t, senderConn, "@\r\n")
	if got := readUntil(t, senderConn, senderReader, "Message sent!", 5*time.Second); got == "" {
		t.Fatal("production message was not completed")
	}

	mailPath := filepath.Join(fixtureRoot, "lib", "data", "mail")
	mailBytes := readMailFile(t, mailPath)
	if len(mailBytes) != 0 {
		t.Fatalf("mail store after reference-guarded send has %d bytes, want 0", len(mailBytes))
	}
	t.Logf("production_send completed=true sender_id=%d recipient_id=%d body=%q mail_bytes=0", sender.ID, recipient.ID, productionMailBody)

	mustWrite(t, senderConn, "quit\r\n")
	_ = readFor(t, senderConn, senderReader, time.Second)
	_ = senderConn.Close()
	serverOne.stop(t)

	returnedSender, err := database.GetPlayer(senderName)
	if err != nil || returnedSender == nil {
		t.Fatalf("sender identity after first shutdown: record=%+v err=%v", returnedSender, err)
	}
	if returnedSender.ID != sender.ID {
		t.Fatalf("sender ID changed across shutdown: before=%d after=%d", sender.ID, returnedSender.ID)
	}
	if returnedSender.Name != sender.Name {
		t.Fatalf("sender name changed across shutdown: before=%q after=%q", sender.Name, returnedSender.Name)
	}

	serverTwo, recipientConn, recipientReader := launchMailServer(t, root, dbURL, fixtureRoot, "receive")
	recipientEntered := loginMailPlayer(t, recipientConn, recipientReader, recipientName, password)
	if !strings.Contains(strings.ToLower(recipientEntered), "postman") {
		t.Fatalf("recipient did not enter the postmaster room after restart: %q", recipientEntered)
	}
	mustWrite(t, recipientConn, "check\r\n")
	check := readUntil(t, recipientConn, recipientReader, "Sorry, you don't have any mail waiting.", 5*time.Second)
	if check == "" {
		t.Fatal("recipient did not see the empty mailbox after restart")
	}
	mustWrite(t, recipientConn, "receive\r\n")
	if got := readUntil(t, recipientConn, recipientReader, "Sorry, you don't have any mail waiting.", 5*time.Second); got == "" {
		t.Fatal("empty production receive did not report no mail")
	}
	if remaining := readMailFile(t, mailPath); len(remaining) != 0 {
		t.Fatalf("mail store after empty check/receive has %d bytes, want 0", len(remaining))
	}
	_ = recipientConn.Close()
	serverTwo.stop(t)
	assertMailServerSaveLogClean(t, serverTwo)
	if reload := serverTwo.logBuffer.String(); !strings.Contains(reload, "Mail file read -- 0 messages.") {
		t.Fatalf("restart did not load an empty pending mailbox; log:\n%s", reload)
	}
	returnedRecipient, err := database.GetPlayer(recipientName)
	if err != nil || returnedRecipient == nil {
		t.Fatalf("recipient identity after restart: record=%+v err=%v", returnedRecipient, err)
	}
	if got := strings.TrimSpace(string(returnedRecipient.Inventory)); got != "[]" {
		t.Fatalf("recipient inventory after empty mail restart = %q, want []", got)
	}
}

func assertMailServerSaveLogClean(t *testing.T, process *mailServerProcess) {
	t.Helper()
	log := process.logBuffer.String()
	for _, marker := range []string{"DB save error", "linkdead save error"} {
		if strings.Contains(log, marker) {
			t.Fatalf("mail server %s log contains %q: %s", process.label, marker, log)
		}
	}
	t.Logf("mail_server_save_log_checked server=%s db_save_errors=0 linkdead_save_errors=0", process.label)
}

type mailServerProcess struct {
	cmd       *exec.Cmd
	cancel    context.CancelFunc
	logBuffer strings.Builder
	label     string
	stopped   bool
}

func launchMailServer(t *testing.T, root, dbURL, fixtureRoot, label string) (*mailServerProcess, net.Conn, *bufio.Reader) {
	t.Helper()
	ports, releasePorts := reserveServerPorts(t)
	ctx, cancel := context.WithCancel(context.Background())
	process := &mailServerProcess{cancel: cancel, label: label}
	process.cmd = exec.CommandContext(ctx, serverBinary(t, root),
		"-world", filepath.Join(fixtureRoot, "lib", "world"),
		"-port", fmt.Sprintf("%d", ports.http),
		"-telnet-port", fmt.Sprintf("%d", ports.telnet),
		"-whod-port", fmt.Sprintf("%d", ports.whod),
		"-db", dbURL,
	)
	process.cmd.Dir = root
	process.cmd.Env = append(os.Environ(),
		"DP_ALLOW_NO_DB=0",
		"JWT_SECRET=e2e-mail-lifecycle-secret-at-least-32-chars-long",
		"ENVIRONMENT=development",
		"DP_SEED=1",
		"DP_FIXED_TIME=650337471",
	)
	process.cmd.Stdout = &process.logBuffer
	process.cmd.Stderr = &process.logBuffer
	releasePorts()
	if err := process.cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start mail server %s: %v", label, err)
	}
	t.Cleanup(func() { process.stop(t) })
	conn := dialWhenReady(t, fmt.Sprintf("127.0.0.1:%d", ports.telnet), 20*time.Second)
	return process, conn, bufio.NewReader(conn)
}

func (p *mailServerProcess) stop(t *testing.T) {
	t.Helper()
	if p == nil || p.stopped {
		return
	}
	p.stopped = true
	if p.cmd.Process != nil {
		if err := p.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			t.Logf("mail server %s signal: %v", p.label, err)
		}
	}
	done := make(chan error, 1)
	go func() { done <- p.cmd.Wait() }()
	var waitErr error
	select {
	case waitErr = <-done:
	case <-time.After(15 * time.Second):
		p.cancel()
		_ = p.cmd.Process.Kill()
		waitErr = <-done
		t.Errorf("mail server %s did not shut down gracefully", p.label)
	}
	p.cancel()
	if waitErr != nil {
		t.Errorf("mail server %s shutdown: %v", p.label, waitErr)
	}
	if preserveDir := os.Getenv("DP_MAIL_LIFECYCLE_E2E_PRESERVE_DIR"); preserveDir != "" {
		if err := os.MkdirAll(preserveDir, 0o700); err != nil {
			t.Errorf("preserve mail server directory: %v", err)
		} else if err := os.WriteFile(filepath.Join(preserveDir, "server-"+p.label+".log"), []byte(p.logBuffer.String()), 0o600); err != nil {
			t.Errorf("preserve mail server log: %v", err)
		}
	}
	if t.Failed() {
		t.Logf("mail server %s log:\n%s", p.label, p.logBuffer.String())
	}
}

// loginMailPlayer walks the real returning-player login: name, password, MOTD,
// menu, and world entry. The entry wait is room-agnostic -- C's load-room rule
// decides the room from the saved row (interpreter.c:2191-2210), so a helper that
// waited for one hardcoded room name would report "did not enter the world" for a
// character C had placed correctly somewhere else. Every wait is recorded, so a
// failure reports the whole login transcript rather than only the stalled step.
func loginMailPlayer(t *testing.T, conn net.Conn, reader *bufio.Reader, name, password string) string {
	t.Helper()
	transcript := ""
	step := func(marker, stepName string) {
		t.Helper()
		chunk := readUntil(t, conn, reader, marker, 10*time.Second)
		transcript += chunk
		if chunk == "" {
			t.Fatalf("%s: never saw %q; login transcript=%q", stepName, marker, transcript)
		}
	}
	step("By what name", "returning player never received the name prompt")
	mustWrite(t, conn, name+"\r\n")
	step("Password", "persistent player was not prompted for a password")
	mustWrite(t, conn, password+"\r\n")
	step("PRESS RETURN", "persistent player did not receive the MOTD")
	mustWrite(t, conn, "\r\n")
	step("Make your choice", "persistent player did not receive the main menu")
	mustWrite(t, conn, "1\r\n")
	// Every room look ends with C's exits line (look.go), which is what "in the
	// world" looks like from the client side whatever room that is.
	entered := readUntil(t, conn, reader, "[ Exits: ", 10*time.Second)
	if entered == "" {
		t.Fatalf("persistent player %q did not enter the world: no room text after menu choice 1; transcript=%q", name, transcript)
	}
	return entered + readFor(t, conn, reader, time.Second)
}

func readMailFile(t *testing.T, path string) []byte {
	t.Helper()
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read isolated Go mail file: %v", err)
	}
	return bytes
}

func seedMailPlayer(t *testing.T, database *db.DB, name, password string, level int) *db.PlayerRecord {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash mail fixture password: %v", err)
	}
	record := &db.PlayerRecord{
		Name:      name,
		Password:  string(hash),
		RoomVNum:  1204,
		Level:     level,
		Exp:       1,
		Health:    100,
		MaxHealth: 100,
		Mana:      100,
		MaxMana:   100,
		Move:      100,
		MaxMove:   100,
		Strength:  10,
		Class:     3,
		Race:      0,
		StatStr:   10,
		StatInt:   10,
		StatWis:   10,
		StatDex:   10,
		StatCon:   10,
		StatCha:   10,
		Hometown:  0,
		Inventory: []byte("[]"),
		Equipment: []byte("{}"),
	}
	if err := database.CreatePlayer(record); err != nil {
		t.Fatalf("seed persistent mail player %s: %v", name, err)
	}
	return record
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
