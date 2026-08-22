#!/usr/bin/env python3

import importlib.util
import pathlib
import unittest


SCRIPT = pathlib.Path(__file__).with_name("parse_db.py")
SPEC = importlib.util.spec_from_file_location("parse_db", SCRIPT)
parse_db = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(parse_db)


class GeneratedTextTests(unittest.TestCase):
    def test_control_characters_are_removed(self):
        self.assertEqual(parse_db.clean_generated_text("halo\b\x00\ntext\t"), "halo\ntext\t")

    def test_yaml_string_escapes_quotes_and_newlines(self):
        self.assertEqual(parse_db.yaml_string('a "halo"\nline'), '"a \\"halo\\"\\nline"')


if __name__ == "__main__":
    unittest.main()
