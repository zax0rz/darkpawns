#!/usr/bin/env python3
"""Generate the Dark Pawns website content inventory.

The CSV is the complete route-level ledger. CONTENT-INVENTORY.md is the
human review queue. Both files are generated from Astro source.
"""

from __future__ import annotations

import csv
import dataclasses
import argparse
import io
import json
import re
from collections import Counter
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SITE = ROOT / "website-astro"
CSV_PATH = SITE / "content-inventory.csv"
REPORT_PATH = SITE / "CONTENT-INVENTORY.md"


@dataclasses.dataclass
class Entry:
    route: str
    source_file: str
    surface: str
    content_mode: str
    review_tier: int
    review_status: str
    words: int | str = ""
    text_kind: str = ""
    voice_layer: str = ""
    source: str = ""


def frontmatter_and_body(path: Path) -> tuple[dict[str, str], str]:
    text = path.read_text(encoding="utf-8")
    if not text.startswith("---\n"):
        return {}, text
    _, raw, body = text.split("---", 2)
    data: dict[str, str] = {}
    for line in raw.splitlines():
        match = re.match(r"^([A-Za-z][A-Za-z0-9]*):\s*[\"']?([^\"']*)", line)
        if match:
            data[match.group(1)] = match.group(2).strip()
    return data, body


def word_count(body: str) -> int:
    body = re.sub(r"```.*?```", " ", body, flags=re.DOTALL)
    body = re.sub(r"<[^>]+>|https?://\S+|[`*_#|>\[\]()]", " ", body)
    return len(re.findall(r"\b[\w'+-]+\b", body))


def content_entries() -> list[Entry]:
    entries: list[Entry] = []
    for path in sorted((SITE / "src/content").glob("**/*.md")):
        relative = path.relative_to(SITE).as_posix()
        collection = path.relative_to(SITE / "src/content").parts[0]
        if path.name == "_index.md":
            continue
        data, body = frontmatter_and_body(path)
        slug = path.relative_to(SITE / "src/content" / collection).with_suffix("").as_posix()
        route = f"/{collection}/{slug}/"

        if collection == "help":
            mode, tier, status = "fidelity-protected reference", 4, "template-and-sample QA"
        elif collection == "archive":
            mode, tier, status = "preserved community artifact", 2, "verify against capture"
        elif collection == "blog":
            mode, tier = "authored editorial", 1
            status = "replace or remove" if path.name == "historical-posts.md" else "rewrite and source review"
        elif collection == "world":
            mode, tier, status = "authored world reference", 2, "source and voice review"
        else:
            mode, tier, status = "authored technical documentation", 3, "technical accuracy review"

        entries.append(
            Entry(
                route,
                relative,
                collection,
                mode,
                tier,
                status,
                word_count(body),
                data.get("textKind", "verbatim" if collection == "help" else ""),
                data.get("voiceLayer", "engine" if collection == "help" else ""),
                data.get("source", data.get("sourcePath", data.get("captureUrl", ""))),
            )
        )
        entries.append(
            Entry(
                route.removesuffix("/") + ".md",
                f"src/pages/{collection}/[...slug].md.ts",
                "agent-markdown",
                "generated Markdown representation",
                3,
                "schema-and-sample QA",
                word_count(body),
                data.get("textKind", "verbatim" if collection == "help" else ""),
                data.get("voiceLayer", "engine" if collection == "help" else ""),
                data.get("source", data.get("sourcePath", data.get("captureUrl", ""))),
            )
        )
    return entries


