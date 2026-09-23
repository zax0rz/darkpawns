# 2026-09-23: Mudlet playtest, harness parity, and the judge as the product

Field notes from one long session (Zach and Claude, with Luna and DeepSeek on
delegated briefs). Observations and leads, not results: each claim that should
reach a paper has a row in `EVIDENCE_LEDGER.tsv` pointing at its artifact.

## 1. The harness is the product (framing for the port-fidelity paper)

Zach found the oracle method by accident (Anthropic's migration write-up, the
Bun Zig-to-Rust port) and credits it with saving the project. Reading the Bun,
Anthropic and Checkly (Node to Go) write-ups side by side, all three treat the
**judge** as the thing being built and the code as what the judge certifies:
Bun ran its language-agnostic test suite against both implementations, Checkly
compared byte-level golden files, and the Anthropic guide says a judge must pass
the original *and fail on deliberately broken code*. Dark Pawns' C oracle is the
strongest form (the original runs), and the day's findings are all failures of
the judge's reach, not of its comparison: see 3 to 5. This is the AI slop-squash
thesis seen from the other side: the model produced the plausible paraphrase;
the judge, where it reached, squashed it; where it didn't reach, the slop
survived until a human playtest.

Sources: https://claude.com/blog/ai-code-migration ,
https://bun.com/blog/bun-in-rust ,
https://www.checklyhq.com/blog/agentic-rewrite-nodejs-to-go/ .
Follow-on work: `docs/fidelity/HARNESS_PARITY.md` (PR #1585).

## 2. The mimic that became a couch (canonical paraphrase, caught statically)

C's `SPECIAL(couch)` (`src/spec_procs2.c:282`) is a joke with a twist: examine
the couch and it is a **mimic**: "Starved and needing food to make more
pillows, the mimic attacks you!" The Go port reads "...the couch attacks you!"
The couch was not invented; the mimic was paraphrased away, and the punchline
with it. Natural-sounding, passes a skim, wrong (R1). No census scenario ever
triggered it. The string census (PR #1587, built by DeepSeek from a Claude
brief) flagged it on its first run as a Go string with no C source. Same shape
as `yank` (PF-002). The same function also drifted in visibility (`CAN_SEE_OBJ`
and exact name vs keyword substring) and audience (act TO_ROOM vs a room
message that also reaches the actor).

## 3. Harness parity: four lifecycle bugs, one afternoon, zero census signal

A human Mudlet playtest found, in one afternoon: logout loses gold, sex,
alignment, skills and every preference (DP-1314); login lands in the room you
were saved in, Limbo included (DP-1310); idle voids a *connected* player after
60 s where C takes ~9 minutes, and typing never resets the timer (DP-1311);
`quit` drops the connection instead of returning to the menu (DP-1312). The
970-scenario census was green throughout. Cause: every scenario was one
unbroken connection, and the harness ran the Go server against an unreachable
database, so Go never persisted anything and nothing across a logout could
diverge. After adding a `<RELOGIN>` step and a throwaway store (PR #1584), three
scenarios reproduced all four against the C oracle immediately. Lesson matching
Checkly's retry-queue incident: a proof counts only for conditions the harness
reproduces.

## 4. Coincidence-green at the edge of a scenario

`quit.safe-logout` was oracle-green for months: C's and Go's goodbye matched.
The scenario ended at the goodbye; C's menu comes after it. The proof was real
and its conclusion (quit is faithful) was false. A scenario's last step is a
boundary where divergence can hide.

## 5. Silent state corruption only visible through persistence work

Fixing DP-1314 required making flags saveable, which exposed: PRF bits stored
at 20+n collided with PLR bits (brief = PLR_REMORT, compact = PLR_EXTRACT), so
a player with `compact` on read as extracted and the lair dragon never breathed
on them; and PLR flags had two separate stores, so lycanthropy set one while
`transform` read the other and an infected player could never transform. The
JSON save also wrote AC/hitroll/damroll totals (double-counted on reload) and
wrote skills without reading them back. None was visible to single-connection
scenarios (PR #1586).

## 6. The port "fixed" C's typo

C: "You are incapacitated an will slowly die" (`src/fight.c:1572`). Go: "...and
will slowly die". The typo is the tell that a line was rewritten rather than
ported; under R1 the typo is the game. Found by the string census.

## 7. A test double that encoded the implementer's assumption

The Mudlet package's Lua stub set `speedWalkFrom`/`speedWalkTo` because the
package read them; real Mudlet 5.0.1 sets them only in custom pathfinding mode
and passes `speedWalkDir` otherwise. The stub agreed with the code and both
disagreed with the platform, so double-click speedwalk failed live while tests
passed. Fixed by deriving stub behaviour from the tagged Mudlet source
(package 1.1.6). Same failure class as coincidence-green, one layer out.

## 8. "It feels like a copy"

Zach's summary of why the game felt off. Every lifecycle and visibility bug in
3 to 6 was found by a human noticing something felt wrong (idle too fast, quit
without a menu, dark city streets, a colour setting not sticking), not by the
census. The human role at this stage is exploratory play that produces bug
*classes*, which the harness then encodes.

## 9. Delegation with briefs and adversarial gating

Cheaper models (GPT-6 Luna, DeepSeek) took fully specified briefs (ground truth
pasted in, decisions made, a check that must fail without the fix); Claude
reviewed from the C and the diff, not the implementer's account. DeepSeek's
string census was sound and its three headline findings verified against C; it
missed CI's gosec job because the ground rules did not name it (fixed in the
rules). Mirrors Bun's cheap-implementer, strong-adversarial-reviewer split.

## 10. The census was wall-bound, not CPU-bound

Load stayed near 2 on 16 cores at 4 parallel jobs: scenarios mostly wait for
quiescence. 4 jobs took over 3 h; 12 took 41 min; 24 took 20.8 min (PR #1586), each
over all 970 scenarios.

## 11. A scenario named "combat-death" in which nobody dies

`combat-death` was written to drive a fight "to the death" and prove the
death path. The character's default wimpy level makes them flee at low hit
points first, identically on both servers, so the scenario has always been
green and has never exercised a player death. Coincidence-green by name: the
title stated an intent the transcript never reached. Found while building a
death-to-menu vehicle for DP-1312 (the replacement uses a death-trap fixture,
PR #1596).

## 12. The oracle's own environment manufactured a divergence

With the C oracle's lib copy missing `plrobjs/`, `Crash_rentsave` silently
failed to open its file, so C never rented a quitter's objects and
`extract_char` dropped them on the temple floor. That looked like a faithful C
behaviour for the port to copy ("C leaves the newbie kit on the floor"). It
was the harness. After the harness created C's per-player file trees, C
rented and restored the kit (PR #1596). The judge needs parity with
production on **both** sides: the Go side's dead database (PF-008) hid bugs;
the C side's missing directories invented one.

## 13. Gating the first delegated batch (Codex and DeepSeek)

Six delegated PRs (#1590 to #1595) were gated by reading each diff against its
C call site rather than the implementer's summary. Three findings:

- **A directive written as a comment.** `gsay-act-depth` declared `# keep-ansi`,
  which the parser treats as a comment, so the scenario never compared colour
  bytes and its `gsay.act-complete-color` case was green without proof. With
  the directive in `[fixture]`, the raw comparison showed the speaker's colour
  reset placed before the line ending (fixed) and two prompt divergences
  belonging to DP-1307/DP-1308. Coincidence-green by a one-character typo.
- **"Focused checks passed" missed a neighbour.** Routing gsay through act()
  made a sleeping group member see "Someone" (the port's canSee treats sleep
  as blindness; C's CAN_SEE has no AWAKE test, DP-1295). The PR's own new
  scenario was green; the existing `gsay-depth` regressed. Only the census
  over everything catches a change's effect on scenarios it didn't write.
- **Undefined behaviour in the oracle.** C's `do_auto` builds its listing with
  `sprintf(buf, "%s...", buf)`; the Linux oracle prints nothing. The delegate
  matched the oracle and deleted the listing. The project's precedent
  (make_prompt, exits) is the opposite: the oracle is patched to the intended
  semantics the original game showed players, and the port keeps them. The
  rulebook had never stated this; it is the gap that let a careful delegate
  follow the oracle's letter over its purpose.

## 14. First census coverage map

The first full census with coverage dumps (976 scenarios, 8,672 C blocks, PR
#1599 batch) makes the C server print 2,321 of 4,902 fixed player-facing text
segments (47.3%); 1,336 C sites are too short to verify (under 8 characters)
and are excluded. It is an upper bound: short common phrases ("the south") can
match text a room description printed. The gaps line up with the day's bugs:
`weather.c` and `gate.c` 0% (the invented dusk broadcasts, DP-1316),
`handler.c` 8%, `objsave.c` 4% and `db.c` 5% (the lifecycle and persistence
layer), `fight.c` 23%, `spells.c` 13.5%, `magic.c` 19%, `shop.c` 17%, and
`mail.c`, `dream.c`, `scripts.c`, `tattoo.c` 0%. A green census had meant
"everything the scenarios reach agrees"; this is the first measure of how much
that is.
