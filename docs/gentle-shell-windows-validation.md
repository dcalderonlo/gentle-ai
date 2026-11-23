# Validate the Windows installation candidate

**Windows installation remains unverified.** This is a manual handoff for a
reviewer with a disposable Windows client laboratory. No Windows installation,
interactive UI, cancellation, or console-restoration PASS is claimed here.
Windows CI compilation, Go tests, smoke checks and benchmark-module tests remain
useful, but Windows Server runners do not qualify this client-only installer.

## Prepare a safe laboratory

Use Windows 10 22H2 (build 19045+) or Windows 11 **native x64**, a normal
unelevated terminal and a private local NTFS destination. Windows Server and
ARM64 are refused; x64 application emulation on ARM64 does not qualify.

Use a dedicated disposable account/VM without personal credentials or data.
Take a recoverable VM snapshot before testing. Do not change UAC, antivirus,
ACL guards, production verification keys, machine PATH or shell profiles to
make a refusal disappear. Preserve and report refusals instead.

Obtain a Windows **product** `gentle-ai.exe` built from the intended source.
A `.test.exe` or benchmark executable is not an installer candidate. Record its
full source commit and SHA-256. For uncommitted code, also record the immutable
source tree and overlay digests; do not label the base commit as the tested code.
An unsigned CI build is not a signed official release or evidence of release
signature verification. Stop if Windows security blocks execution; do not bypass
that protection as part of this guide.

Read [the installer boundaries](gentle-shell-windows-install.md) before proceeding.

## Isolate the operator environment

Run the following in **PowerShell inside the disposable Windows laboratory**,
not on a personal workstation. Replace the candidate path and expected digest.
These settings affect only this process and its children. Close this terminal
after testing rather than persisting them with `setx` or editing profiles.

```powershell
$Candidate = 'C:\path\to\reviewed\gentle-ai.exe'
$ExpectedSha256 = '<64 hexadecimal characters from the build manifest>'
if ($ExpectedSha256 -notmatch '^[0-9a-fA-F]{64}$') { throw 'Set the expected digest first' }
if ((Get-FileHash -Algorithm SHA256 -LiteralPath $Candidate).Hash -ne $ExpectedSha256) {
    throw 'Candidate digest mismatch'
}

$OriginalLocalAppData = $env:LOCALAPPDATA
$Lab = Join-Path $OriginalLocalAppData ('GentleShellValidation-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $Lab -ErrorAction Stop | Out-Null
foreach ($Name in @('home', 'roaming', 'local', 'tmp', 'cache', 'logs')) {
    New-Item -ItemType Directory -Path (Join-Path $Lab $Name) -ErrorAction Stop | Out-Null
}
$env:HOME = Join-Path $Lab 'home'
$env:USERPROFILE = $env:HOME
$env:APPDATA = Join-Path $Lab 'roaming'
$env:LOCALAPPDATA = Join-Path $Lab 'local'
$env:TEMP = Join-Path $Lab 'tmp'
$env:TMP = $env:TEMP
$env:TMPDIR = $env:TEMP
$env:XDG_CONFIG_HOME = Join-Path $Lab 'roaming'
$env:XDG_CACHE_HOME = Join-Path $Lab 'cache'
$env:XDG_DATA_HOME = Join-Path $Lab 'local'
$env:XDG_STATE_HOME = Join-Path $Lab 'local'
$env:ENGRAM_STARTUP_POLICY = 'disabled'
$Target = Join-Path $Lab 'installation'
```

Environment redirection is **not a filesystem sandbox**: native Windows APIs can
resolve real account folders independently of these variables. That is why a
disposable account is required. Check both redirected folders and that account's
actual profile for unintended changes. Launchers may be published beneath
`LOCALAPPDATA\Microsoft\WindowsApps`; do not assume `--target` bounds all writes.
Memory is deliberately disabled to avoid starting or reusing a personal Engram
store. This test does not qualify memory startup or personal-data migration.

## Inspect, then confirm the exact selection

Keep the same terminal, candidate, target, account and environment throughout.
Use Separate and Stable only; do not substitute Shared, update or recovery flags.

