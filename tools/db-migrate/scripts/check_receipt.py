#!/usr/bin/env python3
"""Check a conversion or verification receipt without trusting the exit status.

The converter writes a JSON receipt (`--report`). This script asserts the facts an
operator actually cares about, so a workflow or a cutover checklist can gate on
the receipt rather than on "the command exited 0".

Usage:
    python3 scripts/check_receipt.py <receipt.json> [--expect-mode convert|verify-only]
                                      [--expect-rows TABLE=N ...]

Exit status is 0 only when every assertion holds. Never prints a credential or a
player payload: receipts carry counts, column names, digests and timings.
"""

from __future__ import annotations

import argparse
import json
import sys

FAILURES: list[str] = []


def check(condition: bool, message: str) -> None:
    if not condition:
        FAILURES.append(message)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("receipt", help="path to a JSON receipt written by --report")
    parser.add_argument("--expect-mode", choices=["convert", "verify-only"])
    parser.add_argument(
        "--expect-rows",
        nargs="*",
        default=[],
        metavar="TABLE=N",
        help="exact row count per table in the verification record",
    )
    args = parser.parse_args()

    with open(args.receipt, encoding="utf-8") as handle:
        receipt = json.load(handle)

    check(receipt.get("tool") == "dp-db-migrate", f"tool is {receipt.get('tool')!r}")
    check(receipt.get("ok") is True, f"ok is {receipt.get('ok')!r}: {receipt.get('failure')}")
    if args.expect_mode:
        check(
            receipt.get("mode") == args.expect_mode,
            f"mode is {receipt.get('mode')!r}, expected {args.expect_mode!r}",
        )

    destination = receipt.get("destination", {})
    if receipt.get("mode") == "convert":
        check(destination.get("installed") is True, "destination was not installed")
        check(
            destination.get("durability_uncertain") is not True,
            "destination durability is uncertain (the directory entry was not synced)",
        )
    else:
        check(destination.get("installed") is not True, "verify-only claims it installed the destination")
    check(destination.get("integrity") == "ok" or receipt.get("mode") == "verify-only",
          f"sqlite integrity_check reported {destination.get('integrity')!r}")

    verification = receipt.get("verification") or {}
    tables = verification.get("tables") or []
    check(bool(tables), "the receipt carries no verification")
    wanted_tables = {"players", "abuse_reports", "admin_log", "player_penalties", "word_filters"}
    seen = {entry.get("table") for entry in tables}
    check(seen == wanted_tables, f"verified tables are {sorted(seen)}")

    for entry in tables:
        table = entry.get("table")
        check(entry.get("ok") is True, f"{table}: not ok")
        check(entry.get("rows_match") is True, f"{table}: row counts differ")
        check(entry.get("primary_key_matches") is True, f"{table}: primary keys differ")
        check(entry.get("content_matches") is True, f"{table}: content digests differ")
        check(entry.get("null_shape_matches") is True, f"{table}: null shape differs")
        check(entry.get("max_id_matches") is True, f"{table}: id watermark differs")
        check(
            not entry.get("mismatched_columns"),
            f"{table}: columns differ {entry.get('mismatched_columns')}",
        )
        check(
            not entry.get("unexpected_extra_columns"),
            f"{table}: source columns with no destination and no proof "
            f"{entry.get('unexpected_extra_columns')}",
        )

    check(verification.get("unique_folded_names_hold") is True, "folded-name uniqueness does not hold")

    for expectation in args.expect_rows:
        table, _, count = expectation.partition("=")
        match = next((entry for entry in tables if entry.get("table") == table), None)
        check(match is not None, f"--expect-rows names unknown table {table!r}")
        if match is not None:
            check(
                match.get("source_rows") == int(count),
                f"{table}: source rows {match.get('source_rows')} != {count}",
            )
            check(
                match.get("destination_rows") == int(count),
                f"{table}: destination rows {match.get('destination_rows')} != {count}",
            )

    if FAILURES:
        for failure in FAILURES:
            print(f"FAIL: {failure}", file=sys.stderr)
        return 1
    print(
        "receipt ok:"
        f" mode={receipt.get('mode')}"
        f" tables={len(tables)}"
        f" duration_ms={receipt.get('duration_ms')}"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
