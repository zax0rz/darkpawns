# Attribution notes (v2)

DeepSeek, 2026-11-03. Companion to `attribution.tsv` (69 rows, 15 columns),
`control.md` and `extract.py`. v1 is kept unchanged beside this file.

The question: for each unknown birth in the OpenClaw transcript window, which
session and model wrote it?

## What v2 changes

- **A `match_strength` column.** Every matched row is graded by what its matched
  line actually proves:
  - `mistake-line` — the matched line **is part of the mistake**: the wrong
    message, condition or invented logic that the fix changed.
  - `function-created` — a signature, declaration or call inside the function
    the mistake lives in, but **not** the wrong line.
  - `generic` — a line common enough to be written independently, or a line
    from a different function than the mistake.
- **A retry pass.** For every v1 match that was not `mistake-line`, I searched
  once more for a write containing the mistake's own line — a line the fix
  commit *removed* — inside the 48 h window. Four rows found one and were
  upgraded (`G006`, `S068`, `S078`, `S083`); the other eleven did not.
- **The S001 record** (below) now separates the two `doYank` revisions and
  reports the search for the committed one.

## Headline

**18 of 53 in-window rows matched** (34%); the other 35, plus the 16
pre-transcript rows, are `not-found`.

| | rows |
|---|---:|
| `matched` (nuc window) | 18 |
| `not-found` (nuc window) | 30 |
| `not-found` (mac-mini window) | 5 |
| `not-found` (`no-transcripts`) | 16 |
| **total** | **69** |

### By strength

| Strength | rows |
|---|---:|
| `mistake-line` | 7 |
| `function-created` | 9 |
| `generic` | 2 |

**Read the two groups differently.** Only the seven `mistake-line` rows are
attributions of a *mistake* to a model — a session was found writing the very
line the fix later changed. The nine `function-created` rows are weaker: they
say *which function* the mistake lives in and which session wrote that function,
not that the session wrote the wrong line. They are reported below as
"function created by" leads, not as mistake attributions.

### The seven attributed mistakes (mistake-line)

| Row | Model | Session | Write (UTC) | The wrong line |
|---|---|---|---|---|
| `G044` | `deepseek-chat` | `4ebfc03b` | 2026-04-26 17:36:49 | `player.Stats.Con = 1` — the invented CON floor the fix removed |
| `G077` | `glm-5-turbo` | `7f68ec07` | 2026-05-03 18:46:01 | `buf.WriteString("Someone invisible: ")` — the uncoloured review entry |
| `S042` | `glm-5-turbo` | `b6442111` | 2026-04-26 19:01:10 | `delta := PosFighting - defPos` — the position-damage arithmetic |
| `S057` | `glm-5-turbo` | `9927d65c` | 2026-04-26 20:07:56 | `sendToCaster(ch, "Spell not yet implemented.\r\n")` — the invented line |
| `S068` | `deepseek-chat` | `8e4697db` | 2026-04-25 00:06:30 | `item, ok := p.Equipment.GetItemInSlot(SlotLight)` — the lantern slot lookup |
| `S078` | `glm-5-turbo` | `9927d65c` | 2026-04-26 20:21:07 | `sendToCaster(ch, "You feel informed:\r\n")` — the identify output |
| `S083` | `deepseek-chat` | `8e4697db` | 2026-04-25 00:12:20 | `item.Prototype.Values[1]--` — the unwired light-timer decrement |

`G044` and `S057` are the cleanest specimens: the fix's diff removed exactly the
line the transcript wrote, verbatim.

### The nine "function created by" leads (function-created)

