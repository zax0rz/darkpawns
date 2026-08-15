#!/usr/bin/env python3
"""Verify that every generated Hugo HTML route survives in Astro."""

from __future__ import annotations

import argparse
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]


def html_routes(root: Path) -> set[str]:
    routes: set[str] = set()
    if not root.is_dir():
        raise SystemExit(f"route-parity: missing build directory: {root}")
    for path in root.rglob("*.html"):
        relative = path.relative_to(root).as_posix()
        if relative == "index.html":
            route = "/"
        elif relative.endswith("/index.html"):
            route = f"/{relative[:-10]}"
        else:
            route = f"/{relative}"
        routes.add(route)
    return routes


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--hugo-dir", type=Path, default=ROOT / "website/public")
    parser.add_argument("--astro-dir", type=Path, default=ROOT / "website-astro/dist")
    args = parser.parse_args()

    hugo = html_routes(args.hugo_dir)
    astro = html_routes(args.astro_dir)
    missing = sorted(hugo - astro)
    if missing:
        print(f"route-parity: {len(missing)} Hugo route(s) missing from Astro")
        for route in missing:
            print(route)
        raise SystemExit(1)

    print(
        f"route-parity: clean ({len(hugo)} Hugo routes covered; "
        f"{len(astro - hugo)} Astro-only routes)"
    )


if __name__ == "__main__":
    main()
