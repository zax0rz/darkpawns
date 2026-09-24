package session

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/validation"
)

// The terminal: what a line-oriented client sees and how its input lines are
// routed. Telnet and the browser client both drive a session through it, so a
// player sees the same bytes whichever way they connect (R1). The transports
// keep only their framing: telnet's IAC negotiation, the browser's JSON
// envelope.

// GreetingsLogo is C's login greeting (lib/text/greetings), checked against
// the oracle's bytes by TestGreetingsLogoMatchesCFixture.
const GreetingsLogo = "\r\n\r\n" +
	"         (_____)           (_)    (_____)\r\n" +
	"   _     /  __ \\           | |    |  __ \\                            _\r\n" +
	"  ;*;   /| |  | | __ _ _ __| | __ | |__) |_ _(_      _)_ __ (___)   ;*;\r\n" +
	"   =    /| |  | |/ _` | '__| |/ / |  ___/ _` \\ \\ /\\ / / '_ \\/ __|    =\r\n" +
	" .***.  /| |__| | (_| | |  |   <  | |  | (_| |\\ V  V /| | | \\__ \\  .***.\r\n" +
	" ~~~~~  /|_____/ \\__,_|_|  |_|\\_\\ |||   \\__,_| \\_/\\_/ |_| |_|___/  ~~~~~\r\n" +
	"                                  |||\r\n" +
	"                                  |||\r\n" +
	"                                  `.'\r\n\r\n" +
	"             Based on CircleMUD 3.0 created by J. Elson and\r\n" +
	"            DikuMUD Gamma 0.0 created by K. Nyboe, T. Madsen,\r\n" +
	"                H. Staerfeldt, M. Seifert, and S. Hammer\r\n\r\n" +
	"   As of 10-17-2008 there has been a pwipe.  Enjoy your new adventures!\r\n" +
	"\r\n\r\n"

// TerminalGreeting is what a terminal client receives on connect: the
// greeting, then the name prompt. C emits one visible line break at the
// ident-to-name boundary; it is sent as a well-formed CRLF rather than C's
// legacy LFCR.
func TerminalGreeting() string {
	return NormalizeCRLF(GreetingsLogo) + NormalizeCRLF("\r\nBy what name do you wish to be known? ")
}

// TerminalFrameKind says how a transport frames rendered output.
type TerminalFrameKind int

const (
	// FrameText is ordinary output, written as is.
	FrameText TerminalFrameKind = iota
	// FramePrompt is the command prompt (telnet marks it with IAC EOR).
	FramePrompt
	// FrameEntryPrompt is a login or character-creation prompt. Secret asks
	// the client to stop echoing input (a password).
	FrameEntryPrompt
	// FrameGMCP is out-of-band data for clients that asked for it.
	FrameGMCP
)

// TerminalFrame is one queued session message rendered for a terminal.
type TerminalFrame struct {
	Kind        TerminalFrameKind
	Text        string
	Secret      bool
	GMCPPackage string
	GMCPPayload string
}

