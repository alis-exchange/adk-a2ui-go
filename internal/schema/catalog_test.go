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
