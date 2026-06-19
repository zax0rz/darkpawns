package session

import (
	"fmt"
	"os"
	"runtime"
)

// Static info-text commands.
// Source: src/act.informative.c ACMD(do_gen_ps) — pages a static text file
// from lib/world/text/* verbatim. This Go port sends the whole file at once
// rather than paginating it.

// sendTextFile reads a static text file under lib/world/text/ and sends its
// contents to the player. The server process's working directory is the
// repo root (see game.LoadHelpFiles's "lib/text/help" call), so relative
// paths here are safe and match the existing convention.
func sendTextFile(s *Session, filename string) error {
	data, err := os.ReadFile("lib/world/text/" + filename)
	if err != nil {
		s.Send("That information is not available right now.")
		return nil
	}
	s.Send(string(data))
	return nil
}

// cmdCredits shows who built the game. Source: do_gen_ps SCMD_CREDITS
func cmdCredits(s *Session, args []string) error {
	return sendTextFile(s, "credits")
}

// cmdNews shows current game news. Source: do_gen_ps SCMD_NEWS
func cmdNews(s *Session, args []string) error {
	return sendTextFile(s, "news")
}

// cmdPolicy shows the game's policies. Source: do_gen_ps SCMD_POLICIES
func cmdPolicy(s *Session, args []string) error {
	return sendTextFile(s, "policies")
}

// cmdHandbook shows the immortal handbook. Source: do_gen_ps SCMD_HANDBOOK (LVL_IMMORT)
func cmdHandbook(s *Session, args []string) error {
	return sendTextFile(s, "handbook")
}

// cmdFuture shows planned future content. Source: do_gen_ps SCMD_FUTURE
func cmdFuture(s *Session, args []string) error {
	return sendTextFile(s, "future")
}

// cmdWhoami sends back the player's own name. Source: do_gen_ps SCMD_WHOAMI
func cmdWhoami(s *Session, args []string) error {
	if s.player == nil {
		return nil
	}
	s.Send(s.player.Name)
	return nil
}

// cmdVersion shows the game version, plus build info for immortals.
// Source: do_gen_ps SCMD_VERSION — C prints the SVN revision and compile
// timestamp; this port has no equivalent build-time tracking, so the Go
// runtime version is shown instead as the closest honest substitute.
func cmdVersion(s *Session, args []string) error {
	s.Send("Dark Pawns 2.3")
	if s.player != nil && s.player.Level >= LVL_IMMORT {
		s.Send(fmt.Sprintf("Built with: %s", runtime.Version()))
	}
	return nil
}
