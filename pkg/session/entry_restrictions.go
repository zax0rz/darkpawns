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

// entryBanLevel evaluates the current list against retained connection identity,
// as C isbanned(d->host) does at each nanny boundary. Synthetic callers that
// provide only SetBanLevel retain their explicit admission snapshot.
func (s *Session) entryBanLevel() int {
	if len(s.banHosts) == 0 {
		return s.banLevel
	}
	level := game.BanNot
	if bm := s.manager.GetBanManager(); bm != nil {
		for _, host := range s.banHosts {
			level = max(level, bm.IsBanned(host))
		}
	}
	return level
}

// src/interpreter.c:1825-1831: site ban precedes wizlock at confirmation.
func (s *Session) refuseNewSiteBan() bool {
	if s.entryBanLevel() < game.BanNew {
		return false
	}
	s.refuseEntry("Sorry, new characters are not allowed from your site!\r\n",
		fmt.Sprintf("Request for new char %s denied from [%s] (siteban)", s.charName, s.RemoteIP()))
	return true
}

// src/interpreter.c:1896-1905: select needs the saved PLR_SITEOK bit.
func (s *Session) refuseReturningSiteBan(name string, raw []byte) bool {
	if s.entryBanLevel() != game.BanSelect || game.CharacterDataSiteOK(raw) {
		return false
	}
	s.sendRawEvent("\r\n")
	s.refuseEntry("Sorry, this char has not been cleared for login from your site!\r\n",
		fmt.Sprintf("Connection attempt for %s denied from %s", name, s.RemoteIP()))
	return true
}
