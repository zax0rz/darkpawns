---
name: dp-docs
description: Write or revise a Dark Pawns docs page (darkpawns.org/docs, Starlight) in the house style, from choosing the page type through verified facts, purposeful media and the docs gates. Use whenever the user or a brief asks for a docs page, a guide, an explainer, a reference or troubleshooting page, release-notes page, glossary entry, README section, or updated screenshots and diagrams for the docs.
---

# Dark Pawns docs pages

This skill takes one docs page from "we need a page about X" to a page that passes the gates. Zach approves the rules; the model drafts, verifies and checks, and never approves its own page.

## Ground rules

- **Read first:** `docs/docs-style.md` (the whole thing) and `docs/brand-voice.md` §5.5, §8.5 and §8.6. They are the law; this skill is the workflow.
- **Source or silence.** Every fact on the page is checked against something real: the code, the C source under `src/` for game behavior (R1, R4, R5e), the world files under `lib/world/` for rooms, zones and directions, the in-game help text, or a running local server. Never write a command, setting, menu path, number or direction from memory.
- **Game behavior is the C game's.** A Play page describes what the original does, quoted where possible. Help-file text is quoted exactly (R1).
- **No hand-made images.** Media comes from `make media`. If a page needs a capture that doesn't exist, add it to the capture scripts (with its crop, highlight and alt text) rather than taking a screenshot.
- **Governing docs** (`RULEBOOK.md`, `north-star-1to1.md`, `AGENTS.md`, `brand-voice.md`, `docs-style.md`) are Zach's. Propose changes; never edit them in a docs task.

## Workflow

### 1. Pin the purpose

Write down, before drafting:
- **who** reads it (player, operator, builder, developer);
- **what they're trying to do or know** when they arrive;
- **the page type**: guide, explainer, reference, troubleshooting or release (`docs-style.md` §3).

If the purpose has two answers, it's two pages. Check the sidebar for an existing page first; extend it instead of writing a near-duplicate.

### 2. Gather the facts

Collect every fact the page will state, each with its source (file and line, command output, or help topic). Run the commands against a local server where the page tells the reader to run them. Keep the list; the review step checks it.

### 3. Draft from the template

Fill the template for the page type, in order. Front matter first (`title`, `description`, `type`, `audience`, `verified`). Then:
- **summary** that answers "is this my page?";
- **Before you start** list (guides);
- **steps**: one action each, verb first, result on its own line where it isn't obvious;
- **Check it worked** and **If it went wrong** (guides);
- **How it works** last, for internals;
- **Related**: 2 to 4 links, or none.

Apply the attitude for the section (`brand-voice.md` §8.6). Humor only where §8.6 allows it.

### 4. Plan the media

For each place a picture would help, ask: what point does it make? If there's no answer, skip it. Otherwise specify, in the capture script:
- the screen and state (demo world, logged-in level, page);
- the crop (element and padding), the highlight, and step-number badges;
- for GIFs, the zoom target and length (under 15 seconds);
- the alt text.

For flows, states and data, write a Mermaid diagram or a generated chart instead (`docs-style.md` §10), with a one-line summary before it.

### 5. Run the gates

All must pass:
- `voice-lint` clean on the page;
- no banned words or business jargon (`docs-style.md` §5);
- no em or en dashes;
- front matter complete, `title` under 60 characters, `description` 120 to 160;
- every link resolves; link text says where it goes;
- at most three asides, never two in a row;
- every image comes from `make media` and has alt text; no image is the only place a fact appears;
- the Starlight build and the Markdown copy build;
- every fact from step 2 still matches its source.

### 6. Fresh-reader test

Give the finished page, with no other context, to a sub-agent and ask:
1. What is this page for, and who is it for?
2. Where did you get lost or have to reread?
3. What would you search for to find this page?

Fix what it trips on. Report its answers with the page.

### 7. Hand over

Report to Zach: the page, its sources list, the media added, the gate results and the fresh-reader answers. Docs changes go to a PR (a single small copy fix may go straight to main under the small-site-fix rule). Deploys need Zach's go and the rsync dry-run.
