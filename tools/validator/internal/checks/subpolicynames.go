package checks

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/PolicyStack/PolicyStack/tools/validator/internal/chart"
	"github.com/PolicyStack/PolicyStack/tools/validator/internal/sourceloc"
)

// SubPolicyNameCheck (POLICY003) catches two DIFFERENT elements that render a policy-template with
// the same name.
//
// ACM requires policy-template names to be unique across every Policy placed on a cluster, because
// the templates are replicated into that cluster's namespace. A collision is rejected on-cluster
// with "Template name must be unique", and the whole Policy stops being processed.
//
// POLICY002 only compares names within a single element, and a sub-policy name is
// "<policyRef>-<name>" with no element or cluster component - so short, generic names like
// policies[].name=install plus configPolicies[].name=ns-monitoring collide silently between
// elements. That combination renders and validates cleanly and only fails once ACM sees it, which
// is exactly why this runs repo-wide.
type SubPolicyNameCheck struct{}

func (SubPolicyNameCheck) ID() string   { return "POLICY003" }
func (SubPolicyNameCheck) Phase() Phase { return PhaseRepo }

// owner records where a rendered sub-policy name came from.
type owner struct {
	element   string
	valuePath string
	idx       int
	el        *chart.Element
}

func (c *SubPolicyNameCheck) Run(ctx Context) []Finding {
	// rendered name -> every element/entry that produces it
	seen := map[string][]owner{}

	for _, el := range ctx.AllElements {
		if el == nil || el.Values == nil || el.Values.Component == nil {
			continue
		}
		comp := el.Values.Component
		add := func(name, valuePath string, idx int) {
			if name == "" {
				return
			}
			seen[name] = append(seen[name], owner{el.ChartName, valuePath, idx, el})
		}
		for i, p := range comp.ConfigPolicies {
			if comp.CouldBeEnabled(p.Name, p.Enabled) && p.PolicyRef != "" {
				add(p.PolicyRef+"-"+p.Name, "configPolicies", i)
			}
		}
		for i, p := range comp.OperatorPolicies {
			if !comp.CouldBeEnabled(p.Name, p.Enabled) || p.PolicyRef == "" {
				continue
			}
			base := p.PolicyRef + "-" + p.Name
			// an operator policy renders three templates
			add(base, "operatorPolicies", i)
			add(base+"-ns", "operatorPolicies", i)
			add(base+"-status", "operatorPolicies", i)
		}
		for i, p := range comp.CertificatePolicies {
			if comp.CouldBeEnabled(p.Name, p.Enabled) && p.PolicyRef != "" {
				add(p.PolicyRef+"-"+p.Name, "certificatePolicies", i)
			}
		}
	}

	names := make([]string, 0, len(seen))
	for n := range seen {
		names = append(names, n)
	}
	sort.Strings(names)

	var out []Finding
	for _, name := range names {
		owners := seen[name]
		// only a collision when more than one distinct element produces it; a repeat inside one
		// element is POLICY002's job and is reported there with better locality
		elements := map[string]bool{}
		for _, o := range owners {
			elements[o.element] = true
		}
		if len(elements) < 2 {
			continue
		}
		others := make([]string, 0, len(elements))
		for e := range elements {
			others = append(others, e)
		}
		sort.Strings(others)
		for _, o := range owners {
			loc := sourceloc.Find(o.el.ValuesDoc, "stack", o.el.StackKey, o.valuePath, strconv.Itoa(o.idx), "name")
			out = append(out, Finding{
				RuleID:   c.ID(),
				Severity: SevError,
				Element:  o.element,
				Message: fmt.Sprintf(
					"%s[%d] renders policy-template %q, which %v also render. ACM requires template names to be unique across every Policy on a cluster - give the sub-policy an element-specific name",
					o.valuePath, o.idx, name, others),
				File: o.el.ValuesFile, Line: loc.Line, Col: loc.Col,
			})
		}
	}
	return out
}
