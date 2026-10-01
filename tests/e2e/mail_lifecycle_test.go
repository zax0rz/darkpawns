package e2e

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
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
	"github.com/zax0rz/darkpawns/pkg/game"
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
	if !strings.Contains(blocked, "$n tells you, 'A stamp costs 50 coins.'") ||
		!strings.Contains(blocked, "$n tells you, '...which I see you can't afford.'") {
		t.Fatalf("production mail path did not expose the expected affordability boundary: %q", blocked)
	}
	t.Logf("production_mail_blocked_stage=stamp_affordability output=%q", strings.TrimSpace(blocked))

	// Keep the connection lifecycle ordinary; this is not a restart vehicle.
	mustWrite(t, conn, "quit\r\n")
}

// TestMailProductionLifecycleAcrossRestart proves the production vehicle with
// the supported persistence configuration: SQLite-backed player identity, an
// existing offline recipient, actual telnet composition, a real process
// shutdown/restart, and one-time receipt through the postmaster.
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
	if got := readUntil(t, senderConn, senderReader, "Mail sent.", 5*time.Second); got == "" {
		t.Fatal("production message was not completed")
	}

	mailPath := filepath.Join(fixtureRoot, "lib", "data", "mail")
	sentHeader := readGoMailHeader(t, mailPath)
	if sentHeader.blockType != 1 || sentHeader.from != sender.ID || sentHeader.to != recipient.ID {
		t.Fatalf("sent Go header = %+v, want header marker/from/to = 1/%d/%d", sentHeader, sender.ID, recipient.ID)
	}
	if sentHeader.text != productionMailBody {
		t.Fatalf("sent Go header body = %q, want exact single-line body", sentHeader.text)
	}
	preserveMailLifecycleArtifact(t, "mail-before-restart.bin", readMailFile(t, mailPath))
	t.Logf("production_send completed=true sender_id=%d recipient_id=%d body=%q mail_bytes=%d", sentHeader.from, sentHeader.to, sentHeader.text, len(readMailFile(t, mailPath)))

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
	check := readUntil(t, recipientConn, recipientReader, "You have mail waiting.", 5*time.Second)
	if check == "" {
		t.Fatal("recipient did not see waiting mail after restart")
	}
	mustWrite(t, recipientConn, "receive\r\n")
	receive := readUntil(t, recipientConn, recipientReader, "gives you a piece of mail.", 5*time.Second)
	if receive == "" {
		t.Fatal("recipient did not receive the message after restart")
	}

	receivedHeader := readGoMailHeader(t, mailPath)
	if receivedHeader.blockType != 2 || receivedHeader.from != sender.ID || receivedHeader.to != recipient.ID {
		t.Fatalf("received Go header = %+v, want deleted marker/from/to = 2/%d/%d", receivedHeader, sender.ID, recipient.ID)
	}
	if receivedHeader.text != productionMailBody {
		t.Fatalf("received Go header body = %q, want exact single-line body", receivedHeader.text)
	}
	preserveMailLifecycleArtifact(t, "mail-after-receive.bin", readMailFile(t, mailPath))

	mustWrite(t, recipientConn, "check\r\n")
	secondCheck := readUntil(t, recipientConn, recipientReader, "Sorry, you don't have any mail waiting.", 5*time.Second)
	if secondCheck == "" {
		t.Fatal("second production check still reported mail")
	}
	mustWrite(t, recipientConn, "receive\r\n")
	secondReceive := readUntil(t, recipientConn, recipientReader, "Sorry, you don't have any mail waiting.", 5*time.Second)
	if secondReceive == "" {
		t.Fatal("second production receive did not report an empty mailbox")
	}
	t.Logf("production_restart completed=true check=%q receive=%q sender=%q body=%q second_check=%q second_receive=%q", check, receive, returnedSender.Name, receivedHeader.text, secondCheck, secondReceive)

	// Read the delivered object through the real player-facing observation path.
	// This verifies the note in the recipient's live inventory rather than
	// treating the mail file/header as a proxy for what the recipient got.
	mustWrite(t, recipientConn, "read letter\r\n")
	deliveredText := readUntil(t, recipientConn, recipientReader, productionMailBody, 5*time.Second)
	if deliveredText == "" {
		t.Fatal("recipient could not read the delivered note")
	}
	fromMarker := "From: " + sender.Name + "\r\n\r\n"
	toMarker := "  To: " + recipient.Name + "\r\n"
	if !strings.Contains(deliveredText, toMarker) {
		t.Fatalf("delivered note missing recipient %q: %q", recipient.Name, deliveredText)
	}
	fromIndex := strings.Index(deliveredText, fromMarker)
	if fromIndex == -1 {
		t.Fatalf("delivered note missing sender %q: %q", sender.Name, deliveredText)
	}
	actualBody := strings.TrimRight(deliveredText[fromIndex+len(fromMarker):], "\x00")
	if actualBody != productionMailBody {
		t.Fatalf("delivered note body = %q, want %q", actualBody, productionMailBody)
	}
	mustWrite(t, recipientConn, "inventory\r\n")
	inventory := readFor(t, recipientConn, recipientReader, 3*time.Second)
	if got := strings.Count(inventory, "a piece of mail"); got != 1 {
		t.Fatalf("recipient inventory mail count = %d, want exactly one: %q", got, inventory)
	}
	t.Logf("production_delivered_note verified=true sender=%q body=%q inventory_items=1", sender.Name, actualBody)

	// The recipient quits from the board room, which is not one of C's safe quit
	// rooms (act.other.c:87-112), as a mortal: C refuses the quit and keeps them in
	// the world (act.other.c:159-166). No rent runs, which is why the delivered mail
	// object is still theirs to carry into the next process. The socket then drops,
	// and DP-1323 leaves the playing character linkdead for the reaper.
	mustWrite(t, recipientConn, "quit\r\n")
	refused := readFor(t, recipientConn, recipientReader, time.Second)
	if !strings.Contains(refused, "Type REALLYQUIT to quit the game and lose your eq.") {
		t.Fatalf("recipient quit from the board room was not refused: %q", refused)
	}
	_ = recipientConn.Close()
	serverTwo.stop(t)
	assertMailServerSaveLogClean(t, serverTwo)

	// The drop's lost-link save is what persists the queued receipt, so wait for the
	// row to settle rather than for a clock to pass.
	persistedRecipient := waitForReceiptSave(t, database, recipientName, productionMailBody)
	persistedMailText := assertPersistedMailObject(t, persistedRecipient.Inventory, sender.Name, recipient.Name, productionMailBody)
	preserveMailLifecycleArtifact(t, "recipient-inventory.json", persistedRecipient.Inventory)
	t.Logf("production_recipient_save completed=true inventory_items=1 mail_text_bytes=%d", len(persistedMailText))

	logRedactedPlayerState(t, database, "before-reload-login", recipientName)
	serverThree, reloadedConn, reloadedReader := launchMailServer(t, root, dbURL, fixtureRoot, "reload")
	reloadedEntered := loginMailPlayer(t, reloadedConn, reloadedReader, recipientName, password)
	// The recipient is a level-1 mortal and the row now carries load_room NOWHERE:
	// every entry save writes it (interpreter.c:2186), and the refused quit that
	// followed left it there through the lost-link save (comm.c:2130). C sends that
	// character to the mortal start room, vnum 8004, not to the room the fixture
	// seed originally placed them in.
	if !strings.Contains(reloadedEntered, "At the Temple Altar") {
		t.Fatalf("reloaded recipient did not enter the mortal start room: %q", reloadedEntered)
	}
	logRedactedPlayerState(t, database, "after-reload-login", recipientName)
	mustWrite(t, reloadedConn, "inventory\r\n")
	reloadedInventory := readFor(t, reloadedConn, reloadedReader, 3*time.Second)
	if got := strings.Count(reloadedInventory, "a piece of mail"); got != 1 {
		t.Fatalf("reloaded recipient inventory mail count = %d, want exactly one: %q", got, reloadedInventory)
	}

	mustWrite(t, reloadedConn, "read letter\r\n")
	reloadedRead := readUntil(t, reloadedConn, reloadedReader, productionMailBody, 5*time.Second)
	if reloadedRead == "" {
		t.Fatal("reloaded recipient could not read the persisted mail object")
	}
	if !strings.Contains(reloadedRead, toMarker) || !strings.Contains(reloadedRead, fromMarker) {
		t.Fatalf("reloaded note missing exact sender/recipient markers: %q", reloadedRead)
	}
	reloadedFromIndex := strings.Index(reloadedRead, fromMarker)
	reloadedBody := strings.TrimRight(reloadedRead[reloadedFromIndex+len(fromMarker):], "\x00")
	if reloadedBody != productionMailBody {
		t.Fatalf("reloaded note body = %q, want %q", reloadedBody, productionMailBody)
	}

	// The postmaster's MAIL/CHECK/RECEIVE are a room-bound special procedure
	// (postmaster.go), and the room C returns this character to holds no postmaster,
	// so the reload phase reads the mailbox from the store itself: the delivered
	// block is still marked deleted and the reload server reports nothing pending.
	// The pre-restart phase already took the postmaster's own word for the empty
	// mailbox, in the room that has one.
	reloadedHeader := readGoMailHeader(t, mailPath)
	if reloadedHeader.blockType != 2 || reloadedHeader.from != sender.ID || reloadedHeader.to != recipient.ID {
		t.Fatalf("reloaded mail header = %+v, want the deleted marker still 2 with from/to %d/%d",
			reloadedHeader, sender.ID, recipient.ID)
	}
	if reloadedHeader.text != productionMailBody {
		t.Fatalf("reloaded mail header body = %q, want %q", reloadedHeader.text, productionMailBody)
	}
	t.Logf("production_relogin completed=true inventory_items=1 sender=%q body=%q mail_block_type=%d",
		sender.Name, reloadedBody, reloadedHeader.blockType)

	mustWrite(t, reloadedConn, "quit\r\n")
	_ = readFor(t, reloadedConn, reloadedReader, time.Second)
	_ = reloadedConn.Close()
	serverThree.stop(t)
	assertMailServerSaveLogClean(t, serverThree)
	// The reload server read the mail store on the way up and found nothing
	// pending: the delivered message is not waiting for the recipient a second
	// time after a process restart.
	if reloadLog := serverThree.logBuffer.String(); !strings.Contains(reloadLog, "Mail file read -- 0 messages.") {
		t.Fatalf("reload server did not report an empty pending mailbox; log:\n%s", reloadLog)
	}
}

