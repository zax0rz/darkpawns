# BRIEF 2026-07-13 — Foundation 2 (F0b): one authoritative command table + C-faithful dispatcher gates

**Executor:** ChatGPT / frontier. **Read the refactor plan first for the big picture:**
`docs/research/drafts/2026-07-13-session-game-refactor-plan.md` — this brief is **Foundation F0b** in
that plan (§3 "Foundations first", §4 step 2). It is **independent of F0a/act()** — pure
dispatch/gating — so it can land in parallel. Claude reviews; **every behavior/wording change is
proven by an oracle red→green run** (`docs/research/drafts/2026-07-13-c-oracle-differential-testing.md`
+ `[[darkpawns-oracle-proof-gate]]` — player-facing output is first-class fidelity, equal to the
mechanical effect).
**Branch:** `refactor/command-table-gates` off current `main`. **One PR** (or a tight series).

---

## The headline: the gates aren't broken — the DATA is, and the ORDER + MESSAGES are

C's command dispatch is a single ordered table, `cmd_info[]` (`src/interpreter.c:310+`), one row per
command:

```c
struct command_info { char *command; byte minimum_position; void (*command_pointer)();
                      sh_int minimum_level; int subcmd; };  // interpreter.h:60
// examples (interpreter.c:310+):
{ "cutthroat", POS_FIGHTING, do_cutthroat, 1,          0 },
{ "date"     , POS_DEAD    , do_date     , LVL_IMMORT,  SCMD_DATE },
{ "dc"       , POS_DEAD    , do_dc       , LVL_IMMORT+1, 0 },
{ "deposit"  , POS_STANDING, do_not_here , 1,          0 },
```

Go has the same shape already — `command.Entry{MinLevel, MinPosition, Aliases}` (`pkg/command/registry.go:24`),
populated by `cmdRegistry.Register(name, handler, help, minLevel, minPosition, aliases...)`
(`registry.go:53`). **The machinery is fine. Three things are wrong:**

### Problem 1 — the table values are mostly `0,0` (the real leak)
**131 of 273** `cmdRegistry.Register(...)` calls in `pkg/session/*` pass `minLevel=0, minPosition=0`
— i.e. no level gate, no position gate. That is how `summon`/`hcontrol` leaked to mortals
(DP-1108/1109, patched ad-hoc in PR #295) and how position gates vanished. **The fix is to populate
every row from C's `cmd_info[]`, not to change the guard.**

> ⚠️ **Do not "fix" the `>0` guards — they are already equivalent to C at the boundary.** In C the
> level test is `GET_LEVEL(ch) >= minimum_level` (interpreter.c:912) and the position test is
> `GET_POS(ch) < minimum_position` (interpreter.c:923). At value `0`: `level >= 0` is always true and
> `pos < 0` is never true — so a `0` gate means "no restriction," exactly what Go's
> `if MinLevel > 0 { … }` / `if MinPosition > 0 { … }` (commands.go:619/610) already does. The guards
> are not the bug; the **data** is. (You *may* drop the `>0` guards to mirror C's unconditional
> compares 1:1 — cosmetic, low-risk — but that is not where the fidelity gap lives.)

### Problem 2 — the gate ORDER is reversed
Go checks **position first** (commands.go:610) then **level** (commands.go:619). C checks **level
first** — it's folded into command *lookup*: the search loop only matches a row you have the level for,
so an over-level command falls through to `Huh?!?` and is **indistinguishable from an unknown command**
(interpreter.c:910-914). Only *after* a successful level-qualified match does C test frozen →
not-implemented → switched-NPC-immortal → **position** (interpreter.c:916-947).

Consequence of the reversal: a **sleeping mortal** who types an immortal command gets, in C, `Huh?!?`
(the command is hidden by level) — but in Go gets the *position*-fail message, **revealing the command
exists**. F0b must reorder to C's cascade: **level/lookup gate → (frozen) → (not-impl) →
(switched-NPC) → position.**

### Problem 3 — the position-fail messages are invented (wrong wording)
`positionFailMessage` (`pkg/session/commands.go:18`) shares **none** of C's strings and collapses
cases C keeps distinct. C's exact per-position text (interpreter.c:923-947):

| Position (C const / value) | C string (verbatim) | Go current (WRONG) |
| --- | --- | --- |
| `POS_DEAD` (0) | `Lie still; you are DEAD!!! :-(` | "You are dead! You can't do that." |
| `POS_INCAP` (2) | `You are in a pretty bad shape, unable to do anything!` | "You are incapacitated and cannot do that." |
| `POS_MORTALLYW` (1) | `You are in a pretty bad shape, unable to do anything!` *(same as INCAP)* | "You are mortally wounded and cannot do that." *(split — wrong)* |
| `POS_STUNNED` (3) | `All you can do right now is think about the stars!` | "You are stunned and cannot do that." |
| `POS_SLEEPING` (4) | `In your dreams, or what?` | "You are asleep and cannot do that!" |
| `POS_RESTING` (5) | `Nah... You feel too relaxed to do that..` | "You need to stand up first." |
| `POS_SITTING` (6) | `Maybe you should get on your feet first?` | *(falls to generic default)* |
| `POS_FIGHTING` (7) | `No way!  You're fighting for your life!` | *(falls to generic default)* |

(`POS_STANDING` (8) is the top; nothing gates above it.) Note C groups `INCAP`+`MORTALLYW` under one
string; Go must too. Copy C's strings **verbatim** (including the `:-(`, the double `..`, the double
space in "No way!  You're"). This is first-class fidelity per the proof gate.

