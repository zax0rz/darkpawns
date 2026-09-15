package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
)

const mailSaveFailureBody = "proof-mail-body"

// TestMailProductionRecipientSave proves the repaired fixed-block
// conversion through the real server path and a disposable PostgreSQL
// database. The recipient is saved by the server's shutdown cleanup after
// receiving the live note; the persisted row is inspected directly so the
// proof never reconstructs or sanitizes a replacement object.
func TestMailProductionRecipientSave(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e: builds and launches the server binary; skipped in -short")
	}
	dbURL := os.Getenv("DP_TEST_DB_URL")
	if dbURL == "" {
		t.Skip("DP_TEST_DB_URL is required for the disposable PostgreSQL proof; a skip is not proof")
	}

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
	senderName := fmt.Sprintf("SaveSender%d", suffix)
	recipientName := fmt.Sprintf("SaveRcpt%d", suffix)
	const password = "mailproof"
	sender := seedMailPlayer(t, database, senderName, password, 34)
	seededIDs = append(seededIDs, sender.ID)
	recipient := seedMailPlayer(t, database, recipientName, password, 1)
	seededIDs = append(seededIDs, recipient.ID)
	serverSender, err := database.GetPlayer(senderName)
	if err != nil || serverSender == nil {
		t.Fatalf("verify seeded sender before server: record=%+v err=%v", serverSender, err)
	}
	serverRecipient, err := database.GetPlayer(recipientName)
	if err != nil || serverRecipient == nil {
		t.Fatalf("verify seeded recipient before server: record=%+v err=%v", serverRecipient, err)
	}
	t.Logf("server_fixture sender=%s id=%d recipient=%s id=%d", serverSender.Name, serverSender.ID, serverRecipient.Name, serverRecipient.ID)

	before, err := database.GetPlayer(recipientName)
	if err != nil || before == nil {
		t.Fatalf("load recipient before proof save: record=%+v err=%v", before, err)
	}
	if err := database.SavePlayer(before); err != nil {
		t.Fatalf("recipient save before receipt: %v", err)
	}
	beforeReload, err := database.GetPlayer(recipientName)
	if err != nil || beforeReload == nil {
		t.Fatalf("reload recipient before proof: record=%+v err=%v", beforeReload, err)
	}
	if got := strings.TrimSpace(string(beforeReload.Inventory)); got != "[]" {
		t.Fatalf("recipient inventory before receipt = %q, want []", got)
	}
	t.Logf("recipient_before_receipt save_succeeded=true reload_inventory=%s", strings.TrimSpace(string(beforeReload.Inventory)))

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
	if got := readUntil(t, senderConn, senderReader, "Mail sent.", 5*time.Second); got == "" {
		t.Fatal("production message was not completed")
	}

	mailPath := filepath.Join(fixtureRoot, "lib", "data", "mail")
	sentHeader := readGoMailHeader(t, mailPath)
	if sentHeader.blockType != 1 || sentHeader.from != sender.ID || sentHeader.to != recipient.ID || sentHeader.text != mailSaveFailureBody {
		t.Fatalf("sent Go header = %+v, want single-block body and sender/recipient IDs %d/%d", sentHeader, sender.ID, recipient.ID)
	}
	preserveMailSaveFailureArtifact(t, preserveDir, "mail-before-receive.bin", readMailFile(t, mailPath))
	t.Logf("production_send completed=true sender_id=%d recipient_id=%d body=%q mail_bytes=%d", sentHeader.from, sentHeader.to, sentHeader.text, len(readMailFile(t, mailPath)))

	mustWrite(t, senderConn, "quit\r\n")
	_ = readFor(t, senderConn, senderReader, time.Second)
	_ = senderConn.Close()
	serverOne.stop(t)

	serverTwo, recipientConn, recipientReader := launchMailServer(t, root, dbURL, fixtureRoot, "save-receive")
	recipientEntered := loginMailPlayer(t, recipientConn, recipientReader, recipientName, password)
	if !strings.Contains(strings.ToLower(recipientEntered), "postman") {
		t.Fatalf("recipient did not enter the postmaster room: %q", recipientEntered)
	}
	mustWrite(t, recipientConn, "check\r\n")
	if got := readUntil(t, recipientConn, recipientReader, "You have mail waiting.", 5*time.Second); got == "" {
		t.Fatal("recipient did not see waiting mail after restart")
	}
	mustWrite(t, recipientConn, "receive\r\n")
	if got := readUntil(t, recipientConn, recipientReader, "gives you a piece of mail.", 5*time.Second); got == "" {
		t.Fatal("recipient did not receive the message after restart")
	}

	receivedHeader := readGoMailHeader(t, mailPath)
	if receivedHeader.blockType != 2 || receivedHeader.from != sender.ID || receivedHeader.to != recipient.ID || receivedHeader.text != mailSaveFailureBody {
		t.Fatalf("received Go header = %+v, want deleted marker and body %q", receivedHeader, mailSaveFailureBody)
	}
	preserveMailSaveFailureArtifact(t, preserveDir, "mail-after-receive.bin", readMailFile(t, mailPath))

	mustWrite(t, recipientConn, "check\r\n")
	if got := readUntil(t, recipientConn, recipientReader, "Sorry, you don't have any mail waiting.", 5*time.Second); got == "" {
		t.Fatal("second check still reported mail")
	}
	mustWrite(t, recipientConn, "read letter\r\n")
	delivered := readUntil(t, recipientConn, recipientReader, mailSaveFailureBody, 5*time.Second)
	if delivered == "" {
		t.Fatal("recipient could not read the delivered note")
	}
	fromMarker := "From: " + sender.Name + "\r\n\r\n"
	toMarker := "  To: " + recipient.Name + "\r\n"
	if !strings.Contains(delivered, toMarker) {
		t.Fatalf("delivered note missing recipient %q: %q", recipient.Name, delivered)
	}
	fromIndex := strings.Index(delivered, fromMarker)
	if fromIndex == -1 {
		t.Fatalf("delivered note missing sender %q: %q", sender.Name, delivered)
	}
	actualBody := delivered[fromIndex+len(fromMarker):]
	if actualBody != mailSaveFailureBody {
		t.Fatalf("delivered note body = %q, want exact %q", actualBody, mailSaveFailureBody)
	}
	if strings.IndexByte(delivered, 0) >= 0 {
		t.Fatal("player-facing delivered note contains fixed-block NUL padding")
	}
	mustWrite(t, recipientConn, "inventory\r\n")
	inventory := readFor(t, recipientConn, recipientReader, 3*time.Second)
	if got := strings.Count(inventory, "a piece of mail"); got != 1 {
		t.Fatalf("recipient inventory mail count = %d, want exactly one: %q", got, inventory)
	}
	t.Logf("production_delivered_note verified=true sender=%q body=%q inventory_items=1 runtime_nul_count=0", sender.Name, actualBody)

	// SIGTERM exercises the actual server session cleanup path, including
	// PlayerToRecord and DB.SavePlayer for the live received object.
	serverTwo.stop(t)
	_ = recipientConn.Close()
	serverLog := serverTwo.logBuffer.String()
	if strings.Contains(serverLog, "DB save error") || strings.Contains(serverLog, "linkdead save error") {
		t.Fatalf("production recipient shutdown save logged an error: %q", serverLog)
	}
	persisted, err := database.GetPlayer(recipientName)
	if err != nil || persisted == nil {
		t.Fatalf("reload recipient after server save: record=%+v err=%v", persisted, err)
	}
	persistedMailText := assertPersistedMailObject(t, persisted.Inventory, sender.Name, recipient.Name, mailSaveFailureBody)
	t.Logf("production_recipient_save path=server_shutdown_cleanup succeeded=true inventory_items=1 runtime_nul_count=0 mail_text_bytes=%d", len(persistedMailText))

	summary, err := json.Marshal(map[string]interface{}{
		"sender":                  sender.Name,
		"recipient":               recipient.Name,
		"body":                    mailSaveFailureBody,
		"persisted_inventory":     json.RawMessage(persisted.Inventory),
		"persisted_mail_text_len": len(persistedMailText),
		"reload_proof":            "blocked: RecordToPlayer skips synthetic VNum -1 mail objects",
	})
	if err != nil {
		t.Fatalf("marshal proof summary: %v", err)
	}
	preserveMailSaveFailureArtifact(t, preserveDir, "proof-summary.json", append(summary, '\n'))
}

func assertPersistedMailObject(t *testing.T, inventory []byte, senderName, recipientName, body string) string {
	t.Helper()
	if !json.Valid(inventory) {
		t.Fatalf("persisted inventory is not valid JSON: %q", inventory)
	}
	var items []game.SaveItemData
	if err := json.Unmarshal(inventory, &items); err != nil {
		t.Fatalf("decode persisted inventory: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("persisted inventory items = %d, want exactly one: %q", len(items), inventory)
	}
	if items[0].VNum != -1 || items[0].State == nil {
		t.Fatalf("persisted item = %+v, want synthetic mail object with state", items[0])
	}
	mailText, ok := items[0].State["mail_text"].(string)
	if !ok {
		t.Fatalf("persisted mail state missing string mail_text: %+v", items[0].State)
	}
	if strings.IndexByte(mailText, 0) >= 0 {
		t.Fatalf("persisted Runtime.MailText contains NUL padding: %q", mailText)
	}
	if !strings.Contains(mailText, "  To: "+recipientName+"\r\n") || !strings.Contains(mailText, "From: "+senderName+"\r\n\r\n") || !strings.HasSuffix(mailText, body) {
		t.Fatalf("persisted mail text lost exact sender/body: %q", mailText)
	}
	return mailText
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
