package game

import (
	"github.com/zax0rz/darkpawns/pkg/combat"
)

// postmaster is the special procedure for the postmaster mob.
// Intercepts MAIL, CHECK, and RECEIVE.
func postmaster(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch == nil || me == nil {
		return false
	}
	if ch.GetPosition() < combat.PosSitting {
		return false
	}

	// C mail.c:postmaster checks no_mail only for recognized mail commands
	// and returns false after the diagnostic, preserving command fallthrough.
	if mailDisabled && (cmd == "mail" || cmd == "check" || cmd == "receive") {
		ch.SendMessage("Sorry, the mail system is having technical difficulties.\r\n")
		return false
	}

	switch cmd {
	case "mail":
		w.PostmasterSendMail(ch, me, arg)
		return true
	case "check":
		w.PostmasterCheckMail(ch, me)
		return true
	case "receive":
		w.PostmasterReceiveMail(ch, me)
		return true
	}
	return false
}

func init() {
	RegisterSpec("postmaster", postmaster)
}
