package game

import (
	"os"
	"strings"
	"testing"
)

// TestMailReceivePreservesFixedBlockPaddingInRuntimeText characterizes the
// native Go mail path without touching a database. C zero-terminates each
// fixed text field before concatenation; the current Go conversion copies the
// whole fixed array into the delivered string.
func TestMailReceivePreservesFixedBlockPaddingInRuntimeText(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("data", 0o700); err != nil {
		t.Fatalf("create mail data directory: %v", err)
	}

	oldIndex, oldFreeList, oldFileEndPos := mailIndex, freeList, fileEndPos
	oldDisabled := mailDisabled
	oldNameFunc, oldIDFunc := worldNameFunc, worldIDFunc
	t.Cleanup(func() {
		mailIndex, freeList, fileEndPos = oldIndex, oldFreeList, oldFileEndPos
		mailDisabled = oldDisabled
		worldNameFunc, worldIDFunc = oldNameFunc, oldIDFunc
	})

	w, sender, postmasterFn, postmasterMob, _ := newMailLifecycleWorld(t)
	if !InitMailSystem(mailLifecycleNameByID, mailLifecycleIDByName) {
		t.Fatal("mail initialization failed")
	}

	postmasterFn(w, sender, postmasterMob, "mail", "Recipient body")
	if HandleMailInput(sender, mailLifecycleBody) {
		t.Fatal("mail body line unexpectedly completed composition")
	}
	if !HandleMailInput(sender, "@") {
		t.Fatal("@ did not complete mail composition")
	}

	recipient := NewPlayer(mailLifecycleRecipientID, "Recipient", 1001)
	if err := w.AddPlayer(recipient); err != nil {
		t.Fatalf("add recipient: %v", err)
	}
	postmasterFn(w, recipient, postmasterMob, "receive", "")

	items := recipient.Inventory.FindItems("")
	if len(items) != 1 {
		t.Fatalf("received inventory items = %d, want 1", len(items))
	}
	got := items[0].Runtime.MailText
	wantPadding := MailHeaderDataSize - len(mailLifecycleBody)
	if got == "" || !strings.HasSuffix(got, mailLifecycleBody+strings.Repeat("\x00", wantPadding)) {
		t.Fatalf("runtime mail text does not preserve fixed header tail: bytes=%d sanitized=%q", len(got), strings.ReplaceAll(got, "\x00", "<NUL>"))
	}
	if gotNULs := strings.Count(got, "\x00"); gotNULs != wantPadding {
		t.Fatalf("runtime mail text NUL count = %d, want %d", gotNULs, wantPadding)
	}
	t.Logf("mail_runtime_text_bytes=%d mail_runtime_text_nul_count=%d first_nul_offset=%d body=%q", len(got), strings.Count(got, "\x00"), strings.IndexByte(got, 0), mailLifecycleBody)
}
