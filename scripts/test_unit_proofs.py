"""R5h proofs against a real temporary Go module, not string-only stubs."""
from pathlib import Path
import shutil
import tempfile
import unittest

from unit_proofs import test_index, resolve, proof_symbols, execution_batches, outcome, validate_unit_rows

class UnitProofTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.directory = tempfile.TemporaryDirectory(prefix="dp-unit-fixture-")
        cls.root = Path(cls.directory.name)
        fixture = Path(__file__).parent / "testdata/unit-proofs"
        for file in fixture.iterdir():
            shutil.copyfile(file, cls.root / file.name)
        cls.index = test_index(cls.root)

    @classmethod
    def tearDownClass(cls):
        cls.directory.cleanup()

    def test_exact_declaration(self):
        self.assertIsNotNone(resolve("TestPass", self.index)[0])
        self.assertEqual(resolve("TestPrefix", self.index)[1], "matched only as a string")
        self.assertEqual(resolve("TestOnlyString", self.index)[1], "matched only as a string")
        self.assertEqual(resolve("TestAbsent", self.index)[1], "missing")
        self.assertEqual(proof_symbols("TestPass; TestSub/real_name"), ["TestPass", "TestSub/real_name"])

    def test_runtime_status_and_batch(self):
        names = ["TestPass", "TestSkip", "TestFail", "TestEmpty", "TestSub"]
        resolved = [(resolve(name, self.index)[0], name) for name in names]
        with tempfile.TemporaryDirectory() as directory:
            runs = execution_batches(resolved, self.root, Path(directory))
            self.assertEqual(len(runs), 1)
            run = runs["."]
            for declaration, name in resolved:
                want = {"TestPass":"PASS", "TestSkip":"skipped", "TestFail":"failing", "TestEmpty":"asserts nothing", "TestSub":"skipped"}[name]
                self.assertEqual(outcome(declaration,name,run)[0], want, name)
            # Subtest spelling is the actual emitted name; the unreachable Run is not proof.
            decl = resolve("TestSub", self.index)[0]
            self.assertEqual(outcome(decl,"TestSub/real_name",run)[0], "PASS")
            self.assertEqual(outcome(decl,"TestSub/unreachable",run)[0], "missing")
            self.assertEqual(outcome(decl,"TestSub/empty",run)[0], "asserts nothing")
            selected = run["command"][run["command"].index("-run")+1]
            self.assertEqual(selected,"^(TestEmpty|TestFail|TestPass|TestSkip|TestSub)$")
            text = (Path(directory)/run["log"]).read_text()
            self.assertNotIn('"Test":"TestPrefixSuffix"',text)

    def test_package_failure(self):
        with tempfile.TemporaryDirectory() as directory, tempfile.TemporaryDirectory() as output:
            root = Path(directory)
            shutil.copyfile(self.root / "go.mod", root / "go.mod")
            (root / "proofs_test.go").write_text("package proof\nimport \"testing\"\nfunc TestBroken(t *testing.T) { undefined() }\n")
            index = test_index(root)
            declaration = resolve("TestBroken", index)[0]
            run = execution_batches([(declaration, "TestBroken")], root, Path(output))["."]
            self.assertNotEqual(run["status"], 0)
            self.assertEqual(outcome(declaration, "TestBroken", run)[0], "package failure")

    def test_subtest_validation(self):
        row = {"status":"unit-green", "proof":"TestSub/real_name", "manifest":"fixture.tsv", "line":"2", "case_id":"fixture"}
        # A skipped sibling makes the root unproven, but the exact passing subtest is valid.
        errors = validate_unit_rows([row], self.root, self.index)
        self.assertEqual(errors, [])
        row["proof"] = "TestSub/unreachable"
        errors = validate_unit_rows([row], self.root, self.index)
        self.assertEqual(len(errors),1)
        self.assertIn("missing",errors[0])

    def test_divergent_approved_is_unit_proven_and_cites_its_approval(self):
        row = {"status":"divergent-approved", "proof":"TestSub/real_name", "manifest":"fixture.tsv",
               "line":"3", "case_id":"fixture-divergence", "notes":"Approved by Zach (DP-1373)."}
        self.assertEqual(validate_unit_rows([row], self.root, self.index), [])
        row["proof"] = "TestSub/unreachable"
        errors = validate_unit_rows([row], self.root, self.index)
        self.assertEqual(len(errors), 1)
        self.assertIn("missing", errors[0])
        row["proof"] = "TestSub/real_name"
        row["notes"] = "Approved, no issue cited."
        errors = validate_unit_rows([row], self.root, self.index)
        self.assertEqual(len(errors), 1)
        self.assertIn("must cite its approving issue", errors[0])

if __name__ == "__main__":
    unittest.main()
