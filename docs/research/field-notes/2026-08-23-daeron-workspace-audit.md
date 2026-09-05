# Daeron Workspace Research Audit — 2026-08-23

## Provenance

Read-only audit of `/Users/zach/.openclaw/workspace-daeron` on the `mac-mini`
host, plus the referenced `/Users/zach/darkpawns/RESEARCH-LOG.md` and
`/Users/zach/brain/darkpawns/research-log.md`. This note records promising
material for later verification. It does not import secrets, `.env` contents,
private transcripts, or agent relationship/persona records.

## High-Value Material Found

### Plausible-but-wrong semantic paraphrase

Daeron identified the historical `yank` implementation as a compact example of
AI port drift: it reportedly compiled, was registered, and did something
reasonable while differing from C in exact messages, missing branches, position
threshold, pronoun rendering, and mob-follower handling. The accompanying thesis
is preserved in `drafts/2026-07-24-ai-slop-squash-yank.md`.

Its proposed general claim is useful but unverified: model capability alone did
not distinguish the faulty port from the repair; grounding and verification
protocol did. The original C handler, Go history, tests, and issue DP-1214 must
be checked before publication.

### Dated research ledger

Daeron's longer research log contains contemporaneous entries on:

- the first executable C-oracle run and its initial divergences;
- verified-dead-code and live-call-path failures;
- wiring/reachability gaps;
- formula and magnitude drift;
- taxonomy of `simplified` implementations;
- crawler false positives and fabricated references;
- multi-model brief and review workflows;
- rough token and issue-throughput estimates;
- silent infrastructure divergence, including CWD-relative data;
- institutional-memory and enforcement-tax hypotheses.

These entries are more valuable as a claim inventory than as polished prose.
Many counts refer to historical snapshots and some citations are demonstrably
wrong, so they should be migrated through `EVIDENCE_LEDGER.tsv`, not copied into
a paper wholesale.

### Agent-generated research correcting itself

The brain research log contains an amended weekly digest. Its original headline
named a nonexistent `find_mob_start`, nonexistent commit, and incorrect Linear
citations. A later agent checked the tree and forced a correction. The underlying
silent-failure story survived, but its claimed mechanism did not.

This is itself research evidence: an agent can maintain fluent institutional
memory that fails contact with primary artifacts. The correction supports the
same external-referent principle as the C oracle, now applied to research notes.
Before using this as a case study, preserve the original digest revision and the
correcting revision by commit hash.

## Lower-Value or High-Risk Material

- `research/AIIDE Research.md` is generic venue background and should not anchor
  related work without a new literature search.
- Daily dream reports often summarize inaccessible transcripts, stale metrics,
  or unrelated system health. Their source declarations are useful; their
  narratives are not automatically evidence.
- PR-review logs frequently say `oracle: clean` without preserving invocation,
  scenario matrix, seed, or transcript. Treat this as a lead only.
- Relationship, persona, and emotional-memory files are outside the fidelity
  paper's present scope and may raise consent/privacy questions.

## Recommended Cron Improvements

Separate collection from synthesis:

1. A frequent evidence cron records new PRs, scenarios, manifests, corrections,
   usage records, and contradictions as ledger candidates.
2. A twice-weekly writing cron selects only verified or explicitly provisional
   entries and develops one named research track.
3. A periodic audit cron samples citations against Git, C/Go source, oracle
   artifacts, and the issue tracker, recording refutations rather than erasing them.

Every cron report should state which sources it could not access. `No findings`
and `all clear` are invalid conclusions when the underlying transcripts or
canonical artifacts were unavailable.
