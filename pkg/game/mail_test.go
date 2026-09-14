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

	oldNameFunc, oldIDFunc := worldNameFunc, worldIDFunc
	t.Cleanup(func() {
		worldNameFunc, worldIDFunc = oldNameFunc, oldIDFunc
	})
	if InitMailSystem(nil, nil) {
		t.Fatal("mail initialization succeeded for a directory mail store")
	}
	info, err := os.Stat("data/mail")
	if err != nil {
		t.Fatalf("stat mail store: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("mail initialization replaced the unusable store")
	}
}
