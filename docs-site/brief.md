# Dark Pawns — Frontend Design Brief

**Prepared for:** Claude Design (handoff)  
**Project:** Dark Pawns MUD (1997–2010) — modern web frontend for the Go port  
**Status:** v1 scope, pre-build  
**Domain:** darkpawns.io ($35/year, pending purchase)

---

## 1. The product, in one sentence

A web home for a resurrected ROM-derivative dark-fantasy MUD — a place where lapsed players come back, MUD enthusiasts evaluate, and curious newcomers (humans *and* AI agents) get hooked enough to log in.

## 2. What makes this project unusual

Three things to anchor the design around:

1. **The 13-year run.** This isn't a new MUD pretending to be old. Real history, real names, real player memory. Lean into archival framing — patch dates, area authors, version numbers.
2. **Faithful ROM 2.4b mechanics.** Combat formulas, skills (bash/kick/trip/backstab/headbutt), 187 ported socials, original area files. The site should feel like documentation that respects the source.
3. **AI agents as first-class players.** Same WHO list, same death, same rules as humans. This is a genuine differentiator — surface it without making the whole site about AI.

## 3. Audience priority

1. **Returning '97–'10 players** (nostalgia trigger, reactivation)
2. **Modern MUD enthusiasts** (TMC/MudConnect crowd evaluating a new server)
3. **Recruiters / portfolio viewers** (this is a serious engineering project)
4. **Friends & family** (bragging rights)
5. **MUD-curious newcomers** (need a gentle on-ramp)

The split matters. Returning players want the news feed and connection info above the fold. Newcomers need a "what is this" path. Don't pick one and abandon the other — the splash routes them.

## 4. Primary user action

**Read the lore and project history → get hooked → connect.** Not "play instantly." The web client is a try-it demo, not the main attraction. Telnet address is the real CTA for serious play.

## 5. v1 scope

| Page | Purpose | Notes |
|---|---|---|
| Splash | First-visit gate | One-time, dismissable; sets tone |
| News / devlog (home) | Returning player landing | Reverse-chron, long-form, low chrome |
| About / history | The 1997–2010 story + the port | Editorial long-read |
| Help / command reference | Player docs | Searchable, dense, terminal-flavored |
| Web client | Casual try-it telnet wrapper | Not for serious play |
| Class / race / skill DB | Browse + detail | Cards or tables; lift from area files |
| Maps & directions | Static area maps | Likely image gallery for v1 |
| Live status | Who's online, uptime | Small widget, embedded on home |
| Patch notes / changelog | Version history | Append-only stream |

**Deferred to v2:** forum, live chat, battle logs, hall of fame, item DB, quotes wall. Discussion → Discord link out for v1.

## 6. Voice and tone

**Adjectives:** nostalgic, scary, intriguing, professional.  
**Anti-adjectives:** cliche, corny, goofy.

Write like a serious archive that happens to be about a fantasy game. The dread should come from restraint, not from adjectives. Avoid in-fiction voice on UI chrome. The MUD itself is in-fiction; the site is the dust jacket and the colophon.

**Self-seriousness dial: ~35/100.** Mostly straight, with room for dry wit. The 187 socials and player memory will provide humor on their own without the chrome chasing it.

## 7. Aesthetic direction

> "Retro Stephen King paperback, but completely modern design standards."

Translation:
- **Pulp two-color print:** cream paper + one hot color (oxblood-red is our primary recommendation; cyan is the alt). Black ink for text. That's the whole palette.
- **Type does the work.** No commissioned illustration in v1. No SVG flourishes. The single ASCII logo from the game's login screen is the only "image" we have, and it's enough.
- **Modern restraint.** Generous whitespace. Strong grid. Editorial hierarchy. The pulp influence is in the *colors and typographic choices*, not in textures, not in distressing, not in faux-cracked anything.
- **Dark fantasy without theme-park dressing.** The mood is "abandoned library where something happened" — not "haunted manor." Quiet, cold, slightly off.

### Hard NOs (do not do these)

- Floating bats, ravens, spiderwebs, fog, candles
- Dripping-blood typography, glowing red eyes
- Gradient purple/black "dark fantasy" backgrounds
- Cracked stone or parchment textures behind anything
- Wax seals, scrolls, "Ye Olde" copy
- Skull bullets, flame dividers, animated torches
- Cinzel, Cormorant, Uncial, Blackletter

### Color tokens (proposed)

| Token | Hex | Use |
|---|---|---|
| `--paper` | `#EFE7D6` | Page background (warm cream) |
| `--paper-deep` | `#E5DAC1` | Cards, elevated surfaces |
| `--ink` | `#1A1614` | Primary text (warm near-black) |
| `--ink-muted` | `#56504A` | Secondary text |
| `--rule` | `#1A1614` | Hairlines, dividers |
| `--accent` | `#A8201A` | Oxblood — links, hot type, CTAs |
| `--accent-deep` | `#7A1812` | Hover/pressed |
| `--registration` | `#0F0F0F` | "Black plate" — body copy on cream |

