# Clusters and Applications

The ApplicationSet `policystack` in `openshift-gitops` on the hub creates one Argo CD Application per element for every managed cluster that carries a revision label. The cluster's labels pick the Git revision, the values files and the Placement selector. The template is [`appset/templates/appset.yaml`](https://github.com/PolicyStack/PolicyStack/blob/main/appset/templates/appset.yaml).

## Cluster labels

Labels go on the cluster's ManagedCluster on the hub. The GitOpsCluster copies them to the cluster's Argo CD secret, where the ApplicationSet reads them ([Generators](#generators)). Label keys use the appset chart's `baseDomain`, `example.com` here ([The appset chart](#the-appset-chart)).

| Label | Required | Effect |
|---|---|---|
| `git.example.com/revision` | Yes | Branch or tag every Application for the cluster renders from. Without it the cluster gets no Applications. Label values cannot contain `/`, so a branch with `/` in its name cannot be used. Changing it promotes or rolls back ([Promoting and rolling back](rollout.md#promoting-and-rolling-back)). |
| `clusterID` | Yes. ACM sets it on OpenShift clusters | Becomes `selectedId`. The template uses `missingkey=error`, so a cluster without it fails to render. One failed render stops the whole ApplicationSet. Argo CD creates, updates and deletes no Applications for any cluster until the label is set. |
| `config.example.com/<category>.<priority>=<value>` | No | Adds `values/<category>s/<value>.yaml` to the cascade and an `In` expression for this label to the Placement selector. The priority orders the files ([Order](values.md#order)). |
| `config.example.com/datacenter.<priority>=<datacenter>` | On the hub | A config label like any other. On the hub it also names the Applications and the hub values files. |
| `local-cluster` | ACM sets it on the hub | Switches the cluster to hub naming and hub values files. |
| `name` | ACM sets it | The ManagedCluster name. The Placement selector matches on it ([Placement](policies.md#placement)). |
| `env.example.com/defaultstorageclass` | No | Becomes `selectedDefaultStorageClass`. |

Categories are dynamic. `config.example.com/tenant.40=payments` adds `values/tenants/payments.yaml`. The repo's priorities are environment `.10`, datacenter `.20`, platform `.30` and tenant `.40`. [Onboarding a cluster](rollout.md#onboarding-a-cluster) labels `prod-east-1` with them.

Hub Application names and hub values files are keyed by datacenter, so hubs in the same datacenter share hub values files.

## Generators

A matrix generator combines:

1. A clusters generator. Every Argo CD cluster secret with a `git.example.com/revision` label.
2. A git generator. Every directory under `stack/` in `gitRepo`, at that cluster's revision.

Each cluster and directory pair becomes one Application. The directory list comes from the cluster's own revision, so an element added on one branch reaches only clusters on a revision that contains it. Every directory gets an Application whether the element is enabled or not. A disabled element renders no objects.

The GitOpsCluster from [Import clusters into Argo CD](install.md#import-clusters-into-argo-cd) creates one cluster secret per managed cluster. Clusters without the revision label are imported but get no Applications.

## Applications

| Field | Value |
|---|---|
| Name | `<element>-<cluster>`, or `<element>-acm-<datacenter>` on a cluster labeled `local-cluster` |
| Project | `default` |
| Source | `stack/<element>` in `gitRepo` at the cluster's revision, rendered by Helm with the [values cascade](values.md#order) |
| Destination | The hub, `https://kubernetes.default.svc`, namespace `open-cluster-management` |
| Sync | Automated with `prune` and `selfHeal`, sync option `RespectIgnoreDifferences=true` |

`<element>` is the directory name and `<cluster>` the ManagedCluster name. Argo CD uses the Application name as the Helm release name, and every rendered object name derives from it ([Rendered objects](policies.md#rendered-objects), [Naming limits](policies.md#naming-limits)).

Argo CD deploys only to the hub. ACM delivers the policies to the managed clusters ([Placement](policies.md#placement)). The chart sets `policyNamespace` on every object it renders, so the destination namespace is unused.

Automated sync leaves `allowEmpty` unset, so it never prunes an Application down to no objects ([Disabling an element](rollout.md#disabling-an-element)).

The ApplicationSet sets `preserveResourcesOnDeletion: true`, so its Applications carry no resources finalizer and deleting one leaves its objects on the hub ([Removing an element](rollout.md#removing-an-element)).

## Injected values

Each Application passes these keys to Helm in `valuesObject`, built from the cluster's labels. [Order](values.md#order) gives their precedence over the values files.

| Key | Value |
|---|---|
| `selected<Category>` | The value of the cluster's config label in that category, for example `selectedEnvironment`. When a category has several labels, the one that sorts last in [Order](values.md#order) wins. |
| `selected<Category>Values` | Every value of that category, as a list. |
| `selectedName` | The cluster name, or `acm-<datacenter>` on the hub. |
| `selectedId` | The `clusterID` label. |
| `selectedDefaultStorageClass` | The `env.example.com/defaultstorageclass` label. Absent when the label is not set. |
| `selector.matchExpressions` | A map of `In` expressions, one per config label plus one for the `name` label, keyed by the label key with `.`, `/` and `-` replaced by `_`. |

Elements can read every key. The policy-library chart turns `selector` into each Placement's cluster selector ([Placement](policies.md#placement)). On the hub, `selectedName` is `acm-<datacenter>`, but the `name` expression holds the hub's ManagedCluster name. For `prod-east-1`:

```yaml
selectedDatacenter: dc1
selectedEnvironment: prod
selectedPlatform: aws
selectedDatacenterValues: [dc1]
selectedEnvironmentValues: [prod]
selectedPlatformValues: [aws]
selectedName: prod-east-1
selectedId: <clusterID>
selector:
  matchExpressions:
    config_example_com_datacenter_20:
      key: config.example.com/datacenter.20
      operator: In
      values:
        - "dc1"
    config_example_com_environment_10:
      key: config.example.com/environment.10
      operator: In
      values:
        - "prod"
    config_example_com_platform_30:
      key: config.example.com/platform.30
      operator: In
      values:
        - "aws"
    name:
      key: name
      operator: In
      values:
        - "prod-east-1"
```

## The appset chart

The [appset chart](https://github.com/PolicyStack/PolicyStack/tree/main/appset) installs the ApplicationSet and reads these values:

| Value | Set in | Use |
|---|---|---|
| `baseDomain` | `appset/values.yaml` | The domain in the `git.`, `config.` and `env.` label keys. |
| `gitRepo` | `appset/values.yaml` | The repository the git generator lists and every Application renders from. |
| `policyNamespace` | Root `values.yaml` | The namespace the chart creates. Elements read the same value. Do not change it. |

The install command passes both values files ([Install the ApplicationSet](install.md#install-the-applicationset)). The chart also binds the `global` ManagedClusterSet into `policyNamespace`, so element Placements can select clusters ([Placement](policies.md#placement)).

[`appset/appset-noformatting.yaml`](https://github.com/PolicyStack/PolicyStack/blob/main/appset/appset-noformatting.yaml) is the ApplicationSet as `helm template` renders it with the default values, without the Helm escaping around Argo CD's template expressions. It sits outside `templates/`, so Helm never installs it.
