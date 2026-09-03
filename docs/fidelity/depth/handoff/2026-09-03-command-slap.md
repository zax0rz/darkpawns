# Depth-fidelity handoff — `slap`

Date: 2026-09-03

Branch: `glm/depth-slap`

Feature PR: #1244 (merged green)

Feature commit: `2acff52cd`

Main merge: `b9172a81d`

## Queue position and result

This round returned to `main`, ran `git pull --ff-only`, confirmed the
frontier with `make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`
and the newest dated handoff, and then audited the interpreter table against
the depth manifests. The special-procedure inventory remains exhausted. The
one blocked row, `objmagic.sleep-entry-gates`, remains blocked after its one
allowed cast-sleep outlaw/reagent vehicle and was not repicked.

The next genuinely unclaimed source-order row after `sleeper` was `slap` at
`src/interpreter.c:707`. The intervening `slowns` row is owned by the
`gen-tog` family. `smile` is represented by the shared `socials-depth`
high-traffic proof. The next source-order family after this slice is the
dedicated `slug` handler at `src/interpreter.c:709`; confirm that claim from a
fresh `main` checkout before starting it.

Pre-slice frontier: 3,723 total, 3,622 proven/delegated, 48 blocked, and 53
excluded. The slap manifest adds 11 proven/delegated cases. Post-slice
frontier: 3,734 total, 3,633 proven/delegated, 48 blocked, and 53 excluded;
actionable completion is 3,633/3,681 = 98.7%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:707 */
{ "slap"     , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` first resolves the social record,
then checks `PLR_NOSHOUT`, parses only the first target token with
`one_argument` when `char_found` exists, and dispatches actor, observer,
victim, self, or missing-target branches. The `slap` record at
`lib/misc/socials:715-723` is `slap 0 0`: it has no hide flag and no minimum
victim position.

The reachable player-visible branches are:

- no argument: actor `Normally you slap SOMEBODY.`; the `#` room sentinel
  emits no room bytes;
- visible target: actor `You slap $N.`, observer `$n slaps $N.`, victim
  `You are slapped by $n.`;
- leading fill word and trailing words: C resolves only the first target;
- missing target: `How about slapping someone in the same room as you??`;
- self target: `You slap yourself, silly you.` plus the authored room self
  line `$n slaps $mself, really strange...`;
- sleeping target: the zero victim-position minimum admits the target, while
  ordinary `TO_VICT` delivery is filtered for the sleeping descriptor.

The command-level POS_RESTING gate, PLR_NOSHOUT refusal, and shared visible
lookup/audience mechanics are delegated to the existing `fade.position-gate`,
`dance-noshout`, and `socials-depth` evidence under R5b/R5c.

## Evidence and implementation boundary

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/slap-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/slap-sleeping-depth.txt`;
- `docs/fidelity/depth/slap.tsv`; and
- `pkg/session/slap_depth_test.go`.

The unchanged Go social path was already faithful. The required clean-main
RED check was therefore a pure-coverage GREEN: `slap-depth` with
`--show-oracle` at seed 1 reported no normalized divergence before adding the
manifest and test artifacts. The branch made no behavior change. This is the
pure-coverage exception in the loop definition of done; inventing a fix would
violate R4.

`slap-depth` reported `result: no normalized divergence` at seeds 1, 2, 3, 5,
and 8, with seed 1 inspected using `--show-oracle`. The sleeping-target
vehicle reported the same result at seeds 1, 2, 3, 5, and 8, with seed 1
inspected using `--show-oracle`. No file under `src/` or
`darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed on the feature branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` clean
- `git diff --check`

PR #1244's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. CI fired normally, so no
workflow retry was used. The PR was self-merged only after the applicable
checks were green, per the 2026-08-27 amendment.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed matrix and audience ordering), R4 (no invented behavior), R5/R5e
(verify the actual C path and let C win), and R5b/R5c (shared social gate,
lookup, and audience ownership).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and audit/claim
`slug` at `src/interpreter.c:709` before touching any implementation. Do not
repick `slap`, `slowns`, or the shared `smile` social proof.
