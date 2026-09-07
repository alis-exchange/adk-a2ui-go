package schema

import (
	"context"
	"errors"
	"testing"

	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
)

func TestResolveOrder(t *testing.T) {
	inline := map[string]any{"catalogId": "acme:ui", "from": "inline"}
	reg := kit.NewRegistry()
	_ = reg.Register(map[string]any{"catalogId": "acme:ui", "from": "registry"})
	_ = reg.Register(map[string]any{"catalogId": "acme:other", "from": "registry"})
	opts := kit.ValidateOptions{Params: kit.VersionParams{InlineCatalogs: []map[string]any{inline}}, Resolver: reg}

	c, ok, err := ResolveCatalog(context.Background(), spec.MajorV09, "acme:ui", opts)
	if err != nil || !ok || c["from"] != "inline" {
		t.Errorf("inline should win: %v %v %v", c, ok, err)
	}
	c, ok, _ = ResolveCatalog(context.Background(), spec.MajorV09, "acme:other", opts)
	if !ok || c["from"] != "registry" {
		t.Error("registry should be consulted after inline")
	}
	for _, id := range spec.BasicCatalogIDs(spec.MajorV09) {
		if _, ok, _ := ResolveCatalog(context.Background(), spec.MajorV09, id, kit.ValidateOptions{}); !ok {
			t.Errorf("embedded basic catalog not found for %s", id)
		}
	}
	if _, ok, _ := ResolveCatalog(context.Background(), spec.MajorV09, "https://nope", opts); ok {
		t.Error("unknown id resolved")
	}
	if _, ok, _ := ResolveCatalog(context.Background(), spec.MajorV09, "", opts); ok {
		t.Error("empty id resolved")
	}
}

type failing struct{}

func (failing) ResolveCatalog(context.Context, string) (map[string]any, bool, error) {
	return nil, false, errors.New("boom")
}

func TestResolveError(t *testing.T) {
	if _, _, err := ResolveCatalog(context.Background(), spec.MajorV09, "x", kit.ValidateOptions{Resolver: failing{}}); err == nil {
		t.Error("resolver error must propagate")
	}
}

func TestCatalogInstructions(t *testing.T) {
	c09, _, _, _ := spec.BasicCatalog(spec.MajorV09)
	if CatalogInstructions(spec.MajorV09, c09) == "" {
		t.Error("v0_9 basic catalog should yield rules.txt")
	}
	c10, _, _, _ := spec.BasicCatalog(spec.MajorV10)
	if CatalogInstructions(spec.MajorV10, c10) == "" {
		t.Error("v1_0 basic catalog should yield its instructions field")
	}
	if got := CatalogInstructions(spec.MajorV09, map[string]any{"catalogId": "x", "instructions": " hi "}); got != "hi" {
		t.Errorf("custom instructions = %q", got)
	}
}
