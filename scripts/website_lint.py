#!/usr/bin/env python3
"""
website_lint.py — Link checker and content validator for the Dark Pawns Hugo site.

Usage:
    python3 scripts/website_lint.py [--build] [--external] [--output DIR]

Flags:
    --build       Run Hugo build before checking (default: use existing public/)
    --external    Check external links (slow, rate-limited)
    --output DIR  Output directory for reports (default: docs/reports/website)

Output: docs/reports/website/YYYY-MM-DD-links.md
"""

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import time
from collections import defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import urljoin, urlparse

try:
    import requests
    HAS_REQUESTS = True
except ImportError:
    HAS_REQUESTS = False

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

SITE_BASE = "https://darkpawns.labz0rz.com"
IGNORE_EXTERNAL_PATTERNS = [
    r"github\.com/zax0rz/darkpawns",  # repo links may 404 if private
    r"discord\.gg",                     # invite links expire
    r"linear\.app",                     # auth-gated
]
CONCURRENCY = 20
REQUEST_TIMEOUT = 10
EXTERNAL_DELAY = 0.5  # seconds between external requests
CACHE_DIR = Path(__file__).parent.parent / ".link-cache"

# ---------------------------------------------------------------------------
# HTML link extractor
# ---------------------------------------------------------------------------

class LinkExtractor(HTMLParser):
    """Extract links, images, and anchors from HTML."""

    def __init__(self):
        super().__init__()
        self.links = []       # (href, line, context)
        self.images = []      # (src, alt, line)
        self.anchors = set()  # id attributes
        self._tag_stack = []

    def handle_starttag(self, tag, attrs):
        attrs_dict = dict(attrs)
        line = self.getpos()[0]

        if tag == "a" and "href" in attrs_dict:
            href = attrs_dict["href"]
            self.links.append((href, line, ""))

        if tag == "img" and "src" in attrs_dict:
            src = attrs_dict["src"]
            alt = attrs_dict.get("alt", "")
            self.images.append((src, alt, line))

        if "id" in attrs_dict:
            self.anchors.add(attrs_dict["id"])

        # Also collect anchor targets from name attribute
        if tag == "a" and "name" in attrs_dict:
            self.anchors.add(attrs_dict["name"])

    def handle_data(self, data):
        pass

# ---------------------------------------------------------------------------
# Link checker
# ---------------------------------------------------------------------------

def find_html_files(public_dir):
    """Find all HTML files in the Hugo output directory."""
    public = Path(public_dir)
    if not public.exists():
        print(f"Error: {public_dir} does not exist. Run with --build or run 'hugo' first.", file=sys.stderr)
        sys.exit(1)
    return sorted(public.rglob("*.html"))


def extract_links_from_file(html_path, public_dir):
    """Extract all links, images, and anchors from an HTML file."""
    try:
        content = html_path.read_text(encoding="utf-8", errors="replace")
    except Exception as e:
        return None, None, None, str(e)

    parser = LinkExtractor()
    try:
        parser.feed(content)
    except Exception as e:
        return None, None, None, f"HTML parse error: {e}"

    # Make links relative to site root
    rel_path = html_path.relative_to(public_dir)
    base_url = str(rel_path.parent) + "/" if rel_path.parent != Path(".") else ""

    public_root = Path(public_dir)

    resolved_links = []
    for href, line, ctx in parser.links:
        # Skip external links, mailto, telnet, javascript
        if href.startswith(("http://", "https://")):
            resolved_links.append((href, line, ctx, "external"))
        elif href.startswith(("mailto:", "tel:", "javascript:")):
            continue
        elif href.startswith("#"):
            # Anchor-only link — skip (would need full page parse to validate)
            continue
        elif href.startswith("/"):
            # Absolute path — strip query string and anchor
            path_part = href.split("?")[0].split("#")[0]
            if not path_part:
                continue
            target = (public_root / path_part.lstrip("/")).resolve()
            # Check with index.html appended (directory links)
            if not target.exists() and not target.suffix:
                target_dir = (public_root / path_part.lstrip("/") / "index.html").resolve()
                if target_dir.exists():
                    target = target_dir
            resolved_links.append((str(target), line, ctx, "internal"))
        else:
            # Relative link — strip query string and anchor
            path_part = href.split("?")[0].split("#")[0]
            if not path_part:
                continue
            target = (html_path.parent / path_part).resolve()
            resolved_links.append((str(target), line, ctx, "internal"))

    resolved_images = []
    for src, alt, line in parser.images:
        if src.startswith(("http://", "https://")):
            resolved_images.append((src, alt, line, "external"))
        elif src.startswith("/"):
            # Absolute path — resolve relative to site root
            target = (public_root / src.lstrip("/")).resolve()
            resolved_images.append((str(target), alt, line, "internal"))
        else:
            target = (html_path.parent / src).resolve()
            resolved_images.append((str(target), alt, line, "internal"))

    return resolved_links, resolved_images, parser.anchors, None


