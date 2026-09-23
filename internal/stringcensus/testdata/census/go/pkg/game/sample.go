package game

import (
	"fmt"
	"strings"
)

// cSourced has a C source: src/sample.c send_to_char in do_quit.
func cSourced(s *Session) {
	s.SendMessage("Goodbye, friend.. Come back soon!\r\n")
}

// invented has no C source at all.
func invented(s *Session) {
	s.Send("The gate hums with a pale light.\r\n")
}

// dataDriven matches a room description in lib/world.
func dataDriven(s *Session) {
	s.Send("A cold wind blows through the ruined hall.")
}

// composed exercises the fmt.Sprintf and literal-concatenation paths.
func composed(p *Actor) {
	p.SendMessage(fmt.Sprintf("You strike %s and deal a lot of damage.", "the rat"))
	p.SendMessage("Return to the temple and QUIT to leave" +
		" the game and keep your equipment.")
}

// unresolvedLocal holds a value the census cannot follow.
func unresolvedLocal(s *Session, lines []string) {
	body := strings.Join(lines, "\r\n")
	s.SendMessage(body)
}
