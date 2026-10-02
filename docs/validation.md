# Validation

`tools/validator` renders every element under `stack/` for each fixture cluster in
`tools/validator/testdata/clusters`, building the list of values files the way the ApplicationSet
does ([Order](values.md#order)). It checks the element sources and the rendered output. The
fixtures carry the labels of the example clusters: the hub (`local-cluster` in `dc1`, which reads
the `acm-dc1` files), `prod-east-1` and `nonprod-west-1`. Their files in `values/` are rendered too.

It needs Go and Helm. Use the Helm major version that CI pins
([Policy Validate](workflows.md#policy-validate)). kubeconform is optional; without it on `PATH`,
POLICY080 is skipped.

## Running it

From the repo root, the same commands CI runs, without `--github`:

```sh
make -C tools/validator lint test build
tools/validator/bin/policystack-validator --repo-root . --extra-values tools/validator/testdata/baseline.yaml --skip POLICY080
```

The first checks formatting, vets, tests and builds the validator. The second runs it.
[Policy Validate](workflows.md#policy-validate) explains the POLICY080 skip.

`baseline.yaml` supplies a stand-in `selector`: one `Exists` expression on the `name` label. The
chart builds each Placement from `selector.matchExpressions`, and outside Argo CD nothing injects
it, so `helm template` fails for an enabled element without one. In production the ApplicationSet
injects a selector for the one cluster instead ([Placement](policies.md#placement)). Files passed
with `--extra-values` take precedence over the whole cascade.

The validator runs `helm dependency update` on an element only when its `charts/` directory holds no
`.tgz`; otherwise it renders with the archive already there. After bumping the policy-library
version, delete the stale archives with `rm -rf stack/*/charts`. A CI checkout has none.

## Output

A clean run prints `ok: no findings`. Otherwise each finding gives its severity, rule, element (and
fixture cluster, for checks on rendered output), and the file to fix, with the line where known:

```text
error POLICY020 [metallb] configPolicies[0] "metallb-instance" references templateNames[0] "metallb-crd" but converters/metallb-crd.yaml does not exist
    at /path/to/policystack/stack/metallb/values.yaml:107:19
error POLICY030 [kiali] policies[0].severity = "hgh"; allowed: [low medium high critical]
    at /path/to/policystack/stack/kiali/values.yaml:22:19
warning POLICY021 [metallb] converters/metallb-cr.yaml is not referenced by any templateNames[].name
    at /path/to/policystack/stack/metallb/converters/metallb-cr.yaml:1

2 error(s), 1 warning(s)
```

| Exit code | Meaning |
| --------- | ------- |
| `0` | No errors; warnings alone pass |
| `1` | Errors, or warnings with `--severity warning` |
| `2` | The validator itself failed, for example Helm is not on `PATH` |

| Flag | Effect |
| ---- | ------ |
| `--only` | Comma-separated rule IDs to run; all others are skipped |
| `--skip` | Comma-separated rule IDs to skip |
| `--severity warning` | Fail on warnings as well as errors |
| `--github` | Write GitHub Actions annotations instead of terminal output |

`--help` lists the rest. The validator does not read the appset chart's `baseDomain`: `--base-domain`
sets the label prefix it reads from the fixtures and defaults to `example.com`, the fixtures' domain.

## Rules

--8<-- "tools/validator/README.md:rules"
