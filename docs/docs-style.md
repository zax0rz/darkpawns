# Dark Pawns docs style

How to write a page for the Dark Pawns docs at darkpawns.org/docs. Voice comes from [`brand-voice.md`](brand-voice.md): §5.5 (Anti-LLM Prose Standard), §8.5 (product copy) and §8.6 (docs attitude) bind every page. This file covers everything else: page types, formatting, words, callouts, steps, links, images, diagrams and freshness.

Approved by Zach, 2026-10-29. Changes go through Zach, like `brand-voice.md`.

---

## 1. Every page serves one purpose

A reader should know in five seconds whether this is their page, and find their step in ten.
- **The title and summary answer "is this my page?"**
- **Headings are a table of contents you can act on.** A guide's headings are its sub-tasks; an explainer's are its reader's questions. Starlight lists them in "On this page".
- **The first sentence of every section carries its point.** The rest is detail for whoever keeps reading.
- **Progressive disclosure.** Edge cases, internals and long examples go in a collapsed `<details>` block or a "How it works" section at the end, never in the main path.
- **One purpose per page.** If a page needs two "Before you start" lists, it's two pages.
- **Tables for comparisons, lists for options, steps for order, prose for why, diagrams for flow.**

## 2. Sections

| Section | For | Examples |
|---|---|---|
| **Play** | players | Your first hour, clients, classes, the help files |
| **Host** | operators | install, first login, the admin console page by page, updates, backup and restore, troubleshooting |
| **Build** | builders | webOLC, zones, scripts |
| **Reference** | everyone | commands, configuration, the admin and public APIs, glossary |
| **Project** | contributors and researchers | fidelity rules, contributing, research notes |
| **Releases** | everyone | one page per release, newest first |

**Play opens with "Your first hour":** what to type first (`wear all`), how to get from the temple to the newbie zones, what to do there and what kills new characters. Every direction, room and zone name is checked against the world files (`lib/world/`) and the help text, never written from memory (R4).

**No room numbers on player pages, ever.** Mortals can't see them in the game, so Play pages never show one. Give directions the way a player would: start from a landmark the player can see, then list the exits. For example: "From the temple cross, where the bulletin board is: west, north, west." Check every route against `lib/world/`. Room numbers stay in source evidence and review notes, never on the page.

## 3. Page types

Every page has front matter:

```yaml
title: Back up your world      # sentence case, under 60 characters
description: Make a backup of your Dark Pawns world, store it off the server, and check that it restores.   # 120 to 160 characters
type: guide                    # guide | explainer | reference | troubleshooting | release
audience: operators            # players | operators | builders | developers
verified: v0.1.7               # the release this page was last checked against
```

### Guide: do a task

Title starts with a verb: "Install Dark Pawns", "Back up your world".

1. **Title**
2. **Summary**, 1 to 3 sentences: what you'll have done and who this is for.
3. **Before you start**: what you need, as a short list.
4. **One aside** if it's earned (a Caution before step 1 for anything risky).
5. **Steps**, split into sub-task sections when long.
6. **Did it work?**: how the reader knows.
7. **If it didn't work**: the two or three likely failures, each with its fix. Link to Troubleshooting for the rest.
8. **Related**

### Explainer: understand something

Title is a noun phrase: "How updates work", "Immortal levels". Title → summary → sections that answer the reader's questions in order → a diagram where flow matters → "How it works" last (the only place for internals, file paths and C references) → Related.

### Reference: look something up

Title is the thing: "Commands", "Configuration settings". Title → one-line summary → a table or definition list → nothing else. Generate it from source:
- commands from the in-game help files, so the text is the game's own;
- the APIs from the Huma OpenAPI export (starlight-openapi);
- settings from the settings schema.

### Troubleshooting: fix something

Title is the symptom: "Players can't connect". One section per symptom, each with **Why it happens** (one line) and **Fix it** (steps).

### Release notes

One page per release at `/releases/v0-1-7/`, generated from `docs/releases/vX.Y.Z.md`, the same file GitHub and Discord publish. `/releases/` lists them newest first.

## 4. Formatting

- **Sentence case** for titles, headings, sidebar labels and button names in prose. Proper nouns keep their capitals: Dark Pawns, Mudlet, Kir Drax'in.
- **UI labels in bold**, exactly as on screen: click **Save**. Menu paths use an arrow: **Admin → Updates**.
- **Commands, file names, flags and code in backticks:** `darkpawns update`, `--check`, `lib/data/settings.json`. In-game commands too: type `score`.
- **Keys** in plain text with a plus: Ctrl+C.
- **Numbers:** numerals for anything measurable (3 players, 7 days, port 7777). One to nine may be spelled out in narrative Play prose.
- **Dates:** 29 October 2026 in prose, 2026-10-29 in tables. **Versions:** v0.1.7.

## 5. Words

**Banned:** simply, just (as in "just run"), easy, easily, obviously, of course, quick (unless timed), "click here", "please" in instructions, "e.g." and "i.e." (write "for example", "that is"), and "etc." closing a list (finish the list or say "and more").

**SaaS and business jargon is banned unless the page is mocking it:** solution, leverage, utilize, empower, unlock, seamless, robust, powerful, streamline, best-in-class, world-class, cutting-edge, game-changer, synergy, ecosystem, stakeholder, onboarding, user journey, use case, value, deliverable, out of the box, at scale.

**Docs exception to brand-voice §5 "words avoided":** Host and Build pages may say "admin console", "web client", "browser" and "settings". Players are still players, never users.

