"""Offline routing and failure-propagation checks; never executes real Go."""
import os
import pathlib
import shutil
import subprocess
import tempfile
import textwrap
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / '.github/workflows/homebrew-supervisor-acquisition.yml'
LINUX = 'Verify Linux release security boundaries'
WINDOWS = 'Verify Windows installer boundaries'
LINUX_NAMES = ('TestReleaseAssetVerifierPreservesReadOnlyRotationVerification',
               'TestReleaseDistributionPolicyAssertionFailsClosed',
               'TestReleaseDistributionPolicyAcceptsSemanticYAMLFormatting',
               'TestModifiedReleaseVerifierCannotGainWriteAuthority',
               'TestWindowsInstallAndUpgradeContainNoRemoteBinaryOrScriptPath')
WINDOWS_NAMES = ('TestUserWindowsGoReusable', 'TestUserWindowsStageMainMetadataKeepsReusedSnapshot',
                 'TestUserWindowsInventoryCarriedAdmitsOperatorStateOnly',
                 'TestUserWindowsRestartManagerReportsOwnProcess',
                 'TestWindowsStableCheckNodeComparesRetainedArchive')


def step(text, name):
    return text.split('      - name: ' + name + '\n', 1)[1].split('\n      - name:', 1)[0]


class Contract(unittest.TestCase):
    def test_manual_budget_and_os_association(self):
        text = WORKFLOW.read_text()
        header = text.split('\non:\n', 1)[1].split('\npermissions:', 1)[0]
        self.assertIn('workflow_dispatch:', header)
        self.assertNotIn('  push:', header)
        self.assertIn('default: false', header)
        for expected in ('timeout-minutes: 20', 'os: [ubuntu-latest, macos-latest, windows-latest]',
                         'inputs.confirm_matrix == true', 'contents: read',
                         'actions/checkout@93cb6efe18208431cddfb8368fd83d5badbf9bfd',
                         'actions/setup-go@4a3601121dd01d1626a1e23e37211e3254c1c06c'):
            self.assertIn(expected, text)
        for name, platform, inventory in [(LINUX, 'Linux', LINUX_NAMES), (WINDOWS, 'Windows', WINDOWS_NAMES)]:
            body = step(text, name)
            self.assertIn("if: runner.os == '" + platform + "'", body)
            self.assertIn('shell: bash', body)
            self.assertIn('set -euo pipefail', body)
            self.assertIn('-timeout=5m', body)
            for required in inventory:
                self.assertIn(required, body)

    def execute(self, name, result='pass', inventory='complete'):
        with tempfile.TemporaryDirectory(prefix='supervisor-contract-') as directory:
            root = pathlib.Path(directory)
            binary = root / 'bin'
            binary.mkdir()
            names = LINUX_NAMES if name == LINUX else WINDOWS_NAMES
            fake_go = '#!/bin/bash\ncase " $* " in\n *" -list "*)\n'
            fake_go += 'if [ "$MOCK_INVENTORY" = missing ]; then exit 0; fi\n'
            fake_go += ''.join("printf '%s\\n' '" + item + "'\n" for item in names)
            fake_go += ';;\n *)\n'
            fake_go += "if [ \"$MOCK_RESULT\" = skip ]; then printf '%s\\n' '--- SKIP: TestReleaseAssetVerifierPreservesReadOnlyRotationVerification (0.00s)'; exit 0; fi\n"
            fake_go += "printf '%s\\n' '--- PASS: TestReleaseAssetVerifierPreservesReadOnlyRotationVerification (0.00s)'\n"
            fake_go += 'if [ "$MOCK_RESULT" = fail ]; then exit 7; fi\n;;\nesac\n'
            for filename, content in [('go', fake_go), ('uname', "#!/bin/bash\nprintf '%s\\n' x86_64\n"),
                                      ('jq', '#!/bin/bash\nexit 0\n'), ('sha256sum', '#!/bin/bash\nexit 0\n')]:
                path = binary / filename
                path.write_text(content)
                path.chmod(0o700)
            env = dict(os.environ, PATH=str(binary) + ':/usr/bin:/bin', RUNNER_TEMP=str(root),
                       MOCK_RESULT=result, MOCK_INVENTORY=inventory)
            script = textwrap.dedent(step(WORKFLOW.read_text(), name).split('        run: |\n', 1)[1])
            result = subprocess.run(['/bin/bash', '-c', script], env=env, capture_output=True, timeout=5)
            result.expected_prerequisites = (''.join(shutil.which(tool, path=env['PATH']) + '\n'
                                                    for tool in ('bash', 'jq', 'sha256sum')).encode()
                                              if name == LINUX else b'')
            return result

    def test_selected_steps_accept_complete_inventory(self):
        for name in (LINUX, WINDOWS):
            with self.subTest(step=name):
                result = self.execute(name)
                self.assertEqual((result.returncode, result.stderr), (0, b''), result.stdout)

    def test_go_failure_is_not_hidden_by_tee_or_pass_text(self):
        for name in (LINUX, WINDOWS):
            with self.subTest(step=name):
                result = self.execute(name, result='fail')
                self.assertEqual((result.returncode, result.stderr), (7, b''), result.stdout)
                self.assertIn(b'--- PASS:', result.stdout)

    def test_mandatory_linux_skip_is_not_pass(self):
        result = self.execute(LINUX, result='skip')
        self.assertEqual((result.returncode, result.stderr), (1, b''), result.stdout)
        self.assertIn(b'--- SKIP:', result.stdout)

    def test_missing_inventory_fails_before_execution(self):
        for name in (LINUX, WINDOWS):
            with self.subTest(step=name):
                result = self.execute(name, inventory='missing')
                self.assertEqual((result.returncode, result.stdout, result.stderr),
                                 (1, result.expected_prerequisites, b''))


if __name__ == '__main__':
    unittest.main()
