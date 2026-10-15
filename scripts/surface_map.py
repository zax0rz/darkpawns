#!/usr/bin/env python3
"""Validate the per-site C surface maps in docs/fidelity/depth/surface/.

A surface map lists every act()/send_to_char()/stc() call site of one C file
and classifies it (proven, gap-reachable, gap-blocked, unreachable). This
checker proves the map is mechanically sound; it cannot prove the reasoning
behind a classification (that is review's job, R5e):

- the header is exact and every row has five fields;
- each c_site is src/<file>.c:<line>, inside src/, and that line is an output
  call (act, send_to_char, stc), so a stale line number fails;
- no site is listed twice, and every output call in the C file is listed;
- the classification is one of the four;
- a proven site cites case_ids, and every cited case_id exists in a depth
  manifest;
- the family's surface-inventory row agrees: units equals the gap count, and
  the row is proven-already exactly when there are no gaps.
"""
from __future__ import annotations

import csv
import pathlib
import re
import sys
import tempfile

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from fidelity_manifest import MANIFEST_DIR, ROOT, load_rows  # noqa: E402

SURFACE_DIR = MANIFEST_DIR / "surface"
INVENTORY = MANIFEST_DIR / "surface-inventory.tsv"
HEADER = ["c_site", "function", "classification", "case_ids", "notes"]
CLASSES = {"proven", "gap-reachable", "gap-blocked", "unreachable"}
GAPS = {"gap-reachable", "gap-blocked"}
SITE = re.compile(r"^(src/[A-Za-z0-9_.-]+\.c):([1-9][0-9]*)$")
# An output call at the start of an expression: act(, send_to_char(, stc(.
CALL = re.compile(r"(?<![A-Za-z0-9_])(act|send_to_char|stc)\s*\(")
COMMENT = re.compile(r"/\*.*?\*/", re.S)


def output_lines(source: str) -> set[int]:
    """Line numbers of output calls, ignoring calls inside C comments."""
    stripped = COMMENT.sub(lambda m: re.sub(r"[^\n]", " ", m.group(0)), source)
    return {i for i, line in enumerate(stripped.split("\n"), 1) if CALL.search(line)}


def family_for(map_path: pathlib.Path) -> str:
    """surface/spec-procs.tsv -> 'src/spec_procs.c act-send family'."""
    return f"src/{map_path.stem.replace('-', '_')}.c act-send family"


def check(surface_dir=SURFACE_DIR, inventory=INVENTORY, root=ROOT, case_ids=None) -> list[str]:
    errors: list[str] = []
    if case_ids is None:
        case_ids = {row["case_id"] for row in load_rows()}
    families = {}
    if inventory.exists():
        with inventory.open(encoding="utf-8", newline="") as stream:
            for row in csv.DictReader(stream, delimiter="\t"):
                families[row["family"]] = row
    for path in sorted(surface_dir.glob("*.tsv")):
        name = path.relative_to(root) if path.is_relative_to(root) else path
        with path.open(encoding="utf-8", newline="") as stream:
            rows = list(csv.reader(stream, delimiter="\t"))
        if not rows or rows[0] != HEADER:
            errors.append(f"{name}: header must be {HEADER}")
            continue
        seen: dict[str, int] = {}
        gaps = 0
        c_file = None
        for line_no, row in enumerate(rows[1:], 2):
            where = f"{name}:{line_no}"
            if len(row) != len(HEADER):
                errors.append(f"{where}: {len(row)} fields, want {len(HEADER)}")
                continue
            site, _function, kind, cited, _notes = row
            match = SITE.match(site)
            if not match:
                errors.append(f"{where}: c_site {site!r} is not src/<file>.c:<line>")
                continue
            if c_file is None:
                c_file = match[1]
            elif match[1] != c_file:
                errors.append(f"{where}: {site} is not in {c_file}; one map covers one C file")
            if site in seen:
                errors.append(f"{where}: {site} already listed at line {seen[site]}")
            seen[site] = line_no
            if kind not in CLASSES:
                errors.append(f"{where}: classification {kind!r} not in {sorted(CLASSES)}")
            if kind in GAPS:
                gaps += 1
            ids = [part for part in cited.split(";") if part and part != "-"]
            if kind == "proven" and not ids:
                errors.append(f"{where}: a proven site must cite the case_ids that execute it")
            for case_id in ids:
                if case_id not in case_ids:
                    errors.append(f"{where}: case_id {case_id!r} is in no depth manifest")
        if c_file is None:
            continue
        source_path = root / c_file
        if not source_path.exists():
            errors.append(f"{name}: {c_file} does not exist")
            continue
        calls = output_lines(source_path.read_text(encoding="latin-1"))
        listed = {int(site.rsplit(":", 1)[1]) for site in seen}
        for line in sorted(listed - calls):
            errors.append(f"{name}: {c_file}:{line} is not an act/send_to_char/stc call (stale line?)")
        for line in sorted(calls - listed):
            errors.append(f"{name}: {c_file}:{line} is an output call the map does not list")
        family = families.get(family_for(path))
        if family is None:
            errors.append(f"{name}: no surface-inventory row {family_for(path)!r}")
            continue
        if family["units"] != str(gaps):
            errors.append(f"{name}: surface-inventory units {family['units']} != {gaps} gaps in the map")
        if (family["status"] == "proven-already") != (gaps == 0):
            errors.append(f"{name}: inventory status {family['status']!r} disagrees with {gaps} gaps")
    return errors


