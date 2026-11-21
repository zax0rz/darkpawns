# Test audit: pkg/session — notes

Audit run: 2026-11-20, branch `kimi/test-audit-session`, base `origin/main`
(b2e884aca). Method: `docs/research/test-audit/pilot-combat/BRIEF.md` with the
three changes from PROMPT-2026-11-20 (single agent self-checks non-R marks;
fix-as-you-go; output in-repo). Continues the pkg/command audit whose summary
lives at `docs/research/test-audit/command/notes.md`.

Baseline: `go test ./pkg/session/...` passes at b2e884aca (19.6s).
Coverage baseline: **74.4%** of statements.

Ground truth: 1,537 test functions in 572 files (60,762 test lines).
`go test -list` shows 1,536 on Linux — `TestPortableWindowsEditorDelete`
(file_editor_access_windows_test.go) is GOOS-windows-tagged and counted from
source; it is in this inventory and will be audited from source. 145 tests
born before the first oracle harness (2026-07-12): 86 from 2026-05, 39 from
2026-06, 20 from early July. The bulk (1,032 of 1,537) is Sep–Oct 2026.

## Predictions (written before reading any session test)

pkg/command (audited this morning) landed at R 161 / F 9 / D 3 — 93% keepers,
weak marks concentrated pre-oracle (13.5% weak) vs later (5.1%), no merges
needed anywhere. pkg/session is bigger and more wizard-surface-heavy, with a
large pre-oracle May–June layer from the port's first months. I expect the
same shape, slightly worse pre-oracle (the May layer predates even the
coverage discipline of July) and slightly better later (most session tests
postdate the depth-test style established in August).

| Mark | Pre-oracle (145) | Later (1392) | Total (1537) |
|---|---|---|---|
| R keep | 116 | 1330 | 1446 (94.1%) |
| F fix | 18 | 40 | 58 (3.8%) |
| C merge | 6 | 12 | 18 (1.2%) |
| D delete | 5 | 10 | 15 (1.0%) |

Junk codes expected: J3 (name promises more than asserted) dominates, then
J2 (assertions that cannot fail) in the May smoke-test layer; some J5
(implementation-shape pins) around the session manager; and — because pkg/session
carries nanny/login entry flow — at least one Go/C divergence pinned by a test
(J10), the class both earlier audits each found exactly one of. Coverage:
near-unchanged overall (deletions offset by strengthened assertions).

Confidence: low on the F/C/D partition; the R share is the firm guess.

## Batches

- Batch 1–3: the 145 pre-oracle tests (~50 per batch).
- Then the 1,392 later tests in date order, ~50–100 per batch.

## Batch 1 results (first 50 pre-oracle tests, 2026-05-13..19)

Marks: R 47, F 2, C 1, D 0. Junk: J3 ×1, J4 ×1, J8 ×1.

The May layer is the port's earliest infrastructure work (manager broadcasts,
agent vars, WebSocket char creation), all authored by BRENDA69 with no PR or
trailer — agent attribution `unknown` across the batch.

F fixes (both proven by mutation, see mutations.tsv):
- `TestBuildRoomItems_KeywordDisambiguation` (J3): the second coin was given
  *different* keywords and the missing `2.coin` prefix only produced a t.Log.
  Worse, `registerObject`'s synthetic prototype uses `obj<vnum>` keywords, so
  the first coin never had "coin" either — the test passed without ever
  exercising disambiguation. Fixed: both coins share `coin gold` keywords and
  `foundSecond` is strict.
- `TestSetCommandExecFunc` (J8): the test installed its own closure over
  `CommandExecFunc` and asserted that. Fixed: exercise the production closure's
  no-session branch; green under a lookup-skipping mutation while the old
  version stayed green.

C: `TestManager_GetSession_Lookup` was a byte-for-byte duplicate of
`TestManager_GetSession` (same registration, same lookup, same assertions);
folded — remaining proof `TestManager_GetSession`.

