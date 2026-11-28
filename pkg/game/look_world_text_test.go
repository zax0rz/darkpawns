package game

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C writes a room's name, an extra description and an exit's description with
// send_to_char, which writes the stored string as it is (src/comm.c:2335-2339).
// The port framed each one as an act() line, and a framed line is other bytes
// wherever the string begins in lowercase, ends in blank lines, or leaves its
// last line open. worldText frames only where the framed line is C's bytes.
//
// The parser stores a line break as LF and fread_string as CRLF
// (src/db.c:2705-2717); the line carrying the tilde gets none (:2705-2710).
// Each want below is C's write, in that stored form.

func newWorldTextWorld(t *testing.T, world *parser.World) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(world)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	ch := NewPlayer(1, "Viewer", 1001)
	ch.SetAutoExit(false)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, ch
}

// checkWorldText renders one look result and compares its writes with C's:
// want is every write in delivery order, and wantLiteral the part that must
// reach the descriptor untouched. A framed write is a line act() has ended;
// a literal one is the bytes themselves, which is where a send_to_char buffer
// that act() could not reproduce must arrive.
func checkWorldText(t *testing.T, w *World, result ObservationResult, want, wantLiteral string) {
	t.Helper()
	var all, literal strings.Builder
	w.MessageSink = func(_ string, msg []byte) { all.Write(msg) }
	w.LiteralMessageSink = func(_ string, msg []byte) {
		all.Write(msg)
		literal.Write(msg)
	}
	w.RenderObservationMessages(result)
	if got := all.String(); got != want {
		t.Errorf("look writes = %q, want C's writes %q", got, want)
	}
	if got := literal.String(); got != wantLiteral {
		t.Errorf("literal writes = %q, want C's untouched bytes %q", got, wantLiteral)
	}
}

// look_at_room sends a room's name with send_to_char, then the colour reset
// and "\r\n" (src/act.informative.c:805-811). Framed, act()'s cap()
// capitalized a name that begins in lowercase; the shipped world has one,
// 8111 in lib/world/wld/80.wld. With colour on, the three list brackets are
// literal writes too (src/act.informative.c:823-836).
func TestRoomLookNameIsWrittenAsCSendsIt(t *testing.T) {
	const brackets = "\x1b[32m\x1b[33m\x1b[0m"
	lowercase := parser.Room{VNum: 1001, Name: "stable room, do not delete"}
	cases := []struct {
		name        string
		room        parser.Room
		colour      bool
		roomFlags   bool
		want        string
		wantLiteral string
	}{
		{
			name:        "name begins in lowercase",
			room:        lowercase,
			want:        "stable room, do not delete\r\n",
			wantLiteral: "stable room, do not delete\r\n",
		},
		{
			// CCCYN and CCNRM expand around the name, so cap() sees the
			// escape first and changes nothing.
			name:        "name begins in lowercase, colour on",
			room:        lowercase,
			colour:      true,
			want:        "\x1b[36mstable room, do not delete\x1b[0m\r\n" + brackets,
			wantLiteral: brackets,
		},
		{
			name: "name begins in uppercase",
			room: parser.Room{VNum: 1001, Name: "Temple of Midgaard"},
			want: "Temple of Midgaard\r\n",
		},
		{
			name:      "room flags replace the name",
			room:      lowercase,
			roomFlags: true,
			want:      "[ 1001] stable room, do not delete [ NOBITS ] [ Inside ]\r\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := newWorldTextWorld(t, &parser.World{Rooms: []parser.Room{tc.room}})
			if tc.colour {
				ch.SetPlrFlag(PrfColor1, true)
				ch.SetPlrFlag(PrfColor2, true)
			}
			if tc.roomFlags {
				ch.SetPlrFlag(PrfRoomFlags, true)
			}
			checkWorldText(t, w, w.DoLookRoomAt(ch, 1001, true), tc.want, tc.wantLiteral)
		})
	}
}