def structural_entries() -> list[Entry]:
    core = {
        "/": "src/pages/index.astro",
        "/about/": "src/pages/about.astro",
        "/about/dpreturns/": "src/pages/about/dpreturns.astro",
        "/about/project/": "src/pages/about/project.astro",
        "/credits/": "src/pages/credits.astro",
        "/contact/": "src/pages/contact.astro",
        "/play/": "src/pages/play.astro",
        "/map/": "src/pages/map.astro",
        "/database/": "src/pages/database.astro",
    }
    entries = [Entry(route, source, "core", "authored static page", 1, "full editorial review") for route, source in core.items()]

    hubs = {
        "/archive/": "src/pages/archive/index.astro",
        "/blog/": "src/pages/blog/index.astro",
        "/docs/": "src/pages/docs/index.astro",
        "/help/": "src/pages/help/index.astro",
        "/world/": "src/pages/world/index.astro",
    }
    entries.extend(Entry(route, source, "hub", "authored collection framing", 1, "full editorial review") for route, source in hubs.items())

    agent_resources = {
        "/llms.txt": "../website/static/llms.txt",
        "/.well-known/agent-skills/index.json": "../website/static/.well-known/agent-skills/index.json",
        "/.well-known/agent-skills/darkpawns/SKILL.md": "../website/static/.well-known/agent-skills/darkpawns/SKILL.md",
    }
    entries.extend(Entry(route, source, "agent-discovery", "machine-readable discovery", 3, "schema-and-link QA") for route, source in agent_resources.items())

    for section in ("getting-started", "server", "agents", "research"):
        entries.append(Entry(f"/docs/{section}/", "src/pages/docs/[section]/index.astro", "docs-hub", "generated section index", 3, "template review"))
    for category in ("commands", "info", "socials", "spells", "wizhelp"):
        entries.append(Entry(f"/help/{category}/", "src/pages/help/[category]/index.astro", "help-hub", "generated section index", 3, "template review"))
    return entries


def database_entries() -> list[Entry]:
    data_path = ROOT / "website/static/data/database.json"
    data = json.loads(data_path.read_text(encoding="utf-8"))
    entries: list[Entry] = []
    for surface, source_file in (("mobs", "src/pages/mobs/[vnum].astro"), ("items", "src/pages/items/[vnum].astro")):
        entries.append(Entry(f"/{surface}/", f"src/pages/{surface}/index.astro", f"{surface}-index", "generated database index", 3, "template-and-sample QA"))
        for vnum in sorted(data[surface], key=int):
            entries.append(Entry(f"/{surface}/{vnum}/", source_file, surface, "generated world-file record", 3, "template-and-sample QA", text_kind="engine data", voice_layer="engine", source="active world files"))
            entries.append(Entry(f"/{surface}/{vnum}.json", f"src/pages/{surface}/[vnum].json.ts", "agent-json", "generated machine-readable record", 3, "schema-and-sample QA", text_kind="engine data", voice_layer="engine", source="active world files"))
    return entries


def redirect_entries() -> list[Entry]:
    config = (SITE / "astro.config.mjs").read_text(encoding="utf-8")
    matches = re.findall(r"^\s*'(/[^']*)':\s*'([^']+)'", config, flags=re.MULTILINE)
    return [Entry(source, "astro.config.mjs", "redirect", f"redirect to {target}", 5, "no copy review") for source, target in matches]


def render_csv(entries: list[Entry]) -> str:
    fields = [field.name for field in dataclasses.fields(Entry)]
    handle = io.StringIO(newline="")
    writer = csv.DictWriter(handle, fieldnames=fields, lineterminator="\n")
    writer.writeheader()
    writer.writerows(dataclasses.asdict(entry) for entry in entries)
    return handle.getvalue()