// RenderTerminalFrame renders one message from a session's send channel as
// the bytes a terminal shows. ok is false for messages a terminal does not
// display (structured state and vars).
func RenderTerminalFrame(msg []byte) (TerminalFrame, bool) {
	var sm ServerMessage
	if err := json.Unmarshal(msg, &sm); err != nil {
		return TerminalFrame{}, false
	}
	data, _ := sm.Data.(map[string]interface{})
	switch sm.Type {
	case MsgEvent:
		text, ok := data["text"].(string)
		if !ok {
			return TerminalFrame{}, false
		}
		// A raw event carries control bytes that must reach the terminal
		// untouched (C's infobar), with no line ending added.
		if eventType, _ := data["type"].(string); eventType == "raw" {
			return TerminalFrame{Kind: FrameText, Text: text}, true
		}
		return TerminalFrame{Kind: FrameText, Text: NormalizeCRLF(ensureLineEnded(text))}, true
	case MsgError:
		message, ok := data["message"].(string)
		if !ok {
			return TerminalFrame{}, false
		}
		return TerminalFrame{Kind: FrameText, Text: NormalizeCRLF(fmt.Sprintf("\r\n!! %s\r\n", message))}, true
	case MsgText:
		text, ok := data["text"].(string)
		if !ok {
			return TerminalFrame{}, false
		}
		return TerminalFrame{Kind: FrameText, Text: NormalizeCRLF(text + "\r\n")}, true
	case MsgPrompt:
		// The prompt travels through the send channel so it is written only
		// after the command's queued output (C: comm.c:643-648 flush output,
		// then prompt).
		prompt := "> "
		if text, ok := data["text"].(string); ok && text != "" {
			prompt = text
		}
		if raw, _ := data["raw"].(bool); raw {
			return TerminalFrame{Kind: FramePrompt, Text: prompt}, true
		}
		return TerminalFrame{Kind: FramePrompt, Text: NormalizeCRLF(prompt)}, true
	case MsgCharCreate:
		// Nanny prompts are already byte-exact C strings. MENU contains mixed
		// LF/CR ordering on purpose, so they bypass the newline normalizer.
		secret, _ := data["secret"].(bool)
		prompt, _ := data["prompt"].(string)
		return TerminalFrame{Kind: FrameEntryPrompt, Text: prompt, Secret: secret}, true
	case "gmcp":
		pkg, _ := data["package"].(string)
		payload, _ := data["json"].(string)
		if pkg == "" {
			return TerminalFrame{}, false
		}
		return TerminalFrame{Kind: FrameGMCP, GMCPPackage: pkg, GMCPPayload: payload}, true
	case MsgState, MsgVars, MsgTokenRefresh:
		// Structured client data and credentials; the text stream carries
		// everything a terminal shows. (A rotated token used to fall through
		// to the default case and print its JSON envelope at the player.)
		return TerminalFrame{}, false
	default:
		return TerminalFrame{Kind: FrameText, Text: NormalizeCRLF(fmt.Sprintf("[%s]\r\n", string(msg)))}, true
	}
}

// ensureLineEnded appends CRLF only to text that carries no line ending at all.
// A trailing '\r' already ends the line: C's historical LFCR pair ("\n\r") ends
// most handler output, and appending another CRLF after it injects a blank line
// the oracle never wrote whenever one command emits two messages (do_string's
// WARNING/Ok pair is the first vehicle that exposed it — modify.c:632,765).
func ensureLineEnded(text string) string {
	if strings.HasSuffix(text, "\n") || strings.HasSuffix(text, "\r") {
		return text
	}
	return text + "\r\n"
}

