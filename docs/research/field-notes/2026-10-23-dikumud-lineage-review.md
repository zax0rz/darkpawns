# DikuMUD lineage review, 2026-10-23

Track: historical provenance of the game code. Question: does Zach's supplied family tree describe Dark Pawns' ancestry, and which other connections survive primary-source checking?

This is a bounded source review, not a declaration that the entire tree is correct. Original release readmes, licenses and credits were read through GitHub's API. Their mirrors are preservation sources, not the original authors' current repositories. Public project histories provide context. No downloaded code was executed.

## Findings

- The preserved Frontline/Orodreth history records CircleMUD 1.7 (September 1994), 2.2 (December 1995), 3.0pl9 (March 1996), and 3.0pl11 (June 1996). `src/config.c:249` and `lib/text/credits:1` independently identify the surviving C game as CircleMUD 3.0. Place the later game below Circle 3; do not imply that was the founding base.
- CircleMUD's author calls 2.0 the first public release. The official archive checked here does not corroborate a public 1.7 release. Preserve 1.7 as the historical account's wording, without inventing an explanation or identifying a specific 1.x archive.
- AberMUD is design influence here, not an asserted code parent.
- Lost Lands' preserved release credits CircleMUD 3.0. Move that surviving release under Circle 3 while leaving the earliest fork point unresolved. This does not refute the possibility of an earlier Circle 2 release.
- The supplied diagram places Vego under Merc 1. A contemporary operator description in the 1996 FTPLIST identifies Merc 2.1 with 2.2 code: https://groups.google.com/g/rec.games.mud.admin/c/N3_W_9E33As . Display it under 2.x but retain unconfirmed status because earlier ancestry is not settled.
- SMAUG and GodWars identify Merc 2.1; Envy 1.0 identifies Merc 2.2. The compact chart's Merc 2.x node spans those releases and must not imply that all three start from the same patch level.
- ROM starts on Merc 1.0 and later uses 2.1. This is not a strict tree: reuse and upgrades create additional relationships. The explanatory notes disclose that limitation.
- NakedMUD's maintainer confirms SocketMUD as its foundation. No primary source checked here establishes the dotted Merc-to-SocketMUD connection. Keep that family detached rather than asserting independent invention or Merc descent. RocketMUD is not silently substituted with the similarly named RockMUD.
- DPReturns and this Go restoration are separate descendants of the surviving original Dark Pawns.

## Boundaries and provenance

The user selected the broader family rather than direct ancestry only. Preserve reported but unresolved connections with dashed lines and explicit text status. A solid line means the cited document supports ancestry or progression, not that every intermediate release or code contribution has been traced. The chart's many minor branches remain leads; no claim of exhaustive verification is made.

The supplied bitmap is not redistributed. Its original author/license and exact publication revision are unknown. The new vector is independently laid out from relationship data, with sources attached to edges. No generated illustration or raster is needed.

Source bitmap SHA-256: `7814d61fb3465ff7131bd1c87976b060d6347fc7060bf7bee3d1c44af4550879`.

The reconstruction has 57 nodes and 55 connections: 32 supported, 22 unconfirmed, 1 influence. Node/version grouping is editorial compression, not a completeness claim.

## Connection audit

