package checks

import (
	"fmt"

	"github.com/PolicyStack/PolicyStack/tools/validator/internal/chart"
	"github.com/PolicyStack/PolicyStack/tools/validator/internal/sourceloc"
)

// CamelCaseCheck (POLICY090) checks that the key under `stack:` in
// values.yaml is the camelCase form of Chart.yaml's `name:`. On a mismatch
// policy-library renders no policies and reports no error.
type CamelCaseCheck struct{}

func (CamelCaseCheck) ID() string   { return "POLICY090" }
func (CamelCaseCheck) Phase() Phase { return PhaseChart }

func (c *CamelCaseCheck) Run(ctx Context) []Finding {
	if ctx.Element == nil || ctx.Element.StackKey == "" {
		return nil
	}
	want := chart.CamelFromKebab(ctx.Element.ChartName)
	if ctx.Element.StackKey == want {
		return nil
	}
	loc := sourceloc.Find(ctx.Element.ValuesDoc, "stack", ctx.Element.StackKey)
	return []Finding{{
		RuleID:   c.ID(),
		Severity: SevError,
		Element:  ctx.Element.ChartName,
		Message: fmt.Sprintf("stack key %q does not match camelCase form of Chart.yaml name %q (expected %q)",
			ctx.Element.StackKey, ctx.Element.ChartName, want),
		File: ctx.Element.ValuesFile,
		Line: loc.Line, Col: loc.Col,
	}}
}
