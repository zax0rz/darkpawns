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
import subprocess
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
    # .mdx pages publish the same routes as .md; the ledger must list both.
    paths = sorted(
        path
        for pattern in ("**/*.md", "**/*.mdx")
        for path in (SITE / "src/content").glob(pattern)
    )
    for path in paths:
        relative = path.relative_to(SITE).as_posix()
        collection = path.relative_to(SITE / "src/content").parts[0]
        if path.name == "_index.md":
            continue
        if path.relative_to(SITE / "src/content").parts[0] == "help":
            # Help files remain source data; the generated grouped reference
            # pages below are their only published routes.
            continue
        data, body = frontmatter_and_body(path)
        slug = path.relative_to(SITE / "src/content" / collection).with_suffix("").as_posix()
        route = f"/{collection}/{slug}/"
        if collection == "docs" and (slug == "index" or slug.endswith("/index")):
            route = "/docs/" if slug == "index" else f"/docs/{slug[:-6]}/"

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
                ("scripts/markdown-twins.mjs" if collection == "docs" else f"src/pages/{collection}/[...slug].md.ts"),
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


def generated_docs_entries() -> list[Entry]:
    """Inventory release and command docs from the files the generator reads."""
    entries: list[Entry] = []
    release_root = ROOT / "docs/releases"
    release_files = sorted(
        (path for path in release_root.iterdir() if re.fullmatch(r"v\d+\.\d+\.\d+\.md", path.name)),
        key=lambda path: tuple(int(part) for part in path.stem[1:].split(".")),
        reverse=True,
    )
    release_body_lines = []
    for path in release_files:
        version = path.stem
        slug = version.replace(".", "-")
        body = path.read_text(encoding="utf-8").strip()
        source = path.relative_to(ROOT).as_posix()
        route = f"/docs/releases/{slug}/"
        entries.extend(_generated_doc_pair(
            route,
            source,
            "generated release reference",
            body,
            source,
        ))
        release_body_lines.append(f"- [{version}](/docs/releases/{slug}/)")

    release_index_body = "\n".join(release_body_lines)
    entries.extend(_generated_doc_pair(
        "/docs/releases/",
        "docs/releases/",
        "generated release index",
        release_index_body,
        "docs/releases/",
    ))

    helper = SITE / "scripts/command-taxonomy.mjs"
    data = json.loads(subprocess.run(
        ["node", str(helper)], cwd=SITE, check=True, capture_output=True, text=True,
    ).stdout)
    group_links = []
    for group in data["groups"]:
        topics = group["topics"]
        source = "website-astro/src/content/help/"
        body = "\n".join(f"- {topic['title']}" for topic in topics)
        group_links.append(f"- [{group['title']}](/docs/reference/commands/{group['slug']}/)")
        entries.extend(_generated_doc_pair(
            f"/docs/reference/commands/{group['slug']}/",
            source,
            "generated grouped help reference",
            body,
            source,
            text_kind="verbatim",
            voice_layer="engine",
        ))

    entries.extend(_generated_doc_pair(
        "/docs/reference/commands/",
        "lib/text/help/commands.hlp",
        "generated grouped command index",
        "\n".join(group_links),
        "lib/text/help",
        text_kind="verbatim",
        voice_layer="engine",
    ))
    return entries


def _generated_doc_pair(
    route: str,
    source_file: str,
    content_mode: str,
    body: str,
    source: str,
    *,
    text_kind: str = "",
    voice_layer: str = "",
) -> list[Entry]:
    """Return a generated docs route and its Markdown twin ledger row."""
    return [
        Entry(
            route,
            source_file,
            "docs",
            content_mode,
            3,
            "verify generated source and representative output",
            word_count(body),
            text_kind,
            voice_layer,
            source,
        ),
        Entry(
            route.removesuffix("/") + ".md",
            "scripts/markdown-twins.mjs",
            "agent-markdown",
            "generated Markdown representation",
            3,
            "schema-and-sample QA",
            word_count(body),
            text_kind,
            voice_layer,
            source,
        ),
    ]


