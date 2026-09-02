# Depth-fidelity handoff — `burp`

Date: 2026-09-02

## Queue position and result

This round started from synced `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, rereading `docs/fidelity/DEPTH_TESTING.md`, and reading
the newest prior handoff, `2026-09-02-command-brb.md`. The special-procedure
inventory remains exhausted. The one-time blocked `objmagic.sleep-entry-gates`
row was already attempted through the cast-sleep outlaw/reagent vehicle and
was not repicked. The interpreter sweep consumed the next genuinely
unmanifested family, `burp`, at `src/interpreter.c:368`.

The pre-slice frontier was 3,558 total cases, with 3,457 proven/delegated, 48
blocked, and 53 excluded. The burp manifest contributes eight fully
proven/delegated cases. The resulting frontier is:

- 3,566 total cases
- 3,465 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,465/3,513 = 98.6%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:368 */
{ "burp"     , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-127`. `do_action` resolves the social
record, applies the shared PLR_NOSHOUT gate, and checks `char_found`. The burp
record in `lib/misc/socials:92-96` is `burp 0 0` followed by `You burp
loudly.`, `$n burps loudly.`, and `#`. Because `char_found` is `#`, C clears
the target buffer and always takes the no-argument branch: the actor receives
the first message and the room audience receives the second. Target lookup,
self-target, not-found, and victim-position branches are unreachable for
this record, while the command-level POS_RESTING gate remains applicable.

## Evidence and verification

The C-first vehicle is `cmd/dp-oracle-diff/scenarios/burp-depth.txt`. It uses
named actor, observer, and target peers and probes no argument, a visible
target, a missing target, a self target, and trailing words. The scenario was
run on clean pre-change `main` with `--show-oracle` at seed 1 and at seeds 2,
3, 5, and 8. Every run reported `result: no normalized divergence`, and the
shown C blocks confirm that all arguments remain on the no-argument actor/
room path. No Go source change was needed: the generic `DoAction` and existing
burp record were already byte-faithful.

Durable evidence is:

- `cmd/dp-oracle-diff/scenarios/burp-depth.txt`;
- `docs/fidelity/depth/burp.tsv`; and
- `pkg/session/burp_depth_test.go`.

The manifest delegates the shared POS_RESTING command gate, PLR_NOSHOUT
boundary, and social audience/visibility behavior to their existing owners
under R5b/R5c.

## Integration and gates

The required local gates passed on `glm/depth-burp`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `b890eaf24` (`test: add burp depth fidelity coverage`).

Feature PR: #1217 (`glm/depth-burp`). Hosted lint, security, and test checks
completed green; conditional build-and-push and deploy jobs were skipped. CI
fired normally, so no workflow retry was used. The PR was self-merged only
after all applicable checks were green. The resulting `main` merge commit is
`a108611d3`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented target output), R5/R5e (the actual C
call path), and R5b/R5c (shared social gate, lookup, and audience ownership).

## Continuation

The source-order sweep should next audit `buy` at the following interpreter
row, after confirming that no existing manifest already owns it:

```c
/* src/interpreter.c:369 */
{ "buy"      , POS_STANDING, do_not_here , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map the next
genuinely unmanifested family in interpreter-table order.

