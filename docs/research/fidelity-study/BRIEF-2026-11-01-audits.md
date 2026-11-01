# Brief: the witness audit and the normalization audit

**For:** Sol (OpenResearch). **Written by:** Claude, 2026-11-01, approved by Zach. Read [`STUDY.md`](STUDY.md) first; its definitions (at `56571bbc6` or later) govern the coding here.

## Why these audits

The screening register ([`screening/`](screening/README.md)) counts the checking failures the project *noticed*: 16 A and 17 B. Both groups may be bigger than the record shows, because an A or B failure that nobody noticed leaves no trace. These two audits look for the unnoticed ones directly:

- **The witness audit tests A.** Does each scenario actually reach what its name, comment or manifest row claims?
- **The normalization audit tests B.** For every byte class the normalizer erases or masks, is there at least one scenario that could still fail on it?

Neither audit fixes anything. They measure.

## Ground truth (checked 2026-11-01)

- **Scenarios:** 1,841 files in `cmd/dp-oracle-diff/scenarios/`. 38 declare `keep-ansi`; 23 declare `keep-prompts`.
- **The normalizer:** `internal/oraclediff/normalize.go`. `Normalize` (ordinary mode) runs `normalize(raw, false)`; `NormalizeKeepANSI` skips only the ANSI strip; `NormalizeKeepPrompts` compares raw captured text, masking only the `users` Login@ clock and the `show player` Started/Last dates. The harness modes are documented in `cmd/dp-oracle-diff/README.md` ("Raw ANSI proof mode", "Raw prompt proof mode").
- **Latest release census:** `/home/zach/Archives/darkpawns/oracle-runs/2026-10-31/v0.1.8-release/` (1,841 scenarios; 1,836 passed, 4 expected, 1 infra). Each scenario's **normalized** C and Go blocks are in `census-dump/<scenario>.txt` and in `seed*/initial-logs/attempt.combined/<scenario>.capture1/blocks.json`. **Raw bytes are not retained**: anything the normalizer erased is gone from the archive.

## Audit 1: witness

**Question:** for each scenario, does the C oracle's output show the event the scenario claims?

1. **Write your predictions first.** Before reading any output, record in a dated file what share of scenarios you expect to fail the witness test, and why. This is the pre-registration; don't edit it afterwards.
2. **Sampling frame:** all 1,841 scenarios, sorted by filename. Draw a simple random sample of **150** with `random.Random(20261102).sample(frame, 150)`. Freeze the frame and its sha256, as you did for screening.
3. **For each sampled scenario, record:**
   - **the claim:** the filename, the header comment, and every manifest row in `docs/fidelity/depth/*.tsv` that cites the scenario as proof;
   - **the witness:** the line(s) in the release census's normalized C blocks that show the claimed event (a death message, a successful steal, a cast spell's effect, the post-quit menu);
   - **the verdict:** `witnessed`, `not-witnessed` (the claimed event is absent from C's output), `refused` (C refused the action or its setup: a peaceful room, "Cast what?!?", a failed login), or `unclear` (with the reason).
4. **Mechanical sweep, all 1,841:** flag scenarios whose name or comment claims death, a kill, success, or a specific spell, and whose normalized C output contains a refusal string (for example "Cast what?!?", a peaceful-room refusal, "Huh?!?", a failed-login message) or lacks the claimed event's message. Hand-check every flag. This catches what the sample misses; report it separately from the sample.

**Coding:** a `not-witnessed` or `refused` scenario that a manifest row cites as proof is an **A candidate** (a named check stopped short of its claim). A refused *setup* step under a PASS is a **C candidate** under STUDY.md's setup rule. A scenario no manifest row relies on is a weakness, not an incident; count it, but separately.

## Audit 2: normalization

**Question:** for every byte class the normalizer changes, could some scenario still fail if the port got it wrong?

1. **List every rule** in `normalize.go`, including at least these:
   - ANSI escapes stripped;
   - line terminators folded (CRLF, LFCR, CR become LF) and trailing spaces and tabs trimmed; leading whitespace trimmed on auto-exit lines;
   - prompt-only lines collapsed to `<PROMPT>`, the `> ` prompt prefix removed, vitals prompts masked;
   - `HP:`, `Mana:` and `Move:` values masked as `<VITALS>`;
   - wall-clock, compile-time, Last Death, Started/Last and Login@ stamps masked;
   - the rolled stat block masked as `<ROLLED_STATS>`;
   - "volatile" lines dropped (the `volatileLine` pattern: uptime, player counts, version strings, MOTD dates);
   - leading and trailing blank lines trimmed.
2. **For each rule, record:**
   - **what it erases**, as a player would see it;
   - **mask or erasure:** a *mask* hides genuinely run-varying bytes (the wall clock); an *erasure* hides deterministic, player-facing bytes the port could get wrong;
   - **coverage:** which scenarios run in a mode that keeps this class (`keep-ansi`, `keep-prompts`, or none), and which C sites emit it;
   - **verdict:** `covered` (a scenario keeps the class on a surface where C emits it) or `exposed` (no scenario could fail on it).
3. **Look for over-broad patterns.** Some rules match more than their comment says. For example, check whether `volatileLine` or `statusVitals` can match ordinary game text (a mob's speech containing "version", a help file, a `score` line), and grep `src/` for C output that would be dropped or masked. Cite C lines from `src/` (R5g), never from the oracle checkout.
4. **Show that coverage can fail (R5h), sparingly.** For up to **5** classes you code `covered`, prove a scenario would fail: in a private clone (never the shared checkout, never pushed), make the smallest Go change that breaks that byte class, and run only the covering scenario through `scripts/census.sh start --scenarios <name>`. A green under the broken Go means the coverage is false; code that class `exposed`. Retain every run.

**Coding:** each `exposed` erasure is a **B exposure**. It's a place where an unnoticed B failure could hide, not an incident by itself. If you find an actual divergence hiding in an exposed class, record it as a new finding with its evidence, and stop and report it; don't fix it.

## Rules

- Read-only in `/home/zach/darkpawns`: don't edit, commit, push, branch, stash or switch branches there. Work in your own clone or under `~/wt/`.
- Censuses only through `scripts/census.sh`, targeted with `--scenarios`, at most 36 workers; never the underlying runner. No full census. The witness audit needs no runs at all.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Don't invoke another agent, model or paid service. When done, stop and report; Claude reviews it in Zach's session.
- Never invent a scenario name, commit, line number or quote. Unknowns go in an open-questions list.

## Deliverables

In a new `audits-v1/` folder in your workspace:

1. `predictions.md`: dated, written before any output was read.
2. `witness-sample.jsonl` and `witness-sample.md`: the 150 records, the frame, its sha256 and the seed.
3. `witness-sweep.md`: mechanical flags and their hand-check verdicts.
4. `normalization-rules.md`: one row per rule, with the verdict and evidence.
5. `mutation-runs.md`: each R5h proof attempt, its census directory and its verdict.
6. `audit-handoff.md`: the counts (witness verdicts in the sample; A and C candidates; covered and exposed classes; any new findings), how your predictions compared, and what you couldn't resolve.
