package checks

import (
	"context"
	"strings"

	"github.com/PolicyStack/PolicyStack/tools/validator/internal/render"
)

// LintCheck (POLICY070) runs `helm lint` on each element. Runner is set by
// run.buildRegistry; the check does nothing without it.
type LintCheck struct {
	Runner *render.Runner
}

func (LintCheck) ID() string   { return "POLICY070" }
func (LintCheck) Phase() Phase { return PhaseChart }

func (c *LintCheck) Run(ctx Context) []Finding {
	if ctx.Element == nil || c.Runner == nil {
		return nil
	}
	out, err := c.Runner.Lint(context.Background(), ctx.Element.Dir)
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(out)
	if msg == "" {
		msg = err.Error()
	}
	return []Finding{{
		RuleID:   c.ID(),
		Severity: SevError,
		Element:  ctx.Element.ChartName,
		Message:  "helm lint failed: " + truncate(msg, 800),
		File:     ctx.Element.Dir,
		Line:     0,
	}}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
