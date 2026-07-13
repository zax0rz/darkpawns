# BRIEF 2026-07-13 — Domain 1: Observation (look / examine / exits / diagnose) — the first full Result-DTO migration

**Executor:** ChatGPT / frontier. **Read the refactor plan first — this brief IS its §3 "First full domain:
Observation":** `docs/research/drafts/2026-07-13-session-game-refactor-plan.md`. Foundations F0a (act(),
PR #297) and F0b (command-gate table, PR #298) are **merged** — this is the first domain to ride on them.
Claude reviews; **every behavior/wording change is proven by an oracle red→green run**
(`[[darkpawns-oracle-proof-gate]]` — player-facing output is first-class fidelity, equal to the effect).
**Branch:** `refactor/domain-observation` off current `main`. Series of tight PRs is fine.

---

## Why this domain first, and what it must PROVE

Observation is deliberately the first full migration because it is **read-only** (no state mutation to get
wrong) and because `pkg/game/look.go` is already the closest-to-canonical implementation in the tree. Its
job is not just to fix the look findings — it is to **establish the Result-DTO + dual-renderer pattern that
every later domain (consumables, inventory, movement, comm, …) will copy.** Get the boundary right here on
the safe surface.

### The core problem: a room has THREE divergent renderers, none authoritative
1. **`pkg/game/look.go`** (`doLook`/`lookAtRoom`/`lookAtTarget`/`lookInDirection`/`lookInObj`) — emits **text**
   via `ch.SendMessage`, and is the **closest to C** (it gates vnum behind roomflags at ~:94, iterates room
   extra-descs at ~:249). But the session/telnet `look` path does **not** use it.
2. **`pkg/session/cmd_look.go`** (`cmdLook`/`cmdLookAt`/`cmdLookDirection`/`cmdLookIn`) — a **separate**
   reimplementation that builds the structured `StateData` for the web client, and has **drifted** (skips
   room extra-descs → DP-1086; one-line directional peek → DP-1089; examine over/under-exposure → DP-1084/1107).
3. **`pkg/telnet/listener.go:663 formatState`** — a **third** renderer that turns the pushed `StateData` into
   telnet text, wrapping the room in a `\r\n---\r\n … ---\r\n` box, printing `name [vnum]` **unconditionally**
   (DP-1083 O1 leak), and emitting a **player status line** (`name the class  Lvl N  HP:…`) that C's
   `do_look` never produces.

Every finding below traces to the session/telnet path bypassing the game core. **The migration collapses all
three into one canonical op + two thin renderers.**

## Target architecture (THE deliverable — this is the pattern to establish)

```
   look/examine/exits/diagnose command
              │
              ▼
   pkg/game canonical observation op(s)     ← owns C rules + visibility; NO transport knowledge
     DoLookRoom / DoLookTarget / DoLookDir / DoExits / DoExamine / DoDiagnose
     returns an ObservationResult, NOT text:
       { Messages  []ActMessage   // C-faithful, viewer-resolved (look is ToChar-only; use act())
         Room      *RoomView      // structured facts for WS (nil for non-room observations)
         Events    []SemanticEvent }
              │
      ┌───────┴─────────────────────┐
      ▼                             ▼
  telnet renderer               WS/session renderer
  print Messages as C-faithful  build StateData/RoomState from RoomView
  text; viewer-aware vnum        (schema PRESERVED — see hard constraint);
  (roomflags→[vnum]); NO ---box  render any text via the same Messages
```

Because `look` output is **single-viewer** (goes only to the looking char — unlike multi-audience act()),
the op resolves visibility **for the looker** and the telnet text can be produced directly. That single-viewer
simplicity is exactly why Observation is the safe place to prove the boundary.