// waitForReceiptSave waits for the dropped transport's save to persist what the
// receipt added: the row carries the delivered mail object and C's lost-link load
// room, NOWHERE (comm.c:2130). It reads the row rather than trusting a clock, so
// the reload phase asserts on state the store has actually settled.
func waitForReceiptSave(t *testing.T, database *db.DB, name, mailBody string) *db.PlayerRecord {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		record, err := database.GetPlayer(name)
		if err != nil {
			t.Fatalf("read %q while waiting for the receipt save: %v", name, err)
		}
		if record != nil && record.RoomVNum == game.LoadRoomNowhere && strings.Contains(string(record.Inventory), mailBody) {
			return record
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("the receipt save for %q never landed: no delivered %q in the row with load room NOWHERE", name, mailBody)
	return nil
}

// logRedactedPlayerState prints the persisted facts a restart has to restore, with
// the password hash and the bulk blob columns reduced to sizes. A failing restart
// phase reports the row it was handed, not just the transcript that stalled.
func logRedactedPlayerState(t *testing.T, database *db.DB, stage, name string) {
	t.Helper()
	record, err := database.GetPlayer(name)
	if err != nil || record == nil {
		t.Fatalf("%s: player %q: record=%+v err=%v", stage, name, record, err)
	}
	var items []game.SaveItemData
	if err := json.Unmarshal(record.Inventory, &items); err != nil {
		t.Fatalf("%s: decode persisted inventory for %q: %v", stage, name, err)
	}
	mailText := ""
	for _, item := range items {
		if text, ok := item.State["mail_text"].(string); ok {
			mailText = text
		}
	}
	t.Logf("persisted_player_state stage=%s name=%q id=%d level=%d exp=%d stored_load_room=%d health=%d/%d mana=%d/%d inventory_items=%d mail_text_bytes=%d character_data_bytes=%d equipment_bytes=%d",
		stage, record.Name, record.ID, record.Level, record.Exp, record.RoomVNum,
		record.Health, record.MaxHealth, record.Mana, record.MaxMana,
		len(items), len(mailText), len(record.CharacterData), len(record.Equipment))
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
	httpPort := freePort(t)
	telnetPort := freePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	process := &mailServerProcess{cancel: cancel, label: label}
	process.cmd = exec.CommandContext(ctx, serverBinary(t, root),
		"-world", filepath.Join(fixtureRoot, "lib", "world"),
		"-port", fmt.Sprintf("%d", httpPort),
		"-telnet-port", fmt.Sprintf("%d", telnetPort),
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
	if err := process.cmd.Start(); err != nil {
		cancel()
		t.Fatalf("start mail server %s: %v", label, err)
	}
	t.Cleanup(func() { process.stop(t) })
	conn := dialWhenReady(t, fmt.Sprintf("127.0.0.1:%d", telnetPort), 20*time.Second)
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

type goMailHeaderSnapshot struct {
	blockType int
	from      int
	to        int
	text      string
}

func readGoMailHeader(t *testing.T, path string) goMailHeaderSnapshot {
	t.Helper()
	bytes := readMailFile(t, path)
	if len(bytes) != 512 {
		t.Fatalf("Go mail fixture bytes = %d, want 512", len(bytes))
	}
	return goMailHeaderSnapshot{
		blockType: int(binary.LittleEndian.Uint32(bytes[0:4])),
		from:      int(binary.LittleEndian.Uint32(bytes[16:20])),
		to:        int(binary.LittleEndian.Uint32(bytes[20:24])),
		text:      strings.TrimRight(string(bytes[24:]), "\x00"),
	}
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

func preserveMailLifecycleArtifact(t *testing.T, name string, bytes []byte) {
	t.Helper()
	preserveDir := os.Getenv("DP_MAIL_LIFECYCLE_E2E_PRESERVE_DIR")
	if preserveDir == "" {
		return
	}
	if err := os.MkdirAll(preserveDir, 0o700); err != nil {
		t.Fatalf("preserve mail artifact directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(preserveDir, name), bytes, 0o600); err != nil {
		t.Fatalf("preserve mail artifact %s: %v", name, err)
	}
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
