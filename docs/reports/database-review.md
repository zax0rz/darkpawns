# /database — End-to-End Review + Implementation Tracker

**Date:** 25 May 2026
**Source read:** `zax0rz/darkpawns@main` —
- `website/layouts/_default/database.html`
- `website/static/js/database.js`
- `website/static/data/database.json` (1.58 MB)
- `website/assets/css/style.css`
- `website/content/database/_index.md`

**Companion artifact:** `Database Redesign.html` — three hi-fi directions, desktop + mobile artboards. (External file — not in repo; ask The Architect for a copy.)

---

## Implementation Status

**Quick wins shipped (commit `0cec6a5`):**
- F-5 ✅ `--font-display` → DM Serif Display. Detail titles drop `font-weight: 800` and `text-transform: uppercase`. Tab buttons and result names adjusted.
- F-7 ✅ Editorial scaffolding — `№ 003 · Knowledge Codex` mono slug, `The codex.` h1 in DM Serif Display, italic tagline, mob/item count numerals populate from data on load.

**Decisions made:**
- Direction B (Bestiary) — **dropped.** Not viable without per-entry prose content sprint.
- Direction A (The Index) — **confirmed as build target.**
- Map ↔ Database integration — **deferred to its own phase** (needs design discussion).
- HP display — **toggle between average HP and raw dice notation.**

**Linear issues (DP team, need project assignment):**

| Issue | Title | Phase | Priority | Effort | Status |
|-------|-------|-------|----------|--------|--------|
| DP-328 | F-1: Mobile bottom-sheet detail + filter sheet | 1 | Urgent | ~4h | Todo |
| DP-323 | F-4: Skyline overview + skeleton loading | 2 | High | ~2h | Backlog |
| DP-327 | F-3: Deeper filter surface — zone, race, flags, sort | 2 | High | ~2h | Backlog |
| DP-329 | F-2: Zone gutters, tier rules, flag glyphs in list | 2 | High | ~1h | Backlog |
| DP-324 | F-6: Richer drops/carriers rows — type, affects chips | 2 | High | ~1h | Backlog |
| DP-330 | F-8: Mobile polish — type floor, hover guards, scroll shadows | 3 | Medium | ~30m | Backlog |
| DP-326 | F-9: Hash routing → mobile sheet auto-open + sort state in URL | 3 | Medium | ~1h | Backlog |
| DP-325 | P-1: Performance — split JSON, virtualize list, service worker | 4 | Medium | ~4h | Backlog |
| DP-331 | P-2: Map ↔ Database integration (TBD) | 5 | Low | TBD | Backlog |

**Not yet scoped:**
- HP/dice toggle — small feature, can fold into Phase 2 or standalone issue
- `make deploy-site` — pending (quick wins not yet live)

---

> **Quick read.** The page is genuinely good. The structural decisions are right — master-detail at desktop, hash-deep-linkable entries (`#mob-3001`, `#item-1205`), real cross-references between mobs ↔ items ↔ rooms ↔ shopkeepers, a distinctive Power Index visualizer. The complaints I'd flag, in order of impact: the mobile breakpoint flips to a 40/60 split that crams every control AND a 150-row list into 40% of the viewport (genuinely unusable on a phone); the entry list is undifferentiated visually so 3,000 rows read identically; the filter surface is too thin (no zone, no flag, no race); and the page has no overview — no skyline, no count, no "what's in here?" — so the user lands and has to start searching cold. None of these need a rewrite of the page; they're four surgical moves on a v1 that's already 70% there.

> v1 of this review was written before I could read the source. This version corrects three wrong assumptions: there is no Spells tab (just Mobs and Items); cross-references already exist (drops link to items, items link to mobs, both link to rooms via `/map?room=N`); and player level cap is 40, not 60.

---

## 1 · What the live /database gets right

Don't lose any of this in the rewrite.

