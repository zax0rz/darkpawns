# Depth-fidelity handoff — 2026-09-02 — `scrounge`

## Session result

The `scrounge` command-family slice is complete. Its feature PR #1145 was
hosted-green and self-merged into `main` as `929a80630`
(`fix: prove scrounge depth fidelity (R1/R3/R5e)`). The handoff is being
written from a clean, synchronized `main` checkpoint before the next slice.

The special-procedure inventory remains exhausted. The one blocked
`objmagic.sleep-entry-gates` row remains unchanged; its single cast-sleep
vehicle was already attempted in the prior queue phase.

## C path audited

- Registration: `src/interpreter.c:679`, `{ "scrounge", POS_STANDING,
  do_scrounge, 0, 0 }`.
- Handler: `src/new_cmds2.c:64-135`.
- The path was verified in source before changing Go: zero-skill refusal and
  two-round wait; mounted refusal before the roll; inclusive
  `number(1,101)`; forest/field/hills/desert/water capture branches; mountain
  find branch; default-sector refusal with the unconditional room search act;
  success/failure waits; object acquisition; and post-success
  `improve_skill` ordering.
- Object prototypes were verified in the read-only world data: vnums 27–31
  are fish, small game, field mouse, sandworm, and edible berries.

## Proof and implementation

- Added `docs/fidelity/depth/scrounge.tsv` with 22 explicit D1–D5 cases.
- Added ten isolated oracle vehicles covering the gates, default sector,
  failure, every sector family, room audience, and object wording.
- Clean-main RED evidence was captured before the Go fix. Confirmed
  divergences were the wrong sector constants, wrong capture/find text,
  missing unconditional room act, missing waits, missing deferred
  improvement draw, and incorrect `number(1,100)` bound.
- Added focused Go tests for the C entry gate, sector/object mapping, audience
  text, gate ordering, cooldowns, and deferred RNG draw parity.
- The five-seed matrix (`1,2,3,5,8`) was green across all ten vehicles; the
  forest run was also inspected with `--show-oracle`.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Gates and frontier

All required local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
git diff --check
```

Final manifest report for this session:

```text
Cases: 3147 total, 3069 proven/delegated, 26 blocked, 52 excluded
Actionable completion: 3069/3095 = 99.2%
```

The next unclaimed, un-manifested interpreter family is `sell` at
`src/interpreter.c:680`, registered as `POS_STANDING` with `do_not_here`.
`scream` precedes it in the table but has no own manifest because its shared
social behavior is already covered by the existing social evidence, as
recorded by the prior `scratch` handoff. The next session must start with
`main` checkout/pull, `make fidelity-depth`, a fresh read of
`docs/fidelity/DEPTH_TESTING.md`, and this newest handoff, then audit and take
`sell` without re-picking any claimed item.

— R1/R2/R3/R4/R5e; shared-callee delegation follows R5b/R5c.
