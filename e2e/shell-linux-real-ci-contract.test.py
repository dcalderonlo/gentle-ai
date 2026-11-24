"""Offline controls for the single-dispatch, uncontaminated Linux lab."""
import pathlib
import subprocess
import unittest

WORKFLOW = pathlib.Path(__file__).resolve().parents[1] / '.github/workflows/shell-linux-real-suppliers.yml'


class CleanLinuxLabContract(unittest.TestCase):
    def test_job_is_owned_manual_bounded_and_uses_unmodified_trust(self):
        text = WORKFLOW.read_text()
        for marker in ['permissions: {}', 'workflow_dispatch:', 'timeout-minutes: 60',
                       "github.repository == 'dcalderonlo/gentle-ai'", "github.event_name == 'workflow_dispatch'",
                       '--property=NoNewPrivileges=yes', '--property=MemoryMax=3221225472',
                       '--property=MemorySwapMax=0', '--property=CPUQuota=100%', '--property=TasksMax=512',
                       'CapabilityBoundingSet=', 'ProtectSystem=strict', 'KillMode=control-group',
                       'XDG_RUNTIME_DIR=/run/user/1002', 'DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1002/bus',
                       'go build -mod=readonly -p 1', '/fixture/harness.py physical',
                       'shell-macos-qualification.py', '--receipt /work/supplier-receipt.json',
                       'shell-linux-real-evidence.py', 'if-no-files-found: error']:
            self.assertIn(marker, text, marker)
        for forbidden in ['pull_request:', 'secrets.', 'actions/checkout', 'openssl ',
                          'update-ca-certificates', 'tee -a /etc/hosts', 'ip_unprivileged_port_start=',
                          '/fixture/harness.py full', '/fixture/harness.py smoke', '/fixture/harness.py probe', '-ldflags', 'continue-on-error']:
            self.assertNotIn(forbidden, text, forbidden)
        self.assertLess(text.index('# BEGIN clean-lab-scope'), text.index(' fetch '))
        self.assertLess(text.index('# END clean-lab-scope'), text.index('sudo -n apt-get'))
        self.assertIn('git rev-parse HEAD)', text)
        self.assertIn('1330', text)
        self.assertIn("payload['sourceSHA']", text)
        self.assertIn("payload['candidate']['sha256']", text)
        self.assertIn("payload['result'] == 'pass'", text)
        # Capture/collection cannot turn the original failed worker green.
        self.assertIn('exit "$status"', text)

    def test_actual_case_sensitive_guard_precedes_source_and_preparation(self):
        text = WORKFLOW.read_text()
        guard = text.split('# BEGIN clean-lab-scope\n', 1)[1].split('# END clean-lab-scope', 1)[0]
        rows = [
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/verification/linux-real-check', 0),
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/verification/nested/branch', 0),
            ('dcalderonlo/gentle-ai', 'push', 'refs/heads/verification/linux-real-check', 1),
            ('other/gentle-ai', 'workflow_dispatch', 'refs/heads/verification/linux-real-check', 1),
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/VERIFICATION/test', 1),
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/Verification/test', 1),
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/main', 1),
            ('dcalderonlo/gentle-ai', 'workflow_dispatch', 'refs/heads/verification/', 1),
        ]
        for repo, event, ref, code in rows:
            with self.subTest(repo=repo, event=event, ref=ref):
                result = subprocess.run(['/bin/bash', '-c', guard], env={
                    'LAB_REPOSITORY': repo, 'LAB_EVENT': event, 'LAB_REF': ref,
                    'PATH': '/usr/bin:/bin'}, capture_output=True, timeout=5)
                self.assertEqual(result.returncode, code)
                self.assertEqual(result.stdout, b'')
                self.assertEqual(result.stderr, b'' if code == 0 else b'Refused: exact owned-fork manual verification/ scope required\n')


if __name__ == '__main__':
    unittest.main()