| Row | Model | Session | Write (UTC) | What the line is |
|---|---|---|---|---|
| `G006` | `glm-5-turbo` | `9927d65c` | 2026-04-26 20:19:34 | `castIdentify(level, ch, cvict, ovict)` — the dispatch call |
| `G013` | `deepseek-chat` | `a3299772` | 2026-04-25 05:28:16 | `if hasMobFlag(ch, "aggr24") {` — the branch the fix added a gate to |
| `G032` | `deepseek-chat` | `089773c9` | 2026-04-25 01:41:00 | `reag := checkReagents(ch, SpellMagicMissile, level, "shard of obsidian",` |
| `G046` | `deepseek-chat` | `fb63d15c` | 2026-04-25 04:00:40 | `ch.StopFighting()` — in the attack loop the fix corrected |
| `G084` | `deepseek-chat` | `089773c9` | 2026-04-25 01:39:19 | `func SaySpell(...)` — the signature |
| `S001` | `deepseek-chat` | `8e4697db` | 2026-04-24 22:42:11 | `func (w *World) doYank(...)` — the signature (see below) |
| `S029` | `glm-5-turbo` | `9927d65c` | 2026-04-26 20:25:01 | the `GetValue` instance-override comment |
| `S033` | `deepseek-chat` | `b46661ce` | 2026-04-23 15:47:53 | `func (e *Engine) LState() *lua.LState {` — the signature |
| `S054` | `deepseek-chat` | `e01dcd7e` | 2026-04-26 17:37:38 | `victim.StopFighting()` — in the combat consumers the fix corrected |

In each case the fix changed the *body* around or after this line; the line
itself was not the defect. `G013` and `G032` are the shape where the defect was
an **absence** — the fix only added lines (a `canSee` gate; a `SpellWaterwalk`
reagent check), so no wrong line exists in any transcript to find.

### The two `generic` matches

`G067` matched `line := strings.TrimSpace(scanner.Text())` — a `TrimSpace` in the
*ban-file reader*, while the row's mistake is `DoUnban`'s argument parsing. The
mistake's own line (`argument = strings.TrimSpace(argument)`) has no write in the
window. `S028` matched `ValuesOverride *[4]int // copy-on-write override of
prototype Values` — the *object* override mechanism, while the row is about
*mob* flags. Both are retained in the table for completeness but should not be
read as attributions.

## By model and window

Every match is `deepseek-chat` or `glm-5-turbo` — the two models the NUC archive
covers most heavily in this window. All 18 are in the `nuc` window; the
mac-mini window yielded none.

| Model | matched rows | of which mistake-line |
|---|---:|---:|
| `deepseek-chat` | 11 | 3 |
| `glm-5-turbo` | 7 | 4 |

Eleven distinct sessions carry the 18 matches: `089773c9`, `4ebfc03b`,
`7f68ec07`, `8e4697db`, `9927d65c`, `a3299772`, `b46661ce`, `b6442111`,
`e01dcd7e`, `e805c210`, `fb63d15c`. No row had a second session or model writing
the same line, so `other_writers` is empty throughout.

## S001: two `doYank` revisions, and the search for the committed one

The control is a `function-created` match, and v2 records why that matters.

**The two revisions are different code.** `deepseek-chat`'s write (session
`8e4697db`, 2026-04-24 22:42:11 UTC) and the revision committed in `070dfe54e`
(2026-04-24 23:53 UTC) share only the function signature
`func (w *World) doYank(ch *Player, me *MobInstance, cmd string, arg string) bool {`.
They differ in every yank string:

| C (`src/act.other.c`) | deepseek-chat's write (22:42) | committed in `070dfe54e` |
|---|---|---|
| `"Who do you wish to yank?"` (:1629) | kept | `"Yank whom from what?"` |
| `NOPERSON` (:1631) | `"I see no one here by that name."` | `"They aren't here."` |
| `"That's wierd."` (:1633) | `"That's weird."` | absent |
| — | — | `"They aren't following you!"`: an invented follower rule |
| `"$N is is no position to be yanked around!"` (:1652) | `"%s is in no position to be yanked around!"` | absent |

**Search for the session that wrote the committed revision: not found.** I
searched all three extracted corpora (NUC 2,940 writes; mac-mini 556; orphan
backup 49) for every distinctive string of the committed revision:

| Needle | hits |
|---|---:|
| `Yank whom from what?` | **0** |
| `They aren't following you!` | **0** |
| `They aren't here.` | 14, all in other commands' files (`pkg/session/wizard_cmds.go`, `pkg/command/skill_commands.go`) |
| `That's weird.` / `yanked around` / `I see no one here by that name` | 1 each — only deepseek-chat's 22:42:11 write |

