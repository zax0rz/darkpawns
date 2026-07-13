# BRIEF 2026-07-13 — FIX: the three Urgent privilege/corruption leaks (summon, hcontrol, zap)

**Executor:** a capable worker (Kimi or GLM — mechanical, well-scoped). **Security-sensitive; Claude
reviews carefully against `src/*.c` before merge.** This is the **first fix under the oracle-proof
gate** — it sets the template: **a green build is NOT sign-off; each fix must be PROVEN by an oracle
red→green run (or, where the harness can't observe the bug, a targeted test that asserts the C
invariant).**
**Branch:** `fix/security-three` off current `main`. **One PR, three commits** (one per finding) +
the proof artifacts.
**Fixes:** DP-1108 (O42 `summon`), DP-1109 (O43 `hcontrol`), DP-1110 (O41 `zap`). Read each Linear
issue; they have the C/Go citations.

---

## Ground rule for fixes (new — read first)

Unlike the audits, this brief **changes game logic** — so the guardrail flips: **match C exactly.**
- C (`~/.openclaw/workspace/darkpawns-c-oracle/src/*.c`) is the oracle. Read it; don't invent behavior.
- **Player-facing output text is first-class fidelity — as important as the mechanical effect.** A
  fix isn't done when the side effect is right; it's done when the **response text also matches C**.
  A command that behaves correctly but prints the wrong (or no) message is still a fidelity bug. The
  oracle diff checks exactly this: response text must match C (the normalizer only masks
  prompts/ANSI/RNG/timestamps — noise, not wording).
- Do **NOT** modify the C oracle clone or its `DP_SEED` patch.
- Build gate green (`go build ./... && go vet ./... && go test ./...`, plus lint/gofumpt).
- **Do not sign off on the build gate alone** — attach the oracle/ test proof described per fix.

## Fix 1 — DP-1108 / O42: `summon` leaks a mortal player-teleport

**C-faithful target:** C has **no** player/admin `summon` command (summon is a *spell*). A mortal
typing `summon` should get the ordinary unknown/unavailable-command response — never a teleport.
**Change:** gate the `summon` registration to immortal (`LVL_IMMORT`) so mortals can't reach it
(`pkg/session/commands.go:171`), OR remove the debug command entirely (immortals already have
`goto`/`transfer`). Prefer **gating to immortal** unless Zach/Claude says remove. If kept, route the
relocation through the canonical teleport path rather than a raw `RoomVNum` assignment
(`pkg/session/cmd_info.go:508-531`) so room/light/events/scripts fire.
**Proof (oracle red→green):** add a probe to a new scenario (below) where the **mortal** test char
types `summon someone`. Pre-fix Go accepts/attempts the teleport; C rejects. Post-fix Go must
reject (no teleport side effect), matching C's "mortal cannot use this command" behavior. Show the
block going from divergent → clean/localized.

## Fix 2 — DP-1109 / O43: `hcontrol` mortal-accessible house control

**C-faithful target:** C gates `hcontrol` at `LVL_GRGOD` (`src/interpreter.c:491`). A mortal can't
see or run it.
**Change:** register `hcontrol` at the GRGOD level (`pkg/session/commands.go:363`) **and** add a
defense-in-depth level check at the top of `World.Hcontrol` (`pkg/game/house_control.go:333`) before
any dispatch — so the persistent mutation is gated even if another caller reaches it. (Belt-and-
suspenders because house mutations save to disk.)
**Proof (oracle red→green):** probe the **mortal** char typing `hcontrol show` (read-only, safe).
Pre-fix Go executes; C rejects. Post-fix Go rejects, matching C. Show red→green. (Do NOT probe
build/destroy in the oracle run — it mutates state; `show` is enough to prove the gate.)

## Fix 3 — DP-1110 / O41: `zap` corrupts shared prototype charges

