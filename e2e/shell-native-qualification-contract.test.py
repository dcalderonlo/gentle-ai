"""Offline platform contracts; never invoke an installer or spoof execution admission."""
import os
import pathlib
import runpy
import sys
import unittest
from unittest import mock

SCRIPT = pathlib.Path(__file__).with_name('shell-macos-qualification.py')


def load(system, machine):
    uname = os.uname_result((system, 'fixture', 'fixture', 'fixture', machine))
    with mock.patch.object(sys, 'platform', system), mock.patch.object(os, 'uname', return_value=uname):
        return runpy.run_path(str(SCRIPT), run_name='qualification_contract')


class NativeQualificationContract(unittest.TestCase):
    def test_linux_assets_are_exact_and_unsupported_architectures_refused(self):
        module = load('linux', 'x86_64')
        self.assertTrue(callable(module.get('qualification_assets')), 'Linux asset selection is missing')
        assets = module['qualification_assets']('linux', 'x86_64')
        self.assertEqual(assets['schema'], 'gentle-shell-linux-qualification/v1')
        self.assertEqual(assets['triplet'], 'linux/amd64')
        self.assertEqual(assets['suffix'], 'linux_amd64')
        # This is the extracted ELF pin, not the compressed Node archive pin.
        self.assertEqual(assets['node'], (
            '41a74efb34cbde5c7632cdac0cf8bd1a14d0b8d73dc1e82755014d9a9ce70f5c', 123655872))
        self.assertEqual(assets['tools'], {
            'fd': ('761c72dc8e120d85b22292063be8a796e2eeb20eb3e4f38b8fa2343ccf3514a7',
                   1573549, 'fd-v10.5.0-x86_64-unknown-linux-musl/fd', 'fd 10.5.0'),
            'rg': ('33e15bcf1624b25cdd2a55813a47a2f95dbe126268203e76aa6a585d1e7b149c',
                   2265718, 'ripgrep-15.2.0-x86_64-unknown-linux-musl/rg', 'ripgrep 15.2.0'),
        })
        self.assertEqual(module['NODE_PIN'], assets['node'])
        self.assertEqual(module['TOOL_PINS'], assets['tools'])
        self.assertLessEqual(module['WHOLE_RUN_SECONDS'] + 60, 1330, 'Linux receipt/cleanup must precede worker kill')
        row = '"linux/amd64": asset("gentle-ai_4.3.1_linux_amd64.tar.gz", "' + 'a' * 64 + '", "' + 'b' * 64 + '", "gentle-ai")'
        self.assertEqual(module['MEMBER_ROW'].search(row).groups(), ('4.3.1', 'a' * 64, 'b' * 64))
        for system, machine in [('linux', 'aarch64'), ('darwin', 'x86_64'), ('win32', 'AMD64')]:
            with self.subTest(system=system, machine=machine):
                with self.assertRaises(module['QualificationError']):
                    module['qualification_assets'](system, machine)

    def test_linux_context_keeps_required_session_and_refuses_missing_bus(self):
        module = load('linux', 'x86_64')
        session = {'XDG_RUNTIME_DIR': '/run/user/1002',
                   'DBUS_SESSION_BUS_ADDRESS': 'unix:path=/run/user/1002/bus'}
        with mock.patch.object(sys, 'platform', 'linux'), mock.patch.dict(os.environ, session, clear=True):
            context = module['Context']('/work/fixture', '/fixture/supervisor')
            self.assertEqual(context.env['XDG_RUNTIME_DIR'], session['XDG_RUNTIME_DIR'])
            self.assertEqual(context.env['DBUS_SESSION_BUS_ADDRESS'], session['DBUS_SESSION_BUS_ADDRESS'])
            self.assertEqual(context.env['GOMAXPROCS'], '1')
            self.assertEqual(context.env['HOME'], '/work/fixture/home')
            self.assertEqual(context.env['XDG_CONFIG_HOME'], '/work/fixture/xdg/config')
            self.assertEqual(context.env['PATH'], '/usr/bin:/bin')
        with mock.patch.object(sys, 'platform', 'linux'), mock.patch.dict(os.environ, {}, clear=True):
            with self.assertRaises(module['QualificationError']):
                module['Context']('/work/fixture', '/fixture/supervisor')

    def test_existing_darwin_assets_schema_and_minimal_environment_hold(self):
        module = load('darwin', 'arm64')
        self.assertEqual(module['SCHEMA'], 'gentle-shell-macos-qualification/v1')
        self.assertEqual(module['NODE_PIN'], (
            'ee6fb0e015284d83a91e8ec5213f43a157f8a392b58555301682892ba928c04a', 120965360))
        self.assertEqual(module['TOOL_PINS'], {
            'fd': ('b67e1836c468e42e411984b56e52fa7abec08c2bd22c867398e7cc134aac5e12',
                   1334374, 'fd-v10.5.0-aarch64-apple-darwin/fd', 'fd 10.5.0'),
            'rg': ('3750b2e93f37e0c692657da574d7019a101c0084da05a790c83fd335bad973e4',
                   1764284, 'ripgrep-15.2.0-aarch64-apple-darwin/rg', 'ripgrep 15.2.0'),
        })
        with mock.patch.object(sys, 'platform', 'darwin'):
            context = module['Context']('/private/fixture', '/fixture/supervisor')
        self.assertEqual(context.env, {
            'HOME': '/private/fixture/home', 'TMPDIR': '/private/fixture/tmp/',
            'PATH': '/usr/bin:/bin', 'TERM': 'dumb', 'LANG': 'C',
            'XDG_CONFIG_HOME': '/private/fixture/xdg/config',
            'XDG_STATE_HOME': '/private/fixture/xdg/state',
            'XDG_CACHE_HOME': '/private/fixture/xdg/cache',
            'XDG_DATA_HOME': '/private/fixture/xdg/data',
        })


if __name__ == '__main__':
    unittest.main()