Dark mode is **out of scope for v1.** Cream paper is the brand.

### Type system

**A. Mass-market workhorse** *(recommended)*
- Display: **Söhne Breit** or **National 2 Condensed Bold** — tight-tracked all-caps for headers and the wordmark
- Body serif: **Source Serif 4** or **Lora** — workhorse, readable, slightly literary
- Mono: **JetBrains Mono** — for the client, command refs, tags

**B. Manuscript / typewriter**
- Display: **JetBrains Mono Bold** at large sizes
- Body serif: **EB Garamond**
- Accent: **Special Elite** for stamps/marginalia *(use sparingly)*

**C. Brutalist editorial**
- Display: **Inter Tight** at heavy weights, tight-tracked
- Body: **Newsreader** or **Crimson Pro**
- Mono: **IBM Plex Mono**

### Wordmark — DARK PAWNS

All-caps, tight-tracked display sans (per your selection).

**Claude Design recommendation:**
- **Tacked monolith · dossier line below** for the splash hero
- **Serial card** for section headers
- **Negative panel** for favicon and social
- **Pure type** as the system default inline
- Variant: wordmark + tiny serial-number subtitle (`MUD · EST. 1997 · v2.0`)

## 8. Information architecture

```
/                          splash (first visit only, then redirect)
/news                      home — devlog stream
/news/:slug                individual post
/about                     history of Dark Pawns + port story
/play                      web client (casual demo)
/connect                   telnet address + how to connect
/help                      command reference index
/help/:command             single help entry
/world                     classes, races, skills landing
/world/classes
/world/races
/world/skills/:slug
/maps                      area maps
/status                    who's online, uptime, stats
/changelog                 patch notes
/credits                   original team + port credits
```

## 9. Page-level direction

### Splash
**Job:** set the tone in under three seconds, route to news.
- The original ASCII logo, centered, oxblood on cream.
- Tagline: *"A MUD from 1997. Rebuilt for now."*
- Two affordances: `[ ENTER ]` (→ /news) and connection address (`telnet darkpawns.io 8080`).
- One-time only — set a cookie, skip on return.

### News / devlog home
**Job:** prove the project is alive; route returning players to play.
- Long-form posts, reverse chron, full content visible (no card-stack).
- Sticky right rail: live status (4–7 online, uptime, version).
- No hero carousel, no featured tiles. Just dated entries.

### About / history
**Job:** make returning players feel something; explain the port to newcomers.
- Editorial long-read. Drop-cap. Wide measure.
- Three acts: 1997–2010 (the original), 2010–2024 (the gap), 2024+ (the resurrection + the agents angle).
- Pulled quotes set in oxblood serif italic.

### Web client (`/play`)
**Job:** let curious people poke around without committing.
- Big terminal, dark inside the terminal *only* (the rest of the page stays cream).
- Single input row. Command history. Connect/disconnect button. Nothing fancy.
- Sidebar: "This is a casual demo. For serious play use Mudlet/TinTin++ → telnet …"

### Item / equipment DB *(deferred but wireframed)*
**Job:** give power players a reference.
- Browse: dense table. Sortable. Filter by slot/class/level.
- Detail: spec-sheet layout with stat blocks set in mono.

## 10. Production notes

- **Static-first.** News, help, world DB should ship as MD/JSON-driven static pages. Live status and the client are the only dynamic parts.
- **No tracking, no cookie banner beyond the splash flag.** This audience hates that.
- **Accessibility:** the cream/oxblood palette must hit AA at 16px body — verified, our `--ink on --paper` is well above 7:1; oxblood-on-cream link color hits 4.5:1 at body size. Don't drop type below 16px for body.
- **Print stylesheet** for help and patch notes — the original players printed these on dot-matrix and they'll appreciate it.

## 11. Open questions (answered)

1. **Original site assets:** Wayback Machine captures available in workspace. Original dp-players.com was basic early-2000s HTML.
2. **Canonical telnet address:** `telnet darkpawns.labz0rz.com 4000` (→ `darkpawns.io` after domain purchase). WebSocket: `wss://darkpawns.labz0rz.com/ws`.
3. **Featured quotes:** `docs/lore/quotes.md` has in-game quotes. `RESEARCH-LOG.md` has resurrection-era content. The port story (69K lines C→Go in 8 days) is the strongest about-page content.
4. **Forum vs Discord:** Discord-only for v1.
5. **Logotype:** ASCII login screen art is the only brand asset from the original era. That's enough.

## 12. Success criteria

- A returning '04 player lands on the home page, sees their character class still listed, gets a lump in their throat, and copies the telnet address inside two minutes.
- A recruiter clicks /about, reads to the bottom, and walks away thinking *this person is serious*.
- Nobody, at any point, sees a floating bat.
