# 2026-08-29 — `pray_for_items` depth slice

## Frontier and queue

- Started from a clean `main` boundary, pulled `bb39f7686`, ran
  `make fidelity-depth`, and reread `docs/fidelity/DEPTH_TESTING.md` plus the
  newest committed handoffs at the boundary (`elemental_room`, followed by the
  pulled Dracula handoff).
- Main baseline: 913 total cases; 888 proven/delegated; 6 blocked; 19
  excluded; actionable completion 888/894 (99.3%).
- The feature branch adds six manifest claims and leaves one current-world-
  unreachable branch explicitly excluded. Because its PR received no GitHub
  checks after the one permitted retry, those rows and the Go implementation
  are not on `main`; main therefore remains at 913 total cases; 888
  proven/delegated; 6 blocked; 19 excluded; actionable completion 888/894
  (99.3%).

## C call path and branch census

- `SPECIAL(pray_for_items)` is defined at `src/spec_procs.c:2071-2150` and
  assigned to room 8008 at `src/spec_assign.c:605`. `src/interpreter.c:1407-1415`
  invokes the room special before ordinary command dispatch, and
  `src/interpreter.c:947-948` calls the command handler when the special
  returns `FALSE`.
- For `pray`, C first uses `one_argument`, then has two top-level branches:
  `immortality`, which always returns `TRUE` and applies independent exact-name
  level assignments/messages, and the item-key branch, which scans room object
  extra descriptions for `item_for_<GET_NAME(ch)>`, creates matching objects,
  charges the sentinel-plus-cost total, and returns `TRUE` only when it found
  one. A no-match invocation returns `FALSE` and reaches `do_action`.
- The authoritative room reset puts object 8099 in room 8008; its only extra
  keyword is `board`. No `item_for_<player>` key exists in `lib/world`, and no
  player command creates one. Per R2/R4/R5e, the latent reward loop is not
  claimed as reachable player behavior and is recorded as D5 `excluded`.

## RED/GREEN evidence and port result

- The first live fallback vehicle was RED on pre-fix main: the Go room-special
  handler dereferenced `me`, but room specials pass `me=nil`.
- The pre-fix Serapis vehicle was RED: Go omitted C's `Welcome back Serapis.`
  line and did not mutate the level to 40 (the following `score` exposed it).
- On `glm/spec-pray-for-items`, both `spec-proc-pray-for-items --show-oracle`
  and `spec-proc-pray-immortality --show-oracle` are GREEN with no normalized
  divergence. Focused tests cover the command gate, all hard-coded name arms,
  silent unlisted immortality, and the latent item loop.
- Go now uses C's `one_argument` parsing, exact name cascade, exact welcome
  bytes, object extra-description scan, room/actor audience split, checked
  object placement, gold clamp, and `FALSE` ordinary-social fallthrough. No
  `src/` or `darkpawns-c-oracle/` file was edited.

## Integration and next queue item

- PR #745 (`glm/spec-pray-for-items`) is open. GitHub reported no checks; the
  one permitted retry was issued with `gh workflow run "Dark Pawns CI/CD"
  --ref glm/spec-pray-for-items`, after which checks still did not fire. It is
  therefore not green and was not merged. The implementation, manifest rows,
  and live proof remain on that branch. This dated handoff claims the
  source-order item so it must not be repicked; its pending integration is
  recorded rather than relabeled as an excluded behavior.
- Local branch gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .` (the first lint attempt needed `/usr/local/go/bin` on PATH).
- Next source-order special is `fearface`; continue only after a fresh
  `main`/pull/frontier boundary. Do not repick `pray_for_items` or the earlier
  claimed specials. After active specials, attempt the one blocked
  `objmagic.sleep-entry-gates` vehicle, then sweep interpreter command
  families in table order.

This handoff applies R1 (player-facing bytes), R2 (registered command surface),
R4 (no invented world data), and R5/R5e (actual C call path and registration).
