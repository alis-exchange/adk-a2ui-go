package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.alis.build/adk/a2ui/spec"
)

func loadExample(t *testing.T, file string) []any {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil { // encoding/json on purpose: float64 like real tool input
		t.Fatalf("%s: %v", file, err)
	}
	if m, ok := doc.(map[string]any); ok {
		doc = m["messages"]
	}
	list, ok := doc.([]any)
	if !ok {
		t.Fatalf("%s: not a message list", file)
	}
	return list
}

func rewriteVersions(inst []any, version string) {
	for _, m := range inst {
		if obj, ok := m.(map[string]any); ok {
			obj["version"] = version
		}
	}
}

func TestOfficialExamples(t *testing.T) {
	cases := []struct {
		major, entry, wire string
		v091               bool
	}{
		{spec.MajorV09, EntryOutboundV09, "v0.9", false},
		{spec.MajorV09, EntryOutboundV09, "v0.9.1", true},
		{spec.MajorV10, EntryOutboundV10, "v1.0", false},
	}
	for _, tc := range cases {
		t.Run(tc.wire, func(t *testing.T) {
			cat, _, _, err := spec.BasicCatalog(tc.major)
			if err != nil {
				t.Fatal(err)
			}
			s, err := For(tc.major).Compile(CompileOptions{Entry: tc.entry, Catalog: cat, V091: tc.v091})
			if err != nil {
				t.Fatal(err)
			}
			files, _ := filepath.Glob(filepath.Join("..", "..", "spec", tc.major, "testdata", "examples", "*.json"))
			if len(files) == 0 {
				t.Fatal("no examples found; run scripts/sync-spec.sh")
			}
			for _, f := range files {
				inst := loadExample(t, f)
				if tc.wire == "v0.9.1" {
					rewriteVersions(inst, "v0.9.1")
				}
				if err := s.Validate(inst); err != nil {
					t.Errorf("%s: %s", filepath.Base(f), strings.SplitN(err.Error(), "\n", 2)[0])
				}
			}
		})
	}
}
