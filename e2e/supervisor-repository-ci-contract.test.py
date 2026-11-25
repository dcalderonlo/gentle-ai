"""Routing and canonical-command controls; not executed suite evidence."""
import fnmatch
import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / '.github/workflows/supervisor-repository-suites.yml'


def driven_body(text):
    marker = '      - name: Run benchmark evidence\n'
    if marker not in text:
        return None
    tail = text.split(marker, 1)[1]
    lines = tail.split('        run: |\n', 1)[1].splitlines()
    result = []
    for line in lines:
        if line.strip() and not line.startswith('          '):
            break
        result.append(line[10:])
    return '\n'.join(result).rstrip()


def job_body(text, name):
    tail = text.split('\n  ' + name + ':\n', 1)[1]
    lines = []
    for line in tail.splitlines():
        if line.startswith('  ') and not line.startswith('    ') and line.strip():
            break
        lines.append(line)
    return '\n'.join(lines)


class RepositoryCIContract(unittest.TestCase):
    def test_diagnostic_branch_and_existing_feature_branch_are_routed(self):
        line = next(line for line in WORKFLOW.read_text().splitlines()
                    if line.strip().startswith('branches: ['))
        patterns = [item.strip().strip("'\"") for item in
                    line.split('[', 1)[1].split(']', 1)[0].split(',')]
        for branch in ('fix/homebrew-verified-supervisor', 'verification/repository-suites-probe'):
            self.assertTrue(any(fnmatch.fnmatchcase(branch, pattern) for pattern in patterns), branch)
        self.assertFalse(any(fnmatch.fnmatchcase('main', pattern) for pattern in patterns))

    def test_default_matrix_remains_baseline_gated(self):
        text = WORKFLOW.read_text()
        for name in ('root', 'bench', 'windows'):
            body = job_body(text, name)
            self.assertIn('    needs: baseline', body)
            self.assertIn('role: [baseline, candidate]', body)
            self.assertIn('fail-fast: false', body)
            self.assertNotIn('always()', body)
        baseline = job_body(text, 'baseline')
        conditions = [line.strip() for line in baseline.splitlines() if line.startswith('    if:')]
        self.assertIn(conditions, [[], ['if: inputs.macos_gaps_only != true']])
        # No input or explicit false keeps ordinary push/manual matrix admission.
        for selected in (None, False):
            self.assertTrue(selected is not True)

    def test_macos_opt_in_uses_same_ref_lane_and_skips_matrix(self):
        text = WORKFLOW.read_text()
        dispatch = text.split('  workflow_dispatch:\n', 1)[1].split('\npermissions:', 1)[0]
        for marker in ('macos_gaps_only:', 'type: boolean', 'default: false'):
            self.assertIn(marker, dispatch)
        self.assertIn('    if: inputs.macos_gaps_only != true', job_body(text, 'baseline'))
        mac = job_body(text, 'macos-gaps-only')
        self.assertIn("github.repository == 'dcalderonlo/gentle-ai'", mac)
        self.assertIn("github.event_name == 'workflow_dispatch'", mac)
        self.assertIn('inputs.macos_gaps_only == true', mac)
        self.assertIn("startsWith(github.ref, 'refs/heads/verification/macos-gaps-')", mac)
        self.assertIn('uses: ./.github/workflows/supervisor-macos-gaps.yml', mac)
        self.assertNotIn('runs-on:', mac)
        self.assertNotIn('secrets:', mac)
        callee = (ROOT / '.github/workflows/supervisor-macos-gaps.yml').read_text()
        self.assertIn('  workflow_call:', callee)
        self.assertIn('timeout-minutes: 40', callee)
        for selected in (None, False, True):
            baseline = selected is not True
            macos = selected is True
            self.assertNotEqual(baseline, macos)

    def test_driven_commands_and_assertions_match_canonical_ci(self):
        text = WORKFLOW.read_text()
        actual = driven_body(text)
        self.assertIsNotNone(actual, 'module Go tests are not driven evidence')
        self.assertEqual(actual, driven_body((ROOT / '.github/workflows/ci.yml').read_text()))
        header = text.split('      - name: Run benchmark evidence\n', 1)[1].split('        run: |', 1)[0]
        self.assertIn("if: runner.os == 'Linux'", header)


if __name__ == '__main__':
    unittest.main()
