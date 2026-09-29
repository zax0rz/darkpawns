package oraclediff

import (
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"
)

// closeRecordingConn is a scriptedConn whose Close is reported on a shared
// event log, so the crash ordering tests can see when the transport was
// disposed relative to the kill.
type closeRecordingConn struct {
	scriptedConn
	events *[]string
}

func (c *closeRecordingConn) Close() error {
	*c.events = append(*c.events, "close")
	return nil
}

// newCrashRecordingConn builds a CrashConn whose kill, restart, dial and
// login callbacks append to a shared event log. The kill callback records
// the signal and the process-exit wait separately, which is what lets the
// tests assert the exit precedes the restart.
func newCrashRecordingConn(events *[]string, second Conn) *CrashConn {
	dials := 0
	return NewCrashConn(&closeRecordingConn{scriptedConn: scriptedConn{outputs: []string{""}}, events: events},
		func() (Conn, error) {
			*events = append(*events, "dial")
			dials++
			return second, nil
		},
		func(c Conn) (string, error) {
			*events = append(*events, "login")
			return RunSetup(c, []string{"Tester", "pass", "", "1"}, time.Millisecond)
		},
		func() error {
			*events = append(*events, "sigkill", "exit-wait")
			return nil
		},
		func() error {
			*events = append(*events, "restart")
			return nil
		})
}

// TestCrashStepOrderingIsKillFirst pins the step's mandatory order: SIGKILL
// (and nothing before it), the process-exit wait, the transport close, the
// engine restart, then the dial and the relogin. No settle callback exists
// to call, and no SIGINT may appear.
func TestCrashStepOrderingIsKillFirst(t *testing.T) {
	events := &[]string{}
	second := &scriptedConn{outputs: []string{
		"greeting\r\n", "Password: ", "PRESS RETURN", "menu", "Temple Square\r\n", "You have 12 gold.\r\n",
	}}
	actor := newCrashRecordingConn(events, second)

	blocks, err := RunAudienceProbe(actor, nil, []string{"look", CrashStep, "gold"}, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 3 || blocks[1].Command != CrashStep {
		t.Fatalf("blocks = %#v, want the crash step in place", blocks)
	}
	if blocks[1].Output != "greeting\r\nPassword: PRESS RETURNmenuTemple Square\r\n" {
		t.Fatalf("crash block = %#v", blocks[1])
	}
	if blocks[2].Output != "You have 12 gold.\r\n" {
		t.Fatalf("post-crash block = %#v", blocks[2])
	}

	want := []string{"sigkill", "exit-wait", "close", "restart", "dial", "login"}
	if !slices.Equal(*events, want) {
		t.Fatalf("crash event order = %v, want %v (SIGKILL first, exit before restart, close after the kill, no settle, no SIGINT)", *events, want)
	}
}

// TestCrashStepKillsOnlyItsEngine models the two probe passes: each engine
// has its own CrashConn whose kill closure is bound to that engine alone
// (exactly how main.go wires the oracle and Go sides). Crashing one pass's
// connection must leave the other engine's recorder untouched.
func TestCrashStepKillsOnlyItsEngine(t *testing.T) {
	engineA := &[]string{}
	engineB := &[]string{}
	login := func(c Conn) (string, error) {
		return RunSetup(c, []string{"Tester", "pass", "", "1"}, time.Millisecond)
	}
	restartFor := func(log *[]string) func() error {
		return func() error { *log = append(*log, "restart"); return nil }
	}
	killFor := func(log *[]string) func() error {
		return func() error { *log = append(*log, "sigkill", "exit-wait"); return nil }
	}
	freshConn := func() *scriptedConn {
		return &scriptedConn{outputs: []string{"greeting\r\n", "Password: ", "PRESS RETURN", "menu", "ok\r\n"}}
	}
	connA := NewCrashConn(&closeRecordingConn{scriptedConn: scriptedConn{outputs: []string{""}}, events: engineA},
		func() (Conn, error) { return freshConn(), nil }, login, killFor(engineA), restartFor(engineA))
	connB := NewCrashConn(&closeRecordingConn{scriptedConn: scriptedConn{outputs: []string{""}}, events: engineB},
		func() (Conn, error) { return freshConn(), nil }, login, killFor(engineB), restartFor(engineB))

	if _, err := RunAudienceProbe(connA, nil, []string{CrashStep}, time.Millisecond); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(*engineA, []string{"sigkill", "exit-wait", "close", "restart"}) {
		t.Fatalf("crashed engine's events = %v", *engineA)
	}
	if len(*engineB) != 0 {
		t.Fatalf("the other engine was signalled from this pass: %v", *engineB)
	}

	// The other pass drives its own crash the same way.
	if _, err := RunAudienceProbe(connB, nil, []string{CrashStep}, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(*engineB, []string{"sigkill", "exit-wait", "close", "restart"}) {
		t.Fatalf("second engine's events = %v", *engineB)
	}
	if !slices.Equal(*engineA, []string{"sigkill", "exit-wait", "close", "restart"}) {
		t.Fatalf("first engine was signalled from the second pass: %v", *engineA)
	}
}

// TestCrashStepPropagatesKillFailure: a crash whose kill fails is a harness
// failure, never a silently short scenario.
func TestCrashStepPropagatesKillFailure(t *testing.T) {
	errKillFailed := errors.New("kill failed")
	actor := NewCrashConn(&scriptedConn{outputs: []string{""}},
		func() (Conn, error) { return &scriptedConn{}, nil },
		func(c Conn) (string, error) { return "", nil },
		func() error { return errKillFailed },
		func() error { return nil })
	if _, err := RunAudienceProbe(actor, nil, []string{CrashStep}, time.Millisecond); !errors.Is(err, errKillFailed) {
		t.Fatalf("kill failure error = %v, want %v", err, errKillFailed)
	}
}

// TestRunAudienceProbeCrashNeedsCrasher: the step on a connection that
// cannot crash its engine is a clear error, as <RESTART> is.
func TestRunAudienceProbeCrashNeedsCrasher(t *testing.T) {
	actor := &scriptedConn{outputs: []string{""}}
	if _, err := RunAudienceProbe(actor, nil, []string{CrashStep}, time.Millisecond); !errors.Is(err, errNoCrash) {
		t.Fatalf("error = %v, want %v", err, errNoCrash)
	}
}

// TestParseScenarioCrashGuards pins the structural guards: a crash needs
// both login line sets, may appear once, cannot be combined with <RESTART>
// or with passive peers, and cannot be sent by a named probe actor.
func TestParseScenarioCrashGuards(t *testing.T) {
	login := "[relogin:oracle]\nA\n[relogin:port]\nA\n"
	valid := "[setup:oracle]\nA\n[setup:port]\nA\n" + login + "[probe]\nquit\n<CRASH>\nlook\n"
	sc, err := ParseScenario("valid", strings.NewReader(valid))
	if err != nil {
		t.Fatalf("a well-formed crash scenario was rejected: %v", err)
	}
	if len(sc.Probe) != 3 || sc.Probe[1] != CrashStep {
		t.Fatalf("probe = %v, want the step in place", sc.Probe)
	}

	cases := map[string]string{
		"no relogin sections": "[setup:oracle]\nA\n[setup:port]\nA\n[probe]\nquit\n<CRASH>\n",
		"two crashes":         "[setup:oracle]\nA\n[setup:port]\nA\n" + login + "[probe]\nquit\n<CRASH>\nlook\n<CRASH>\n",
		"with a peer":         "[setup:oracle]\nA\n[setup:port]\nA\n[setup:oracle:peer]\nA\n[setup:port:peer]\nA\n" + login + "[probe]\nquit\n<CRASH>\n",
		"with a probe actor":  "[setup:oracle]\nA\n[setup:port]\nA\n[setup:oracle:peer]\nA\n[setup:port:peer]\n" + login + "[probe:peer]\nquit\n<CRASH>\n",
		"restart and crash":   "[setup:oracle]\nA\n[setup:port]\nA\n" + login + "[probe]\nquit\n<RESTART>\nlook\n<CRASH>\n",
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseScenario(name, strings.NewReader(src)); err == nil {
				t.Fatalf("an invalid crash scenario (%s) was accepted", name)
			}
		})
	}
}

