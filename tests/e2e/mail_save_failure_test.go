package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
)

const mailSaveFailureBody = "proof-mail-body"

// TestMailProductionRecipientSave proves the production completion path does not
// create a pending message or persist a synthetic mail object. On the reference
// host, C store_mail returns at its struct-size guard before indexing or writing
// (mail.c:285-289), while modify.c still reports "Message sent!".
func TestMailProductionRecipientSave(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}
	dbURL := filepath.Join(t.TempDir(), "darkpawns.db")

	fixtureRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(fixtureRoot, "lib", "data"), 0o700); err != nil {
		t.Fatalf("create isolated mail storage: %v", err)
	}
	root := repoRoot(t)
	if err := os.Symlink(filepath.Join(root, "lib", "world"), filepath.Join(fixtureRoot, "lib", "world")); err != nil {
		t.Fatalf("link world fixture: %v", err)
	}

	preserveDir := os.Getenv("DP_MAIL_SAVE_FAILURE_PRESERVE_DIR")
	if preserveDir != "" {
		if err := os.MkdirAll(preserveDir, 0o700); err != nil {
			t.Fatalf("create proof evidence directory: %v", err)
		}
		t.Setenv("DP_MAIL_LIFECYCLE_E2E_PRESERVE_DIR", preserveDir)
	}

	database, err := db.New(dbURL)
	if err != nil {
		t.Fatalf("open disposable proof database: %v", err)
	}
	var seededIDs []int
	t.Cleanup(func() {
		for _, id := range seededIDs {
			if err := database.DeletePlayer(id); err != nil {
				t.Logf("cleanup delete player %d: %v", id, err)
			}
		}
		if err := database.Close(); err != nil {
			t.Logf("cleanup close proof database: %v", err)
		}
	})

	suffix := time.Now().UnixNano() % 1000000000
	senderName := entryFixtureName("SaveSender", suffix)
	recipientName := entryFixtureName("SaveRcpt", suffix)
	const password = "mailproof"
	sender := seedMailPlayer(t, database, senderName, password, 34)
	seededIDs = append(seededIDs, sender.ID)
	recipient := seedMailPlayer(t, database, recipientName, password, 1)
	seededIDs = append(seededIDs, recipient.ID)

	before, err := database.GetPlayer(recipientName)
	if err != nil || before == nil {
		t.Fatalf("load recipient before proof save: record=%+v err=%v", before, err)
	}
	if err := database.SavePlayer(before); err != nil {
		t.Fatalf("recipient save before send: %v", err)
	}
	if got := strings.TrimSpace(string(before.Inventory)); got != "[]" {
		t.Fatalf("recipient inventory before send = %q, want []", got)
	}

	serverOne, senderConn, senderReader := launchMailServer(t, root, dbURL, fixtureRoot, "save-send")
	senderEntered := loginMailPlayer(t, senderConn, senderReader, senderName, password)
	if !strings.Contains(strings.ToLower(senderEntered), "postman") {
		t.Fatalf("sender did not enter the postmaster room: %q", senderEntered)
	}
	mustWrite(t, senderConn, "set "+senderName+" gold 51\r\n")
	if got := readUntil(t, senderConn, senderReader, "gold set to 51.", 5*time.Second); got == "" {
		t.Fatal("sender could not be funded through the production session path")
	}
	mustWrite(t, senderConn, "mail "+recipientName+" body\r\n")
	if got := readUntil(t, senderConn, senderReader, "Write your message", 5*time.Second); got == "" {
		t.Fatal("sender did not reach production mail composition")
	}
	mustWrite(t, senderConn, mailSaveFailureBody+"\r\n")
	mustWrite(t, senderConn, "@\r\n")
	if got := readUntil(t, senderConn, senderReader, "Message sent!", 5*time.Second); got == "" {
		t.Fatal("production message did not complete with C's completion text")
	}

	mailPath := filepath.Join(fixtureRoot, "lib", "data", "mail")
	mailBytes := readMailFile(t, mailPath)
	if len(mailBytes) != 0 {
		t.Fatalf("mail store after guarded production send has %d bytes, want 0", len(mailBytes))
	}
	preserveMailSaveFailureArtifact(t, preserveDir, "mail-after-send.bin", mailBytes)
	_ = senderConn.Close()
	serverOne.stop(t)
	assertMailServerSaveLogClean(t, serverOne)

	serverTwo, recipientConn, recipientReader := launchMailServer(t, root, dbURL, fixtureRoot, "save-receive")
	recipientEntered := loginMailPlayer(t, recipientConn, recipientReader, recipientName, password)
	if !strings.Contains(strings.ToLower(recipientEntered), "postman") {
		t.Fatalf("recipient did not enter the postmaster room: %q", recipientEntered)
	}
	mustWrite(t, recipientConn, "check\r\n")
	if got := readUntil(t, recipientConn, recipientReader, "Sorry, you don't have any mail waiting.", 5*time.Second); got == "" {
		t.Fatal("recipient did not see the empty mailbox after restart")
	}
	mustWrite(t, recipientConn, "receive\r\n")
	if got := readUntil(t, recipientConn, recipientReader, "Sorry, you don't have any mail waiting.", 5*time.Second); got == "" {
		t.Fatal("recipient receive did not report the empty mailbox")
	}
	if got := readMailFile(t, mailPath); len(got) != 0 {
		t.Fatalf("mail store after empty check/receive has %d bytes, want 0", len(got))
	}
	_ = recipientConn.Close()
	serverTwo.stop(t)
	assertMailServerSaveLogClean(t, serverTwo)

	persisted, err := database.GetPlayer(recipientName)
	if err != nil || persisted == nil {
		t.Fatalf("reload recipient after server save: record=%+v err=%v", persisted, err)
	}
	if got := strings.TrimSpace(string(persisted.Inventory)); got != "[]" {
		t.Fatalf("recipient inventory after send/restart/shutdown = %q, want []", got)
	}
	t.Logf("production_recipient_save completed=true inventory=%s mail_bytes=0", strings.TrimSpace(string(persisted.Inventory)))

	summary, err := json.Marshal(map[string]interface{}{
		"sender":              sender.Name,
		"recipient":           recipient.Name,
		"body":                mailSaveFailureBody,
		"persisted_inventory": json.RawMessage(persisted.Inventory),
		"mail_bytes":          len(mailBytes),
		"reference_guard":     "mail.c:285-289 returns before index/write on this host",
	})
	if err != nil {
		t.Fatalf("marshal proof summary: %v", err)
	}
	preserveMailSaveFailureArtifact(t, preserveDir, "proof-summary.json", append(summary, '\n'))
}

func preserveMailSaveFailureArtifact(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if dir == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatalf("preserve proof artifact %s: %v", name, err)
	}
}
