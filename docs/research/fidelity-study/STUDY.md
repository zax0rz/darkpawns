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

**The groups, provisional:**

| Group | Meaning | Pilot cases |
|---|---|---|
| **A. The test stopped short** | The comparison was right about what it saw, but it didn't watch long enough or require the event it claimed | logout menu, combat-death |
| **B. The comparison was wrong** | The reference binary or the normalization was itself defective | `load mob`, accuse, gsay color, the `-O2` build |
| **C. Verdicts assume a healthy oracle** *(candidate)* | The harness had no verdict for "the reference is broken", so a reference defect, recognized or not, was routed into a non-failing verdict (INFRA, blocked, EXPECTED_UNSTABLE) | `load mob` (unrecognized: INFRA and empty C blocks, Oct 6-13); accuse (recognized as a C anomaly on 2026-09-18, `d7e816b03`, and accepted green by verdict policy until the root cause on Oct 11) |

C stands only if the wider inventory supports it. Code *concealment* (what hid the defect) separately from *cause*, and record whether the defect was recognized at the time: accepting a known anomaly by policy is different from misreading it as noise.

**Next to it, not inside it:** the slop-squash thread. AI-introduced plausible-but-wrong code (yank PF-002, the couch PF-010, foreign level scale PF-055, scenario-fitted special cases PF-040) is what weak checking let survive. Label these with Pan et al.'s taxonomy later; cases outside it are candidate new categories.

**Out of scope for this paper:** agents playing the MUD. That's the next paper, and this one is its foundation.

## Where the evidence lives

| What | Where |
|---|---|
| Ledger (claims and their status) | `docs/research/EVIDENCE_LEDGER.tsv` on **origin/main** |
| Field notes | `docs/research/field-notes/` |
| Timeline | `docs/research/timeline/` |
| Incident records already written up | `docs/research/field-notes/2026-10-31-incident-records-oracle-validity.md` (PF-091, PF-092) |
| The five reproductions (Sol) | `~/.local/share/openresearch/files/darkpawns/research-scoping/` (protocol, inventory, evidence packets) |
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

## Next steps, in order

1. **Sol:** rebase onto main, re-freeze the inventory, drop its own research-operations rows, and add source batches: the RULEBOOK amendment log, the timeline, field notes, and commits citing R5h or "coincidence green". Code concealment separately from cause.
2. **Sol:** screen and deduplicate the inventory under the protocol. Claude reviews the inclusions and a sample of exclusions.
3. **Audits that test the groups:** a witness audit (does each scenario reach what its name or comment claims?) and a normalization audit (does every erased byte class have a scenario that can fail?). Brief to be written.
4. **Zach:** read Pan et al. (the taxonomy section), McKeeman, and the Anthropic migration post.
5. **One-page research brief**, after the audits, for outreach.

## Decisions waiting on Zach

- Whether group C is promoted, once screening is done.
- Whether to ask Egon and Lucci for permission to mention them by handle (Sai is Zach).
- When to start outreach, and to whom.
