# Depth fidelity handoff — 2026-08-27

## Checkpoint

This three-item follow-on goal is complete on `main` at `bc8d8175c`:

- `make fidelity-depth`: **516 total, 505 proven/delegated, 1 blocked, 10
  excluded**; exit 0.
- Actionable completion: **505/506 = 99.8%**.
- The only remaining blocked row is
  `object-magic.tsv:objmagic.sleep-entry-gates`.

The score round was merged as PR #685 at `9add28247`. The charm round was
merged as PR #686 at `bc8d8175c`. Both were branched from current `main`,
reviewed through hosted checks, and self-merged only after green checks.

## `score.state-variants` — closed

The God-set vehicle was extended to verify the C oracle's stat and position
support before relying on those controls. The Go `cmdSet` stat behavior was
fixed where it diverged, and the affect/state matrix is now live-proven across
the score vehicle's seeds. The owning `info.tsv` rows are
`score.state-variants`, `score.state-variants.affect-flags`, and
`score.state-variants.position`.

## `hit.charm-master` + `assist.mob-helpee-pers` — closed

`combat-entry-charm-relations` was run C-first and then across seeds 1, 2, 3,
5, and 8. The pre-fix RED transcript established two real Go divergences:

1. C permits an Outlaw caster to charm a PC, creating the master relation
   needed by `do_hit`; Go rejected all PC charm victims.
2. Go's `do_assist` enrolled the fight and used its combat engine opener
   without the immediate `hit()` low-level PC gate.

The Go-only fix ports the C PC charm save/Outlaw gate and victim-facing charm
text, preserves C assist audience ordering, and applies the immediate hit
protection before enrollment. The vehicle also proves C `PERS` rendering of a
mob helpee as `a guard trainee` in the room audience. The owning
`combat-entry.tsv` rows are now `oracle-green-multiseed` with proof
`combat-entry-charm-relations`.

No C or oracle files were edited. The work follows R1/R3/R4/R5e: player bytes
and draw behavior come from the C call path, no new game behavior was invented,
and the confirmed fixes are confined to Go.

## `objmagic.sleep-entry-gates` — audited, intentionally still blocked

The C call path confirms there is no honest potion vehicle for this row:

- `mag_objectmagic` is the entry point for quaff/use/recite object magic
  (`src/spell_parser.c:544-558`).
- The potion arm always calls `call_magic(ch, ch, NULL, ...)`, targeting the
  quaffer (`src/spell_parser.c:685-714`).
- `SPELL_SLEEP` is registered with `TAR_CHAR_ROOM | TAR_NOT_SELF`
  (`src/spell_parser.c:1395-1396`), and the target parser rejects `tch == ch`
  for `TAR_NOT_SELF` (`src/spell_parser.c:886-892`).

Therefore the sleep effect body (`src/magic.c:1199-1249`) cannot be reached
through the potion/object entry without inventing a command path or changing
the game. The reachable cast surface remains separately proven by
`objmagic.sleep-entry-gates.cast` in `sleep-spell-depth`; the original
`objmagic.sleep-entry-gates` row remains blocked with the object-magic owner.

No deep-engine backlog item was attempted or reclassified in this goal.

## Verification

- `make fidelity-depth` — pass, counts above.
- `go build ./...` — pass.
- `go vet ./...` — pass.
- `go test ./...` — pass.
- `golangci-lint run ./...` — 0 issues.
- `gofumpt -l .` — no output.
- PR #686 hosted test, lint, and security checks — pass.

Next vehicle should start from this clean `main` checkpoint and take the
remaining blocked row only if a real C-reachable object-magic carrier is found;
otherwise keep the blocker and move to the next unproven class without
collapsing it into the already-proven cast row.
