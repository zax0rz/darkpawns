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
