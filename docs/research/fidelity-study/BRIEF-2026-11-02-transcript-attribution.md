# Brief: attributing early mistakes from the OpenClaw transcripts

**For:** DeepSeek. **Written by:** Claude, 2026-11-02, approved by Zach. Read [`STUDY.md`](STUDY.md), the [slop-track README](slop-track/README.md) and the field note [`2026-11-02-rules-without-a-check.md`](../field-notes/2026-11-02-rules-without-a-check.md) (PF-098) first.

## The question

The slop table ([`slop-track/slop.tsv`](slop-track/slop.tsv)) names the agent that introduced a mistake only where git shows it. 118 of 178 coded mistakes are `unknown`, mostly because the early OpenClaw agents committed under Zach's identity (`zach@karl-havoc.labz0rz.com`). The surviving OpenClaw transcripts record which model wrote which code. **For each unknown birth in the transcript window, find the session and the model that wrote it.**

## The target (checked 2026-11-02)

Of the 118 unknown rows, 94 have a birth commit:

| Birth date | Rows | Transcripts |
|---|---:|---|
| before 2026-04-22 | 16 | none survive; record `no-transcripts` |
| 2026-04-22 to 05-04 | 48 | the NUC archive (local) |
| 2026-05-05 to 05-24 | 5 | `mac-mini` (over SSH) |
| after 2026-05-24 | 25 | out of scope here |

Work on the 53 rows in the middle two windows.

## Where the transcripts are

- **The NUC archive:** `/home/zach/Archives/openclaw-nuc-2026-11-02/agents/*/sessions/` (1,928 files, 1.1 GB; `.jsonl`, `.trajectory.jsonl`, and `.bak`/`.deleted` copies; covers 2026-04-22 to 05-04).
- **`mac-mini`:** `ssh mac-mini`, then `~/.openclaw/agents/*/sessions/` including `.bak` and `.deleted` copies, and `~/.openclaw/backups/orphan-transcripts-20260711/` (covers 2026-04-27 to 05-24). Read-only.

**These files are private.** They include Zach's personal conversations unrelated to Dark Pawns. So:

1. **Never print, read or paste raw transcript text.** No `cat`, `less`, `head` or full-record reads of these files.
2. **Write one extraction script first** (`extract.py`, in your output folder) that walks the files and emits only, for each assistant tool call that writes or edits a file under a `darkpawns` path: session id, file name (relative to the archive root, or `mac-mini:` plus path), the tool call's timestamp in UTC, the model that produced it, the target file path, and the written code. Nothing else: no user messages, no other tool calls, no prose.
3. Work only from the extraction's output. If you need a field the script doesn't emit, change the script, not your access.
4. The deliverables contain only metadata and code strings (each excerpt at most 3 lines).

## Method

For each target row:

1. From the birth commit's diff (`git show <birth_commit>` in `/home/zach/darkpawns`), pick 1–3 distinctive added lines tied to the row's mistake: a message string, a function signature, an unusual expression.
2. Search the extracted writes for those lines. Candidates are writes to the same file path, or a path that later became it, in the 48 hours before the commit's time.
3. **The match** is the earliest such write containing the line, verbatim or with whitespace differences only. Record its session, timestamp and model.
4. If several sessions or models wrote the line, record the earliest and note the others. If routing switched model mid-session (it did on auth failures), use the model on that specific tool call, not the session's first model.
5. No match: `not-found`, with which strings you searched for.

**Control first.** Before coding any target row, run the method on S001 (yank, birth `070dfe54e`). PF-098 established that `doYank` was first written by `deepseek-chat` in `main` session `8e4697db` at 2026-04-24 22:42 UTC. Your method must reproduce that. If it doesn't, fix the method before going on, and record what you changed. Name the JSON field you read the model from, and show it on the control.

**Bonus, only if cheap:** for each matched session, whether the session ever read `CLAUDE.md` (a file-read tool call on it or `cat` of it; count it from tool calls only, the same privacy rule applies).

## Rules

- **Read-only everywhere:** the archive, `mac-mini` and `/home/zach/darkpawns`. Don't edit, commit, push, branch, stash or switch branches in the repo. Don't touch `karl-havoc` itself; the archive is the copy.
- **Don't copy transcripts** anywhere, including into your output folder or any other service.
- **No census, no oracle runs.**
- **Don't edit** `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- **Don't invoke another agent, model or paid service.** When done, stop and report; Claude reviews it in Zach's session.
- **Never invent** a session id, timestamp or model name. Unknown stays `not-found`.

## Deliverables

In `/home/zach/dp-research/attribution/`:

1. `extract.py`: the extraction script.
2. `control.md`: the S001 result, the model field used, and any method changes.
3. **`attribution.tsv`**, one row per target, tab-separated:
   `id`, `birth_commit`, `birth_date`, `window` (`nuc` | `mac-mini` | `no-transcripts`), `status` (`matched` | `not-found`), `session_id`, `transcript_file`, `tool_call_utc`, `model`, `match_type` (`verbatim` | `whitespace`), `matched_line` (one line of code, at most 120 characters), `other_writers` (other sessions or models that wrote the same line, if any), `read_claude_md` (`yes` | `no` | `not-checked`), `notes`.
   Include the 16 pre-transcript rows with `no-transcripts`.
4. `attribution-notes.md`: matched count by model and by window; the not-found rows and why; how often matched sessions read `CLAUDE.md`; open questions.

## Check before you report

Run this and include its output, which must be empty:

```bash
cd /home/zach/darkpawns
F=/home/zach/dp-research/attribution/attribution.tsv
tail -n +2 "$F" | cut -f2 | sort -u | while read -r c; do
  git cat-file -e "$c^{commit}" 2>/dev/null || echo "MISSING COMMIT: $c"
done
tail -n +2 "$F" | awk -F'\t' '$5=="matched" && $11!=""{print $2"\t"$11}' | while IFS=$'\t' read -r c l; do
  git show "$c" | grep -qF -- "$l" || echo "LINE NOT IN BIRTH COMMIT: $c"
done
grep -c . "$F" >/dev/null || echo "EMPTY FILE"
```

Then stop and report: the control result, matched and not-found counts, the models behind the matched rows, and anything you couldn't resolve.
