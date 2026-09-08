package schema

import (
	"strings"
	"testing"

	"go.alis.build/adk/a2ui/spec"
)

// v1.0 inline catalog in the catalog_definition.json shape: no $defs at all.
func v10Catalog() map[string]any {
	return map[string]any{
		"catalogId": "https://example.com/v10.json",
		"components": map[string]any{
			"Badge": map[string]any{
				"type":       "object",
				"properties": map[string]any{"component": map[string]any{"const": "Badge"}, "label": map[string]any{"type": "string"}},
				"required":   []any{"component", "label"},
			},
		},
		"functions": map[string]any{
			"ping": map[string]any{
				"type":       "object",
				"returnType": "string",
				"properties": map[string]any{"call": map[string]any{"const": "ping"}, "args": map[string]any{"type": "object", "properties": map[string]any{"n": map[string]any{"type": "number"}}, "required": []any{"n"}, "unevaluatedProperties": false}},
				"required":   []any{"call", "args"},
			},
		},
	}
}

// v0.9 inline catalog in the client_capabilities.json#/$defs/Catalog shape: components are
// schemas, functions is an array of {name, parameters, returnType}, theme maps property to schema.
func v09Catalog() map[string]any {
	return map[string]any{
		"catalogId": "https://example.com/v09.json",
		"components": map[string]any{
			"Badge": map[string]any{
				"type":       "object",
				"properties": map[string]any{"id": map[string]any{"type": "string"}, "component": map[string]any{"const": "Badge"}, "label": map[string]any{"type": "string"}},
				"required":   []any{"id", "component", "label"},
			},
		},
		"functions": []any{
			map[string]any{"name": "trim", "returnType": "string", "parameters": map[string]any{"type": "object", "properties": map[string]any{"value": map[string]any{"type": "string"}}, "required": []any{"value"}}},
		},
		"theme": map[string]any{"primaryColor": map[string]any{"type": "string"}},
	}
}

func firstLine(err error) string {
	if err == nil {
		return "<nil>"
	}
	return strings.SplitN(err.Error(), "\n", 2)[0]
}