def render_report(public: list[Entry], redirects: list[Entry]) -> str:
    surfaces = Counter(entry.surface for entry in public)
    tiers = Counter(entry.review_tier for entry in public)
    words = sum(entry.words for entry in public if isinstance(entry.words, int))
    report = f"""# Content inventory

Generated by `python3 website-astro/scripts/content_inventory.py`. Do not hand-edit the route ledger in `content-inventory.csv`.

## Inventory snapshot

- **{len(public)} built public routes**
- **{len(redirects)} compatibility redirects**
- **{words:,} words in Markdown bodies**
- **{surfaces['help']} fidelity-protected help entries**
- **{surfaces['archive']} recovered community artifacts**
- **{len(public) - surfaces['help'] - surfaces['archive']} authored, generated-index, or interactive routes**

| Surface | Routes | Editorial treatment |
|---|---:|---|
| Help entries | {surfaces['help']} | Preserve source text. Review templates and samples, not prose-rewrite every entry. |
| Archive artifacts | {surfaces['archive']} | Verify against captures. Preserve bodies and review all framing copy. |
| World handbook | {surfaces['world']} | Verify claims, remove generated filler, then apply voice. |
| Technical docs | {surfaces['docs']} | Verify against the repository and intended audience. |
| Blog posts | {surfaces['blog']} | Full source, purpose, and voice review. |
| Core pages | {surfaces['core']} | Full editorial review. These establish the site's identity. |
| Collection hubs | {surfaces['hub']} | Full editorial review. These tell readers what each section is for. |
| Generated section indexes | {surfaces['docs-hub'] + surfaces['help-hub']} | Review the shared templates once, then inspect representative output. |
| Mob and item records | {surfaces['mobs'] + surfaces['items']} | Generated from active world files. Review both templates and representative records. |
| Agent JSON records | {surfaces['agent-json']} | Verify the shared schema, provenance fields, links, and representative output. |
| Agent Markdown records | {surfaces['agent-markdown']} | Verify the shared frontmatter, canonical links, and representative output. |

## Review queue

### Tier 1: identity and trust ({tiers[1]} routes)

Review every word on the home page, About, Credits, Play, Map, Database, the five collection hubs, and all three Blog posts. These pages answer what Dark Pawns is, why it exists, and whether the archive can be trusted.

Immediate decisions:

1. Remove or reconstruct `/blog/historical-posts/`. It is labeled as a summary now, but it remains a placeholder rather than a recovered post.
2. Rework `/about/` with Zach after the inventory findings are assembled. Its facts need citations and its voice is not final.
3. Audit homepage, Archive, and Credits claims against primary sources. Preserve the names and history without turning them into promotional legend.
4. Review Map and Database interface copy separately from their generated data.

### Tier 2: world and archive ({tiers[2]} routes)

Review the ten World pages for invented mechanics, generic fantasy prose, changing statistics, and unsourced claims. Compare each of the nine Archive bodies with its cited capture and review descriptions, date labels, warnings, and attribution as original copy.

### Tier 3: documentation and templates ({tiers[3]} routes)

Review technical documentation for accuracy and audience fit. Review each dynamic collection template and each generated index template once, with representative desktop and mobile pages. Validate the mob, item, and agent JSON templates against the source database and inspect records with empty, ordinary, and unusually dense relationships.

### Tier 4: fidelity-protected reference ({tiers[4]} routes)

Do not rewrite 429 help entries. Validate provenance, rendering, navigation, category assignment, and a sample from commands, info, socials, spells, and wizhelp. The original text remains the oracle.

### Tier 5: redirects ({len(redirects)} routes)

Test destinations and retain compatibility. Redirects have no editorial copy.

## Working rule

Inventory is not approval. A route leaves the queue only after its source, text kind, voice layer, factual claims, and reader purpose have been reviewed.
"""
    return report


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="fail when generated inventory is stale")
    args = parser.parse_args()
    public = sorted(content_entries() + structural_entries() + database_entries(), key=lambda entry: entry.route)
    redirects = sorted(redirect_entries(), key=lambda entry: entry.route)
    routes = [entry.route for entry in public]
    if len(routes) != len(set(routes)):
        raise SystemExit("content-inventory: duplicate public route")
    csv_content = render_csv(public + redirects)
    report_content = render_report(public, redirects)
    if args.check:
        stale = []
        if not CSV_PATH.exists() or CSV_PATH.read_text(encoding="utf-8") != csv_content:
            stale.append(CSV_PATH.name)
        if not REPORT_PATH.exists() or REPORT_PATH.read_text(encoding="utf-8") != report_content:
            stale.append(REPORT_PATH.name)
        if stale:
            raise SystemExit(f"content-inventory: stale generated files: {', '.join(stale)}")
    else:
        CSV_PATH.write_text(csv_content, encoding="utf-8")
        REPORT_PATH.write_text(report_content, encoding="utf-8")
    print(f"content-inventory: {len(public)} public routes, {len(redirects)} redirects")


if __name__ == "__main__":
    main()
