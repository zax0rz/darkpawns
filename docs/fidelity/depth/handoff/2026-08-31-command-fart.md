# Depth handoff — 2026-08-31 — `fart`

## Frontier and queue position

- Started from clean `main` at `8af4e1d86` after the merged farewell handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-farewell.md`.
- The frontier before this slice was 1,702 total, with 1,645
  proven/delegated, 16 blocked, and 41 excluded. The fart manifest adds 7
  cases, all proven/delegated: 4 direct cases and 3 shared delegations. The
  post-slice frontier is 1,709 total, 1,652 proven/delegated, 16 blocked, and
  41 excluded; actionable completion is 1,652/1,668 (99.0%).
- The source-order command gap was `fart`, registered at
  `src/interpreter.c:442`. The next command-table gap is `fill` at line 443;
  the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:442` registers `fart` with `POS_RESTING`, no minimum level,
and `do_action` in `src/act.social.c:102-151`. Its social record is
`lib/misc/socials:226-234`:

- the actor receives `Where are your manners?`;
- the room receives `$n lets off a real rip-roarer!`;
- `char_found` is `#`, so the record has no target branch.

The C handler therefore does not consume a target token, clears its target
buffer, and always takes the no-argument actor/room branch. Not-found,
self-target, and victim-position branches are unreachable for this social; the
shared `PLR_NOSHOUT`, command-position, and `Act` audience behavior is covered
by delegated matrices. The Go social table has the same three-message shape,
and `DoAction`'s missing-`char_found` branch follows the same no-target path.

## Coverage proof

The live GREEN vehicle is `fart-depth`, covering no argument, an unknown typed
argument, and a self-looking typed argument with trailing words. The observer
proves the room audience and exact `$n` substitution; `--show-oracle`
confirmed the intended C no-target block. The vehicle was GREEN for seeds
`1,2,3,5,8`.

The entry gate is covered by `TestFartRegistrationUsesCEntryGate`. The shared
POS_RESTING rejection delegates to `fade.position-gate`; shared PLR_NOSHOUT
and Act visibility delegate to `dance-noshout` and `socials-depth`.

No `src/` or `darkpawns-c-oracle/` file was edited. The work follows R1/R2/R4,
R5e, and R5c: C bytes and reachability remain authoritative, the actual
`do_action` path was checked before claiming target branches unreachable, and
shared behavior is delegated rather than re-invented.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,709 total / 1,652 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

PR #859 (`test: prove fart command depth fidelity`) was merged only after
hosted `lint`, `security`, and `test` checks were all green. The workflow's
`build-and-push` and `deploy` jobs were skipped by policy. The next session
must return to clean `main`, pull, rerun the frontier check, and begin
`fill`.
