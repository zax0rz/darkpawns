package session

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// Static info-text commands.
// Source: src/act.informative.c ACMD(do_gen_ps) — pages a static text file
// from lib/text/* verbatim.  Files are cached on first access (matching C's
// file_to_string_alloc boot caching) and served through PageString.

// cachedText holds the 2010 static text files, loaded lazily on first access.
// Source: src/db.c file_to_string_alloc boot caching.
var (
	cachedText = map[string]string{}
	cacheMu    sync.RWMutex
)

// sendCachedText serves a static text file through the pager, loading and
// caching it from lib/text/ on first access. Matches C's
// page_string(ch->desc, <cached>, 0) pattern.
func sendCachedText(s *Session, filename string) {
	cacheMu.RLock()
	text, ok := cachedText[filename]
	cacheMu.RUnlock()
	if !ok {
		// LibTextDir is derived from the -world flag at boot (World.LibTextDir);
		// a CWD-relative path here is dead under the oracle harness, which runs
		// the server in a scratch dir (the help-system gate's lesson, #440).
		data, err := os.ReadFile(filepath.Join(s.manager.world.LibTextDir, filename))
		if err != nil {
			s.Send("That information is not available right now.")
			return
		}
		text = string(data)
		cacheMu.Lock()
		cachedText[filename] = text
		cacheMu.Unlock()
	}
	PageString(s, text)
}

// cmdCredits shows who built the game. Source: do_gen_ps SCMD_CREDITS
func cmdCredits(s *Session, args []string) error {
	sendCachedText(s, "credits")
	return nil
}

// cmdNews shows current game news. Source: do_gen_ps SCMD_NEWS
func cmdNews(s *Session, args []string) error {
	sendCachedText(s, "news")
	return nil
}

// cmdInfoText shows game information. Source: do_gen_ps SCMD_INFO
func cmdInfoText(s *Session, args []string) error {
	sendCachedText(s, "info")
	return nil
}

// cmdWizlist shows the wizard list. Source: do_gen_ps SCMD_WIZLIST
func cmdWizlist(s *Session, args []string) error {
	sendCachedText(s, "wizlist")
	return nil
}

// cmdImmlist shows the immortal list. Source: do_gen_ps SCMD_IMMLIST
func cmdImmlist(s *Session, args []string) error {
	sendCachedText(s, "immlist")
	return nil
}

// cmdHandbook shows the immortal handbook. Source: do_gen_ps SCMD_HANDBOOK (LVL_IMMORT)
func cmdHandbook(s *Session, args []string) error {
	sendCachedText(s, "handbook")
	return nil
}

// cmdPolicy shows the game's policies. Source: do_gen_ps SCMD_POLICIES
func cmdPolicy(s *Session, args []string) error {
	sendCachedText(s, "policies")
	return nil
}

// cmdFuture shows planned future content. Source: do_gen_ps SCMD_FUTURE
func cmdFuture(s *Session, args []string) error {
	sendCachedText(s, "future")
	return nil
}

// cmdMotd shows the message of the day. Source: do_gen_ps SCMD_MOTD
func cmdMotd(s *Session, args []string) error {
	sendCachedText(s, "motd")
	return nil
}

// cmdImotd shows the immortal message of the day. Source: do_gen_ps SCMD_IMOTD (LVL_IMMORT)
func cmdImotd(s *Session, args []string) error {
	sendCachedText(s, "imotd")
	return nil
}

// cmdClear sends a terminal clear-screen escape. Source: do_gen_ps SCMD_CLEAR
func cmdClear(s *Session, args []string) error {
	s.Send("\033[H\033[J")
	return nil
}

// cmdVersion shows the game version, plus build info for immortals.
// Source: do_gen_ps SCMD_VERSION — C prints the SVN revision and compile
// timestamp; this port has no equivalent build-time tracking, so the Go
// runtime version is shown instead as the closest honest substitute.
func cmdVersion(s *Session, args []string) error {
	s.Send("Dark Pawns 2.3-")
	if s.player != nil && s.player.Level >= LVL_IMMORT {
		s.Send(fmt.Sprintf("Built with: %s", runtime.Version()))
	}
	return nil
}

// cmdWhoami sends back the player's own name. Source: do_gen_ps SCMD_WHOAMI
func cmdWhoami(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	s.Send(s.player.Name)
	return nil
}

// cmdPlayers lists all registered players. Source: do_gen_ps SCMD_PLAYER_LIST (LVL_GRGOD)
func cmdPlayers(s *Session, args []string) error {
	names, err := s.manager.db.ListPlayerNames()
	if err != nil {
		s.Send("That information is not available right now.")
		return nil
	}
	var buf strings.Builder
	buf.WriteString("A list of registered players:\r\n")
	count := 0
	for _, name := range names {
		fmt.Fprintf(&buf, "%-20.20s", name)
		count++
		if count == 3 {
			count = 0
			buf.WriteString("\r\n")
		}
	}
	buf.WriteString("\r\n")
	s.Send(buf.String())
	return nil
}
