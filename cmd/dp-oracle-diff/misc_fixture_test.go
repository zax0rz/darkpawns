package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

func TestMiscFixturePairedBytesAndCleanup(t *testing.T) {
	sc, err := oraclediff.ParseScenario("misc", strings.NewReader("[fixture]\nmisc-file bugs 3 Paired evidence.\n[probe]\nsysfile bugs\n"))
	if err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, side := range []string{"c", "go"} {
		root, err := os.MkdirTemp(t.TempDir(), side)
		if err != nil {
			t.Fatal(err)
		}
		if err := applyMiscFileFixtures(root, sc.MiscFiles); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(root, "misc", "bugs"))
		if err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, string(data))
		if err := os.RemoveAll(root); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(root); !os.IsNotExist(err) {
			t.Fatalf("fixture root survived cleanup: %v", err)
		}
	}
	want := "01 Paired evidence.\n02 Paired evidence.\n03 Paired evidence.\n"
	if bodies[0] != want || bodies[1] != want {
		t.Fatalf("paired bodies: %q", bodies)
	}
	for _, fixture := range []string{"misc-file ../bugs 3 x", "misc-file bugs 0 x", "misc-file bugs 129 x", "misc-file bugs 3 " + strings.Repeat("x", 8192)} {
		if _, err := oraclediff.ParseScenario("bad", strings.NewReader("[fixture]\n"+fixture+"\n[probe]\nlook\n")); err == nil {
			t.Fatalf("accepted invalid fixture %q", fixture)
		}
	}
}