// look_at_target sends a room's extra description with send_to_char and
// returns (src/act.informative.c:1033-1037).
func TestRoomExtraDescriptionIsWrittenAsCSendsIt(t *testing.T) {
	cases := []struct {
		name   string
		desc   string
		framed bool // the framed line is C's bytes as well
		want   string
	}{
		{
			name: "trailing blank line",
			desc: "The sign says:\nWelcome.\n\n",
			want: "The sign says:\r\nWelcome.\r\n\r\n",
		},
		{
			// The painting in 20073 (lib/world/wld/200.wld:1929-1932).
			name: "two trailing blank lines",
			desc: "and find yourself...\n\n\n",
			want: "and find yourself...\r\n\r\n\r\n",
		},
		{
			name: "last line carries the tilde",
			desc: "The sign is blank.",
			want: "The sign is blank.",
		},
		{
			name: "first letter lowercase",
			desc: "a small sign stands here.\n",
			want: "a small sign stands here.\r\n",
		},
		{
			name:   "one line ending",
			desc:   "The sign says:\nWelcome.\n",
			framed: true,
			want:   "The sign says:\r\nWelcome.\r\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := newWorldTextWorld(t, &parser.World{Rooms: []parser.Room{{
				VNum: 1001, Name: "Square",
				ExtraDescs: []parser.ExtraDesc{{Keywords: "sign", Description: tc.desc}},
			}}})
			wantLiteral := tc.want
			if tc.framed {
				wantLiteral = ""
			}
			checkWorldText(t, w, w.DoLookTarget(ch, "sign"), tc.want, wantLiteral)
		})
	}
}

// look_at_target sends an object's extra description with send_to_char and
// follows it with show_obj_to_char's mode 6: the object's flags and one CRLF,
// or that CRLF alone (src/act.informative.c:1058-1071, :124-178). The flags
// line is a framed act() line; the lone CRLF is a bare write.
func TestObjectExtraDescriptionIsWrittenAsCSendsIt(t *testing.T) {
	cases := []struct {
		name        string
		desc        string
		glow        bool
		framed      bool // the description's framed line is C's bytes as well
		want        string
		wantLiteral string
	}{
		{
			name:        "trailing blank line",
			desc:        "A label.\n\n",
			want:        "A label.\r\n\r\n\r\n",
			wantLiteral: "A label.\r\n\r\n\r\n",
		},
		{
			name:        "trailing blank line before the flags line",
			desc:        "A label.\n\n",
			glow:        true,
			want:        "A label.\r\n\r\n (glowing)\r\n",
			wantLiteral: "A label.\r\n\r\n",
		},
		{
			name:        "first letter lowercase",
			desc:        "a label.\n",
			want:        "a label.\r\n\r\n",
			wantLiteral: "a label.\r\n\r\n",
		},
		{
			// create_money builds a description with no line ending; C's
			// next write continues the line.
			name:        "no line ending",
			desc:        "some coins",
			want:        "some coins\r\n",
			wantLiteral: "some coins\r\n",
		},
		{
			name:        "one line ending",
			desc:        "A label.\n",
			framed:      true,
			want:        "A label.\r\n\r\n",
			wantLiteral: "\r\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flags := [4]int{}
			if tc.glow {
				flags[0] = 1 << itemExtraGlow
			}
			proto := parser.Obj{
				VNum: 7001, Keywords: "label parchment", ShortDesc: "a label", LongDesc: "A label lies here.",
				ExtraFlags: flags,
				ExtraDescs: []parser.ExtraDesc{{Keywords: "label", Description: tc.desc}},
			}
			w, ch := newWorldTextWorld(t, &parser.World{
				Rooms: []parser.Room{{VNum: 1001, Name: "Square"}},
				Objs:  []parser.Obj{proto},
			})
			w.AddItemToRoom(NewObjectInstance(&proto, 1001), 1001)
			checkWorldText(t, w, w.DoLookTarget(ch, "label"), tc.want, tc.wantLiteral)
		})
	}
}

// look_in_direction sends an exit's description with send_to_char and returns
// (src/act.informative.c:856-860). 418 of the shipped exits end theirs on the
// line that carries the tilde, so C leaves the line open.
func TestExitDescriptionIsWrittenAsCSendsIt(t *testing.T) {
	cases := []struct {
		name   string
		dir    string
		desc   string
		framed bool   // the framed line is C's bytes as well
		want   string // C's writes after "You look <dir>wards."
	}{
		{
			name: "last line carries the tilde",
			dir:  "north",
			desc: "The street continues north.",
			want: "The street continues north.",
		},
		{
			name: "several lines, the last carries the tilde",
			dir:  "north",
			desc: "Line one.\nLine two.",
			want: "Line one.\r\nLine two.",
		},
		{
			// The west door of 2885 (lib/world/wld/27.wld).
			name: "trailing blank line",
			dir:  "west",
			desc: "You look towards Caine's rooms.\n\n",
			want: "You look towards Caine's rooms.\r\n\r\n",
		},
		{
			name: "first letter lowercase",
			dir:  "north",
			desc: "the door is barred.\n",
			want: "the door is barred.\r\n",
		},
		{
			name:   "one line ending",
			dir:    "north",
			desc:   "It leads away.\n",
			framed: true,
			want:   "It leads away.\r\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := newWorldTextWorld(t, &parser.World{Rooms: []parser.Room{
				{VNum: 1001, Name: "Square", Exits: map[string]parser.Exit{
					tc.dir: {Direction: tc.dir, ToRoom: 1002, Description: tc.desc},
				}},
				{VNum: 1002, Name: "Beyond"},
			}})
			preface := "You look " + tc.dir + "wards.\r\n"
			wantLiteral := tc.want
			if tc.framed {
				wantLiteral = ""
			}
			checkWorldText(t, w, w.DoLookDirection(ch, directionIndex(tc.dir)), preface+tc.want, wantLiteral)
		})
	}
}

