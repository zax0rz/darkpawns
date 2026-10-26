#!/usr/bin/env python3
"""Validate the 299 x five-route player spell matrix (R5e/R5h)."""
from __future__ import annotations

import csv
import pathlib
import re
import sys
import tempfile

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from fidelity_manifest import MANIFEST_DIR, ROOT, load_rows  # noqa: E402

MATRIX = MANIFEST_DIR / "surface" / "spell-matrix.tsv"
INVENTORY = MANIFEST_DIR / "surface-inventory.tsv"
ROUTES = ("spoken", "potion", "scroll", "wand", "staff")
STATUSES = {"proven", "delegated", "divergent-approved", "excluded-with-C-reason",
            "gap-reachable", "gap-blocked", "unreachable"}
R1A_EXCLUDED_CELLS = {
    (105, "spoken"): ("R1a", "src/spells.c:1055"),
    (31, "potion"): (
        "R1a",
        "src/spells.c:361",
        "src/spell_parser.c:713",
    ),
    (31, "staff"): (
        "R1a",
        "src/spells.c:361",
        "src/spell_parser.c:602",
        "src/spell_parser.c:605",
        "src/spell_parser.c:804",
        "src/spell_parser.c:809",
    ),
}
ROUTINE_MAP = (
    ("MAG_DAMAGE", "mag_damage"),
    ("MAG_AFFECTS", "mag_affects"),
    ("MAG_UNAFFECTS", "mag_unaffects"),
    ("MAG_POINTS", "mag_points"),
    ("MAG_ALTER_OBJS", "mag_alter_objs"),
    ("MAG_GROUPS", "mag_groups"),
    ("MAG_MASSES", "mag_masses"),
    ("MAG_AREAS", "mag_areas"),
    ("MAG_SUMMONS", "mag_summons"),
    ("MAG_CREATIONS", "mag_creations"),
    ("MAG_MANUAL", None),
)
HEADER = ["spell_id", "symbol", "spell_name", "definition", "routine", "body_file"]
for route in ROUTES:
    HEADER.extend((f"{route}_status", f"{route}_case_ids"))
HEADER.append("notes")


def read_tsv(path: pathlib.Path) -> list[dict[str, str]]:
    with path.open(encoding="utf-8", newline="") as stream:
        return list(csv.DictReader(stream, delimiter="\t"))


def c_catalog(root: pathlib.Path):
    """Read the C names, numeric symbols, and active spello() definitions."""
    header = root / "src" / "spells.h"
    parser = root / "src" / "spell_parser.c"
    if not header.exists() or not parser.exists():
        return None
    source = parser.read_text(encoding="latin-1")
    without_comments = re.sub(r"/\*.*?\*/", "", source, flags=re.S)
    table = re.search(r"char\s*\*spells\[\]\s*=\s*\{(.*?)\};", without_comments, re.S)
    if table is None:
        return None
    names = re.findall(r'"((?:\\.|[^"\\])*)"', table.group(1))
    symbols: dict[int, str] = {}
    constants = (root / "src" / "spells.h").read_text(encoding="latin-1")
    for symbol, number_text in re.findall(
        r"^#define\s+((?:SPELL|SKILL)_[A-Z0-9_]+)\s+([0-9]+)\b", constants, re.M
    ):
        number = int(number_text)
        if symbol.startswith("SPELL_TYPE_"):
            continue
        if (1 <= number <= 105 and symbol.startswith("SPELL_")) or (
            131 <= number <= 200 and symbol.startswith("SKILL_")
        ) or (201 <= number <= 206 and symbol.startswith("SPELL_")):
            symbols[number] = symbol
    active: dict[int, int] = {}
    routines: dict[int, str] = {}
    symbol_numbers = {symbol: number for number, symbol in symbols.items()}
    for match in re.finditer(r"\bspello\s*\((.*?)\);", source, re.S):
        args = match.group(1)
        first = re.match(r"\s*(SPELL_[A-Z0-9_]+)", args)
        if not first or first.group(1) not in symbol_numbers:
            continue
        spell_id = symbol_numbers[first.group(1)]
        active[spell_id] = source.count("\n", 0, match.start()) + 1
        flags = args.rsplit(",", 1)[-1]
        routine_names = []
        for macro, routine in ROUTINE_MAP:
            if re.search(rf"\b{macro}\b", flags):
                if macro == "MAG_MANUAL":
                    routine_names.append("manual")
                else:
                    routine_names.append(routine)
        routines[spell_id] = "+".join(routine_names)

    manual_functions: dict[int, str] = {}
    for match in re.finditer(
        r"case\s+(SPELL_[A-Z0-9_]+)\s*:\s*(?:case\s+(SPELL_[A-Z0-9_]+)\s*:\s*)?MANUAL_SPELL\((\w+)\)",
        source,
    ):
        for symbol in (match.group(1), match.group(2)):
            if symbol in symbol_numbers:
                manual_functions[symbol_numbers[symbol]] = match.group(3)

    bodies: dict[int, str] = {}
    for spell_id in active:
        routine = routines.get(spell_id, "")
        if "manual" in routine.split("+"):
            function = manual_functions.get(spell_id)
            routines[spell_id] = routine.replace("manual", function or "MANUAL_UNMAPPED")
            body = None
            if function:
                pattern = re.compile(rf"\bASPELL\s*\(\s*{re.escape(function)}\s*\)")
                body = next((p for p in sorted((root / "src").glob("*.c")) if pattern.search(p.read_text(encoding="latin-1"))), None)
            bodies[spell_id] = body.relative_to(root).as_posix() if body else "src/spell_parser.c"
        else:
            bodies[spell_id] = "src/magic.c"
    return names, symbols, active, routines, bodies


