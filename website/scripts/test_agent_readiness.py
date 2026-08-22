#!/usr/bin/env python3
"""Regression checks for machine-readable website surfaces and Caddy routing."""

import json
import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[2]


class AgentReadinessTests(unittest.TestCase):
    def test_openapi_document_is_valid_and_discoverable(self):
        spec = json.loads((ROOT / "web/api/openapi.json").read_text())
        self.assertEqual(spec["openapi"], "3.0.0")
        self.assertIn("HTTPError", spec["components"]["schemas"])
        llms = (ROOT / "website/static/llms.txt").read_text()
        self.assertIn("/openapi.json", llms)

    def test_caddy_negotiates_markdown_without_masking_404s(self):
        caddy = (ROOT / "website/deploy/Caddyfile").read_text()
        self.assertIn("@markdown header Accept *text/markdown*", caddy)
        self.assertIn('header Vary "Accept, Accept-Encoding"', caddy)
        self.assertIn("rewrite /404.md", caddy)
        self.assertNotIn("try_files {path} {path}/index.html /index.html", caddy)

    def test_markdown_404_has_recovery_links(self):
        body = (ROOT / "website/static/404.md").read_text()
        self.assertIn("# 404", body)
        for path in ("/sitemap.xml", "/llms.txt", "/docs/", "/openapi.json"):
            self.assertIn(path, body)


if __name__ == "__main__":
    unittest.main()
