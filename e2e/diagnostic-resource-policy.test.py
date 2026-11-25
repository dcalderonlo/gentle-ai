"""Offline safety contract for task-owned diagnostic workflow resource policy."""
import pathlib
import runpy
import subprocess
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
WORKFLOWS = ROOT / '.github/workflows'
OWNED = (
    'homebrew-supervisor-acquisition.yml', 'linux-direct-nnp-diagnostic.yml',
    'linux-lock-context-verification.yml', 'shell-linux-real-suppliers.yml',
    'supervisor-initialization-differential.yml', 'supervisor-repository-suites.yml',
    'supervisor-macos-gaps.yml', 'supervisor-native-qualification.yml',
)
MATRICES = {'homebrew-supervisor-acquisition.yml': 'acquisition',
            'linux-direct-nnp-diagnostic.yml': 'diagnostic',
            'supervisor-initialization-differential.yml': 'baseline',
            'supervisor-repository-suites.yml': 'baseline'}
job_body = runpy.run_path(str(ROOT / 'e2e/supervisor-repository-ci-contract.test.py'),
                         run_name='resource_contract_helpers')['job_body']


def events(text):
    header = text.split('\non:\n', 1)[1].split('\npermissions:', 1)[0]
    return {line.strip().rstrip(':') for line in header.splitlines()
            if line.startswith('  ') and not line.startswith('    ') and line.strip() and not line.strip().startswith('#')}


class DiagnosticResourcePolicy(unittest.TestCase):
    def test_no_task_owned_automatic_entrypoint(self):
        for name in OWNED:
            with self.subTest(workflow=name):
                text = (WORKFLOWS / name).read_text()
                expected = {'workflow_dispatch', 'workflow_call'} if name == 'supervisor-macos-gaps.yml' else {'workflow_dispatch'}
                self.assertEqual(events(text), expected)
                self.assertNotIn("github.event_name == 'push'", text)

    def test_every_task_matrix_requires_false_by_default_confirmation(self):
        for name, gate in MATRICES.items():
            with self.subTest(workflow=name):
                text = (WORKFLOWS / name).read_text()
                dispatch = text.split('  workflow_dispatch:\n', 1)[1].split('\npermissions:', 1)[0]
                confirmation = dispatch.split('      confirm_matrix:\n', 1)[1]
                self.assertIn('type: boolean', confirmation)
                self.assertIn('default: false', confirmation)
                body = job_body(text, gate)
                self.assertIn("github.event_name == 'workflow_dispatch'", body)
                self.assertIn('inputs.confirm_matrix == true', body)
                if name == 'supervisor-initialization-differential.yml':
                    self.assertIn('needs: baseline', job_body(text, 'differential'))
                    self.assertIn('baseline_sha:', dispatch)
                    self.assertIn('required: true', dispatch)
        repo = (WORKFLOWS / 'supervisor-repository-suites.yml').read_text()
        self.assertIn('inputs.macos_gaps_only != true', job_body(repo, 'baseline'))
        self.assertIn('inputs.confirm_matrix != true', job_body(repo, 'macos-gaps-only'))
        for confirm, macos, expected in [(None, None, (False, False)), (False, False, (False, False)),
                                         (True, False, (True, False)), (False, True, (False, True)),
                                         (True, True, (False, False))]:
            matrix = confirm is True and macos is not True
            mac = macos is True and confirm is not True
            self.assertEqual((matrix, mac), expected)

    def test_previous_push_only_diagnostics_keep_explicit_scope_guards(self):
        for name, prefix in [('linux-direct-nnp-diagnostic.yml', 'linux-nnp-'),
                             ('linux-lock-context-verification.yml', 'linux-lock-fix-')]:
            with self.subTest(workflow=name):
                text = (WORKFLOWS / name).read_text()
                self.assertIn("github.event_name == 'workflow_dispatch'", text)
                self.assertIn("github.repository == 'dcalderonlo/gentle-ai'", text)
                self.assertIn('refs/heads/verification/' + prefix, text)
                # This exact shell gate must precede checkout, even for uppercase expression matches.
                self.assertLess(text.index('# BEGIN diagnostic-manual-scope'), text.index('uses: actions/checkout'))
                self.assertIn('refs/heads/verification/' + prefix + '?*) ;;', text)
                guard = text.split('# BEGIN diagnostic-manual-scope\n', 1)[1].split('# END diagnostic-manual-scope', 1)[0]
                for ref, code in [('refs/heads/verification/' + prefix + 'test', 0),
                                  ('refs/heads/Verification/' + prefix + 'test', 1),
                                  ('refs/heads/verification/' + prefix, 1), ('refs/heads/main', 1)]:
                    with self.subTest(ref=ref):
                        result = subprocess.run(['/bin/bash', '-c', guard], env={'GITHUB_REF': ref},
                                                capture_output=True, timeout=5)
                        self.assertEqual(result.returncode, code)
                        self.assertEqual(result.stdout, b'')
                        self.assertEqual(result.stderr, b'' if code == 0 else b'Refused: exact diagnostic branch required\n')


if __name__ == '__main__':
    unittest.main()