func TestV10CatalogWithoutDefsCompiles(t *testing.T) {
	cat := v10Catalog()
	comp, err := For(spec.MajorV10).CompileRef("agent_to_renderer.json#/$defs/Component", cat, false)
	if err != nil {
		t.Fatalf("catalog without $defs must compile: %v", err)
	}
	if err := comp.Validate(map[string]any{"id": "x", "component": "Badge", "label": "hi"}); err != nil {
		t.Errorf("valid Badge rejected: %v", err)
	}
	err = comp.Validate(map[string]any{"id": "x", "component": "Badge"})
	if err == nil {
		t.Error("Badge without label accepted")
	}
	inst := map[string]any{"id": "x", "component": "Nope"}
	if err := comp.Validate(inst); err == nil {
		t.Error("unknown component accepted")
	} else if got := Format(err, inst, "c"); len(got) != 1 || !strings.Contains(got[0].Message, `unknown component "Nope" (catalog components: Badge)`) {
		t.Errorf("synthesised union must prune like the basic catalog's, got %v", got)
	}
	fn, err := For(spec.MajorV10).CompileRef("common_types.json#/$defs/FunctionCall", cat, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := fn.Validate(map[string]any{"call": "ping", "args": map[string]any{"n": 1.0}}); err != nil {
		t.Errorf("valid ping rejected: %v", err)
	}
	if err := fn.Validate(map[string]any{"call": "ping", "args": map[string]any{}}); err == nil {
		t.Error("ping without n accepted")
	}
	if _, has := cat["$defs"]; has {
		t.Error("the caller's catalog must not be mutated")
	}
}

func TestV09CatalogWithoutDefsCompiles(t *testing.T) {
	cat := v09Catalog()
	s, err := For(spec.MajorV09).Compile(CompileOptions{Entry: EntryOutboundV09, Catalog: cat})
	if err != nil {
		t.Fatalf("v0.9 capability-shaped catalog must compile: %v", err)
	}
	batch := func(theme any, comp map[string]any) []any {
		cs := map[string]any{"surfaceId": "s", "catalogId": "https://example.com/v09.json"}
		if theme != nil {
			cs["theme"] = theme
		}
		return []any{
			map[string]any{"version": "v0.9", "createSurface": cs},
			map[string]any{"version": "v0.9", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{comp}}},
		}
	}
	badge := map[string]any{"id": "root", "component": "Badge", "label": "hi"}
	if err := s.Validate(batch(map[string]any{"primaryColor": "#fff"}, badge)); err != nil {
		t.Errorf("valid theme and Badge rejected: %s", firstLine(err))
	}
	if err := s.Validate(batch(map[string]any{"primaryColor": 1.0}, badge)); err == nil {
		t.Error("theme property of the wrong type accepted")
	}
	if err := s.Validate(batch(nil, map[string]any{"id": "root", "component": "Badge"})); err == nil {
		t.Error("Badge without label accepted")
	}
	fn, err := For(spec.MajorV09).CompileRef("common_types.json#/$defs/FunctionCall", cat, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := fn.Validate(map[string]any{"call": "trim", "args": map[string]any{"value": "x"}}); err != nil {
		t.Errorf("valid trim rejected: %s", firstLine(err))
	}
	if err := fn.Validate(map[string]any{"call": "trim", "args": map[string]any{}}); err == nil {
		t.Error("trim without value accepted")
	}
}

// A derived v0.9 function branch must require "args" when the catalog's own parameters schema
// has mandatory parameters -- a function like trim cannot be called without them -- but stay
// permissive for a function like now, whose parameters take no required fields.
func TestV09DerivedFunctionRequiresArgsOnlyWhenParametersDo(t *testing.T) {
	cat := v09Catalog()
	cat["functions"] = append(cat["functions"].([]any), map[string]any{
		"name": "now", "returnType": "string", "parameters": map[string]any{"type": "object"},
	})
	fn, err := For(spec.MajorV09).CompileRef("common_types.json#/$defs/FunctionCall", cat, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := fn.Validate(map[string]any{"call": "trim"}); err == nil {
		t.Error("trim without args accepted, but trim's parameters require value")
	}
	if err := fn.Validate(map[string]any{"call": "now"}); err != nil {
		t.Errorf("now without args rejected, but now's parameters require nothing: %s", firstLine(err))
	}
}

func TestCatalogDeclaringNothingIsPermissive(t *testing.T) {
	cat := map[string]any{"catalogId": "https://example.com/empty.json"}
	comp, err := For(spec.MajorV10).CompileRef("agent_to_renderer.json#/$defs/Component", cat, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := comp.Validate(map[string]any{"id": "x", "component": "Anything", "whatever": 1.0}); err != nil {
		t.Errorf("a catalog that declares no components cannot reject one: %s", firstLine(err))
	}
	if err := comp.Validate(map[string]any{"component": "Anything"}); err == nil {
		t.Error("the envelope's own rules (id required) must still apply")
	}
	fn, err := For(spec.MajorV10).CompileRef("common_types.json#/$defs/FunctionCall", cat, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := fn.Validate(map[string]any{"call": "anything", "args": map[string]any{"a": 1.0}}); err != nil {
		t.Errorf("a catalog that declares no functions cannot reject a call: %s", firstLine(err))
	}
}

func TestDeclaredDefsWin(t *testing.T) {
	cat := v10Catalog()
	cat["$defs"] = map[string]any{
		"anyComponent": map[string]any{"type": "object", "required": []any{"zzz"}},
		"anyFunction":  map[string]any{"type": "object"},
	}
	comp, err := For(spec.MajorV10).CompileRef("agent_to_renderer.json#/$defs/Component", cat, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := comp.Validate(map[string]any{"id": "x", "component": "Badge", "label": "hi"}); err == nil {
		t.Error("a declared anyComponent must be used as is, not replaced by the synthesised union")
	}
}

// The components and functions unions fall back to the permissive stub independently: a catalog
// that declares one but not the other must stay strict on the declared side and permissive on
// the other, in every combination.
func TestUnionsFallBackIndependently(t *testing.T) {
	cases := []struct {
		name                        string
		hasComponents, hasFunctions bool
	}{
		{"components only", true, false},
		{"functions only", false, true},
		{"both", true, true},
		{"neither", false, false},
	}
	assertUnion := func(t *testing.T, declared bool, declaredErr, undeclaredErr error) {
		t.Helper()
		if declared && declaredErr == nil {
			t.Error("a declared union must reject an unknown name")
		}
		if !declared && undeclaredErr != nil {
			t.Errorf("an undeclared union must accept an arbitrary value: %v", undeclaredErr)
		}
	}
	for _, tc := range cases {
		t.Run("v1.0/"+tc.name, func(t *testing.T) {
			cat := map[string]any{"catalogId": "https://example.com/x.json"}
			if tc.hasComponents {
				cat["components"] = map[string]any{"Badge": map[string]any{
					"type": "object", "properties": map[string]any{"component": map[string]any{"const": "Badge"}}, "required": []any{"component"},
				}}
			}
			if tc.hasFunctions {
				cat["functions"] = map[string]any{"ping": map[string]any{
					"type": "object", "properties": map[string]any{"call": map[string]any{"const": "ping"}}, "required": []any{"call"},
				}}
			}
			comp, err := For(spec.MajorV10).CompileRef("agent_to_renderer.json#/$defs/Component", cat, false)
			if err != nil {
				t.Fatal(err)
			}
			compErr := comp.Validate(map[string]any{"id": "x", "component": "Anything"})
			assertUnion(t, tc.hasComponents, compErr, compErr)

			fn, err := For(spec.MajorV10).CompileRef("common_types.json#/$defs/FunctionCall", cat, false)
			if err != nil {
				t.Fatal(err)
			}
			fnErr := fn.Validate(map[string]any{"call": "anything"})
			assertUnion(t, tc.hasFunctions, fnErr, fnErr)
		})
		t.Run("v0.9/"+tc.name, func(t *testing.T) {
			cat := map[string]any{"catalogId": "https://example.com/y.json"}
			if tc.hasComponents {
				cat["components"] = map[string]any{"Badge": map[string]any{
					"type":       "object",
					"properties": map[string]any{"id": map[string]any{"type": "string"}, "component": map[string]any{"const": "Badge"}},
					"required":   []any{"id", "component"},
				}}
			}
			if tc.hasFunctions {
				cat["functions"] = []any{map[string]any{"name": "ping", "parameters": map[string]any{"type": "object"}}}
			}
			s, err := For(spec.MajorV09).Compile(CompileOptions{Entry: EntryOutboundV09, Catalog: cat})
			if err != nil {
				t.Fatal(err)
			}
			// theme absent: a createSurface without one must still compile and validate.
			batch := []any{
				map[string]any{"version": "v0.9", "createSurface": map[string]any{"surfaceId": "s", "catalogId": cat["catalogId"]}},
				map[string]any{"version": "v0.9", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{
					map[string]any{"id": "root", "component": "Anything"},
				}}},
			}
			compErr := s.Validate(batch)
			assertUnion(t, tc.hasComponents, compErr, compErr)

			fn, err := For(spec.MajorV09).CompileRef("common_types.json#/$defs/FunctionCall", cat, false)
			if err != nil {
				t.Fatal(err)
			}
			fnErr := fn.Validate(map[string]any{"call": "anything"})
			assertUnion(t, tc.hasFunctions, fnErr, fnErr)
		})
	}
}
