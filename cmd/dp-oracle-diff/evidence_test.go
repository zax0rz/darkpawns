package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

func TestEvidenceRejectsHostileNames(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"../outside", "safe/../../outside", "safe/../outside", filepath.Join(root, "absolute"), `safe\..\outside`, "", "."} {
		t.Run(name, func(t *testing.T) {
			if err := retainEvidence(root, name, "", "", nil, "", ""); err == nil {
				t.Fatalf("accepted hostile evidence name %q", name)
			}
		})
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("rejected names wrote evidence: %v", entries)
	}
}

func TestEvidenceStaysUnderRoot(t *testing.T) {
	root := t.TempDir()
	blocks := []oraclediff.BlockDiff{{Command: "look", Oracle: "Room\n", Go: "Room\n"}}
	if err := retainEvidence(filepath.Join(root, "."), "run/scenario.capture1", "", "", blocks, "C log", "Go log"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "run/scenario.capture1/go-server.log"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "Go log" {
		t.Fatalf("wrong retained log: %q", data)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := retainEvidence(root, "escape/scenario.capture1", "", "", blocks, "", ""); err == nil {
		t.Fatal("accepted symlink outside evidence root")
	}
	entries, err := os.ReadDir(outside)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("escaped evidence root: %v", entries)
	}
}
