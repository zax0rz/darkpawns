# Depth handoff — 2026-08-30 — `faint`

## Frontier and queue position

- Started from clean `main` at `5ff757750` after the corrected fade/force
  ledger handoff, ran `git pull --ff-only`, confirmed `make fidelity-depth`,
  and reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-30-command-fade.md`.
- The frontier before this slice was 1,685 total, with 1,628
  proven/delegated, 16 blocked, and 41 excluded. The faint manifest adds 7
  cases, all proven/delegated: 4 direct cases and 3 shared delegations. The
  post-slice frontier is 1,692 total, 1,635 proven/delegated, 16 blocked, and
  41 excluded; actionable completion is 1,635/1,651 (99.0%).
- The source-order command gap was `faint`, registered at
  `src/interpreter.c:440`. The next command-table gap is `farewell` at line
  441; the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:440` registers `faint` with `POS_RESTING`, no minimum level,
and `do_action` in `src/act.social.c:102-151`. Its record is
`lib/misc/socials:221-224`:

- the actor receives `The world fades to black...to black... to black....`;
- the room receives `$n faints.  Luckily, you catch $m in time.`;
- `char_found` is `#`, so the record has no target branch.

The C handler therefore consumes no target token at all, clears its target
buffer, and always takes the no-argument actor/room branch. The not-found,
self-target, and victim-position branches are unreachable for this social;
the shared `PLR_NOSHOUT`, command-position, and `Act` audience behavior is
covered by delegated matrices. The Go social table contains the same
three-message shape, and `DoAction`'s missing-`char_found` branch follows the
same no-target path.

## Coverage proof

The live GREEN vehicle is `faint-depth`, covering no argument, an unknown
typed argument, and a self-looking typed argument with trailing words. The
observer proves the room audience and the exact `$m` capitalization/substitute;
`--show-oracle` confirmed the intended C no-target block. The vehicle was
GREEN for seeds `1,2,3,5,8`.

The entry gate is covered by `TestFaintRegistrationUsesCEntryGate`. The shared
POS_RESTING rejection delegates to `fade.position-gate`; the shared
PLR_NOSHOUT and Act visibility classes delegate to `dance-noshout` and
`socials-depth`.

No `src/` or `darkpawns-c-oracle/` file was edited. The work follows R1/R2/R4,
R5e, and R5c: C bytes and reachability remain authoritative, the actual
`do_action` path was checked before claiming target branches unreachable, and
shared behavior is delegated rather than re-invented.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,692 total / 1,635 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

PR #855 (`test: prove faint command depth fidelity`) was merged only after
hosted `lint`, `security`, and `test` checks were all green. The workflow's
`build-and-push` and `deploy` jobs were skipped by policy. The next session
must return to clean `main`, pull, rerun the frontier check, and begin
`farewell`.
