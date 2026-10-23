"""Every pinned copy of the publisher key must agree.

The key lives in three places that rotation must update together:
internal/releaseauth/trusted-key.pem (what dp-release verifies against),
the website install stub's PEM (website/static/install.sh), and the
documented Windows one-liner's RSA modulus/exponent
(docs/portable-release.md). A stale copy turns into either a broken
install (verification always fails) or a silent trust downgrade.
"""
import base64
import pathlib
import re
import shutil
import subprocess
import unittest

ROOT = pathlib.Path(__file__).resolve().parent.parent
KEY = ROOT / 'internal/releaseauth/trusted-key.pem'
STUB = ROOT / 'website/static/install.sh'
DOCS = ROOT / 'docs/portable-release.md'


class PublisherKeyCopiesTest(unittest.TestCase):
    def test_stub_pins_the_pinned_key(self):
        key_pem = [line for line in KEY.read_text().splitlines() if line.strip()]
        match = re.search(r"-----BEGIN PUBLIC KEY-----\n.*?\n-----END PUBLIC KEY-----", STUB.read_text(), flags=re.S)
        self.assertIsNotNone(match, 'install stub has no pinned public key')
        stub_pem = [line for line in match.group(0).splitlines() if line.strip()]
        self.assertEqual(stub_pem, key_pem, 'install stub pins a different key than trusted-key.pem')

    @unittest.skipUnless(shutil.which('openssl'), 'openssl needed to derive key material')
    def test_docs_one_liner_pins_the_same_key(self):
        modulus = subprocess.check_output(
            ['openssl', 'rsa', '-pubin', '-in', str(KEY), '-noout', '-modulus'], text=True)
        modulus = modulus.strip().split('=', 1)[1]
        text = subprocess.check_output(
            ['openssl', 'rsa', '-pubin', '-in', str(KEY), '-noout', '-text'], text=True)
        exponent = int(re.search(r'Exponent: (\d+)', text).group(1))
        exponent_b64 = base64.b64encode(exponent.to_bytes((exponent.bit_length() + 7) // 8, 'big')).decode()
        docs = DOCS.read_text()
        self.assertIn(f'<Modulus>{modulus}</Modulus>', docs, 'docs one-liner modulus differs from trusted-key.pem')
        self.assertIn(f'<Exponent>{exponent_b64}</Exponent>', docs, 'docs one-liner exponent differs from trusted-key.pem')


if __name__ == '__main__':
    unittest.main()
