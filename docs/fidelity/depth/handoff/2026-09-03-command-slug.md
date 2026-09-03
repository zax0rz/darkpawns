# Depth-fidelity handoff: `slug`

Date: 2026-09-03

## Completed slice

- Feature branch: `glm/depth-slug`
- Feature PR: #1246, merged green
- Feature commit: `cc55d6574`
- Main merge commit: `03a0df349`
- No files under `src/` or `darkpawns-c-oracle/` were changed.

The source-order row is `slug` at `src/interpreter.c:709`:

```c
{ "slug", POS_FIGHTING, do_slug, 1, 0 },
```

The C call path was traced through `src/new_cmds.c:820-877`, `src/fight.c:1023-1092`, and the skill-message set 146 in `lib/misc/messages:350-364`. The proven contract includes one-argument parsing, the no-skill response, room lookup followed by the fighting fallback on any miss, self/weapon/mounted gates, the exact damage and percentage branches, set-146 combat output, deferred improvement, combat start, and `PULSE_VIOLENCE*2` wait behavior. The Go changes only corrected confirmed divergences: command parsing/fighting fallback and the self/weapon/mounted plus hit/miss routing in `DoSlug` (R1, R2, R3, R4, R5, R5e).

## Evidence and gates

The following scenarios were green against the C oracle at seeds 1, 2, 3, 5, and 8 where applicable:

- `slug-depth` (seed 1 included `--show-oracle`)
- `slug-outcome-depth`
- `slug-mounted-depth` (seed 1 included `--show-oracle`)

Focused coverage is in `pkg/game/slug_depth_test.go`, `pkg/command/slug_depth_test.go`, and `pkg/session/slug_depth_test.go`; the manifest is `docs/fidelity/depth/slug.tsv`. All local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .`, and `git diff --check`. Hosted PR #1246 lint, security, and test checks were green; build-and-push/deploy were skipped, CI fired normally, and no retry was needed.

The frontier after slug is:

```text
Cases: 3752 total, 3651 proven/delegated, 48 blocked, 53 excluded
Actionable completion: 3651/3699 = 98.7%
```

Slug added 18 proven/delegated rows from the post-slap frontier of 3734 total and 3633 proven. The special-procedure inventory remains exhausted. The blocked `objmagic.sleep-entry-gates` row remains blocked after its single cast-sleep outlaw/reagent attempt; the vehicle reached only the documented subset and the remainder stays blocked (R4/R5b/R5c).

## Next queue claim

A fresh source-order audit on merged `main` confirms that `slap` and `slug` are complete, `slowns` is owned by the generic `gen-tog` coverage, and `smile` is owned by shared social coverage. The next dedicated unclaimed command family is:

```text
smackheads — src/interpreter.c:711 — do_smackheads
```

The next session must checkout `main`, pull, run `make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then claim `glm/depth-smackheads`. Do not repick `slap`, `slug`, `slowns`, or `smile`. Continue one `glm/depth-<family>` PR per family and one dated handoff per session. The queue advances only after a RED-or-GREEN proof, focused manifest rows/tests, and all required local and hosted gates; a second honest proof failure becomes a sharp blocked note and advances (R1-R5, R5e, R5b, R5c).
