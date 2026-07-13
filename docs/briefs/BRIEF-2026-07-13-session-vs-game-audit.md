# BRIEF 2026-07-13 — Audit: `pkg/session` command layer vs `pkg/game` core vs C oracle

**Executor:** ChatGPT / frontier (deep, judgment-heavy read-and-report — plays to its strength;
rate limit is lifted today, so a full sweep is on the table). This is an **AUDIT, not a fix spree**
— read three codebases, classify, report. Claude triages the output into `[Fidelity]` issues.
**Branch:** `docs/session-vs-game-audit` off current `main`. **One PR** adding a research doc (+
optional supporting notes). No code changes.
**Read first:** `docs/research/drafts/2026-07-12-c-oracle-differential-testing.md` (the oracle
program this feeds) and the 9 findings already filed (DP-1083…DP-1093) so you don't re-file them.

---

## Thesis (already evidenced — your job is to map its full extent)

The Go port has **two layers that both implement player commands**, and they've **drifted** from
each other and from C:

- **`pkg/session/*`** — the command handlers wired into `cmdRegistry` (`pkg/session/commands.go`,
  **~264 registrations**) that actually serve players over telnet + WebSocket. Entry: a command
  name → a `cmdXxx` handler → either it reimplements the logic inline, or it delegates to `pkg/game`.
- **`pkg/game/*`** — the world/game core. It contains **faithful C ports** of much of the same
  logic (function names often mirror `src/*.c` exactly), used by mobprogs / spec-procs / combat —
  but **not always wired to the player command**.

**7 of the 9 oracle findings so far live in `pkg/session`**, several diverging from `pkg/game`'s
*own, more faithful* implementation. Two parallel-implementation pairs are already confirmed:

1. **`look`:** `pkg/game/look.go` is a full C-faithful port (`lookAtRoom`, `listObjToChar`,
   `lookAtTarget` *with* room extra-descs, `lookInDirection`, `doAutoExits`) — but the **player**
   `look` command (`commands.go:61`) is wired to `pkg/session/cmd_look.go`, a thinner reimpl that
   drifted → **DP-1083** (leaks room vnum), **DP-1086** (no room extra-descs), **DP-1089** (one-line
   directional look). `pkg/game/look.go` gets vnum-gating, extra-descs, and directional look *right*.
2. **`doGenDoor`:** exists in **both** `pkg/session/door_cmds.go` AND `pkg/game/act_movement.go:609`
   — and **both are door-only**, neither handles containers like C's `do_gen_door` → **DP-1091**
   (`open <container>` fails).

## The classification framework (apply to every player command)

For each player-facing command, put it in ONE bucket and record file:line on all three sides
(session handler / game-core equivalent if any / C `src/*.c`):

- **Bucket 1 — REIMPLEMENTED & DRIFTED.** Session handler reimplements logic that also exists in
  `pkg/game` (or duplicates another impl), and diverges. *These are the systemic wins:* the fix is
  often "delete the session copy, delegate to the canonical `pkg/game` function" — one refactor can
  close several findings. Sub-note whether the `pkg/game` copy is C-faithful (delegate to it) or
  ALSO wrong (consolidate + fix once). Examples: `look`, `doGenDoor`.
- **Bucket 2 — DELEGATES CORRECTLY, but the game function has a bug.** Session correctly calls
  `pkg/game`, and the divergence is *in* the game function. Fix in `pkg/game`. Example: `get` →
  `World.DoGet` (item_transfer.go) carries **DP-1092** (wrong container name in message).
- **Bucket 3 — SESSION-ONLY, no game equivalent.** Logic lives only in `pkg/session`, or the
  command is missing entirely. Fix/implement in place. Examples: `score` (session-only, **DP-1093**
  wrong hometown table); `exits` (missing entirely, **DP-1090**).

The **headline deliverable** is not just "N bugs" — it's: *"Here is the `session → game` delegation
refactor that eliminates a whole class of divergences, plus the residual Bucket-2/3 gaps that need
real work."* Quantify it: how many current + newly-found findings would a delegation pass close?

## Method

1. **Enumerate** the player commands from `pkg/session/commands.go` (the registry is the ground
   truth for what players can type — names, aliases, handlers, min-position, min-level).
2. **For each, pair** the session handler with its `pkg/game` equivalent (grep for `func (w *World)
   doXxx` / `DoXxx`) and the C command in `src/interpreter.c`'s command table → the `ACMD(do_xxx)`
   in `src/act.*.c` / `src/*.c`.
