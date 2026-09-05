# 2026-08-29 — `stableboy` depth slice

## Frontier and queue

- Continued on `glm/spec-stableboy` from the pulled `main` frontier, after
  running `make fidelity-depth` and rereading `docs/fidelity/DEPTH_TESTING.md`
  and the newest dated handoff.
- The pre-slice frontier was 980 total cases: 946 proven/delegated, 12
  blocked, and 22 excluded. This slice adds 21 stableboy cases and one
  source-inventory exclusion, yielding 1002 total, 967 proven/delegated, 12
  blocked, and 23 excluded; actionable completion is 967/979 (98.8%).
- `couch` was verified as unassigned under R5e in the correction handoff;
  it was not repicked. `tipster` is likewise recorded as excluded because its
  C body has no actual `ASSIGNMOB` registration. The next active source-order
  special is `rescuer`, defined at `src/spec_procs2.c:523` and registered for
  mob vnums 7909 and 15808 in `src/spec_assign.c`.
- The unrelated untracked brief
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

## C call path and branch census

- `SPECIAL(stableboy)` is defined at `src/spec_procs2.c:315-461` and assigned
  to mob vnum 8022 at `src/spec_assign.c:282`. Its player-command path is
  `src/interpreter.c:1407-1456`, where the room's stablehand is scanned after
  room, equipment, and inventory specials. It returns TRUE for `list`, all
  `buy` outcomes, all `stable` outcomes, and all `collect` outcomes; unrelated
  commands return FALSE to ordinary command dispatch.
- The `buy` path first requires the post-`skip_spaces` argument to equal
  exactly `horse`, then checks `num_followers(ch) >= GET_CHA(ch)/2` and emits
  direct `stc` bytes before the gold gate. A successful purchase reads horse
  vnum 8021, charms/follows it, sets carry and movement state, broadcasts the
  room act, deducts 300 gold, and sends the stablehand tell.
- `list` emits one direct tell. `stable` has a mounted branch that unmounts and
  stops the follower, plus an unmounted follower scan using `IS_MOUNTABLE`; an
  absent mount emits the direct rejection, while success stores the mount
  vnum, current time, and five-gold daily rent before extracting the horse.
- `collect` gates on a nonzero stored mount, computes integer days with a
  minimum of one, rejects insufficient gold without clearing state, then reads
  the stored horse, clears rent state, restores charm/follow and movement
  state, deducts rent, broadcasts the room act, and tells the actor. A failed
  read has a separate direct error tell. All direct tells are actor-only; the
  room act is peer-visible and actor-excluded.

## RED/GREEN evidence and port result

- The first valid live vehicle used `spawn-mob 8022 1 8105 80`, stripped the
  Lua script, and teleported the actor/peer into the disposable room. The
  intended C stablehand instance was visible. The RED probe proved that Go's
  `strings.Contains` accepted `horseman` and `horse extra`, while C returned
  `Buy what, fine adventurer?`; the exact `Horse` case also remained rejected.
- The known-mob control in the same room/zone proved the spawn vehicle before
  the stablehand run. Earlier attempts without the valid room warmup were not
  used as behavior evidence.
- The funded two-client vehicle then exposed the missing C follower start/stop
  audience bytes, the extra generic `a horse appears.` spawn line, lowercase
  room-action substitution, and premature gold deduction on failed horse
  loading. The Go fix in `pkg/game/spec_procs2.go` now requires exact `horse`,
  checks `World.NumFollowers(ch.Name)` against `GetCha()/2` before gold, emits
  the C direct actor-only follower-cap line, uses silent read-mobile placement,
  routes C `add_follower`/`stop_follower` audience notices, and preserves C
  transaction ordering. `pkg/game/spec_stableboy_test.go` covers exact
  argument/cap gates, autonomous entry, mountable and mounted stable branches,
  load failures, affordability, purchase, stable, and collect state.
- GREEN live proof:
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario spec-proc-stableboy --show-oracle`
  reports `result: no normalized divergence`. The vehicle proves list,
  exact-buy gates, affordability, empty stable/collect gates, audience
  routing, and FALSE fallthrough. Focused unit proof covers the silent state
  cycle, failure branches, autonomous entry, and both stable follower modes.
- GREEN funded live proof:
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario spec-proc-stableboy-success --show-oracle`
  reports `result: no normalized divergence` for successful buy, stable, and
  collect action/follower/tell ordering across the actor and peer.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification and integration

- The slice must pass `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, and clean `gofumpt -l .` before
  review. The stableboy focused test and live oracle vehicle are already
  green; full gates and GitHub checks remain required before merge.
- This slice applies R1 (exact player-facing bytes), R2 (command and FALSE
  fallthrough surface), R3 (rent/state arithmetic), R4 (no invented output),
  and R5/R5e (actual registered call path and C-first verification).

## Next queue item

Continue the active special-procedure inventory with `rescuer` in source and
registration order (`src/spec_procs2.c:523`, mob vnums 7909 and 15808). Do not
repick `stableboy`, `tipster`, `couch`, `whirlpool`, or earlier claimed
procedures. After the active inventory is exhausted, attempt the single
blocked `objmagic.sleep-entry-gates` vehicle, then sweep remaining
un-manifested command families in `src/interpreter.c` table order.
