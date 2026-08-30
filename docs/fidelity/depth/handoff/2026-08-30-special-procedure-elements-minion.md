# Depth-fidelity handoff — `elements_minion` — 2026-08-30

## Frontier and queue position

- Started from a clean, freshly pulled `main`; the post-merge `make fidelity-depth` report is **1232 total, 1186 proven/delegated, 13 blocked, 33 excluded**, actionable **1186/1199 (98.9%)**.
- Consumed the next active special-procedure slice in C file/registration order: `SPECIAL(elements_minion)` at `src/spec_procs3.c:1217-1240`, actively registered by `ASSIGNMOB(1313, elements_minion)` at `src/spec_assign.c:195`.
- Next active special: `elements_guardian`, registered by `ASSIGNMOB(1314, elements_guardian)` at `src/spec_assign.c:196`, defined immediately after the minion procedure at `src/spec_procs3.c:1242+`.

## C call path and observable contract

The procedure is reached from the player-command mobile-special loop in
`src/interpreter.c:1407-1456`, where `ch` is the player and `me` is the
registered mob, before ordinary `give` handling. It is also reached from the
autonomous `MOB_SPEC` path in `src/mobact.c:68-93`, where C passes the mob as
both `ch` and `me` with command `0` and an empty argument. The body ignores the
command text, scans `mobile->carrying` in the ordered keyword passes
`talisman`, `element`, `earth`, `fire`, `water`, `air`, and for each pass takes
the first visible matching object. Each match emits the exact room Act
`"$n utters the words 'eradico paratus' and $p disintegrates."`, fully extracts
the object, invokes the shared `elements_remove_cylinders(mobile)` helper, and
continues. It returns `FALSE` after the full scan, including the commandless
autonomous path.

## RED → GREEN

- Vehicle: `cmd/dp-oracle-diff/scenarios/spec-proc-elements-minion.txt`.
- The initial RED exposed two confirmed Go mismatches: mob targeting used the
  rendered short description instead of C's authored keyword aliases, and the
  existing minion implementation dereferenced `ch` during autonomous pulses,
  classified only vnums 1300-1307, destroyed all matches in one pass, used the
  wrong room/audience helper, and left extracted objects in the active registry.
- The vehicle now gives vnum 1300 to the registered mob through the command
  dispatcher, then uses a frozen `~dpclock pulse 40` to reach the autonomous
  mobile-activity path. The C and Go transcripts are GREEN for seeds **1, 2,
  3, 5, and 8**.
- Focused proof in `pkg/game/spec_elements_minion_test.go` covers ordered
  visible-keyword passes, arbitrary-vnum classification, invisible-object
  filtering, room-wide audience, full extraction, commandless nil-player
  safety, and authored mob aliases.

## Go changes

- Replaced the vnum-only bulk deletion with C's six ordered visible-keyword
  passes, canonical room `Act`, full `ExtractObject`, and per-match shared
  cylinder cleanup.
- Updated `FindMobInRoom` to match C's authored mob keyword namelist, which the
  live `give` vehicle confirmed was required for the registered minion aliases.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Manifest, gates, and PR

Added seven rows to `docs/fidelity/depth/spec-procs.tsv` for command entry,
command fallthrough, autonomous dispatch, keyword predicate, destruction
audience, extraction, and delegated cylinder cleanup. The rows cite R1/R2/R3/R4/R5e;
the shared cleanup delegation also cites R5b/R5c.

Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, `gofumpt -l .` clean, and
`git diff --check`.

Branch `glm/spec-elements-minion` was committed as `fa3baee8f`; PR **#789**
passed `test`, `lint`, and `security` (build/deploy skipped) and was self-merged
into `main` as `f3ad7d8e29`.

## Next action

Start the next session from `main`, pull, run `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this handoff, then map and prove
`elements_guardian` in C file/registration order. Continue the special-
procedure inventory, then attempt the one blocked `objmagic.sleep-entry-gates`
row through the cast-sleep outlaw/reagent vehicle before sweeping the remaining
un-manifested `src/interpreter.c` command families. Leave one dated handoff per
session.
