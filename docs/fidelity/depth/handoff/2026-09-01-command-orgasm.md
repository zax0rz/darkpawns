# Depth-fidelity handoff — 2026-09-01 — `orgasm`

## Frontier

- Clean-main start: `Cases: 2690 total, 2620 proven/delegated, 22 blocked, 48 excluded`.
- Clean-main finish after the merged slice: `Cases: 2702 total, 2632 proven/delegated, 22 blocked, 48 excluded`.
- Actionable completion: `2632/2654 = 99.2%`.
- The special-procedure inventory remains exhausted. The one permitted
  `objmagic.sleep-entry-gates` attempt remains split: the cast-sleep vehicle is
  proven, while the unreachable remainder stays blocked with its existing note.

## Source path and queue decision

The next source-order command row was:

```c
src/interpreter.c:587: { "orgasm" , POS_RESTING , do_otouch , LVL_IMMORT, 0 }
```

The authoritative C path was read first at `src/new_cmds.c:666-724`. It keeps
the handler-level non-NPC immortal guard, parses with `one_argument`, handles
empty and unresolved targets, has a self-target `TO_NOTVICT` branch, and then
emits the actor/victim/room touch audiences. The male-victim branch adds two
hit points, sends the hunger/food line, and resets `FULL`; the female-victim
follow-up is conditional on a male actor. The registered command gate and
handler path were verified under R1/R2/R5e.

`offer` is covered by the existing `do_not_here` family, `open` by the door
family, `put` by the existing `do_put` manifest, and the intervening social
rows by the existing social-family claim. The next genuinely unclaimed
interpreter family is `page` at `src/interpreter.c:598`:

```c
{ "page"     , POS_DEAD    , do_page     , LVL_IMMORT, 0 },
```

Continue from a clean `main`, map `do_page`'s C call path first, and use branch
`glm/depth-page`. Do not repick `orgasm` or any of the already-claimed rows.

## Proof and implementation

The initial seed-1 vehicle was RED because Go had no registered `orgasm`
command and returned `Huh?!?` for every probe, while C reached all of the
branches above. The first corrected implementation then exposed one confirmed
transport divergence: C's embedded LFCR in the two-line victim message became
an extra blank line after Go telnet normalization. The fix split that message
into the existing `Act` line plus one canonical direct victim line. No source
or C-oracle file was edited.

Added evidence:

- `cmd/dp-oracle-diff/scenarios/orgasm-depth.txt` — empty, missing, self, NPC,
  male, female, and one-argument fill-word/trailing-token branches.
- `pkg/session/orgasm_depth_test.go` — registration, handler guard, male state,
  and female-actor state checks.
- `docs/fidelity/depth/orgasm.tsv` — 12 manifest rows covering entry, handler,
  arguments, target branches, audiences, state, and shared Act delegation.

The oracle vehicle returned `no normalized divergence` for seeds `1,2,3,5,8`;
`--show-oracle --seed 1` also showed the complete C blocks. All local gates
passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

## Review state

- Commit: `c2bc95a3c fix: deepen orgasm fidelity (R1 R2 R3 R5e)`.
- PR: #1053, `fix: deepen orgasm fidelity`.
- The initial hosted workflow did not fire; the one permitted retry was run:
  `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-orgasm`.
- Hosted lint, security, and test checks passed; build/deploy were skipped. The
  PR was squash-merged to `main` as `99fe48d8a`.

This handoff records R1/R2/R3/R5e and the shared-boundary treatment under
R5b/R5c. Continue from clean `main`; do not repick `orgasm`.
