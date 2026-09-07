package schema

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.alis.build/adk/a2ui/spec"
)

const basicV09 = "https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json"

func batch(t *testing.T, version, component string) []any {
	t.Helper()
	src := `[{"version":"` + version + `","createSurface":{"surfaceId":"s","catalogId":"` + basicV09 + `"}},
	         {"version":"` + version + `","updateComponents":{"surfaceId":"s","components":[` + component + `]}}]`
	var v []any
	if err := json.Unmarshal([]byte(src), &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCompileWithBasicCatalogValidatesExample(t *testing.T) {
	cat, _, _, err := spec.BasicCatalog(spec.MajorV09)
	if err != nil {
		t.Fatal(err)
	}
	s, err := For(spec.MajorV09).Compile(CompileOptions{Entry: EntryOutboundV09, Catalog: cat})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join("..", "..", "spec", "v0_9", "testdata", "examples", "00_complex-layout.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(doc["messages"]); err != nil {
		t.Fatalf("official example rejected: %v", err)
	}
	if err := s.Validate(batch(t, "v0.9", `{"id":"root","component":"Text","text":"hi","colour":"red"}`)); err == nil {
		t.Error("unknown property accepted with basic catalog")
	}
}

func TestStubCatalogRequiresIDOnly(t *testing.T) {
	s, err := For(spec.MajorV09).Compile(CompileOptions{Entry: EntryOutboundV09})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(batch(t, "v0.9", `{"id":"root","component":"Anything","whatever":1}`)); err != nil {
		t.Errorf("stub should accept unknown components/props: %v", err)
	}
	if err := s.Validate(batch(t, "v0.9", `{"component":"Text","text":"hi"}`)); err == nil {
		t.Error("stub should require id")
	}
}

func TestV091Patch(t *testing.T) {
	strict, _ := For(spec.MajorV09).Compile(CompileOptions{Entry: EntryOutboundV09})
	if err := strict.Validate(batch(t, "v0.9.1", `{"id":"root","component":"Text","text":"hi"}`)); err == nil {
		t.Error("v0.9.1 accepted without the patch")
	}
	patched, _ := For(spec.MajorV09).Compile(CompileOptions{Entry: EntryOutboundV09, V091: true})
	for _, v := range []string{"v0.9", "v0.9.1"} {
		if err := patched.Validate(batch(t, v, `{"id":"root","component":"Text","text":"hi"}`)); err != nil {
			t.Errorf("%s rejected with patch: %v", v, err)
		}
	}
}

func TestCompileIsCached(t *testing.T) {
	a, _ := For(spec.MajorV10).Compile(CompileOptions{Entry: EntryOutboundV10})
	b, _ := For(spec.MajorV10).Compile(CompileOptions{Entry: EntryOutboundV10})
	if a != b {
		t.Error("same options should return the cached schema")
	}
	cat, _, _, _ := spec.BasicCatalog(spec.MajorV10)
	c, _ := For(spec.MajorV10).Compile(CompileOptions{Entry: EntryOutboundV10, Catalog: cat})
	if c == a {
		t.Error("different catalog must not share a cache entry")
	}
}

func TestCompileRefComponent(t *testing.T) {
	cat, _, _, _ := spec.BasicCatalog(spec.MajorV09)
	s, err := For(spec.MajorV09).CompileRef(RefAnyComponent, cat, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(map[string]any{"id": "x", "component": "Text", "text": "hi"}); err != nil {
		t.Errorf("valid Text rejected: %v", err)
	}
	if err := s.Validate(map[string]any{"id": "x", "component": "Text"}); err == nil {
		t.Error("Text without text accepted")
	}
}

func TestRegexpShimHandlesXID(t *testing.T) {
	re, err := goRegexp(`^[\p{XID_Start}_][\p{XID_Continue}]*$`)
	if err != nil {
		t.Fatal(err)
	}
	if !re.MatchString("vendor_key1") || re.MatchString("1bad") {
		t.Error("shimmed pattern misbehaves")
	}
}

func TestToInstance(t *testing.T) {
	got := ToInstance([]map[string]any{{"a": 1.0}})
	if len(got) != 1 {
		t.Fatal("length")
	}
	if _, ok := got[0].(map[string]any); !ok {
		t.Error("element type")
	}
}
