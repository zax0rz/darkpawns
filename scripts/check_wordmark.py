#!/usr/bin/env python3
"""Enforce The Wordmark Lockup Rule across every surface that draws the mark.

The mark is drawn in four runtimes that cannot share a component: Astro
(website-astro), React (admin-ui), static HTML (web/public) and a standalone
favicon. Nothing but a check keeps four independent copies identical, and
without one they drift: by 2026-09-16 there were three different pawn drawings
and two different lockups in the tree at the same time.

This asserts two things:

  1. Every pawn is the same five shapes at the same coordinates. Scale and
     translate are free; a redraw is not. A simplified pawn is a different
     mark, not the same mark smaller.
  2. Wherever the lettering appears, it is "DARK" over "PAWNS" with oxblood on
     PAWNS alone, never one line and never a single colour.

Run via `make check-wordmark`, or as part of `make site-check`.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

# The canonical pawn, from website-astro/src/components/Header.astro. Ordered,
# because a reordered drawing is still a different file to review.
CANONICAL_SHAPES = [
    ("circle", "cx=50 cy=23 r=12"),
    ("rect", "x=37 y=38 width=26 height=5"),
    ("polygon", "points=43,46 57,46 62,75 38,75"),
    ("rect", "x=31 y=77 width=38 height=6"),
    ("rect", "x=26 y=85 width=48 height=7"),
]

# Files that draw the pawn. Each must reproduce CANONICAL_SHAPES exactly.
PAWN_SURFACES = [
    "website-astro/src/components/Header.astro",
    "web/public/index.html",
    "web/public/static/pawn.svg",
    "admin-ui/src/components/Wordmark.tsx",
    "admin-ui/src/components/Icon.tsx",
]

# Files that set the lettering. Each must split DARK / PAWNS and accent PAWNS.
LOCKUP_SURFACES = [
    "website-astro/src/components/Header.astro",
    "web/public/index.html",
    "admin-ui/src/components/Wordmark.tsx",
]

ATTR = re.compile(r'([a-zA-Z-]+)\s*=\s*"([^"]*)"')


def shapes_in(text: str) -> list[tuple[str, str]]:
    """Extract pawn shapes in document order, normalised for comparison."""
    found: list[tuple[str, str]] = []
    for match in re.finditer(r"<(circle|rect|polygon)\b([^>]*)>", text):
        tag, raw = match.group(1), match.group(2)
        attrs = dict(ATTR.findall(raw))
        if tag == "circle":
            keys = ("cx", "cy", "r")
        elif tag == "rect":
            keys = ("x", "y", "width", "height")
        else:
            keys = ("points",)
        if not all(k in attrs for k in keys):
            continue  # a shape used for something else, not part of the mark
        found.append((tag, " ".join(f"{k}={attrs[k].strip()}" for k in keys)))
    return found


def check_pawns() -> list[str]:
    problems: list[str] = []
    for rel in PAWN_SURFACES:
        path = ROOT / rel
        if not path.exists():
            problems.append(f"{rel}: listed as a pawn surface but missing")
            continue
        found = shapes_in(path.read_text(encoding="utf-8"))
        # A file may contain other SVGs; the mark is the canonical run within it.
        window = len(CANONICAL_SHAPES)
        runs = [found[i : i + window] for i in range(max(0, len(found) - window + 1))]
        if CANONICAL_SHAPES not in runs:
            problems.append(
                f"{rel}: pawn geometry is not canonical. "
                f"Expected {CANONICAL_SHAPES}, found {found or 'no shapes'}. "
                f"Scale and translate are allowed; redrawing is not."
            )
    return problems


def check_lockups() -> list[str]:
    problems: list[str] = []
    for rel in LOCKUP_SURFACES:
        path = ROOT / rel
        if not path.exists():
            problems.append(f"{rel}: listed as a lockup surface but missing")
            continue
        text = path.read_text(encoding="utf-8")
        flat = re.sub(r"\s+", " ", text)
        # DARK and PAWNS must be separated by a break, not sitting on one line.
        if not re.search(r"DARK\s*<br\s*/?>", flat):
            problems.append(f"{rel}: DARK and PAWNS are not stacked (no <br> between them)")
        # PAWNS, and only PAWNS, carries the accent.
        if not re.search(r'(class|className)="accent"[^>]*>\s*PAWNS|accent">PAWNS', flat):
            problems.append(f"{rel}: PAWNS does not carry the accent class")
        if re.search(r'(class|className)="[^"]*accent[^"]*"[^>]*>\s*DARK', flat):
            problems.append(f"{rel}: DARK is accented; the accent belongs to PAWNS alone")
        # The React lockup can be dropped into a centred container, where the
        # two lines would centre against each other instead of sharing a left
        # edge. The other surfaces sit in left-aligned contexts by construction.
        if rel.endswith(".tsx") and "text-left" not in flat:
            problems.append(
                f"{rel}: lettering does not pin text-left, so a centred parent "
                f"will centre DARK against PAWNS instead of sharing a left edge"
            )
    return problems


def main() -> int:
    problems = check_pawns() + check_lockups()
    if problems:
        print("check-wordmark: The Wordmark Lockup Rule is broken\n", file=sys.stderr)
        for p in problems:
            print(f"  {p}", file=sys.stderr)
        print(
            "\nThe rule and the canonical drawing live in website-astro/DESIGN.md.",
            file=sys.stderr,
        )
        return 1
    print(
        f"check-wordmark: clean "
        f"({len(PAWN_SURFACES)} pawn surfaces, {len(LOCKUP_SURFACES)} lockups)"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