### What to build
1. **Canonical ops in `pkg/game`** (extend `look.go`; do not add a 4th copy). Bring each to full C fidelity
   against `src/act.informative.c` — the current `look.go` is a strong starting point, not the target:
   - `look_at_room` (:725) — dark/blind messages (verbatim: `"Darkness\r\n\r\n"` + the cyan code, blind →
     `"You see nothing but infinite darkness..."`, dark → `"It is too dark here to see much of anything..."`,
     plus the per-occupant movement-sound / glowing-eyes lines); room name with **`[vnum] name [flags][sector]`
     only under `PRF_ROOMFLAGS`** else bare name; cyan/normal color codes; description (respecting BRIEF /
     ignore_brief / ROOM_DEATH); autoexits under `PRF_AUTOEXIT`; then `list_obj_to_char(...,20,FALSE)` (:832)
     and `list_char_to_char` (:835) with C's exact ordering, colors (green objects / yellow chars), and
     visibility.
   - `look_at_target` (:1005) — resolve in **C's order**: (1) **room extra-descs FIRST** (:1033), (2) equipped
     ex-descs, (3) inventory ex-descs, (4) room-object ex-descs, (5) chars; else `"You do not see that here."`
     (fixes DP-1086; `read` and `examine` both route here).
   - `look_in_direction` (:841) — `"You look <dir>wards."`, exit `general_description` / door-state, and when
     the exit is open, render the **full destination room** via the room path (:900-904) (fixes DP-1089).
   - `look_in_obj` (container/drinkcon) — contents listing per C.
   - `do_exits` (:683) — the missing `exits` command; obvious exits + destination/door state (DP-1090). The
     **gate row already exists** in F0b's `command_gates.tsv` (`exits POS_RESTING`); you only need the handler
     + `registerCommand("exits", …)`.
   - `do_examine` (:1137) — routes through target look, **also searches equipped**, shows **contents** for
     containers/drinkcon/fountain, and renders **only C-visible fields** (fixes DP-1107 and, by construction,
     the DP-1084 stat-block over-exposure — no raw stat dump).
   - `do_diagnose` (:2433) — port C's condition wording (O35 — confirm its DP id in the tiers-bc audit).
2. **The `ObservationResult` type + the two renderers.** Define the result in `pkg/game` (or a shared pkg),
   with the telnet and WS renderers as thin adapters. Telnet prints the C-faithful `Messages`; **retire the
   room path of `formatState`** (no `---` box, no status line on look) and make vnum **viewer-aware** (print
   `[vnum]` only when the looker has roomflags/holylight). WS builds `StateData`/`RoomState` from `RoomView`.
3. **Delete the session reimplementations** once the command routes to the canonical op: the room/target/dir
   builders in `cmd_look.go` and the `examine.go` stat path. One code path.

## HARD CONSTRAINT — do not break the WebSocket schema
The web client consumes `StateData{Player PlayerState, Room RoomState}` (`pkg/session/protocol.go:71-120`).
**Golden this schema BEFORE touching it** (serialize a representative `StateData` to a checked-in golden JSON;
add a test that fails on breaking changes). You may **add** fields (e.g. a room-flags/viewer-level hint so the
renderer can gate vnum, or richer object/char entries) but must not rename/remove/retype existing ones. Note
`RoomState.VNum` stays in the WS payload always — the web client keeps it; only the **telnet** renderer gates
its display. This is the "Result feeds both WS structured + telnet text" problem the whole refactor hinges on —
solve it cleanly here.

## Scope & the ONE allowed cross-cut (movement auto-look)
**In scope:** `look` (room / `look <target>` / `look in <container>` / `look <dir>` / `read`), `examine`,
`exits`, `diagnose`.

**The coupling you must handle:** walking into a room also displays it. Today that room display is produced by
the same drifted `StateData`→`formatState` path. If you fix `look` but leave the post-move display alone,
telnet players will see a C-faithful room on `look` but the old `---`-box room after moving — an inconsistent
regression. **Trace how the post-move room display is produced** (does movement call session `cmdLook`, push
`StateData` directly, or call `game.doLook`?) and **repoint that display at the new canonical room renderer**
so both paths agree. That call-site repoint is the ONLY movement change allowed — **do NOT touch movement
mechanics** (charm/mount/follower/hide/room-scripts); those are the Movement domain. If the repoint turns out
to be non-trivial, stop and flag it rather than pulling movement into this PR.

