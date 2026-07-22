# GOAL — Drive every oracle differential scenario from RED to GREEN

**Date:** 2026-07-17
**Target:** Codex (OpenAI, via `/goal` command)
**Owner:** Zach Greene (The Architect)
**Gate:** Daeron (review + Linear updates)

---

## The Goal (paste into Codex)

```
/goal Drive every scenario in cmd/dp-oracle-diff/scenarios/ to normalized-green parity between the C oracle and Go port, verified by the oracle diff harness producing zero normalized divergence. Do not modify C oracle source. Land each scenario group as a separate PR with before/after transcript proof.
```

---

## Constraints

1. **Do not modify C oracle source** (`src/*.c`). The C server is a read-only oracle. If you need to understand C behavior, read `src/` — never edit it.
2. **Do not weaken scenarios** to make them pass. If the normalizer is wrong, fix the normalizer. If the scenario is wrong, fix the scenario. The scenario represents the *correct C behavior*.
3. **Do not break green scenarios.** After every fix, re-run the full committed suite under `DP_CLOCK=1` to verify no regressions. Green must stay green.
4. **Do not touch the harness normalizer** (`internal/oraclediff/normalize.go`) unless a specific scenario proves it is eating real lines or producing false normalization. Document every normalizer change with the scenario that required it.
5. **Route all RNG through `dprng`** (the CMWC stream). No `math/rand`. Import guard enforces this.
6. **One PR per scenario group.** Don't land a monolith. Each domain (combat, movement, items, etc.) is a separate PR with its own green verification.
7. **Cross-compile before pushing.** `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./cmd/server` for any deployed binary. The oracle harness itself runs locally (macOS).

---

## Verification Surface

- **Per-scenario:** `DP_SEED=1 DP_CLOCK=1 DP_ORACLE_BIN=<path> go run ./cmd/dp-oracle-diff --scenario <name>` → zero normalized divergence
- **Suite-wide:** all committed scenarios pass under `DP_CLOCK=1` (no regressions)
- **Build gate:** `go build ./... && go vet ./... && go test ./...` green
- **PR gate:** each scenario group PR includes pasted before/after transcript diffs as proof

---

## Artifact

When complete (or when a major milestone is reached), write a report to `docs/reports/oracle-green/`:

```
docs/reports/oracle-green/
  SUMMARY.md          # overall status: N/32 scenarios green, broken down by domain
  combat.md           # combat domain: what diverged, what was fixed, key findings
  movement.md         # movement domain
  items.md            # item/equipment/inventory domain
  social.md           # communication/social domain
  skills.md           # skill/spell domain
  creation.md         # character creation domain
  gates.md            # command gating domain
  objects.md          # object interaction domain
  harness.md          # harness improvements made (normalizer fixes, new features)
  clock-seam.md       # DP_CLOCK findings and design notes
```

Each domain file should contain:
- Scenarios covered and their GREEN proof (before/after transcript hunks)
- C source locations referenced for each fix
- Linear issue IDs created for each finding
- Any architectural decisions or C-vs-Go disagreements that need The Architect's call

---

## Context

This is the **oracle differential testing** project — a live C CircleMUD server and the Go port
driven by the same scripted input, with transcripts normalized and diffed. The harness lives at
`cmd/dp-oracle-diff/`. 32 scenarios currently exist covering combat, movement, items, socials,
skills, creation, and command gating. The `DP_CLOCK` deterministic pulse-seam (PR #384/#385)
enables reproducible combat scenarios by freezing real-time pulses and pumping heartbeats
deterministically. `DP_SEED` controls the C oracle's RNG for reproducibility.

Key reference files:
- `docs/research/drafts/2026-07-12-c-oracle-differential-testing.md` — the why + two-tier plan
- `docs/research/drafts/2026-07-12-c-oracle-build-notes.md` — how the C oracle builds/boots
- `docs/briefs/BRIEF-2026-07-12-oracle-tier1-harness.md` — the original harness brief
- `docs/briefs/BRIEF-2026-07-15-combat-round-output-fidelity.md` — combat message fidelity
- `docs/briefs/DESIGN-2026-07-17-dp-clock-pulse-sync.md` — clock-seam design + gate results
- `internal/oraclediff/` — harness packages (scenario, normalize, conn, diff, report)
- `cmd/dp-oracle-diff/scenarios/` — the 32 scenario files
- Linear project: "Oracle Differential Testing"
