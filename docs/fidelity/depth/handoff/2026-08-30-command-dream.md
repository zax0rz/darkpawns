# Depth handoff — 2026-08-30 — `dream`

## Frontier and queue position

- Started from clean `main` at `a3449b9da` after the dragon slice, with the
  required `make fidelity-depth` confirmation and a read of
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff,
  `2026-08-30-command-dragon.md`.
- The clean-main frontier before this slice was 1,603 total, with 1,548
  proven/delegated, 14 blocked, and 41 excluded. After adding this slice,
  `make fidelity-depth` reports 1,610 total; 1,555 proven/delegated, 14
  blocked, 41 excluded, and actionable completion 1,555/1,569 (99.1%).
- The source-order command gap was `dream`, registered at
  `src/interpreter.c:423`. The already-manifested `drink` and `drop` rows
  precede it; the next un-manifested command after this slice is `drool` at
  `src/interpreter.c:425`.

## C call path and branch inventory

`src/interpreter.c:423` registers `dream` with `POS_SLEEPING` and
`do_dream`. The handler is `src/act.social.c:294-305`:

1. An actor above `POS_SLEEPING` receives
   `You daydream about better times.\r\n` and returns.
2. A sleeping actor calls `act()` first with
   `"$n dreams of running naked through a field of tulips."`,
   `hide_invisible=TRUE`, and `TO_ROOM`.
3. The actor then receives
   `You dream of running naked through a field of tulips.\r\n`.

The `comm.c:2480-2555` `SENDOK` path means plain `TO_ROOM` excludes sleeping
recipients, excludes the actor, and suppresses the room line for observers who
cannot see the actor. The Go implementation had reversed the room/private
order, passed `hide_invisible=false`, and added `TO_SLEEP`, which was a
confirmed divergence for sleeping peers.

## RED, confirmed fix, and GREEN proof

The new `cmd/dp-oracle-diff/scenarios/dream-depth.txt` vehicle uses an awake
peer, a sleeping peer, and the primary actor. On clean `main`, the sleeping
peer incorrectly received `Someone dreams of running naked through a field of
tulips.`; the C oracle emitted no line. The actor and awake-peer bytes matched.

`pkg/game/act_social.go` now follows the C call order and flags: it calls
`Act(..., true, ..., ToRoom)` before sending the actor's private line. The
focused `TestDoDreamMatchesCOrderAudienceAndVisibility` test proves the exact
room-before-actor event order, sleeping-recipient suppression, blind-observer
suppression, actor exclusion, and CRLF bytes.

The oracle vehicle is GREEN for seeds 1, 2, 3, 5, and 8. Seed 1 was also run
with `--show-oracle`, and the sleeping branch plus audience blocks were
visible. The durable inventory is `docs/fidelity/depth/dream.tsv`, with seven
cases covering the awake branch, sleeping branch, ignored arguments, awake and
sleeping room audiences, emission order, and `hide_invisible` behavior.

## Gates

On `glm/depth-dream`:

- `make fidelity-depth` — PASS
- `go build ./...` — PASS
- `go vet ./...` — PASS
- `go test ./...` — PASS
- `golangci-lint run ./...` — PASS, 0 issues
- `gofumpt -l .` — clean
- `git diff --check` — clean

This slice follows R1/R2/R4 and R5e: C bytes, the registered command surface,
and the actual handler/callee path remain authoritative; the shared `Act`
audience contract is exercised rather than invented; and neither `src/` nor
the C oracle tree was edited. The next session should checkout `main`, pull,
confirm the frontier, and take `drool` in interpreter-table order.
