# BRIEF (zcode / GLM-5.2) — Oracle coverage expansion (autonomous `/goal`)

**Owner:** zcode (zai/glm-5.2), self-directed via `/goal`. **Delivered by Zach through the zcode GUI.**
**Mission:** drive oracle differential-test coverage across the command surface — move every
**registered, non-social** command from `never-probed` to `probed`, one small cluster per PR.
**Metric:** `make scenario-coverage` → `docs/reports/scenario-coverage-<date>.tsv`, the `coverage`
column. **Baseline (2026-08-11):** ~62 `probed` / ~446 `never-probed`.
**Gate law (R5a):** a command is "covered" only when its `cmd/dp-oracle-diff` scenario runs GREEN
against the C oracle — with **no invented behavior to force the green**. C is the authority (R2).

This follows the reachability effort (now done: PR #503 wired 19 commands, `dig` left as an honest
DP-1225 gap). Reachability proved the pattern: **a generated metric with a hard target, ground it
down cluster by cluster.** `make scenario-coverage` is that metric for the oracle layer.

---

## Read first (do not skip)

- `AGENTS.md` — Prime Directive (Fidelity Law).
- `docs/fidelity/RULEBOOK.md` — R1–R5; cite by number. Especially **R5a** (oracle-green = done),
  **R5c** (find-the-class), **R5d** (flag uncertainty, don't fudge), **R5e** (verify the call path).
- `GLM.md` — your operating manual (brief workflow, gate chain, R5 lessons).
- `docs/fidelity/oracle-differential-testing.md` and `cmd/dp-oracle-diff/README.md` — harness + invocation.
- `docs/briefs/README.md` — git isolation discipline and the review checklist.

## The loop (repeat; each iteration = one PR)

1. **Pick a batch.** Run `make scenario-coverage`. Choose the next never-probed, **registered,
   non-social** commands, **clustered by shared C handler / source file** (commands that share a
   `do_*` function test together and reuse fixtures). Keep each PR to one coherent cluster.
2. **Author scenarios.** For each command, read its C handler in `src/` and the Go port. Add a
   scenario to `cmd/dp-oracle-diff/scenarios/<name>.txt` using the existing format:
   `[fixture]` / `[setup:oracle]` / `[setup:port]` / `[setup:*:observer]` / `[probe]` /
   `[probe:<peer>]` / `[warmup]`. **Copy creation sequences from `communication.txt` verbatim** —
   C rejects some invented player names. Immortal-gated commands use the God harness
   (`DP_FRESH_MUD`, `empty-players` fixture, `skillset` warmup, `[probe:<peer>]`).
3. **Run the oracle:**
   ```bash
   DP_ORACLE_BIN=/Users/zach/.openclaw/workspace/darkpawns-c-oracle/bin/circle DP_SEED=1 \
     go run ./cmd/dp-oracle-diff --scenario <name>
   ```
   (~40s; **exit 0 even on divergence** — read the `result:` line.)
4. **Resolve:**
   - **GREEN** → coverage win; that command is done (R5a).
   - **RED** → a real fidelity divergence. Diagnose the **class** (R5c), then **fix the port to
     match C**. NEVER invent behavior or pin output to force green (R2/R5a). If the fix is deep or
     C's intent is genuinely ambiguous, **stop forcing it**: file a DP issue with the diff evidence
     (R5d) and leave that command flagged rather than shipped wrong.
5. **Gate + ship.** `gofumpt -w` every Go file you touched; `make lint` and `go test ./...` clean;
   report the probed-count delta in the PR body (the `scenario-coverage-*.tsv` is gitignored —
   do NOT commit it; it serializes every PR on one generated file); open the PR. **Do not merge; do not
   self-approve** — surface findings/uncertainties as Linear DP comments.

## Hard rules

- **Fidelity Law is supreme.** A red oracle means the **port** is wrong until proven otherwise.
  Oracle-green with no invented behavior = done.
- **Git discipline** (see `docs/briefs/README.md`): **rebase onto `origin/main` before every new
  branch** (recurring hazard: branching off a diverged local main); branch **from `main`**; verify
  `git rev-parse --abbrev-ref HEAD` before any commit; one cluster per PR; commits only your own.
- **Additive only.** New scenarios must not weaken existing ones to make a probe pass.
- **CI:** the reachability ratchet must not regress (`make reachability`); lint runs `gofumpt` +
  `golangci-lint`; tests run `-race`.

## Batch-1 focus — momentum first (Zach's call, 2026-08-11)

Start with **deterministic, low-RNG mortal informative/status commands** (fast greens, no combat
wait-state) to build probed-count velocity and prove the loop this session — the `score` / `time` /
`weather` / `equipment` / `inventory`-adjacent informative handlers and `do_gen_ps`-style status
commands. Then broaden into the immortal / God-harness clusters. Probe the recently-wired PR #503
commands early — they are fresh territory and back-test the wiring (`freeze`, `thaw`, `pardon`,
`notitle`, `mute`, `uptime`, `qsay`, `qecho`, `socials`, `wizhelp`, `poofin`, `poofout`, `finger`,
`gold`, `murder`, `rsay`, `shadow`, `kabuki`). *(An oracle scenario over `qsay` would have caught the
`PLR_NOSHOUT` bug found in the #503 review — coverage here directly defends the wiring work.)*

## Stop and surface when

A divergence points at a **rules ambiguity** (not a port bug); a cluster needs **harness capability
that doesn't exist yet**; or you've cleared a natural batch — report progress (probed-count delta +
PR links) and await direction.

---

## The `/goal` prompt (paste this into zcode)

```
/goal Expand oracle coverage across the command surface. The metric is
`make scenario-coverage` (docs/reports/scenario-coverage-<date>.tsv): drive the
`coverage` column from `never-probed` to `probed` for every REGISTERED,
NON-SOCIAL command, one small cluster per PR, until the registered non-social
surface is fully probed or you hit a blocker worth surfacing. Baseline today is
~62 probed / ~446 never-probed.

READ FIRST (do not skip): AGENTS.md (Prime Directive — Fidelity Law), RULEBOOK.md
(R1–R5, esp. R5a oracle-green=done, R5c find-the-class, R5d flag-uncertainty,
R5e verify-call-path), GLM.md (your operating manual), docs/fidelity/
oracle-differential-testing.md, and docs/briefs/README.md (git discipline).
The full charter is docs/briefs/BRIEF-2026-08-11-oracle-coverage-expansion.md.

THE LOOP (repeat; each iteration = one PR):
1. Run `make scenario-coverage`. Pick the next never-probed, registered, non-social
   commands — CLUSTER by shared C handler / source file. Start with deterministic,
   low-RNG informative/status commands for momentum; probe the recently-wired #503
   commands early (freeze, thaw, pardon, notitle, mute, uptime, qsay, qecho, socials,
   wizhelp, poofin, poofout, finger, gold, murder, rsay, shadow, kabuki).
2. For each command: read its C handler in src/ and the Go port. Author a scenario
   in cmd/dp-oracle-diff/scenarios/<name>.txt using the existing format
   ([fixture]/[setup:oracle]/[setup:port]/[setup:*:observer]/[probe]/[probe:<peer>]/
   [warmup]). Copy creation sequences from communication.txt VERBATIM (C rejects some
   invented names). Immortal-gated commands: God harness (DP_FRESH_MUD, empty-players
   fixture, skillset warmup, [probe:<peer>]).
3. Run:
   DP_ORACLE_BIN=/Users/zach/.openclaw/workspace/darkpawns-c-oracle/bin/circle DP_SEED=1 go run ./cmd/dp-oracle-diff --scenario <name>
   (~40s; exit 0 even on divergence — READ the `result:` line.)
4. GREEN → coverage win, done (R5a). RED → real fidelity divergence: diagnose the
   CLASS (R5c), fix the PORT to match C — NEVER invent behavior or pin output to force
   green; C is authority (R2). If deep/ambiguous, STOP forcing it: file a DP issue with
   diff evidence (R5d) and leave that command flagged rather than wrong.
5. gofumpt -w every Go file touched; `make lint` and `go test ./...` clean; update the
   probed-count delta in the PR body (scenario-coverage-*.tsv is gitignored — do NOT commit it);
   open the PR.

HARD RULES:
- Fidelity Law is supreme. Red oracle = the PORT is wrong until proven otherwise.
- REBASE onto origin/main before every new branch (recurring diverged-local-main
  hazard). Branch from main. Verify `git rev-parse --abbrev-ref HEAD` before any commit.
- One coherent cluster per PR. New scenarios are additive — never weaken existing ones.
- Do NOT self-approve or merge; surface findings/uncertainties as Linear DP comments.

STOP AND SURFACE when: a divergence points at a rules ambiguity (not a port bug), a
cluster needs harness capability that doesn't exist yet, or you've cleared a natural
batch — report the probed-count delta + PR links and await direction.
```
