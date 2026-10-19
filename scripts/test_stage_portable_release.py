import hashlib
import pathlib
import subprocess
import tempfile
import unittest

from stage_portable_release import SEEDS, stage


class PortableStagingTest(unittest.TestCase):
    def fixture(self, root):
        for path in SEEDS:
            name = path if path.startswith('lib/misc/') else path + '/index'
            if path == 'web/public':
                name = path + '/index.html'
            target = root / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_bytes(b'starter\n')
        for name in ['lib/etc/players', 'lib/plrobjs/A-E/alice.objs', 'lib/world/etc/board.mort']:
            target = root / name
            target.parent.mkdir(parents=True, exist_ok=True)
            target.write_text('must not ship\n')
        subprocess.run(['git', 'init', '-q'], cwd=root, check=True)
        subprocess.run(['git', 'add', '.'], cwd=root, check=True)
        subprocess.run(['git', '-c', 'user.name=Portable test', '-c', 'user.email=portable@example.invalid', 'commit', '-qm', 'fixture'], cwd=root, check=True)
        ui = root / 'admin-ui/dist'
        ui.mkdir(parents=True)
        (ui / 'index.html').write_text('compiled admin')

    def test_committed_allowlist_excludes_runtime_and_untracked_world(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = pathlib.Path(temporary)
            self.fixture(root)
            (root / 'lib/world/wld/untracked.wld').write_text('local world edits')
            (root / 'web/public/index.html').write_text('uncommitted web edits')
            manifest = stage(root, 'test')
            files = manifest['files']
            self.assertIn('lib/world/wld/index', files)
            self.assertIn('admin-ui/index.html', files)
            self.assertEqual(files['web/index.html'], hashlib.sha256(b'starter\n').hexdigest())
            self.assertFalse(any('untracked' in p or p.startswith(('lib/etc/', 'lib/plrobjs/', 'lib/world/etc/')) for p in files))
            # A second build replaces only its recognized generated staging tree.
            self.assertEqual(stage(root, 'test')['files'], files)

    def test_refuses_unknown_output_directory(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = pathlib.Path(temporary)
            target = root / 'release-assets'
            target.mkdir()
            (target / 'keep').write_text('operator data')
            with self.assertRaises(RuntimeError):
                stage(root, 'test')
            self.assertEqual((target / 'keep').read_text(), 'operator data')


if __name__ == '__main__':
    unittest.main()
