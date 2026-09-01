# Depth-fidelity handoff — `nod`

Date: 2026-09-01

## Frontier

This session began from clean `main` after the `nibble` handoff at:

```text
Cases: 2659 total, 2589 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2589/2611 = 99.2%
```

The `nod` slice is now merged to `main` in PR #1046 (`f8b101e1b`). The
post-merge frontier is:

```text
Cases: 2670 total, 2600 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2600/2622 = 99.2%
```

The special-procedure inventory remains exhausted. The intentionally blocked
row `objmagic.sleep-entry-gates` remains blocked; its cast-sleep vehicle does
not prove the blocked direct-entry row. The interpreter sweep continues.

## Slice proof

The next source-order unclaimed row was `src/interpreter.c:568`:

```c
{ "nod"      , POS_RESTING , do_action   , 0, 0 },
```

The C call path was read first: `src/act.social.c:102-151`, with the authored
record in `lib/misc/socials:1044-1051`:

```text
nod 0 0
You nod.
$n nods.
You nod at $M.
$n nods at $N.
$n nods at you.
Who?
#
<blank others_auto record>
```

The reachable branches are: the POS_RESTING command gate; the shared
`PLR_NOSHOUT` refusal; no-argument actor and room bytes; `one_argument`
fill-word skipping and trailing-token discard; visible-target actor,
non-victim room, and victim delivery; missing-target refusal; the silent
self-target `char_auto=#` and blank `others_auto` branch; and the zero
`min_victim_position` sleeping-target branch. Shared position, emote-refusal,
target visibility, and Act delivery seams were delegated to established
manifests under R5b/R5c.

Go already had the matching `nod` social registration and shared `DoAction`
implementation. No behavior change was justified by the oracle comparisons
(R4).

Added:

- `cmd/dp-oracle-diff/scenarios/nod-depth.txt`
- `cmd/dp-oracle-diff/scenarios/nod-sleeping-depth.txt`
- `pkg/session/nod_depth_test.go`
- `docs/fidelity/depth/nod.tsv` (11 rows)

The normal vehicle used an actor, target, and observer in room 8162. Its
`--show-oracle --seed 1` run confirmed exact no-argument, target, self,
missing-target, and fill-word behavior. The sleeping vehicle put a target to
sleep and confirmed the actor and awake observer bytes while the target
received none. Both vehicles returned `no normalized divergence` for seeds 1,
2, 3, 5, and 8.

Focused tests passed, and all local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Hosted lint, security, and test checks for PR #1046 were green; build and
deploy were skipped by the workflow. The PR was merged only after required
hosted checks were green.

## Fidelity rules applied

- R1: preserved exact social bytes, pronoun substitutions, CRLF, and silent
  self-target output.
- R2: preserved the registered `nod` command and POS_RESTING gate.
- R3: repeated both observable vehicles across deterministic seeds 1, 2, 3,
  5, and 8.
- R4: made no Go behavior change without a confirmed divergence.
- R5b/R5c: delegated only the already-proven shared social seams after tracing
  their common call path.
- R5e: verified the actual C registration, dispatcher, and social record.

## Next queue item

The fresh source-order sweep confirms `noauction` at
`src/interpreter.c:569` is already owned by `gen-tog.tsv`; the intervening
`nobroadcast`, `noctell`, `nogossip`, `nograts`, `nohassle`, `nonewbie`,
`norepeat`, `noshout`, `nosummon`, and `notell` rows are likewise already
claimed by the generic-toggle family, while `noogie`, `nudge`, and `nuzzle`
belong to the existing social-family claim. The next genuinely unclaimed row
is `notitle` at `src/interpreter.c:580`. Map its `do_wizutil` path and use
branch `glm/depth-notitle`, one family PR, and one dated handoff. Do not
re-pick `newbie`, `nibble`, or `nod`; preserve the existing claims for `mail`,
`social`, and `murder`.