// look_at_char sends a mobile's detailed description with send_to_char
// (src/act.informative.c:394-395) and follows it with diag_char_to_char's one
// line (src/act.informative.c:354-384). 55 shipped descriptions end in a blank
// line. They take the raw path, not literal(), so nothing trimmed them; this
// guards that the class's largest member stays as it is.
func TestLookAtMobileKeepsDescriptionTrailingBlankLines(t *testing.T) {
	w, ch := newWorldTextWorld(t, &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Pond"}},
		Mobs: []parser.Mob{{
			VNum: 9001, Keywords: "frog", ShortDesc: "a frog", LongDesc: "A frog sits here.\n",
			DetailedDesc: "He is tall.\n\n",
		}},
	})
	mob, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	condition := cap("a frog " + diagCondition(mob.GetHP(), mob.GetMaxHP()))
	// The raw write hands the sink the parser's LF line breaks; the session's
	// text frame turns each into CRLF (NormalizeCRLF, pkg/session/terminal.go)
	// and, unlike literal(), trims nothing.
	checkWorldText(t, w, w.DoLookTarget(ch, "frog"), "He is tall.\n\n"+condition+"\r\n", "")
}

// list_one_char writes a mobile that stands at its default position with one
// send_to_char of its long description (src/act.informative.c:511-526): no
// capital, and the line breaks as they are stored. Framed, act() capitalized
// it; the shipped world has one such mobile, 9918, "a cloven footed goat-like
// demon wanders around here." (lib/world/mob/99.mob). The sanctuary line
// after it is an act() line (src/act.informative.c:528-535).
func TestRoomLookMobileLongDescriptionIsWrittenAsCSendsIt(t *testing.T) {
	cases := []struct {
		name        string
		long        string
		sanctuary   bool
		framed      bool // the framed line is C's bytes as well
		want        string
		wantLiteral string
	}{
		{
			name:        "first letter lowercase",
			long:        "a cloven footed goat-like demon wanders around here.\n",
			want:        "a cloven footed goat-like demon wanders around here.\r\n",
			wantLiteral: "a cloven footed goat-like demon wanders around here.\r\n",
		},
		{
			name:        "trailing blank line",
			long:        "A frog sits here.\n\n",
			want:        "A frog sits here.\r\n\r\n",
			wantLiteral: "A frog sits here.\r\n\r\n",
		},
		{
			// C's next write continues the description's last line.
			name:        "no line ending, then an act() line",
			long:        "A frog sits here.",
			sanctuary:   true,
			want:        "A frog sits here....it glows with a bright light!\r\n",
			wantLiteral: "A frog sits here.",
		},
		{
			name:   "one line ending",
			long:   "A frog sits here.\n",
			framed: true,
			want:   "A frog sits here.\r\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, ch := newWorldTextWorld(t, &parser.World{
				Rooms: []parser.Room{{VNum: 1001, Name: "Pond"}},
				Mobs: []parser.Mob{{
					VNum: 9001, Keywords: "frog", ShortDesc: "a frog", LongDesc: tc.long,
					Position: posStanding, DefaultPos: posStanding,
				}},
			})
			mob, err := w.SpawnMob(9001, 1001)
			if err != nil {
				t.Fatalf("SpawnMob: %v", err)
			}
			if tc.sanctuary {
				mob.SetAffected(affSanctuary)
			}
			checkWorldText(t, w, w.DoLookRoomAt(ch, 1001, true), "Pond\r\n"+tc.want, tc.wantLiteral)
		})
	}
}

