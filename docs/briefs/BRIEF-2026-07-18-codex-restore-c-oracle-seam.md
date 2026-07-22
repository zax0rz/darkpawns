# BRIEF (codex) — durably RESTORE the wiped C-oracle determinism seam (DP_SEED + DP_CLOCK + settle-pump)

**Owner:** codex. **Gate:** Claude (workers have no `DP_ORACLE_BIN`; Claude runs the red→green + determinism check). **Two commit targets — see "Durability".** This is a *restoration to a known-good pre-wipe state*, not a new feature. The Go side is already committed and intact — **do NOT touch the Go repo's runtime code**; this is C-oracle + harness-tooling only.

## What happened (context — read, don't re-derive)
The C oracle at `~/.openclaw/workspace/darkpawns-c-oracle` had a determinism seam (getenv-gated `DP_SEED` + `DP_CLOCK` freeze + a `~dpclock pulse N` settle-pump). It was never committed to that repo — it lived as **uncommitted working-tree edits** — and a broad `git checkout -- src/` (or re-clone) during instrumentation cleanup **wiped all of it**. Verified 2026-07-18: `src/comm.c:263` is pristine `prng_seed(time(0));`, no `getenv`/`DP_SEED`/`DP_CLOCK` anywhere in `src/`, the binary has no seam strings, git is clean, reflog = single clone. Consequence: the harness passes `DP_SEED=1 DP_CLOCK=1` to a binary that **ignores both**, so C is non-deterministic run-to-run (measured: `hunger-thirst` C HP = 21/23/22 across three runs). Your job: rebuild the seam faithfully **and commit it so this can never happen again.**

The full behavioral spec already exists — implement from these, they are authoritative:
- `docs/briefs/DESIGN-2026-07-17-dp-clock-pulse-sync.md` (DP_SEED location, DP_CLOCK freeze at the `while (missed_pulses--) heartbeat(++pulse);` loop, heartbeat sub-activity dispatch order at `comm.c:810`).
- `docs/briefs/BRIEF-2026-07-17-codex-dpclock-settle-pump.md` (the `~dpclock pulse N` primitive: draw-neutral, pre-interpreter, fires `heartbeat(++pulse)` n times; the 40-pulse settle contract).

## Scope — three C seams to restore, in `~/.openclaw/workspace/darkpawns-c-oracle/src`

### 1. DP_SEED (comm.c, ~:263)
Currently `prng_seed(time(0));`. Restore the getenv gate: seed from `DP_SEED` when set, else `time(0)`. Must byte-match the Go semantics in `pkg/dprng/cmwc.go seedFromEnvironment` — Go parses `DP_SEED` as an unsigned 32-bit int (`strconv.ParseUint(value,10,32)`). Mirror that in C:
```c
char *dp_seed_env = getenv("DP_SEED");
prng_seed(dp_seed_env ? (uint32_t) strtoul(dp_seed_env, NULL, 10) : (uint32_t) time(0));
```

### 2. DP_CLOCK pulse-freeze (comm.c)
- Add a file-scope flag set once at init (next to the DP_SEED read): `int dp_clock = (getenv("DP_CLOCK") != NULL);` (presence, not value — matches Go's `internal/dpclock.Frozen()` which keys on `os.LookupEnv` presence).
- Gate the wall-clock pulse loop `while (missed_pulses--) heartbeat(++pulse);` (design note cites ~:688-689; locate the exact block) so that **when `dp_clock` is set, it does NOT fire heartbeats off wall-clock.** Under `dp_clock`, `heartbeat` fires *only* from the pump (seam 3). Byte-identical when unset.

### 3. `~dpclock pulse N` settle-pump (control primitive)
Per the settle-pump brief. A reserved telnet line intercepted **before** `command_interpreter`, active **only when `dp_clock` is set** (inert otherwise so production is untouched):
- Syntax: `~dpclock pulse <n>` (leading `~` = never a real command; keep C and Go byte-identical — the Go side already ships this).
- Behavior: fire `heartbeat(++pulse)` exactly `n` times, then return. **Draw-neutral itself:** do NOT route through `command_interpreter` (must not consume the per-command `number(0,3)` AFF_HIDE-clear at `interpreter.c:889`) and do NOT touch wait-state. Only the heartbeats it fires may draw.
- The pumped `heartbeat(pulse)` must dispatch sub-activities in C's existing order (`comm.c:810`+: `zone_update` → 15s → `mobile_activity`/`room_activity`/`object_activity` → `perform_violence` → `point_update` → …). You are **restoring** C's own `heartbeat` — so just make sure the pump calls the real `heartbeat(++pulse)`; do not re-order or trim C's dispatch (that Go-side concern doesn't apply to C).

