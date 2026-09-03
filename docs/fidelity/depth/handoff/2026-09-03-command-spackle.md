# Depth-fidelity handoff — `spackle`

Date: 2026-09-03

Branch: `glm/depth-spackle` (feature merged); handoff branch:
`handoff/2026-09-03-command-spackle`

Feature PR: #1275 (merged green)

Feature commit: `803bc61cc`

Main merge: `8783a278e`

## Queue position and result

This round checked out `main`, pulled with `git pull --ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter table in source order. The
special-procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed
cast-sleep outlaw/reagent vehicle and was not repicked.

The next genuinely unclaimed interpreter-table family after `split` was the
`do_action` social `spackle` at `src/interpreter.c:727`. The following
source-order row is the unmanifested social `spank` at
`src/interpreter.c:728`; confirm that claim from a fresh `main` checkout
before starting it. Do not repick `spackle` or earlier claimed families.

Pre-slice frontier: 3,826 total, 3,723 proven/delegated, 48 blocked, and 55
excluded. The spackle manifest adds 11 proven/delegated cases. Post-slice
frontier: 3,837 total, 3,734 proven/delegated, 48 blocked, and 55 excluded;
actionable completion is 3,734/3,782 = 98.7%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:727 */
{ "spackle"  , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` first resolves the social record,
checks `PLR_NOSHOUT`, consumes only the first target token with `one_argument`
when `char_found` exists, and then selects no-argument, missing-target,
self-target, or target-success branches. The `spackle` record at
`lib/misc/socials:1118-1126` has social level 1, hide 0, the default zero
victim-position minimum, and eight authored message slots.

The reachable player-visible branches are:

- no argument: actor `You inspect the walls for damage.` and observer
  `$n hauls out $s pail of spackle and looks for a crack to repair.`;
- visible target: actor, non-victim observer, and victim receive the three
  authored target-success lines;
- leading fill word and trailing words: C resolves only the first target token;
- self target: actor and room receive the authored self-target lines; and
- missing target: C sends the authored `not_found` string directly, including
  its literal `$N` token.

The command-level POS_RESTING gate, PLR_NOSHOUT refusal, zero victim-position
sleeping-target behavior, and visible lookup/Act audience filtering are shared
mechanics delegated to `fade.position-gate`, `dance-noshout`,
`slap.sleeping-target`, and `socials-depth` under R5b/R5c.

## Confirmed divergence and implementation boundary

The clean-main RED vehicle matched every spackle branch except missing-target
output. C emitted:

```text
You throw spackle all over in frustration because $N is not here.
```

while Go routed the message through `Act` with no victim and invented
`someone` for `$N`. C's source path sends `action->not_found` directly with
`send_to_char`, so the confirmed class-level fix changes the shared Go
`do_action` missing-target branch to direct `SendMessage` plus CRLF. It does
not alter any authored social data or invent any player-visible behavior.

Durable evidence:

- `cmd/dp-oracle-diff/scenarios/spackle-depth.txt`;
- `docs/fidelity/depth/spackle.tsv`; and
- `pkg/session/spackle_depth_test.go`.

The final `spackle-depth` vehicle reported `result: no normalized divergence`
at seeds 1, 2, 3, 5, and 8. Seed 1 was run with `--show-oracle` and showed
the intended C blocks for every annotated case. No file under `src/` or
`darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed after the manifest and focused record test were
added:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` clean
- `git diff --check`

PR #1275's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. CI fired normally, so no
workflow retry was used. The PR was self-merged only after all applicable
checks were green, per the 2026-08-27 amendment.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (stable audience matrix), R4 (no invented behavior), R5/R5e (verify the
actual C path and let C win), and R5b/R5c (shared social gate, target lookup,
and audience ownership).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and audit/claim
`spank` at `src/interpreter.c:728` before touching any implementation. Do not
repick `spackle` or any earlier claimed family.
