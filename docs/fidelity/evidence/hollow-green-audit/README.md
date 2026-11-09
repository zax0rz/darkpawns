# Hollow-green audit (DP-1511)

Screening audit for **hollow greens**: oracle scenarios whose manifest row
claims an event the transcript never shows because both engines refused the
same setup step. The harness's warmup-refusal gate
(`internal/oraclediff/setuprefusal.go`) matches whole lines exactly, so a
refusal containing a name ("You don't see a chalice here.") slips through —
DP-1511. This audit finds what the gap may have hidden; fixing the gate is
Zach's (DP-1511), not this audit's.

## Files

| file | contents |
|---|---|
| `audit_hollow_greens.py` | the screening script (stdlib only, run with `python3 -I`) |
| `suspects.tsv` | every (row, scenario) pair that fired ≥1 signal; `note=intended` rows legitimately claim a refusal and are excluded from the count |
| `unauditable.tsv` | rows that cannot be checked against the retained seed-1 dump, with the reason — never guessed |

## Run

```bash
python3 -I audit_hollow_greens.py \
  --repo /path/to/worktree \
  --census ~/Archives/darkpawns/oracle-runs/2026-11-07/claude-rehearsal-1107
```

`--census-ref` defaults to the rehearsal census HEAD
(`b7fdd6ef58eb48ac407387ab62cdb0ed44a58863`).

## Method

The audit universe is the **census-commit snapshot**: depth manifests,
scenario files, `src/` and both `lib/misc` data tables as they existed at
`b7fdd6ef5`, read through one `git archive`. Rows repaired on main after the
census (the DP-1510 repair among them) are judged in their census-era form —
the form whose green this census produced. The C dumps are the seed-1
normalized blocks the census retained (`census-dump/`).

Signals (per `oracle-green*` row × scenario named in its proof):

- **warmup-refusal** — the scenario has `[warmup]` steps and the *first* C
  actor probe block contains a setup refusal. Warmup output is discarded by
  the harness (`oraclediff.RunWarmup` drops the blocks), so a refusal
  surfacing in the first probe is the retained fingerprint of a failed setup —
  the DP-1510 shape. Refusals in later sibling probes are often the intended
  claim of other rows sharing the scenario and are not attributed, and the
  signal is suppressed outright when any C actor block shows the
  refused-for object being used (`You recite|quaff|zap|brandish|use|eat|hold`,
  plus C's `hold` success verb `grab`, src/act.item.c:1395) — the setup had
  already provided the item and the refused `get` was redundant (the
  wand/staff-olc vehicles hold and fire the item right after the refused
  `get`; the quaff-lump rows quaff successfully in their first block).
- **claimed-absent** — two tests must both hold. (1) No string literal printed
  by any player-output call on the row's `c_site` lines appears in any C probe
  block. Literals are extracted from the snapshot's `src/` (R5g: every
  fragment grepped in the authoring session), format codes stripped, escaped
  newlines split. The presence check widens to a ±(6/12) line window and to
  the row's own `lib/misc/socials` / `lib/misc/messages` entries; only
  *distinctive* fragments (≥8 chars with a space) may suppress, so a generic
  neighbor like the bare `failed.` cannot mask a hollow. Log-fed buffers
  (`sprintf` straight into `mudlog`/`log`) are skipped. (2) The row's claimed
  command actually failed somewhere: some actor block for the manifest
  command's first word contains a refusal or a spell/skill failure
  fingerprint (`You failed.`, `You lost your concentration!`) that no sibling
  row claims. Cited ranges often span sibling branches whose gates
  legitimately stay silent while the claimed event runs fine — silence or
  success in the claimed command's own blocks means the row is not hollow,
  even when the citation is misaligned. Rows whose manifest command matches
  no block (data-table claims) fall back to test (1) alone.
- **probe-refusal** — every C probe block (actor and peers) is empty or a
  refusal.

Rows whose case_id, notes or scenario header name (or quote) the refusal get
`note=intended`.

## Known limits (checked by hand during the audit)

The screening is deliberately over-inclusive; confirmed suspects are repaired
by hand afterwards. Hand verification found three residual false-positive
classes that a transcript-only audit cannot separate from hollows:

1. *Mis-cited gate*: the claim's bytes are in the dump, but the cited line
   names a gate the scenario passes through (`stat.file-success`,
   `stat.fresh-mortal-idnum`, `cast.unknown`, `lines.decimal-prefix-atoi`,
   `weather.sky-variants`).
2. *Claim in a callee*: the cited range names the dispatcher; the claimed
   bytes print in a function it calls, beyond the presence window
   (`peek.failure-original-look-argument`).
3. *Claim in scenario-owned data*: Lua-script text (`lua.bind-*` rows).

All three were verified against the dumps: the claimed event *did* happen.
They stay in `suspects.tsv` (their evidence line is still accurate) and are
dropped at confirmation time.

## Repairs

Confirmed suspects are repaired scenario-side (batch PRs, targeted census
only) — see the audit PR for the running list. Scenarios whose rows cite
`spells.c`, `magic.c` or `spell_parser.c` are excluded while DeepSeek works on
them and are listed for that effort instead.