**C-faithful target:** wand/staff charges are **instance** values, decremented on the used object
(`src/spell_parser.c:724-810`).
**Change:** the duplicate `zap` handler must stop writing `item.Prototype.Values[2]`
(`pkg/session/use_cmds.go:144-163`). Route `zap` through the canonical `DoUse`
(`pkg/game/other_economy.go:177-240`) which already uses instance `GetValue`/`SetValue`
(`pkg/game/object.go:432-455`), or replicate that instance-safe decrement. Remove the separate
registration if `DoUse` covers it (`pkg/session/use_cmds.go:199-202`).
**Proof — two properties, two mechanisms (both required):**
- **State (the corruption):** cross-instance state isn't visible in one command's output, so prove it
  with a **Go test** — spawn two instances of the same wand/staff vnum, `zap` one, assert (a) the
  other instance's charges are unchanged and (b) `Prototype.Values` is unchanged. Regression test.
- **Output (the wording):** `zap`'s success/charge-depleted/failure messages are player-facing and
  must match C (`src/spell_parser.c:724-810` + the wand/staff use messages) — this is first-class,
  not optional. Verify the message text against C. Prove it via an oracle probe (give the setup char
  a wand so `zap`'s output can be diffed) — and if wiring a wand into the oracle setup is genuinely
  heavy, at minimum assert the exact C message strings in a Go test AND file a short follow-up to add
  the oracle wand scenario. Do not skip output verification.
State clearly in the PR which proof covers which property.

## The oracle proof scenario

Add `cmd/dp-oracle-diff/scenarios/security-gates.txt` using the existing setup/probe format (mortal
setup already lands a level-1 char). Probe phase (shared, diffed):
```
[probe]
summon someone
hcontrol show
quit
```
- **Before the fix:** the `summon`/`hcontrol` blocks diverge (Go executes/accepts; C rejects) — this
  is the "red" baseline; capture it.
- **After the fix:** those blocks are clean or localized to a benign text delta (both reject) — the
  "green." Paste both the red (pre-fix) and green (post-fix) reports in the PR. **That red→green pair
  IS the sign-off**, per the oracle-proof gate.
- **Acceptance bar = both the effect AND the wording match C.** The mortal must (a) get no side
  effect (no teleport / no house dispatch) AND (b) see the response text C shows. In C, a command
  above your level reads as the standard unknown-command reply (`Huh?!?`-style — C hides
  over-level commands), so a mortal's `summon`/`hcontrol` should produce Go's equivalent of that
  **matching C's text**. Check what Go's dispatcher prints for an over-level/unknown command:
  - If it already normalizes-matches C → done.
  - If Go's over-level/unknown wording differs from C's, that's a real fidelity gap in scope here.
    If it's **local** to these commands, fix it. If it's a **global dispatcher-wording** issue
    (affects every unknown/over-level command — high leverage), fix it at the dispatcher if
    low-risk, otherwise file it as a linked `[Fidelity]` finding and reference it in this PR — but
    do **not** dismiss it as "minor." Wording parity is part of the definition of done; if you defer
    it, say so explicitly and link the follow-up.

## ⛔ Do not
- Change any behavior beyond these three gates/routing fixes (no scope creep into the other O-findings).
- Touch the C oracle clone / `src/*.c` / `DP_SEED`.
- Weaken any existing immortal capability — immortals must retain `summon`/`hcontrol` if gated (not removed).

## Success criteria (PR must show ALL)
1. Three commits, one per finding, each matching the C-faithful target above.
2. **Oracle proof:** the pre-fix (red) and post-fix (green) `security-gates` reports pasted inline
   for summon + hcontrol — "green" means **both** no side effect **and** response text matching C
   (or an explicitly linked follow-up if the wording gap is a global dispatcher issue).
3. **State + output for zap:** the two-instance regression test (instance + prototype isolation)
   AND verification that `zap`'s player-facing messages match C (oracle probe or asserted C strings).
4. `go build ./... && go vet ./... && go test ./...` + lint + gofumpt green.
5. Immortal `summon`/`hcontrol` still work when invoked by an immortal (note how verified).

## Wrap-up
Commit; push; open PR with the red→green oracle reports + the zap state test inline; STOP — Claude
reviews against `origin/main` + `src/*.c` (esp. the gate levels and that `zap` no longer touches
`Prototype.Values`) and merges. This PR is the reference example for every fix block that follows.
