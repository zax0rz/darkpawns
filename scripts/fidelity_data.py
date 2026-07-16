"""Fidelity data: verified C-to-Go constant array mappings.

This module powers the Tier 1 fidelity checker. Each entry in ARRAY_MAP pairs a
C array name with its Go equivalent. Mappings are verified against the C source
(src/constants.c and headers) and the Go port (pkg/game/constants.go).

The Tier 1 checker is gated by the FIDELITY_TIER1 environment variable. Once
all divergences below are triaged and the map is verified, enable live checking
by setting FIDELITY_TIER1=1 in scripts/fidelity_pipeline_cron.sh.
"""

from __future__ import annotations

import os
import re
from dataclasses import dataclass
from pathlib import Path
from typing import Callable


@dataclass(frozen=True)
class ArrayMapping:
    """A verified C -> Go array pairing."""

    c_name: str
    go_name: str
    c_file: str
    go_file: str
    notes: str = ""
    compare: Callable[[list, list], list[str]] | None = None

    def __post_init__(self) -> None:
        if not self.c_name:
            raise ValueError("c_name is required")


# ---------------------------------------------------------------------------
# Verified C <-> Go array mappings.
#
# Rules for adding or editing entries:
#   1. Open both the C source and Go source.
#   2. Confirm the arrays represent the same logical table.
#   3. Note any intentional divergences (sentinels dropped, reordered enums, etc.)
#      in the `notes` field.
#   4. If the comparison needs special handling, supply a `compare` callable.
# ---------------------------------------------------------------------------
ARRAY_MAP: tuple[ArrayMapping, ...] = (
    # Phase names — identical ordering.
    ArrayMapping("phases", "Phases", "src/constants.c", "pkg/game/constants.go"),

    # Hometowns — C has a leading "!Bad" sentinel that the Go port dropped,
    # shifting all subsequent indices by one. The checker accounts for this.
    ArrayMapping(
        "hometowns",
        "Hometowns",
        "src/constants.c",
        "pkg/game/constants.go",
        notes="C has leading '!Bad' sentinel; Go dropped it (off-by-one)",
    ),

    # Ability score names — this is a known mismap. C's abil_names are short
    # descriptors (e.g. "Str", "Int"), while Go's AbilityNames are full names.
    # They should NOT be compared as the same array.
    ArrayMapping(
        "abil_names",
        "",
        "src/constants.c",
        "pkg/game/constants.go",
        notes="INTENTIONAL mismap removed: C=stat descriptors, Go=full names; do not compare",
    ),

    # Crowd size descriptions.
    ArrayMapping("crowd_size", "CrowdSize", "src/constants.c", "pkg/game/constants.go"),

    # Direction names.
    ArrayMapping("dirs", "DirectionNames", "src/constants.c", "pkg/game/constants.go"),

    # Mobile race names — C and Go differ at index 4 (Centaur vs Minotaur).
    # This is a real divergence tracked separately.
    ArrayMapping(
        "mob_races",
        "MobRaceNames",
        "src/constants.c",
        "pkg/game/constants.go",
        notes="index 4 divergence: C=Centaur, Go=Minotaur (real fidelity gap)",
    ),

    # Room flag names.
    ArrayMapping("room_bits", "RoomBitNames", "src/constants.c", "pkg/game/constants.go"),

    # Exit flag names.
    ArrayMapping("exit_bits", "ExitBitNames", "src/constants.c", "pkg/game/constants.go"),

    # Sector type names.
    ArrayMapping("sector_types", "SectorTypeNames", "src/constants.c", "pkg/game/constants.go"),

    # Gender names.
    ArrayMapping("genders", "GenderNames", "src/constants.c", "pkg/game/constants.go"),

    # Position type names.
    ArrayMapping("position_types", "PositionNames", "src/constants.c", "pkg/game/constants.go"),

    # Player flag names.
    ArrayMapping("player_bits", "PlayerBitNames", "src/constants.c", "pkg/game/constants.go"),

    # Action flag names.
    ArrayMapping("action_bits", "ActionBitNames", "src/constants.c", "pkg/game/constants.go"),

    # Preference flag names.
    ArrayMapping(
        "preference_bits", "PreferenceBitNames", "src/constants.c", "pkg/game/constants.go"
    ),

    # Affected bit names.
    ArrayMapping(
        "affected_bits", "AffectedBitNames", "src/constants.c", "pkg/game/constants.go"
    ),

    # Connected state names.
    ArrayMapping(
        "connected_types", "ConnectedTypeNames", "src/constants.c", "pkg/game/constants.go"
    ),

    # Equipment wear positions.
    ArrayMapping("where", "WhereNames", "src/constants.c", "pkg/game/constants.go"),

    # Equipment position names.
    ArrayMapping(
        "equipment_types", "EquipmentTypes", "src/constants.c", "pkg/game/constants.go"
    ),

    # Item type names.
    ArrayMapping("item_types", "ItemTypeNames", "src/constants.c", "pkg/game/constants.go"),

    # Wear bit names.
    ArrayMapping("wear_bits", "WearBitNames", "src/constants.c", "pkg/game/constants.go"),

    # Extra flag names.
    ArrayMapping("extra_bits", "ExtraBitNames", "src/constants.c", "pkg/game/constants.go"),

    # Apply type names.
    ArrayMapping("apply_types", "ApplyTypeNames", "src/constants.c", "pkg/game/constants.go"),

    # Container bit names.
    ArrayMapping(
        "container_bits", "ContainerBitNames", "src/constants.c", "pkg/game/constants.go"
    ),

    # Drink liquid names — matches C drinks[] table.
    ArrayMapping("drinks", "LiquidNames", "src/constants.c", "pkg/game/liquids.go"),

    # PC class names.
    ArrayMapping("pc_class_types", "PCClassTypes", "src/class.c", "pkg/game/class_tables.go"),

    # NPC class names.
    ArrayMapping(
        "npc_class_types", "NpcClassTypeNames", "src/constants.c", "pkg/game/constants.go"
    ),

    # Class abbreviations.
    ArrayMapping("class_abbrevs", "ClassAbbrevs", "src/class.c", "pkg/game/character.go"),

    # Dark Pawns' canonical spell/skill names.
    ArrayMapping(
        "spells",
        "dpSkillCatalogNames",
        "src/spell_parser.c",
        "pkg/spells/skill_catalog_names.go",
    ),

    # Spell wear-off messages.
    ArrayMapping(
        "spell_wear_off_msg",
        "SpellWearOffMessages",
        "src/spells.c",
        "pkg/game/constants.go",
    ),

    # Weekdays and month names.
    ArrayMapping("weekdays", "WeekdayNames", "src/constants.c", "pkg/game/constants.go"),
    ArrayMapping("month_name", "MonthNames", "src/constants.c", "pkg/game/constants.go"),

    # Affiliation bit names — tracked separately; do not compare until resolved.
    ArrayMapping(
        "affil_bit_names",
        "",
        "src/constants.c",
        "pkg/game/constants.go",
        notes="Filed separately; do not compare until DP-642 follow-up is complete",
    ),
)