| Parent | Child | State | Evidence / limitation |
|---|---|---|---|
| AberMUD | DikuMUD | influence | [DikuMUD: history](https://dikumud.com/history/) Design influence, not a claim that Diku uses AberMUD source. |
| DikuMUD | SillyMUD | confirmed | [DaleMUD: surviving credits](https://github.com/DikuMUDOmnibus/DaleMUD/blob/b92ba1bf4809826c465d774b179f3f856dbf9462/lib/credits)  |
| DikuMUD | VieMUD | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| DikuMUD | CircleMUD 1.x | confirmed | [Jeremy Elson: the history of CircleMUD](https://www.circlemud.org/general.html)  |
| DikuMUD | Diku II | confirmed | [DikuMUD: history](https://dikumud.com/history/)  |
| DikuMUD | Sequent | confirmed | [DikuMUD: history](https://dikumud.com/history/)  |
| DikuMUD | Eltanin | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| DikuMUD | Techno | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| DikuMUD | Crimson | confirmed | [CrimsonMUD: original credits](https://github.com/DikuMUDOmnibus/Crimson/blob/fa4486011e7e7fdb4b35fe367093d4959cbb1a65/lib/credits)  |
| SillyMUD | BlackMUD | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| SillyMUD | DaleMUD | confirmed | [DaleMUD: surviving credits](https://github.com/DikuMUDOmnibus/DaleMUD/blob/b92ba1bf4809826c465d774b179f3f856dbf9462/lib/credits)  |
| CircleMUD 1.x | CircleMUD 2.x | confirmed | [Jeremy Elson: the history of CircleMUD](https://www.circlemud.org/general.html)  |
| CircleMUD 2.x | Archipelago | confirmed | [Archipelago: original README](https://github.com/DikuMUDOmnibus/Archipelago/blob/e512852b07c894bbea086f79c49dd89b36500808/README)  |
| CircleMUD 2.x | CircleMUD 3.x | confirmed | [Jeremy Elson: the history of CircleMUD](https://www.circlemud.org/general.html)  |
| CircleMUD 3.x | tbaMUD | confirmed | [The Builder Academy](https://www.tbamud.com/) Continuation through CircleMUD 3.1 and later releases; intermediate versions are omitted. |
| CircleMUD 3.x | Dark Pawns | confirmed | [Frontline: a brief history of Dark Pawns](/archive/history/) March 1996: 3.0pl9. June 1996: 3.0pl11. Earlier bases were 1.7 and 2.2. |
| CircleMUD 3.x | Lost Lands | confirmed | [Lost Lands: surviving credits](https://github.com/DikuMUDOmnibus/LostLands/blob/b889e86937e6ebb49abbca5b2c4d286b5e4e158f/lib/text/credits) Surviving release credits 3.0; moved here from Circle 2 in the supplied chart. Earlier ancestry is unresolved. |
| Dark Pawns | Go restoration | confirmed | [Dark Pawns: Go restoration](https://github.com/zax0rz/darkpawns) Port of the surviving original C game, not of DPReturns. |
| Dark Pawns | DPReturns | confirmed | [Dark Pawns Returns: project history](https://github.com/RustySpeedy/darkpawns-returns/blob/master/README.md) A separate revival of the C game. |
| Sequent | Nilgiri | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| Sequent | Copper I | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| Sequent | Fajita | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| Copper I | Copper II | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| Copper I | Pirate | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| Copper I | Merc 1.x | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| Copper II | Copper III | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| Merc 1.x | ROM 1.0 | confirmed | [ROM: original credits in Dawn](https://github.com/DikuMUDOmnibus/Dawn/blob/127020af06eac484b98da75ab613430cba6affdb/help/credits.txt)  |
| Merc 1.x | Merc 2.x | confirmed | [Merc 2.2: original README](https://github.com/DikuMUDOmnibus/Merc/blob/1d2ea7a92f826ca50945622c9df8945eff7d6ae3/README)  |
| Merc 2.x | Vego | unconfirmed | Supplied chart only; primary confirmation remains open. The supplied chart places Vego under Merc 1; a 1996 operator listing describes Merc 2.1 with 2.2 code. Exact early history remains unresolved. |
| ROM 1.0 | ROM 2.3 | confirmed | [ROM: original credits in Dawn](https://github.com/DikuMUDOmnibus/Dawn/blob/127020af06eac484b98da75ab613430cba6affdb/help/credits.txt)  |
| ROM 2.3 | ROM 2.4 | confirmed | [ROM: original credits in Dawn](https://github.com/DikuMUDOmnibus/Dawn/blob/127020af06eac484b98da75ab613430cba6affdb/help/credits.txt)  |
| ROM 2.3 | SunderMUD | confirmed | [SunderMUD: original credits](https://github.com/DikuMUDOmnibus/SunderMUD/blob/3dfedeab7e7bf8863d53e2d9d971dbfc0ab1fa0b/doc/SunderMud.credits)  |
| ROM 2.3 | EmberMUD | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| ROM 2.4 | Anatolia | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| ROM 2.4 | RoT | confirmed | [RoT: preserved credits](https://wiki.cleftofdimensions.net/index.php?title=Credits&oldid=2375)  |
| ROM 2.4 | Oblivion | confirmed | [Dawn: original licenses](https://github.com/DikuMUDOmnibus/Dawn/blob/127020af06eac484b98da75ab613430cba6affdb/src/docs/licenses.txt)  |
| Oblivion | Dawn of Time | confirmed | [Dawn: original licenses](https://github.com/DikuMUDOmnibus/Dawn/blob/127020af06eac484b98da75ab613430cba6affdb/src/docs/licenses.txt)  |
| Merc 2.x | ACK!MUD | confirmed | [ACK!MUD: original license and credits](https://github.com/DikuMUDOmnibus/AckMUD/blob/a2b2e17373b3b95bb8acce747ea1db17afba7bd3/docs/acklicense.txt)  |
| Merc 2.x | Envy 1.0 | confirmed | [EnvyMUD 2.0: original README](https://github.com/DikuMUDOmnibus/EnvyMUD/blob/0bab3b701ab23247d2788cf37335106adbedb85c/README.old)  |
| Merc 2.x | SMAUG | confirmed | [SMAUG: original license](https://github.com/DikuMUDOmnibus/AFKMud/blob/f2f464c598015c23373775130c3d9f8f1ee80654/doc/license.smaug)  |
| Merc 2.x | GodWars | confirmed | [GodWars: original license](https://github.com/DikuMUDOmnibus/God-Wars/blob/e4cecaf3627f424132b0322f7db6ec0942d8e717/doc/license.gw)  |
| Merc 2.x | MrMud | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| Merc 2.x | NiMUD | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| ACK!MUD | ASSAULT | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| Envy 1.0 | Envy 2.x | confirmed | [EnvyMUD 2.0: original README](https://github.com/DikuMUDOmnibus/EnvyMUD/blob/0bab3b701ab23247d2788cf37335106adbedb85c/README.old)  |
| Envy 2.x | Mythran | confirmed | [Mythran: original license](https://github.com/DikuMUDOmnibus/MythranMUD/blob/e45ef5a4500e7d69f25a3502017c3010438732cf/doc/license.myt)  |
| Envy 2.x | UltraEnvy | confirmed | [UltraEnvy: original license](https://github.com/DikuMUDOmnibus/Ultra-Envy/blob/240666877f5c616bfbf9af803bd9dbade0f48bd3/doc/license.ue)  |
| Envy 2.x | EOS | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| SMAUG | AFKMud | confirmed | [AFKMud: original README](https://github.com/DikuMUDOmnibus/AFKMud/blob/f2f464c598015c23373775130c3d9f8f1ee80654/README.AFKMUD)  |
| SMAUG | SERF | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| GodWars | Dystopia | confirmed | [Dystopia: original license](https://github.com/DikuMUDOmnibus/Dystopia/blob/9b9310d89642a8d0086c4690a03c7e609ec88256/doc/license/license.dystopia)  |
| MrMud | Emud | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| NiMUD | EmlenMUD | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| SocketMUD | RocketMUD | unconfirmed | Supplied chart only; primary confirmation remains open.  |
| SocketMUD | NakedMUD | confirmed | [NakedMUD: maintainer’s project history](https://github.com/NakedMud/nakedmud)  |

## Source notes

- [DikuMUD: history](https://dikumud.com/history/): The original team’s account of Diku, Sequent, Diku II, and the influences behind the game.
- [Jeremy Elson: the history of CircleMUD](https://www.circlemud.org/general.html): Diku ancestry, the original Circle game, and the first public release, CircleMUD 2.0.
- [Frontline: a brief history of Dark Pawns](/archive/history/): Records CircleMUD 1.7 in 1994, 2.2 in December 1995, 3.0pl9 in March 1996, and 3.0pl11 in June 1996.
- [Dark Pawns: surviving C credits](https://github.com/zax0rz/darkpawns/blob/42d7252f1722085bea89b5d9ce5e8694d820450f/lib/text/credits): The surviving game credits CircleMUD 3.0 and DikuMUD Gamma 0.0.
- [The Builder Academy](https://www.tbamud.com/): The project describes tbaMUD as the continued development of CircleMUD.
- [Archipelago: original README](https://github.com/DikuMUDOmnibus/Archipelago/blob/e512852b07c894bbea086f79c49dd89b36500808/README): Explicitly based on CircleMUD 2.2.
- [Lost Lands: surviving credits](https://github.com/DikuMUDOmnibus/LostLands/blob/b889e86937e6ebb49abbca5b2c4d286b5e4e158f/lib/text/credits): This surviving release credits CircleMUD 3.0. Its earliest fork point is not established here.
- [DaleMUD: surviving credits](https://github.com/DikuMUDOmnibus/DaleMUD/blob/b92ba1bf4809826c465d774b179f3f856dbf9462/lib/credits): Preserves the SillyMUD and DikuMUD credits.
- [Merc 2.2: original README](https://github.com/DikuMUDOmnibus/Merc/blob/1d2ea7a92f826ca50945622c9df8945eff7d6ae3/README): Describes Merc as a Diku derivative. The precise Copper fork point in the supplied chart remains unconfirmed.
- [ROM: original credits in Dawn](https://github.com/DikuMUDOmnibus/Dawn/blob/127020af06eac484b98da75ab613430cba6affdb/help/credits.txt): Records ROM beginning on Merc 1.0 and the later ROM 2.3 and 2.4 releases.
- [ROM 2.4: original README](https://github.com/DikuMUDOmnibus/ROM/blob/f03fc881e1b7c9e06465bc61d86d28317fd52a6b/README.rom): Identifies the release as ROM 2.4 on Merc 2.1 base code.
- [RoT: preserved credits](https://wiki.cleftofdimensions.net/index.php?title=Credits&oldid=2375): The RoT credits identify ROM 2.4 as its base.
- [SunderMUD: original credits](https://github.com/DikuMUDOmnibus/SunderMUD/blob/3dfedeab7e7bf8863d53e2d9d971dbfc0ab1fa0b/doc/SunderMud.credits): Identifies the ROM 2.3 variant used at Dark Times as its base.
- [Dawn: original licenses](https://github.com/DikuMUDOmnibus/Dawn/blob/127020af06eac484b98da75ab613430cba6affdb/src/docs/licenses.txt): Names ROM 2.4, Oblivion 2.0, and Dawn of Time in the ancestry.
- [ACK!MUD: original license and credits](https://github.com/DikuMUDOmnibus/AckMUD/blob/a2b2e17373b3b95bb8acce747ea1db17afba7bd3/docs/acklicense.txt): Identifies modified Merc 2.2 and credits Merc 2.0, 2.1, and 2.2 code.
- [EnvyMUD 2.0: original README](https://github.com/DikuMUDOmnibus/EnvyMUD/blob/0bab3b701ab23247d2788cf37335106adbedb85c/README.old): Explicitly traces Envy 1.0 to Merc 2.2 and Envy 2.0 to Envy 1.0.
- [UltraEnvy: original license](https://github.com/DikuMUDOmnibus/Ultra-Envy/blob/240666877f5c616bfbf9af803bd9dbade0f48bd3/doc/license.ue): The surviving UltraEnvy release traces through Envy 2.2, itself based on Envy 2.0.
- [Mythran: original license](https://github.com/DikuMUDOmnibus/MythranMUD/blob/e45ef5a4500e7d69f25a3502017c3010438732cf/doc/license.myt): Explicitly based on EnvyMUD 2.0.
- [SMAUG: original license](https://github.com/DikuMUDOmnibus/AFKMud/blob/f2f464c598015c23373775130c3d9f8f1ee80654/doc/license.smaug): Explicitly based on Merc 2.1.
- [AFKMud: original README](https://github.com/DikuMUDOmnibus/AFKMud/blob/f2f464c598015c23373775130c3d9f8f1ee80654/README.AFKMUD): The release retains the SMAUG licensing and documents changes to the SMAUG engine.
- [GodWars: original license](https://github.com/DikuMUDOmnibus/God-Wars/blob/e4cecaf3627f424132b0322f7db6ec0942d8e717/doc/license.gw): Explicitly based on Merc 2.1.
- [Dystopia: original license](https://github.com/DikuMUDOmnibus/Dystopia/blob/9b9310d89642a8d0086c4690a03c7e609ec88256/doc/license/license.dystopia): Explicitly based on God Wars.
- [NakedMUD: maintainer’s project history](https://github.com/NakedMud/nakedmud): Identifies Brian Graversen’s SocketMUD as the foundation. No Merc descent is asserted here.
- [Dark Pawns Returns: project history](https://github.com/RustySpeedy/darkpawns-returns/blob/master/README.md): The C revival of Dark Pawns, distinct from this Go restoration.
- [Dark Pawns: Go restoration](https://github.com/zax0rz/darkpawns): The current project ports the surviving C game. It does not descend from DPReturns.
- [CrimsonMUD: original credits](https://github.com/DikuMUDOmnibus/Crimson/blob/fa4486011e7e7fdb4b35fe367093d4959cbb1a65/lib/credits): Names the original Diku source and CrimsonMUD modifications.

## Evidence retention

API tree manifests, original text files and commit revisions are retained locally under `~/Archives/darkpawns/lineage/2026-10-23/`. Public claims cite frozen GitHub commit paths where available. Existing `docs/research/EVIDENCE_LEDGER.tsv` receives only the verified Dark Pawns claim and the explicitly partial wider review. New About copy remains for Zach’s editorial approval before publication; no production change is part of this task.

## Editorial follow-up, 2026-10-24

Zach reviewed the About reconstruction. Requested plain “red line” wording and a collapsed reference register by default; both applied and verified. BlackMUD’s connection stays unconfirmed. A private first-hand lead may guide further investigation but is not reproduced or promoted to a public source. No production deployment requested.
