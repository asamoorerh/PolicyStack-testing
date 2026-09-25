# CI Workflows

Three GitHub Actions workflows in `.github/workflows/`. The two checks run on pull requests and on
pushes to `main`; the third fixes stale docs on request.

| Workflow | Runs on | Fails when |
| -------- | ------- | ---------- |
| Policy Validate (`validate.yml`) | Changes to `stack/`, `values/`, `values.yaml`, `tools/validator/` | Validator tests fail or the validator reports an error |
| Documentation Check (`docs-check.yml`) | Changes to `stack/`, `docs/`, the doc generator or its tests | Generator tests fail or `docs/` does not match the values files |
| Update Documentation (`docs-update.yml`) | `update-docs` label added to a PR | Not a gate; it fixes what Documentation Check reports |

Both checks also run from the Actions tab (`workflow_dispatch`). A newer push to the same PR cancels
the run in progress.

## Policy Validate

1. Checks gofmt, vets, tests and builds `tools/validator` (`make -C tools/validator lint test build`)
   with the Go version from `tools/validator/go.mod`.
2. Runs the validator against the fixture clusters with `--github`, so each finding is annotated on
   the PR diff at the file and line to fix. Errors fail the job; warnings are annotated only.

Helm is pinned by `HELM_VERSION` in the workflow. Keep it on the major the hub's Argo CD uses.
POLICY080 is skipped: the chart only emits Policy, PlacementBinding and PlacementRule, which have no
kubeconform schemas.

To reproduce locally, see [Validation](../README.md#validation) in the README.

## Documentation Check

1. Runs the doc generator tests (`python -m unittest tools/test_doc_generator.py`).
2. Runs `python tools/doc-generator.py --check`, which regenerates every page in memory and compares
   it with `docs/`, ignoring the `Generated:` timestamp.

If `docs/` is stale, the `Verify Documentation` check fails with an annotation saying how to fix it:

- run `python tools/doc-generator.py` and commit `docs/`, or
- add the `update-docs` label to the PR.

## Update Documentation

Runs when the `update-docs` label is added to a PR from a branch in this repository.

1. Regenerates `docs/` from the PR branch. Pages whose content is unchanged are not rewritten, so the
   commit only touches pages that changed.
2. If anything changed, commits it as `github-actions[bot]` with the message
   `docs: update documentation [auto-generated]` and pushes to the PR branch.
3. A push made with `GITHUB_TOKEN` does not start workflows, so it dispatches Documentation Check and
   Policy Validate on the branch. Both report real results on the new commit.
4. Removes the label.

PRs from forks are skipped: their token is read-only. Run the generator locally instead.

## Dependencies

Actions are pinned to commit SHAs, with the version in a trailing comment. Dependabot
(`.github/dependabot.yml`) opens weekly PRs for the actions, the validator's Go modules and
`tools/requirements.txt`. The Helm version in `validate.yml` is not covered and is updated by hand.