**Explicitly OUT of scope** (do not add): freezing the boot *calendar* (`reset_time` / `db.c:415-420` still uses `time(0)` — that's a separate follow-up for the `time` command; no stat-reaching boot draw is calendar-gated, so DP_SEED alone restores stat determinism). No draw-site edits, no message-table edits, no combat-logic edits.

## Durability — the whole point of this brief (do BOTH)
The seam was lost because it was never version-controlled. Prevent recurrence:
1. **Commit the seam to the C-oracle repo.** After editing `src/`, in `~/.openclaw/workspace/darkpawns-c-oracle` create a branch `dp-oracle-seam` and commit the seam edits there (message: "DP determinism seam: DP_SEED + DP_CLOCK freeze + ~dpclock pulse settle-pump"). A committed seam means a future `git checkout -- src/` restores *to the seam*, not to pristine.
2. **Export a reproducible patch into the Go repo.** `git -C ~/.openclaw/workspace/darkpawns-c-oracle diff <pristine>..dp-oracle-seam -- src/ > tools/oracle-seam/dp-determinism.patch` and add a `tools/oracle-seam/README.md` documenting: the pristine base commit (`d2cb13e`), how to apply (`git apply` / `patch -p1`), and `cd src && make` to rebuild `bin/circle`. Commit these to `darkpawns_repo` — this is the reviewable PR artifact and the recovery path if the oracle is ever re-cloned fresh.

## Build / self-check (what codex CAN verify without the gate)
- `cd ~/.openclaw/workspace/darkpawns-c-oracle/src && make` → clean build, produces `bin/circle`.
- Sanity: with `DP_SEED` **unset**, the binary still boots (byte-identical-to-pristine behavior path). You cannot run `dp-oracle-diff` (no `DP_ORACLE_BIN` policy) — Claude gates determinism.
- The patch in `tools/oracle-seam/` must `git apply --check` cleanly against a pristine `d2cb13e` checkout.

## Acceptance (Claude-gated)
1. C source rebuilds clean; seam committed on `dp-oracle-seam` in the oracle repo; `tools/oracle-seam/dp-determinism.patch` + README committed in `darkpawns_repo`.
2. **Determinism restored:** Claude runs an RNG-exposing scenario (`hunger-thirst`) **twice** — C output is now byte-identical run-to-run (was 21/23/22). This is the primary proof the seam is back.
3. Full committed suite returns to its pre-wipe green set under `DP_SEED=1 DP_CLOCK=1`; the `hunger-thirst`/`guild-practice` +2 becomes a *stable* gap (real per-command draw diff, to be hunted separately — do NOT try to fix it here).
4. `DP_SEED`/`DP_CLOCK` unset ⇒ byte-identical to pristine on the oracle.

## Notes for the gate (Claude, not codex)
- After this lands, re-verify which prior "greens" were real vs coincidental-same-second, and re-baseline the sweep.
- Separately open: **Go also drifts run-to-run** (23/25/24) despite `DP_SEED=1` and its intact seam — lockstep with C's drift implies a shared wall-clock input on the Go boot path (goroutine draw-order race or a direct `time.Now()` draw). Investigate post-restoration; not codex's scope here.
