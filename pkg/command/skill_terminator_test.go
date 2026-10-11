package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// PF-063: sendSkillResult appended "\r\n" to every literal message, so any
// message transcribed with its C terminator (send_to_char writes the string
// and nothing else, src/comm.c) went out double-terminated — a blank line C
// never prints. These tests pin the exact wire bytes: fail them by reverting
// skillMessageTerminator to an unconditional append.

// TestSendSkillResultSendsTerminatedMessageExactlyOnce is the cluster-1
// shape: do_detect's opening line carries its own terminator
// (src/new_cmds2.c:516) and must reach the player exactly as C wrote it.
func TestSendSkillResultSendsTerminatedMessageExactlyOnce(t *testing.T) {
	session := newSkillCommandSession(t)
	result := game.SkillResult{
		Success:     false,
		MessageToCh: "You carefully check the room...\r\n",
	}
	if err := sendSkillResult(session, session.player, nil, result); err != nil {
		t.Fatalf("sendSkillResult: %v", err)
	}
	if len(session.messages) != 1 {
		t.Fatalf("session.messages = %q, want exactly one message", session.messages)
	}
	if got, want := session.messages[0], "You carefully check the room...\r\n"; got != want {
		t.Fatalf("actor bytes = %q, want %q (double terminator prints a blank C never sends)", got, want)
	}
}

// TestSendSkillResultUnterminatedLiteralStillGainsTerminator guards the
// seam's other half: literals transcribed without a terminator (e.g.
// skill_special.go's "Yeah, right.") still get exactly one.
func TestSendSkillResultUnterminatedLiteralStillGainsTerminator(t *testing.T) {
	session := newSkillCommandSession(t)
	result := game.SkillResult{MessageToCh: "Yeah, right."}
	if err := sendSkillResult(session, session.player, nil, result); err != nil {
		t.Fatalf("sendSkillResult: %v", err)
	}
	if len(session.messages) != 1 || session.messages[0] != "Yeah, right.\r\n" {
		t.Fatalf("actor bytes = %q, want [\"Yeah, right.\\r\\n\"]", session.messages)
	}
}

// TestSendSkillResultLiteralAudiencesExactBytes covers the sendLiteralMessages
// path (SkillMsgAfterDamage): actor, victim, and both room messages each
// carry C's single terminator and must arrive byte-exact on every audience.
func TestSendSkillResultLiteralAudiencesExactBytes(t *testing.T) {
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	sink := make(map[string]string)
	world.MessageSink = func(name string, message []byte) { sink[name] += string(message) }

	actor := game.NewPlayer(1, "Actor", 1001)
	if err := world.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	victim := game.NewPlayer(2, "Victim", 1001)
	if err := world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	observer := game.NewPlayer(3, "Observer", 1001)
	if err := world.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}

	session := &skillCommandSession{player: actor, world: world}
	result := game.SkillResult{
		SkillMsgAfterDamage: true,
		MessageToCh:         "You carefully check the room...\r\n",
		MessageToVict:       "You can't seem to find anything.\r\n",
		MessageToRoom:       "Actor searches the room carefully.\r\n",
		MessageToRoomSecond: "Actor gives up the search.\r\n",
	}
	if err := sendSkillResult(session, actor, victim, result); err != nil {
		t.Fatalf("sendSkillResult: %v", err)
	}

	if len(session.messages) != 1 || session.messages[0] != "You carefully check the room...\r\n" {
		t.Fatalf("actor bytes = %q, want exactly one terminated message", session.messages)
	}
	if got, want := sink[victim.Name], "You can't seem to find anything.\r\n"; got != want {
		t.Fatalf("victim bytes = %q, want %q", got, want)
	}
	wantRoom := "Actor searches the room carefully.\r\n" + "Actor gives up the search.\r\n"
	if got := sink[observer.Name]; got != wantRoom {
		t.Fatalf("observer bytes = %q, want %q", got, wantRoom)
	}
	if got := sink[actor.Name]; got != "" {
		t.Fatalf("actor sink bytes = %q, want none (actor excluded from room sends)", got)
	}
}

// TestSkillMessageTerminator pins the helper itself across the shapes a
// SkillResult literal can have. C spells message terminators both ways —
// "\r\n" and "\n\r" (do_compare: src/new_cmds.c:1984/1991,
// do_flesh_alter: src/new_cmds.c:1900) — and either spelling terminates the
// message; the LFCR rows fail on the CRLF-only suffix test (PF-063 cluster 3).
func TestSkillMessageTerminator(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Yeah, right.\r\n", "Yeah, right.\r\n"},
		{"Yeah, right.", "Yeah, right.\r\n"},
		{"multi\r\nline\r\n", "multi\r\nline\r\n"},
		{"", "\r\n"},
		{"They're the same thing!\n\r", "They're the same thing!\n\r"},
		{"Looks like you don't have those objects..\n\r", "Looks like you don't have those objects..\n\r"},
		{"You know nothing of altering your flesh!\n\r", "You know nothing of altering your flesh!\n\r"},
	}
	for _, tc := range cases {
		if got := skillMessageTerminator(tc.in); got != tc.want {
			t.Errorf("skillMessageTerminator(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestSendSkillResultSendsLFCRBytesExactly pins the exact wire bytes —
// including the LFCR spelling itself, which neither comparator can see
// (normalization folds both spellings) — for do_compare's two LFCR
// literals (src/new_cmds.c:1984, :1991) and do_flesh_alter's gate
// (src/new_cmds.c:1900) through the full sendSkillResult seam. Fails on the
// CRLF-only suffix test: the appended "\r\n" both adds a blank line and
// corrupts the terminator bytes.
func TestSendSkillResultSendsLFCRBytesExactly(t *testing.T) {
	for _, msg := range []string{
		"Looks like you don't have those objects..\n\r",
		"They're the same thing!\n\r",
		"You know nothing of altering your flesh!\n\r",
	} {
		session := newSkillCommandSession(t)
		result := game.SkillResult{MessageToCh: msg}
		if err := sendSkillResult(session, session.player, nil, result); err != nil {
			t.Fatalf("sendSkillResult: %v", err)
		}
		if len(session.messages) != 1 || session.messages[0] != msg {
			t.Fatalf("actor bytes = %q, want exactly [%q] (LFCR terminator preserved, nothing appended)",
				session.messages, msg)
		}
	}
}