def check_internal_links(files, public_dir):
    """Check all internal links resolve to existing files."""
    broken = []
    redirects = []
    all_anchors = {}  # file_path -> set of anchor ids

    for html_path in files:
        links, images, anchors, err = extract_links_from_file(html_path, public_dir)
        if err:
            broken.append({"file": str(html_path.relative_to(public_dir)), "error": err})
            continue

        all_anchors[str(html_path)] = anchors

        for target, line, ctx, link_type in (links or []):
            if link_type == "internal":
                if not Path(target).exists():
                    broken.append({
                        "file": str(html_path.relative_to(public_dir)),
                        "line": line,
                        "link": target,
                        "type": "link",
                    })

        for src, alt, line, img_type in (images or []):
            if img_type == "internal":
                if not Path(src).exists():
                    broken.append({
                        "file": str(html_path.relative_to(public_dir)),
                        "line": line,
                        "link": src,
                        "type": "image",
                    })
                if not alt:
                    broken.append({
                        "file": str(html_path.relative_to(public_dir)),
                        "line": line,
                        "link": src,
                        "type": "missing-alt",
                    })

    return broken, redirects


def check_external_links(files, public_dir):
    """Check external links (rate-limited, cached)."""
    if not HAS_REQUESTS:
        print("Warning: 'requests' not installed, skipping external link check", file=sys.stderr)
        return []

    CACHE_DIR.mkdir(exist_ok=True)
    cache_file = CACHE_DIR / "external_links.json"

    # Load cache
    cache = {}
    if cache_file.exists():
        try:
            cache = json.loads(cache_file.read_text())
        except Exception:
            cache = {}

    external_urls = set()
    url_to_files = defaultdict(list)

    for html_path in files:
        links, _, _, _ = extract_links_from_file(html_path, public_dir)
        if not links:
            continue
        for href, line, ctx, link_type in links:
            if link_type == "external":
                # Skip patterns
                skip = False
                for pattern in IGNORE_EXTERNAL_PATTERNS:
                    if re.search(pattern, href):
                        skip = True
                        break
                if skip:
                    continue
                external_urls.add(href)
                url_to_files[href].append(str(html_path.relative_to(public_dir)))

    broken = []
    checked = 0

    for url in sorted(external_urls):
        # Check cache (valid for 7 days)
        if url in cache:
            entry = cache[url]
            if time.time() - entry.get("timestamp", 0) < 7 * 86400:
                if entry.get("status") >= 400:
                    broken.append({
                        "file": url_to_files[url][0],
                        "link": url,
                        "status": entry["status"],
                        "type": "external",
                    })
                checked += 1
                continue

        try:
            resp = requests.head(url, timeout=REQUEST_TIMEOUT, allow_redirects=True,
                                 headers={"User-Agent": "DarkPawns-LinkChecker/1.0"})
            status = resp.status_code
        except requests.RequestException:
            try:
                resp = requests.get(url, timeout=REQUEST_TIMEOUT, allow_redirects=True,
                                    headers={"User-Agent": "DarkPawns-LinkChecker/1.0"})
                status = resp.status_code
            except requests.RequestException:
                status = 0

        cache[url] = {"status": status, "timestamp": time.time()}
        checked += 1

        if status >= 400 or status == 0:
            broken.append({
                "file": url_to_files[url][0],
                "link": url,
                "status": status,
                "type": "external",
            })

        time.sleep(EXTERNAL_DELAY)

        # Save cache periodically
        if checked % 50 == 0:
            cache_file.write_text(json.dumps(cache, indent=2))

    # Final cache save
    cache_file.write_text(json.dumps(cache, indent=2))

    return broken


def run_hugo_build(website_dir):
    """Run Hugo build and return (success, output)."""
    result = subprocess.run(
        ["hugo", "--minify", "--gc"],
        cwd=website_dir,
        capture_output=True,
        text=True,
        timeout=120,
    )
    return result.returncode == 0, result.stdout + result.stderr


# ---------------------------------------------------------------------------
# Report generator
# ---------------------------------------------------------------------------

