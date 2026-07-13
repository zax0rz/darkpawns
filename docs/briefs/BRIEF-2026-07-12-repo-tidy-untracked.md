# BRIEF 2026-07-12 — Repo tidy: sort the untracked provenance pile

**Executor:** daeron or mimocode (pure git/file housekeeping — no code logic, no C fidelity).
**Branch:** `chore/repo-tidy-untracked` off current `main` (@ `af63156` or later).
**One PR.** Zach or Claude merges after a glance. **Do NOT delete anything not explicitly
listed as delete-safe below.**

## Context

Working tree is on `main`, clean except for a large **untracked** pile that's accumulated
across sessions, plus **one uncommitted `.gitignore` edit** (Claude added `.mimocode/`).
This brief commits the provenance that belongs in the repo and quarantines the scratch.
`.mimocode/` (MiMo agent tooling: `node_modules/`, `package.json`, cron lock) is already
handled by the pending `.gitignore` edit — just include it in the commit.

## Do this

1. **Commit the pending `.gitignore` edit** (adds `.mimocode/`). Confirm `git status`
   no longer shows `.mimocode/` as untracked.

2. **Commit the `docs/` provenance** — this project embraces docs-as-provenance
   (`docs/research/`, `docs/session-notes/`, `docs/reviews/` are already conventions), so
   these belong in-repo:
   - `git add docs/briefs/BRIEF-*.md`
   - `git add docs/research/drafts/*.md`
   - `git add docs/session-notes/*.md`
   - Includes today's new scoping doc `docs/research/drafts/2026-07-12-c-oracle-differential-testing.md`.

3. **`cmd/dp-agent/SKILL.md`** — commit it (source doc for the `dp-agent` command; the
   `.gitignore` only excludes the *built* `/dp-agent` binary, not the source dir).

4. **Top-level `briefs/` (LEGACY location — `docs/briefs/` is canonical).** RECOMMENDED
   default: add `/briefs/` to `.gitignore` and leave it as local scratch (it holds a
   June-5 batch + 9 dated 2026-07-12 briefs whose work is already merged). **Decision
   toggle for Zach:** if he wants the 7-12 briefs preserved in-repo, instead `git mv
   briefs/2026-07-12-*.md docs/briefs/` (rename to the `BRIEF-2026-07-12-*` convention)
   before ignoring the rest. Default to ignore unless Zach says preserve.

5. **Sanity gate** (docs-only changes, but confirm nothing broke):
   `go build ./... && go vet ./... && go test ./...` — all green.

6. Commit, push, open PR:
   ```
   git commit -m "chore: commit docs provenance, ignore agent-tooling scratch"
   git push -u origin chore/repo-tidy-untracked
   gh pr create --title "chore: repo tidy — commit provenance, ignore scratch" \
     --body "Commits accumulated docs/ provenance (briefs, research drafts, session notes) + cmd/dp-agent/SKILL.md; gitignores .mimocode/ agent tooling and the legacy top-level briefs/ dir. See docs/briefs/BRIEF-2026-07-12-repo-tidy-untracked.md."
   ```
   Then STOP — do not merge. Zach or Claude reviews and merges.

## Explicitly out of scope / do not touch
- Any `src/*.c`, any Go code logic, any test behavior.
- Do not `rm` the top-level `briefs/` contents — gitignore only (Zach may still want them locally).
- Do not rewrite history or touch already-merged commits.
