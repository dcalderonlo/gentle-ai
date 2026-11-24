"""Read owned evidence as the lab user; emit a bounded regular-file-only tar.

Usage: python3 -I shell-linux-real-evidence.py --receipt FILE --base DIRECTORY
A valid failure receipt is retained unchanged: capture success is not a passing
qualification. The controller must check the worker exit and receipt separately.
No installer, subprocess, extraction, filesystem writes or privilege escalation.
"""
import argparse
import io
import json
import os
import pathlib
import re
import stat
import sys
import tarfile

MAX_BYTES = 16 << 20
MAX_LOGS = 128
LOG_NAME = re.compile(r'[0-9]{2,4}-[a-z0-9-]{1,80}\.(out|err)\Z')


def require(condition, message):
    if not condition:
        raise ValueError(message)


def directory(path, parent=None):
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=parent)
    info = os.fstat(fd)
    if info.st_uid != os.getuid() or stat.S_IMODE(info.st_mode) != 0o700:
        os.close(fd)
        raise ValueError('evidence directory ownership or mode differs')
    return fd


def owned_bytes(name, parent, remaining):
    fd = os.open(name, os.O_RDONLY | os.O_NOFOLLOW, dir_fd=parent)
    try:
        before = os.fstat(fd)
        require(stat.S_ISREG(before.st_mode) and before.st_uid == os.getuid()
                and before.st_nlink == 1 and stat.S_IMODE(before.st_mode) == 0o600,
                'evidence must be an owned exclusive 0600 regular file')
        require(0 <= before.st_size <= remaining, 'evidence byte bound exceeded')
        with os.fdopen(fd, 'rb', closefd=False) as handle:
            data = handle.read(remaining + 1)
        after = os.fstat(fd)
        identity = lambda value: (value.st_dev, value.st_ino, value.st_size, value.st_mtime_ns, value.st_ctime_ns)
        require(identity(before) == identity(after) and len(data) == before.st_size,
                'evidence changed during read')
        return data
    finally:
        os.close(fd)


def capture(receipt, base):
    require(os.getuid() != 0, 'evidence collector must not run as root')
    require(receipt.is_absolute() and base.is_absolute(), 'evidence paths must be absolute')
    require(receipt.parent.resolve() == receipt.parent and base.resolve() == base,
            'evidence parent paths must be canonical')
    parent = directory(str(receipt.parent))
    try:
        data = owned_bytes(receipt.name, parent, 1 << 20)
    finally:
        os.close(parent)
    payload = json.loads(data)
    require(isinstance(payload, dict), 'receipt must be a JSON object')
    require(payload.get('schema') == 'gentle-shell-linux-qualification/v1', 'receipt schema differs')
    require(payload.get('result') in ('pass', 'fail'), 'receipt result missing')
    require(re.fullmatch(r'[0-9a-f]{40}', payload.get('sourceSHA', '')) is not None, 'source binding missing')
    members = [('receipt.json', data)]
    if 'root' not in payload:
        require(payload['result'] == 'fail', 'passing receipt has no qualification root')
        return members
    root = pathlib.Path(payload['root'])
    require(root.parent == base and re.fullmatch(r'gentle-linux-qualification\.[a-zA-Z0-9_-]+', root.name),
            'qualification root is outside the expected base')
    base_fd = directory(str(base))
    try:
        root_fd = directory(root.name, base_fd)
        try:
            logs_fd = directory('logs', root_fd)
            try:
                total = len(data)
                names = []
                with os.scandir(logs_fd) as entries:
                    for entry in entries:
                        require(len(names) < MAX_LOGS, 'command log count bound exceeded')
                        names.append(entry.name)
                for name in sorted(names):
                    require(LOG_NAME.fullmatch(name) is not None, 'unexpected command log name')
                    content = owned_bytes(name, logs_fd, MAX_BYTES - total)
                    members.append(('logs/' + name, content))
                    total += len(content)
            finally:
                os.close(logs_fd)
        finally:
            os.close(root_fd)
    finally:
        os.close(base_fd)
    return members


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--receipt', required=True, type=pathlib.Path, help='owned new qualification receipt')
    parser.add_argument('--base', required=True, type=pathlib.Path, help='canonical private qualification base')
    arguments = parser.parse_args()
    try:
        # Validate/read everything before emitting any archive bytes.
        members = capture(arguments.receipt, arguments.base)
    except (OSError, ValueError, TypeError) as error:
        sys.stderr.write('EVIDENCE-FAIL: %s\n' % str(error)[:300])
        return 2
    with tarfile.open(fileobj=sys.stdout.buffer, mode='w|', format=tarfile.USTAR_FORMAT) as archive:
        for name, data in members:
            info = tarfile.TarInfo(name)
            info.mode, info.size = 0o600, len(data)
            archive.addfile(info, io.BytesIO(data))
    return 0


if __name__ == '__main__':
    sys.exit(main())