def self_test() -> None:
    """Every rule must be able to fail (R5h)."""
    with tempfile.TemporaryDirectory() as tmp:
        root = pathlib.Path(tmp)
        (root / "src").mkdir()
        (root / "src/demo.c").write_text("void f() {\n  act(\"a\");\n  /* stc(\"no\"); */\n  send_to_char(\"b\", ch);\n}\n")
        surface = root / "surface"
        surface.mkdir()
        inventory = root / "inventory.tsv"
        good = ["\t".join(HEADER), "src/demo.c:2\tf\tproven\tcase.a\tok", "src/demo.c:4\tf\tgap-reachable\t-\tgap"]

        def run(rows, units="1", status="blocked", ids=frozenset({"case.a"})):
            (surface / "demo.tsv").write_text("\n".join(rows) + "\n")
            inventory.write_text("surface_class\tfamily\tunits\tstatus\tproof_or_attempt\tc_source_scope\tnotes\n"
                                 f"activity-call-sites\tsrc/demo.c act-send family\t{units}\t{status}\t-\t-\t-\n")
            return check(surface, inventory, root, set(ids))

        assert run(good) == [], run(good)
        cases = {
            "header": (["bad"] + good[1:], {}),
            "stale line": (good[:1] + ["src/demo.c:3\tf\tproven\tcase.a\tx"] + good[2:], {}),
            "unlisted call": (good[:2], {"units": "0"}),
            "duplicate": (good + [good[1]], {}),
            "class": ([good[0], good[1].replace("proven", "maybe"), good[2]], {}),
            "proven without ids": ([good[0], good[1].replace("case.a", "-"), good[2]], {}),
            "unknown case": ([good[0], good[1].replace("case.a", "case.zz"), good[2]], {}),
            "units": (good, {"units": "2"}),
            "status": (good, {"status": "proven-already"}),
        }
        for label, (rows, kwargs) in cases.items():
            assert run(rows, **kwargs), f"self-test: {label} was not caught"


def main(argv: list[str]) -> int:
    if "--self-test" in argv:
        self_test()
        print("surface-map: self-test ok")
        return 0
    errors = check()
    for error in errors:
        print(error)
    maps = len(list(SURFACE_DIR.glob("*.tsv"))) if SURFACE_DIR.exists() else 0
    print(f"surface-map: {maps} map(s), {len(errors)} error(s)")
    return 1 if errors else 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
