# Dark Pawns Research

This directory is the open research notebook for Dark Pawns and the prospective
AIIDE 2027 work. It is intentionally public while the project is ongoing. Drafts
and field notes are provisional; only claims linked to primary evidence in the
evidence ledger should migrate into a submission as results.

## Research Tracks

Dark Pawns currently supports three related but separable lines of research:

1. **Legacy-port fidelity:** how differential oracles and constrained agents can
   reconstruct the behavioral contract of a legacy interactive system.
2. **Multi-agent software engineering:** briefs, specialization, verification,
   false-positive correction, institutional memory, and the enforcement cost of
   long-running agent pipelines.
3. **AI agents in games:** server-hosted narrative/social memory, agent-facing
   protocols, behavioral persistence, and evaluation inside a live MUD.

The likely port-fidelity paper should not inherit experimental claims from the
agent-memory track merely because both use Dark Pawns. Each track needs its own
research questions, methods, evidence, and limitations.

## Start Here

- `EVIDENCE_LEDGER.tsv` — claim inventory and verification state; this is the
  bridge between prose and primary artifacts.
- `field-notes/` — dated observations and imported agent notes. Useful leads,
  not ground truth.
- `drafts/` — working prose. A draft can be insightful while factually stale.
- `evaluation-methodology.md` — original agent-memory evaluation design.
- `design-research-log.md` — agent protocol and memory-system design history.
- `metrics/` — machine-readable historical measurements.
- `related-work-draft-v2.md` — current related-work draft, with citation caveats.
- `foundations/` — longer technical and literature notes.

For the active fidelity method itself, also read `docs/fidelity/RULEBOOK.md` and
`docs/fidelity/DEPTH_TESTING.md`.

## Evidence States

Use these exact values in the ledger:

- `verified` — checked against a named primary artifact at a recorded revision.
- `partially-verified` — the core event is real, but a count, cause, or citation remains open.
- `needs-verification` — plausible field note with no completed artifact check.
- `refuted` — failed contact with the repository, oracle, issue tracker, or history.
- `historical` — was true at the cited revision but must not be used as current status.

Never silently rewrite a refuted or corrected claim. Preserve the original note,
record the correction, and use the pair as data about agent-generated research.

## Open-Notebook Policy

Keeping this work public is useful: it timestamps hypotheses, exposes negative
results, makes methodology inspectable, and reduces retrospective storytelling.
The costs are manageable if labeling is disciplined:

- Do not present drafts as accepted or peer-reviewed work.
- Do not commit secrets, private player data, raw credentials, or unredacted
  human/agent transcripts without consent review.
- Record model/tool versions and dates when available.
- Prefer immutable or versioned evidence: commits, PRs, scenario files, oracle
  transcripts, manifests, issue exports, and machine-readable metrics.
- Keep analysis separate from product requirements. The game must not be shaped
  merely to make a preferred research result easier to obtain.

## Cron and Agent-Writing Contract

Different research crons may serve different tracks. Every run should declare:

1. research track and question;
2. time window and sources actually accessible;
3. new observations versus repeated context;
4. every proposed claim and its ledger state;
5. contradictions or corrections;
6. artifacts created or updated.

A writing cron may synthesize verified ledger entries into prose. It must not
promote `needs-verification` notes into facts. A monitoring cron should collect
evidence and contradictions, not manufacture a narrative merely because it ran.
