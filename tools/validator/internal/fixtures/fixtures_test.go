package fixtures

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadDir(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string // path under the dir -> content; a trailing "/" makes a directory
		want  []Cluster         // Name and SourceFile (relative) are compared, in order
	}{
		{
			name: "spokes then hubs, named by file stem",
			files: map[string]string{
				"prod-east-1.yaml":  "revision: main\n",
				"a.yaml":            "revision: main\n",
				"hubs/acm-dc1.yaml": "revision: main\n",
			},
			want: []Cluster{
				{Name: "a", SourceFile: "a.yaml"},
				{Name: "prod-east-1", SourceFile: "prod-east-1.yaml"},
				{Name: "acm-dc1", SourceFile: "hubs/acm-dc1.yaml"},
			},
		},
		{
			name:  "missing hubs dir",
			files: map[string]string{"prod-east-1.yaml": "revision: main\n"},
			want:  []Cluster{{Name: "prod-east-1", SourceFile: "prod-east-1.yaml"}},
		},
		{
			name: "only .yaml files are read",
			files: map[string]string{
				"prod-east-1.yaml": "revision: main\n",
				"other.yml":        "revision: main\n",
				"README.md":        "# fleet\n",
				"dir.yaml/":        "",
				"hubs/other.yml":   "revision: main\n",
				"hubs/nested/":     "",
			},
			want: []Cluster{{Name: "prod-east-1", SourceFile: "prod-east-1.yaml"}},
		},
		{
			name:  "hub and spoke may share a name",
			files: map[string]string{"x.yaml": "revision: main\n", "hubs/x.yaml": "revision: main\n"},
			want:  []Cluster{{Name: "x", SourceFile: "x.yaml"}, {Name: "x", SourceFile: "hubs/x.yaml"}},
		},
		{
			name:  "empty dir",
			files: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for p, body := range tt.files {
				full := filepath.Join(dir, p)
				if strings.HasSuffix(p, "/") {
					mustMkdir(t, full)
					continue
				}
				mustMkdir(t, filepath.Dir(full))
				if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := LoadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d clusters, want %d: %+v", len(got), len(tt.want), got)
			}
			for i, w := range tt.want {
				g := got[i]
				if g.Name != w.Name || g.SourceFile != filepath.Join(dir, w.SourceFile) {
					t.Errorf("[%d] got {%s %s}, want {%s %s}", i, g.Name, g.SourceFile, w.Name, w.SourceFile)
				}
			}
		})
	}
}

func TestLoadDir_missingDir(t *testing.T) {
	if _, err := LoadDir(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected an error for a missing dir")
	}
}

