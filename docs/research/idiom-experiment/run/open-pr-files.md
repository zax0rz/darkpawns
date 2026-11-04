# Open PR file exclusions at setup

Queried `gh pr list --repo zax0rz/darkpawns --state open` on 2026-11-04.
The only open PR was #2271, “fix(mobs): render ambient mob sound through
act(), not raw $-templates”. Its `gh pr diff 2271 --name-only` files are:

- `cmd/dp-oracle-diff/scenarios/mob-sound-mime-emote.txt`
- `cmd/dp-oracle-diff/scenarios/mob-sound-zealot-say.txt`
- `docs/fidelity/depth/mobprog.tsv`
- `docs/fidelity/depth/spec-procs.tsv`
- `docs/research/EVIDENCE_LEDGER.tsv`
- `pkg/game/mobprogs.go`
- `pkg/game/spec_act_templates_test.go`
- `pkg/game/spec_procs4.go`

These paths are excluded from refactor targets.
