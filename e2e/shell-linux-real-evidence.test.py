"""Public CLI checks for the unprivileged evidence reader; no installer calls."""
import hashlib
import io
import json
import os
import pathlib
import subprocess
import sys
import tarfile
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).with_name('shell-linux-real-evidence.py')


class EvidenceReaderContract(unittest.TestCase):
    def fixture(self, base):
        base = pathlib.Path(base)
        root = base / 'gentle-linux-qualification.fixture'
        logs = root / 'logs'
        root.mkdir(mode=0o700)
        logs.mkdir(mode=0o700)
        receipt = base / 'receipt.json'
        data = {'schema': 'gentle-shell-linux-qualification/v1', 'sourceSHA': 'a' * 40,
                'result': 'fail', 'root': str(root), 'failure': {'reason': 'retained failure'}}
        receipt.write_text(json.dumps(data))
        receipt.chmod(0o600)
        log = logs / '01-install.err'
        log.write_bytes(b'original stderr\n')
        log.chmod(0o600)
        stdout = logs / '01-install.out'
        stdout.write_bytes(b'original stdout\n')
        stdout.chmod(0o600)
        return receipt, log

    def invoke(self, receipt, base):
        return subprocess.run([sys.executable, '-I', '-B', str(SCRIPT), '--receipt', str(receipt),
                               '--base', str(base)], capture_output=True, timeout=5)

    def snapshot(self, base):
        result = {}
        for path in sorted(pathlib.Path(base).rglob('*')):
            info = path.lstat()
            value = (info.st_mode, info.st_nlink, info.st_size, info.st_mtime_ns)
            if path.is_symlink():
                value += (os.readlink(path),)
            elif path.is_file():
                value += (hashlib.sha256(path.read_bytes()).hexdigest(),)
            result[str(path)] = value
        return result

    def test_capture_retains_failure_and_exact_stdout_stderr_without_mutation(self):
        with tempfile.TemporaryDirectory(dir='/private/tmp' if sys.platform == 'darwin' else '/tmp') as base:
            receipt, log = self.fixture(base)
            before = self.snapshot(base)
            result = self.invoke(receipt, base)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stderr, b'')
            with tarfile.open(fileobj=io.BytesIO(result.stdout), mode='r:') as archive:
                self.assertEqual(archive.getnames(), ['receipt.json', 'logs/01-install.err', 'logs/01-install.out'])
                self.assertTrue(all(member.isfile() for member in archive.getmembers()))
                self.assertEqual(archive.extractfile('receipt.json').read(), receipt.read_bytes())
                self.assertEqual(archive.extractfile('logs/01-install.err').read(), log.read_bytes())
                self.assertEqual(archive.extractfile('logs/01-install.out').read(), b'original stdout\n')
            self.assertEqual(self.snapshot(base), before)

    def test_file_count_boundary_is_bounded_before_any_output(self):
        with tempfile.TemporaryDirectory(dir='/private/tmp' if sys.platform == 'darwin' else '/tmp') as base:
            receipt, log = self.fixture(base)
            for index in range(126):
                path = log.parent / ('02-extra-%03d.out' % index)
                path.write_bytes(b'')
                path.chmod(0o600)
            before = self.snapshot(base)
            result = self.invoke(receipt, base)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertEqual(result.stderr, b'')
            with tarfile.open(fileobj=io.BytesIO(result.stdout), mode='r:') as archive:
                self.assertEqual(len(archive.getmembers()), 129)  # receipt plus 128 logs
            self.assertEqual(self.snapshot(base), before)
            overflow = log.parent / '99-overflow.out'
            overflow.write_bytes(b'')
            overflow.chmod(0o600)
            before = self.snapshot(base)
            result = self.invoke(receipt, base)
            self.assertEqual(result.returncode, 2)
            self.assertEqual(result.stdout, b'')
            self.assertEqual(result.stderr, b'EVIDENCE-FAIL: command log count bound exceeded\n')
            self.assertEqual(self.snapshot(base), before)

    def test_rejected_links_and_unsafe_modes_emit_nothing_and_preserve_data(self):
        for case in ('receipt-link', 'log-link', 'log-mode', 'log-hardlink'):
            with self.subTest(case=case), tempfile.TemporaryDirectory(dir='/private/tmp' if sys.platform == 'darwin' else '/tmp') as base:
                receipt, log = self.fixture(base)
                target = pathlib.Path(base) / 'outside'
                target.write_bytes(b'do not read through a link\n')
                target.chmod(0o600)
                if case == 'receipt-link':
                    original = receipt.read_bytes()
                    target.write_bytes(original)
                    receipt.unlink()
                    receipt.symlink_to(target)
                elif case == 'log-link':
                    log.unlink()
                    log.symlink_to(target)
                elif case == 'log-mode':
                    log.chmod(0o644)
                else:
                    os.link(log, pathlib.Path(base) / 'extra-link')
                before = self.snapshot(base)
                result = self.invoke(receipt, base)
                self.assertEqual(result.returncode, 2, result.stderr)
                self.assertEqual(result.stdout, b'')
                self.assertTrue(result.stderr.startswith(b'EVIDENCE-FAIL: '), result.stderr)
                self.assertEqual(self.snapshot(base), before)


if __name__ == '__main__':
    unittest.main()