// do_switch moves the wizard's descriptor to the mobile body
// (src/act.wizard.c:1198-1199), so a send_to_char to that body is a write to
// the wizard's connection, the same bytes it would be to the wizard. The port
// framed them for a mobile: do_description's space became a line of its own,
// and a lowercase room name was capitalized. A mobile has no colour
// preferences, so the list brackets are empty writes here.
func TestSwitchedMobileLookWritesThroughTheLiteralSink(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "stable room, do not delete", Description: "A stable.\n",
				ExtraDescs: []parser.ExtraDesc{{Keywords: "straw", Description: "Straw."}},
			},
			{VNum: 1002, Name: "Yard"},
		},
		Mobs: []parser.Mob{{VNum: 9001, Keywords: "pony", ShortDesc: "a pony", LongDesc: "a pony stands here.\n"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	pony, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}

	var framed, literal strings.Builder
	w.MobileMessageSink = func(_ *MobInstance, msg []byte) { framed.Write(msg) }
	w.MobileLiteralSink = func(_ *MobInstance, msg []byte) { literal.Write(msg) }
	w.MobileDescriptorOwner = func(*MobInstance) *Player { return NewPlayer(2, "Wizard", 1002) }

	// look_at_room writes the name, then do_description's sprintf("%s ",
	// description) (src/act.informative.c:2784), whose space starts
	// whatever is written next.
	w.RenderObservationMessages(w.DoLookRoom(pony, true))
	const wantName, wantDescription = "stable room, do not delete\r\n", "A stable.\r\n"
	if got := literal.String(); got != wantName+" " {
		t.Errorf("switched mobile's literal writes = %q, want C's name write and do_description's space %q", got, wantName+" ")
	}
	if got := framed.String(); got != wantDescription {
		t.Errorf("switched mobile's framed writes = %q, want only the description line %q", got, wantDescription)
	}
}

// The shipped world files, read by the shipped parser. fread_string keeps an
// empty last line as a CRLF (src/db.c:2705-2717) and so does readTildeString:
// nine shipped room descriptions end in one, which a note in #3116 doubted.
func TestShippedWorldTextReachesTheLookAsStored(t *testing.T) {
	libWorld, err := filepath.Abs("../../lib/world")
	if err != nil {
		t.Fatal(err)
	}
	var rooms []parser.Room
	for _, name := range []string{"27.wld", "80.wld", "200.wld"} {
		parsed, err := parser.ParseWldFile(filepath.Join(libWorld, "wld", name))
		if err != nil {
			t.Fatalf("ParseWldFile(%s): %v", name, err)
		}
		rooms = append(rooms, parsed...)
	}
	w, ch := newWorldTextWorld(t, &parser.World{Rooms: rooms})
	byVNum := make(map[int]parser.Room, len(rooms))
	for _, room := range rooms {
		byVNum[room.VNum] = room
	}

	t.Run("2854 description keeps its empty last line", func(t *testing.T) {
		// This held before worldText: writeRoomDescription already wrote such
		// a buffer verbatim. It answers #3116's note; it is not a fix.
		const stored = "  This is a small balcony over the Great Hall where the musicians sit\nand play.\n\n"
		if got := byVNum[2854].Description; got != stored {
			t.Fatalf("parser stored %q, want fread_string's string with LF breaks %q", got, stored)
		}
		description := strings.ReplaceAll(stored, "\n", "\r\n")
		checkWorldText(t, w, w.DoLookRoomAt(ch, 2854, true), "Musicians' Loft\r\n"+description, description)
	})
	t.Run("8111 name", func(t *testing.T) {
		const want = "stable room, do not delete\r\nstable room\r\n"
		checkWorldText(t, w, w.DoLookRoomAt(ch, 8111, true), want, want)
	})
	t.Run("20073 painting", func(t *testing.T) {
		ch.SetRoom(20073)
		want := "The painting is very very large and seems to depict some sort of strange\r\n" +
			"garden hedge maze.  As you look at it you swear it is moving!  You begin\r\n" +
			"to lose track of time as your eyes search out every last detail of the\r\n" +
			"artist's masterpiece.  Soon you realize that the air has begun to smell\r\n" +
			"like flowers and the room has grown a great deal brighter.  The world\r\n" +
			"spins and you feel like you are high.  Abruptly, you snap to your senses\r\n" +
			"and find yourself...\r\n\r\n\r\n"
		checkWorldText(t, w, w.DoLookTarget(ch, "painting"), want, want)
	})
	t.Run("2885 west door", func(t *testing.T) {
		ch.SetRoom(2885)
		const want = "You look westwards.\r\nYou look towards Caine's rooms.\r\n\r\n"
		checkWorldText(t, w, w.DoLookDirection(ch, directionIndex("west")), want, "You look towards Caine's rooms.\r\n\r\n")
	})
}
