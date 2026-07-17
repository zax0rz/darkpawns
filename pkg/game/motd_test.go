package game

import (
	"bytes"
	"os"
	"testing"
)

func TestShippedMOTDMatchesCFixture(t *testing.T) {
	want, err := os.ReadFile("testdata/c_motd.txt")
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile("../../lib/world/text/motd")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("shipped MOTD differs from C fixture\ngot:  %q\nwant: %q", got, want)
	}
}