def api_entries() -> list[Entry]:
    """Inventory OpenAPI pages from the Go exporter, without generated files."""
    result = subprocess.run(
        ["go", "run", "./cmd/admin-openapi"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
        timeout=120,
    )
    spec = json.loads(result.stdout)
    entries: list[Entry] = []
    methods = {"get", "put", "post", "delete", "options", "head", "patch", "trace"}
    for kind, paths in (
        ("public", {path: item for path, item in spec.get("paths", {}).items() if not path.startswith("/admin")}),
        ("admin", {path: item for path, item in spec.get("paths", {}).items() if path.startswith("/admin")}),
    ):
        base = f"/docs/reference/api/{kind}"
        source_file = "cmd/admin-openapi/main.go"
        pointer = "cmd/admin-openapi/main.go#/OpenAPI"
        entries.extend(_generated_doc_pair(
            base + "/",
            source_file,
            "generated OpenAPI reference index",
            "",
            pointer,
        ))
        for path, path_item in paths.items():
            for method, operation in path_item.items():
                method = method.lower()
                if method not in methods or not isinstance(operation, dict):
                    continue
                operation_id = operation.get("operationId")
                if not operation_id:
                    continue
                pointer = f"cmd/admin-openapi/main.go#/paths/{path}/{method}"
                entries.extend(_generated_doc_pair(
                    f"{base}/operations/{operation_id}/",
                    source_file,
                    "generated OpenAPI operation reference",
                    "",
                    pointer,
                ))
    return entries


def structural_entries() -> list[Entry]:
    core = {
        "/": "src/pages/index.astro",
        "/about/": "src/pages/about.astro",
        "/about/dpreturns/": "src/pages/about/dpreturns.astro",
        "/about/project/": "src/pages/about/project.astro",
        "/credits/": "src/pages/credits.astro",
        "/contact/": "src/pages/contact.astro",
        "/privacy/": "src/pages/privacy.astro",
        "/play/": "src/pages/play.astro",
        "/download/": "src/pages/download.astro",
        "/map/": "src/pages/map.astro",
        "/database/": "src/pages/database.astro",
    }
    entries = [Entry(route, source, "core", "authored static page", 1, "full editorial review") for route, source in core.items()]

    hubs = {
        "/archive/": "src/pages/archive/index.astro",
        "/blog/": "src/pages/blog/index.astro",
        "/world/": "src/pages/world/index.astro",
    }
    entries.extend(Entry(route, source, "hub", "authored collection framing", 1, "full editorial review") for route, source in hubs.items())

    entries.append(Entry("/download.md", "src/pages/download.md.ts", "agent-markdown", "generated Markdown representation", 3, "schema-and-sample QA"))

    agent_resources = {
        "/llms.txt": "src/pages/llms.txt.ts",
        "/llms-full.txt": "scripts/markdown-twins.mjs",
        "/.well-known/agent-skills/index.json": "../website/static/.well-known/agent-skills/index.json",
        "/.well-known/agent-skills/darkpawns/SKILL.md": "../website/static/.well-known/agent-skills/darkpawns/SKILL.md",
        "/.well-known/security.txt": "../website/static/.well-known/security.txt",
        "/rss.xml": "src/pages/rss.xml.ts",
    }
    entries.extend(Entry(route, source, "agent-discovery", "machine-readable discovery", 3, "schema-and-link QA") for route, source in agent_resources.items())

    return entries


def database_entries() -> list[Entry]:
    data_path = ROOT / "website/static/data/database.json"
    data = json.loads(data_path.read_text(encoding="utf-8"))

    def zone_room_ids(zone_id: int, _world_rooms: dict[int, list[int]] | None = None) -> list[int]:
        return (_world_rooms or {}).get(zone_id, [])

    world = json.loads((ROOT / "website/static/map/world.json").read_text(encoding="utf-8"))
    world_rooms = {zone["id"]: sorted(room["id"] for room in zone["rooms"]) for zone in world["zones"]}

    entries: list[Entry] = []
    for surface, source_file in (("mobs", "src/pages/mobs/[vnum].astro"), ("items", "src/pages/items/[vnum].astro")):
        entries.append(Entry(f"/{surface}/", f"src/pages/{surface}/index.astro", f"{surface}-index", "generated database index", 3, "template-and-sample QA"))
        for vnum in sorted(data[surface], key=int):
            entries.append(Entry(f"/{surface}/{vnum}/", source_file, surface, "generated world-file record", 3, "template-and-sample QA", text_kind="engine data", voice_layer="engine", source="active world files"))
            entries.append(Entry(f"/{surface}/{vnum}.json", f"src/pages/{surface}/[vnum].json.ts", "agent-json", "generated machine-readable record", 3, "schema-and-sample QA", text_kind="engine data", voice_layer="engine", source="active world files"))
    map_index = json.loads((ROOT / "website/static/map/map-index.json").read_text(encoding="utf-8"))
    entries.append(Entry("/zones/", "src/pages/zones/index.astro", "zone-index", "generated world-file index", 3, "template-and-sample QA"))
    for zone in map_index["zones"]:
        entries.append(Entry(f"/zones/{zone['id']}/", "src/pages/zones/[id].astro", "zone", "generated world-file record", 3, "template-and-sample QA", text_kind="engine data", voice_layer="engine", source="active world files"))
        entries.append(Entry(f"/zones/{zone['id']}.json", "src/pages/zones/[id].json.ts", "agent-json", "generated machine-readable record", 3, "schema-and-sample QA", text_kind="engine data", voice_layer="engine", source="active world files"))
        for room in zone_room_ids(zone["id"], world_rooms):
            entries.append(Entry(f"/rooms/{room}/", "src/pages/rooms/[vnum].astro", "room", "generated world-file record", 3, "template-and-sample QA", text_kind="engine data", voice_layer="engine", source="active world files"))
            entries.append(Entry(f"/rooms/{room}.json", "src/pages/rooms/[vnum].json.ts", "agent-json", "generated machine-readable record", 3, "schema-and-sample QA", text_kind="engine data", voice_layer="engine", source="active world files"))
    return entries


def redirect_entries() -> list[Entry]:
    config = (SITE / "astro.config.mjs").read_text(encoding="utf-8")
    matches = re.findall(r"^\s*'(/[^']*)':\s*'([^']+)'", config, flags=re.MULTILINE)
    redirects = json.loads((SITE / "scripts/docs-redirects.json").read_text(encoding="utf-8"))
    redirects.update({source: target for source, target in matches})
    helper = SITE / "scripts/command-taxonomy.mjs"
    generated = json.loads(subprocess.run(
        ["node", str(helper)], cwd=SITE, check=True, capture_output=True, text=True,
    ).stdout)
    redirects.update(generated["redirects"])
    return [
        Entry(source, "astro.config.mjs", "redirect", f"redirect to {target}", 5, "no copy review")
        for source, target in redirects.items()
    ]


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

Review the ten World pages for invented mechanics, generic fantasy prose, changing statistics, and unsourced claims. Compare each of the {surfaces['archive']} Archive bodies with its cited capture and review descriptions, date labels, warnings, and attribution as original copy.

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
    public = sorted(content_entries() + generated_docs_entries() + api_entries() + structural_entries() + database_entries(), key=lambda entry: entry.route)
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
