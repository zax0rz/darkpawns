# Depth-fidelity handoff: `insult`

Date: 2026-08-31

## Queue position and result

This session consumed the next unclaimed interpreter-table command after
`infobar`: `insult`, registered at `src/interpreter.c:518` as
`do_insult` at `POS_RESTING` with no level gate. The special-procedure
inventory remains exhausted, and the single blocked
`objmagic.sleep-entry-gates` row remains deferred; neither was repicked.

The pre-slice frontier was 2,294 total cases, with 2,234
proven/delegated, 16 blocked, and 44 excluded. After adding the 12 `insult`
rows, the frontier is 2,306 total, with 2,246 proven/delegated, 16 blocked,
and 44 excluded: 2,246/2,262 actionable cases, or 99.3%.

Feature PR #975 (`fix: depth-prove insult C fidelity`) passed hosted lint,
test, and security checks; build-and-push and deploy were skipped. It merged
to main as `55fadd0a4` (feature commit `89a61257f`).

The next unmanifested command in source registration order is `invis` at
`src/interpreter.c:519`.

## C call path and reachable branches

The authoritative path is `src/interpreter.c:518` →
`src/act.social.c:153-201` (`ACMD(do_insult)`) → `one_argument` →
`get_char_room_vis` → `send_to_char` and `act`. The command consumes only the
first argument token. Its reachable player-visible branches are:

- empty argument: `I'm sure you don't want to insult *everybody*...`;
- unresolved first token: `Can't hear you!`;
- visible self target: `You feel insulted.`;
- visible other target: actor confirmation `You insult %s.`, one
  `number(0,2)` draw, one of the four sex-matrix case-zero victim messages,
  the mother branch, or the get-lost branch;
- for a live other target, the target receives the selected insult and every
  other co-located character receives `$n insults $N.` through `TO_NOTVICT`.

The case-zero matrix is C's `SEX_MALE` comparison: male actor/male target
gets the fighting-like-a-woman line; male actor/nonmale target gets the
women-cannot-fight line; nonmale actor/male target gets the smallest-brain
line; and nonmale actor/nonmale target gets the beauty-contest line. Neutral
characters therefore follow C's nonmale branch. The case-one branch calls
the mother insult, and the default branch calls the get-lost insult.

## RED diagnosis and confirmed fix

On main-equivalent Go code, the live oracle exposed the sex branches as
reversed because `pkg/game/act_social.go` compared sex values to literal `1`
as though it meant male. Go's typed boundary constants encode `SexMale = 0`
and `SexFemale = 1` in `pkg/game/player.go`; C's `SEX_MALE` comparison is
therefore represented by `SexMale`, not the literal value `1` (R1/R3/R5e).

The only behavior fix changes those three comparisons to `SexMale`. No
`src/` or `darkpawns-c-oracle/` file was edited. The focused registration test
`TestInsultRegistrationUsesCEntryGate` also pins the Go entry at level 0 and
`POS_RESTING`, preserving the command surface under R2.

## Durable proof

The manifest `docs/fidelity/depth/insult.tsv` contains 12 rows covering the
entry gate, empty and invalid arguments, self-target, first-token parsing,
actor/victim/room audiences, all four sex-matrix case-zero outcomes, the
mother branch, and the get-lost branch. The durable vehicles are:

- `cmd/dp-oracle-diff/scenarios/insult-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/insult-male-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/insult-sex-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/insult-sex-female-depth.txt`.

The harness permits only three clients per source IP, so the sex/audience
coverage is split across four vehicles. Each vehicle was run with
`--show-oracle --seed 1`; every intended C block was reached and normalized
green. Each was then rerun at seeds 2, 3, 5, and 8. All 20 vehicle/seed runs
reported no normalized divergence, covering the deterministic draw sequence
and the actor, victim, and room-recipient split (R1/R3/R5e).

The handoff and manifest keep shared registration and dispatch ownership
explicit rather than duplicating it (R5b/R5c). No behavior was invented
under R4.

## Verification and continuation

Before the feature PR, the branch passed:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...`;
- `test -z "$(gofumpt -l .)"` and `git diff --check`.

After merging, clean main ran `make fidelity-depth` successfully at the
2,306-case frontier. Hosted checks for PR #975 were initially pending only
for test; the watcher later reported all three required checks green. No
merge was performed while checks were pending.

The next session must start from clean main, pull, reconfirm the frontier,
read the depth-testing guide and this newest handoff, then map and prove
`invis` in table order. Continue to leave one dated handoff per session.
