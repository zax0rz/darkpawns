# BRIEF 2026-07-12 — Harden the oracle-diff scenario/alignment (kill creation-flow desync)

**Executor:** Kimi. This is **test-harness tooling only** — NOT a change to either game's
logic. Hold tightly to this brief and **prove every success criterion with real pasted
output** (verification discipline is the bar here — a green claim without a pasted transcript
doesn't count). Claude reviews the PR against `origin/main`.
**Branch:** `fix/oracle-diff-scenario-hardening` off current `main`. **One PR.**
**Read first:** `cmd/dp-oracle-diff/main.go`, `internal/oraclediff/{normalize,conn,scenario,diff}.go`,
and the current scenario `cmd/dp-oracle-diff/scenarios/look-start-room.txt`. Background:
`docs/research/drafts/2026-07-12-c-oracle-differential-testing.md` (two-tier plan).

---

## The bug in the harness (diagnosis — already done, don't re-derive)

The first differential run reported the C start room as **"Temple Infirmary"** and Go as
**"Temple Altar [8004]"**. That is **NOT a game divergence** — it's a harness artifact.
Verified against C source:
- C `mortal_start_room = 8004` (`config.c:142`) = the room literally named **"At the Temple
  Altar"** (`lib/world/wld/80.wld` #8004). A new mortal enters there.
- `do_start()` (`class.c:501-533`) grants starting gear and sets level 1 — it does **not**
  relocate the char. So both games genuinely start a new mortal at 8004.
- "Temple Infirmary" is a **different room, #8162**. The char was never supposed to be there.

**Root cause:** the scenario sends **one shared input stream** verbatim to **two structurally
different creation flows** (C `nanny()` in `interpreter.c` vs the Go CON_MENU flow). The
comment in the scenario file even says the extra `Y` "exposes the different creation-flow
structure." Because the two menus consume a *different number of prompts*, every line after the
first divergence is **misaligned** — the two characters end up in different creation states, and
by the time `look`/`examine sword` fire they're in different contexts (or one is still mid-menu).
The room-name mismatch is a *symptom* of that desync, not a world-data bug.

**Landmine to respect:** if the test char is the **first record in an empty player file**,
CircleMUD promotes it to **Implementor** → `LVL_IMMORT` → start room becomes `immort_start_room`
= **1204**, not 8004. codex's harness already preserves a baseline player file to avoid this —
**keep that**; the setup phase must land a **mortal** at 8004.

## The fix: split every scenario into SETUP (per-server, not diffed) + PROBE (shared, diffed)

The core mistake is conflating two different comparisons in one stream. Separate them:

### 1. Setup phase — per-server, output NOT diffed
Its only job is to drive each server's char into the **same known state**: a **level-1 mortal
standing in room 8004 at a normal command prompt.** Because the creation flows differ, the setup
input lines **may differ per server** — that's expected and fine. Derive them:
- **Go flow:** from the existing CON_MENU E2E tests (the char-creation E2E work around DP-1067)
  — they already encode a valid keypress sequence that creates a character and enters the game.
- **C flow:** from `nanny()` (`interpreter.c`, the login/creation state machine) — the sequence
  that creates a char, picks a class, and reaches `CON_PLAYING`.
The driver plays the setup lines, drains output, and does **not** diff it. (Optional, nice: assert
each server reached a playing prompt before starting the probe — fail fast if setup broke.)

### 2. Probe phase — shared, diffed
Identical commands to both servers; **only this tail is normalized + diffed.** Use
**deterministic, class/inventory-independent** probes (pure world data both servers load):
- `look` — room 8004 name + description. Both must show **"At the Temple Altar"** + the desc.
- `look sign` — room 8004 has an **extra-desc** keyword `sign` →
  *"It reads: Please don't leave your pets in the Temple area."* This is the cleanest possible
  text-fidelity probe: static world data, no RNG, no class/inventory dependence.
- `quit`.

**Drop `examine sword` from the deterministic probe** — the starting weapon is class-dependent
(warrior gets small sword 8037, mage a dagger, etc.), so it's a confound. If you want an object
probe, either (a) make setup deterministically pick a fixed class so both inventories match, or
(b) prefer the world-data `sign`. Note DP-1084 already captured the `examine` stat-block
divergence, so we don't need `examine` to re-find it.

### Scenario file format
Extend the format to express per-server setup + a shared probe, kept dead simple. Suggested:
```
[setup:oracle]      # sent only to the C oracle; not diffed
<C creation keystrokes…>
[setup:port]        # sent only to the Go port; not diffed
<Go creation keystrokes…>
[probe]             # sent to BOTH; this is the only diffed section
look
look sign
quit
```
Section names your call — match repo conventions. Migrate `look-start-room.txt` to the new
format (or add a new `look-start-room-v2.txt` and leave the old one). Update `ParseScenario`
accordingly; keep the parser covered by a pure unit test.

## Secondary hardening (do these too)

- **Per-command block alignment.** Even in the probe phase, don't diff one giant concatenated
  blob. Send one command → `ReadUntilQuiescent` → capture that command's response as a **labeled
  block** keyed by the command → diff **block-by-block**. This localizes a divergence to the
  command that caused it and stops a single extra/missing line from cascading through the whole
  transcript (part of why the first run looked globally misaligned).
- **Watch `volatileLine` (normalize.go).** It's a suspect for dropping real lines and shifting
  alignment. With per-command blocks it's less dangerous, but still **bias toward
  under-normalizing** — a false diff is a cheap human glance; a dropped real divergence is
  invisible. If a probe line (e.g. the sign text) is being eaten, tighten the regex.
- **Keep codex's two correct behaviors:** preserve the baseline player file (mortal, not
  Implementor — see landmine above) and reserve `oraclePort+1` for Circle's WHOD listener.