// NormalizeCRLF converts any mix of "\r\n", C's historical "\n\r", lone "\r",
// and lone "\n" line endings into canonical "\r\n". Much of the game's text
// (MOTD, room and help files) is stored with bare "\n"; a raw terminal treats
// a lone LF as line-feed-only and staircases the output. LFCR is a single C
// line ending, not two lines; preserving that pair matters for handlers such
// as do_skillset that build output from mixed-order strings. Idempotent.
func NormalizeCRLF(s string) string {
	if !strings.ContainsAny(s, "\r\n") {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n\r", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

// TerminalNamed reports whether the terminal has accepted a name and handed
// the login to the nanny. Telnet reads the name line with its pre-login
// reader until then.
func (s *Session) TerminalNamed() bool {
	return s.terminalNamed
}

// TerminalLine routes one complete input line from a terminal client to the
// state that owns it, as C's nanny and command interpreter do. It reports
// false when the connection should close.
func (s *Session) TerminalLine(rawLine string) bool {
	line := strings.TrimSpace(rawLine)
	if !s.terminalNamed {
		return s.terminalName(line)
	}

	// The oracle harness control is intercepted before player/session command
	// handling so the trigger itself consumes no command RNG, wait state, or
	// activity state. Only the pumped heartbeats may draw.
	if s.HandleClockControl(line) {
		return true
	}

	// DP-928: any inbound traffic proves the connection is alive, so the
	// linkdead reaper sees it.
	s.OnInboundActivity()

	switch {
	case s.IsCharCreating() || s.IsMenuActive():
		// A blank line is meaningful during character creation (the "PRESS
		// RETURN" step), so it is forwarded, not swallowed.
		s.terminalError(s.terminalCharInput(rawLine))
	case s.IsPaging():
		// Output pager (DP-1195): while paging, every input line, including a
		// bare RETURN (next page), goes to the pager, never the interpreter
		// (C: comm.c:617 showstr_count routing). It sits above the empty-line
		// refresh so RETURN reaches the pager; SendPrompt then selects C's
		// pager or ordinary playing prompt from the resulting state.
		s.terminalError(s.terminalPagerInput(line))
		if !s.SendClosed() {
			s.SendPrompt()
		}
	case s.IsTextEditing():
		// CON_TEDIT owns every complete line, including an empty one: C's
		// string_add appends it to d->str.
		s.terminalError(s.terminalCommand("", nil, rawLine))
		if !s.SendClosed() {
			s.SendPrompt()
		}
	case s.IsRoomEditing() || s.IsMeditEditing() || s.IsOeditEditing() || s.IsZoneEditing() || s.IsSeditEditing():
		// CON_REDIT, CON_MEDIT, CON_OEDIT, CON_ZEDIT and CON_SEDIT own every
		// complete line, including a bare RETURN at a numeric prompt
		// ("Field must be numerical, try again : ").
		s.terminalError(s.terminalCommand("", nil, rawLine))
		if !s.SendClosed() {
			s.SendPrompt()
		}
	case line == "":
		// RETURN with no command refreshes the prompt, queued after any
		// pending output (C: comm.c:643-648).
		s.SendPrompt()
	default:
		// C tokenization (interpreter.c:883-907): a non-letter first character
		// is a one-character command, no separating space needed ("'hello").
		cmdWord, cmdArgs := SplitCommandInput(line)
		s.terminalError(s.terminalCommand(cmdWord, cmdArgs, rawLine))
		// Queued after the command so its output drains first, then the
		// prompt, as C flushes before prompting.
		if !s.SendClosed() {
			s.SendPrompt()
		}
	}
	return !s.SendClosed()
}

// terminalName is C's CON_GET_NAME: the connection stays open until it
// receives a usable name, and an empty line ends it.
func (s *Session) terminalName(name string) bool {
	if name == "" {
		s.sendTerminalText("\r\nGoodbye.\r\n")
		return false
	}
	if !strings.HasPrefix(strings.ToLower(name), "guest") &&
		(!validation.IsValidPlayerName(name) || !game.ValidNameNoActive(name)) {
		s.sendTerminalText("Invalid name, please try another.\r\nName: ")
		return true
	}
	s.terminalNamed = true
	data, err := json.Marshal(LoginData{PlayerName: name})
	if err == nil {
		err = s.sendClientMessage(MsgLogin, data)
	}
	if err != nil {
		s.sendTerminalText(fmt.Sprintf("\r\nLogin failed: %v\r\n", err))
		return false
	}
	// handleLogin rejects bad credentials without an error: it has queued the
	// reason and closed the send channel.
	return !s.SendClosed()
}

func (s *Session) terminalCharInput(choice string) error {
	data, err := json.Marshal(map[string]interface{}{"choice": choice})
	if err != nil {
		return err
	}
	return s.sendClientMessage(MsgCharInput, data)
}

func (s *Session) terminalPagerInput(line string) error {
	data, err := json.Marshal(map[string]interface{}{"choice": line})
	if err != nil {
		return err
	}
	return s.sendClientMessage(MsgPagerInput, data)
}

func (s *Session) terminalCommand(cmd string, args []string, rawLine string) error {
	data, err := json.Marshal(CommandData{
		Command: cmd,
		Args:    args,
		RawLine: rawLine,
		RawArgs: CommandArgumentText(rawLine),
	})
	if err != nil {
		return err
	}
	return s.sendClientMessage(MsgCommand, data)
}

func (s *Session) sendClientMessage(msgType string, data json.RawMessage) error {
	msg, err := json.Marshal(ClientMessage{Type: msgType, Data: data})
	if err != nil {
		return err
	}
	return s.handleMessage(msg)
}

// terminalError shows a routing error the way telnet always has.
func (s *Session) terminalError(err error) {
	if err != nil {
		s.sendTerminalText(fmt.Sprintf("Error: %v\r\n", err))
	}
}

// sendTerminalText queues transport text that is not game output (the name
// dialogue, routing errors), normalized like all terminal text.
func (s *Session) sendTerminalText(text string) {
	s.sendRawEvent(NormalizeCRLF(text))
}
