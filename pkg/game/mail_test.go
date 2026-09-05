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
