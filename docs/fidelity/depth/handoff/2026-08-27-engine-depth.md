# Dated Handoff: 2026-08-27 (engine depth and special-procedure scout)

This wave began from clean `main` at `668e895e1`, after the socials,
channels, and closer wave. The starting depth checkpoint was **513 total, 498
proven/delegated, 5 blocked, 10 excluded**. Two rounds completed, one PR per
round, with each green PR self-merged after hosted test, lint, and security
checks passed. The final checkpoint is **515 total, 501 proven/delegated, 4
blocked, 10 excluded**; actionable completion is **501/505 = 99.2%**.

## Round 1 — deferred extraction and menu return (PR #683)

Branch `glm/depth-deferred-extraction`; commit `726b40bd8`; merged as
`f8d1f53fd`.

- C tracing followed `do_kill()`'s implementor arm into `raw_kill()` in
  `src/act.offensive.c:145-164` and `src/fight.c:534-577`, then through
  `extract_char()`/`extract_char_final()` and
  `extract_pending_chars()` in `src/handler.c:1088-1248`. `heartbeat()` calls
  the pending drain in `src/comm.c:805`, and `extract_char_final()` moves a
  connected victim's descriptor to `CON_MENU`.
- The required RED on clean `main` was `kill chopvict` followed by
  `~dpclock pulse 1`: C emitted the main menu on the victim connection while
  Go emitted nothing. The normal `DoSpellDamage` → `HandleDeath` mortal path
  was audited and left outside this focused immortal `raw_kill` row; it uses
  the existing mortal death/respawn contract and was not silently broadened.
- Go now queues an instakilled player for the next heartbeat, drains the
  queue after event processing, removes the player from the world, and returns
  a connected session to the existing main menu. Menu option 1 restores the
  C-shaped minimum playable state and level-based start room.
- `kill.immortal-postdeath-menu` is green in `combat-entry.tsv`; the live
  `combat-entry-instakill` vehicle is green for seeds 1, 2, 3, 5, and 8.
  The full combat-opener sweep was serialised because concurrent C oracle
  processes collide on the shared WHOD port; all isolated opener vehicles
  completed without normalized divergence.

## Round 2 — special-procedure inventory and deterministic arms (PR #684)

Branch `glm/depth-spec-procs`; commit `d9cf0b84c`; merged as `ee0ddbacd`.

- The C census in `docs/fidelity/depth/spec-procs-scout.md` covers all three
  `src/spec_procs*.c` files: 113 `SPECIAL` bodies (`41 + 43 + 29`), 233
  active `ASSIGNMOB` calls, 228 unique mob VNums, and 66 unique mob proc
  names. Overlapping trigger tags count 45 fight, 26 greet/entry, 34
  percent/random, 4 explicit timer/clock, 57 named command/direction, and
  72 pulse/no-command gates.
- `getMobVNumSpec` in `pkg/game/mobact.go` and the command-side `GetMobSpec`
  path were inventoried. Go has 228 mob assignments and all 66 assigned names
  registered. C-last-assignment comparison found and corrected the effective
  map drifts `8014 → guild` and `11024 → cleric`; a unit guard preserves both.
- The required simple live vehicle is `spec-proc-movement`: C's assigned mob
  2106 (`no_move_west`) and mob 14410 (`no_move_east`) emitted act pairs while
  Go emitted invented “heavy object” text. Go now routes the C act pair through
  the canonical audience-aware `Act` primitive. Seeds 1, 2, 3, 5, and 8 are
  green.
- `docs/fidelity/depth/spec-procs.tsv` owns the two new D3 cases. No
  mob-caster vehicle emerged, so `hit.charm-master` and
  `assist.mob-helpee-pers` remain blocked with their combat-entry owner.

## Remaining blocked frontier and owners

The four blocked rows are intentional proof gaps, not exclusions:

- `combat-entry.tsv:hit.charm-master` — combat-entry owner; needs a charmed
  attacker/master relation vehicle.
- `combat-entry.tsv:assist.mob-helpee-pers` — combat-entry owner; needs a mob
  helpee vehicle proving C `PERS` versus Go short-description rendering.
- `info.tsv:score.state-variants` — info owner; affect, mount/pet, tattoo,
  and naked/armed branches remain unproven after the position subset.
- `object-magic.tsv:objmagic.sleep-entry-gates` — object-magic owner; the
  quaff entry remains unreachable because sleep is `TAR_NOT_SELF`; the
  reachable cast surface is separately proven.

The explicitly deferred deep-engine backlog was not attempted beyond the
requested deferred extraction row and the special-procedure scout. No C or
oracle files were edited.

## Verification

- `make fidelity-depth` — **515 total, 501 proven/delegated, 4 blocked, 10
  excluded**; exit 0.
- `go build ./...` — pass.
- `go vet ./...` — pass.
- `go test ./...` — pass.
- `golangci-lint run ./...` — **0 issues**.
- `gofumpt -l .` — no output.
- PR #683 — hosted test, lint, and security checks passed; self-merged.
- PR #684 — hosted test, lint, and security checks passed; self-merged.

Rules applied: R1 player-facing bytes, R2 command surface, R3 deterministic
draw/state parity, R4 no invention, and R5/R5e C call-path verification.
