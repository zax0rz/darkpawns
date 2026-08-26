## Dated Handoff: 2026-08-25
- Object-magic effect messages (round 1 of the depth-fidelity loop) are captured
  in `docs/fidelity/depth/object-magic.tsv`: every `mag_affects` switch arm was
  audited against `src/magic.c`'s to_vict/to_room/to_self strings and the
  magic.c:1414-1421 send block (to_self gated on `ch != victim`, then to_vict,
  then to_room). Missing lines were added across the whole class (R5c),
  invented curse/poison/sleep lines were removed (R4), and a `$M`/`$m`
  objective-pronoun helper now backs the substitutions.
- Nine single-spell, zero-RNG potions/scrolls prove the live bytes (one
  wait-setting item command per scenario, quaff-effect pattern), including
  peer-proven to_room lines and targeted-recite to_self lines.
- Save-gated spells (chill touch, blindness, curse, poison, sleep, flamestrike)
  are unit-proven with the save outcome fixed; live oracle parity stays blocked
  pending call_magic RNG-stream work.
- Harness: quaff/recite set WAIT_STATE PULSE_VIOLENCE, so post-item commands
  stall behind the frozen clock — keep the item command last and pad with
  `~dpclock pulse 20` steps. Zone-reset loads roll `percent_load`, so
  probabilistic prototypes need the new `force-load <vnum>` fixture; boot-loaded
  vnums need spawn max headroom.
