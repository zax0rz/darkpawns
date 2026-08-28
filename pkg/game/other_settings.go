package game

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// do_wimpy — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doWimpy(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	arg = strings.TrimSpace(arg)

	if arg == "" {
		if ch.WimpLevel > 0 {
			ch.SendMessage(fmt.Sprintf("Your current wimp level is %d hit points.\r\n", ch.WimpLevel))
		} else {
			ch.SendMessage("At the moment, you're not a wimp. (sure, sure...)\r\n")
		}
		return true
	}

	wimpLevel := 0
	if _, err := fmt.Sscanf(arg, "%d", &wimpLevel); err != nil {
		ch.SendMessage("That doesn't look like a number.\r\n")
		slog.Warn("wimpy parse failed", "player", ch.Name, "arg", arg, "error", err)
		return true
	}

	if wimpLevel > 0 {
		if wimpLevel < 0 {
			ch.SendMessage("Heh, heh, heh.. we are jolly funny today, eh?\r\n")
		} else if wimpLevel > ch.GetMaxHP() {
			ch.SendMessage("That doesn't make much sense, now does it?\r\n")
		} else if wimpLevel > (ch.GetMaxHP() / 3) {
			ch.SendMessage("You can't set your wimp level above one third your hit points.\r\n")
		} else {
			ch.WimpLevel = wimpLevel
			ch.SendMessage(fmt.Sprintf("Okay, you'll wimp out if you drop below %d hit points.\r\n", wimpLevel))
		}
	} else {
		ch.WimpLevel = 0
		ch.SendMessage("Okay, you'll now tough out fights to the bitter end.\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_display (do_prompt) — from act.other.c
// ---------------------------------------------------------------------------

func (w *World) doDisplay(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		ch.SendMessage("Monsters don't need displays.  Go away.\r\n")
		return true
	}

	arg = strings.TrimSpace(arg)

	if arg == "" {
		ch.SendMessage("Usage: prompt { H | M | V | T | F | all | none }\r\n")
		return true
	}

	if strings.EqualFold(arg, "on") || strings.EqualFold(arg, "all") {
		ch.SetPlrFlag(PrfDisphp, true)
		ch.SetPlrFlag(PrfDispmmana, true)
		ch.SetPlrFlag(PrfDispmove, true)
		ch.SetPlrFlag(PrfDispTank, true)
		ch.SetPlrFlag(PrfDispTarget, true)
		ch.SendMessage("Okay.\r\n") // C OK macro (config.c:92) is "Okay.\r\n"
		return true
	}

	ch.SetPlrFlag(PrfDisphp, false)
	ch.SetPlrFlag(PrfDispmmana, false)
	ch.SetPlrFlag(PrfDispmove, false)
	ch.SetPlrFlag(PrfDispTank, false)
	ch.SetPlrFlag(PrfDispTarget, false)

	// C do_display (act.other.c:1053-1055): "off" clears the bits and returns
	// silently — the `if (!str_cmp(argument,"off")) return;` skips the trailing
	// send_to_char(OK). Mirror the silent early return.
	if strings.EqualFold(arg, "off") {
		return true
	}

	for _, c := range strings.ToLower(arg) {
		switch c {
		case 'h':
			ch.SetPlrFlag(PrfDisphp, true)
		case 'f':
			ch.SetPlrFlag(PrfDispTarget, true)
		case 'm':
			ch.SetPlrFlag(PrfDispmmana, true)
		case 't':
			ch.SetPlrFlag(PrfDispTank, true)
		case 'v':
			ch.SetPlrFlag(PrfDispmove, true)
		}
	}

	ch.SendMessage("Okay.\r\n") // C OK macro (config.c:92) is "Okay.\r\n"
	return true
}

// ---------------------------------------------------------------------------
// do_gen_write — from act.other.c subcmd=SCMD_BUG/SCMD_TYPO/SCMD_IDEA/SCMD_TODO
// These are player-submitted bug/typo/idea reports stored in files.
// ---------------------------------------------------------------------------

func (w *World) doGenWrite(ch *Player, me *MobInstance, cmd string, arg string) bool {
	// Map command to file — from src/db.h BUG_FILE, TYPO_FILE, IDEA_FILE, TODO_FILE
	var filename string
	switch cmd {
	case "bug":
		filename = "misc/bugs"
	case "typo":
		filename = "misc/typos"
	case "idea":
		filename = "misc/ideas"
	case "todo":
		filename = "misc/todo"
	default:
		// C returns from the subcommand switch before checking the caller or
		// argument (act.other.c:1086-1100).
		return true
	}

	if isPlayerNPC(ch, me) {
		ch.SendMessage("Monsters can't have ideas - Go away.\r\n")
		return true
	}

	// C's skip_spaces() removes only leading whitespace. Its
	// delete_doubledollar() turns each $$ pair into a single $ before the
	// report is logged and written (act.other.c:1108-1117).
	arg = strings.TrimLeft(arg, " \t\r\n\v\f")
	arg = strings.ReplaceAll(arg, "$$", "$")
	if arg == "" {
		ch.SendMessage("That must be a mistake...\r\n")
		return true
	}

	if err := os.MkdirAll("misc", 0o755); err != nil {
		slog.Error("failed to create report directory", "type", cmd, "error", err)
		ch.SendMessage("Could not open the file.  Sorry.\r\n")
		return true
	}
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("failed to open report file", "type", cmd, "file", filename, "error", err)
		ch.SendMessage("Could not open the file.  Sorry.\r\n")
		return true
	}
	defer func() { _ = f.Close() }()

	// C formats asctime()'s month/day slice with %-8s, (%6.6s), and a
	// five-column room VNUM (act.other.c:1120-1121).
	if _, err := fmt.Fprintf(f, "%-8s (%6.6s) [%5d] %s\n", ch.Name, time.Now().Format("Jan _2"), ch.GetRoomVNum(), arg); err != nil {
		slog.Error("failed to write report", "type", cmd, "error", err)
	}

	// All four successful subcommands share C's single response.
	ch.SendMessage("Okay.  Thanks!\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_gen_tog — from act.other.c subcmd toggle commands
//
// cmd is the literal command name the player typed (e.g. "brief", "nosummon",
// "noshout") — matching src/interpreter.c's command table, where each toggle
// is its own top-level command rather than a "toggle <name>" dispatcher.
// ---------------------------------------------------------------------------

// These are process-wide C configuration toggles (config.c:202,282), rather
// than player preference bits. They are kept here because do_gen_tog is the
// only C command that mutates them. The Go server currently has no ident or
// reverse-DNS worker, but the command state and player-facing bytes remain
// part of the command surface.
var (
	nameserverIsSlow = true
	identEnabled     bool
)

func (w *World) doGenTog(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	// tog_messages[subcmd][TOG_OFF=0/TOG_ON=1] — copied verbatim from C
	// act.other.c:1147. Here each entry is keyed by the C command name and
	// stored as {TOG_ON, TOG_OFF}: index 0 is printed when the flag is being
	// switched ON (PRF_TOG_CHK returned the new state = on), index 1 when it
	// is being switched OFF. SCMD_* constants are from interpreter.h:117-136.
	toggleMessages := map[string][2]string{
		"nosummon":    {"You may now be summoned by other players.\r\n", "You are now safe from summoning by other players.\r\n"},                                                   // SCMD_NOSUMMON
		"nohassle":    {"Nohassle enabled.\r\n", "Nohassle disabled.\r\n"},                                                                                                          // SCMD_NOHASSLE
		"brief":       {"Brief mode on.\r\n", "Brief mode off.\r\n"},                                                                                                                // SCMD_BRIEF
		"compact":     {"Compact mode on.\r\n", "Compact mode off.\r\n"},                                                                                                            // SCMD_COMPACT
		"notell":      {"You are now deaf to tells.\r\n", "You can now hear tells.\r\n"},                                                                                            // SCMD_NOTELL
		"noauction":   {"You are now deaf to auctions.\r\n", "You can now hear auctions.\r\n"},                                                                                      // SCMD_NOAUCTION
		"noshout":     {"You are now deaf to shouts.\r\n", "You can now hear shouts.\r\n"},                                                                                          // SCMD_DEAF
		"nogossip":    {"You are now deaf to gossip.\r\n", "You can now hear gossip.\r\n"},                                                                                          // SCMD_NOGOSSIP
		"nograts":     {"You are now deaf to the congratulation messages.\r\n", "You can now hear the congratulation messages.\r\n"},                                                // SCMD_NOGRATZ
		"nowiz":       {"You are now deaf to the Wiz-channel.\r\n", "You can now hear the Wiz-channel.\r\n"},                                                                        // SCMD_NOWIZ
		"quest":       {"Okay, you are part of the Quest!\r\n", "You are no longer part of the Quest.\r\n"},                                                                         // SCMD_QUEST
		"roomflags":   {"You will now see the room flags.\r\n", "You will no longer see the room flags.\r\n"},                                                                       // SCMD_ROOMFLAGS
		"norepeat":    {"You will no longer have your communication repeated.\r\n", "You will now have your communication repeated.\r\n"},                                           // SCMD_NOREPEAT
		"holylight":   {"HolyLight mode on.\r\n", "HolyLight mode off.\r\n"},                                                                                                        // SCMD_HOLYLIGHT
		"nonewbie":    {"Newbie channel off.\r\n", "Newbie channel on.\r\n"},                                                                                                        // SCMD_NONEWBIE
		"noctell":     {"Clan tells are now off.\r\n", "Clan tells are now on.\r\n"},                                                                                                // SCMD_NOCTELL
		"nobroadcast": {"Broadcast channel is now off.\r\n", "Broadcast channel is now on.\r\n"},                                                                                    // SCMD_NOBROAD
		"slowns":      {"Nameserver_is_slow changed to YES; sitenames will no longer be resolved.\r\n", "Nameserver_is_slow changed to NO; IP addresses will now be resolved.\r\n"}, // SCMD_SLOWNS
		"ident":       {"Ident changed to YES;  remote usernames lookups will be attempted.\r\n", "Ident changed to NO;  remote username lookups will not be attempted.\r\n"},       // SCMD_IDENT
	}

	toggleFlags := map[string]int{
		"nosummon":    PrfSummonable,
		"nohassle":    PrfNohassle,
		"brief":       PrfBrief,
		"compact":     PrfCompact,
		"notell":      PrfNotell,
		"noauction":   PrfNoAuctions,
		"noshout":     PrfDeaf,
		"nogossip":    PrfNoGossip,
		"nograts":     PrfNoGratz,
		"nowiz":       PrfNowiz,
		"quest":       PrfQuest,
		"roomflags":   PrfRoomFlags,
		"norepeat":    PrfNoRepeat,
		"holylight":   PrfHolyLight,
		"nonewbie":    PrfNoNewbie,
		"noctell":     PrfNoCTell,
		"nobroadcast": PrfNoBroad,
	}

	msgs, ok := toggleMessages[cmd]
	if !ok {
		ch.SendMessage("Unknown toggle.\r\n")
		return true
	}

	// SCMD_NOCTELL clan gate — act.other.c:1194. A clanless player cannot toggle
	// clan tells. GET_CLAN(ch) is 0 for a fresh mortal (no clan joined).
	if cmd == "noctell" && ch.ClanID == 0 {
		ch.SendMessage("You aren't even in a clan!\r\n")
		return true
	}

	// SCMD_NOWIZ gate — act.other.c:1200. Below LVL_IMMORT and not a chosen of
	// the gods: "Huh?!?" (nowiz is also LVL_IMMORT-gated at the dispatcher, so
	// this in-handler check is faithful C defense for the rare chosen case).
	if cmd == "nowiz" && ch.GetLevel() < LVL_IMMORT && ch.GetFlags()&(1<<PlrChosen) == 0 {
		ch.SendMessage("Huh?!?\r\n")
		return true
	}

	var result bool
	switch cmd {
	case "ident":
		identEnabled = !identEnabled
		result = identEnabled
	case "slowns":
		nameserverIsSlow = !nameserverIsSlow
		result = nameserverIsSlow
	default:
		flag, ok := toggleFlags[cmd]
		if !ok {
			ch.SendMessage("Unknown toggle.\r\n")
			return true
		}

		// PRF_TOG_CHK: toggle the bit and capture the NEW state. result==1 means
		// the flag is now ON → print TOG_ON (msgs[0]); result==0 → print TOG_OFF
		// (msgs[1]).
		if ch.GetFlags()&(1<<flag) != 0 {
			ch.SetPlrFlag(flag, false)
			result = false
		} else {
			ch.SetPlrFlag(flag, true)
			result = true
		}
	}

	// SCMD_NOSUMMON sets a WAIT_STATE of PULSE_VIOLENCE*2 — act.other.c:1210.
	if cmd == "nosummon" {
		ch.SetWaitState(2)
	}

	if result {
		ch.SendMessage(msgs[0])
	} else {
		ch.SendMessage(msgs[1])
	}

	return true
}
