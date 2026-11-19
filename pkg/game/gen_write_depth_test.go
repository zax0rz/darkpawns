package game

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// TestDoGenWriteNPCGate pins the handler-level NPC refusal from
// src/act.other.c:1103-1106. A switched mobile must receive this through its
// descriptor; the detached original player must receive nothing.
func TestDoGenWriteNPCGate(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)
	mob := &MobInstance{world: w}
	var bodyOutput string
	w.MobileDescriptorOwner = func(body *MobInstance) *Player {
		if body == mob {
			return ch
		}
		return nil
	}
	w.MobileMessageSink = func(body *MobInstance, message []byte) {
		if body == mob {
			bodyOutput += string(message)
		}
	}

	w.doGenWrite(ch, mob, "bug", "ignored report")
	if got, want := bodyOutput, "Monsters can't have ideas - Go away.\r\n"; got != want {
		t.Fatalf("switched NPC response = %q, want %q", got, want)
	}
	if got := lastMsg(); got != "" {
		t.Fatalf("descriptorless owner received NPC response %q", got)
	}
}

// TestDoGenWritePlayerRecordAndResponse pins the successful C path from
// src/act.other.c:1107-1127. It uses a temporary working directory because
// the original and the port append to relative misc/* files. The test checks
// C's six-byte month/day rendering, room field, doubled-dollar deletion, and
// the one shared success response for all four subcommands.
func TestDoGenWritePlayerRecordAndResponse(t *testing.T) {
	t.Chdir(t.TempDir())
	w, ch, lastMsg := newDonateTestWorld(t)

	cases := []struct {
		cmd      string
		filename string
	}{
		{"bug", "misc/bugs"},
		{"typo", "misc/typos"},
		{"idea", "misc/ideas"},
		{"todo", "misc/todo"},
	}

	for _, tc := range cases {
		w.doGenWrite(ch, nil, tc.cmd, "report $$details")
		if got, want := lastMsg(), "Okay.  Thanks!\r\n"; got != want {
			t.Errorf("%s success response = %q, want %q", tc.cmd, got, want)
		}

		record, err := os.ReadFile(tc.filename)
		if err != nil {
			t.Fatalf("read %s: %v", tc.filename, err)
		}
		day := time.Now().Format("Jan _2")
		prefix := fmt.Sprintf("%-8s (%6.6s) [%5d] ", ch.Name, day, ch.GetRoomVNum())
		if !strings.HasPrefix(string(record), prefix) {
			t.Errorf("%s record = %q, want prefix %q", tc.cmd, record, prefix)
		}
		if !strings.HasSuffix(string(record), "report $details\n") {
			t.Errorf("%s record = %q, want doubled dollars deleted", tc.cmd, record)
		}
	}
}

// TestDoGenWriteEntryGates pins skip_spaces and the subcommand switch from
// src/act.other.c:1086-1117. Unknown subcommands return silently before the
// NPC and argument branches; whitespace-only arguments reach C's common
// mistake message.
func TestDoGenWriteEntryGates(t *testing.T) {
	t.Chdir(t.TempDir())
	w, ch, lastMsg := newDonateTestWorld(t)

	w.doGenWrite(ch, nil, "bug", " \t")
	if got, want := lastMsg(), "That must be a mistake...\r\n"; got != want {
		t.Errorf("whitespace-only response = %q, want %q", got, want)
	}

	w.doGenWrite(ch, nil, "unknown", "report")
	if got := lastMsg(); got != "" {
		t.Errorf("unknown subcommand response = %q, want silent return", got)
	}

	if err := os.Mkdir("misc", 0o500); err != nil {
		t.Fatalf("make unwritable report directory: %v", err)
	}
	w.doGenWrite(ch, nil, "bug", "report")
	if got, want := lastMsg(), "Could not open the file.  Sorry.\r\n"; got != want {
		t.Errorf("open-failure response = %q, want %q", got, want)
	}
}