- **Hash-based deep linking** (`database.js:43-65`). Every entry is `#mob-N` or `#item-N`, parsed on `hashchange`, auto-switches tab, auto-scrolls list into view, auto-renders detail. Reload-safe, share-safe, link-from-Discord-safe. **Excellent.**
- **Real cross-references baked in.** Mob detail links every drop to `#item-{vnum}` (`database.js:243`); item detail links every carrier to `#mob-{vnum}` (`database.js:307`); both link rooms to `/map?room=N`. Shopkeepers list their inventory with prices (`database.js:330-342`); items list every merchant who sells them with the asking price. **This is the page's hidden superpower** and the current design under-sells it.
- **The Power Index visualizer** (`database.js:191-216`). Four bars — base level / avg HP / armor rating / avg attack — scaled into a single visual. Distinctive, on-brand, lifts the detail card above "spreadsheet." Keep it. Steal it for /items too (e.g. for weapons: damage curve vs weight vs cost).
- **Search across name + keywords + vnum** (`database.js:101`). `r.k` is the original area-file keyword list; including it means typing `bard` finds "a cat-eyed bard" via the `bard musician` keywords even if the visible name didn't match. **Lift this idea into a future zone filter.**
- **Dice average computed for display.** `calcDiceAvg("3d5+10") → 19` is the right move; players don't think in raw dice notation, they think in averages. Don't lose this.
- **Hash → tab switching round-trips through the same routing layer** (`switchTab` in `database.js:73-91`). Means clicking "a halberd" from inside the Vault Keeper's drops correctly switches the tab AND selects the item AND keeps the URL clean. Already works.
- **Color tokens + paper aesthetic** carried over cleanly. `--paper / --paper-deep / --ink / --accent` are right.
- **Extra descriptions for items** (`o.edesc`, rendered at `database.js:368-378`). Real ROM-era touch — the keyword-gated `look` text. Great that it's surfaced.
- **The 320px sidebar at desktop width.** Fixed width, no resizing — the right call. Index always in view, master-detail metaphor intact.

---

## 2 · The seven things to fix

Ordered by impact-per-hour.

### F-1 — The mobile breakpoint is the single biggest defect on the site (4 hours · highest impact)

```css
@media (max-width: 900px) {
  #db-page { flex-direction: column; }
  #db-sidebar { height: 40%; }
  #db-main { height: 60%; }
}
```
— `database.html:299-310`

At ≤900px (so every phone, plus iPads in portrait), the sidebar becomes 40% of `100vh - var(--nav-height)`. On a 390×844 viewport, that's ~316px of vertical space — into which the layout crams: the **two tabs**, the **search input**, the **filter group** (slider + alignment select for mobs), AND the **scrollable result list**. The result list typically gets ~120px — enough to display **two and a half rows** at a time. Scrolling that micro-region with a phone thumb is borderline impossible because every flick lands on a filter control instead.

Then on tap, the detail card renders in the bottom 60% — and the user can no longer browse the list. They have to scroll back into a 120px-tall region to find the next entry.

This is the central UX failure.

**Fix.** Replace the 40/60 split with two distinct mobile patterns:

1. **List mode (default).** Full viewport: sticky tabs on top, sticky search below, the filter sheet hidden behind a `[ FILTERS · 3 ]` button (full-screen sheet when opened), then the list filling the rest of the viewport. Active filters appear as removable chips between search and list.
2. **Detail mode (after tap).** Detail card slides up as an 85vh **bottom sheet**, the list staying mounted underneath. Drag handle at top, `[ ✕ CLOSE ]` mono button top-right. Closing returns to *exactly the same scroll position in the list* — `result-item.active` already handles selection persistence.

The bottom-sheet pattern is what every native iOS/Android database app uses for exactly this reason. It's also the cleanest fit for the hash routing — `#mob-3001` could specifically mean "sheet open on 3001," and clearing the hash (or back gesture) closes the sheet.

See `Database Redesign.html` → Direction A · Mobile.

### F-2 — The list looks identical at row 4 as at row 2,847 (1 hour)

The result row template (`database.js:140-152`) produces a flat list of `name + level/sex + alignment + vnum`. Three thousand rows of that read as undifferentiated soup. The eye has nothing to lock onto.

**Fix.** Three structural anchors in the list, all CSS-only:

1. **Group by zone**, not by VNUM. The data already has spawn locations (`m.spw[].zone`); pick the primary spawn zone for grouping. Insert a sticky `// ZONE — Midgaard · 214 mobs` mono caption between groups. The list now has structure.
2. **Tier rules between level bands** (1–10 / 11–20 / 21–30 / 31–40). Hairlines + tiny mono label. Free editorial scaffolding.
3. **Flag glyphs as a 16px gutter** at the start of each row. The area-file flags exist (`m.k` keywords often encode them; `m.alg` distinguishes aggressive temperament implicitly). Render `◆` for unique mobs (no duplicates in spawn data), `‼` for low-alignment aggro candidates (`m.alg <= -700`), `§` for shopkeepers (`m.shop` is present). Three glyphs, oxblood mono. The list reads as scored music.