```powershell
& $Candidate shell install --target $Target --mode separate --channel stable --inspect `
    1> (Join-Path $Lab 'logs\inspect.stdout') 2> (Join-Path $Lab 'logs\inspect.stderr')
$InspectExit = $LASTEXITCODE
Get-Content -LiteralPath (Join-Path $Lab 'logs\inspect.stdout')
Get-Content -LiteralPath (Join-Path $Lab 'logs\inspect.stderr')
if ($InspectExit -ne 0) { throw "Inspect refused: exit $InspectExit; retain the logs" }
```

Inspect must return a `Confirmation:` value. Record the exit code, exact stdout
and stderr, and whether the target/profile changed before confirmation. Inspect
success is not installation success. If the selection changes, inspect again;
never reuse another target, mode, channel or experience profile's token.

```powershell
$Confirmation = '<paste this inspection confirmation>'
if ($Confirmation -notmatch '^[0-9a-fA-F]{64}$') { throw 'Invalid confirmation format' }
& $Candidate shell install --target $Target --mode separate --channel stable --confirm $Confirmation `
    1> (Join-Path $Lab 'logs\install.stdout') 2> (Join-Path $Lab 'logs\install.stderr')
$InstallExit = $LASTEXITCODE
Get-Content -LiteralPath (Join-Path $Lab 'logs\install.stdout')
Get-Content -LiteralPath (Join-Path $Lab 'logs\install.stderr')
if ($InstallExit -ne 0) { throw "Installation failed: exit $InstallExit; preserve its paths" }
```

A timeout or uncertain stop is a failure, not permission to delete matching stage
folders. Preserve reported workspaces and ownership evidence. Do not overwrite an
occupied destination, remove another process's files, or rerun blindly.

## Verify the installed bindings

From the same isolated environment, run each command separately and retain its
exit code, exact stdout and stderr:

```powershell
& (Join-Path $Target 'bin\gentle-shell.cmd') --version
$LASTEXITCODE
& (Join-Path $Target 'bin\gentle-shell.cmd') --help
$LASTEXITCODE
& (Join-Path $Target 'bin\pi.cmd') --version
$LASTEXITCODE
& (Join-Path $Target 'bin\pi.cmd') --help
$LASTEXITCODE
```

Then open each binding without arguments in a normal interactive console,
**without output redirection**, one at a time. Do not enter provider credentials
or send model requests. Record whether the authentic UI opens, how you exit,
whether the child settles, and whether the caller's CWD, foreground, cursor,
console modes and subsequent typing are restored. Version/help output alone
cannot establish these UI/terminal properties.

The public no-flags installer TUI also needs its own observation:
`& $Candidate shell install`. Review/cancel before confirmation unless using a
second fresh owned target. The command-line installation above does not prove
the TUI's selection/confirmation or cancellation behavior.

## Return a bounded report

| Evidence | Record |
|---|---|
| Candidate | Full source identity, overlay/tree if applicable, executable SHA-256 and build origin |
| Host | Windows edition/build, native architecture, normal token, NTFS and account isolation |
| Installation | Target/mode/channel/profile, inspected selection, exact command exits/stdout/stderr |
| Runtime | Installed manifest/lock identities, observed versions, both authentic UI openings |
| Preservation | Before/after profile/PATH/CWD/console observations and reported residue |
| Failures | First error, checkpoint, duration, preserved workspaces and owner/process uncertainty |

Mark every item **PASS, FAIL, SKIP or NOT RUN**. A single machine qualifies only
its observed edition/build, not both Windows 10 and 11 or other architectures.
Keep Shared, update, recovery, uninstall, real-model use, memory startup, timeout
and Ctrl-C cancellation **NOT RUN** unless separately exercised under an agreed
plan. Do not invent cleanup/uninstall commands or equate recovery with uninstall.

Redact credentials, private keys and personal data before sharing evidence.
Never upload the temporary SSH/private access material used for other platforms.
Retain the laboratory until its owner can review failures; use the owner's VM
snapshot/retirement process rather than deleting uncertain installer data.
