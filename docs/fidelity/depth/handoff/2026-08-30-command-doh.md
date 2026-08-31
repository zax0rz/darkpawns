# Depth handoff — `doh` command

Date: 2026-08-30
Queue slice: `src/interpreter.c:419`, `doh` / `do_action`
Starting main: `01d731d39`
Merged main: pending PR for this handoff

## Queue decision

The special-procedure inventory remains exhausted and the one permitted
`objmagic.sleep-entry-gates` cast-sleep attempt remains blocked. The command
table sweep advanced from `dns` to the next un-manifested family, `doh`.
The next un-manifested family after this slice is `donate`, but its command
row is already covered by `docs/fidelity/depth/donate.tsv`; the sweep should
continue to the next genuinely un-manifested row after the refresh.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## C path and proof

The command table registers `doh` at `src/interpreter.c:419` with
`POS_RESTING`, dispatching to `do_action` in `src/act.social.c:102-151`.
The C social record at `lib/misc/socials:1351` is self-only: its actor line is
`You smack your hand to your forehead, 'DOH!'`, its room line is
`You see $n smack $s hand to $s head exclaiming, 'DOH!'`, and its
`char_found` slot is `#`. Therefore no argument, a typed target, or a missing
target all use the no-argument actor/room pair; no target lookup or not-found
branch is reachable for this record. The interpreter position gate rejects a
sleeping actor with `In your dreams, or what?` before `do_action`, while wake
returns the actor to an allowed position and the social runs again.

The clean-main vehicle was GREEN at seed 1 with `--show-oracle`. It proves the
actor and peer room audiences for no argument, confirms typed-target ignoring,
captures the sleeping position rejection with no peer output, and confirms
the wake/recovery path. The port already uses the C-derived social table and
command gate, so no Go behavior required changing.

This follows R1/R2/R3/R4 and R5/R5e: exact actor/room bytes, authoritative
command position, deterministic social data, no invented target branch, and
verification of the actual table and `do_action` call path.

## Evidence and gates

Added:

- `cmd/dp-oracle-diff/scenarios/doh-depth.txt`
- `docs/fidelity/depth/doh.tsv`
- `docs/fidelity/depth/handoff/2026-08-30-command-doh.md`

Focused proof:

```
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff --scenario doh-depth --seed 1 --show-oracle
```

The full local gates passed after adding the manifest:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
```

The post-manifest frontier is 1,589 total, 1,534 proven/delegated, 14
blocked, and 41 excluded; actionable completion is 1,534/1,548 (99.1%).

## Hosted review

The `doh` change is a proof/manifest-only PR because clean main was already
GREEN. It must be merged only after all hosted required checks are green.

## Carry-forward

Return to clean `main`, pull, run `make fidelity-depth`, reread the depth guide
and this newest handoff, then continue the interpreter table in order. Skip
already-manifested `donate` and `drink` rows; the next genuinely un-manifested
family is `dragon` if no earlier un-manifested row is found by the refresh.