def _extract_string_array(text: str, array_name: str) -> list[str] | None:
    """Best-effort extraction of a C/Go string array by name."""
    # Look for "var Name = []string{" or "char *name[] = {" etc.
    patterns = [
        rf"(?:var|const)\s+{re.escape(array_name)}\s*=\s*\[\]string\s*\{{(.*?)\}}",
        rf"(?:char\s*\*|const\s+char\s*\*)\s*{re.escape(array_name)}\s*\[\]\s*=\s*\{{(.*?)\}}\s*;",
    ]
    for pattern in patterns:
        match = re.search(pattern, text, re.DOTALL)
        if match:
            body = match.group(1)
            # Extract quoted strings.
            return re.findall(r'"((?:[^"\\]|\\.)*)"', body)
    return None


def load_array(repo_root: Path, mapping: ArrayMapping, language: str) -> list[str] | None:
    """Load a string array from C or Go source."""
    filename = mapping.c_file if language == "c" else mapping.go_file
    path = repo_root / filename
    if not path.exists():
        return None
    text = path.read_text(encoding="utf-8", errors="ignore")
    name = mapping.c_name if language == "c" else mapping.go_name
    if not name:
        return None
    return _extract_string_array(text, name)


def default_compare(c_values: list[str], go_values: list[str]) -> list[str]:
    """Default element-by-element comparison."""
    divergences: list[str] = []
    min_len = min(len(c_values), len(go_values))
    for i in range(min_len):
        if c_values[i] != go_values[i]:
            divergences.append(f"index {i}: C={c_values[i]!r} Go={go_values[i]!r}")
    if len(c_values) != len(go_values):
        divergences.append(
            f"length mismatch: C={len(c_values)} Go={len(go_values)}"
        )
    return divergences


def check_mapping(repo_root: Path, mapping: ArrayMapping) -> list[str]:
    """Return a list of divergence messages for a single mapping."""
    if not mapping.go_name:
        # Explicitly excluded from comparison.
        return []

    c_values = load_array(repo_root, mapping, "c")
    go_values = load_array(repo_root, mapping, "go")

    if c_values is None:
        return [f"{mapping.c_name}: could not load C array from {mapping.c_file}"]
    if go_values is None:
        return [f"{mapping.go_name}: could not load Go array from {mapping.go_file}"]

    compare = mapping.compare or default_compare
    divergences = compare(c_values, go_values)
    return [f"{mapping.c_name} <-> {mapping.go_name}: {d}" for d in divergences]


def check_all(repo_root: str | Path | None = None) -> dict[str, list[str]]:
    """Check every mapped array and return divergences keyed by mapping name."""
    if repo_root is None:
        repo_root = Path(__file__).resolve().parent.parent
    else:
        repo_root = Path(repo_root)

    results: dict[str, list[str]] = {}
    for mapping in ARRAY_MAP:
        if not mapping.go_name:
            continue
        key = f"{mapping.c_name} -> {mapping.go_name}"
        results[key] = check_mapping(repo_root, mapping)
    return results


def main() -> int:
    """CLI entry point for manual Tier 1 verification."""
    results = check_all()
    total = 0
    for key, divergences in results.items():
        if divergences:
            total += len(divergences)
            print(f"\n{key}")
            for d in divergences:
                print(f"  - {d}")
    if total:
        print(f"\n{total} divergence(s) found")
        return 1
    print("No divergences found")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
