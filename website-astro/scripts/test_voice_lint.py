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

    def test_style_and_script_blocks_skip_prose_heuristics(self):
        findings = self.lint(
            "<div class=\"status\"></div>\n"
            "<style>\n"
            "  .status.online .dot { background: var(--online); }\n"
            "</style>\n"
            "<script>\n"
            "  settle(response.ok ? 'online' : 'offline', 'server online');\n"
            "</script>\n",
            "src/components/Test.astro",
        )
        self.assertEqual(findings, [])

    def test_hard_rules_still_run_inside_script_blocks(self):
        findings = self.lint(
            "<script>\n"
            "  label.textContent = 'server offline — try again';\n"
            "</script>\n",
            "src/components/Test.astro",
        )
        self.assertEqual([finding.rule for finding in findings], ["dash-ban"])
        self.assertEqual(findings[0].severity, "error")

    def test_prose_outside_a_style_block_is_still_linted(self):
        findings = self.lint(
            "<p>A lost world. A dead server. One final resurrection.</p>\n"
            "<style>\n"
            "  .a.b .c { color: red; }\n"
            "</style>\n",
            "src/components/Test.astro",
        )
        self.assertEqual([finding.rule for finding in findings], ["trailer-rhythm"])
        self.assertEqual(findings[0].line, 1)

    def test_tsx_lints_copy_and_ignores_the_code_around_it(self):
        # A no-data em dash and a numeric range are typography, not prose; the
        # dash ban is aimed at sentences. Reading whole lines reported eight of
        # these as errors across the console.
        findings = self.lint(
            "export function Row({ n }: { n: number }) {\n"
            "  return <td className=\"px-4 text-ink-muted\">{n > 0 ? `Lvl ${n}` : '\u2014'}</td>;\n"
            "}\n",
            "src/pages/Row.tsx",
        )
        self.assertEqual(findings, [])

    def test_tsx_still_lints_real_copy(self):
        findings = self.lint(
            "export function Empty() {\n"
            "  return <p>A lost world. A dead server. One final resurrection.</p>;\n"
            "}\n",
            "src/pages/Empty.tsx",
        )
        self.assertEqual([f.rule for f in findings], ["trailer-rhythm"])

    def test_svg_paths_and_class_lists_are_not_prose(self):
        findings = self.lint(
            "const Icon = () => (\n"
            "  <svg><path d=\"M4.5 6.5 L6.5 8 L4.5 9.5\" /></svg>\n"
            ");\n"
            "const cls = 'w-1.5 h-1.5 border-t border-l border-rule';\n",
            "src/components/Icon.tsx",
        )
        self.assertEqual(findings, [])

    def test_chrome_register_flags_the_games_voice_in_the_interface(self):
        # Every one of these was live in the admin console and broke no rule.
        findings = self.lint(
            "---\ntextKind: original\nsource: test\nvoiceLayer: mythic-admin\n---\n"
            "Mythic Administrative Console\n",
        )
        self.assertEqual([f.rule for f in findings], ["chrome-register"])
        self.assertEqual(findings[0].severity, "warning")

    def test_chrome_register_ignores_frontmatter_schema(self):
        # `voiceLayer: mythic-admin` names the register; it is not written in it.
        findings = self.lint(
            "---\ntextKind: original\nsource: test\nvoiceLayer: mythic-admin\n---\n"
            "The archive is live.\n",
        )
        self.assertEqual(findings, [])

if __name__ == "__main__":
    unittest.main()
