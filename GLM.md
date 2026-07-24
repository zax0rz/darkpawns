# GLM — Operating Manual (brief-execution agent)

Companion to **[AGENTS.md](AGENTS.md)** (repo-wide instructions, build/gates, conventions, website) and **[docs/fidelity/RULEBOOK.md](docs/fidelity/RULEBOOK.md)** (the C→Go translation law, R1–R5). This file is **GLM-specific**: the brief-driven workflow, the toolchain, and the pitfalls already paid for. Read it once; don't re-derive it each session.

> When this file and AGENTS.md agree, they're saying the same thing. Where this file is more specific (briefs, oracle gate, C verification), it governs GLM's behavior. **The C source is always the final authority** (R5e).

---

## The workflow (every brief, every time)

1. **Read the brief in full.** `docs/briefs/BRIEF-<date>-glm-<feature>.md`. Note the `**Cite:**` field — it names the C functions that are law.
2. **Verify the brief's claims against the code and C source before implementing** (R5e — see below). Briefs are excellent but not infallible; line citations drift. The C source wins.
3. **Plan** (EnterPlanMode for M/L tasks; small fixes can go straight). Get sign-off before writing code for anything architectural.
4. **Implement faithfully** — port the C behavior, don't invent.
5. **Run ALL gates** (below) — never skip, never self-report green without the output.
6. **Branch `glm/<feature>` off `main`**, commit (conventional commits), push, open a PR.
7. **Do NOT merge.** Review is Claude/Daeron's. Relay Claude's feedback, fix, re-push.

## The gates (run every one before committing — AGENTS.md §Build & Verify)

```bash
go build ./...           && \
go vet ./...             && \
go test ./...            && \
go test -race ./...      && \   # race is first-class — command execution moved onto the game-loop goroutine (DP-1201)
golangci-lint run ./...  && \
gofumpt -l .             && \   # must be EMPTY (gofumpt is the ONLY formatter — `make fmt` fixes)
make reachability        && \   # committed-baseline ratchet; scripts/reachability_ci_gate.py is the CI check
python3 scripts/reachability_ci_gate.py   # exits 0 on pass/regression-free, 1 on regression
```

"Subagents have lied about passing builds before." Run them; paste the tail if asked.

## R5e — verify the brief against C before you trust it

The brief is the anchor; **the C source is law.** Every brief's line citations and message tables must be confirmed by reading the C function (and tracing the Go call path) before copying. Confirmed corrections from this loop:

- **DP-1203/1207 backstab message:** a brief's table cited `"You'd better leave the sneaky stuff to the thieves."` for backstab; the C source (`act.offensive.c:174`) says `"You have no idea how."` (the sneaky-stuff line is `do_trip`). **The C source won.**
- **DP-1206 disarm:** the brief listed disarm as a Wave-1 skill needing the gate fix; `CmdDisarm` never calls `CanUseSkill` and `DoDisarm` already gates faithfully. Correctly omitted.

When you find a correction, say so in the PR (what the brief claimed vs what the C source says). Don't silently "fix" the brief — surface it so Claude can reconcile.

## Pitfalls already paid for

