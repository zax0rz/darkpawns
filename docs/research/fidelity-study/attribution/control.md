# Control: S001 (yank), birth `070dfe54e`

**Result: reproduced.** `doYank` was first written by **`deepseek-chat`** in
`main` session **`8e4697db`** at **2026-04-24 22:42:11 UTC** — the session, the
model and the minute PF-098 reports.

## The JSON field the model is read from

**`message.model`** — the top-level transcript record is
`{"type":"message","id":…,"timestamp":…,"message":{…}}`, and the assistant
message object carries `model` (a sibling of `role`, `content`, `provider`,
`api`, `usage`). It is present on assistant messages only (1,343 of 3,332
`message` records in the first 40 files probed). The per-call model is read from
the same message object that owns the `toolCall` content item, so a mid-session
model switch is followed per call, not per session.

Supporting fields: `message.content[]` items have `type` (`text` | `toolCall` |
`thinking`), `name` (tool), `arguments`; `message.role` is
`assistant`/`user`/`toolResult`; the record's `timestamp` is the UTC call time;
the session id is the transcript file-name UUID.

## The matched write

| Field | Value |
|---|---|
| session | `8e4697db-74af-409e-bd59-293e5cc5514c` |
| transcript file | `main/sessions/8e4697db-74af-409e-bd59-293e5cc5514c.jsonl.reset.2026-04-25T00-35-02.200Z` |
| tool call UTC | `2026-04-24T22:42:11Z` |
| model | `deepseek-chat` |
| target | `/home/zach/.openclaw/workspace/darkpawns/pkg/game/act_other.go` |
| tool | `write` (whole-file, 20,708 bytes) |
| match type | verbatim |
| matched line | `func (w *World) doYank(ch *Player, me *MobInstance, cmd string, arg string) bool {` |

The same write carries the paraphrase PF-098 names: `"That's weird."` (C:
`"That's wierd."`, `src/act.other.c:1633`), `"%s is in no position to be yanked
around!"` (C: `"$N is is no position to be yanked around!"`, `:1652`) and
`"I see no one here by that name."` (C: `NOPERSON`). All three appear in exactly
one write in the whole archive — this one.

## Method changes made while establishing the control

1. **The tool surface.** File writes arrive as `write` (arguments `path` +
   `content`) and `edit` (arguments `path` + `edits[]` of `{oldText,newText}`);
   `exec` also exists but writes no repo files in the covered window. The
   extractor handles `write` and `edit`.
2. **Which line to match on.** The birth commit `070dfe54e` adds a *different*
   revision of `doYank` from the one in the transcript write: the committed
   function's empty-argument message is `"Yank whom from what?"`, which appears
   in **no** transcript write, while the transcript write's yank messages
   (`"Who do you wish to yank?"`, `"That's weird."`, `"…yanked around!"`) appear
   in **no** other commit. Matching on the paraphrase *strings* alone would have
   produced `not-found` against the birth commit's own text. The method
   therefore matches on lines that are **both** distinctive in the birth diff
   and likely to survive a later revision — here the function signature, which
   is identical in both. (Recorded because it changes the match rule: prefer a
   signature or an unchanged literal over a literal the birth commit may have
   rewritten.)
3. **De-duplication.** The archive holds whole-file `.bak`/`.reset`/`.deleted`
   copies of the same session, so the same tool call appears in several files.
   Records are de-duplicated on `(session_id, call_id)`; the transcript file
   named is the first one walked.
4. **Prefilter.** A line is parsed only if it contains both `darkpawns` and
   `toolCall`, which keeps the 1.1 GB walk fast and never opens a line that
   cannot be a darkpawns file write. Paths in the corpus are
   `/home/zach/darkpawns/…`, `~/darkpawns/…` and
   `/home/zach/.openclaw/workspace/darkpawns/…`; all contain `darkpawns`.

## v2: what the control proves, and what it does not

`S001` is graded **`function-created`**, not `mistake-line`. The matched line is
the `doYank` signature, which is the only line the two revisions share: the
`doYank` committed in `070dfe54e` is *not* the one `deepseek-chat` wrote at
22:42:11. The committed revision's strings (`"Yank whom from what?"`,
`"They aren't here."`, `"They aren't following you!"`) appear in **no** write in
any of the three extracted corpora, so the session that wrote the shipped
paraphrase is `not-found` (search detail in `attribution-notes.md`). The control
therefore reproduces PF-098's *session and model* for the first yank write, and
confirms — rather than contradicts — the 2026-11-03 PF-098 correction that the
committed text is a different revision.

## Coverage note (affects the control's confidence, not its result)

The control session's transcript survives only as a `.reset` copy taken
2026-04-25 00:35. `claude-sonnet` and `gemini` agent directories hold only
`sessions.json` (no transcripts), so work by those two agents in the window is
invisible to this method and will read `not-found`.
