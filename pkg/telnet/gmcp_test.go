package telnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/session"
)

// telnetEvent is one unit of a parsed server byte stream: visible text, a
// GMCP message, or an IAC EOR prompt mark.
type telnetEvent struct {
	kind    string // "text", "gmcp", "eor"
	text    string
	pkg     string
	payload string
}

// parseTelnetStream splits raw server output into visible text, GMCP frames,
// and EOR marks, discarding every other telnet command. Unlike
// stripTelnetCommands it understands subnegotiation, so a GMCP payload never
// leaks into the visible text it is compared against.
func parseTelnetStream(t *testing.T, data []byte) []telnetEvent {
	t.Helper()
	var events []telnetEvent
	var text bytes.Buffer
	flush := func() {
		if text.Len() > 0 {
			events = append(events, telnetEvent{kind: "text", text: text.String()})
			text.Reset()
		}
	}
	for i := 0; i < len(data); i++ {
		if data[i] != IAC {
			text.WriteByte(data[i])
			continue
		}
		if i+1 >= len(data) {
			t.Fatalf("stream ends inside an IAC command")
		}
		cmd := data[i+1]
		switch cmd {
		case IAC:
			text.WriteByte(IAC)
			i++
		case WILL, WONT, DO, DONT:
			i += 2
		case EOR:
			flush()
			events = append(events, telnetEvent{kind: "eor"})
			i++
		case SB:
			opt := data[i+2]
			var body []byte
			j := i + 3
			for ; j+1 < len(data); j++ {
				if data[j] == IAC && data[j+1] == SE {
					break
				}
				if data[j] == IAC && data[j+1] == IAC {
					j++
				}
				body = append(body, data[j])
			}
			if j+1 >= len(data) {
				t.Fatalf("unterminated subnegotiation for option %d", opt)
			}
			i = j + 1
			if opt == OPT_GMCP {
				flush()
				pkg, payload, _ := strings.Cut(string(body), " ")
				events = append(events, telnetEvent{kind: "gmcp", pkg: pkg, payload: payload})
			}
		default:
			i++
		}
	}
	flush()
	return events
}

func visibleText(events []telnetEvent) string {
	var out strings.Builder
	for _, event := range events {
		if event.kind == "text" {
			out.WriteString(event.text)
		}
	}
	return out.String()
}

func gmcpFrames(events []telnetEvent, pkg string) []telnetEvent {
	var frames []telnetEvent
	for _, event := range events {
		if event.kind == "gmcp" && event.pkg == pkg {
			frames = append(frames, event)
		}
	}
	return frames
}

func gmcpFrame(pkg, payload string) []byte {
	return buildGMCPFrameRaw(pkg, payload)
}

