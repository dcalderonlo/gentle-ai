# Diagnostic CI resource policy

## No automatic diagnostic runs

The eight workflows added for verified-supervisor development are manual-only
at this revision. No `push`, pull-request or schedule event starts them. The
macOS gap lane also supports a same-ref reusable call from its guarded manual
entry. Upstream `ci.yml` and the pre-existing `shell-linux-first-ci.yml` are
unchanged.

## Confirm matrices deliberately

`confirm_matrix` is a boolean defaulting to **false** for acquisition,
initialization differential, NNP diagnosis and repository-suite matrices.
Without explicit confirmation their matrix admission jobs are skipped. The
full repository diagnostic can allocate 33 jobs: confirm its need and source
before selecting it. Existing test commands, limits, platform coverage,
provenance checks and permissions are preserved; this is resource admission,
not a reduced test suite.

The repository entry supports two mutually exclusive manual choices:

| Inputs | Result |
| --- | --- |
| Neither flag true | No diagnostic runners |
| `confirm_matrix=true`, `macos_gaps_only=false` | Full existing repository matrix |
| `macos_gaps_only=true`, `confirm_matrix=false` | Single guarded macOS gap lane |
| Both true | Neither branch runs |

The NNP and lock diagnostics retain their original exact lowercase branch
scope, checked before checkout. Linux lab guards and source binding remain
unchanged. Manual admission is not permission to replay an expired or
human-bounded attempt.

## Reuse valid proof before spending another run

Do not repeat checks merely to replace a cancelled job label. The necessity
audit records existing macOS VM release-security proof and unchanged bench
module proof in `odd/artifacts/ci-resource-necessity-audit.json`. Reuse is bounded
to those unchanged inputs; it does not repaint historical CI failures or prove
new root-driven journeys.

Validate this policy with local contracts and independent read-only review,
not native CI dispatches. Source restrictions apply to revisions containing
them: they do not rewrite
old branches or change GitHub repository settings. Publication to a feature or
default branch remains separately authorized; this document grants no delivery
permission.
