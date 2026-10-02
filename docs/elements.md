# Writing an element

An element is a Helm chart under `stack/` that renders ACM policies through the
[policy-library](https://github.com/PolicyStack/PolicyStack-chart/blob/main/charts/policy-library/README.md)
chart. Create one from the repo root with `tools/create-element.sh`, which copies `sample-element/`
under the name you give it ([usage](https://github.com/PolicyStack/PolicyStack/blob/main/tools/README.md#create-elementsh)).

Name it in lowercase kebab-case and keep the directory name and chart `name` identical. The
Application, and so the Helm release, takes the directory name
([Applications](applicationset.md#applications)). The values key comes from the chart name.
`element:` dependencies and the validator assume the two match.

## Layout

| Path | Contents |
| ---- | -------- |
| `Chart.yaml` | Name, description and the `policy-library` dependency |
| `templates/policy.yaml` | One line: `{{- include "policy-library.render" . -}}` |
| `values.yaml` | Defaults for every setting, under the key described in [Element values](policies.md#element-values) |
| `converters/<name>.yaml` | The manifest for one `configPolicies[].templateNames` entry, rendered with Helm's `tpl`. It reads `.Values`, and `.Parameters` (that config policy's `templateParameters`) when the config policy sets `enableTemplateParameters` |
| `README.md` | Generated [element reference](#element-reference) |
| `.gitignore` | Keeps `Chart.lock`, `charts/` and `*.tgz` out of Git |

## Conventions

The examples come from [`stack/metallb`](https://github.com/PolicyStack/PolicyStack/tree/main/stack/metallb).
Its `values.yaml`, trimmed:

```yaml
stack:
  metallb:
    enabled: false
    toggles:
      quota: false
      addressing: false
    config:
      ipAddressPools: {}
      l2Advertisements: {}
      # bgpAdvertisements, bgpPeers, quota

    policies:
      - name: install
        enabled: true
      - name: config
        enabled: true
        dependencies:
          - name: install
      # quota
      - name: addressing
        enabled: true
        dependencies:
          - name: config

    operatorPolicies:
      - name: metallb
        enabled: true
        policyRef: install
        # namespace, operatorGroup, subscription

    configPolicies:
      - name: metallb-instance
        enabled: true
        policyRef: config
        waitForOperator: metallb
        ignorePending: true
        templateNames:
          - name: metallb-cr
      # resourcequota
      - name: pools
        enabled: true
        policyRef: addressing
        rawTemplate: true
        templateNames:
          - name: metallb-ipaddresspools
      # l2, bgp, peers
```

**Off by default.** The element ships with `enabled: false`. A layer in `values/` turns it on for the
clusters that need it ([Values cascade](values.md)).

**Toggles for optional pieces.** The `quota` and `addressing` policies ship toggled off, so a
cluster turns one on in one line instead of restating the `policies` list ([toggles](https://github.com/PolicyStack/PolicyStack-chart/blob/main/charts/policy-library/README.md#toggles)).

**A `config` map for site data.** Converters read addresses, peers and quotas from
`.Values.stack.metallb.config` rather than hard-coding them. Maps merge across the cascade
([Merge rules](values.md#merge-rules)), so a cluster file sets only its own keys.
`values/clusters/nonprod-west-1.yaml`:

```yaml
stack:
  metallb:
    enabled: true
    toggles:
      addressing: true
    config:
      ipAddressPools:
        services:
          addresses:
            - 10.20.30.100-10.20.30.120
          autoAssign: true
      l2Advertisements:
        services:
          ipAddressPools:
            - services
```

**Install, then configure.** The operator and the objects that need its CRDs live in separate
policies. `config` depends on `install`, and `addressing` depends on `config`, so ACM holds each
policy `Pending` until the one before it is compliant. `waitForOperator: metallb` makes
`metallb-instance` also wait for the operator's CSV, and `ignorePending: true` keeps that template's
`Pending` state from making its policy noncompliant
([policy dependencies](https://github.com/PolicyStack/PolicyStack-chart/blob/main/charts/policy-library/README.md#policy-dependencies),
[waiting for an operator](https://github.com/PolicyStack/PolicyStack-chart/blob/main/charts/policy-library/README.md#waiting-for-an-operator)).

**`rawTemplate` when the object count varies.** The pool, advertisement and peer converters range
over a `config` map and emit one object per key, so they set `rawTemplate: true`. An empty map
renders an empty list
([raw object templates](https://github.com/PolicyStack/PolicyStack-chart/blob/main/charts/policy-library/README.md#raw-object-templates)).

**Short names.** Element names, policy names and the names of their config and operator policies
end up in length-limited object names ([Naming limits](policies.md#naming-limits)).

Run the [validator](validation.md) before opening a pull request.

## Element reference

`stack/<element>/README.md` is generated from the element's `values.yaml` and its `@description` and
`@desc` comments; do not edit it by hand. After changing `values.yaml`, run
`python tools/doc-generator.py` and commit the README; `--check` reports stale READMEs without
writing them. Flags and comment syntax are in
[tools/README.md](https://github.com/PolicyStack/PolicyStack/blob/main/tools/README.md#doc-generatorpy).

CI runs the check on pull requests, and the `update-docs` label regenerates the READMEs on the pull
request branch ([Documentation Check](workflows.md#documentation-check)).
