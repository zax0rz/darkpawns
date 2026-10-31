# 2026-10-31: Two incident records, when the comparison itself was wrong

## Scope

- **Tracks:** legacy-port fidelity (oracle validity, normalization blind spots)
- **Purpose:** two incident records in the six-field format used by the OpenResearch session of 2026-10-31 (exact revisions; inputs and environment; observable difference; why the checks missed it; fix and new check; evidence that the check fails before and passes after). They complement that session's five incidents, and are written so the two sets can be compared.
- **Status:** the facts below were re-checked on 2026-10-31 against `src/`, git history, PR text and the retained binaries (SHA-256 recomputed). No scenario was re-run today. Each record says what is reproduced and what is only documented.

## The shared mechanism

The usual story of a missed defect is a test that covers too little. These two records are different: the test covered the behaviour, and *the comparison* was wrong.

- **Incident A:** the reference binary was not the reference. A compiler flag changed what the C printed, and the port copied the accident.
- **Incident B:** the normalizer declared a class of player-facing bytes to be noise, so no census could see a difference in that class, however many scenarios covered it.

In both cases a green result was the system working as built. The defect sat in what the build declared to be ground truth.

---

## Incident A: the `-O2` reference oracle, and the port that copied it

### 1. Exact revisions

| Item | Value |
|---|---|
| Faulty reference | shared `DP_ORACLE_BIN` built `-g -O2`, sha256 `66fe1516e3a482553ae1422c4da48515a609cdbdd774c0035b060ad62b075879` (retained: `~/Archives/darkpawns/oracle-runs/2026-09-28/oracle-promotion/binaries/circle-before-promotion-O2`) |
| Corrected reference | `-g -O0 -fcommon`, darkpawns-c-oracle `46940fc`, sha256 `25fc83e5003b4118632068a632b3c3e4e5b41a8ca996f24de6242b4658b6c02d` (same directory, `circle-46940fc-O0-shared-path`) |
| Go commit that removed correct code to match the faulty reference | `49c4d1167` (PR #691, 2026-08-28) |
| Go fix | PR #1682, merge `7b59e3faa` (2026-09-28); review follow-up `b75852523` |
| Rule changes | `cc1105906` (R1a extended to the `-O0` build contract, R5h added); later `cmd/dp-oracle-diff/reference-oracle.sha256` pin (`7a55e62c5`, 2026-09-29) |

### 2. Inputs and environment

Any scenario that reaches a C report built by appending to its own buffer. The clearest is `guild-practice` (`practice`, `practice kick` twice, `practice`), against `list_skills()`:

```c
/* src/spec_procs.c:165-171 */
if (!GET_PRACTICES(ch))
  strcpy(buf, "You have no practice sessions remaining.\r\n");
else
  sprintf(buf, "You have %d practice session%s remaining.\r\n", ...);

sprintf(buf, "%sYou know of the following %ss:\r\n", buf, SPLSKL(ch));
```

The last line passes `buf` as both destination and `%s` source. That is undefined behaviour in C. DEV-SETUP counts about 180 such self-aliasing appends in the source.

### 3. Observable difference

At `-O0`, the overlapping `sprintf` happens to keep the prefix, and the player sees `You have 4 practice sessions remaining.` before the skill list. In the `-O2` build the prefix was lost and the report began at `You know of the following skills:`. The same UB class truncated reports in `show`, `stat`, `mlist`, `home`, `hcontrol`, clan info, the infobar, the editor listing and the start-room vision (R1a lists them).

Which bytes did 1990s players see? The answer depends on the compiler that built the original server, and nobody recorded it. Choosing `-O0` is a decision about what the reference *is*, not a discovery. R1a records that decision: a compiler accident is not a specification.

### 4. Why the existing checks missed it

- **The port matched the oracle exactly.** That was the problem. PR #691 had the C-logic prelude in Go, saw the `-O2` oracle omit it, and **deleted the correct code**, leaving this comment: "The source's practice-count prelude is overwritten by its overlapping sprintf(buf, "%s...", buf, ...) call on the live C oracle; keep the observed bytes rather than inventing that line in Go." Each rule was followed (trust the oracle; don't invent bytes), and the result was wrong.
- **The build contract existed, but nothing enforced it.** DEV-SETUP had specified `-O0` since `fd9fa1506` (PR #567, 2026-08-13), two weeks *before* #691. Nothing checked that the shared binary was built that way. The census had no pin on the binary's identity.
- **A rebuild was the only detector.** The defect appeared only when the reference was rebuilt reproducibly; every census against the `-O2` binary was green.

### 5. Fix and new check

- Go follows the C logic, not the observed accident (`7b59e3faa`, `b75852523`; `pkg/game/practice.go` now renders all three prelude branches).
- The reference is rebuilt at `-O0` with `-ffile-prefix-map`, so any checkout builds identical bytes, and promoted only through DEV-SETUP's procedure (full census green against the candidate, old binary and both hashes retained).
- `scripts/census.sh` refuses any binary whose SHA-256 differs from `cmd/dp-oracle-diff/reference-oracle.sha256`, unless an override is set explicitly. This check closes the class: it fails on a silently replaced reference.
- R5h: a fixed report must have a test that fails when the fix is reverted. Some of the restored reports first went green on fixtures that could not fail (the second R5b instance after DP-1212).

### 6. Fails before, passes after

| Run | Go | Reference | Result | Retained at |
|---|---|---|---|---|
| Baseline | pre-#1682 | `-O2` | green (historical; no single retained run identified yet) | — |
| Swap only the reference | pre-#1682 | `-O0` | **21 FAIL**, 997 pass, 1 stale (`2026-09-27T21:45`) | `~/Archives/darkpawns/oracle-runs/2026-09-27/oracle-reproducibility/full-census-o0-baseline.log`, `dump-o0-baseline/` |
| Fix | `7b59e3faa` | `-O0` candidate | 1,018 pass, 6 expected, 0 fail (1,025 scenarios) | `.../2026-09-28/oracle-promotion/MANIFEST.md`, `census-full.log` |

The 21 failures include `guild-practice`, both `character-creation` scenarios, six clan scenarios, `home*`, `hcontrol-depth`, three infobar scenarios and three `mlist` scenarios. This is a controlled comparison: Go held fixed, only the reference changed.

**Gap:** a retained green census against the `-O2` binary at the exact pre-#1682 Go commit would make row 1 a fact instead of history. Candidates: any 2026-09-2x run directory whose manifest names the `66fe1516…` binary. Re-running is possible, since both binaries are retained.

---

## Incident B: bytes the normalizer declared to be noise

### 1. Exact revisions

| Item | Value |
|---|---|
| Terminator instance | PR #532 (2026-08-12) folded cutthroat into a shared message map; PR #537, merge `4ed61c041` (same day), restored the C terminator |
| Color instance | PR #2231, merge `ee12a71c2` (2026-10-30): ANSI audit plus fixes F1/F2 |
| Rule change | R1b, `eb6c70c2a` (approved 2026-10-30) |

### 2. Inputs and environment

- **Terminator:** a character without the skill types `cutthroat`. C (`src/new_cmds.c:561`) sends `You're not trained in slitting throats!\n\r`, line-feed first, unlike its siblings.
- **Color:** any damaging melee hit, with a character whose color level is at least `C_CMP`. C wraps the attacker's message in yellow and the victim's in red (`src/fight.c:1000-1013`), and colors the wounded-band `BLEEDING` line red at `C_SPR` (`src/fight.c:1594-1597`).

### 3. Observable difference

- After #532, Go sent the cutthroat line with `\r\n`, because the map's callers append `\r\n` uniformly.
- Before #2231, every damaging hit in Go was uncolored, and the BLEEDING line was never red. For a player with color on, that is the most visible output in combat.

### 4. Why the existing checks missed it

The differential harness normalizes the transcripts before it compares them: it folds line terminators and strips ANSI escapes. Each rule exists for a reason (transport noise, client differences), and each one quietly asserts that a whole byte class is not part of the game.

- The cutthroat change passed green because `\r\n` and `\n\r` compare equal after folding. It was caught by a person reading the C, not by the harness.
- Before the audit, **17 of 1,801 scenarios** kept ANSI (`keep-ansi` disables the stripping rule), and **none of them involved combat** (audit §2). Combat color had never been compared, though combat is among the most heavily tested surfaces.

Coverage was not the issue. Hundreds of scenarios exercised the code paths. The comparison was blind in exactly the dimension that differed.

### 5. Fix and new check

- #537 restores `\n\r` for cutthroat; #2231 restores the yellow/red sandwich and the red BLEEDING line at three Go call sites.
- Two `keep-ansi` scenarios, `damage-message-color-depth` and `bleed-line-color-depth`, plus revert-red unit tests (`pkg/combat/damage_message_color_test.go`).
- A 355-site machine-extracted inventory of C color calls (`docs/fidelity/audit-reports/ansi-color-c-inventory.tsv`), matched to the Go counterpart. The static pass finds a surface nobody wrote a scenario for; the differential pass finds an RNG-gated branch the static pass can't (audit §2).
- R1b: any byte class a normalizer erases needs its own proof mode and at least one scenario per surface that can fail.

### 6. Fails before, passes after

- **Color:** the PR and audit §9 report both new scenarios red before the fix and green at seeds 1, 2, 3, 5 and 8 after, and the unit tests observed failing on revert. **Gap:** I found no retained oracle-run directory for these probes. Re-run the two commands in audit §9 at `ee12a71c2^1` (expected red) and `ee12a71c2` (expected green), and retain both through `scripts/census.sh start --scenarios damage-message-color-depth,bleed-line-color-depth`.
- **Terminator:** no automated check can fail on it while the terminator fold is in place, which is the point of the incident. The evidence is the C line, the #532 diff and the #537 commit message. A byte-level test of the Go string would be the R5h-style check; I haven't confirmed one exists.

---

## Comparison with the OpenResearch set

The OpenResearch set contains two incidents of the same family: the 64-bit `load mob` crash (PF-066: the reference was wrong on a new platform) and the social-loader overflow (the "port bug" was in C). Its color-setting incident ("the comparison setting was a comment") is a third normalization case. Together they suggest the first paper has two halves:

1. **Tests that cover too little:** the prefix-only logout test, the "combat-death" scenario that never reached death, and the coincidence greens behind R5h.
2. **Comparisons that are wrong:** a reference built differently, a reference on a different platform, a reference with its own memory bugs, and a normalizer that erases a byte class.

The second half is the less common finding. Work that validates a migration against the original executable usually treats the executable as fixed truth. These records show four separate ways it wasn't.

## Open

- Retain a green `-O2` census at the pre-#1682 commit (Incident A, row 1).
- Re-run and retain the two color proofs before and after (Incident B).
- Check whether the cutthroat terminator now has a byte-level test.
- Ledger rows PF-091 and PF-092 cite this note.
