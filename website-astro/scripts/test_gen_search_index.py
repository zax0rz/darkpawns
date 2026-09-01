#!/usr/bin/env python3
"""
Unit tests for Astro Unified Search Index Generator
"""

import unittest
import json
from pathlib import Path
import subprocess

SCRIPTS_DIR = Path(__file__).resolve().parent
ASTRO_ROOT = SCRIPTS_DIR.parent
PROJECT_ROOT = ASTRO_ROOT.parent
INDEX_FILE = PROJECT_ROOT / "website" / "static" / "data" / "search-index.json"

class TestSearchIndexGenerator(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        # Run generator
        res = subprocess.run(
            ["python3", str(SCRIPTS_DIR / "gen_search_index.py")],
            cwd=str(ASTRO_ROOT),
            capture_output=True,
            text=True
        )
        assert res.returncode == 0, f"Generator failed:\n{res.stderr}"

    def test_index_file_exists_and_valid_json(self):
        self.assertTrue(INDEX_FILE.exists(), f"Index file {INDEX_FILE} does not exist")
        with open(INDEX_FILE, 'r', encoding='utf-8') as f:
            data = json.load(f)
        self.assertIsInstance(data, list)
        self.assertGreater(len(data), 3000, "Index should contain at least 3,000 entries")

    def test_schema_keys(self):
        with open(INDEX_FILE, 'r', encoding='utf-8') as f:
            data = json.load(f)
        required_keys = {"t", "c", "s", "u", "k", "d", "v"}
        for item in data[:200]:
            self.assertTrue(required_keys.issubset(item.keys()), f"Missing keys in item: {item}")
            self.assertTrue(item["u"].startswith("/"), f"URL must be absolute path: {item['u']}")

    def test_categories_represented(self):
        with open(INDEX_FILE, 'r', encoding='utf-8') as f:
            data = json.load(f)
        categories = {item["c"] for item in data}
        self.assertIn("help", categories)
        self.assertIn("mobs", categories)
        self.assertIn("items", categories)
        self.assertIn("world", categories)
        self.assertIn("docs", categories)
        self.assertIn("pages", categories)

if __name__ == "__main__":
    unittest.main()