## Deliverables (bounded — this is a foundation, not the whole migration)

1. **One authoritative command table, sourced from C.** Cross-reference **every** Go command against
   C's `cmd_info[]` (`src/interpreter.c:310`–the `{ "\n" … }` sentinel). For each command set
   `minLevel` / `minPosition` to C's value using the named constants (add Go equivalents where
   missing): positions `POS_DEAD..POS_STANDING = 0..8` (`combat.PosDead..PosStanding` already exist —
   movement already uses `combat.PosStanding` correctly, so follow that pattern); levels
   from C `structs.h:610-620`: `LVL_IMMORT=31, LVL_GOD=34, LVL_HIGOD=36, LVL_GRGOD=38, LVL_IMPL=40`
   (note `LVL_IMMORT+1`, `LVL_GRGOD`, etc. appear as literal table values). Map C's level constants to
   Go's level scheme — confirm Go's immortal thresholds match C's numeric values (this is a known trap; if Go's
   scale differs, document the mapping table you used).
   - Keep the registration ergonomic, but make the **values auditable** — the point is a single
     source of truth that a test can diff against C (see proof). A comment citing the C row (e.g.
     `// C: { "hcontrol", POS_DEAD, do_hcontrol, LVL_GRGOD, 0 } interpreter.c:NNN`) on non-trivial
     gates is worth it.
   - **Go-only commands** (WS/engine commands with no C equivalent — there will be some): pick a
     defensible gate, default to the safe side (mortal-usable informational → `0, <sensible pos>`;
     admin/debug → immortal), and **document each** in the PR with a one-line rationale. Do not
     silently leave them `0,0`.
   - **Re-land the PR #295 ad-hoc gates on the table:** `summon` (DP-1108/O42) and `hcontrol`
     (DP-1109/O43) were patched inline; move them onto the authoritative table so there's one source
     of truth (their handlers/defense-in-depth checks stay).

2. **Fix the dispatcher cascade order** (`pkg/session/commands.go:~609-624`) to match C
   (interpreter.c:910-947): **level gate first** (over-level ⇒ `Huh?!?`, command hidden — same reply
   as unknown), **then** position gate. Preserve the existing `Huh?!?\r\n` reply for over-level
   (already correct — matches C interpreter.c:915). If Go models frozen/switched-NPC/not-implemented
   states, order them as C does between level and position; if it doesn't model them, skip (note it) —
   don't invent them.

3. **Replace `positionFailMessage` strings with C's verbatim** (table above), including the
   INCAP+MORTALLYW merge and the SITTING/FIGHTING cases. This is the wording half of the proof.

**Out of scope (say so if you touch the edge):**
- **Command abbreviation / prefix-match order.** C's lookup is prefix `strncmp` over the table *in
  declaration order*, so abbreviations resolve to the first level-qualified row — an order-sensitive
  behavior Go's map-based registry does **not** replicate. That is a real, separate fidelity concern;
  **do not tackle it here** — if you can cheaply file a `[Fidelity]` finding for it, do, and reference
  it. F0b is level/position gates + messages only.
