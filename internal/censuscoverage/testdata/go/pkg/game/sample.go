//go:build ignore

// Fixture source for the census tools: parsed with go/parser, never built.
package game

// The Go port prints this same sentence. It must not make the C site look
// covered: coverage is measured against C output only, never against Go's.
func quitHint(s *Session) {
	s.sendText("You have to type quit--no less, to quit!")
}
