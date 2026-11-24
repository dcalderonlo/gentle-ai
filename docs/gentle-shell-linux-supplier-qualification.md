# What does the clean Linux supplier lane prove?

It exercises a private-source candidate against actual upstream suppliers on
an ephemeral Ubuntu x64 runner. It is **not** an official release, signed
Homebrew bootstrap, uninstall or Windows installation qualification.

## Before dispatch

The lane is `.github/workflows/shell-linux-real-suppliers.yml`. It accepts only
manual dispatches in `dcalderonlo/gentle-ai` on literal `verification/*` branches.
Push events register the workflow but never run its privileged laboratory.
The case-sensitive shell gate runs before source fetch or preparation.

Each authorized attempt is one dispatch, at most 60 minutes. Failure, timeout
or missing evidence stays a failure or gap: do not retry, raise budgets, skip
guards or rerun the repository matrix to manufacture GREEN.

## What stays isolated?

The fresh runner provisions UID/GID 1002, a real delegated user manager, empty
capabilities and NNP before Python, Go or installer execution. Worker limits
remain one CPU, 3 GiB memory, no swap and 512 tasks. Build and installation run
inside that worker, not as the privileged preparer. The existing native
coordinator gate is not enabled.

The clean lane uses the harness's `physical` mode: every UID, capability, NNP,
cgroup and read-only-root check remains mandatory. Existing `probe`, `direct`,
`full` and `smoke` fixture modes still require their original local TLS-listener
prerequisite. That fixture-only prerequisite must not change the clean VM's
privileged-port policy. The first attempt exposed this coupling and failed
before build/install; its RED evidence is retained, not converted into a pass.

There is no laboratory CA, supplier-host override, TLS interceptor or test
signing-key injection. The system CA bundle is hashed before and checked after
qualification. Go uses a checksum-verified isolated toolchain; the candidate
uses its default supplier trust. No developer-host installation is changed.

## What is checked?

The historical `e2e/shell-macos-qualification.py` filename now also supports
Linux x86_64. Darwin settings remain separate. Linux requires its inherited
user-manager bus/runtime, NNP, empty capabilities and private `/work/` TMPDIR.
Its whole-run deadline is 1,260 seconds, inside the existing 1,330-second
worker ceiling, with time for failure settlement and receipt publication.

The qualifier checks Separate and Shared installations, real launches, exact
Node/tool pins, the authenticated SDK installer's native-member row, resolved
SDK integrity, refusal without mutation, token-bound recovery, retained
quarantines, launcher bindings, residue and real-HOME preservation. SDK
`stable/latest` is recorded as resolved, not misrepresented as a repository pin.

## How is evidence retained?

The artifact `linux-real-suppliers-<source SHA>` retains source/candidate hashes,
worker exit codes, raw worker output, receipt, summary and original command
stdout/stderr. The unprivileged `e2e/shell-linux-real-evidence.py` reader accepts
only stable, owned, exclusive 0600 regular files; it rejects links and unsafe
modes before emitting anything. Its regular-file-only tar is bounded to 128 command logs and 16 MiB
of payload. Capture runs in its own bounded worker; the tar is never
automatically extracted by the controller.

A successful **capture** can contain a failed qualification: the original
worker exit and receipt result remain separate. Missing or withheld evidence
is not proof. A passing lane establishes only these observed checks; existing
Windows bench failures, cancelled macOS jobs and unavailable independent review
must still be reported separately. See `odd/tasks/homebrew-verified-supervisor.md`
for the current execution and verification state.
