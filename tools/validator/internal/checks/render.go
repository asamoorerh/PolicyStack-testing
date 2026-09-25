package checks

// RenderCheck (RENDER000) registers the rule ID for `helm template`
// failures. run.Run produces the findings; Run is a no-op.
type RenderCheck struct{}

func (RenderCheck) ID() string   { return "RENDER000" }
func (RenderCheck) Phase() Phase { return PhaseCluster }

func (c *RenderCheck) Run(_ Context) []Finding { return nil }
