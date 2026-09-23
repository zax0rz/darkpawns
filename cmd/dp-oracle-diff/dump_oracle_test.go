package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

// TestOracleBlocksTextMatchesShowOracle pins the dump format to the
// --show-oracle output: a dump file is that text, written to disk, so anything
// that reads a dump reads exactly what a human sees on the terminal.
func TestOracleBlocksTextMatchesShowOracle(t *testing.T) {
	diffs := []oraclediff.BlockDiff{
		{Command: "look", Oracle: "The temple is quiet.\n"},
		{Command: "quit", Oracle: "Goodbye, friend.. Come back soon!\n"},
	}
	want := "normalized C oracle blocks:\n" +
		"--- [look]\nThe temple is quiet.\n" +
		"--- [quit]\nGoodbye, friend.. Come back soon!\n"
	if got := oracleBlocksText(diffs); got != want {
		t.Fatalf("oracleBlocksText =\n%q\nwant\n%q", got, want)
	}
}

func TestWriteOracleDumpCreatesDirectoryAndFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dump")
	diffs := []oraclediff.BlockDiff{{Command: "look", Oracle: "The temple is quiet.\n"}}
	if err := writeOracleDump(dir, "look-start-room", diffs); err != nil {
		t.Fatalf("writeOracleDump: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(dir, "look-start-room.txt"))
	if err != nil {
		t.Fatalf("read dump: %v", err)
	}
	if string(body) != oracleBlocksText(diffs) {
		t.Fatalf("dump body = %q, want the --show-oracle text", body)
	}
}
