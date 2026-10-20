"""Exercise the stable bootstrap entry without a network or real installation."""
import os
import pathlib
import shutil
import subprocess
import tempfile
import unittest


@unittest.skipUnless(os.name != 'nt' and shutil.which('sh'), 'POSIX bootstrap entry')
class InstallEntryTest(unittest.TestCase):
    def test_download_failure_never_runs_and_success_propagates_status(self):
        entry = pathlib.Path(__file__).resolve().parent.parent / 'website/static/install.sh'
        with tempfile.TemporaryDirectory(prefix='darkpawns entry ') as temporary:
            root = pathlib.Path(temporary)
            bins = root / 'bin'
            bins.mkdir()
            curl = bins / 'curl'
            curl.write_text('''#!/bin/sh
printf '%s\\n' "$@" > "$ENTRY_ROOT/arguments"
while [ "$#" -gt 0 ]; do
    if [ "$1" = '-o' ]; then shift; destination=$1; fi
    shift
done
printf '%s' "$destination" > "$ENTRY_ROOT/temporary"
# Even a failed download can leave a complete, executable-looking script.
printf '%s\\n' 'printf "%s" "$1" > "$ENTRY_ROOT/executed"' 'exit 23' > "$destination"
exit "$ENTRY_CURL_STATUS"
''')
            curl.chmod(0o755)
            environment = dict(os.environ, PATH=str(bins) + os.pathsep + os.environ['PATH'], ENTRY_ROOT=str(root), ENTRY_CURL_STATUS='22')
            result = subprocess.run(['sh', str(entry), 'argument with spaces'], env=environment, capture_output=True)
            self.assertEqual(result.returncode, 22)
            self.assertFalse((root / 'executed').exists())
            self.assertFalse(pathlib.Path((root / 'temporary').read_text()).exists())
            environment['ENTRY_CURL_STATUS'] = '0'
            result = subprocess.run(['sh', str(entry), 'argument with spaces'], env=environment, capture_output=True)
            self.assertEqual(result.returncode, 23)
            self.assertEqual((root / 'executed').read_text(), 'argument with spaces')
            self.assertFalse(pathlib.Path((root / 'temporary').read_text()).exists())
            arguments = (root / 'arguments').read_text().splitlines()
            self.assertEqual(arguments[:5], ['-fsSL', '--proto', '=https', '--proto-redir', '=https'])
            self.assertIn('https://github.com/zax0rz/darkpawns/releases/latest/download/install.sh', arguments)


if __name__ == '__main__':
    unittest.main()
