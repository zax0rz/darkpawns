package session

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// src/interpreter.c:1841-1847: any nonzero restriction rejects new players
// at name confirmation, before password, RNG, persistence or world admission.
func (s *Session) refuseNewWizlock() bool {
	if s.manager.WizlockLevel() == 0 {
		return false
	}
	s.refuseEntry("Sorry, new players can't be created at the moment.\r\n",
		fmt.Sprintf("Request for new char %s denied from [%s] (wizlock)", s.charName, s.RemoteIP()))
	return true
}

// src/interpreter.c:1906-1914: compare the saved character's level only
// after password success, before duplicate takeover, MOTD or admission.
func (s *Session) refuseReturningWizlock(name string, level int) bool {
	if level >= s.manager.WizlockLevel() {
		return false
	}
	s.sendRawEvent("\r\n") // echo_on(), src/interpreter.c:1871; src/comm.c:954-967.
	s.refuseEntry("The game is temporarily restricted.. try again later.\r\n",
		fmt.Sprintf("Request for login denied for %s [%s] (wizlock)", name, s.RemoteIP()))
	return true
}

func (s *Session) refuseEntry(message, log string) {
	s.sendCharCreatePrompt("closing", message, nil)
	game.MudLog(log, game.MudlogNormal, game.LVL_GOD, true)
	s.CloseSend()
}