3. **Diff behavior** against C (the oracle is ground truth): messages, ordering, gating
   (level/position/flags), what data is shown/hidden, command existence, argument forms. Cite the C
   line that proves the correct behavior.
4. **Classify** (bucket 1/2/3) and record. Flag delegation-fixable ones explicitly.

**Prioritize by hit-rate & impact** (don't try to boil all 264 at once — go in this order, land the
doc incrementally if needed):
- **Tier A (do first):** informative (`look`, `exits`, `score`, `inventory`, `equipment`,
  `examine`, `who`, `where`, `consider`, `time`, `weather`) and object/container (`get`, `drop`,
  `put`, `wear`, `remove`, `wield`, `hold`, `open`, `close`, `lock`, `unlock`, `give`, `drink`,
  `eat`, `fill`, `pour`). Highest density of divergence so far.
- **Tier B:** movement (`north`…`down`, `enter`, `follow`, `rest`/`sit`/`stand`/`sleep`/`wake`),
  communication (`say`, `tell`, `emote`, `shout`, `gtell`), grouping.
- **Tier C:** the rest, noting immortal/wiz commands separately (lower player-facing priority).

**Explicitly OUT of scope for divergence-diffing:** RNG-outcome parity for combat/skills/spells
(that's the oracle's Tier-2, needs the `random.c` port). BUT *structural* issues in those commands
— missing command, wrong message text, wrong gating — ARE in scope.

## Deliverable format

A research doc at `docs/research/drafts/2026-07-13-session-vs-game-audit.md` containing:
- **Executive summary:** the delegation-refactor headline + counts per bucket.
- **A divergence table**, ranked by player impact, each row:
  `command | bucket | C (file:line + correct behavior) | session (file:line + what it does) |
  game equiv (file:line, faithful?) | divergence | impact | fix (delegate / fix-in-game /
  implement)`.
- Each **new** divergence in the O-finding format (C: / Go-session: / Go-game: / Divergence: /
  Player impact: / Fix sketch:) so Claude can file it fast.
- A **"delegation candidates"** section: the Bucket-1 commands where routing session → the
  `pkg/game` function is the fix, with a note on any interface/DTO obstacle (see §WS nuance).

Do **not** file Linear issues yourself — deliver the doc; Claude verifies against source and files
the real ones (same loop as O1-O9). Dedup against **DP-1083..DP-1093** (list them as "already
filed" in the doc so they're not re-reported, but DO note when your audit confirms one's root cause).

## WS/telnet nuance (know this before proposing delegation)

Session handlers often emit a **structured `MsgState`/`ServerMessage` JSON** (for the WebSocket
client); the telnet listener's `formatState` (`pkg/telnet/listener.go:672`) renders that JSON to
text. So some divergences live in the *structured payload* (e.g. `RoomState.VNum` populated
unconditionally) and some in `formatState`'s *rendering* (e.g. DP-1083's unconditional `[vnum]`).
`pkg/game/look.go` by contrast sends text directly via `SendMessage`. A naive "delegate session →
game" can therefore lose the structured WS payload — so for each delegation candidate, note whether
the game function can feed BOTH transports (structured for WS, text for telnet) or whether a shared
core + two thin renderers is the right shape. This is the main design subtlety; call it out, don't
hand-wave it.

## ⛔ Guardrails
1. **Read-and-report only.** No code changes to `pkg/session`, `pkg/game`, or anywhere — the PR adds
   a doc.
2. Cite **file:line on all three sides** for every claim; the C line is the ground-truth proof.
3. Do NOT touch the C oracle clone or `src/*.c`.
4. Don't re-file DP-1083..DP-1093; reference them.

## Success criteria
1. Tier-A commands fully audited and classified (bucket 1/2/3) with three-sided citations.
2. The delegation-refactor headline quantified (how many findings a session→game pass would close).
3. New divergences captured in O-finding format, ready for Claude to verify + file.
4. The WS/telnet transport subtlety addressed for each delegation candidate.
5. `go build ./... && go vet ./... && go test ./...` still green (doc-only PR; trivially true).

## Wrap-up
Commit the doc; push; open PR; STOP — Claude reviews the doc against `origin/main` + `src/*.c`,
verifies the divergences, and files the confirmed `[Fidelity]` issues. Tiers B/C can be a follow-up
PR if Tier A fills the day.
