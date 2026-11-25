"""Offline contract for missing-only macOS evidence; never runs Go or installers."""
import pathlib
import subprocess
import unittest

FILE = pathlib.Path(__file__).resolve().parents[1] / '.github/workflows/supervisor-macos-gaps.yml'


class MacGapContract(unittest.TestCase):
    def test_missing_only_scope_and_isolated_modern_bash(self):
        text = FILE.read_text()
        for marker in ["github.repository == 'dcalderonlo/gentle-ai'", "github.event_name == 'workflow_dispatch'",
                       'timeout-minutes: 40', 'credential.helper=',
                       'https://ftp.gnu.org/gnu/bash/bash-5.3.tar.gz',
                       '0d5cd86965f869a26cf64f4b71be7b96f90a3ba8b3d74e27e8e9d9d5550f31ba',
                       'shasum -a 256 -c -', '--disable-nls', '--prefix=', 'GITHUB_PATH',
                       'TestReleaseSecurityScriptsAreSyntacticallyValidAndFailClosed',
                       'TestCanonicalReleasePublicKeysControlRealLinkerBuild',
                       'working-directory: bench', 'go vet ./...', 'go test -v ./... -count=1',
                       'exit "$status"', 'if: always()', 'if-no-files-found: error']:
            self.assertIn(marker, text, marker)
        for bad in ['brew install', 'brew upgrade', '/opt/homebrew/Cellar', 'sudo ', 'continue-on-error',
                    '-timeout=', '-skip', 'matrix:', 'actions/checkout', 'go test -v ./internal/cli', 'secrets.']:
            self.assertNotIn(bad, text, bad)
        self.assertLess(text.index('# BEGIN macos-gap-scope'), text.index(' fetch '))
        self.assertLess(text.index('# END macos-gap-scope'), text.index(' fetch '))
        self.assertEqual(text.count('runs-on:'), 1)

    def test_exact_manual_fork_guard(self):
        text = FILE.read_text()
        guard = text.split('# BEGIN macos-gap-scope\n')[1].split('# END macos-gap-scope')[0]
        for repo, event, ref, code in [
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/verification/macos-gaps-test', 0),
            ('dcalderonlo/gentle-ai', 'push', 'refs/heads/verification/macos-gaps-test', 1),
            ('other/gentle-ai', 'workflow_dispatch', 'refs/heads/verification/macos-gaps-test', 1),
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/Verification/macos-gaps-test', 1),
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/main', 1),
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/verification/macos-gaps-', 1),
        ]:
            with self.subTest(repo=repo, event=event, ref=ref):
                run = subprocess.run(['/bin/bash', '-c', guard], env={'PATH': '/usr/bin:/bin',
                    'LAB_REPOSITORY': repo, 'LAB_EVENT': event, 'LAB_REF': ref}, capture_output=True, timeout=5)
                self.assertEqual(run.returncode, code)
                self.assertEqual(run.stdout, b'')
                self.assertEqual(run.stderr, b'' if code == 0 else b'Refused: exact manual macOS gap scope required\n')


if __name__ == '__main__':
    unittest.main()