// The shipped fixtures are the documented fleet file examples, so they must load cleanly.
func TestLoadDir_testdata(t *testing.T) {
	got, err := LoadDir("../../testdata/clusters")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"nonprod-west-1", "prod-east-1", "acm-dc1"}
	if len(got) != len(want) {
		t.Fatalf("got %d clusters, want %d", len(got), len(want))
	}
	for i, c := range got {
		if c.Name != want[i] || len(c.Issues) != 0 || c.Revision != "main" || len(c.ValueFiles) == 0 {
			t.Errorf("unexpected fixture %+v", c)
		}
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		in         string
		revision   string
		valueFiles []string
		issues     []Issue // Message is a substring of the actual message
	}{
		{
			name:       "valid",
			in:         "revision: main\nvalueFiles:\n  - environments/prod.yaml\n  - datacenters/dc1.yaml\n",
			revision:   "main",
			valueFiles: []string{"environments/prod.yaml", "datacenters/dc1.yaml"},
		},
		{
			name:       "list order kept",
			in:         "revision: main\nvalueFiles:\n  - platforms/aws.yaml\n  - environments/prod.yaml\n",
			revision:   "main",
			valueFiles: []string{"platforms/aws.yaml", "environments/prod.yaml"},
		},
		{name: "missing valueFiles", in: "revision: v1.4.0\n", revision: "v1.4.0"},
		{name: "empty valueFiles", in: "revision: v1.4.0\nvalueFiles: []\n", revision: "v1.4.0"},
		{name: "null valueFiles", in: "revision: v1.4.0\nvalueFiles:\n", revision: "v1.4.0"},
		{name: "quoted numeric revision", in: "revision: \"1.10\"\n", revision: "1.10"},
		{name: "tagged numeric revision", in: "revision: !!str 1.10\n", revision: "1.10"},
		{name: "revision with slash", in: "revision: feature/fleet-files\n", revision: "feature/fleet-files"},
		{
			name:   "numeric revision",
			in:     "revision: 1.10\n",
			issues: []Issue{{Line: 1, Message: "revision is read by Argo CD as 1.1 (float64), not a string"}},
		},
		{
			name:   "all-digit SHA",
			in:     "revision: 1234567\n",
			issues: []Issue{{Line: 1, Message: "revision is read by Argo CD as 1.234567e+06 (float64)"}},
		},
		{
			name:       "missing revision",
			in:         "valueFiles:\n  - environments/prod.yaml\n",
			valueFiles: []string{"environments/prod.yaml"},
			issues:     []Issue{{Message: "revision is required"}},
		},
		{name: "null revision", in: "revision:\n", issues: []Issue{{Line: 1, Message: "revision is required"}}},
		{name: "tilde revision", in: "revision: ~\n", issues: []Issue{{Line: 1, Message: "revision is required"}}},
		{name: "bool revision", in: "revision: true\n", issues: []Issue{{Line: 1, Message: "revision is read by Argo CD as true (bool)"}}},
		{name: "exponent revision", in: "revision: 1e3\n", issues: []Issue{{Line: 1, Message: "revision is read by Argo CD as 1000 (float64)"}}},
		// YAML 1.1 booleans: plain strings to yaml.v3, booleans to Argo CD.
		{name: "yes revision", in: "revision: yes\n", issues: []Issue{{Line: 1, Message: "(bool)"}}},
		{name: "empty file", in: "", issues: []Issue{{Message: "revision is required"}}},
		{name: "comment-only file", in: "# nothing\n", issues: []Issue{{Message: "revision is required"}}},
		{name: "empty revision", in: "revision: \"\"\n", issues: []Issue{{Line: 1, Message: "revision is empty"}}},
		{
			name:   "map revision",
			in:     "revision:\n  branch: main\n",
			issues: []Issue{{Line: 2, Message: "revision is read by Argo CD as map[branch:main]"}},
		},
		{
			name:     "unknown key",
			in:       "revision: main\nlabels:\n  a: b\n",
			revision: "main",
			issues:   []Issue{{Line: 2, Message: "unknown key labels: only revision and valueFiles are allowed"}},
		},
		{
			name:     "old config key",
			in:       "revision: main\nconfig:\n  environment.10: prod\n",
			revision: "main",
			issues:   []Issue{{Line: 2, Message: "unknown key config: only revision and valueFiles are allowed"}},
		},
		{
			name: "misspelt revision",
			in:   "revison: main\n",
			issues: []Issue{
				{Line: 1, Message: "unknown key revison"},
				{Message: "revision is required"},
			},
		},
		{
			// Argo CD's parser keeps the last value.
			name:     "duplicate key",
			in:       "revision: a\nrevision: b\n",
			revision: "b",
			issues:   []Issue{{Line: 2, Message: `mapping key "revision" already defined at line 1`}},
		},
		{
			name:       "numeric entry",
			in:         "revision: main\nvalueFiles:\n  - 1.10\n  - environments/prod.yaml\n",
			revision:   "main",
			valueFiles: []string{"environments/prod.yaml"},
			issues:     []Issue{{Line: 3, Message: "valueFiles entry is read by Argo CD as 1.1 (float64), not a string"}},
		},
		{
			name:     "YAML 1.1 boolean entries",
			in:       "revision: main\nvalueFiles:\n  - yes\n  - off\n",
			revision: "main",
			issues: []Issue{
				{Line: 3, Message: "valueFiles entry is read by Argo CD as true (bool)"},
				{Line: 4, Message: "valueFiles entry is read by Argo CD as false (bool)"},
			},
		},
		{
			name:       "quoted numeric entry",
			in:         "revision: main\nvalueFiles:\n  - \"1.10\"\n",
			revision:   "main",
			valueFiles: []string{"1.10"},
		},
		{
			name:       "null entry",
			in:         "revision: main\nvalueFiles:\n  -\n  - environments/prod.yaml\n",
			revision:   "main",
			valueFiles: []string{"environments/prod.yaml"},
			issues:     []Issue{{Line: 3, Message: "valueFiles entry has no value"}},
		},
		{
			name:       "duplicate entry",
			in:         "revision: main\nvalueFiles:\n  - environments/prod.yaml\n  - platforms/aws.yaml\n  - environments/prod.yaml\n",
			revision:   "main",
			valueFiles: []string{"environments/prod.yaml", "platforms/aws.yaml"},
			issues:     []Issue{{Line: 5, Message: `valueFiles "environments/prod.yaml" is listed twice`}},
		},
		{
			name:     "valueFiles is a map",
			in:       "revision: main\nvalueFiles:\n  environment.10: prod\n",
			revision: "main",
			issues:   []Issue{{Line: 3, Message: "cannot unmarshal !!map"}},
		},
		{
			name:     "valueFiles is a string",
			in:       "revision: main\nvalueFiles: environments/prod.yaml\n",
			revision: "main",
			issues:   []Issue{{Line: 2, Message: "cannot unmarshal !!str"}},
		},
		{
			name: "list instead of a map",
			in:   "- revision: main\n",
			issues: []Issue{
				{Line: 1, Message: "cannot unmarshal !!seq"},
				{Message: "revision is required"},
			},
		},
		{
			name:   "invalid YAML",
			in:     "revision: [main\n",
			issues: []Issue{{Line: 1, Message: "did not find expected"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := parse([]byte(tt.in))
			if c.Revision != tt.revision {
				t.Errorf("Revision = %q, want %q", c.Revision, tt.revision)
			}
			if !slices.Equal(c.ValueFiles, tt.valueFiles) {
				t.Errorf("ValueFiles = %v, want %v", c.ValueFiles, tt.valueFiles)
			}
			if len(c.Issues) != len(tt.issues) {
				t.Fatalf("got %d issues, want %d: %+v", len(c.Issues), len(tt.issues), c.Issues)
			}
			for i, w := range tt.issues {
				g := c.Issues[i]
				if g.Line != w.Line || !strings.Contains(g.Message, w.Message) {
					t.Errorf("[%d] got {%d %q}, want {%d %q}", i, g.Line, g.Message, w.Line, w.Message)
				}
			}
		})
	}
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}
