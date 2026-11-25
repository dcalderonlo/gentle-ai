# Missing-only macOS CI verification

`.github/workflows/supervisor-macos-gaps.yml` addresses two failed release-security
tests and the previously unstarted candidate macOS bench module. It does not
rerun the repository matrix or claim a new complete root-suite pass.

## Why an isolated Bash?

The recorded baseline run selected Apple Bash 3.2. The release scripts require
features such as `[[ -v ... ]]` and associative arrays. The tests invoke `bash`
through PATH, including the real canonical-public-key linker build.

The lane verifies the pinned GNU Bash 5.3 source archive checksum before building
it into a private `RUNNER_TEMP` prefix. Its next steps assert that exact binary
is selected. It does not change Homebrew, system Bash, profiles, production
scripts, release keys or security-test assertions. Nothing is installed on the
developer's host.

## Registered entry

When GitHub has not registered the new standalone lane, manually dispatch the
already-registered `Supervisor repository suites` workflow with boolean input
`macos_gaps_only=true` on the authorized verification branch. It calls this
lane from the **same commit**, without publishing the workflow on `main`.
The baseline resolver and all dependent matrix jobs are skipped only for this
explicit opt-in; an omitted/false input preserves the ordinary matrix exactly.
The callee still checks the exact owned-fork/manual branch before source fetch.

## Scope and limits

- Owned fork, manual dispatch on literal `verification/macos-gaps-*` only.
- Exact source SHA and native non-root Darwin/arm64 admission before checks.
- One job, original 40-minute ceiling; step ceilings total 37 minutes.
- Only the two previously failed release-security tests, then candidate bench
  build, vet and tests. Go package timeout defaults remain unchanged.
- Raw build/test outputs and original exit codes retained on failure or success.
- No automatic retry, runner migration, new shards, timeout increase or test
  omission to turn RED into GREEN.

The old CLI package hit its 10-minute cumulative alarm while its active test
had run for only two seconds. This does not prove that individual test hung.
The separately recorded complete VM CLI pass remains evidence; this lane does
not replay it or replace the old timeout record. Cancelled jobs with no steps
remain cancelled historical attempts, not test successes.

Current outcomes, source identities and remaining proof boundaries are recorded
in `odd/tasks/homebrew-verified-supervisor.md` and `odd/artifacts/`.
