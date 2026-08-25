# Brief: Depth-Fidelity Command Loop (GLM-5.3 autonomous `/goal`) — 2026-08-25

**Executor:** GLM-5.3 via zcode `/goal` (autonomous, self-verifying rounds).
**Repo:** `/home/zach/darkpawns` (Linux workstation), remote
`git@github-darkpawns:zax0rz/darkpawns.git`, base branch **`main`**.
**Oracle binary:** `/home/zach/darkpawns-c-oracle/bin/circle` (branch
`dp-oracle-seam`, already built on this machine).
**Build gate:** `go build ./... && go vet ./... && go test ./... && golangci-lint run ./... && gofumpt -l .` — ALL must pass, `gofumpt -l .` must print nothing.

---

## Mission

Advance the port **by behavior, not file count**: keep opening the *next*
per-command **depth-fidelity manifest**, prove each reachable player-visible
branch byte-for-byte against the C oracle, and fix confirmed divergences. This is
the same pass that shipped `get`/`drop`/`give`/`put`/`wear`/`movement`/`flee`
(19 manifests live in `docs/fidelity/depth/`). The backlog is deep — do not stop
until the usage budget is spent.

**The authority is the C source in `src/`, not the existing Go, not summaries,
not this brief.** Read `docs/fidelity/DEPTH_TESTING.md` in full before round 1
and follow its Per-Command Workflow and D1–D5 proof levels exactly. Cite
`docs/fidelity/RULEBOOK.md` rules by number (R1–R5e).

## The loop (one round = one command family = one PR)

Each round, per `docs/fidelity/DEPTH_TESTING.md`:

1. **Pick the next target** from the seed backlog below (top-down); if all are
   done, pick the next high-value un-manifested registered command family and
   say which in the PR body.
