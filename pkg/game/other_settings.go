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
		ch.SendMessage("Ok.\r\n")
		return true
	}

	ch.SetPlrFlag(PrfDisphp, false)
	ch.SetPlrFlag(PrfDispmmana, false)
	ch.SetPlrFlag(PrfDispmove, false)
	ch.SetPlrFlag(PrfDispTank, false)
	ch.SetPlrFlag(PrfDispTarget, false)

	if !strings.EqualFold(arg, "off") {
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
	}

	ch.SendMessage("Ok.\r\n")
	return true
}

// ---------------------------------------------------------------------------
// do_gen_write — from act.other.c subcmd=SCMD_BUG/SCMD_TYPO/SCMD_IDEA/SCMD_TODO
// These are player-submitted bug/typo/idea reports stored in files.
// The original writes to ~lib/%s.ideas, ~lib/%s.bugs, etc.
// ---------------------------------------------------------------------------

func (w *World) doGenWrite(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	if arg == "" {
		switch cmd {
		case "bug":
			ch.SendMessage("Describe the bug you've discovered?\r\n")
		case "typo":
			ch.SendMessage("What typo did you find?\r\n")
		case "idea":
			ch.SendMessage("What is your idea?\r\n")
		case "todo":
			ch.SendMessage("What would you like to see added?\r\n")
		default:
			ch.SendMessage("Report what?\r\n")
		}
		return true
	}

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
		ch.SendMessage("Reported. Thanks!")
		return true
	}

	// Append report to file
	if err := os.MkdirAll("misc", 0755); err == nil {
		f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			defer f.Close()
			if _, err := fmt.Fprintf(f, "%s [%s]: %s\n", ch.Name, time.Now().Format("2006-01-02 15:04"), arg); err != nil {
				slog.Error("failed to write report", "type", cmd, "error", err)
			}
		}
	}

	switch cmd {
	case "bug":
		ch.SendMessage("Bug reported. Thanks!\r\n")
	case "typo":
		ch.SendMessage("Typo reported. Thanks!\r\n")
	case "idea":
		ch.SendMessage("Idea noted. Thanks!\r\n")
	case "todo":
		ch.SendMessage("Todo noted. Thanks!\r\n")
	default:
		ch.SendMessage("Reported. Thanks!\r\n")
	}
	return true
}

// ---------------------------------------------------------------------------
// do_gen_tog — from act.other.c subcmd toggle commands
// ---------------------------------------------------------------------------

func (w *World) doGenTog(ch *Player, me *MobInstance, cmd string, arg string) bool {
	if isPlayerNPC(ch, me) {
		return true
	}

	toggleMessages := map[string][2]string{
		"summon":    {"You are now summonable by others.\r\n", "You are no longer summonable.\r\n"},
		"nohassle":  {"You are now immune to annoying players.\r\n", "You may now be hassled again.\r\n"},
		"brief":     {"Brief mode on.\r\n", "Brief mode off.\r\n"},
		"compact":   {"Compact mode on.\r\n", "Compact mode off.\r\n"},
		"notell":    {"You are now deaf to tells.\r\n", "You may now receive tells again.\r\n"},
		"noauction": {"You are now deaf to auctions.\r\n", "You can now hear auctions again.\r\n"},
		"deaf":      {"You are now deaf to all shouts.\r\n", "You can now hear shouts again.\r\n"},
		"nogossip":  {"You are now deaf to gossip.\r\n", "You can now hear gossip again.\r\n"},
		"nogratz":   {"You will no longer see grat messages.\r\n", "You will now see grat messages again.\r\n"},
		"nowiz":     {"You are now deaf to the WizChannel.\r\n", "You can now hear the WizChannel again.\r\n"},
		"quest":     {"You will now see quest announcements.\r\n", "You will no longer see quest announcements.\r\n"},
		"roomflags": {"Room flags on.\r\n", "Room flags off.\r\n"},
		"norepeat":  {"No repeat mode on.\r\n", "No repeat mode off.\r\n"},
		"holylight": {"Holy light mode on.\r\n", "Holy light mode off.\r\n"},
		"autocxits": {"Auto exits on.\r\n", "Auto exits off.\r\n"},
		"npcident":  {"NPC identify mode on.\r\n", "NPC identify mode off.\r\n"},
		"nonewbie":  {"You will now see newbie chat.\r\n", "You will no longer see newbie chat.\r\n"},
		"noctell":   {"You are now deaf to clan tells.\r\n", "You can now hear clan tells again.\r\n"},
		"nobroad":   {"You are now deaf to broadcasts.\r\n", "You can now hear broadcasts again.\r\n"},
	}

	toggleFlags := map[string]int{
		"summon":    PrfSummonable,
		"nohassle":  PrfNohassle,
		"brief":     PrfBrief,
		"compact":   PrfCompact,
		"notell":    PrfNotell,
		"noauction": PrfNoAuctions,
		"deaf":      PrfDeaf,
		"nogossip":  PrfNoGossip,
		"nogratz":   PrfNoGratz,
		"nowiz":     PrfNowiz,
		"quest":     PrfQuest,
		"roomflags": PrfRoomFlags,
		"norepeat":  PrfNoRepeat,
		"holylight": PrfHolyLight,
		"autocxits": PrfAutoexit,
		"npcident":  PrfAutoexit, // reuse autoexit flag for ident
		"nonewbie":  PrfNoNewbie,
		"noctell":   PrfNoCTell,
		"nobroad":   PrfNoBroad,
	}

	// Map user-facing cmd to internal toggle key
	cmdMap := map[string]string{
		"summon":    "summon",
		"nohassle":  "nohassle",
		"brief":     "brief",
		"compact":   "compact",
		"notell":    "notell",
		"noauction": "noauction",
		"deaf":      "deaf",
		"nogossip":  "nogossip",
		"nogratz":   "nogratz",
		"nowiz":     "nowiz",
		"quest":     "quest",
		"roomflags": "roomflags",
		"norepeat":  "norepeat",
		"holylight": "holylight",
		"autocxits": "autocxits",
		"npcident":  "npcident",
		"nonewbie":  "nonewbie",
		"noctell":   "noctell",
		"nobroad":   "nobroad",
	}

	key, ok := cmdMap[cmd]
	if !ok {
		ch.SendMessage("Unknown toggle.\r\n")
		return true
	}

	flag, ok := toggleFlags[key]
	if !ok {
		ch.SendMessage("Unknown toggle.\r\n")
		return true
	}
	msgs, ok := toggleMessages[key]
	if !ok {
		ch.SendMessage("Unknown toggle.\r\n")
		return true
	}

	// Special checks
	if key == "nowiz" && ch.GetLevel() < LVL_IMMORT {
		ch.SendMessage("You are not holy enough to use that feature.\r\n")
		return true
	}

	// "noctell": clan check skipped — clan field not yet implemented

	if ch.GetFlags()&(1<<flag) != 0 {
		ch.SetPlrFlag(flag, false)
		ch.SendMessage(msgs[1])
	} else {
		ch.SetPlrFlag(flag, true)
		ch.SendMessage(msgs[0])
	}

	return true
}
