"""R3/R5h controls for the shared modernizer patch boundary."""

import json
from pathlib import Path
import tempfile
import unittest

import modernize


class ModernizeTests(unittest.TestCase):
    def setUp(self):
        self.policy = json.loads((modernize.ROOT / "scripts/modernize-policy.json").read_text())

    def test_retained_behavior_exclusions(self):
        for path in ("pkg/admin/activity.go", "cmd/dp-release/main.go", "pkg/parser/mob.go"):
            self.assertTrue(modernize.exclusion(path, "package example", self.policy), path)

    def test_new_rng_callers_are_excluded(self):
        for source in ("dprng.Number(1, 6)", "roller.Number(0, 2)", "Dice(2, 6)", "combat.GetRoller()", "spells.Cast()"):
            self.assertTrue(modernize.exclusion("pkg/game/new_file.go", source, self.policy), source)
        self.assertTrue(modernize.exclusion("pkg/dprng/new.go", "package dprng", self.policy))

    def test_filter_preserves_only_eligible_patches_without_writing(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            patches = []
            for name, text in (("eligible.go", "package x\nvar x interface{}\n"),
                               ("rng.go", "package x\n// dprng.Number\nvar x interface{}\n")):
                path = root / name
                path.write_text(text)
                patches.append(f"--- {path} (old)\n+++ {path} (new)\n@@ -1 +1 @@\n-interface{{}}\n+any\n")
            accepted, excluded = modernize.filter_diff("".join(patches), root, self.policy)
            self.assertIn("--- a/eligible.go", accepted)
            self.assertNotIn("rng.go", accepted)
            self.assertIn("rng.go", excluded)
            self.assertIn("interface{}", (root / "eligible.go").read_text())

    def test_rejects_outside_tree_or_malformed_patch(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory).resolve()
            source = root / "source.go"
            source.write_text("package x\n")
            patch = f"--- {source} (old)\n+++ {source} (new)\n@@ -1 +1 @@\n-package x\n+package y\n"
            with self.assertRaises(ValueError):
                modernize.filter_diff(patch, root / "other", self.policy)
            with self.assertRaises(ValueError):
                modernize.filter_diff("unrecognized output", root, self.policy)


if __name__ == "__main__":
    unittest.main()
