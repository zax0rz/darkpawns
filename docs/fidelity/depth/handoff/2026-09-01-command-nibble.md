# Depth-fidelity handoff — `nibble`

Date: 2026-09-01

## Frontier

This session began from clean `main` after the `newbie` handoff at:

```text
Cases: 2648 total, 2578 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2578/2600 = 99.2%
```

The `nibble` slice is now merged to `main` in PR #1044 (`da970d913`). The
post-merge frontier is:

```text
Cases: 2659 total, 2589 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2589/2611 = 99.2%
```

The special-procedure inventory remains exhausted. The intentionally blocked
row `objmagic.sleep-entry-gates` remains blocked; its cast-sleep vehicle does
not prove the blocked direct-entry row. The interpreter sweep continues.

## Slice proof

The next source-order unclaimed row was `src/interpreter.c:567`:

```c
{ "nibble"   , POS_RESTING , do_action   , 0, 0 },
```

The C call path was read first: `src/act.social.c:102-151`, with the authored
record in `lib/misc/socials:520-529`:

```text
nibble 0 0
Nibble on who?
#
You nibble on $N's ear.
$n nibbles on $N's ear.
$n nibbles on your ear.
Sorry, not here, better go back to dreaming about it.
You nibble on your OWN ear???????????????????
$n nibbles on $s OWN ear (I wonder how it is done!!).
```

The reachable branches are: the POS_RESTING command gate; the shared
`PLR_NOSHOUT` refusal; the no-argument actor prompt and absent room branch;
`one_argument` fill-word skipping and trailing-token discard; visible-target
actor/room/victim delivery; missing-target refusal; self-target actor/room
delivery; and the zero `min_victim_position` branch that admits a sleeping
target. Shared position, emote-refusal, target visibility, and Act delivery
seams were delegated to their established manifests under R5b/R5c.

Go already had the matching `nibble` social registration and shared
`DoAction` implementation. No behavior change was justified by the oracle
comparisons (R4).

Added:

- `cmd/dp-oracle-diff/scenarios/nibble-depth.txt`
- `cmd/dp-oracle-diff/scenarios/nibble-sleeping-depth.txt`
- `pkg/session/nibble_depth_test.go`
- `docs/fidelity/depth/nibble.tsv` (11 rows)

The normal vehicle used an actor, target, and observer in room 8162. Its
`--show-oracle --seed 1` run confirmed exact prompt, target trio, self, missing
target, and fill-word behavior. The sleeping vehicle put a target to sleep and
confirmed the actor and awake observer bytes while the target received none.
Both vehicles returned `no normalized divergence` for seeds 1, 2, 3, 5, and 8.

Focused tests passed, and all local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Hosted lint, security, and test checks for PR #1044 were green; build and
deploy were skipped by the workflow. The PR was merged only after required
hosted checks were green.

## Fidelity rules applied

- R1: preserved exact social bytes, substitutions, CRLF, and empty audience.
- R2: preserved the registered `nibble` command and POS_RESTING gate.
- R3: repeated both observable vehicles across deterministic seeds 1, 2, 3,
  5, and 8.
- R4: made no Go behavior change without a confirmed divergence.
- R5b/R5c: delegated only the already-proven shared social seams after tracing
  their common call path.
- R5e: verified the actual C registration, dispatcher, and social record.

## Next queue item

The next unclaimed interpreter-table family is `nod` at
`src/interpreter.c:568`. Map its C `do_action` call path and record before
creating branch `glm/depth-nod`. Do not re-pick `newbie` or `nibble`; preserve
the existing claims for `mail`, `social`, and `murder`.