**One name per thing.** The glossary page is the source; each page links a term to it the first time it appears.

| Use | Not |
|---|---|
| character | toon, avatar, account (unless it really is the login) |
| immortal (level 31 and up); implementor (level 40) | god or admin, as a level name |
| operator (the person running a server) | admin, sysadmin, user |
| world (the game data and players on one server) | instance, except in CLI flag names |
| admin console | dashboard (that's one page), admin panel, backend |
| web client | web player, browser client |

**Contractions.** Write like a person talking: "don't", "you'll", "it's". It's a MUD; keep it relaxed and conversational. At most one exclamation point per page, and never in steps or warnings.

## 6. Callouts (Starlight asides)

| Type | Use for | Never for |
|---|---|---|
| **Note** | Context that helps but isn't required. The one place a joke may live on a Host page. | Anything the reader must read |
| **Tip** | A faster or better way | Required steps |
| **Caution** | Something that can lose work or confuse; read before step 1 | Jokes |
| **Danger** | Data loss, security, can't be undone | Anything else |

At most three per page, never two in a row. "Before you start" is a list, not an aside. If everything is a warning, nothing is.

## 7. Steps

- Number only things done in order; bullets for everything else.
- **One action per step**, starting with the verb: "Open **Admin → Updates**."
- Add the result on its own line when it isn't obvious: "The page shows *Up to date* or the new version."
- The condition comes before the action: "If the check fails, run `darkpawns doctor`."
- 3 to 9 steps. More than 9: split into sections.
- An image goes after the step it shows, never inside the sentence.

## 8. Links and navigation

- Link text says where it goes: "see [Back up your world]", never "click here" or a bare URL.
- **Previous and next** come from the sidebar order, so order every section as a path someone would follow.
- **Related**: 2 to 4 links at the end, only pages a reader of this page will want next. None qualify, no section.
- C source and fidelity notes are linked only from Project pages and "How it works" sections.

## 9. Images: every one makes a point

An image exists to illustrate an idea, point or step. If it doesn't, cut it.

**Use one when:**
- the reader must find something on a screen they haven't seen (a new admin page);
- a result is easier to recognize than to describe (the web client after login);
- a short terminal recording beats a paragraph (install, `doctor`).

**Never** to decorate, to repeat the text, or for every step. The page must make sense with images off.

**Focus the reader's eye:**
- **Crop to the part that matters,** with a little context around it. A full window only when position on the page is the point.
- **Highlight the target:** an outline in the accent color around the control, the rest of the screen dimmed slightly.
- **Number the highlights** to match the step numbers when one image covers several steps.
- **GIFs zoom** to where the action is (the admin console is a lot to take in at full size), then pull back if the reader needs to see where it sits.
- **Terminal recordings** use a large font, hide setup and waiting, type at a readable speed, and pause on the result.

**Rules:**
- **Every image is generated by `make media`** from the seeded demo world. No hand-taken screenshots. Highlights, crops and zooms live in the capture script, so they come back on every regeneration.
- **Alt text** says what the reader needs from the picture, under 150 characters: "The Updates page with **Check for updates** and the current version." Decorative images get empty alt text. Alt text lives in the capture script next to the shot.
- **Caption** only when the image needs context the alt text can't carry.
- **No personal data.** Demo world names only; never a real player, IP or email.
- **GIFs:** under 15 seconds, no audio, loop, and a first frame that makes sense on its own.
- **Freshness:** the capture manifest records each image's source files and release; CI warns when a source changed after its image.
- One theme (the site's), so no light/dark variants.
- **Media minimum:** every guide with more than three steps on a screen gets at least one focused image or recording, and every Host and Build page gets one unless text really is clearer.

## 10. Diagrams and data

Draw a diagram when the reader needs to see how parts connect or what happens in what order:
- **flows:** install → setup → first login; an update with automatic rollback; how the launcher and the game talk; a character's first login;
- **states:** a character's positions, a fight round;
- **data:** real numbers from the game as charts or tables, such as zone level ranges or class progression, generated from the world files.

**Rules:**
- **Write diagrams as Mermaid in the Markdown.** They render on GitHub and in Starlight, agents can read them, and they never go stale like a screenshot.
- **A one-line text summary** comes before every diagram, saying what it shows.
- **Labels in plain words**, sentence case, with no function names except in Project pages.
- **Charts come from data**, generated by a script from the world files, never typed numbers.
- Color is never the only way to tell things apart.

## 11. Delight

At most one small surprise per page: a real help-file line as an epigraph, an aside from the world, a joke in a Note. Quote the game exactly (R1); a misquoted help file is worse than none. Delight never goes in steps, warnings or anything about deleting, restoring or security.

## 12. Things every page and the site need

- **Glossary:** every acronym and MUD term (AC, THAC0, HP/MP/MV, PK, mob, tank, rent, immortal levels and more).
- **Redirects** for every old docs and wiki URL (the site's Caddy redirect table).
- **"Edit on GitHub"** and the **"Last verified"** release from front matter on every page.
- **404 page** with search; **Pagefind** search.
- **For agents:** `llms.txt`, a Markdown copy of every page, stable heading anchors, nothing that exists only inside an image.
- **Accessibility:** no skipped heading levels, alt text, color never the only signal, header rows on tables.
- **Ownership:** Host pages change with the admin console and CLI, Play pages rarely. The release checklist runs `make media` and bumps `verified:` on every page the release touched.
- **Versioning:** one set of docs for the latest release; new things are flagged inline ("New in v0.1.7").
