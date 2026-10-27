package main

// fault_fixture_test.go — symmetry checks for the fault-injection verbs
// (docs/design/fault-injection-fixture.md): every logical target must
// resolve on BOTH engines' trees, and the board map must cover vnums both
// sides actually define.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

func TestFaultPathsResolveOnBothSides(t *testing.T) {
	trees := faultTrees{oracleLib: "/tmp/o-lib", goRoot: "/tmp/go-root", goWorld: "/tmp/go-world"}
	for _, ff := range []oraclediff.FaultFixture{
		{Verb: "fault-dir", Target: "mail"},
		{Verb: "fault-dir", Target: "object_idlist"},
		{Verb: "fault-corrupt-mail", Target: "unaligned"},
		{Verb: "fault-board", Target: "19652", Detail: "empty-message-huge-slot"},
	} {
		cPath, goPath, err := faultPaths(trees, ff)
		if err != nil {
			t.Fatalf("%v: %v", ff, err)
		}
		if cPath == "" || !strings.HasPrefix(cPath, trees.oracleLib) {
			t.Fatalf("%v: C path %q must resolve under the oracle lib", ff, cPath)
		}
		if goPath == "" || !strings.HasPrefix(goPath, trees.goRoot) && !strings.HasPrefix(goPath, trees.goWorld) {
			t.Fatalf("%v: Go path %q must resolve under the Go game root or world", ff, goPath)
		}
	}

	// The crash-file fault is deliberately one-sided on the filesystem: C's
	// file becomes a directory while Go's fault arrives through
	// DP_FAULT_OBJECT_SAVE. The path must still name C's bucketed location.
	cPath, goPath, err := faultPaths(trees, oraclediff.FaultFixture{Verb: "fault-crash-file", Target: "Unlucky"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(cPath, "plrobjs/U-Z/unlucky.objs") {
		t.Fatalf("crash file C path %q lacks the U-Z bucket (src/utils.c:555-578)", cPath)
	}
	if goPath != "" {
		t.Fatalf("crash file has no Go filesystem twin (the env seam owns it), got %q", goPath)
	}
}

func TestFaultBoardFilesCoveredVnums(t *testing.T) {
	// Both engines define the trinity board at vnum 19652 with the same
	// filename (src/boards.c:92-102; pkg/boards/boards.go defaultBoardInfo).
	if got := faultBoardFiles[19652]; got != "etc/board.trinity" {
		t.Fatalf("board map: 19652 -> %q, want etc/board.trinity", got)
	}
}

func TestParseFaultStepRejectsUnknown(t *testing.T) {
	for _, bad := range [][]string{
		{"fault-dir"},
		{"fault-dir", "etc/passwd"},
		{"fault-corrupt-mail", "sideways"},
		{"fault-crash-file", "x"},
	} {
		if _, ok := parseFaultStep(bad); ok {
			t.Fatalf("parseFaultStep(%q) accepted", bad)
		}
	}
	if _, ok := parseFaultStep([]string{"fault-dir", "mail"}); !ok {
		t.Fatal("parseFaultStep rejected a valid fault-dir mail step")
	}
	if _, ok := parseFaultStep([]string{"fault-corrupt-mail", "bad-header"}); !ok {
		t.Fatal("parseFaultStep rejected a valid corrupt-mail step")
	}
}