## ⛔ Guardrails (do not violate)
1. Do **NOT** modify game logic, messages, tables, or creation flow in **either** codebase to
   make outputs match. The harness observes; it never "fixes" divergences.
2. Do **NOT** touch `src/*.c` in the C clone, or the `DP_SEED` patch (already applied there).
3. Do **NOT** commit the C oracle source or the `circle` binary into the Go repo.
4. Do **NOT** break the default build gate. The harness must **skip cleanly** (exit 0) when
   `DP_ORACLE_BIN` is unset — `go build ./... && go vet ./... && go test ./...` stays green.

## Success criteria (PR must show ALL, with pasted proof)
1. **Setup lands both at 8004 as a mortal** — paste both servers' raw room lines from the first
   probe `look`, showing **"At the Temple Altar"** on both (no more "Temple Infirmary", no vnum
   1204/Implementor promotion).
2. **The probe diff is clean or localized** — `look` and `look sign` either match (empty diff for
   those blocks) or any remaining divergence is **localized to the specific command** via the new
   block alignment. Paste the report. (If `look sign` reveals a real Go divergence, great — that's
   a finding to hand back for triage, not something to hide.)
3. Per-server setup + shared probe format works; `ParseScenario` has a pure unit test.
4. `DP_ORACLE_BIN` unset → skips cleanly; full build gate green. Show it.
5. ⛔ items honored: no game-logic/message/creation changes in either tree; no C source/binary
   committed.

## Out of scope (future briefs)
- A **dedicated creation-flow-parity scenario** — creation IS worth diffing against the oracle,
  just not while also asserting room/object text. Separate scenario later; note the existing
  char-creation fidelity work (DP-1063..1081) already covered much of it manually.
- More probe scenarios (movement/sector cost, shops, score/equipment).
- Tier-2 RNG parity (porting `random.c` into Go).

## Wrap-up
`go build ./... && go vet ./... && go test ./...` green; commit; push; open PR with the pasted
proof-of-life (both room lines + the probe report) inline; then STOP — Claude reviews against
`origin/main` (esp. the ⛔ items, the mortal-not-Implementor start, and CI-skip) and merges.
