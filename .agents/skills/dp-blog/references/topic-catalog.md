# Topic catalog

Established lanes for the Dark Pawns blog. The catalog provides boundaries and
seed ideas, not a publication queue. A post should serve one lane and one
reader; combining lanes (e.g. "what is a MUD" + "build process") usually means
two posts.

## Onboarding lane (reader: never played a MUD)

- What is a MUD, explained without nostalgia goggles
- How to actually play tonight: telnet, clients, the in-browser client
- Your first hour: rolling a character, the mud school, first death
- MUD vocabulary survival guide (mob, rent, remort, tick, afk)

## History lane (reader: returning player or curious outsider)

- Dark Pawns history: the eras, the server, the community (needs Zach's
  memories and archive material; see `website-astro/src/content/archive/`)
- CircleMUD and the DikuMUD lineage: where this code comes from
- Frontline's original website as a time capsule
- Whatever happened to MUDs: the current state of text games

## Build lane (reader: developer)

- Devlog entries on the port: what bit, what broke, what clicked
- The fidelity/oracle process: porting C to Go byte-for-byte, and why
  (`docs/fidelity/RULEBOOK.md` is the source; the post is the story)
- AI-assisted development and design: what worked, what lied, what the
  harness actually bought
- Rebuilding the website: Astro, the generated map/database, the archive
  policy

## World lane (reader: player or lore-curious)

- Zone spotlights: one zone per post, rooms and mobs pulled from the actual
  world files
- Class guides: how each class actually plays (mechanics from the oracle,
  voice from the help files)
- Item lore: notable equipment, where it drops, why anyone cares
- Systems explained: dreams, remort, the Wyldlands, player killing

## Guides lane (reader: active player)

- Getting Started adjacent content for the site's world handbook
- Progression guides, zone-by-level recommendations
- The economy: gold, rent, and why dying is expensive

## Technical articles lane (reader: developer/contributor)

Long-form contributor pieces under the site's docs collection. Not started
yet; standing it up requires a `section` enum addition in
`website-astro/src/content.config.ts`.

- How to apply the oracle: you showed up with a Claude Code subscription, now
  what? Running scenarios, reading reds, earning rulebook citations
- The fidelity law for newcomers: why "equivalent" is a bug
- Anatomy of an oracle red: walking through one real divergence end to end
- How to write a scenario file: replayable ghosts of play sessions

## Notes

- The current state of MUDs post wants real research (other live MUDs,
  communities), not vibes. Flag it for a research pass when chosen.
- Zone and class posts must quote the real files under `pkg/parser/` fixtures
  or the oracle; invented room text violates the fidelity law even in prose.
- AI-assisted development posts can pull real transcript moments, but check
  with Zach before quoting anything a model said verbatim.
