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
| `GLM.md` | move | Moved to .agents/GLM.md alongside other agent tooling. |
| `LICENSE` | keep | Current project entry point, build configuration, or dependency manifest. |
| `Makefile` | keep | Current project entry point, build configuration, or dependency manifest. |
| `Makefile.optimization` | move | Consolidated into profiling/Makefile.optimization. |
| `README.md` | keep | Current project entry point, build configuration, or dependency manifest. |
| `RESEARCH-LOG.md` | move | Consolidated into docs/research/RESEARCH-LOG.md. |
| `TEST_BUILD.sh` | remove | Retired container test script. |
| `ZACHLOG.md` | move | Archived with obsolete database claim corrected; remaining entries are not a verified task queue. |
| `connect.py` | move | Moved to scripts/connect.py helper. |
| `docker-compose.moderation.yml` | remove | Retired container recipe. |
| `docker-compose.monitoring.yml` | remove | Retired container recipe. |
| `docker-compose.privacy.yml` | remove | Retired container recipe. |
| `docker-compose.yml` | remove | Retired container recipe. |
| `example_agent.py` | move | Moved to examples/example_agent.py. |
| `go.mod` | keep | Current project entry point, build configuration, or dependency manifest. |
| `go.sum` | keep | Current project entry point, build configuration, or dependency manifest. |
| `map-generation-example.js` | move | Archived to archive/map-generation-example.js. |
| `parse_world.py` | remove | Obsolete root parser; authoritative parser is website/scripts/parse_world.py. |
| `parse_world_fixed.py` | remove | Obsolete variant parser; authoritative parser is website/scripts/parse_world.py. |
| `parse_world_test.py` | remove | Obsolete root parser test; authoritative parser is website/scripts/parse_world.py. |
| `performance` | remove | Untrack build artifact; preserve local file and ignore future builds. |
| `performance_config.yaml` | move | Consolidated into profiling/performance_config.yaml. |
| `quick_perf_test.sh` | move | Consolidated into profiling/quick_perf_test.sh. |
| `requirements-lock.txt` | keep | Current project entry point, build configuration, or dependency manifest. |
| `requirements.txt` | keep | Current project entry point, build configuration, or dependency manifest. |
| `test.sh` | keep | Called by Makefile test targets (test-all, test-unit, test-e2e). |
| `test_direct_memory.py` | move | Moved to tests/integration/python/test_direct_memory.py. |
| `test_parse_world_report.py` | remove | Obsolete root parse report script. |
| `test_performance.sh` | remove | Obsolete script referencing non-existent file. |
| `test_simple.lua` | remove | Unused 7-line scratch test. |
| `website-mockup.html` | move | Archived to archive/website-mockup.html. |

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
