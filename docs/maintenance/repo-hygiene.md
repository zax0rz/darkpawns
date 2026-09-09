# Repository housekeeping — 2026-09-09

This is a bounded first-pass inventory, not a launch backlog. Decisions cover every
tracked root file at the start of the pass, the duplicate brief directory, and
selected generated artifacts. `investigate` means retained pending evidence.

## Root files

| Original path | Decision | Reason / disposition |
| --- | --- | --- |
| `.env.example` | keep | Current project entry point, build configuration, or dependency manifest. |
| `.env.privacy.example` | keep | Current project entry point, build configuration, or dependency manifest. |
| `.gitignore` | keep | Current project entry point, build configuration, or dependency manifest. |
| `.golangci.bck.yml` | remove | Unreferenced backup; active configuration is .golangci.yml. |
| `.golangci.yml` | keep | Current project entry point, build configuration, or dependency manifest. |
| `.hugo_build.lock` | remove | Obsolete Hugo lock; website is Astro. |
| `AGENTS.md` | keep | Current project entry point, build configuration, or dependency manifest. |
| `DEPLOYMENT.md` | keep | Current project entry point, build configuration, or dependency manifest. |
| `Dockerfile` | investigate | Requires prebuilt admin-ui/dist; verify clean-checkout build separately. |
| `Dockerfile.ai-agent` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `Dockerfile.local` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `Dockerfile.prebuilt` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `Dockerfile.privacy-filter` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `GLM.md` | keep | Agent-specific operating manual; relocation requires checking external agent entry points. |
| `LICENSE` | keep | Current project entry point, build configuration, or dependency manifest. |
| `Makefile` | keep | Current project entry point, build configuration, or dependency manifest. |
| `Makefile.optimization` | investigate | Separate benchmark/profiling workflow; verify targets before consolidation. |
| `README.md` | keep | Current project entry point, build configuration, or dependency manifest. |
| `RESEARCH-LOG.md` | investigate | Preserve at current path: research documents cite it; reconcile automation consumers before moving. |
| `TEST_BUILD.sh` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `ZACHLOG.md` | move | Archived with obsolete database claim corrected; remaining entries are not a verified task queue. |
| `connect.py` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `docker-compose.moderation.yml` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `docker-compose.monitoring.yml` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `docker-compose.privacy.yml` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `docker-compose.yml` | investigate | Signing configuration and telnet publication missing; Docker build requires prebuilt admin UI. |
| `example_agent.py` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `go.mod` | keep | Current project entry point, build configuration, or dependency manifest. |
| `go.sum` | keep | Current project entry point, build configuration, or dependency manifest. |
| `map-generation-example.js` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `parse_world.py` | investigate | Root parser overlaps other parser tools; verify imports and generated outputs. |
| `parse_world_fixed.py` | investigate | Variant parser; do not delete based on filename alone. |
| `parse_world_test.py` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `performance` | remove | Untrack build artifact; preserve local file and ignore future builds. |
| `performance_config.yaml` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `quick_perf_test.sh` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `requirements-lock.txt` | keep | Current project entry point, build configuration, or dependency manifest. |
| `requirements.txt` | keep | Current project entry point, build configuration, or dependency manifest. |
| `test.sh` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `test_direct_memory.py` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `test_parse_world_report.py` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `test_performance.sh` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `test_simple.lua` | investigate | Retain for now; verify callers and supported workflow before moving or removing. |
| `website-mockup.html` | investigate | Candidate historical design artifact; inspect consumers before archiving. |

## Briefs and other artifacts

- Seventeen root briefs were byte-identical to their canonical `docs/briefs/`
  copies and were removed. Git history and canonical copies retain the content.
- The root deployment brief differed only in supersession wording and link path.
  It is preserved in `archive/briefs/deploy-production-root.md`.
- All 191 canonical brief files are listed in [brief-inventory.tsv](brief-inventory.tsv).
  Their presence does not indicate open work. Only the deployment brief explicitly
  declares itself superseded at the document level; other task status remains unverified.
- Two tracked Python cache files under `website-astro/scripts/__pycache__/` were
  untracked, preserving the existing local files. Ignore rules already cover them.
- `src/`, the oracle fixtures, fidelity manifests, research evidence, and generated
  website data remain in place. Generated does not automatically mean disposable:
  some website assets are required deployment inputs.
- The pre-existing changes in `website-astro/src/generated/project-activity.json`
  and `website/static/data/search-index.json` were left untouched.

## Remaining decisions

The follow-up [native installation check](native-install-check.md) now verifies
fresh-database creation and saved login after restart. It also corrects the world
path, `make run`, and environment example. The subsequent
[container retirement](container-retirement.md) supersedes the original Docker
investigation items in the root-file inventory above.

1. Reconcile the archived Zachlog entries against commits and current behavior.
   `grantClassSpells` exists and is called from login and creation today, so the old
   "uncommitted fix" note is not evidence of pending work. Self-cast behavior and
   the historical QA branch have not been audited in this housekeeping pass.
2. Verify brief completion against commits/issues before archiving. Keep cited
   paths stable where code, rulebook, or research evidence depends on them.
3. Resolved by retirement: Docker/Compose is no longer a supported install workflow.
4. Review older contributor/architecture docs and root utilities before reorganizing
   them. In particular, lineage and absolute port-completion claims in the README
   need primary-source reconciliation; this pass does not certify those claims.
5. Audit additional tracked build outputs in nested tooling separately, including
   `cmd/dp-goat/`; its own instructions and distribution workflow apply.

## Retention rule

Root contains project entry points and root-dependent tooling. Maintained guides
live under `docs/`; historical handoffs are explicitly labeled. Keep unique research
evidence public at stable paths. Keep official infrastructure and secrets in the
private ops repo. Do not infer completion from age or a brief filename.
