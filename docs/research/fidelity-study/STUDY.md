# The port-fidelity study: shared brief

*Approved by Zach 2026-10-31. Every agent working on the study reads this file first. When it and an agent's own notes disagree, this file wins until Zach changes it. Claude keeps it current.*

## What we're studying

**The frame (the paper's story).** Dark Pawns was ported from C to Go with AI agents. At each stage, the current way of checking said the port was done, and the next way of checking proved it wasn't:

1. **Winging it:** the game ran and felt right (README: "the Go port is complete", 2026-07-06).
2. **Oracle, breadth:** the original C ran beside the port; every command matched at least once.
3. **Oracle, depth:** long agent runs proved behaviour site by site.
4. **Playtesting:** a green census missed bugs a 30-minute human session found.
5. **Returning players:** former players of the 1990s game found more.
6. **The oracle itself was wrong:** the reference binary and the comparison had defects of their own.

The stages overlap rather than follow one another, and the later stages stopped equating green with done. Both facts are part of the story, not exceptions to it. Evidence: [`timeline/timeline.tsv`](../timeline/timeline.tsv) and its notes (PF-093).

**The question (the study's core).** How did faults in test design and in the reference execution limit differential validation during the migration, and what evidence exposed them? (Sol's protocol v0.1.)

**Draft research questions (Zach, 2026-11-01).** Framed as an extension of Pan et al., "Lost in Translation" (ICSE 2024): three of their four questions transfer to a 2026 agent-driven migration of a real system, and the fourth is replaced by the checking study. In story order:

1. **"Working" vs "faithful"** *(their RQ1)*: what did each claim of completion miss? Answered descriptively from the timeline's done/disproved pairs; no success-rate percentages.
2. **The mistakes** *(their RQ2)*: what kinds of mistakes did agents introduce, labelled with Pan et al.'s categories, and how do they compare to 2024: which categories disappeared, and what is new? This is the slop track ([`BRIEF-2026-11-01-slop-track.md`](BRIEF-2026-11-01-slop-track.md)). Prevalence covers caught mistakes only, and the paper says so.
3. **The checks** *(new)*: how did the checking itself mislead? Groups A, B and C; the study above.
4. **The fixes** *(their RQ4, reshaped)*: did each process change (rulebook amendments, R5h, playtesting) stop its class of mistake from recurring? Answered from the rulebook amendment log and the timeline.

Their RQ3 (non-LLM translators) is dropped: no traditional translator was run. Where git shows which agent introduced a mistake, record it, and report it descriptively only: tasks weren't randomized, the harness is a confound, and model versions changed.

**The groups (definitions below; screening results in [`screening/`](screening/README.md)):**

| Group | Meaning | Examples | Core count (screening v2) |
|---|---|---|---:|
| **A. The test stopped short** | A named check claimed a behaviour, ran, and didn't reach it | logout menu, combat-death | 16 |
| **B. The comparison was wrong** | The reference binary or the normalization was itself defective | `load mob`, accuse, gsay color, the `-O2` build | 17 |
| **C. A failure recorded as a non-failure** | A check failed, never ran or gave known-bad output, and was recorded as pass, expected, INFRA, flake or blocked without its cause resolved | container-save, peer-login, the CI panic called a flake; `load mob` and accuse as secondary C | 5 (7 with secondary codes) |

Another 43 bugs were **uncovered** (no check claimed them), and they feed the slop track. Code *concealment* (what hid a defect) separately from *cause*, and record whether it was recognized at the time: accepting a known anomaly by policy is different from misreading it as noise.

**Group C adopted (Zach, 2026-10-31 and 2026-11-01)** in its broader wording: it covers any check, not only the oracle. The definitions below govern.

**Scope ruling (Zach, 2026-10-31): supporting evidence.** Incidents outside the game port that show one of the groups' mechanisms count as *supporting* evidence: cited to show the mechanism recurs, but kept out of the core incident count and out of any group's core tally. This covers CI and test tooling (the deadcode ratchet that read a crash as no findings, `6ecba615f`; ARM-suffixed tests that never ran on amd64, `ca6deb315`) and test stubs built from an implementer's assumption (the Mudlet stub, MA-006). Revisit before drafting; cutting them later is cheap.

**Noticed vs unnoticed.** The screening register counts what the project noticed. The audits ([`BRIEF-2026-11-01-audits.md`](BRIEF-2026-11-01-audits.md)) looked for what it didn't, and found a 52-scenario C cluster (269 spell-item manifest rows, refused setup since 2026-10-18, shipped in v0.1.8; PF-095) and a scenario hollowed by a newer test (transfer, PF-096). Report the two separately: the gap between them is a finding. Port fix tracked in DP-1456.

**What counts as "the check" (Zach, 2026-10-31).** Use this when coding group C:

- **A check** is any step whose result the project used as evidence: a scenario verdict, a test exit code, a CI job, or a manifest or proof status.
- **Group C** applies when that step failed, never ran, or produced known-bad output, and it was recorded as a non-failure (pass, expected, INFRA, flake, blocked) without its cause being resolved.
- **Not C:** the step ran properly but watched too little (A), or ran against a wrong reference or erased bytes (B).
- **Setup is part of the check.** A scenario whose required setup was refused, yet still recorded PASS, is C.
- **A false green withdrawn before anyone relied on it is not C.** The process caught it; code it as a detection, in the project's favour.
- **"Relied on" means the claim reached main, a release, a report to Zach, or a decision** (Zach, 2026-11-01). A claim credited and withdrawn on an unmerged branch was not relied on, even if an agent's working handoff counted it.

**What counts as A, and "uncovered" (Zach, 2026-11-01).**
- **A requires a named check that claimed the behaviour:** a scenario name or comment, a manifest row, a proof status, or a test name. The check ran, but stopped short of what it claimed.
- **A bug no check claimed to cover is "uncovered"**, not A. Record it, with what found it (playtest, returning player, later depth work), but keep it out of the A/B/C tallies.
- **Uncovered bugs and excluded inventions feed the slop track.** Tag them so the second study can pick them up; an exclusion from the checking study is not a discard.

Code against this definition; a case it leaves ambiguous stays boundary, with the reason.

**The second study: the slop track.** AI-introduced plausible-but-wrong code (yank PF-002, the couch PF-010, foreign level scale PF-055, scenario-fitted special cases PF-040) is what weak checking let survive. It answers research question 2, labelled with Pan et al.'s taxonomy; cases outside it are candidate new categories. Its starting list is the slop-tagged screening records and git leads.

**Out of scope for this paper:** agents playing the MUD. That's the next paper, and this one is its foundation.

## Where the evidence lives

| What | Where |
|---|---|
| Ledger (claims and their status) | `docs/research/EVIDENCE_LEDGER.tsv` on **origin/main** |
| Field notes | `docs/research/field-notes/` |
| Timeline | `docs/research/timeline/` |
| Incident records already written up | `docs/research/field-notes/2026-10-31-incident-records-oracle-validity.md` (PF-091, PF-092) |
| Screening register (reviewed, frozen) | `docs/research/fidelity-study/screening/` (PF-094) |
| Audit findings | `docs/research/field-notes/2026-11-01-witness-audit-hollow-proofs.md` (PF-095, PF-096) |
| Sol's working packets (reproductions, screening versions, audits) | `~/.local/share/openresearch/files/darkpawns/research-scoping/` |
| Oracle run archives | `~/Archives/darkpawns/oracle-runs/` |
| Related work | Zach's Zotero library |

## Who does what

- **Zach:** decides direction and scope, reads the related work, contacts people, approves anything that lands on main.
- **Claude:** reviews every result before it counts (re-runs checks, verifies citations and quotes against sources, checks C lines in `src/`), writes briefs, lands reviewed work on main, keeps this file current.
- **Sol (OpenResearch):** the protocol, the incident inventory and screening, reproductions.
- **DeepSeek:** reconstruction from the record (timelines, archive sweeps), from a written brief.
- **Gemini and Kimi:** literature searches. Every reference is checked before it enters Zotero.

Agents don't invoke each other. Results come back to Zach, and Claude reviews them in Zach's session.

## Rules for every agent

- **Read from current `origin/main`.** Run `git fetch` first. A worktree more than a day old is stale for this study.
- **Every claim cites a primary artifact** that resolves (commit, PR, file and line, run directory). Probable but uncited goes in open questions.
- **Quotes are word for word**, checked against the source. A paraphrase is never presented as a quote.
- **Keep failures:** refuted predictions, red runs and contrary evidence stay in the record.
- **No new writes to the shared checkout** at `/home/zach/darkpawns`. Use a worktree under `~/wt/`. Census only through `scripts/census.sh`. Never edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- **Nothing counts until reviewed.** Output goes into the agent's own folder or `~/dp-research/`; Claude reviews it, and Zach approves it before it lands in `docs/research/`.

## Done so far

- **Sol:** the five pilot reproductions, the incident inventory, screening (v2 frozen and reviewed, PF-094), and the witness and normalization audits (PF-095, PF-096).
- **DeepSeek:** the dated timeline (PF-093).
- **Zach:** the definitions of the check, reliance, A and uncovered; the draft research questions.

## Next steps, in order

1. **DeepSeek: the slop track** ([`BRIEF-2026-11-01-slop-track.md`](BRIEF-2026-11-01-slop-track.md)). Claude reviews the result.
2. **Zach:** read Pan et al. (the taxonomy section), McKeeman, and the Anthropic migration post.
3. **One-page research brief** for outreach, once the slop track has first numbers, so it shows both studies.
4. **Later:** after DP-1456 lands, check whether its class fix stops refused-setup passes (research question 4).

## Decisions waiting on Zach

- Whether to ask Egon and Lucci for permission to mention them by handle (Sai is Zach).
- When to start outreach, and to whom.