The current sort-by-VNUM is also wrong as default. **Sort by name within zone group**; offer a `SORT BY: VNUM / NAME / LEVEL` dropdown. Returning '04 players know VNUMs; newcomers don't.

### F-3 — The filter surface is too thin (2 hours)

For mobs you currently get: **min level slider (0–40)** + **alignment (good/neutral/evil/all)**. That's two controls for 1,319 entries. Items get **type** + **wear position**, again two.

Real filters that would unlock the data:

**Mobs:**
- **Zone** (multi-select) — the single most useful filter. Players think in zones.
- **Race** (`m.rc`) — `Human / Orc / Drow / Undead / …`
- **Flags** — shopkeeper / quest / unique / aggro (derived). Boolean chips.
- **Level band quick-chips** (`1–10 / 11–20 / 21–30 / 31–40 / boss`) in addition to the slider. Chips for the 90%, slider for the 10%.
- **Sort** — VNUM / Name / Level / Race.

**Items:**
- **Magical affect** (filter by `o.aff[].location` — `+hitroll`, `+damroll`, `+stat`, `+mana`, etc). This is the only filter equipment-hunters care about.
- **Min cost** — separates trash from real loot.
- **Has script** (boolean) — flags the interesting ones.
- **Zone** (from `o.rms[].zone`) — same as mobs.

This is a bigger ask than the others on the list but pays off forever. Filters are how power users get lost in data.

### F-4 — There is no overview (2 hours)

Today, after the 1.58 MB JSON downloads, the user sees: an empty `db-empty` panel that says "← Select a database record to inspect" — and… nothing else. No counts. No distribution. No shape. No invitation.

The page is a search box on a corpus you have no map for. New users bounce; returning players have to remember what to type.

**Fix.** A **skyline strip** below the masthead, before the table — visible on first paint *while the JSON is still loading*, then populated from the data:

1. **Level histogram** (40 bars or 8 bands of 5). Clickable: click a bar → applies a level filter.
2. **Zones by density** — top 6 zones as horizontal bars, each clickable.
3. **Flag breakdown** — `◆ Unique 47 · § Shop 31 · ‼ Aggro 312`. Each clickable.
4. **By type** — for items only: `Weapon / Armor / Container / Light / Key / Other` with counts.

Three small charts, no D3, no SVG illustration, just type and bars. Solves "what is in this thing" in 180px of vertical space. Doubles as the loading state — drop the current 1.6s progress sweep entirely; instead, render the skyline shell with skeleton bars, replace with real bars when the JSON arrives. *The page never appears empty.*

### F-5 — `--font-display` still points to Archivo Narrow on the database page (10 minutes)

```css
--font-display: 'Archivo Narrow', 'Arial Narrow', sans-serif;
```
— `style.css:14`

The `.detail-short` (the entry title in oxblood) is `font-family: var(--font-display)` — so every entry name on the database is set in Archivo Narrow, with `font-weight: 800; text-transform: uppercase; letter-spacing: -0.02em`. The prior audit (`Live Site Audit.md`) flagged this as the single change that would move the whole site 60% toward the spec; the splash got DM Serif Display, the masthead got DM Serif Display, but the rest of the body — including this page's titles — still reads as Archivo Narrow.

**Fix.** Swap `--font-display` to `'DM Serif Display'`, drop `text-transform: uppercase` from `.detail-short`, drop `font-weight: 800` (DM Serif Display ships at 400 only), and the database immediately reads as the same brand as the splash and home. This is a one-line CSS change.

