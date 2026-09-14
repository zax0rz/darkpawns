package game

import (
	"bytes"
	"os"
	"testing"
)

func TestMailReadWriteSharedFilePositioning(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("data", 0o700); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}

	want := make([]byte, MailBlockSize)
	for i := range want {
		want[i] = byte(i)
	}
	writeToFile(want, len(want), 0)

	got := make([]byte, MailBlockSize)
	readFromFile(got, len(got), 0)
	if !bytes.Equal(got, want) {
		t.Fatalf("read mail block differs from written block")
	}
}

func TestMailInitializationFailsClosedWithoutOverwritingUnusableStore(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("data/mail", 0o700); err != nil {
		t.Fatalf("mkdir mail directory: %v", err)
	}

	oldDisabled := mailDisabled
	oldNameFunc, oldIDFunc := worldNameFunc, worldIDFunc
	t.Cleanup(func() {
		worldNameFunc, worldIDFunc = oldNameFunc, oldIDFunc
		mailDisabled = oldDisabled
	})
	worldNameFunc = func(int) string { return "unexpected" }
	worldIDFunc = func(string) int { return 999 }
	if InitMailSystem(nil, nil) {
		t.Fatal("mail initialization succeeded for a directory mail store")
	}
	if got := GetIDByName("Recipient"); got != -1 {
		t.Fatalf("failed mail initialization left name lookup enabled: %d", got)
	}
	if got := GetNameByID(101); got != "Player(101)" {
		t.Fatalf("failed mail initialization left ID lookup enabled: %q", got)
	}
	info, err := os.Stat("data/mail")
	if err != nil {
		t.Fatalf("stat mail store: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("mail initialization replaced the unusable store")
	}
}

// C mail.c:484-487: all three commands emit the same bytes and fall through.
func TestPostmasterDisabledDispatch(t *testing.T) {
	oldDisabled := mailDisabled
	t.Cleanup(func() { mailDisabled = oldDisabled })
	mailDisabled = true
	w, sender, fn, mob, messages := newMailLifecycleWorld(t)
	for _, cmd := range []string{"mail", "check", "receive", "look"} {
		messages[sender.Name] = nil
		if fn(w, sender, mob, cmd, "Recipient body") {
			t.Fatalf("disabled %s was handled", cmd)
		}
		got := messages[sender.Name]
		if cmd == "look" {
			if len(got) != 0 {
				t.Fatalf("unrelated command output: %q", got)
			}
		} else if len(got) != 1 || got[0] != "Sorry, the mail system is having technical difficulties.\r\n" {
			t.Fatalf("%s output: %q", cmd, got)
		}
	}
}

func TestMailObjectCKeywords(t *testing.T) {
	obj := (&World{}).CreateMailObject(nil, "body")
	if obj.Runtime.Keywords != "mail paper letter" || obj.GetShortDesc() != "a piece of mail" || obj.GetTypeFlag() != ITEM_NOTE || !obj.CanPickUp {
		t.Fatalf("mail object does not match C identity: %+v", obj)
	}
}
