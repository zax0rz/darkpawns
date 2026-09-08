# Luna handoff: entry lifecycle proof

Use the user's chosen GPT-5.6-luna with extra-high reasoning. This document is a
ready-to-run prompt, not an instruction to launch another task automatically.

## Starting context

Work in `/home/zach/darkpawns` on `codex/entry-flow-fidelity`. The repair and its
tests are currently uncommitted, including untracked evidence and a moved browser
client. Preserve them. Inspect status and diff before editing; do not reset,
clean, switch branches, or start from main and lose this baseline. No other agent
should mutate this checkout while you work.

Read AGENTS.md, docs/fidelity/RULEBOOK.md, docs/fidelity/DEPTH_TESTING.md,
docs/briefs/2026-09-08-entry-flow-fidelity.md, and docs/fidelity/depth/entry.tsv.
C source and darkpawns-c-oracle are read-only. C call paths are the authority.

## Goal command

Create one goal with this objective, without inventing a token budget:

> Prove the repaired Dark Pawns entry lifecycle from name submission through
> persisted character acceptance, disconnect/reconnect, and first world entry.
> Cover browser-protocol and telnet boundaries, case-insensitive saved identities,
> failure-safe persistence, and God/mortal RNG boundaries. Produce reproducible
> tests and evidence, fix only confirmed divergences within this lifecycle, and
> deliver an honest proof matrix with remaining gaps. Do not deploy or modify
> production data.

## Bounded acceptance criteria

1. Trace C name lookup/retry, password selection, stat acceptance save, and
   level-zero do_start paths. Record exact source sites for each assertion.
2. Exercise actual local WebSocket JSON and telnet listeners against a disposable
   PostgreSQL store: existing Aiko entered as aiko/AIKO routes to authentication;
   unknown name N then existing name reruns lookup; accepted new character
   survives disconnect at MOTD and at menu and enters once on reconnect.
   Verify the same persisted ID, no duplicate rows, and no lost accepted stats.
   Keep real browser rendering proof distinct from JSON protocol proof; retain
   and run scripts/entry-client.test.mjs. Do not infer visual/secret echo proof
   from server responses alone.
3. Prove lookup/count/insert/entry-save failures close or otherwise prevent world
   entry, never fall into creation on unavailable storage, and never mutate the
   winning record. Exercise two concurrent case-variant creation sessions, not
   only two SQL inserts. Prove no losing candidate is registered or enterable.
4. Extend lifecycle oracle coverage only as needed for fresh God and ordinary
   mortal acceptance/entry, at seeds 1,2,3,5,8. Inspect --show-oracle to establish
   the intended branch executes. Assert relevant draw counts/values and that
   restore consumes zero draws. A no-database oracle cannot prove persistence;
   use separate storage evidence. Do not alter normalizers to hide dialogue.
5. Run the existing creation/retry and God smoke scenarios, explicit PostgreSQL
   tests, browser handler tests, depth validator, and required repository gates.
   Discover current harness commands from the repository. Run the full oracle
   census once after focused checks pass; classify unrelated failures with
   reproducible evidence instead of repairing the entire game. Rebuild the site
   if browser source changes. Follow AGENTS formatting and commit requirements.
6. Update entry.tsv and evidence with actual commands, outcomes, proof boundaries,
   and remaining gaps. Split broad rows when only a subset is proven; do not
   promote a broad row on narrow evidence. Provide a reviewable final diff and
   report any unfulfilled acceptance criterion; do not call it complete merely
   because time or usage runs low.

## Scope and stopping rules

Do not expand into the full nanny matrix: name-validation combinations, race/help
and choice parsing, all password policies, all menu actions, bans/wizlock,
usurp/unswitch, deleted-name reuse, or every identity consumer belong to later
bounded goals. Record newly discovered issues with C sites and repros. Repair a
shared cause only when necessary for this goal, and audit its immediate siblings
under R5c. Do not remove account protections or redesign authentication.

No production SQL writes, collision resolution, deployment, merging, or publishing.
The known production aiko/test/brenda69 collisions need a separate owner-reviewed
record proposal. Do not choose a winning character. Use only disposable local
fixtures; stop only the test processes you created. Local test services from the
repair run were shut down, so establish and clean up your own fixtures.

If the harness cannot express a case, first attempt a narrowly scoped fixture or
test seam. If it requires a broader harness redesign, deliver the evidence and
concrete follow-up scope without pretending the branch is proven. Honor the goal
tool's completion and blocked-state rules.
