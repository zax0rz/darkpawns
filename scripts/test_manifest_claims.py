"""R5h fixture proof for shared manifest parsing and pair enumeration."""
import pathlib
import subprocess
import sys
import unittest
from fidelity_manifest import load_rows
from gen_expected_divergences import proof_scenarios
from manifest_claims import claimed_pairs, parse_oracle_proof, proof_scenario

class ManifestClaimsTest(unittest.TestCase):
    def test_claims(self):
        root = pathlib.Path(__file__).parent / "testdata" / "manifest-claims"
        rows = load_rows(root, root)
        self.assertEqual(len(rows), 5)
        self.assertEqual(rows[0]["line"], "2")
        expected = "1\talpha\tC1,C2\n2\talpha\tC2\n3\tbeta\tC5\n5\tgamma\tC5\n8\talpha\tC2\n"
        output = subprocess.check_output([sys.executable, str(pathlib.Path(__file__).parent / "manifest_claims.py"), "--manifest-dir", str(root), "--root", str(root)], text=True)
        self.assertEqual(output, expected)
        self.assertEqual("".join("\t".join(map(str, pair)) + "\n" for pair in claimed_pairs(rows)), expected)

    def test_ws_proof_is_parsed_but_not_scheduled_as_telnet(self):
        parsed = parse_oracle_proof("entry-name-gates@1[ws]")
        self.assertEqual(parsed.scenario, "entry-name-gates")
        self.assertEqual(parsed.seeds, ("1",))
        self.assertEqual(parsed.transport, "ws")
        row = {"status": "oracle-green", "proof": "entry-name-gates@1[ws]",
               "case_id": "browser-routing", "manifest": "entry.tsv", "line": "5"}
        self.assertEqual(claimed_pairs([row]), [])
        self.assertEqual(proof_scenario("entry-name-gates@1[ws]"), "entry-name-gates")
        self.assertEqual(proof_scenarios("entry-name-gates@1[ws]", "entry.tsv:5"),
                         ["entry-name-gates"])

    def test_rejects_unknown_transport_suffix(self):
        for suffix in ("foo", "telnet"):
            with self.subTest(suffix=suffix), self.assertRaisesRegex(ValueError, r"only \[ws\] is allowed"):
                parse_oracle_proof(f"entry-name-gates@1[{suffix}]")

    def test_rejects_malformed_transport_suffix(self):
        for token in ("entry-name-gates@1[ws", "entry-name-gates@1[ws]extra",
                      "entry-name-gates@1[ws][ws]", "entry-name-gates[ws]"):
            with self.subTest(token=token), self.assertRaisesRegex(ValueError, "transport suffix|seed list"):
                parse_oracle_proof(token)

    def test_plain_proof_keeps_ordinary_claim_behavior(self):
        parsed = parse_oracle_proof("alpha@1,2")
        self.assertEqual(parsed.scenario, "alpha")
        self.assertEqual(parsed.seeds, ("1", "2"))
        self.assertIsNone(parsed.transport)
        row = {"status": "oracle-green-multiseed", "proof": "alpha@1,2",
               "case_id": "C1", "manifest": "entry.tsv", "line": "6"}
        self.assertEqual(claimed_pairs([row]), [(1, "alpha", "C1"), (2, "alpha", "C1")])

if __name__ == "__main__":
    unittest.main()