def generate_report(broken_internal, broken_external, build_output, build_ok):
    """Generate markdown report."""
    lines = []
    lines.append("# Website Lint Report")
    lines.append("")
    lines.append(f"**Date:** {time.strftime('%Y-%m-%d %H:%M')}")
    lines.append(f"**Site:** {SITE_BASE}")
    lines.append("")

    # Summary
    total_internal = len([b for b in broken_internal if b.get("type") != "missing-alt"])
    total_alt = len([b for b in broken_internal if b.get("type") == "missing-alt"])
    total_external = len(broken_external)

    if total_internal == 0 and total_external == 0 and (build_ok or not build_output):
        lines.append("## ✅ All Clear")
        lines.append("")
        lines.append("No broken links found.")
    else:
        lines.append("## Summary")
        lines.append("")
        if total_internal > 0:
            lines.append(f"- **{total_internal}** broken internal links")
        if total_alt > 0:
            lines.append(f"- **{total_alt}** images missing alt text")
        if total_external > 0:
            lines.append(f"- **{total_external}** broken external links")
        if not build_ok:
            lines.append(f"- **Hugo build failed**")
        lines.append("")

    # Hugo build output
    if build_output:
        lines.append("## Hugo Build")
        lines.append("")
        if build_ok:
            lines.append("✅ Build succeeded")
        else:
            lines.append("❌ Build failed")
        lines.append("")
        lines.append("```")
        # Truncate to last 50 lines
        output_lines = build_output.strip().split("\n")
        if len(output_lines) > 50:
            lines.append(f"... ({len(output_lines) - 50} lines omitted)")
            output_lines = output_lines[-50:]
        lines.extend(output_lines)
        lines.append("```")
        lines.append("")

    # Broken internal links
    if total_internal > 0:
        lines.append("## Broken Internal Links")
        lines.append("")
        lines.append("| File | Line | Link | Type |")
        lines.append("|------|------|------|------|")
        for b in sorted(broken_internal, key=lambda x: (x.get("file", ""), x.get("line", 0))):
            if b.get("type") == "missing-alt":
                continue
            f = b.get("file", "?")
            ln = b.get("line", "?")
            link = b.get("link", b.get("error", "?"))
            t = b.get("type", "link")
            lines.append(f"| `{f}` | {ln} | `{link}` | {t} |")
        lines.append("")

    # Missing alt text
    if total_alt > 0:
        lines.append("## Missing Alt Text")
        lines.append("")
        lines.append("| File | Line | Image |")
        lines.append("|------|------|-------|")
        for b in sorted(broken_internal, key=lambda x: (x.get("file", ""), x.get("line", 0))):
            if b.get("type") != "missing-alt":
                continue
            f = b.get("file", "?")
            ln = b.get("line", "?")
            img = b.get("link", "?")
            lines.append(f"| `{f}` | {ln} | `{img}` |")
        lines.append("")

    # Broken external links
    if total_external > 0:
        lines.append("## Broken External Links")
        lines.append("")
        lines.append("| File | Status | URL |")
        lines.append("|------|--------|-----|")
        for b in sorted(broken_external, key=lambda x: x.get("link", "")):
            f = b.get("file", "?")
            status = b.get("status", "?")
            url = b.get("link", "?")
            lines.append(f"| `{f}` | {status} | {url} |")
        lines.append("")

    return "\n".join(lines)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Website link checker and content validator")
    parser.add_argument("--build", action="store_true", help="Run Hugo build before checking")
    parser.add_argument("--external", action="store_true", help="Check external links (slow)")
    parser.add_argument("--output", default=None, help="Output directory for reports")
    parser.add_argument("--website-dir", default=None, help="Path to website/ directory")
    args = parser.parse_args()

    # Find repo root
    script_dir = Path(__file__).parent
    repo_root = script_dir.parent
    website_dir = Path(args.website_dir) if args.website_dir else repo_root / "website"
    public_dir = website_dir / "public"
    output_dir = Path(args.output) if args.output else repo_root / "docs" / "reports" / "website"

    if not website_dir.exists():
        print(f"Error: {website_dir} does not exist", file=sys.stderr)
        sys.exit(1)

    # Optionally run Hugo build
    build_output = ""
    build_ok = True
    if args.build:
        print("Running Hugo build...")
        build_ok, build_output = run_hugo_build(website_dir)
        if build_ok:
            print("✅ Build succeeded")
        else:
            print("❌ Build failed")
            print(build_output[-500:])

    # Find HTML files
    files = find_html_files(public_dir)
    print(f"Found {len(files)} HTML files")

    # Check internal links
    print("Checking internal links...")
    broken_internal, redirects = check_internal_links(files, public_dir)
    print(f"  {len(broken_internal)} issues found")

    # Check external links (optional)
    broken_external = []
    if args.external:
        print("Checking external links (this may take a while)...")
        broken_external = check_external_links(files, public_dir)
        print(f"  {len(broken_external)} broken external links")

    # Generate report
    report = generate_report(broken_internal, broken_external, build_output, build_ok)

    # Write report
    output_dir.mkdir(parents=True, exist_ok=True)
    date_str = time.strftime("%Y-%m-%d")
    report_path = output_dir / f"{date_str}-links.md"
    report_path.write_text(report)
    print(f"\nReport written to {report_path}")

    # Also print summary
    print("\n" + report)

    # Exit code: non-zero if broken links found
    if broken_internal or broken_external:
        sys.exit(1)


if __name__ == "__main__":
    main()
