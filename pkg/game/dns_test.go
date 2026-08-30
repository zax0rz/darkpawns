package game

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestExecDNSAddDeletePrefixAndPersistenceState(t *testing.T) {
	w, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.WorldPath = filepath.Join(t.TempDir(), "world")

	var output strings.Builder
	w.MessageSink = func(_ string, message []byte) { output.Write(message) }
	ch := &Player{Name: "Admin"}
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	if page := w.ExecDns(ch, "add 001.002.003.004 HostFour"); page != "" {
		t.Fatalf("add returned page %q", page)
	}
	if page := w.ExecDns(ch, "add 001.002.003 HostThree"); page != "" {
		t.Fatalf("three-octet add returned page %q", page)
	}
	if got, want := output.String(), "OK!\r\nOK!\r\n"; got != want {
		t.Fatalf("add output = %q, want %q", got, want)
	}

	w.dnsMu.Lock()
	entries := append([]dnsEntry(nil), w.dnsCache[6]...)
	w.dnsMu.Unlock()
	if len(entries) != 2 {
		t.Fatalf("DNS bucket 6 has %d entries, want 2", len(entries))
	}
	if entries[0].name != "hostthree" || entries[0].ip[3] != -1 {
		t.Fatalf("three-octet entry = %#v, want prepended hostthree with fourth -1", entries[0])
	}
	if entries[1].name != "hostfour" || entries[1].ip[3] != 4 {
		t.Fatalf("four-octet entry = %#v, want hostfour with fourth 4", entries[1])
	}

	output.Reset()
	if page := w.ExecDns(ch, "del 1.2.3"); page != "" {
		t.Fatalf("delete returned page %q", page)
	}
	if got, want := output.String(), "Deleting hostthree.\r\nDeleting hostfour.\r\n"; got != want {
		t.Fatalf("delete output = %q, want %q", got, want)
	}

	w.dnsMu.Lock()
	deletions := append([]dnsEntry(nil), w.dnsCache[6]...)
	w.dnsMu.Unlock()
	for _, entry := range deletions {
		if entry.ip[0] != -1 {
			t.Fatalf("entry %q was not marked deleted: %#v", entry.name, entry)
		}
	}
}
