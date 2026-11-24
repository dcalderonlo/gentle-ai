"""Standard-library policy checks; not Linux installation qualification."""
import itertools
import pathlib
import subprocess
import textwrap
import types
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / '.github/workflows/shell-linux-first-ci.yml'
UPSTREAM = 'Gentleman-Programming/gentle-ai'
EXISTING_FORK = 'decode2/gentle-ai'
OWNED_FORK = 'dcalderonlo/gentle-ai'
EXPECTED = (
    "((github.repository == 'Gentleman-Programming/gentle-ai' || "
    "github.repository == 'decode2/gentle-ai') && github.event_name != 'push') || "
    "(github.repository == 'dcalderonlo/gentle-ai' && "
    "github.event_name == 'workflow_dispatch' && "
    "startsWith(github.ref, 'refs/heads/verification/'))"
)


class LinuxForkPolicy(unittest.TestCase):
    def condition(self):
        conditions = [line.strip()[4:] for line in WORKFLOW.read_text().splitlines()
                      if line.startswith('    if: ')]
        self.assertEqual(len(conditions), 1)
        # Evaluate only this exact, known-safe GitHub boolean expression.
        self.assertEqual(conditions[0], EXPECTED)
        return conditions[0].replace('||', 'or').replace('&&', 'and')

    def permits(self, expression, repository, event, ref):
        context = types.SimpleNamespace(repository=repository, event_name=event, ref=ref)
        coarse = eval(expression, {'__builtins__': {}},
                      {'github': context, 'startsWith': lambda value, prefix:
                       value.casefold().startswith(prefix.casefold())})
        if not coarse:
            return False
        text = WORKFLOW.read_text()
        begin = text.index('# BEGIN exact-fork-scope')
        end = text.index('# END exact-fork-scope')
        self.assertLess(end, text.index('git init -q'))
        self.assertLess(end, text.index('sudo -n apt-get'))
        body = text[begin:].split('\n', 1)[1].split('# END exact-fork-scope')[0]
        result = subprocess.run(['/bin/bash', '-c', 'set -eu\n' + textwrap.dedent(body)],
                                env={'LAB_REPOSITORY': repository, 'LAB_EVENT': event,
                                     'LAB_REF': ref},
                                capture_output=True, text=True, timeout=5)
        self.assertEqual(result.stdout, '')
        self.assertIn(result.returncode, (0, 1))
        return result.returncode == 0

    def test_owned_fork_is_manual_and_verification_only(self):
        expression = self.condition()
        for repository, event, ref in itertools.product(
                (OWNED_FORK, 'unrelated/repository'),
                ('workflow_dispatch', 'push', 'pull_request'),
                ('refs/heads/verification/linux-lab', 'refs/heads/main',
                 'refs/tags/verification/linux-lab', 'refs/heads/verification-unsafe',
                 'refs/heads/VERIFICATION/linux-lab', 'refs/heads/Verification/linux-lab')):
            with self.subTest(repository=repository, event=event, ref=ref):
                expected = (repository == OWNED_FORK and event == 'workflow_dispatch'
                            and ref.startswith('refs/heads/verification/'))
                self.assertEqual(self.permits(expression, repository, event, ref), expected)

    def test_existing_repositories_keep_original_events(self):
        expression = self.condition()
        for repository, event in itertools.product(
                (UPSTREAM, EXISTING_FORK), ('pull_request', 'workflow_dispatch', 'push')):
            with self.subTest(repository=repository, event=event):
                self.assertEqual(self.permits(expression, repository, event, 'refs/heads/main'),
                                 event != 'push')
        self.assertIn('permissions: {}', WORKFLOW.read_text())


if __name__ == '__main__':
    unittest.main()
