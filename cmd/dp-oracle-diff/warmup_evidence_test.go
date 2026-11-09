package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

func TestRetainWarmupEvidence(t *testing.T) {
	root := t.TempDir()
	blocks := []oraclediff.AudienceProbeBlock{{Command: "get chalice", Audience: "actor", Output: "\x1b[31mYou don't see a chalice here.\x1b[0m\r\n"}}
	for _, engine := range []string{"c", "go"} {
		if err := retainWarmupEvidence(root, "water.capture1", engine, blocks); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(root, "water.capture1", engine+"-warmup.json"))
		if err != nil {
			t.Fatal(err)
		}
		var got []oraclediff.AudienceProbeBlock
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0].Command != "get chalice" || got[0].Output != oraclediff.Normalize(blocks[0].Output) {
			t.Fatalf("retained=%#v", got)
		}
		data, err = os.ReadFile(filepath.Join(root, "water.capture1", engine+"-warmup.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), "--- [get chalice [actor]]\nYou don't see a chalice here.") || strings.Contains(string(data), "\x1b") {
			t.Fatalf("retained text=%q", data)
		}
	}
	// Recording never rewrites captured comparison material.
	if !strings.Contains(blocks[0].Output, "\x1b") {
		t.Fatal("mutated original capture")
	}
	if err := retainWarmupEvidence(root, "../escape", "c", blocks); err == nil {
		t.Fatal("accepted escaping evidence directory")
	}
}