def check(matrix=MATRIX, inventory=INVENTORY, root=ROOT, case_ids=None) -> list[str]:
    errors: list[str] = []
    try:
        with matrix.open(encoding="utf-8", newline="") as stream:
            reader = csv.DictReader(stream, delimiter="\t")
            if reader.fieldnames != HEADER:
                return [f"{matrix}: header must be {HEADER}"]
            rows = list(reader)
    except OSError as exc:
        return [f"{matrix}: {exc}"]

    if len(rows) != 299:
        errors.append(f"{matrix}: {len(rows)} spell rows, want 299")
    if case_ids is None:
        proof_rows = load_rows()
        proof_status = {row["case_id"]: row["status"] for row in proof_rows}
        case_ids = set(proof_status)
    else:
        proof_status = {}

    counts = {route: 0 for route in ROUTES}
    catalog = c_catalog(root)
    for expected_id, row in enumerate(rows, 1):
        where = f"{matrix}:{expected_id + 1}"
        try:
            spell_id = int(row["spell_id"])
        except (KeyError, ValueError):
            errors.append(f"{where}: spell_id must be an integer")
            continue
        if spell_id != expected_id:
            errors.append(f"{where}: spell_id {spell_id}, expected {expected_id}")
        if not row.get("spell_name"):
            errors.append(f"{where}: spell_name is required (use <no spells[] entry>)")
        if not row.get("definition"):
            errors.append(f"{where}: definition is required")
        if not row.get("body_file"):
            errors.append(f"{where}: body_file is required")
        if catalog is not None:
            names, symbols, active, routines, bodies = catalog
            expected_name = names[spell_id] if spell_id < len(names) else "<no spells[] entry>"
            expected_symbol = symbols.get(spell_id, "-")
            if row["spell_name"] != expected_name:
                errors.append(f"{where}: spell_name {row['spell_name']!r} != C spells[] value {expected_name!r}")
            if row["symbol"] != expected_symbol:
                errors.append(f"{where}: symbol {row['symbol']!r} != C symbol {expected_symbol!r}")
            if spell_id in active and f"src/spell_parser.c:{active[spell_id]}" not in row["definition"]:
                errors.append(f"{where}: definition does not cite active C spello() at src/spell_parser.c:{active[spell_id]}")
            if spell_id in active and row["routine"] != routines[spell_id]:
                errors.append(f"{where}: routine {row['routine']!r} != C dispatch {routines[spell_id]!r}")
            if spell_id in active and row["body_file"] != bodies[spell_id]:
                errors.append(f"{where}: body_file {row['body_file']!r} != C body file {bodies[spell_id]!r}")
        for route in ROUTES:
            status = row[f"{route}_status"]
            cited = [item for item in row[f"{route}_case_ids"].split(";") if item and item != "-"]
            if status not in STATUSES:
                errors.append(f"{where}: {route} status {status!r} not in {sorted(STATUSES)}")
            if status in {"proven", "delegated"} and not cited:
                errors.append(f"{where}: {route} {status} must cite case_ids")
            if status in {"proven", "delegated"}:
                for case_id in cited:
                    case_status = proof_status.get(case_id)
                    if case_status is not None and case_status not in {
                        "oracle-green", "oracle-green-multiseed", "unit-green", "delegated"
                    }:
                        errors.append(f"{where}: {route} {status} cites {case_id!r} with status {case_status!r}")
            if status.startswith("gap-") and not row["notes"].strip():
                errors.append(f"{where}: {route} gap needs a source-backed reason in notes")
            if status == "unreachable" and not row["notes"].strip():
                errors.append(f"{where}: {route} unreachable needs a source-backed reason in notes")
            excluded_citations = R1A_EXCLUDED_CELLS.get((spell_id, route))
            if status == "excluded-with-C-reason":
                if excluded_citations is None:
                    errors.append(f"{where}: {route} excluded-with-C-reason cell is not in the explicit R1a allowlist")
                else:
                    missing_citations = [cite for cite in excluded_citations if cite not in row["notes"]]
                    if missing_citations:
                        errors.append(
                            f"{where}: {route} excluded-with-C-reason notes must cite {', '.join(missing_citations)}"
                        )
                allowed = {"excluded-with-C-reason"} if excluded_citations is not None else set()
            else:
                allowed = None
            if status == "gap-blocked":
                errors.append(f"{where}: {route} gap-blocked is not an approved terminal classification")
                allowed = set()
            elif allowed is None and route == "spoken":
                allowed = {"proven", "delegated", "gap-reachable"} if 1 <= spell_id <= 102 else {"unreachable"}
            elif allowed is None:
                allowed = {"proven", "delegated", "gap-reachable"} if spell_id <= 103 else {"unreachable"}
            if status not in allowed:
                errors.append(f"{where}: {route} status {status!r} contradicts the C/OLC route boundary for spell {spell_id}")
            for case_id in cited:
                if case_id not in case_ids:
                    errors.append(f"{where}: unknown case_id {case_id!r}")
            if status.startswith("gap-"):
                counts[route] += 1

    try:
        inv_rows = read_tsv(inventory)
    except OSError as exc:
        return errors + [f"{inventory}: {exc}"]
    by_family = {row["family"]: row for row in inv_rows}
    for route in ROUTES:
        family = f"CAST_{'SPELL' if route == 'spoken' else route.upper()} {'player cast' if route == 'spoken' else {'potion': 'quaff', 'scroll': 'recite', 'wand': 'use', 'staff': 'use'}[route]} vector"
        if route == "wand":
            family = "CAST_WAND use vector"
        elif route == "staff":
            family = "CAST_STAFF use vector"
        row = by_family.get(family)
        if row is None:
            errors.append(f"surface-inventory.tsv: missing {family!r}")
            continue
        if row["units"] != str(counts[route]):
            errors.append(f"surface-inventory.tsv: {family} units {row['units']} != {counts[route]} matrix gaps")
        if (row["status"] == "proven-already") != (counts[route] == 0):
            errors.append(f"surface-inventory.tsv: {family} status {row['status']!r} disagrees with {counts[route]} gaps")
        if "surface/spell-matrix.tsv" not in row["proof_or_attempt"].split(";"):
            errors.append(f"surface-inventory.tsv: {family} must reference surface/spell-matrix.tsv")
    return errors