I also listed every write containing `doYank` (three: the 22:42:11 one and two
bridge/export files in session `6d338c07` at 01:12 UTC on 04-25, which contain no
yank strings) and every write to `pkg/game/act_other.go` (five, all session
`8e4697db`, 22:42–22:47; two further `act_other.go` calls at 01:08 are `read`
calls, and the `edit`s there carry only 7-character string fragments).

So the committed revision was written between 22:47 and 23:53 UTC by something
that left no transcript: a session on a host whose agent directory kept no
`.jsonl` (the NUC's `claude-sonnet` and `gemini`), or an edit outside the agent.
This matches the PF-098 correction (`a51825788`, 2026-11-03) and is now backed by
the specific needle counts above. **`S001` therefore stays `not-found` as a
mistake attribution**; what the transcript proves is that `deepseek-chat` created
the `doYank` function, not that it wrote the shipped paraphrase.

## The not-found rows and why

**16 `no-transcripts`** (`G016`, `G019`, `G020`, `G027`, `G033`, `G055`, `G076`,
`G081`, `G097`, `S003`, `S016`, `S018`, `S058`, `S059`, `S093`, `S094`): the
birth precedes 2026-04-22, the first surviving build transcript. Recorded, not
searched.

**5 mac-mini** (`G015`, `G057`, `S015`, `S069`, `S087`): the mac-mini corpus was
extracted (5,184 files, 556 writes) and searched, but no distinctive birth line
has a write in the 48 h before the commit. `S069`'s commit (`b588c79da8`) does
have window writes — but from a *different file* in the same commit
(`pkg/session/spell_level.go`), not the death/affect strip the row is about.

**30 nuc**: the writing agent left no transcript (the `claude-sonnet`, `gemini`
and `opus-reviewer` directories hold few or no sessions, `janitor` seven), or the
code was rewritten before the commit as in `S001`.

## Retry pass: what it changed

| Row | v1 match | v2 match | Outcome |
|---|---|---|---|
| `G006` | `func castIdentify(...)` | `castIdentify(level, ch, cvict, ovict)` | stronger line, still `function-created` (a dispatch call, not the wrong line) |
| `S068` | `func (p *Player) HasLight() bool {` | `item, ok := p.Equipment.GetItemInSlot(SlotLight)` | upgraded to `mistake-line` |
| `S078` | `func castIdentify(...)` | `sendToCaster(ch, "You feel informed:\r\n")` | upgraded to `mistake-line` |
| `S083` | `func (p *Player) HasLight() bool {` | `item.Prototype.Values[1]--` | upgraded to `mistake-line` |
| `G046`, `G067`, `S028`, `S029`, `S033`, `S054` | unchanged | — | no write for the mistake's own line in the window |

## CLAUDE.md

Of the 18 matched sessions, **1 read `CLAUDE.md`** (`b46661ce`, the `S033`
match) and **17 did not**. Counted from tool calls only (a `read` tool call on
it, or a `cat`/`exec` mentioning it), across both hosts. Only 7 sessions in the
whole corpus ever open it. The `yank` session never did (PF-098), and neither
does any of the seven `mistake-line` sessions.

## Open questions

1. **The 30 nuc not-found rows are not evidence of absence.** Most are probably
   attributable to `claude-sonnet`, `gemini` or `opus-reviewer` sessions whose
   transcripts did not survive.
2. **`match_strength` is a judgement call at the boundary.** A dispatch call
   (`G006`) and a branch condition (`G013`) both sit between "the wrong line" and
   "the function"; I graded both `function-created` and named the reason. A
   second coder might grade `G013` differently.
3. **The 48 h window is a guess.** Waves were dispatched over days; widening it
   would raise both the match rate and the false-positive rate.
4. **Coverage gap, small:** the extractor originally skipped `edit` calls whose
   `edits` are bare strings rather than `{oldText,newText}` objects. There are 35
   such string edits in the NUC corpus and **none longer than 40 characters**, so
   no code was missed; `v2/extract.py` now handles both shapes.
5. **No `other_writers` anywhere.** Either these lines were written once, or the
   archives keep only one session's copy.
6. **The mac-mini window yields nothing** (5 rows), against 18 on the NUC.