2. Locate the registered Go handler *and* the C handler + call sites. Enumerate
   reachable branches and audiences **before** touching Go (R5e — a finding
   isn't confirmed until its call path is).
3. Create/extend `docs/fidelity/depth/<command>.tsv`, one row per case, using the
   explicit statuses (`oracle-green`, `unit-green`, `delegated`, `excluded`,
   `blocked` — never relabel a real gap as `excluded`).
4. Write oracle scenario(s) in `cmd/dp-oracle-diff/scenarios/`. Add
   `# depth-case: <case-id>` annotations. Use named peers when room/victim bytes
   matter. Use focused unit tests for exact hidden state/arithmetic (D4).
5. Run `--show-oracle` at least once to confirm the intended C block executed.
6. **Fix only confirmed divergences**, and when you fix one, sweep the whole
   class (R5c). Fixes go in Go/`pkg` only — **never edit `src/`** (C is reference).
7. Gates: `make fidelity-depth`, the scenario(s), and the full build gate above.

### Per-round Definition of Done (the checkable verdict)

- The oracle scenario proving each new `oracle-green` case is **RED on pre-fix
  `main` and GREEN on the branch** — run it, do not trust the claim (R5a). For a
  pure-coverage round where the port was already faithful, state "no source
  change; port already faithful" and show the GREEN run.
- `make fidelity-depth` exits 0 (every `# depth-case:` annotation maps to a
  manifest row; every `unit-green` maps to a real test symbol).
- Full build gate passes; `gofumpt -l .` prints nothing.
- One PR opened on a `glm/depth-<command>` branch. **Never merge.**

### Oracle invocation (run on this machine)

```bash
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario <name>            # add --show-oracle to see C blocks
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario <name> --seed 2   # vary seed for RNG cases (R3)
```

Scenario skeleton (see any file in `cmd/dp-oracle-diff/scenarios/` for real
examples; `drop-basic.txt`, `quaff-effect.txt`, `recite-effect.txt` are close
models). Fixtures available: `quiet-mobs`, `spawn-obj <vnum> <max> <room> <zone>`,
`spawn-mob`, `replace-room-exits`, `set-room-flag`, `empty-players`,
`strip-mob-script`. The `[setup:*]` blocks are the char-creation keystrokes —
copy them verbatim from an existing scenario (name **Mordecai**; avoid names
containing substrings like `me`/`you`/`tit`).

## Round 1 (concrete): object-magic effect messages

Continues Phase 2 (already shipped: `quaff` → real `call_magic` in #633,
`recite` armor message in #635). The pattern: C `mag_objectmagic` casts an item's
spells via `call_magic`, and `magic.c`'s `mag_affects` sends per-spell
`to_vict`/`to_room`/`to_self` messages. Several Go `magAffectsApply` cases in
`pkg/spells/affect_spells.go` **apply the affect but send no message** — a silent
R4/R1 gap.

- **Cite:** `src/magic.c` `mag_affects()` — the `to_vict`/`to_room`/`to_self`
  strings per `case SPELL_*`, dispatched at the send block (`ch != victim` gates
  `to_self`; `to_vict` and `to_room` always fire when set). Rules **R1, R4, R5c**.
- Audit every `case Spell*` in `magAffectsApply` against its `magic.c` block.
  For each missing message, add `sendToVictim(victim, "<to_vict>\r\n")` (and the
  room line via the existing room-send path when C sets `to_room`). The
  `to_self` line for `ch != victim` needs an **objective-pronoun helper** in
  `pkg/spells` (only `possessivePronoun` exists in `say_spell.go`) — add
  `objectivePronoun` (male→"him", female→"her", neutral→"it") and wire the
  `$M` substitutions; if you defer it, mark it `blocked` in-manifest, do not skip
  silently.
- **Prove** each with a single-spell, self-target **affect** item (zero RNG, so
  fully deterministic). Known-good vnums: potion `8051` (see-invisible, already
  proven), scroll `11701` (armor, already proven), potion `7001` (bless), plus
  scan `lib/world/obj/*.obj` for others (`grep`-parse type 2/10 with a single
  affect spell in vals\[1]). Extend `quaff.tsv`/`recite.tsv` or add per-spell
  cases.
- **Explicitly out of scope this round:** RNG-bearing spells (cure/heal dice,
  damage rolls) — those need `call_magic` RNG-stream parity verification and stay
  `blocked`. Do not attempt them autonomously.

## Seed backlog (after round 1, top-down)

Prefer clean, deterministic, high-audience commands first:

1. **object-magic effect messages** (round 1, above).
2. **Position commands** — `sit`, `rest`, `sleep`, `wake`, `stand`
   (`src/act.other.c` `do_sit`/`do_rest`/`do_sleep`/`do_wake`/`do_stand`): a
   small state machine with position gates and terminal + room-audience bytes.
   New manifest `position.tsv` (or one per command).
3. **Object disposal** — `junk`, `donate`, `sacrifice` (`src/act.item.c`):
   item removal, room/actor messages, donation-room routing.
4. **Combat entry gates** — `kill`/`hit`/`murder` openers, `assist`, `rescue`
   (`src/act.offensive.c`): the *entry gates and messages only* (D1–D3); delegate
   the damage matrix + RNG to a combat manifest as `delegated`/`blocked`.
5. **Info displays** — `time`, `weather`, `where`, `score`, `who` (`src/act.
   informative.c`): deterministic rendered text; watch `DP_CLOCK`/`DP_FIXED_TIME`
   for time/weather.
6. **Communication audiences** — `say`, `tell`, `whisper`, `emote`
   (`src/act.comm.c`): multi-audience `$n`/`$N` substitution (rich D3 material).

## Hard rules (do not violate)

- **Never edit `src/`** — it is the C reference. Fixes are Go-only (`pkg/`, `cmd/`).
- **Never merge a PR.** Open it and stop; Claude/Daeron reviews and gates.
- **Branch from `main` explicitly** each round: `git checkout main && git pull
  --ff-only && git checkout -b glm/depth-<command>`. After the round, `git status
  -sb` — do **not** leave HEAD on a feature branch or stack the next round on it.
- **No inventions** (R4): if C emits nothing, Go emits nothing — prove the
  *absence* of an invented message with a unit test where bytes are silent.
- **RNG parity is verification, not vibes** (R3): equal outcomes across one seed
  prove nothing; vary `--seed` and expose the draw. If you cannot establish draw
  parity, mark `blocked` — never fake green.
- Do not convert `fmt.Fprintf` MUD output to `slog`; do not remove `CustomData`.

## References

- `docs/fidelity/DEPTH_TESTING.md` — the methodology (authority).
- `docs/fidelity/RULEBOOK.md` — rules R1–R5e (cite by number).
- `docs/briefs/README.md` — brief/PR conventions, review checklist.
- Model manifests: `docs/fidelity/depth/drop.tsv`, `movement.tsv`, `quaff.tsv`.
- Model scenarios: `cmd/dp-oracle-diff/scenarios/drop-basic.txt`,
  `quaff-effect.txt`, `recite-effect.txt`.
