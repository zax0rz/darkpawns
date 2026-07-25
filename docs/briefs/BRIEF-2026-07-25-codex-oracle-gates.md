# BRIEF (codex) — Oracle gates: close DP-1215 / DP-1200 / DP-1205 (R5a)

**Owner:** codex (gpt-5.6-sol). **Gate:** every scenario below RED on the
pre-fix commit → GREEN on current `main`, with transcripts to prove it.
**Git:** branch `codex/oracle-gates` **from `main`** — never from whatever
HEAD is checked out (the orchestrator pre-creates the branch or hands you a
`git worktree`; see the warning in `docs/briefs/README.md`). New scenario
files + the run report are the deliverable: commit → push → open a PR. Do NOT
merge. When done (including revise rounds), `git checkout main`.
**Context:** three fidelity fixes merged this week. Per **R5a** none closes
until the oracle confirms byte-parity. You have the oracle environment
(`DP_ORACLE_BIN`); the rest of the fleet does not.

**Read first:** `cmd/dp-oracle-diff/README.md` (harness + invocation),
`docs/fidelity/RULEBOOK.md` (R1–R5; cite by number), and the two model
scenarios named below — copy their structure exactly.

Invocation, from the repo root:

```bash
DP_ORACLE_BIN=/path/to/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario <name>
```

---

## Lane 1 — DP-1215 (melee-round parity, merged #479 `07d1e4e8`)

The `damage()`→`skill_message` reroute: opener skills now emit the C message
set and consume the C dice draws. Three opener scenarios gate it.

1. **`combat-bash-opener`** — exists; its header says "Expected to be RED".
   Confirm: RED on `07d1e4e8^` (pre-fix), GREEN on `main`.
2. **Write `combat-trip-opener.txt`** — mirror `combat-bash-opener.txt`
   (same fixture stack: `empty-players`, God+skillset warmup, mortal peer at
   8105, `~dpclock pulse` drains). Probe: `trip trainee`, warmup
   `skillset <mortal> 'trip' 75`. **R5e:** read `do_trip` in
   `src/act.offensive.c` yourself for preconditions (position, move cost,
   WAIT_STATE) — do not trust this summary; adjust the probe if C's gating
   differs from bash's.
3. **Write `combat-headbutt-opener.txt`** — same pattern, `headbutt trainee`,
   `skillset ... 'headbutt' 75`; verify `do_headbutt` in C the same way.

Anchors that must stay green: `combat-swing`, `combat-backstab-opener`,
`combat-unlearned-bash-opener`, `combat-death`.

**DP-1210 check:** if trip/headbutt are GREEN post-#479 with no further Go
change, say so explicitly — DP-1210 is the same stand-up-round class and
closes on your evidence. If either is RED, capture the divergence transcript;
that becomes the next executor brief.

## Lane 2 — DP-1200 (NOPERSON usage-line removal, merged #480 `5b03c3ff`)

**Write `object-give-nobody.txt`** — model on `object-give-mob.txt` (Human
Warrior, hometown K, `recall` to 8004; no mob fixture needed). Probe:
`give sword nobody` (starter sword vnum 8037; victim name matches no one).
Both servers must emit `No-one by that name around.` — pre-#480 Go printed a
usage line instead (R4). RED on `5b03c3ff^`, GREEN on `main`.

## Lane 3 — DP-1205 (immortal start room, PR #481)

**Run `god-harness-smoke`** — the scenario that surfaced the bug. Pre-fix the
first-player God spawned at 8099 with mortals; now it must enter at 1204 and
the mortal's `look` must not show the God co-located. RED on the commit
before #481 merges, GREEN on `main` after. If #481 is not yet merged when you
start, run this lane last — the orchestrator will confirm the merge.

Anchors that must stay green: `character-creation`, `newbie-start-room`,
`look-start-room`.

## Report (the PR body)

Per scenario: commit tested, RED/GREEN pre→post, and for any still-RED
scenario the full divergence transcript. Close with a per-issue verdict line
suitable for pasting into Linear: DP-1215, DP-1210, DP-1200, DP-1205 →
oracle-confirmed / still divergent (with evidence). Name any NEW divergences
you spot along the way (R5c: find one, find the class) — report, don't fix.

## Guardrails

- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/misc/messages` — read-only.
- Scenario files are shared line sequences: no server-specific steps, no
  expected-output branches (README §Scenario format). Use `[fixture]`,
  `[setup:oracle]` / `[setup:port]`, `[warmup]` (drained), `[probe]` exactly
  as the model scenarios do.
- If a scenario diverges for harness reasons (creation-flow drift, timing),
  iterate on the scenario until the divergence is attributable to the bug
  class under test — say which in the report.
- All gates (AGENTS.md §Build & Verify): build, vet, `test ./... -race`,
  `golangci-lint run`, `gofumpt -l .` empty. (`dp-oracle-diff` skips
  gracefully without `DP_ORACLE_BIN`, so CI stays green either way.)