- **Reachability is a dynamic committed-baseline ratchet, not a hardcoded number.** Don't assert "56" or "48" — the floor moves as commands get ported (it *improves* when you register a `missing` command; that's good). Run `make reachability` + the CI gate; report the unchanged/improved count, don't hardcode it.
- **Mixed `\r\n` / `\n\r` terminators are byte-exact and intentional.** C uses both, inconsistently, within the *same function*. Copy each terminator verbatim from the C source. Normalizing to one form is an R1 bug. (Backstab's `do_backstab` uses `\r\n`; `do_skillset` uses `\r\n` for the syntax line and `\n\r` elsewhere.)
- **The double-`\r\n` trap.** Many handlers do `SendMessage(CanUseSkill_msg + "\r\n")` — they *append* the terminator. So any message table that stores strings *with* `\r\n` will double-terminate. Check where the append happens before choosing bare-vs-terminated storage.
- **`lib/` data files are the 2010 game data — read-only.** Never rewrite `lib/text/help`, `lib/misc/messages`, `lib/world/`. They're C-authored ground truth.
- **Test the test.** A unit test built from the same logic as the implementation is a re-skin — it passes with the bug. For byte-exact assertions, build the expectation by tracing C, and add a sharp guard that *would have failed* on the buggy version (then verify it does — temporarily reintroduce the bug, watch it fail, restore). See the `skillset` partial-line fix.
- **No command execution under `m.mu`.** Go's `RWMutex` is not reentrant; handlers re-acquire `m.mu`. Collect-under-RLock, ExecuteCommand outside (mirror `DamageFunc`). (DP-1201.)

## Oracle gate (Claude authors/runs — GLM sketches)

Most briefs end with an **Oracle gate** section: Claude runs the differential oracle (`cmd/dp-oracle-diff`) comparing the live Go server against the live C server, byte-for-byte. GLM's job is to make the Go path faithful and unit-tested; Claude authors the fixture scenarios and runs the gate after merge. The PR description should include an **oracle-gate sketch** (which scenarios go red→green, which stay green as regression anchors) so Claude knows what to run.

- A fix is "done" when its oracle scenario runs green (R5a), not when the unit tests pass.
- After a gate-fix PR lands, the next divergence often *surfaces* — that's expected (e.g. fixing the skill-known gate lets bash-opener pass the gate and reveal the DP-1207 message divergence). Note this in the PR.

## Harness env vars (test controls over initial state, not gameplay output)

These are Go-side inputs in the `DP_SEED`/`DP_CLOCK` category — external controls over *initial state*, not gameplay-output injections. They're fine to add and gate on (Go is ours; the read-only rule is C-oracle-only):

- `DP_CLOCK` — freezes the wall-clock game loop; the oracle pumps pulses deterministically via `GameLoop.PumpPulses`. Tests that need a frozen clock set this.
- `DP_FRESH_MUD` — treats the MUD as fresh (crown the first character God). Set **ONLY** for God-fixture scenarios; for every existing scenario it's UNSET so the primary actor is an ordinary mortal.
- `DP_FIXED_TIME` — pins the game clock.

## Git hygiene

- Branch off `main` as `glm/<feature>`. If local `main` has diverged from `origin/main`, rebase onto `origin/main` first.
- Conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`.
- **Stage only the deliverables.** Do NOT stage:
  - `website/static/map/world-sphere.json`
  - `docs/reports/reek/*`
  - scratch oracle scenarios (`cmd/dp-oracle-diff/scenarios/*.txt` that aren't part of the deliverable)
  - generated reachability reports (`docs/reports/reachability-*.tsv`) — the generator owns these; the committed baseline updates only when a command flips status
  - `.zcode/` and other editor-local state
- Commit/push **only when the user asks** (or when a brief's workflow explicitly says to). Never merge.

## What "faithful" looks like (quick reference)

- **Player-facing bytes match C exactly** (R1) — prompts, messages, menus, ordering, terminators. Copy verbatim, including grammar errors.
- **The command surface matches `cmd_info[]`** (R2) — names, aliases, min-position, min-level, subcommands. `make reachability` is the deterministic diff.
- **RNG draw counts/order match C** (R3) — every `number()`/`dice()` in a C path has a matching Go draw, same order. The shared `dprng` stream is law; a desync looks like unrelated combat divergence.
- **No invention** (R4) — if C doesn't print/do it, Go doesn't (on the player-facing surface). Modern additions (GMCP, agent vars, web admin) are allowed only where a telnet player can't observe them.
- **Find one, find the class** (R5c) — a confirmed finding triggers "what else is in this class?" via grep/script, not a feeling. Note siblings in the PR for the next sweep.