- The Result-DTO/structured-WS work, act()/message routing (F0a), and any per-command behavior beyond
  the gate/message. No handler-logic changes.

## Oracle proof (required — this is the gate)

Extend the existing `cmd/dp-oracle-diff/scenarios/security-gates.txt` (from PR #295, mortal setup
already lands a level-1 char) into a **gate matrix** and add a **position** scenario:

- **Level gate (over-level ⇒ hidden `Huh?!?`):** mortal probes a spread of immortal commands
  (`hcontrol show`, `summon someone`, plus 2-3 more admin commands from the table, read-only/benign).
  **Red (pre-fix):** Go executes/accepts where C says `Huh?!?`. **Green:** every one returns Go's
  `Huh?!?` matching C. Paste both reports.
- **Position gate + messages:** drive the mortal into non-standing positions and probe a
  position-gated command, so each C position string is observed:
  - `sleep` then a `POS_STANDING` command ⇒ C `In your dreams, or what?`
  - `rest` then same ⇒ C `Nah... You feel too relaxed to do that..`
  - `sit` then same ⇒ C `Maybe you should get on your feet first?`
  (DEAD/INCAP/STUNNED/FIGHTING are harder to stage deterministically without combat/RNG — cover what
  the harness can reach deterministically; assert the rest via the unit test below. Note which is
  which.) **Red→green:** the position replies go from Go's invented text to C's verbatim strings.

**Mechanical guard (the highest-value artifact — required):** a **table-driven Go test** that pins
**every** command's `(MinLevel, MinPosition)` to C's `cmd_info[]` value. Encode C's table as test data
(parse it from the oracle `src/interpreter.c` at test-authoring time, or embed a checked-in golden
extracted from it — say which, and how to regenerate). For each Go registration, assert its gate
equals the C row; fail on any Go command missing from C without an explicit documented rationale, and
any C command missing from Go. This test is what stops the `0,0` drift from ever recurring — it is the
permanent proof, alongside the oracle red→green. **A green build alone is not sign-off.**

## ⛔ Guardrails
- Match C exactly; read `src/interpreter.c` + `structs.h`, don't invent gate values or messages.
- Do NOT modify the C oracle clone / `DP_SEED`.
- Do NOT change command *handler* behavior, act()/messaging (F0a), or start structured-WS work.
- Do NOT try to replicate C's abbreviation ordering here (out of scope; file it).
- Verify the C↔Go **level constant mapping** explicitly — it's a known trap; document the table.
- Build gate green: `go build ./... && go vet ./... && go test ./...` + lint + gofumpt.

## Success criteria (PR shows ALL)
1. One authoritative command table with **every** command's `(minLevel, minPosition)` sourced from C
   `cmd_info[]`; Go-only commands each documented with a rationale; `summon`/`hcontrol` re-landed on
   the table.
2. Dispatcher cascade reordered to C's order (level/lookup gate → … → position); over-level still
   returns `Huh?!?`.
3. `positionFailMessage` strings replaced with C's verbatim (incl. INCAP+MORTALLYW merge, SITTING &
   FIGHTING cases).
4. **Oracle red→green** pasted inline for the level matrix + the position messages.
5. **Table-driven matrix test** pinning every gate to C's `cmd_info[]` (with regen instructions),
   plus coverage of the position strings the oracle can't stage deterministically.
6. C↔Go level-constant mapping documented; build gate green; no handler-behavior changes.

## Closes (Linear)
O10/O24/O40 privilege leaks + the position cluster: **DP-1094, DP-1095, DP-1108, DP-1109, DP-1111,
DP-1118, DP-1119, DP-1120** (confirm exact IDs against each issue's C citation as you land them; some
were patched inline in PR #295 and are re-landing on the table here). Mark each Done only with its
oracle red→green attached.

## Wrap-up
Commit; push; open PR with the red→green oracle reports + the matrix test inline; STOP — Claude reviews
against `origin/main` + `src/interpreter.c` (esp. gate-value parity, cascade order, and verbatim
position strings) and merges. **After F0a + F0b:** the **Observation** domain is the first full
Result-DTO migration (refactor plan §3).
