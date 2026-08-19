---
name: dp-blog
description: Co-author Dark Pawns blog posts in the site's brand voice, from topic selection through drafting, voice gates, and a social blurb. Use whenever the user wants to write, draft, revise, or publish a blog post for darkpawns.com, mentions "blog post", "write about" a game/tech topic, or wants a tweet-style blurb for a new post.
---

# Dark Pawns blog co-authoring

This skill walks one blog post from idea to publishable draft plus a social
blurb. Zach is the editor and the approval gate at every checkpoint. The model
drafts, critiques, and checks; it never approves.

The pattern is adapted from Anthropic's document co-authoring skill: gather
context before writing, get outline approval before prose, draft section by
section, then test with a fresh reader. Keep it conversational. If Zach says
"just write it", compress the checkpoints into one pass, but never skip the
mechanical gates.

## Ground rules

- **Voice law:** read `docs/brand-voice.md` (v2.1) before drafting. Blog posts
  are Layer 3 (The Mythic Admin), written as Zach in Frontline's register:
  warm, concrete, willing to pair lore with mechanics. The Anti-LLM Prose
  Standard (Section 5.5) is binding: no em or en dashes, no synthetic
  importance, no trailer rhythm, concrete nouns in every paragraph.
- **Fidelity law:** if the post describes game behavior, the C source under
  `src/` and `darkpawns-c-oracle/` is ground truth (R1, R4, R5e). Quote real
  help text, room descriptions, and numbers from the repo. Never describe a
  mechanic from memory or plausible inference; verify it or cut it.
- **Source or silence:** no date, number, or historical claim ships without a
  source. Repository data, the archive under `website-astro/src/content/`,
  preserved Wayback captures, and Zach's own account are sources. Model
  confidence is not.
- **People:** the original community was real people. Write about them the way
  you would want a stranger writing about you. Nameddeveloper in-jokes (BFT,
  PAW, the socials file) stay out of public blog copy per the brand voice
  guide; historical anecdotes need Zach's explicit sign-off.

## The workflow

### Stage 1: Frame the post

Establish three things before any prose:

1. **Topic and thesis.** Check `references/topic-catalog.md` for established
   lanes; a post outside them is fine but name which reader it serves.
2. **Reader.** Usually one of: someone who has never played a MUD; a returning
   Dark Pawns player; a developer interested in the port or the AI angle.
3. **The one thing the post delivers.** A post that is "about MUDs" is not
   ready; a post that answers "how do I actually play this thing tonight" is.

Confirm the frame with Zach in two or three sentences. Stop for his reply
unless he has already given the frame.

### Stage 2: Gather evidence

Collect before drafting:

- exact quotes from help files, room descriptions, or the original website
  (note file paths; preserve spelling and typos in verbatim material)
- numbers with their source (line counts from the repo, dates from the archive
  or Wayback, dates Zach states firsthand)
- links the post will use, and where they go

Where sources disagree, surface the disagreement instead of smoothing it over.
If a fact cannot be sourced, mark it `[unsourced: ask Zach or cut]` in the
draft so it cannot silently survive to publication.

### Stage 3: Outline approval

Propose a short outline: working title, the reader's question each section
answers, and where the post ends. Every section earns its place; no sections
exists to hit a length. Zach approves or redirects before prose begins.

### Stage 4: Co-author section by section

Draft one or two sections at a time. At each step:

- Ask a focused question where Zach's judgment matters (a memory, a name, a
  joke that only lands if it's true). Don't batch ten questions.
- Offer alternatives when the voice could go two ways (dry aside vs. straight
  fact, mythic open vs. terminal-grime open).
- Preserve accepted sections. Revise surgically; do not rewrite approved text
  unless asked.

Voice reminders for drafting: "magick" with a k; "players" and "mortals", not
"users"; second person; consequences are punchlines and the punchline is "the
game killed you"; parentheticals carry the personality; a missing comma is
fine, a corporate sentence is not.

### Stage 5: Fresh-reader pass

Read the finished draft as someone who has never seen Dark Pawns. Check:

- every paragraph contains a concrete Dark Pawns noun (if the paragraph
  survives swapping in another game's name, cut or rewrite it)
- jargon is translated on first use ("mob", "rent", "remort")
- nothing assumes context only Zach has
- nothing threatens the reader; the world is hostile, the developers are not

If a subagent is available, give it only the draft and the question "what is
this post about and would you play this game?", not the intended answer.

### Stage 6: Mechanical gates

Run these before presenting the post as publishable. All are plain shell
commands; no harness-specific tooling required:

```bash
make voice-lint      # hard-fails on dashes, launch copy, missing provenance
make site-check      # voice-lint + tests + content inventory
cd website-astro && npx astro build   # frontmatter schema enforced at build
```

Frontmatter contract for `website-astro/src/content/blog/<slug>.md` (enforced
by `src/content.config.ts`): `title`, `date`, `description`, `draft`,
`textKind` (original posts use `original`), `source`, `voiceLayer`
(`mythic-admin` for new posts). New posts start as `draft: true`; flipping it
to false is Zach's call.

### Stage 7: Social blurb

Every post ships with a blurb Zach can paste to Twitter: one to three
sentences plus the post URL, under 280 characters without the link. Layer 3
compressed: gallows humor, concrete detail, no hashtags, no dashes, no
synthetic-importance words. Offer two variants with different angles (e.g.
lore hook vs. terminal-grime hook) and character counts.

### Technical articles mode

The same workflow produces contributor-facing technical articles under
`website-astro/src/content/docs/` (check `src/content.config.ts` for the
current `section`/`audience` enums; adding a new section is a schema change).
Substitutions when the deliverable is an article rather than a blog post:

- **Voice:** Layer 1 (The Engine), per the brand voice table. Lead with what
  the thing is, show real commands and file paths, personality second. Dry
  asides are allowed; lore is not.
- **Code is the spine.** An article the reader cannot follow along with in a
  terminal is a lecture. Every procedure should be runnable in order from a
  fresh clone, and verified by actually running it during Stage 5.
- **Stage 3 becomes a task analysis:** what will the reader have accomplished
  when they finish, and what does each step assume they already know?
- **Stage 7 is skipped** (no social blurb). Instead, end with where to ask
  questions and what to do after finishing the article.
- **Gates:** voice-lint scans all of `src/content/`, docs included; the dash
  ban and launch-copy rules apply to articles too. Always run the Astro build
  for the frontmatter schema (`section`, `audience`, `order`, `sourcePath`,
  `updated`).

The canonical first article candidate: "How to apply the oracle" for a
developer who shows up with an AI subscription and no context: what
`dp-oracle-diff` is, how to run a scenario, what a red means, how a fix
earns a rulebook citation, and what to do with a green you do not trust.

### Publishing

Deployment is `make deploy-site` only, with explicit authorization, per the
repository root instructions. Commit and push only when Zach asks. The skill
prepares; Zach publishes.
