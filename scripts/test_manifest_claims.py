"""R5h fixture proof for shared manifest parsing and pair enumeration."""
import pathlib
import subprocess
import sys
import unittest
from fidelity_manifest import load_rows
from manifest_claims import claimed_pairs

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

if __name__ == "__main__":
    unittest.main()
