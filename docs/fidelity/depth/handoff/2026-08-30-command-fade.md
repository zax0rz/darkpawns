# Depth handoff — 2026-08-30 — `fade`

## Frontier and queue position

- Started from clean `main` at `6c009047e` after the merged `force` handoff,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-30-command-force.md`.
- The frontier before this slice was 1,675 total, with 1,618
  proven/delegated, 16 blocked, and 41 excluded. The fade manifest adds 10
  cases: 8 proven and 2 delegated. The post-slice frontier is 1,685 total,
  1,626 proven/delegated, 16 blocked, and 41 excluded; actionable completion
  is 1,626/1,644 (98.9%).
- The source-order command gap was `fade`, registered at
  `src/interpreter.c:439`. The next command-table gap is `faint` at line 440;
  the next session must rescan from clean `main` before taking it.

## C call path and branch inventory

`src/interpreter.c:439` registers `fade` with `POS_RESTING`, no minimum level,
and `do_action` in `src/act.social.c:102-151`. The social record is
`lib/misc/socials:211-219`. `do_action` first applies the shared
`PLR_NOSHOUT` gate, then resolves an optional target with `one_argument` and
`find_char_room_vis`. With no target, it emits the actor and room messages
from the record. A missing target emits the record's `Already gone. Good Job.`
line. A self target takes the `char_auto`/`others_auto` branch. A visible other
target takes the actor, non-victim room, and victim branches, subject to the
actual position and `CAN_SEE` audience checks.

The clean-main live baseline was already GREEN for fade's social behavior.
Focused gate proof exposed a shared port divergence: C's interpreter appends
CRLF after position, frozen, and switched-command rejection text, while the
Go common gate omitted it for those branches. The fix adds the missing CRLF
bytes in the shared gate and updates its exact-message tests; this is a
confirmed shared-class correction, not a fade-specific invention.

## Coverage proof

The live GREEN vehicle is `fade-depth`, covering no argument, visible target,
all three target audiences, ignored trailing words, self target, and not-found
behavior. The shared `dance-noshout` and `socials-depth` vehicles delegate the
PLR_NOSHOUT and visibility classes. The focused unit vehicle covers the C
registration gate and the sleeping position rejection. `fade-depth` was run
with `--show-oracle`, and all listed vehicles were GREEN for seeds
`1,2,3,5,8`.

No `src/` or `darkpawns-c-oracle/` file was edited. The work follows R1/R2/R4,
R5e, and R5c: C player-facing bytes and command gates remain authoritative,
the actual social call path was checked before changes, and the shared gate
class was corrected only after a focused oracle comparison.

## Gates and merge

Local gates passed:

- `make fidelity-depth` — 1,685 total / 1,626 proven-or-delegated /
  16 blocked / 41 excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` clean;
- `git diff --check` clean.

PR #852 (`fix: restore fade command depth fidelity`) was merged only after
hosted `lint`, `security`, and `test` checks were all green. The workflow's
`build-and-push` and `deploy` jobs were skipped by policy. The next session
must return to clean `main`, pull, rerun the frontier check, and begin
`faint`.
