package session

import "strings"

// RenderTerminalFrame applies src/comm.c:1285-1286,1376-1378 per descriptor.
// The package-level renderer decodes framing only; both terminal transports
// use this recipient-aware boundary before tracking prompts and writing bytes.
func (s *Session) RenderTerminalFrame(message []byte) (TerminalFrame, bool) {
	frame, ok := RenderTerminalFrame(message)
	if !ok || (frame.Kind != FrameText && frame.Kind != FramePrompt && frame.Kind != FrameEntryPrompt) {
		return frame, ok
	}
	color := s.charColor
	if s.manager != nil {
		s.manager.mu.RLock()
	}
	if s.isSwitched && s.switchedMob != nil {
		color = false // COLOR_ON is zero for an NPC descriptor body.
	} else if s.player != nil {
		color = whoColorEnabled(s.player)
	}
	if s.manager != nil {
		s.manager.mu.RUnlock()
	}
	if color {
		frame.Text = expandTerminalColors(frame.Text)
	}
	return frame, ok
}

// expandTerminalColors follows comm.c color_expansion. Unknown markers and a lone
// ampersand are preserved; the caller gates expansion per recipient.
func expandTerminalColors(text string) string {
	const codes = "&ndbgcrmywDBGCRMYW"
	values := [...]string{"&", "\x1b[0m", "\x1b[0;30m", "\x1b[0;34m", "\x1b[0;32m", "\x1b[0;36m", "\x1b[0;31m", "\x1b[0;35m", "\x1b[0;33m", "\x1b[0;37m", "\x1b[1;30m", "\x1b[1;34m", "\x1b[1;32m", "\x1b[1;36m", "\x1b[1;31m", "\x1b[1;35m", "\x1b[1;33m", "\x1b[1;37m"}
	var out strings.Builder
	for i := 0; i < len(text); i++ {
		if text[i] == '&' && i+1 < len(text) {
			if j := strings.IndexByte(codes, text[i+1]); j >= 0 {
				out.WriteString(values[j])
				i++
				continue
			}
		}
		out.WriteByte(text[i])
	}
	return out.String()
}