// TestCrashStepDropsDeadTransportOutput: the killed transport's output is
// not read into the crash block — reading it would take a quiescence window
// before the kill, and a crashed server's buffered bytes are lost with the
// process, as for a real player.
func TestCrashStepDropsDeadTransportOutput(t *testing.T) {
	events := &[]string{}
	first := &closeRecordingConn{
		scriptedConn: scriptedConn{outputs: []string{"some queued output\r\n"}},
		events:       events,
	}
	second := &scriptedConn{outputs: []string{"greeting\r\n", "Password: ", "PRESS RETURN", "menu", "ok\r\n"}}
	actor := NewCrashConn(first,
		func() (Conn, error) { *events = append(*events, "dial"); return second, nil },
		func(c Conn) (string, error) {
			*events = append(*events, "login")
			return RunSetup(c, []string{"T", "p", "", "1"}, time.Millisecond)
		},
		func() error { *events = append(*events, "sigkill", "exit-wait"); return nil },
		func() error { *events = append(*events, "restart"); return nil })

	blocks, err := RunAudienceProbe(actor, nil, []string{CrashStep}, time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if blocks[0].Output != "greeting\r\nPassword: PRESS RETURNmenuok\r\n" {
		t.Fatalf("crash block = %#v, want only the relogin transcript", blocks[0])
	}
	if slices.Contains(*events, "read") {
		t.Fatalf("the dead transport was read: %v", *events)
	}
	_ = io.EOF
}