**Out of scope:** any other domain; the WS structured payloads for non-observation commands; skill/spell system.

## Oracle proof (required — the gate)
Add scenario(s) (e.g. `cmd/dp-oracle-diff/scenarios/observation.txt`) driving a **mortal** and an **immortal**
(for the roomflags/vnum + brief paths) through:
```
look
look sign            # room extra-desc (DP-1086)
look east            # directional full-room render (DP-1089)
exits                # (DP-1090)
examine <container>  # equipped/contents + no stat dump (DP-1107/1084)
diagnose <target>
```
- **Red (pre-fix):** capture today's divergences — telnet `[vnum]` leak + `---` box + status line, `look sign`
  → "you don't see that", one-line directional peek, `exits` unknown, examine stat-dump.
- **Green:** actor output matches C (normalized). Paste both reports.
- **Plus unit tests:** (a) the **WS schema golden** (StateData/RoomState unchanged); (b) a telnet-render test
  asserting **no `---` box and no player-status line** on `look`, and that `[vnum]` appears **only** for a
  roomflags viewer; (c) room-extra-desc precedence (room exdesc matched before room objects). Oracle red→green
  **and** these tests together are sign-off; a green build alone is not.

## ⛔ Guardrails
- Match C exactly — read `src/act.informative.c` (`look_at_room`/`look_at_target`/`look_in_direction`/
  `do_exits`/`do_examine`/`do_diagnose`/`list_obj_to_char`/`list_char_to_char`); don't invent wording/order/color.
- Use the merged `Act()` for any player-facing message; don't hand-substitute `$`-codes.
- Preserve the WS `MsgState`/`StateData` schema (golden first; additive only).
- Don't touch movement mechanics, command gates (F0b owns them), or other domains.
- Don't modify the C oracle clone / `DP_SEED`.
- Build gate green: `go build ./... && go vet ./... && go test ./...` + lint + gofumpt.

## Success criteria (PR shows ALL)
1. One canonical observation op set in `pkg/game` (extended `look.go`) at full C fidelity; the session
   `cmd_look.go`/`examine.go` reimplementations **deleted**; both telnet + WS render from the same
   `ObservationResult`.
2. Telnet room render is C-faithful: no `---` box, no status line on look, vnum gated behind roomflags;
   `formatState`'s room path retired.
3. `exits` command added + registered (gate already in F0b table); `examine` searches equipped + shows
   contents + no raw stat dump; `look <dir>` renders the full destination room; `look/read <room-feature>`
   resolves room extra-descs first.
4. Post-move room display repointed to the canonical renderer (movement mechanics untouched) — telnet room
   output is consistent between `look` and walking in.
5. **WS schema golden** test present and passing (additive-only changes).
6. **Oracle red→green** for the observation scenario (mortal + immortal) pasted inline, plus the telnet-render
   and extra-desc-precedence unit tests.
7. Build gate green.

## Closes (Linear — confirm each against its C citation as it lands)
**DP-1083** (O1 vnum leak), **DP-1084** (O2 examine stat-block), **DP-1086** (O4 room extra-descs),
**DP-1089** (O5 directional look), **DP-1090** (O6 `exits`), **DP-1107** (O23 examine equipped/contents),
and **O35** (diagnose — confirm DP id). Mark each Done only with its oracle red→green attached.

## Wrap-up
Commit; push; open PR(s) with the red→green oracle reports + schema-golden/telnet-render/extra-desc tests
inline; STOP — Claude reviews against `origin/main` + `src/act.informative.c` (esp. vnum-gating, color codes,
list ordering/visibility, extra-desc precedence, and that the WS schema is intact) and merges. **This PR is
the reference example for every domain migration that follows** — the Result-DTO boundary it establishes is
reused by Consumables (next), Object/inventory, Equipment, Movement, and Communication.
