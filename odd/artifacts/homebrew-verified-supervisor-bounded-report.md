# Verified-supervisor bounded verification report

## Disposition

Closed as a bounded evidence report under the human's explicit choice:
"Cerrar informe acotado, sin más CI — recomendado".
This is **not** global cross-platform qualification, official Homebrew/release
qualification, native review approval, or permission for a PR/merge/release.

The broader T6 completion was dropped, not passed. T15 consolidates the valid
results below; it does not turn historical failures or cancellations green.

## Valid evidence retained

| Area | Observed evidence | Boundary |
| --- | --- | --- |
| Linux lock context | CI `45471264024`: focused lock check and full installer package, 1,406 PASS events / 34 SKIP / 0 FAIL | Normal CLI inherited-NNP context; not arbitrary multithreaded Go callers |
| Linux real suppliers | CI `45489583261`, source `62d38358a`: 18 checkpoints, 40 original command outputs, 106.2s qualification; Separate and Shared modes | Private-source supplier laboratory; not official Homebrew/bootstrap/release distribution |
| Linux laboratory contracts | 31 worker contracts passed, including the YAML-dependent contract unavailable on the host | Existing worker execution, not a new run for this report |
| macOS real installation | Historical VM job 49: 18 checkpoints; launch, provenance/refusal/recovery/residue and 17 real-HOME watches | Isolated private installation, not official Homebrew/signature/release/uninstall proof |
| macOS CLI suite | Corrected VM job 73: 2,048 top-level tests / 4,237 PASS events / 18 SKIP / 0 FAIL | Corrected isolated environment and test fixture; prior timeouts retained |
| macOS release security | Corrected VM: 14 PASS events for the two relevant security tests using GNU Bash 5.3.20; audited relevant sources unchanged | Valid evidence reuse; no new native CI execution |
| macOS bench module | Baseline job `138596782380` in CI `45436798212` passed; bench module unchanged | Module build/vet/unit proof only, not root-driven journeys against changed CLI |
| Windows Go suites | All 22 Windows Go shards passed in historical CI `45436798212` | Go test coverage, not actual Windows installation |
| Resource admission policy | Source `3e23beac2`: 11 local tests, eight real shell-guard cases, eight YAML parses and 34 Bash syntax blocks passed; independent static reviewer `mww4wcmj-1c-spyo` found S22 met/no blockers | Static independent review; execution evidence belongs to the parent, not independently rerun |

No Go/Bash/Homebrew host installation, production trust relaxation, SSH/profile
change or cloud resource was needed for this closure.

## Failures, skips and unavailable proof remain visible

- Historical full matrix `45436798212`: **27 PASS / 4 FAIL / 2 cancelled**.
  This remains its result, not a fresh global green matrix.
- Both Windows bench roles had the same 19 failures, separated from this fix
  under S17. They are not fixed or waived into passes by this report.
- The historical Linux root lock failure is retained. The later inherited-NNP
  fixture correction and installer-package green are narrower subsequent proof.
- Direct NNP diagnostic `45468283155` was RED for both baseline and candidate:
  caller-thread NNP and leader status differ. The arbitrary multithreaded Go API
  limitation remains; production guards were not weakened to hide it.
- Initial real Linux attempt `45485821232` failed at the fixture-port
  prerequisite before build/install. The separately authorized physical-mode
  correction passed in `45489583261`; it does not erase the first attempt.
- macOS baseline release-security failures under Apple Bash and the historical
  CLI timeouts remain recorded. Corrected VM evidence is separate. Historical
  candidate macOS root/bench cancellation is not reclassified as a pass.
- Redundant new macOS execution T11 was dropped after evidence reuse was
  established. Registration absence and HTTP 422 Actions-disabled refusal were
  not native test executions.
- Earlier independent-review timeouts remain unavailable verdicts. The later
  completed bounded static reviews do not retrospectively approve those runs.

## Published resource-control checkpoint

Fork: `dcalderonlo/gentle-ai`.
Branch: `verification/ci-resource-policy-3e23beac`.
Exact source: `3e23beac2fa3d8fc7110cd95f5faa81ec0fc2aae`.

Eight task-owned diagnostics have no automatic push triggers at this revision;
four matrices require false-by-default explicit confirmation. Upstream CI and
the pre-existing Linux lane, runtime commands, permissions, limits, matrices and
provenance checks were preserved. Only the exact new ref was pushed normally.
No PR, merge, dispatch, force push or hook bypass occurred.

The preflight examined 21 candidate and 13 default-branch workflows, finding
zero matching push/create triggers and no existing PR for the new ref. The
single post-push snapshot at `2026-11-25T13:47:51.994686+00:00` confirmed the exact
remote source, unchanged main `4716b3cbca158c35c3b316dbd517f54e5f240f97`, and
**zero Actions runs for that branch/SHA**. It was not a polling loop.

This policy is present on the new ref only. Historical branches and repository
settings were not rewritten; do not reuse old refs for new diagnostic pushes.

## Explicitly pending outside this closure

- Real Windows installation: NOT RUN. Existing external guide:
  `docs/gentle-shell-windows-validation.md`; no paid infrastructure was created.
- A fresh complete root/platform matrix: not claimed or automatically rerun.
- Official Homebrew/authenticated bootstrap, published release identity/signature
  and uninstall qualification: not established by the private laboratory.
- General arbitrary Go thread-local NNP/leader-context limitation: unchanged.
- Historical native review `review-f797b64bcd632051`: remains unapproved; clone
  RDD was not re-enabled. Static CI-policy review is not native review approval.

## Evidence and verification of this report

Use `odd/tasks/homebrew-verified-supervisor.md` for the authorization/history,
`odd/artifacts/ci-resource-necessity-audit.json` for proof-reuse scope,
`odd/artifacts/diagnostic-resource-policy-independent-review.md` for the static
review verdict, and `odd/artifacts/diagnostic-resource-policy-publication.json`
for the preflight/post-push record. Original receipts, logs and failures remain
in their recorded artifact locations.

This is passive consolidation of existing evidence: no meaningful RED applies.
Only structural/readback checks apply to this document; no functional tests,
CI jobs, native review, deployment or additional publication are requested by it.
