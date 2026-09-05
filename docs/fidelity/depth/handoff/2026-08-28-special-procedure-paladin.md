# Depth-fidelity handoff — paladin special procedure

Date: 2026-08-28  
Branch: `glm/spec-paladin`  
Starting main: `cea9d42bb` (fighter handoff)

## Queue position and inventory

The special-procedure inventory was refreshed from the registration tables in
`src/spec_assign.c` and the procedure definitions in `src/spec_procs.c`,
`src/spec_procs2.c`, and `src/spec_procs3.c`. The next source-and-registration
ordered unproven procedure after fighter was `paladin`, defined at
`src/spec_procs.c:537-568` and assigned to mob vnums 71 and 7915 at
`src/spec_assign.c:258,265`. Mob 71 (Death Dealer) was used for the fixed-room
vehicle; mob 7915 is the randload/scripted paladin and was also exercised with
its script removed where required.

Before this slice the frontier was 550 cases: 538 proven/delegated, 1 blocked,
and 11 excluded. This slice adds eight manifest rows. The resulting frontier is
558 total: 545 proven/delegated, 2 blocked, and 11 excluded; actionable
completion is 545/547 (99.6%).

## C path and branch claims

`SPECIAL(paladin)` is reached from the autonomous fighting path after
`perform_violence()` (`src/mobact.c:68-93`, `src/fight.c:1898-2032`). It
returns `FALSE` unless `cmd == 0`, the mob is `POS_FIGHTING`, `GET_HIT >= 0`,
and `FIGHTING(ch)` is present; it also returns `FALSE` when `GET_MOB_WAIT` is
nonzero. With `number(0,8)`, cases 0, 1, 2, 3, and 5 call native `do_parry`,
`do_bash`, `do_charge`, alignment-selected dispel, and `do_disarm`, all with
NPC subcommand 1. The other cases return `TRUE` without output.

The Go slice now mirrors those gates and dispatches. Parry and bash are
delegated to the already-proven fighter matrices. Charge follows
`src/new_cmds.c:880-955`: sword/lance gating, AC/NOBASH/mounted probability,
native dice and mount damage, zero-damage failure, sitting posture when
unmounted, and no `GET_MOB_WAIT` invention. Disarm follows
`src/new_cmds2.c:191-276`: mutual fighting and wielded-weapon gates, native
strict probability boundary, equipment movement, exact victim/room audiences,
and failure posture. Dispel selects `SPELL_DISPEL_GOOD` for evil paladins and
`SPELL_DISPEL_EVIL` otherwise, matching `src/spec_procs.c` and the native cast
path.

## Evidence

Focused RED/GREEN coverage is in `pkg/game/spec_paladin_test.go`:

- `TestSpecPaladin_Golden` proves command, position, HP, fighting, and mob-wait
  entry gates.
- `TestSpecPaladin_DefaultDispatch` proves a default random case returns true
  without invented bytes.
- `TestSpecPaladin_DispelAlignment` proves both native alignment branches.
- `TestSpecPaladin_ChargeNativeArithmetic` proves the charge weapon gate,
  probability inputs, sword/lance dice, failure posture, and wait-state
  behavior.
- `TestSpecPaladin_DisarmNativeAudienceAndState` and
  `TestSpecPaladin_DisarmFailure` prove the disarm state transition, exact
  audiences, and strict success boundary.

The live vehicle is
`cmd/dp-oracle-diff/scenarios/spec-proc-paladin.txt`. It reaches the fixed
mob-71 combat seam with `--show-oracle --seed 1`, but remains blocked in the
manifest row `mob.paladin-combat-action`: the C and Go transcripts diverge in
pre-existing high-level melee message ordering and content. The same RED was
reproduced from clean main at `cea9d42bb`; the alternate mob-7915 vehicle also
reached the seam but had the same unrelated combat-class mismatch. Two honest
vehicle attempts were therefore made, and the unrelated combat class is not
fixed forward in this slice. The paladin-only charge, disarm, dispel, default,
and entry branches are claimed only by their focused proofs; parry/bash remain
delegated to fighter.

## Verification and next queue item

`make fidelity-depth` passes at the frontier above. The complete repository
gates are required before the PR is opened: `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, and a clean `gofumpt -l .`.

The next action is the next source-and-registration ordered unproven special
procedure after `paladin`, determined from the refreshed inventory and prior
handoffs. The blocked `objmagic.sleep-entry-gates` row remains queued for its
single cast-sleep outlaw/reagent attempt after the special-procedure inventory
slice is exhausted.

Rules applied: R1, R3, R4, R5b, R5c, and R5e.
