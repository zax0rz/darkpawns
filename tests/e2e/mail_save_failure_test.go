package e2e

import (
	"crypto/sha256"
	"encoding/hex"
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

// TestMailProductionRecipientSaveFailure proves the post-receipt persistence
// failure through a disposable PostgreSQL database and the real mail path.
// It is deliberately a characterization test: it must fail on 22P05 and does
// not repair or normalize the production value.
func TestMailProductionRecipientSaveFailure(t *testing.T) {
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
	controlName := fmt.Sprintf("SaveCtrl%d", suffix)
	const password = "mailproof"
	sender := seedMailPlayer(t, database, senderName, password, 34)
	seededIDs = append(seededIDs, sender.ID)
	recipient := seedMailPlayer(t, database, recipientName, password, 1)
	seededIDs = append(seededIDs, recipient.ID)
	control := seedMailPlayer(t, database, controlName, password, 1)
	seededIDs = append(seededIDs, control.ID)
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
		t.Fatalf("reload recipient before receipt: record=%+v err=%v", beforeReload, err)
	}
	if got := strings.TrimSpace(string(beforeReload.Inventory)); got != "[]" {
		t.Fatalf("recipient inventory before receipt = %q, want []", got)
	}
	t.Logf("recipient_before_receipt save_succeeded=true reload_inventory=%s", strings.TrimSpace(string(beforeReload.Inventory)))

	serverOne, senderConn, senderReader := launchMailServer(t, root, dbURL, fixtureRoot, "save-failure-send")
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

	serverTwo, recipientConn, recipientReader := launchMailServer(t, root, dbURL, fixtureRoot, "save-failure-receive")
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

	mailText := reconstructDeliveredMailText(t, delivered, recipientName, mailSaveFailureBody)
	probePlayer := game.NewPlayer(recipient.ID, recipientName, 1203)
	probeObject := (&game.World{}).CreateMailObject(probePlayer, mailText)
	probePlayer.Inventory.RestoreItem(probeObject)
	failedRecord, err := db.PlayerToRecord(probePlayer, nil)
	if err != nil {
		t.Fatalf("serialize post-receipt player: %v", err)
	}
	if !json.Valid(failedRecord.Inventory) {
		t.Fatalf("post-receipt inventory is not valid JSON text: %q", failedRecord.Inventory)
	}
	if !strings.Contains(string(failedRecord.Inventory), `\u0000`) {
		t.Fatalf("post-receipt inventory lacks the expected JSON NUL escape: %q", failedRecord.Inventory)
	}
	sanitizedPayload := strings.ReplaceAll(string(failedRecord.Inventory), `\u0000`, "<NUL>")
	payloadHash := sha256.Sum256(failedRecord.Inventory)
	nulCount := strings.Count(mailText, "\x00")
	if nulCount != game.MailHeaderDataSize-len(mailSaveFailureBody) {
		t.Fatalf("reconstructed Runtime.MailText NUL count = %d, want %d", nulCount, game.MailHeaderDataSize-len(mailSaveFailureBody))
	}
	t.Logf("post_receipt_runtime mail_text_bytes=%d mail_text_nul_count=%d first_nul_offset=%d mail_text_sha256=%s", len(mailText), nulCount, strings.IndexByte(mailText, 0), sha256Hex(mailText))
	t.Logf("post_receipt_serialization inventory_json_sha256=%s inventory_json_sanitized=%s", hex.EncodeToString(payloadHash[:]), sanitizedPayload)

	failureErr := database.SavePlayer(failedRecord)
	if failureErr == nil {
		t.Fatal("post-receipt SavePlayer unexpectedly succeeded")
	}
	if !strings.Contains(failureErr.Error(), "unsupported Unicode escape sequence") || !strings.Contains(failureErr.Error(), "22P05") {
		t.Fatalf("post-receipt SavePlayer error = %v, want PostgreSQL 22P05 unsupported Unicode escape sequence", failureErr)
	}
	t.Logf("post_receipt_save operation=db.SavePlayer result=expected_failure error=%q", failureErr)

	afterDirectFailure, err := database.GetPlayer(recipientName)
	if err != nil || afterDirectFailure == nil {
		t.Fatalf("reload recipient after failed save: record=%+v err=%v", afterDirectFailure, err)
	}
	if got := strings.TrimSpace(string(afterDirectFailure.Inventory)); got != "[]" {
		t.Fatalf("recipient inventory after failed save = %q, want unchanged []", got)
	}
	if afterDirectFailure.RoomVNum != beforeReload.RoomVNum {
		t.Fatalf("recipient room after failed save = %d, want unchanged %d", afterDirectFailure.RoomVNum, beforeReload.RoomVNum)
	}
	t.Logf("post_receipt_reload after_failed_save=true inventory=%s room_vnum=%d persisted_mail_object=false", strings.TrimSpace(string(afterDirectFailure.Inventory)), afterDirectFailure.RoomVNum)

	controlPlayer := game.NewPlayer(control.ID, controlName, 1203)
	controlText := strings.TrimRight(mailText, "\x00")
	controlObject := (&game.World{}).CreateMailObject(controlPlayer, controlText)
	controlPlayer.Inventory.RestoreItem(controlObject)
	controlRecord, err := db.PlayerToRecord(controlPlayer, nil)
	if err != nil {
		t.Fatalf("serialize terminated control: %v", err)
	}
	if strings.Contains(string(controlRecord.Inventory), `\u0000`) {
		t.Fatalf("terminated control still contains JSON NUL escape: %q", controlRecord.Inventory)
	}
	if err := database.SavePlayer(controlRecord); err != nil {
		t.Fatalf("terminated control SavePlayer: %v", err)
	}
	controlReload, err := database.GetPlayer(controlName)
	if err != nil || controlReload == nil {
		t.Fatalf("reload terminated control: record=%+v err=%v", controlReload, err)
	}
	if !strings.Contains(string(controlReload.Inventory), mailSaveFailureBody) {
		t.Fatalf("terminated control reload lost mail body: %q", controlReload.Inventory)
	}
	t.Logf("control terminated_value save_succeeded=true reload_contains_body=true inventory=%s", strings.TrimSpace(string(controlReload.Inventory)))

	mustWrite(t, recipientConn, "quit\r\n")
	_ = readFor(t, recipientConn, recipientReader, time.Second)
	_ = recipientConn.Close()
	serverTwo.stop(t)
	serverLog := serverTwo.logBuffer.String()
	if !strings.Contains(serverLog, "22P05") || !strings.Contains(serverLog, "DB save error") {
		t.Fatalf("production shutdown log lacks expected recipient save failure: %q", serverLog)
	}
	t.Logf("production_shutdown_save_log matched=true 22P05=true db_save_error=true")
	preserveMailSaveFailureArtifact(t, preserveDir, "proof-summary.json", []byte(fmt.Sprintf("{\"failure_error\":%q,\"runtime_mail_text_nul_count\":%d,\"serialized_inventory_sanitized\":%q,\"recipient_inventory_after_failed_save\":%q,\"control_save_succeeded\":true}\n", failureErr.Error(), nulCount, sanitizedPayload, strings.TrimSpace(string(afterDirectFailure.Inventory)))))
}

func reconstructDeliveredMailText(t *testing.T, delivered, recipientName, body string) string {
	t.Helper()
	const header = " * * * * Dark Pawns Mail System * * * *\r\n"
	start := strings.Index(delivered, header)
	if start < 0 {
		t.Fatalf("delivered note lacks mail header: %q", delivered)
	}
	bodyOffset := strings.Index(delivered[start:], body)
	if bodyOffset < 0 {
		t.Fatalf("delivered note lacks body %q: %q", body, delivered)
	}
	bodyOffset += start
	prefix := delivered[start:bodyOffset]
	if !strings.Contains(prefix, "  To: "+recipientName+"\r\n") {
		t.Fatalf("delivered note header lacks recipient %q: %q", recipientName, prefix)
	}
	return prefix + body + strings.Repeat("\x00", game.MailHeaderDataSize-len(body))
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

func sha256Hex(value string) string {
	hash := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hash[:])
}
