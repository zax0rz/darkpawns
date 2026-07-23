package oraclediff

import (
	"net"
	"strings"
	"testing"
	"time"
)

func TestNormalizeTier1Rules(t *testing.T) {
	raw := "\x1b[31mRoom\x1b[0m\r\n" +
		"  Str: excellent     Dex: poor          Int: average   \r\n" +
		"  Wis: decent        Con: good          Cha: bad\r\n" +
		"22H 100M 83V >   \r\nPlayer HP: 22/22\r\nPlayers online: 7\r\n" +
		"As of 10-17-2008 there has been a pwipe.\r\n**** Dark Pawns 3.0 beta\r\nDescription  \r\n"
	want := "Room\n<ROLLED_STATS>\nPlayer <VITALS>\nDescription\n"
	if got := Normalize(raw); got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestNormalizeCanonicalizesCRLFAndLFCRAsSingleNewlines(t *testing.T) {
	raw := "crlf\r\nlfcr\n\rlone-cr\rlone-lf\n"
	want := "crlf\nlfcr\nlone-cr\nlone-lf\n"
	if got := Normalize(raw); got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestNormalizeDropsPromptFramingButPreservesTextGreaterThan(t *testing.T) {
	raw := ">\r\n> Huh?!?\r\n>    Indented room text\r\nDoor > north\r\n22H 100M 83V >\r\n"
	want := "Huh?!?\n   Indented room text\nDoor > north\n"
	if got := Normalize(raw); got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestNormalizeCanonicalizesAutoExitIndentation(t *testing.T) {
	raw := "Temple Infirmary\r\n [ Exits: north ]\r\n"
	want := "Temple Infirmary\n[ Exits: north ]\n"
	if got := Normalize(raw); got != want {
		t.Fatalf("Normalize() = %q, want %q", got, want)
	}
}

func TestUnifiedDiff(t *testing.T) {
	diff := UnifiedDiff("c", "go", "same\nold\ntail\n", "same\nnew\ntail\n")
	for _, want := range []string{"--- c", "+++ go", "@@ -1,3 +1,3 @@", "-old", "+new"} {
		if !strings.Contains(diff, want) {
			t.Errorf("diff missing %q:\n%s", want, diff)
		}
	}
}

func TestParseScenarioSections(t *testing.T) {
	input := `
# header comment
[setup:oracle]
name
Y
pass

[setup:port]
name
y
pass
pass
Y

[probe]
look
look sign
quit
`
	sc, err := ParseScenario("test", strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.SetupOracle) != 3 || sc.SetupOracle[2] != "pass" {
		t.Fatalf("SetupOracle = %#v", sc.SetupOracle)
	}
	if len(sc.SetupPort) != 5 || sc.SetupPort[0] != "name" || sc.SetupPort[4] != "Y" {
		t.Fatalf("SetupPort = %#v", sc.SetupPort)
	}
	if len(sc.Probe) != 3 || sc.Probe[1] != "look sign" {
		t.Fatalf("Probe = %#v", sc.Probe)
	}
}

func TestParseScenarioEnter(t *testing.T) {
	sc, err := ParseScenario("test", strings.NewReader("[setup:oracle]\nname\n<ENTER>\n[probe]\nlook\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(sc.SetupOracle) != 2 || sc.SetupOracle[1] != "" {
		t.Fatalf("SetupOracle = %#v", sc.SetupOracle)
	}
	if len(sc.Probe) != 1 || sc.Probe[0] != "look" {
		t.Fatalf("Probe = %#v", sc.Probe)
	}
}

func TestParseScenarioPeersAndFixtures(t *testing.T) {
	input := `
[setup:oracle]
actor
[setup:port]
actor
[setup:oracle:victim]
victim
[setup:port:victim]
victim
[fixture]
inert-scroll 8038
quiet-mobs
spawn-mob 18306 1 8162 80
strip-mob-script 18306
[probe]
recite scroll
`
	sc, err := ParseScenario("act", strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	peer := sc.Peers["victim"]
	if peer == nil || len(peer.SetupOracle) != 1 || peer.SetupOracle[0] != "victim" || len(peer.SetupPort) != 1 {
		t.Fatalf("victim peer = %#v", peer)
	}
	if len(sc.Fixtures) != 1 || sc.Fixtures[0] != (ObjectFixture{ObjectVNum: 8038}) {
		t.Fatalf("Fixtures = %#v", sc.Fixtures)
	}
	if len(sc.MobFixtures) != 1 || sc.MobFixtures[0] != (MobFixture{MobVNum: 18306, MaxExisting: 1, RoomVNum: 8162, ZoneNumber: 80}) {
		t.Fatalf("MobFixtures = %#v", sc.MobFixtures)
	}
	if !sc.QuietAllMobs {
		t.Fatal("QuietAllMobs = false, want true")
	}
	if len(sc.ScriptlessMobIDs) != 1 || sc.ScriptlessMobIDs[0] != 18306 {
		t.Fatalf("ScriptlessMobIDs = %#v", sc.ScriptlessMobIDs)
	}
}

func TestParseScenarioRejectsInvalidFixture(t *testing.T) {
	_, err := ParseScenario("bad", strings.NewReader("[fixture]\ninert-scroll nope\n[probe]\nlook\n"))
	if err == nil || !strings.Contains(err.Error(), "invalid fixture") {
		t.Fatalf("expected invalid fixture error, got %v", err)
	}
}

func TestRunAudienceProbeCapturesEachRecipientInStableOrder(t *testing.T) {
	actor := &scriptedConn{outputs: []string{"actor-one", "actor-two"}}
	observer := &scriptedConn{outputs: []string{"observer-one", "observer-two"}}
	victim := &scriptedConn{outputs: []string{"victim-one", "victim-two"}}

	blocks, err := RunAudienceProbe(actor, map[string]Conn{
		"victim":   victim,
		"observer": observer,
	}, []string{"one", "two"}, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	want := []AudienceProbeBlock{
		{Command: "one", Audience: "actor", Output: "actor-one"},
		{Command: "one", Audience: "observer", Output: "observer-one"},
		{Command: "one", Audience: "victim", Output: "victim-one"},
		{Command: "two", Audience: "actor", Output: "actor-two"},
		{Command: "two", Audience: "observer", Output: "observer-two"},
		{Command: "two", Audience: "victim", Output: "victim-two"},
	}
	if len(blocks) != len(want) {
		t.Fatalf("len(blocks) = %d, want %d: %#v", len(blocks), len(want), blocks)
	}
	for i := range want {
		if blocks[i] != want[i] {
			t.Errorf("blocks[%d] = %#v, want %#v", i, blocks[i], want[i])
		}
	}
	if got := strings.Join(actor.sent, ","); got != "one,two" {
		t.Errorf("actor commands = %q, want one,two", got)
	}
}

type scriptedConn struct {
	outputs []string
	sent    []string
}

func (c *scriptedConn) Send(line string) error {
	c.sent = append(c.sent, line)
	return nil
}

func (c *scriptedConn) ReadUntilQuiescent(time.Duration) (string, error) {
	if len(c.outputs) == 0 {
		return "", nil
	}
	out := c.outputs[0]
	c.outputs = c.outputs[1:]
	return out, nil
}

func (c *scriptedConn) Close() error { return nil }

func TestPumpPulsesSendsReservedControlAndReturnsOutput(t *testing.T) {
	conn := &scriptedConn{outputs: []string{"birth dream\r\n"}}
	got, err := PumpPulses(conn, 40, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got != "birth dream\r\n" {
		t.Fatalf("output = %q", got)
	}
	if len(conn.sent) != 1 || conn.sent[0] != "~dpclock pulse 40" {
		t.Fatalf("sent = %v", conn.sent)
	}
}

func TestRunSetupAndSettlePumpsImmediatelyAfterEnteringGame(t *testing.T) {
	conn := &scriptedConn{outputs: []string{
		"greeting",
		"name prompt",
		"menu",
		"staging room",
		"birth dream",
		"recalled room",
	}}
	got, err := RunSetupAndSettle(conn, []string{"name", "", "1", "recall"}, 40, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if got != "greetingname promptmenustaging roombirth dreamrecalled room" {
		t.Fatalf("transcript = %q", got)
	}
	if sent := strings.Join(conn.sent, ","); sent != "name,,1,~dpclock pulse 40,recall" {
		t.Fatalf("sent = %q", sent)
	}
}

func TestRunSetupAndSettleRejectsAmbiguousEnterGameChoice(t *testing.T) {
	conn := &scriptedConn{}
	_, err := RunSetupAndSettle(conn, []string{"1", "other", "1"}, 40, time.Millisecond)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous setup error = %v", err)
	}
}

func TestParseScenarioUnknownSection(t *testing.T) {
	_, err := ParseScenario("test", strings.NewReader("[setup:unknown]\nlook\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown section") {
		t.Fatalf("expected unknown section error, got %v", err)
	}
}

func TestParseScenarioNoProbe(t *testing.T) {
	_, err := ParseScenario("test", strings.NewReader("[setup:oracle]\nname\n"))
	if err == nil || !strings.Contains(err.Error(), "no [probe] steps") {
		t.Fatalf("expected no probe error, got %v", err)
	}
}

func TestParseScenarioCreationSection(t *testing.T) {
	// [creation:*] feeds the ordinary setup streams but flips DiffSetup, and a
	// creation-only scenario needs no [probe].
	sc, err := ParseScenario("test", strings.NewReader(
		"[creation:oracle]\nname\nY\n[creation:port]\nname\ny\n",
	))
	if err != nil {
		t.Fatalf("creation-only scenario should parse without a probe, got %v", err)
	}
	if !sc.DiffSetup {
		t.Fatal("DiffSetup should be set by a [creation:*] section")
	}
	if len(sc.SetupOracle) != 2 || sc.SetupOracle[1] != "Y" {
		t.Fatalf("creation keystrokes should populate SetupOracle, got %v", sc.SetupOracle)
	}
	if len(sc.SetupPort) != 2 || sc.SetupPort[1] != "y" {
		t.Fatalf("creation keystrokes should populate SetupPort, got %v", sc.SetupPort)
	}
}

func TestReportNoDivergence(t *testing.T) {
	diffs := []BlockDiff{
		{Command: "look", Oracle: "same\n", Go: "same\n"},
		{Command: "quit", Oracle: "bye\n", Go: "bye\n"},
	}
	r := Report(ReportMeta{Scenario: "s", OracleAddr: "a", GoAddr: "b", Seed: "1"}, diffs)
	if !strings.Contains(r, "no normalized divergence") {
		t.Fatalf("expected no divergence report, got:\n%s", r)
	}
	if strings.Count(r, "DP_SEED=1") != 2 {
		t.Fatalf("expected shared seed for both processes, got:\n%s", r)
	}
}

func TestReadUntilQuiescentEscapedIAC(t *testing.T) {
	server, client := net.Pipe()
	t.Cleanup(func() { server.Close() })
	t.Cleanup(func() { client.Close() })

	tcpConn := NewTCPConn(client)

	// Write synchronously: net.Pipe writes rendezvous with reads, so this
	// returns once the reader has consumed all five bytes. Do NOT close the
	// server here — net.Pipe reads after a peer close return ErrClosedPipe
	// (not EOF, unlike real TCP), which races the quiescence deadline and
	// flaked under -race (PR #438 CI). The deadline ends the read instead;
	// t.Cleanup closes both ends afterward.
	go func() {
		server.Write([]byte{255, 255, 65, 66, 67})
	}()

	out, err := tcpConn.ReadUntilQuiescent(250 * time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	// The IAC IAC sequence should produce a literal 0xFF byte before ABC.
	want := "\xffABC"
	if out != want {
		t.Fatalf("ReadUntilQuiescent() = %q, want %q", out, want)
	}
}

func TestReportBlockDivergence(t *testing.T) {
	diffs := []BlockDiff{
		{Command: "look", Oracle: "room\n", Go: "room\n"},
		{Command: "look sign", Oracle: "old\n", Go: "new\n", Diff: UnifiedDiff("c-oracle", "go-port", "old\n", "new\n")},
	}
	r := Report(ReportMeta{Scenario: "s", OracleAddr: "a", GoAddr: "b", Seed: "1"}, diffs)
	if !strings.Contains(r, "normalized divergence detected") {
		t.Fatalf("expected divergence report, got:\n%s", r)
	}
	if !strings.Contains(r, "[look sign]") {
		t.Fatalf("expected command label, got:\n%s", r)
	}
}
