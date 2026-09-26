// Package cascade resolves a ManagedCluster and its labels into the ordered
// list of values files passed to helm.
//
// It mirrors the valueFiles block in appset/templates/appset.yaml. Keep the two
// in sync; any divergence is a bug.
//
// Cascade order (lowest precedence first):
//
//  1. element defaults       <element>/values.yaml
//  2. global root            <repoRoot>/values.yaml
//  3. label-driven entries   <values>/<category>s/<value>.yaml, in ascending
//     priority order (higher priority overrides)
//  4. cluster-specific       hub: <values>/acm/acm-<dc>.yaml then
//     <values>/clusters/acm-<dc>.yaml
//     else: <values>/clusters/<cluster-name>.yaml
//
// Missing files are skipped, matching the appset's ignoreMissingValueFiles: true.
package cascade

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/PolicyStack/PolicyStack/tools/validator/internal/fixtures"
)

// Resolved is the output of Resolve.
type Resolved struct {
	// ClusterName is the helm release suffix and the <cluster> part of
	// policy names. For local-cluster=true hubs it is "acm-<datacenter>".
	ClusterName string
	// ReleaseName is the helm Release.Name, <element>-<cluster>, matching the
	// Application name set in appset.yaml.
	ReleaseName string
	// ValueFiles is the ordered list of -f arguments (absolute paths).
	ValueFiles []string
	// IsLocalHub is metadata.labels["local-cluster"] == "true".
	IsLocalHub  bool
	Datacenter  string
	Environment string
	// LabelIssues lists malformed config labels. POLICY050 reports them.
	LabelIssues []LabelIssue
}

// LabelIssue describes a malformed or duplicate config label key.
type LabelIssue struct {
	Key    string
	Reason string // e.g. "duplicate priority", "missing priority", "non-numeric priority"
}

type entry struct {
	priority int
	category string
	value    string
}

// Resolve builds the cascade for mc. element is the chart directory,
// repoRoot holds the global values.yaml, and valuesRoot is normally
// <repoRoot>/values.
func Resolve(mc *fixtures.ManagedCluster, element, chartName, repoRoot, valuesRoot, baseDomain string) Resolved {
	r := Resolved{}
	prefix := "config." + baseDomain + "/"

	// 1-2. element defaults and global root
	addIfExists(&r.ValueFiles, filepath.Join(element, "values.yaml"))
	addIfExists(&r.ValueFiles, filepath.Join(repoRoot, "values.yaml"))

	// 3. label-driven entries from config.<baseDomain>/<category>.<priority> labels
	type seen struct{ value string }
	dups := map[string]struct{}{}
	priorityKeys := map[string]struct{}{}
	var entries []entry

	for k, v := range mc.Metadata.Labels {
		if !strings.HasPrefix(k, prefix) {
			continue
		}
		rest := strings.TrimPrefix(k, prefix)
		dot := strings.LastIndex(rest, ".")
		if dot < 0 {
			r.LabelIssues = append(r.LabelIssues, LabelIssue{Key: k, Reason: "missing priority"})
			continue
		}
		category := rest[:dot]
		prioStr := rest[dot+1:]
		prio, err := strconv.Atoi(prioStr)
		if err != nil {
			r.LabelIssues = append(r.LabelIssues, LabelIssue{Key: k, Reason: "non-numeric priority"})
			continue
		}
		if category == "" || prioStr == "" {
			r.LabelIssues = append(r.LabelIssues, LabelIssue{Key: k, Reason: "empty category or priority"})
			continue
		}
		// Category comparison is case-insensitive.
		dupKey := strings.ToLower(category) + "." + prioStr
		if _, ok := priorityKeys[dupKey]; ok {
			if _, alreadyReported := dups[dupKey]; !alreadyReported {
				r.LabelIssues = append(r.LabelIssues, LabelIssue{Key: k, Reason: "duplicate <category>.<priority>"})
				dups[dupKey] = struct{}{}
			}
		}
		priorityKeys[dupKey] = struct{}{}

		entries = append(entries, entry{priority: prio, category: category, value: v})

		// Datacenter names the hub values files in step 4; Environment is informational.
		switch strings.ToLower(category) {
		case "environment":
			r.Environment = v
		case "datacenter":
			r.Datacenter = v
		}
	}

	// Ascending priority, so higher priorities come later; helm gives later -f files precedence.
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].priority != entries[j].priority {
			return entries[i].priority < entries[j].priority
		}
		return entries[i].category < entries[j].category
	})
	for _, e := range entries {
		// Pluralised with a plain "s", as appset's printf "%ss" does.
		path := filepath.Join(valuesRoot, e.category+"s", e.value+".yaml")
		addIfExists(&r.ValueFiles, path)
	}

	// 4. cluster-specific
	r.IsLocalHub = strings.EqualFold(mc.Metadata.Labels["local-cluster"], "true")
	r.ClusterName = mc.Metadata.Name
	if r.IsLocalHub && r.Datacenter != "" {
		r.ClusterName = fmt.Sprintf("acm-%s", r.Datacenter)
		addIfExists(&r.ValueFiles, filepath.Join(valuesRoot, "acm", r.ClusterName+".yaml"))
		addIfExists(&r.ValueFiles, filepath.Join(valuesRoot, "clusters", r.ClusterName+".yaml"))
	} else {
		addIfExists(&r.ValueFiles, filepath.Join(valuesRoot, "clusters", mc.Metadata.Name+".yaml"))
	}

	r.ReleaseName = chartName + "-" + r.ClusterName
	return r
}

func addIfExists(list *[]string, path string) {
	if path == "" {
		return
	}
	if _, err := os.Stat(path); err == nil {
		*list = append(*list, path)
	}
}
