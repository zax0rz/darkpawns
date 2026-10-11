# 2026-10-11: Three living descendants of one C codebase

## Scope

- **Tracks:** legacy-port fidelity (context and related work)
- **Window:** observed 2026-10-11
- **Primary artifacts:** the MUDVerse listing for Dark Knights Remorted (https://www.mudverse.com/game/639, read 2026-10-11); `website/data/history.toml:79-80` and `website-astro/PRODUCT.md:53` for darkpawns.net; this repository's `LICENSE`, section 2, for the Frontline source
- **Status:** observation note. The facts about Dark Knights Remorted come only from its public listing and have not been checked against its server or source.

## What was observed

The Dark Pawns 2.2 C source that R.E. Paret (Frontline) published at `github.com/rparet/darkpawns` now has at least three running descendants, each taking a different approach to an old game:

| Descendant | Approach | Source for the claim |
|---|---|---|
| darkpawns.net ("DPReturns") | Former players running the game again | `website/data/history.toml:79-80` (live; revival year unverified, ~2019) |
| Dark Knights Remorted | **Continued development** of the C code: a self-described "fork of the classic Dark Pawns" with "small, targeted tweaks", quality-of-life changes, bug fixes, games of chance in Tyr's Pub, and a new **Necromancer** class. It allows several characters per player ("crew", with "mults"). Online since 2026-03-02, DikuMUD codebase, port 4350. It says it "is not intended to replace the original Dark Pawns, which remains active at darkpawns.net". | MUDVerse listing, 2026-10-11 |
| This project | **Faithful translation:** a 1:1 Go port whose rule is that the game's player-facing bytes, command surface and random draws must match the C (`docs/fidelity/RULEBOOK.md`, R1 to R4) | this repository |

The game was announced publicly in the darkpawns.net Discord on 2026-10-11, about seven months after its listed online date (reported by Zach, who saw the announcement; the Discord message itself is not archived here). The old community's gathering place is still where a new descendant gets announced.

Zach created a character on Dark Knights Remorted on 2026-10-11 and left a congratulatory note on its bulletin board.

## Why it matters for the fidelity paper

The three descendants form a natural comparison for the related-work section: **revival** (run the original), **evolution** (extend the original) and **translation** (re-implement it with behaviour held fixed). Evolution and translation pull in opposite directions on the same artifact:
- Dark Knights Remorted treats the C source as a starting point to improve. A new class or a bug fix is a feature there.
- This project treats it as a specification. A bug fix that changes player-facing bytes is a fidelity violation unless approved (R4; the `divergent-approved` status exists for exactly those exceptions).

That contrast sharpens what "faithful" has to mean. Another team, starting from the same source in the same year, chose to change it. So fidelity is a deliberate preservation choice, not the default outcome of porting.

## Licensing note

Every descendant inherits the CircleMUD and DikuMUD terms (non-commercial, credit kept, licenses shipped), and every one faces the same gap this project's `LICENSE` section 2 records: Frontline's repository carries no license file for the original Dark Pawns content. The planned contact with Frontline at launch (`darkpawns-frontline-launch-contact`) could settle that for all three.

## Open questions

- Is Dark Knights Remorted's source public? If so, its diff against Frontline's tree would be a labelled corpus of intentional changes: the mirror image of this project's divergence ledger.
- What does darkpawns.net run: the unmodified 2.2 source, or its own changes?