// gmcpTestWorld is a four-room keep: a lit hall, a stair behind it with a
// closed door east, and a dark cellar west of the hall.
func gmcpTestWorld(t *testing.T) *game.World {
	t.Helper()
	lit := []string{"0", "0", "0", "0"}
	dark := []string{"1", "0", "0", "0"} // ROOM_DARK
	world, err := game.NewWorld(&parser.World{
		Zones: []parser.Zone{{Number: 80, Name: "The Test Keep", TopRoom: 8099}},
		Rooms: []parser.Room{
			{VNum: game.MortalStartRoom, Name: "The Gate Hall", Zone: 80, Flags: lit, Exits: map[string]parser.Exit{
				"north": {Direction: "north", ToRoom: 8005},
				"west":  {Direction: "west", ToRoom: 8006},
			}},
			{VNum: 8005, Name: "A Narrow Stair", Zone: 80, Flags: lit, Exits: map[string]parser.Exit{
				"south": {Direction: "south", ToRoom: game.MortalStartRoom},
				"east": {
					Direction: "east", ToRoom: 8007, DoorState: 1, Keywords: "door",
					ExitInfo: parser.ExitIsDoor | parser.ExitClosed,
				},
			}},
			{VNum: 8006, Name: "A Lightless Cellar", Zone: 80, Flags: dark, Exits: map[string]parser.Exit{
				"east": {Direction: "east", ToRoom: game.MortalStartRoom},
			}},
			{VNum: 8007, Name: "Behind the Door", Zone: 80, Flags: lit, Exits: map[string]parser.Exit{
				"west": {Direction: "west", ToRoom: 8005},
			}},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)
	return world
}

// gmcpWalkScript is the command sequence both harness clients send: a walk
// that succeeds, fails into a wall, fails into a closed door, returns, enters
// a dark room, leaves it, and speaks.
var gmcpWalkScript = []string{"north", "north", "east", "south", "west", "east", "say hello", "quit"}

// runTelnetScript connects to a real TCP listener, sends preamble then a
// guest name and the script, and returns everything the server wrote until
// it closed the connection.
func runTelnetScript(t *testing.T, port int, preamble []byte, name string, script []string) []byte {
	t.Helper()
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	return runTelnetScriptOn(t, conn, preamble, name, script)
}

// runTelnetScriptOn is runTelnetScript over an already-dialled connection
// (a TLS client, for instance).
func runTelnetScriptOn(t *testing.T, conn net.Conn, preamble []byte, name string, script []string) []byte {
	t.Helper()
	defer conn.Close()

	output := make(chan []byte, 1)
	go func() {
		raw, _ := io.ReadAll(conn)
		output <- raw
	}()

	input := append([]byte(nil), preamble...)
	input = append(input, name+"\r\n"...)
	for _, command := range script {
		input = append(input, command+"\r\n"...)
	}
	if _, err := conn.Write(input); err != nil {
		t.Fatal(err)
	}
	select {
	case raw := <-output:
		return raw
	case <-time.After(10 * time.Second):
		t.Fatal("server did not close the connection after quit")
		return nil
	}
}

// mudletPreamble is what Mudlet sends on connect: agree to GMCP and EOR, then
// identify itself and negotiate modules (ctelnet.cpp's default Supports set,
// plus the Comm.Channel module its starter UI adds).
func mudletPreamble() []byte {
	var p []byte
	p = append(p, IAC, DO, OPT_GMCP, IAC, DO, OPT_EOR)
	p = append(p, gmcpFrame("Core.Hello", `{"client":"Mudlet","version":"4.19.1"}`)...)
	p = append(p, gmcpFrame("Core.Supports.Set", `["Char 1","Char.Skills 1","Char.Items 1","Room 1","IRE.Rift 1","IRE.Composer 1"]`)...)
	p = append(p, gmcpFrame("Core.Supports.Add", `["Comm.Channel 1"]`)...)
	return p
}

// TestGMCPHarnessWalk is the byte-level oracle for the GMCP protocol work:
// a Mudlet-shaped client negotiates over a real socket, walks the keep, and
// the exact frames, their JSON, and their positions in the text are checked.
func TestGMCPHarnessWalk(t *testing.T) {
	manager := session.NewManager(gmcpTestWorld(t), nil)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	port := listenerEntryPort(t)
	if err := Listen(port, manager); err != nil {
		t.Fatal(err)
	}

	events := parseTelnetStream(t, runTelnetScript(t, port, mudletPreamble(), "guest_gmcp_walker", gmcpWalkScript))

	// Room.Info fires once per rendered room, in walk order: entry look, the
	// stair, back to the hall, and out of the dark cellar. The failed moves
	// render no room, and the dark cellar renders "Darkness" with no room name.
	rooms := gmcpFrames(events, "Room.Info")
	wantRooms := []string{
		`{"num":8004,"name":"The Gate Hall","area":"The Test Keep","environment":"Inside","exits":{"n":8005,"w":8006}}`,
		`{"num":8005,"name":"A Narrow Stair","area":"The Test Keep","environment":"Inside","exits":{"s":8004}}`,
		`{"num":8004,"name":"The Gate Hall","area":"The Test Keep","environment":"Inside","exits":{"n":8005,"w":8006}}`,
		`{"num":8004,"name":"The Gate Hall","area":"The Test Keep","environment":"Inside","exits":{"n":8005,"w":8006}}`,
	}
	var gotRooms []string
	for _, frame := range rooms {
		gotRooms = append(gotRooms, frame.payload)
	}
	if !reflect.DeepEqual(gotRooms, wantRooms) {
		t.Fatalf("Room.Info frames:\n got %q\nwant %q", gotRooms, wantRooms)
	}

	// Each Room.Info follows the text that rendered its room: the room name
	// appears in the text since the previous GMCP frame.
	var sinceFrame strings.Builder
	for _, event := range events {
		switch event.kind {
		case "text":
			sinceFrame.WriteString(event.text)
		case "gmcp":
			if event.pkg == "Room.Info" {
				var info struct{ Name string }
				if err := json.Unmarshal([]byte(event.payload), &info); err != nil {
					t.Fatalf("Room.Info JSON: %v", err)
				}
				if !strings.Contains(sinceFrame.String(), info.Name) {
					t.Fatalf("Room.Info %q arrived before its room text; text since last frame: %q", info.Name, sinceFrame.String())
				}
			}
			sinceFrame.Reset()
		}
	}

	// Character state: name first, then vitals and status, all before the
	// first prompt mark and before the entry room is reported.
	first := map[string]int{}
	firstEOR := -1
	for i, event := range events {
		if event.kind == "gmcp" {
			if _, seen := first[event.pkg]; !seen {
				first[event.pkg] = i
			}
		}
		if event.kind == "eor" && firstEOR < 0 {
			firstEOR = i
		}
	}
	for _, pkg := range []string{"Char.Name", "Char.Vitals", "Char.Status"} {
		index, ok := first[pkg]
		if !ok {
			t.Fatalf("no %s frame; events: %+v", pkg, events)
		}
		if index > first["Room.Info"] {
			t.Fatalf("%s arrived after the entry Room.Info", pkg)
		}
		if firstEOR >= 0 && index > firstEOR {
			t.Fatalf("%s arrived after the first prompt", pkg)
		}
	}
	vitalsShape := regexp.MustCompile(`^\{"hp":-?\d+,"maxhp":\d+,"mp":-?\d+,"maxmp":\d+,"mv":-?\d+,"maxmv":\d+\}$`)
	statusShape := regexp.MustCompile(`^\{"name":"[^"]+","level":\d+,"race":"[A-Za-z]+","class":"[A-Za-z ]+","gold":\d+\}$`)
	vitals := gmcpFrames(events, "Char.Vitals")
	for _, frame := range vitals {
		if !vitalsShape.MatchString(frame.payload) {
			t.Fatalf("Char.Vitals shape: %s", frame.payload)
		}
	}
	for _, frame := range gmcpFrames(events, "Char.Status") {
		if !statusShape.MatchString(frame.payload) {
			t.Fatalf("Char.Status shape: %s", frame.payload)
		}
	}
	// Walking north spends movement, so a second Char.Vitals reports it; a
	// vitals frame is only sent when a value changed.
	if len(vitals) < 2 {
		t.Fatalf("want a Char.Vitals update after movement, got %d frames", len(vitals))
	}
	for i := 1; i < len(vitals); i++ {
		if vitals[i].payload == vitals[i-1].payload {
			t.Fatalf("Char.Vitals repeated an unchanged payload: %s", vitals[i].payload)
		}
	}

	// Speech mirrors the sender's own echo, exactly as delivered.
	chat := gmcpFrames(events, "Comm.Channel.Text")
	if len(chat) != 1 {
		t.Fatalf("want one Comm.Channel.Text frame, got %+v", chat)
	}
	var line struct{ Channel, Talker, Text string }
	if err := json.Unmarshal([]byte(chat[0].payload), &line); err != nil {
		t.Fatal(err)
	}
	if line.Channel != "say" || line.Text != "You say 'hello'" || !strings.EqualFold(line.Talker, "guest_gmcp_walker") {
		t.Fatalf("Comm.Channel.Text = %+v", line)
	}
	if !strings.HasPrefix(chat[0].payload, `{"channel":`) {
		t.Fatalf("Comm.Channel.Text field order: %s", chat[0].payload)
	}

	// Every command prompt is marked with IAC EOR, and every mark follows a
	// prompt.
	var eors int
	for i, event := range events {
		if event.kind != "eor" {
			continue
		}
		eors++
		if i == 0 || events[i-1].kind != "text" || !strings.HasSuffix(events[i-1].text, "> ") {
			t.Fatalf("IAC EOR did not follow a prompt (event %d, previous %+v)", i, events[i-1])
		}
	}
	if want := len(gmcpWalkScript) - 1; eors < want {
		t.Fatalf("want at least %d prompt marks (one per command before quit), got %d", want, eors)
	}

	// Nothing the old agent-variable bridge sent survives on telnet.
	if frames := gmcpFrames(events, "Char.Items"); len(frames) != 0 {
		t.Fatalf("unexpected Char.Items frames: %+v", frames)
	}
}

// TestGMCPPlainClientSeesIdenticalText proves GMCP and EOR are purely out of
// band: a client that never negotiates receives no GMCP frames and no EOR
// marks, and the visible text of the same session is byte-identical to what
// the Mudlet-shaped client saw. The session pages a long help screen because
// the pager is where GMCP negotiation once changed text: it marked the
// session as a structured client, and structured clients skip paging.
func TestGMCPPlainClientSeesIdenticalText(t *testing.T) {
	world := gmcpTestWorld(t)
	var screen strings.Builder
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&screen, "Help screen line %d\r\n", i)
	}
	world.HelpScreen = screen.String()
	manager := session.NewManager(world, nil)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	port := listenerEntryPort(t)
	if err := Listen(port, manager); err != nil {
		t.Fatal(err)
	}

	const name = "guest_gmcp_twin"
	script := append([]string{"help", "q"}, gmcpWalkScript...)
	rich := parseTelnetStream(t, runTelnetScript(t, port, mudletPreamble(), name, script))
	waitSessionGone(t, manager, name)
	plain := parseTelnetStream(t, runTelnetScript(t, port, nil, name, script))

	for _, event := range plain {
		if event.kind != "text" {
			t.Fatalf("plain client received out-of-band %s event: %+v", event.kind, event)
		}
	}
	if !strings.Contains(visibleText(plain), "Help screen line 1\r\n") || strings.Contains(visibleText(plain), "Help screen line 40\r\n") {
		t.Fatalf("help screen was not paged; the pager comparison below would prove nothing: %q", visibleText(plain))
	}
	if got, want := visibleText(plain), visibleText(rich); got != want {
		t.Fatalf("plain client text differs from GMCP client text:\nplain: %q\n gmcp: %q", got, want)
	}
}