def self_test() -> None:
    """Mutations of the validator's key boundaries must fail (R5h)."""
    with tempfile.TemporaryDirectory() as tmp:
        root = pathlib.Path(tmp)
        manifest = root / "depth"
        (manifest / "surface").mkdir(parents=True)
        matrix = manifest / "surface" / "spell-matrix.tsv"
        inventory = manifest / "surface-inventory.tsv"
        rows = []
        counts = {route: 0 for route in ROUTES}
        for spell_id in range(1, 300):
            row = {key: "-" for key in HEADER}
            row.update({"spell_id": str(spell_id), "symbol": "-", "spell_name": f"spell {spell_id}",
                        "definition": "unused", "routine": "-", "body_file": "src/spells.c", "notes": "boundary"})
            if spell_id == 31:
                row["notes"] = (
                    "R1a src/spells.c:361 src/spell_parser.c:713 "
                    "src/spell_parser.c:602 src/spell_parser.c:605 "
                    "src/spell_parser.c:804 src/spell_parser.c:809"
                )
            for route in ROUTES:
                row[f"{route}_status"] = (
                    "excluded-with-C-reason" if (spell_id, route) in R1A_EXCLUDED_CELLS else
                    "gap-reachable" if (route == "spoken" and spell_id <= 102) or (route != "spoken" and spell_id <= 103) else
                    "unreachable"
                )
                if row[f"{route}_status"] == "excluded-with-C-reason":
                    row["notes"] += " R1a src/spells.c:1055 src/spell_parser.c:713 src/spells.c:361 src/spell_parser.c:602 src/spell_parser.c:605 src/spell_parser.c:804 src/spell_parser.c:809"
                if row[f"{route}_status"].startswith("gap-"):
                    counts[route] += 1
            rows.append(row)

        def write_inventory():
            with inventory.open("w", encoding="utf-8", newline="") as stream:
                writer = csv.DictWriter(stream, fieldnames=["surface_class", "family", "units", "status", "proof_or_attempt", "c_source_scope", "notes"], delimiter="\t")
                writer.writeheader()
                names = {"spoken": "CAST_SPELL player cast vector", "potion": "CAST_POTION quaff vector", "scroll": "CAST_SCROLL recite vector", "wand": "CAST_WAND use vector", "staff": "CAST_STAFF use vector"}
                for route in ROUTES:
                    writer.writerow({"surface_class": "spell-matrix", "family": names[route], "units": str(counts[route]),
                                     "status": "blocked" if counts[route] else "proven-already", "proof_or_attempt": "surface/spell-matrix.tsv", "c_source_scope": "src/spell_parser.c", "notes": "test"})

        def run():
            with matrix.open("w", encoding="utf-8", newline="") as stream:
                writer = csv.DictWriter(stream, fieldnames=HEADER, delimiter="\t")
                writer.writeheader()
                writer.writerows(rows)
            write_inventory()
            return check(matrix, inventory, root, set())

        assert run() == [], run()
        rows[4]["potion_status"] = "excluded-with-C-reason"
        errors = run()
        assert any("potion excluded-with-C-reason cell is not in the explicit R1a allowlist" in error for error in errors), (
            "self-test: an unapproved R1a item cell was accepted"
        )
        rows[4]["potion_status"] = "gap-reachable"
        rows[102]["spoken_status"] = "gap-reachable"
        counts["spoken"] += 1
        assert run(), "self-test: unreachable route cell was not caught"
        rows[102]["spoken_status"] = "unreachable"
        counts["spoken"] -= 1
        rows[0]["spoken_status"] = "proven"
        rows[0]["spoken_case_ids"] = "missing.case"
        counts["spoken"] -= 1
        assert run(), "self-test: unknown case was not caught"


def main(argv: list[str]) -> int:
    if "--self-test" in argv:
        self_test()
        print("spell-matrix: self-test ok")
        return 0
    errors = check()
    for error in errors:
        print(error)
    if MATRIX.exists():
        rows = read_tsv(MATRIX)
        for route in ROUTES:
            classes = {status: 0 for status in STATUSES}
            for row in rows:
                classes[row[f"{route}_status"]] = classes.get(row[f"{route}_status"], 0) + 1
            print(
                f"{route}: proven={classes['proven']} delegated={classes['delegated']} "
                f"gap-reachable={classes['gap-reachable']} gap-blocked={classes['gap-blocked']} "
                f"unreachable={classes['unreachable']}"
            )
    print(f"spell-matrix: 299 rows, {len(errors)} error(s)")
    return 1 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