Char-creation prompt bytes verified against C `nanny()` this session
(src/interpreter.c:1782, :1857, :1946, :1966, :2024-2025, :2070, :2085). The
`\r\n` prefix on the WS color prompt is deliberate (char_creation.go:219) —
DP-1173's flow, oracle-covered. No J10 divergences in this batch.

Coverage: 74.4% before → after (measured below in the PR body).

## Batch 2 results (pre-oracle tests 51–100, 2026-05-19..06-18)

Marks: R 48, F 0, C 2, D 0. Junk: J4 ×2.

No F marks this batch — the June layer (bucket wiring guards, look/info byte
pins, drink/eat/taste/think/insult contracts, toggle and credits tests) is
fully assertive. The two C marks fold the substring subset pins into the later
exact pins from the same file: TestAcText → TestAcTextEveryBand (DP-1551),
TestPositionText → TestPositionTextEveryPosition (DP-1553).

Highlights: TestCmdLevels pins the exact mortal row count against C's
LVL_IMMORT (src/structs.h:620); TestLook_DarkRoom pins the two-line darkness
block byte-exact; TestCmdColorPersistsLevel pins real flag mutations (a
placebo-toggle regression); TestBucketEToggleCommandsRegistered pins both the
per-name toggle wiring and the dead `gentog` alias's absence (R2).

Coverage: 74.4% → 74.4% (two subset deletions; the exact pins cover the same
lines).

## Batch 2 update (2026-11-21, post-#2923 review)

- **TestSetCommandExecFunc strengthened** (owed fix): the batch-1 version
  asserted only the no-session branch; review mutated the wired closure to
  return false after a successful lookup and every session test stayed green.
  The test now registers a session, invokes the wired `CommandExecFunc` with
  `whoami`, and requires a true return plus observable output (the live
  callers are pkg/game/combat_wire.go:745-752 and
  pkg/game/damage_stubs.go:390-395). Old green, fixed red under the same
  mutation (mutations.tsv).
- **Attribution (PF-152):** BRENDA69 was the original OpenClaw agent, a
  multi-model orchestrator (mostly glm-5-turbo with subagent delegation). All
  99 BRENDA69-authored rows in this ledger now read `OpenClaw (multi-model
  orchestrator)` with no model assigned; attribution.tsv carries an agent
  column.
## Batch 3 results (pre-oracle tests 101–145, 2026-06-18..07-11)

Marks: R 44, F 1, C 0, D 0. Junk: J3 ×1. Pre-oracle layer complete: 145/145.

The single F is the same name-overpromises shape pkg/command had: the
instakill victim-message test admitted in its own comment that capturing the
victim's line "isn't trivial" and asserted the killer's chop confirmation
instead — green when the production victim line is deleted. Fixed by
registering the victim as a session and draining its channel; the fixed test
goes red under the same mutation and incidentally proved production does
deliver "$N chops you to pieces!" (mutations.tsv).

Otherwise the June-early-July layer is strong: DP-1041 instakill gates,
DP-1045 peaceful/low-level combat gates with exact C bytes
(src/fight.c:1336-1357, verified this session), DP-954 registry MinLevel
gates, the spell-level golden table transcribed from src/class.c
init_spell_levels() (spot-checked: MAGIC_MISSILE mu:1, HEADBUTT/SLUG warrior
15/17), sip/taste/think contracts incl. the LFCR "Ok.\n\r" pin, and the
use-after-close race guards. No J10 divergences in the pre-oracle layer.

Pre-oracle totals (145 tests): R 139 (95.9%), F 3, C 3, D 0 — far better
than the 116/18/6/5 predicted; the May–June port layer was written with more
care than its age suggests.

Coverage: 74.4% → 74.4%.


## Batch 3 update (2026-11-21, PF-153 excavation)

The pre-oracle layer's OpenClaw rows now reflect the excavation's three
identities (BRENDA69, zach@karl-havoc.labz0rz.com, zach@mac-mini.local):
101 rows in this ledger attribute to `OpenClaw (multi-model orchestrator)`,
model blank (the excavation file names no model for any of these commits).
The orchestrator model is glm-5-turbo (wording corrected from glm-5-flash).
