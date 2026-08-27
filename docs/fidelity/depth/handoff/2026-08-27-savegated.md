# Dated Handoff: 2026-08-27 (save-gated live round)

- The round-1 blanket note "save-gated live oracle parity blocked pending
  call_magic RNG-stream verification" is retired: quaff-poison-save proves the
  class live. Quaffing 4399 (level 30: invigorate → vitality → poison) rolls
  the poison save after two points spells' dice draws, and twelve seeds are
  GREEN with BOTH arms exercised — seeds 1-4,6-12 fail the save ("You feel
  very sick."), seed 5 saves ("Nothing seems to happen.").
- Scenario shape notes: the start room's ROOM_PEACEFUL (bit 4) must be turned
  off via set-room-flag or call_magic dispels the violent spell before any
  save ("A flash of white light fills the room..."); 4399's boot load is 20%
  so force-load applies; its file 43.obj is the only save-gated-spell item in
  the game's obj index — blindness (13103) and curse (5804) vehicles live in
  131.obj/58.obj, which the index omits entirely (same class of omission as
  180.obj/guinness), so those arms stay unit-green with fixed saves.
- The row also carries forward Claude's #656/#657 context: mag_points draw
  order and damage-spell skill_message parity were the prerequisites that
  made this scenario's stream align.