(If there's a reason this hasn't shipped yet — DM Serif Display might be too display-y for some sub-headers — keep Archivo Narrow for `<h3>` section labels but force DM Serif Display on `.detail-short` specifically. Either way, the heading shouldn't be condensed sans.)

### F-6 — The "Equipment & Drops" list is the most under-leveraged surface (1 hour)

The drops list (`database.js:233-243`) is currently:

```html
<li class="rel-item">
  <a class="rel-link" href="#item-{vnum}">{name}</a>
  <span class="rel-meta">{slot}</span>
</li>
```

A flat list of items by name + slot. But this is exactly the "follow the trail" hook the user needs to get lost. With the data you already have, every drop row should show:

- the **item name** as a link (✓ done)
- the **item type icon or label** (Weapon / Armor / Container)
- if it has **magical affects**, a tiny `+stat / +hitroll / +dam` chip in oxblood
- a small **rarity hint** if you can derive one (load %, or vnum band)

Same for the items page: the "Spawned / Carried By Mobs" list should preview each mob's level + alignment as a one-line glyph + number. The reader should be able to skim a single screen and know which links are worth following.

This is small CSS + JS work — the data is already in `record.aff`, `record.type`, `record.load`.

### F-7 — The page is missing all the site's editorial scaffolding (30 min)

The home and about pages use `1994 / 2010 / 2026` giant numerals, `№ 042 · ENTRY` mono slugs, dot-leader dividers, oxblood section labels. The database page has none of it — it dumps straight into the app, no masthead, no slug, no count.

**Fix.** Adopt the site conventions on the page header:

- `№ 003 · KNOWLEDGE CODEX` mono slug above the masthead.
- A real `<h1>` set in DM Serif Display: "*The codex.*" (or "*Mobs & items.*"). Currently there's no `<h1>` on the page at all — the only `<h1>` is `.detail-short`, which changes per entry and is wrong as the page heading (one `<h1>` per page, see also the home-page audit).
- Big DM Serif Display numerals next to the tabs: `1,319` mobs / `1,674` items.
- The detail card should slug-stamp `№ {vnum} · {n} of 1,319` at the top — gives the reader a sense of position.

---

## 3 · Mobile-specific audit (the user's stated pain)

In addition to F-1, F-3:

- **Tap targets.** `.db-tab-btn` is 0.5rem padding ≈ 32px tall. `.result-item` is 0.6rem 1rem ≈ ~50px (OK). `.filter-pill`-style chips don't exist yet but should be ≥44px on mobile.
- **Type floor.** Result name is 0.8rem (~12.8px), meta is 0.65rem (~10.4px). 12.8px is **below** the recommended mobile minimum (16px body). The list reads as fine print.
- **The 1.6s loading sweep** is fine on desktop but on a 3G phone the 1.58 MB JSON fetch can be ~5–10s. Render skeleton skyline immediately + skeleton list rows; replace as data lands. Never show a generic spinner.
- **No hover guards.** `.result-item:hover` styles will fire on tap and stay sticky until next interaction on touch devices. Wrap hover styles in `@media (hover: hover) { … }`.
- **Sidebar scrolling region is invisible.** On mobile, the only visible scrollbars are the tiny webkit ones. Add a subtle scroll-shadow at top/bottom of the result list so users know there's more.
- **Filter sheet pattern.** As proposed in F-1, collapse all filters behind one button. Don't try to fit them inline at this width.
- **Hash routing on mobile** — when the URL is `#mob-3001`, the page should *open the sheet automatically* on load. Currently it does select the row, but the row is invisible inside a 120px-tall region. Tying hash → sheet-open closes the loop.

---

## 4 · Three redesign directions

See `Database Redesign.html`.

### Direction A — "The Index" (refined master-detail)
The conservative refinement. Same DNA, fixes F-1 through F-7. Sticky filter rail with real filter depth (zone, race, flags), zone gutters + flag marginalia in the list, skyline strip at the top, mobile bottom-sheet detail, hash → sheet-open. **Lowest risk; ships in three days.**

### Direction B — "The Bestiary" (editorial spread)
Reframes the database as a *printed bestiary you flip through*. Two-page spread on desktop: left page is the entry written like a guidebook plate (drop-cap description, marginalia, footnotes), right page is the spec sheet + Power Index + drops + footnotes. Edge-tabbed alphabet index on the outer margin. Mobile = single page, swipe to flip. **Highest "get lost" potential; demands real per-entry writing or it falls apart.** Most of the mobs already have a `m.l` (long description) — the bestiary leans on those.

### Direction C — "The Almanac" (data-density)
A statistical handbook. Skyline of four charts at top (level histogram, zone density bar, flag breakdown, type pie). Below, a **three-column compressed entry list** (more like newspaper agate than a table) with click-to-expand-inline. The whole page reads as one continuous spread. Mobile = chart strip horizontal-scrolls; entries become a single dense column. **Best for the returning '04 player who knows VNUMs by heart; most data-dense.**

**My recommendation:** ship A. The user has clearly invested in the right architecture (hash routing, cross-references, Power Index); A is the minimal set of moves that finishes the job. B is a moonshot for v3 — beautiful but expensive to populate. C is the right answer if you have analytics showing your audience is mostly returning players, not newcomers.

> Caveat in the mocks: I drew them before I read the source, so they show a "Spells" tab and use levels up to 60. The live page has no Spells tab and caps at 40. Treat the mocks as direction, not pixel-spec.

---

## 5 · Priority queue

| # | Effort | Impact | What |
|---|---|---|---|
| 1 | 4h | **Critical** | F-1: Mobile bottom-sheet pattern + filter-sheet button |
| 2 | 10m | High | F-5: `--font-display` → DM Serif Display (1 CSS line) |
| 3 | 30m | High | F-7: Editorial scaffolding (`<h1>`, slug, count numerals) |
| 4 | 2h | High | F-4: Skyline charts (level / zone / flag / type) + skeleton on load |
| 5 | 2h | High | F-3: Filter depth — zone, race, flag chips, sort dropdown |
| 6 | 1h | High | F-2: Zone gutters, tier rules, flag glyphs in list |
| 7 | 1h | High | F-6: Richer drops/carriers rows (type, affects chips) |
| 8 | 30m | Med | Mobile polish: 44px targets, 16px type floor, hover guards, scroll shadows |
| 9 | 1h | Med | Hash routing → mobile sheet auto-open; sort/group state in URL hash |
| 10 | 4h | Stretch | Direction B: Bestiary spread (v3 candidate) |

**~12 hours** ships everything outside the stretch. The first two hours alone (F-5 + F-7 + start of F-1) get you 60% of the perceived quality lift.

---

## 6 · Data shape — useful surface area I noticed

For when the dev sits down to build this:

- **`m.spw[]`** has zone + room name + room number per mob. Use the **first** entry as the primary zone for grouping. Pre-compute a `m.primary_zone` field at Hugo build time to avoid scanning every reload.
- **`m.drp[]`** has `obj_vnum + name + slot`. Slot is wear-position; group by slot in the detail card (`held / worn / inventory`).
- **`m.shop`** is the differentiator. Add a `§` glyph in the list and a strong "**SHOPKEEPER**" badge in the detail header. Players hunt these.
- **`o.aff[]`** is the magical affect list. Items that have one are "magical"; items without are mundane. This is probably the single most-useful boolean filter for items.
- **`o.script`** flags interactive items. Tag them visually.
- **`o.rms[]`** has both rooms and containers (`container_vnum`). Already rendered as "Inside {name}" links — nice. Preserve.
- **`o.shp[]`** is sold-by. Showing the **lowest price across merchants** as a one-liner ("From 250gp · 3 merchants") would replace the current full list as the default, expandable on click.

---

## 7 · Things I'd ask before starting

Three small product decisions worth confirming with you before I sit down to mock more:

1. **Should the database have rooms as a third tab?** The data is there (every `m.spw` and `o.rms` references rooms with names). A `/database/rooms` tab gives players the entire world as a browsable index. But it overlaps with `/map`. Worth the conversation.
2. **Is "average HP" (computed from dice) more useful than raw dice notation?** I'd argue yes for newcomers, but '04 players might want both. A toggle is cheap.
3. **Do you want the bestiary direction (B) seriously prototyped, or is it understood as a v3 aspiration?** It's a real commitment because each entry's prose has to be written or the page reads as Mad Libs. The bones of it (left-page plate + right-page spec) are already drawn; turning it on means a content sprint.

---

## 8 · Notes on data scale + perf

- The 1.58 MB JSON is fetched on every visit. Caddy probably gzip+brotli it down to ~250 KB. Still: split it. `database-mobs.json` + `database-items.json`, lazy-loaded per tab. The user who never clicks "Items" never downloads 800KB of items.
- The list is capped at 150 results, with a "Refine search to see more" hint. **Replace with windowed/virtualized rendering** — `IntersectionObserver` for incremental loading of the rest. The "refine search" hint is a hostile dead-end if the user actually wants to browse the corpus.
- Add `prefers-reduced-motion` guards on the progress sweep (currently animates regardless — `database.html:271-289`). Trivially fixed by `style.css`'s existing `@media (prefers-reduced-motion: reduce) { animation-duration: 0.01ms !important; }` but worth verifying.
- Consider a service worker for `/data/database.json` so subsequent visits are instant. The data only changes when the area files do.
