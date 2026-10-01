"""The depth report counts approved divergences apart from C-parity completion."""
import unittest
import unittest.mock

from gen_fidelity_depth import render, validate


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


class ValidateTests(unittest.TestCase):
    def test_rejects_seed_lists_the_claims_census_cannot_parse(self):
        good = dict(row("ok", "oracle-green-multiseed"), proof="vehicle@1,2,3,5,8", manifest="m.tsv", line="2")
        bad = dict(row("bad", "oracle-green-multiseed"), proof="vehicle@1,vehicle@2", manifest="m.tsv", line="3")
        with unittest.mock.patch("gen_fidelity_depth.validate_unit_rows", return_value=[]):
            self.assertEqual(validate([good], {"vehicle": {"ok"}}), [])
            errors = validate([bad], {"vehicle": {"bad"}})
        self.assertEqual(len(errors), 1)
        self.assertIn("invalid seeds", errors[0])


if __name__ == "__main__":
    unittest.main()
