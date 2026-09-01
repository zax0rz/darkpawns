# Depth-fidelity handoff: `jeer`

Date: 2026-08-31

## Queue position and result

This session consumed the next actually unmanifested interpreter-table command
after `invis`: `jeer`, registered at `src/interpreter.c:522` as `do_action` at
`POS_RESTING` with no command-level minimum. The special-procedure inventory
remains exhausted, and the single blocked `objmagic.sleep-entry-gates` row
remains deferred; neither was repicked.

The earlier `invis` handoff pointed at `junk`, but a fresh source/manifest
sweep found that `junk` was already fully claimed by the object-disposal
handoff (`2026-08-25-disposal`) and `docs/fidelity/depth/disposal.tsv`.
This session therefore did not repick `junk`; it consumed `jeer` as the next
unmanifested command after that already-proven family.

The pre-slice frontier was 2,319 total cases, with 2,259 proven/delegated,
16 blocked, and 44 excluded. After adding the 11 `jeer` rows, the frontier is
2,330 total, with 2,270 proven/delegated, 16 blocked, and 44 excluded:
2,270/2,286 actionable cases, or 99.3%.

Pure-coverage PR #979 (`test: depth-prove jeer C fidelity`) passed hosted
`test`, `lint`, and `security` checks; build-and-push and deploy were skipped
by the PR workflow. It merged to main as `bf5e12bb0` (feature commit
`d5eda484f`). No source behavior change was warranted because the existing Go
social implementation was already faithful.

The next truly unmanifested command in source registration order is `jin` at
`src/interpreter.c:523`.

## C call path and reachable branches

The authoritative path is `src/interpreter.c:522` →
`src/act.social.c:102-151` (`ACMD(do_action)`) → the `jeer` record at
`lib/misc/socials:420-428` → `act()`/`perform_act()` in
`src/comm.c:2392-2555` for room, victim, and actor audiences.

The registered C row requires `POS_RESTING` and has no command-level gate.
The `jeer` social record has hide flag 0 and minimum victim position 0, with
these reachable branches:

- no argument: actor receives `Smirk all you like.  Just don't come whining
  when you lose all your friends.` and the room receives `$n jeers.`;
- a first target token, parsed by C `one_argument`: a visible target receives
  the actor/victim/room trio `You start jeering at $N.`,
  `$n starts jeering at $N.`, and `$n jeers at you -- how rude!`;
- a leading fill word and trailing words: only the first parsed target token
  controls lookup;
- an unresolved target: actor receives `Smirk all you like.`;
- a named self target: actor receives `Eh?  Why would you want to do that?`,
  while the `others_auto` record is `#` and emits no room line;
- a sleeping visible target: minimum position 0 admits it, the actor's
  `TO_CHAR | TO_SLEEP` line is delivered, and ordinary `TO_VICT` does not
  deliver bytes to the sleeping target.

The shared `PLR_NOSHOUT` refusal, `POS_RESTING` dispatcher rejection, and
CAN_SEE/Act audience mechanics are owned by the existing `dance-noshout`,
`fade.position-gate`, and `socials-depth` manifests respectively. No RNG or
wait-state draw occurs on this `jeer` path.

## RED/coverage diagnosis

This was a pure-coverage round under the DEPTH_TESTING exception: the
main-equivalent Go implementation already used the faithfully loaded `jeer`
record and the shared `DoAction` path. The live `--show-oracle` runs were
GREEN before any change, so there was no confirmed divergence to fix. Adding
a duplicate behavior change would violate R4; the durable work is the
manifest, falsifiable vehicles, and focused registration/data test.

No `src/` or `darkpawns-c-oracle/` file was edited. The C call path and shared
callee ownership were verified first under R5e and R5b/R5c.

## Durable proof

The manifest `docs/fidelity/depth/jeer.tsv` contains 11 rows covering the C
entry gate, shared position/noshout/visibility gates, no-argument output,
first-token parsing, mob room audience, player victim audience, self target,
missing target, and sleeping-target recipient behavior.

The durable vehicles are:

- `cmd/dp-oracle-diff/scenarios/jeer-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/jeer-sleeping-depth.txt`.

Both vehicles were run with `--show-oracle --seed 1` on main-equivalent code;
both reached the intended C blocks and reported no normalized divergence. Each
was rerun at seeds 2, 3, 5, and 8; all eight additional runs reported no
normalized divergence. The focused `TestJeerRegistrationUsesCEntryGate`
test pins the C `POS_RESTING`/level-0 command row, zero social metadata, and
all eight authored messages.

The proof establishes R1 player-facing byte parity, R2 command-surface parity,
R3 deterministic multi-seed parity, R4 absence of invented behavior, and
R5e verification of the actual C dispatch/call path. Shared behavior remains
delegated under R5b/R5c; the pure-coverage GREEN result satisfies R5a's
falsifiable proof requirement.

## Verification and continuation

The feature branch passed all local gates:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...`;
- `test -z "$(gofumpt -l .)"` and `git diff --check`.

After merging, clean main reported the 2,330-case frontier successfully.
Hosted checks for PR #979 started normally; `lint` and `security` passed
first, followed by `test`. No workflow retry was needed, and no merge was
performed while a required check was pending.

The next session must start from clean main, pull, reconfirm the frontier,
read the depth-testing guide and this newest handoff, then map and prove
`jin` in table order. Continue to leave one dated handoff per session.
