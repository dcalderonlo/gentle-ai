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


class RepositoryCIContract(unittest.TestCase):
    def test_diagnostic_branch_and_existing_feature_branch_are_routed(self):
        line = next(line for line in WORKFLOW.read_text().splitlines()
                    if line.strip().startswith('branches: ['))
        patterns = [item.strip().strip("'\"") for item in
                    line.split('[', 1)[1].split(']', 1)[0].split(',')]
        for branch in ('fix/homebrew-verified-supervisor', 'verification/repository-suites-probe'):
            self.assertTrue(any(fnmatch.fnmatchcase(branch, pattern) for pattern in patterns), branch)
        self.assertFalse(any(fnmatch.fnmatchcase('main', pattern) for pattern in patterns))

    def test_driven_commands_and_assertions_match_canonical_ci(self):
        text = WORKFLOW.read_text()
        actual = driven_body(text)
        self.assertIsNotNone(actual, 'module Go tests are not driven evidence')
        self.assertEqual(actual, driven_body((ROOT / '.github/workflows/ci.yml').read_text()))
        header = text.split('      - name: Run benchmark evidence\n', 1)[1].split('        run: |', 1)[0]
        self.assertIn("if: runner.os == 'Linux'", header)


if __name__ == '__main__':
    unittest.main()
