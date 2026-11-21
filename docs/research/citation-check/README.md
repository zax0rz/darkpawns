# C citation check

`check_citations.py` checks every `src/<file>:<line>[-<end>]` citation in:
- the depth manifests (`docs/fidelity/depth/*.tsv`);
- the test-audit ledgers (`docs/research/test-audit/*/ledger.tsv`);
- the evidence ledger (`docs/research/EVIDENCE_LEDGER.tsv`).

```bash
python3 -I docs/research/citation-check/check_citations.py "$PWD" /tmp/problems.tsv
```

For each citation it reports one result:

| Result | Meaning |
|---|---|
| `no-such-file` | the cited file isn't in `src/` |
| `out-of-range` | the cited lines are past the end of the file |
| `anchor-match` | the row quotes a C string (8+ characters, in `"..."` or backticks) found within 2 lines of the cited range |
| `anchor-drift` | the quoted string is in the cited file, but not near the cited lines; the report gives where it actually is |
| `resolves-no-anchor-in-file` | the lines exist; the row quotes text that isn't in that C file (a paraphrase, or a Go string) |
| `resolves` | the lines exist and the row quotes nothing, so a script can't judge it |

Only the first four are checked mechanically. `problems-2026-11-21.tsv` lists every `no-such-file`, `out-of-range` and `anchor-drift` row from the first run on 2026-11-21. Of about 19,600 citations it found 3 to files that don't exist, 10 past the end of a real file, and 173 drift candidates. A drift candidate isn't necessarily wrong: a row may quote a message while citing the start of the function that prints it. Each needs a short read.

## Second run, after the fixes (2026-11-21)

GLM flash's #2972 fixed 43 wrong citations: 3 to nonexistent files, 10 past end of file, and 30 drifted. It marked the other 143 flagged rows correct as is, mostly function-start, call-site and spell-matrix template citations; Claude sampled them, and they held up. `fixes-2026-11-21.tsv` records each decision with its grepped evidence.

Flash also found a checker bug. A backtick pair holding fewer than 8 characters (`` `stand` ``) made the pattern pair the wrong backticks. The checker now matches every quote or backtick pair first and then filters, and it skips bare identifiers such as `mag_damage`. Those are code references, not quoted messages.

On main after both fixes: `no-such-file` 0, `out-of-range` 0, `anchor-drift` 132 (all among the reviewed correct-as-is set), `anchor-match` 749.
