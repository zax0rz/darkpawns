#!/usr/bin/env python3

import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[2]


class HugoTemplateTests(unittest.TestCase):
    def test_changelog_uses_site_data_namespace(self):
        template = (ROOT / "website/layouts/section/changelog.html").read_text()
        self.assertIn(".Site.Data.changelog", template)
        self.assertNotIn("hugo.Data", template)


if __name__ == "__main__":
    unittest.main()