func waitSessionGone(t *testing.T, manager *session.Manager, name string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, ok := manager.GetSession(name); !ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("session %s still registered after quit", name)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestGMCPClientGUIReachesPlayerAtNamePrompt reproduces the first Mudlet
// acceptance run: Mudlet negotiates GMCP and identifies itself while the
// player is still at the name prompt, and installs the offered package from
// that moment. The offer was queued but not written until a name was
// entered, so a player who sat at the prompt never got the package.
func TestGMCPClientGUIReachesPlayerAtNamePrompt(t *testing.T) {
	session.SetGMCPClientGUI("http://localhost:8088/darkpawns.xml", "1.0.0")
	t.Cleanup(func() { session.SetGMCPClientGUI("", "") })

	manager := session.NewManager(gmcpTestWorld(t), nil)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	port := listenerEntryPort(t)
	if err := Listen(port, manager); err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	readTelnetUntil(t, conn, "By what name do you wish to be known?")
	if _, err := conn.Write(mudletPreamble()); err != nil { // no name follows
		t.Fatal(err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var raw []byte
	buf := make([]byte, 4096)
	for !bytes.Contains(raw, []byte("Client.GUI")) {
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatalf("no Client.GUI before a name was entered: %v (got %q)", err, raw)
		}
		raw = append(raw, buf[:n]...)
	}
	frames := gmcpFrames(parseTelnetStream(t, raw), "Client.GUI")
	if len(frames) != 1 || frames[0].payload != `{"version":"1.0.0","url":"http://localhost:8088/darkpawns.xml"}` {
		t.Fatalf("Client.GUI at the name prompt = %+v", frames)
	}
	if visible := visibleText(parseTelnetStream(t, raw)); visible != "" {
		t.Fatalf("the GMCP offer came with visible text: %q", visible)
	}
}

// TestPreAuthExitsStopTheWriter: a connection that leaves before logging in
// (blank name, idle timeout, hang-up) ends cleanly, with its writer stopped,
// so the listener's connection count goes back to zero.
func TestPreAuthExitsStopTheWriter(t *testing.T) {
	manager := session.NewManager(gmcpTestWorld(t), nil)
	t.Cleanup(manager.Stop)
	t.Cleanup(Stop)
	port := listenerEntryPort(t)
	if err := Listen(port, manager); err != nil {
		t.Fatal(err)
	}
	for _, input := range [][]byte{[]byte("\r\n"), nil} {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			t.Fatal(err)
		}
		readTelnetUntil(t, conn, "By what name do you wish to be known?")
		if input != nil {
			_, _ = conn.Write(input) // blank name: "Goodbye."
			_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			_, _ = io.ReadAll(conn)
		}
		_ = conn.Close() // hang-up at the prompt
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		connMu.Lock()
		open := connCount
		connMu.Unlock()
		if open == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("pre-auth exits left %d connections counted open", open)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
