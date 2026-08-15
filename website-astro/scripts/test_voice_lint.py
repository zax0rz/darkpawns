import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path


MODULE_PATH = Path(__file__).with_name("voice_lint.py")
SPEC = importlib.util.spec_from_file_location("voice_lint", MODULE_PATH)
assert SPEC and SPEC.loader
voice_lint = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = voice_lint
SPEC.loader.exec_module(voice_lint)


class VoiceLintTest(unittest.TestCase):
    def lint(self, content: str, relative: str = "src/content/blog/test.md"):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / relative
            path.parent.mkdir(parents=True)
            path.write_text(content, encoding="utf-8")
            original_site = voice_lint.SITE
            voice_lint.SITE = Path(directory)
            try:
                return voice_lint.lint_file(path)
            finally:
                voice_lint.SITE = original_site

    def test_dashes_and_launch_copy_are_errors(self):
        findings = self.lint(
            "---\ntextKind: original\nsource: test\nvoiceLayer: mythic-admin\n---\n"
            "A living, breathing world — reborn.\n"
        )
        self.assertEqual({finding.rule for finding in findings}, {"dash-ban", "launch-copy"})
        self.assertTrue(all(finding.severity == "error" for finding in findings))

    def test_archive_body_is_exempt_but_frontmatter_is_not(self):
        findings = self.lint(
            "---\ndescription: New copy — still checked\n---\nA preserved post — unchanged.\n",
            "src/content/archive/test.md",
        )
        dash_findings = [finding for finding in findings if finding.rule == "dash-ban"]
        self.assertEqual(len(dash_findings), 2)

    def test_declared_verbatim_archive_exempts_body(self):
        findings = self.lint(
            "---\ntextKind: verbatim\nsource: archive\nvoiceLayer: frontline\n"
            "description: New copy — still checked\n---\nA preserved post — unchanged.\n",
            "src/content/archive/test.md",
        )
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].line, 5)

    def test_trailer_rhythm_only_warns(self):
        findings = self.lint(
            "---\ntextKind: original\nsource: test\nvoiceLayer: mythic-admin\n---\n"
            "A lost world. A dead server. One final resurrection.\n"
        )
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].rule, "trailer-rhythm")
        self.assertEqual(findings[0].severity, "warning")


if __name__ == "__main__":
    unittest.main()
