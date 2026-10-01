"""The depth report counts approved divergences apart from C-parity completion."""
import unittest

from gen_fidelity_depth import render


def row(case, status):
    return {"handler":"h", "command":"c", "case_id":case, "depth":"D1", "scope":"s",
            "status":status, "proof":"-", "c_site":"-", "notes":"-"}


class RenderTests(unittest.TestCase):
    def test_divergent_approved_is_neither_proven_nor_actionable(self):
        rows = [row("a", "oracle-green"), row("b", "unit-green"), row("c", "blocked"),
                row("d", "excluded"), row("e", "divergent-approved")]
        report = render(rows)
        self.assertIn("1 blocked, 1 excluded, 1 divergent-approved", report)
        # 2 proven over 3 actionable (a, b, c): the divergence is not C parity.
        self.assertIn("Actionable completion: 2/3 = 66.7%", report)


if __name__ == "__main__":
    unittest.main()
