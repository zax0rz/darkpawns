"""Shared, validated fidelity manifest loader; no proof evaluation."""
from __future__ import annotations

import csv
import pathlib

ROOT = pathlib.Path(__file__).resolve().parents[1]
MANIFEST_DIR = ROOT / "docs" / "fidelity" / "depth"
FIELDS = ("handler", "command", "case_id", "depth", "scope", "status", "proof", "c_site", "notes")
# stability is an optional trailing column: "" (default) or "run-varying",
# consumed by gen_expected_divergences.py to mark C-side divergences whose
# bytes differ every run (classified EXPECTED_UNSTABLE by the census).
OPTIONAL_FIELDS = ("stability",)
VALID_STATUSES = {
    "oracle-green",
    "oracle-green-multiseed",
    "unit-green",
    "blocked",
    "excluded",
    "delegated",
    # A deliberate departure from C that Zach approved (security, durability).
    # Proven by unit tests of the approved behaviour; the notes cite the issue.
    # Not C parity, so the depth report counts it apart from completion.
    "divergent-approved",
}
# Statuses whose proof field names Go test symbols that must exist and pass.
UNIT_PROOF_STATUSES = {"unit-green", "divergent-approved"}

def load_rows(manifest_dir: pathlib.Path = MANIFEST_DIR, root: pathlib.Path = ROOT) -> list[dict[str, str]]:
    rows: list[dict[str, str]] = []
    for path in sorted(manifest_dir.glob("*.tsv")):
        # The surface inventory is a separate weighted denominator, not a
        # depth-case manifest. Its schema and status vocabulary intentionally
        # differ from the per-case files consumed below.
        if path.name == "surface-inventory.tsv":
            continue
        with path.open(encoding="utf-8", newline="") as stream:
            reader = csv.DictReader(stream, delimiter="\t")
            if tuple(reader.fieldnames or ()) not in (FIELDS, FIELDS + OPTIONAL_FIELDS):
                raise ValueError(f"{path}: fields {reader.fieldnames!r}, want {FIELDS!r}")
            for line_no, row in enumerate(reader, 2):
                if None in row or any(row.get(field) is None for field in FIELDS):
                    raise ValueError(f"{path}:{line_no}: wrong number of tab-separated fields")
                row = {key: (value or "").strip() for key, value in row.items()}
                if not row["case_id"]:
                    raise ValueError(f"{path}:{line_no}: empty case_id")
                if row["status"] not in VALID_STATUSES:
                    raise ValueError(f"{path}:{line_no}: invalid status {row['status']!r}")
                row["line"] = str(line_no)
                row["manifest"] = str(path.relative_to(root))
                rows.append(row)
    return rows

