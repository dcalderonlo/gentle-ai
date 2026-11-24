"""Deterministic main-entry preflight tests with virtual kernel reads, no Guest."""
import ast
import contextlib
import copy
import io
import json
import pathlib
import types
import unittest

SCRIPT = pathlib.Path(__file__).with_name('shell-linux-user-install-guest.py')
GROUP = '/sys/fs/cgroup/fixture.service'
PORT = '/proc/sys/net/ipv4/ip_unprivileged_port_start'
LIMITS = {'memory.max': '3221225472', 'memory.swap.max': '0', 'cpu.max': '100000 100000', 'pids.max': '512'}


class VirtualKernel:
    def __init__(self):
        self.uid, self.machine, self.readonly, self.canonical = 1002, 'x86_64', True, True
        self.values = {'/proc/self/status': ''.join(key + ':\t0000000000000000\n' for key in
                       ['CapInh', 'CapPrm', 'CapEff', 'CapAmb', 'CapBnd']) + 'NoNewPrivs:\t1\n',
                       '/proc/self/cgroup': '0::/fixture.service\n',
                       '/proc/self/mountinfo': 'fixture - cgroup2 cgroup rw\n', PORT: '1024\n'}
        self.values.update({GROUP + '/' + key: value + '\n' for key, value in LIMITS.items()})
        self.reads = []

    def path(self, value):
        kernel = self

        class Path:
            def __init__(self, text):
                self.text = str(text)

            def __str__(self):
                return self.text

            def __truediv__(self, name):
                return Path(self.text + '/' + name)

            def resolve(self):
                return self if kernel.canonical else Path('/different')

            def read_text(self):
                kernel.reads.append(self.text)
                return kernel.values[self.text]

            def is_socket(self):
                return True  # Existing full/smoke modes reach the kernel gate.
        return Path(value)


def invoke(mode, kernel, script=SCRIPT):
    tree = ast.parse(pathlib.Path(script).read_text())
    wanted = {'require', 'kernel', 'main'}
    nodes = [node for node in tree.body if isinstance(node, ast.FunctionDef) and node.name in wanted]
    assert {node.name for node in nodes} == wanted
    namespace = {'MODE': mode, 'REPORT': {}, 'json': json,
                 'pathlib': types.SimpleNamespace(Path=kernel.path),
                 'os': types.SimpleNamespace(getuid=lambda: kernel.uid,
                     uname=lambda: types.SimpleNamespace(machine=kernel.machine),
                     statvfs=lambda _: types.SimpleNamespace(f_flag=int(kernel.readonly)), ST_RDONLY=1)}
    exec(compile(ast.Module(body=nodes, type_ignores=[]), str(script), 'exec'), namespace)
    out, err, code, reason = io.StringIO(), io.StringIO(), 0, ''
    with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
        try:
            namespace['main']()
        except RuntimeError as error:
            code, reason = 1, str(error)
    return {'exit': code, 'stdout': out.getvalue(), 'stderr': err.getvalue(),
            'reason': reason, 'report': namespace['REPORT']}


class PhysicalPreflightContract(unittest.TestCase):
    def test_clean_physical_mode_never_reads_tls_port_and_keeps_all_checks(self):
        kernel = VirtualKernel()
        before = copy.deepcopy(kernel.values)
        result = invoke('physical', kernel)
        expected = {'physicalWorker': {'uid': 1002, 'limits': LIMITS}, 'functionalReady': False}
        self.assertEqual(result['exit'], 0, result)
        self.assertEqual(result['stdout'], json.dumps(expected, sort_keys=True) + '\n')
        self.assertEqual(result['stderr'], '')
        self.assertEqual(result['report'], {'kernel': expected['physicalWorker']})
        self.assertNotIn(PORT, kernel.reads)
        self.assertEqual(kernel.values, before)

    def test_each_physical_guard_still_refuses_without_partial_report(self):
        mutations = [('uid', 'real UID1002/linuxamd64 required'), ('arch', 'real UID1002/linuxamd64 required'),
                     ('nnp', 'NoNewPrivs missing'), ('membership', 'unified actual membership required'),
                     ('canonical', 'noncanonical cgroup mapping'), ('mount', 'cgroup2 mount missing'),
                     ('readonly', 'physical Guest root mount must be read-only')]
        mutations += [(key, 'capability present') for key in ['CapInh', 'CapPrm', 'CapEff', 'CapAmb', 'CapBnd']]
        mutations += [(key, 'physical cgroup limit differs') for key in LIMITS]
        for change, reason in mutations:
            with self.subTest(change=change):
                kernel = VirtualKernel()
                if change == 'uid': kernel.uid = 1001
                elif change == 'arch': kernel.machine = 'aarch64'
                elif change == 'nnp': kernel.values['/proc/self/status'] = kernel.values['/proc/self/status'].replace('NoNewPrivs:\t1', 'NoNewPrivs:\t0')
                elif change == 'membership': kernel.values['/proc/self/cgroup'] = 'legacy\n'
                elif change == 'canonical': kernel.canonical = False
                elif change == 'mount': kernel.values['/proc/self/mountinfo'] = 'no cgroup2\n'
                elif change == 'readonly': kernel.readonly = False
                elif change in LIMITS: kernel.values[GROUP + '/' + change] = 'wrong\n'
                else: kernel.values['/proc/self/status'] = kernel.values['/proc/self/status'].replace(change + ':\t0000000000000000', change + ':\t0000000000000001')
                before = copy.deepcopy(kernel.values)
                result = invoke('physical', kernel)
                self.assertEqual((result['exit'], result['stdout'], result['stderr'], result['reason'], result['report']),
                                 (1, '', '', reason, {}))
                self.assertEqual(kernel.values, before)

    def test_existing_fixture_modes_still_require_tls_port_and_probe_is_unchanged(self):
        for mode in ['probe', 'full', 'direct', 'smoke']:
            with self.subTest(mode=mode):
                kernel = VirtualKernel()
                before = copy.deepcopy(kernel.values)
                result = invoke(mode, kernel)
                self.assertEqual((result['exit'], result['stdout'], result['stderr'], result['reason'], result['report']),
                                 (1, '', '', 'nonprivileged TLS fixture port unavailable', {}))
                self.assertEqual(kernel.values, before)
        kernel = VirtualKernel()
        kernel.values[PORT] = '0\n'
        result = invoke('probe', kernel)
        self.assertEqual(result['exit'], 0, result)
        self.assertEqual(result['stderr'], '')
        self.assertIn(PORT, kernel.reads)
        self.assertEqual(json.loads(result['stdout']), {'physicalWorker': {'uid': 1002, 'limits': LIMITS}, 'functionalReady': False})


if __name__ == '__main__':
    unittest.main()
