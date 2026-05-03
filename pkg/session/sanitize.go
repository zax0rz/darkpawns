// Package session manages WebSocket connections and player sessions.
package session

import "strings"

// sanitizeMessage strips non-printable characters (except \r\n) from player
// messages to prevent ANSI escape codes or terminal control characters from
// messing up other players' terminals. Applied to say, tell, shout, gossip,
// emote, whisper, and other player-supplied communication.
func sanitizeMessage(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\r' || r == '\n' || r >= ' ' && r != 0x7f {
			b.WriteRune(r)
		}
	}
	return b.String()
}
