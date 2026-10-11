package session

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/mudlog"
)

// DP-1371 proof batch 5 — the discriminators the D7 landed rows for
// notitle, squelch, freeze, thaw and skillset could not evidence
// (review.md "Concrete missing proof work", items 2-4). Each case drives the
// real registered handler on real *Session objects.
//
// Type discrimination. An observer's log level is the sum of its flags
// (LOG1 = 1, LOG2 = 2) and it receives types <= that level. So:
//   - a LOG1-only observer (level 1) receives the line  => type <= BRF,
//   - a no-flag observer (level 0) stays silent         => type >= BRF.
//
// Together: type == BRF, exactly as the call sites declare. This holds under
// both the Merc per-type flag rules and the port's monotonic delivery, so the
// proof does not depend on which of the two the port implements.
//
// Level discrimination. The declared level is MAX(LVL_GOD, invis): a
// full-flag (both bits) observer one level below stays silent, and one at
// LVL_GOD receives.
func TestWizutilMudlogTypeDiscriminators(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		drive   func(t *testing.T, actor, victim *Session)
	}{
		{
			"squelch is BRF, not NRM",
			"(GC) Squelch ON for Victim by Godactor.",
			func(t *testing.T, actor, victim *Session) {
				if err := cmdMute(actor, []string{"Victim"}); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			"freeze is BRF, not NRM",
			"(GC) Victim frozen by Godactor.",
			func(t *testing.T, actor, victim *Session) {
				if err := cmdFreeze(actor, []string{"Victim"}); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			"thaw is BRF, not NRM",
			"(GC) Victim un-frozen by Godactor.",
			func(t *testing.T, actor, victim *Session) {
				victim.player.SetPlrFlag(game.PlrFrozen, true)
				victim.player.FreezeLevel = 34
				if err := cmdThaw(actor, []string{"Victim"}); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, actor, victim, file := mudlogFixture(t)
			brief := addObserver(t, m, "BriefOnly", 40, true, false) // level 1
			off := addObserver(t, m, "OffOnly", 40, false, false)    // level 0
			tc.drive(t, actor, victim)
			// freeze/thaw also broadcast a room act to every occupant, so the
			// bracketed mudlog line is asserted by exact bytes inside the
			// drained stream rather than as the whole stream.
			if got := strings.Join(drainSessionText(t, brief), ""); !strings.Contains(got, "[ "+tc.payload+" ]\r\n") {
				t.Fatalf("LOG1-only observer = %q, want the bracketed BRF line", got)
			}
			if got := strings.Join(drainSessionText(t, off), ""); strings.Contains(got, "[ "+tc.payload+" ]\r\n") {
				t.Fatalf("no-flag observer received the line (%q): type would be OFF (file only)", got)
			}
			if !strings.Contains(file.String(), tc.payload+"\n") {
				t.Fatalf("file = %q, want %q", file.String(), tc.payload)
			}
		})
	}
}

func TestWizutilMudlogLevelDiscriminators(t *testing.T) {
	cases := []struct {
		name    string
		payload string
		drive   func(t *testing.T, actor, victim *Session)
	}{
		{
			"notitle needs LVL_GOD",
			"(GC) Notitle ON for Victim by Godactor.",
			func(t *testing.T, actor, victim *Session) {
				if err := cmdNotitle(actor, []string{"Victim"}); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			"squelch needs LVL_GOD",
			"(GC) Squelch ON for Victim by Godactor.",
			func(t *testing.T, actor, victim *Session) {
				if err := cmdMute(actor, []string{"Victim"}); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			"freeze needs LVL_GOD",
			"(GC) Victim frozen by Godactor.",
			func(t *testing.T, actor, victim *Session) {
				if err := cmdFreeze(actor, []string{"Victim"}); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			"thaw needs LVL_GOD",
			"(GC) Victim un-frozen by Godactor.",
			func(t *testing.T, actor, victim *Session) {
				victim.player.SetPlrFlag(game.PlrFrozen, true)
				victim.player.FreezeLevel = 34
				if err := cmdThaw(actor, []string{"Victim"}); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, tc := range cases {
		// C's threshold is MAX(LVL_GOD, GET_INVIS_LEV(ch)) — Go:
		// wizutilMudlogLevel, wiz_system.go:577-579. The level cell therefore
		// needs both halves: the LVL_GOD floor, and invisibility above LVL_GOD
		// raising the threshold, each with an observer one below and one exactly
		// at it. 33/34 alone would pass a producer that ignored invisibility.
		for _, sc := range []struct {
			name  string
			invis int
			below int
			at    int
		}{
			{"lv_god_floor", 0, 33, 34},
			{"invis_raises_threshold", 36, 35, 36},
		} {
			t.Run(tc.name+"/"+sc.name, func(t *testing.T) {
				m, actor, victim, _ := mudlogFixture(t)
				if sc.invis > 0 {
					actor.player.SetInvisLevel(sc.invis)
				}
				// Both bits: the observer can never be rejected for its flags, so
				// silence below the threshold isolates the level alone.
				below := addObserver(t, m, "Below", sc.below, true, true)
				at := addObserver(t, m, "At", sc.at, true, true)
				tc.drive(t, actor, victim)
				if got := strings.Join(drainSessionText(t, below), ""); strings.Contains(got, "[ "+tc.payload+" ]\r\n") {
					t.Fatalf("level-%d observer received the line (%q)", sc.below, got)
				}
				if got := strings.Join(drainSessionText(t, at), ""); !strings.Contains(got, "[ "+tc.payload+" ]\r\n") {
					t.Fatalf("level-%d observer = %q", sc.at, got)
				}
			})
		}
	}
}

// TestSkillsetMudlogTapMetadata proves the three cells the file-only proof for
// modify.c:334 could not evidence: type BRF, level -1, file TRUE. The level is
// negative, so the producer is deliberately undeliverable and no observer can
// see it; mudlog.Tap records the raw call arguments before the negative-level
// return (pkg/mudlog/feed.go), so the existing feed — not a new production seam
// — is the observer here.
//
// The ring is read with Feed.Since rather than through a live subscription: in a
// full-suite run the feed still holds a backlog, and the dispatcher replays that
// backlog to a fresh subscriber and kicks it once its 256-slot channel fills
// (feed.go:112-121), which makes a subscription a flaky observer.
//
// The ring read is bounded by a unique FIFO barrier event. A stable sequence
// number is not a barrier — the dispatcher can be descheduled with the backlog
// still queued — but the queue is one channel drained by one dispatcher in
// order (feed.go:169-213): once the barrier has been dispatched, everything
// queued before it has been, and this test's event carries a strictly higher
// sequence. Both the barrier text and the payload are unique in the process, so
// no other test's event can satisfy either search.
//
// Isolation: the ntfy provider is explicitly disabled below rather than assumed
// absent. The test is not hermetic — it starts the process feed with
// StartDefault and then reads that shared ring, so the FIFO barrier above is what
// keeps the backlog of earlier tests out of the assertion, and the unique fixture
// names keep their events out of the search.
func TestSkillsetMudlogTapMetadata(t *testing.T) {
	// StartDefault attaches a live outbound ntfy subscription whenever the
	// environment configures one (NtfyFromEnv gates on a non-empty
	// DP_MUDLOG_NTFY_URL, pkg/mudlog/ntfy.go:41-44). Clear every variable it
	// reads so the test cannot emit network traffic from an ambient config.
	// t.Setenv also marks this test non-parallel.
	for _, k := range []string{
		"DP_MUDLOG_NTFY_URL", "DP_MUDLOG_NTFY_TOKEN", "DP_MUDLOG_NTFY_LEVEL",
		"DP_MUDLOG_NTFY_TYPE", "DP_MUDLOG_NTFY_RATE",
	} {
		t.Setenv(k, "")
	}
	feed := mudlog.StartDefault(context.Background(), nil)

	before := feed.LastSeq()
	mudlog.Tap("", tapBarrier, game.MudlogNormal, 0, false)
	barrier, ok := waitForTapText(feed, before, tapBarrier, 5*time.Second)
	if !ok {
		t.Fatalf("FIFO barrier %q never dispatched: the process feed is not draining", tapBarrier)
	}

	// Unique fixture names make the payload unique to this test, so an earlier
	// skillset event from another test's fixture cannot satisfy the search.
	_ = captureMudlogFile(t) // keep the file side off stderr; it is proven elsewhere
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Tapsetactor", 1001, true)
	actor.player.SetLevel(38)
	registerTestSession(t, m, actor, "Tapsetactor")
	victim := makeTestSession(t, m, "Tapsetvictim", 1001, true)
	registerTestSession(t, m, victim, "Tapsetvictim")

	skillNum := game.FindSkillNum("kick")
	payload := fmt.Sprintf("%s changed %s's %s to %d.",
		actor.player.Name, victim.player.Name, game.SkillCatalogName(skillNum), 50)

	if err := cmdSkillset(actor, []string{"Tapsetvictim", "'kick'", "50"}); err != nil {
		t.Fatal(err)
	}

	ev, ok := waitForTapText(feed, barrier.Seq, payload, 5*time.Second)
	if !ok {
		t.Fatalf("no tap event for %q after barrier seq %d", payload, barrier.Seq)
	}
	if ev.Typ != game.MudlogBrief {
		t.Fatalf("tap type = %d, want BRF (%d)", ev.Typ, game.MudlogBrief)
	}
	if ev.Level != -1 {
		t.Fatalf("tap level = %d, want -1", ev.Level)
	}
	if !ev.ToFile {
		t.Fatal("tap to_file = false, want true")
	}
}

// tapBarrier is the FIFO barrier sentinel for TestSkillsetMudlogTapMetadata. Its
// text is unique in the process so no other test's event can be mistaken for it.
const tapBarrier = "DP-1371 batch 5 tap FIFO barrier (not a producer payload)"

// waitForTapText polls the feed ring for an event after afterSeq whose text
// matches, returning it and whether it appeared within the deadline.
func waitForTapText(feed *mudlog.Feed, afterSeq uint64, text string, timeout time.Duration) (mudlog.Event, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, ev := range feed.Since(afterSeq, 0) {
			if ev.Text == text {
				return ev, true
			}
		}
		time.Sleep(2 * time.Millisecond)
	}
	return mudlog.Event{}, false
}
