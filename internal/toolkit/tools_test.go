package toolkit

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool/toolconfirmation"

	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
)

type fakeContext struct{ agent.StrictContextMock }

func (f *fakeContext) ToolConfirmation() *toolconfirmation.ToolConfirmation { return nil }

type runnable interface {
	Run(ctx agent.Context, args any) (map[string]any, error)
}

func testSpec(validate ValidateFunc) Spec {
	return Spec{
		Major: spec.MajorV09, Version: "v0.9", Validate: validate,
		Envelope:    &jsonschema.Schema{Type: "array"},
		MessageKeys: []string{"createSurface", "updateComponents"},
		Notes:       []string{"note one"},
	}
}

func TestCatalogToolResolvesEmbeddedInlineAndUnknown(t *testing.T) {
	basic := spec.BasicCatalogIDs(spec.MajorV09)[0]
	params := kit.VersionParams{
		SupportedCatalogIDs: []string{basic, "https://example.com/unknown.json"},
		InlineCatalogs:      []map[string]any{{"catalogId": "inline:1", "instructions": "be nice"}},
	}
	tl, err := CatalogTool(testSpec(nil), params, kit.ToolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := tl.(runnable).Run(&fakeContext{agent.NewStrictContextMock(context.Background())}, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	ids, _ := out["catalog_ids"].([]any)
	if len(ids) != 3 {
		t.Errorf("catalog_ids = %v", ids)
	}
	cats, _ := out["catalogs"].(map[string]any)
	if _, ok := cats[basic]; !ok {
		t.Error("embedded basic catalog not resolved")
	}
	if _, ok := cats["inline:1"]; !ok {
		t.Error("inline catalog not resolved")
	}
	if un, _ := out["unresolved"].([]any); len(un) != 1 || un[0] != "https://example.com/unknown.json" {
		t.Errorf("unresolved = %v", un)
	}
	ins, _ := out["instructions"].(string)
	if !strings.Contains(ins, "be nice") || !strings.Contains(ins, "REQUIRED PROPERTIES") {
		t.Errorf("instructions = %q", ins)
	}
	if out["version"] != "v0.9" {
		t.Errorf("version = %v", out["version"])
	}
}

func TestGenerateToolPassesOptionsAndEchoes(t *testing.T) {
	var got kit.ValidateOptions
	validate := func(_ context.Context, msgs []map[string]any, opts kit.ValidateOptions) error {
		got = opts
		if len(msgs) == 0 {
			return errors.New("empty")
		}
		return nil
	}
	reg := kit.NewRegistry()
	params := kit.VersionParams{SupportedCatalogIDs: []string{"acme:ui"}}
	tl, err := GenerateTool(testSpec(validate), params, kit.ToolOptions{Resolver: reg, Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tl.Description(), `"catalogId": "acme:ui"`) || !strings.Contains(tl.Description(), "note one") {
		t.Error("description should embed the negotiated catalog id and the notes")
	}
	ctx := &fakeContext{agent.NewStrictContextMock(context.Background())}
	out, err := tl.(runnable).Run(ctx, map[string]any{"messages": []any{map[string]any{"version": "v0.9"}}})
	if err != nil {
		t.Fatal(err)
	}
	if out["status"] != "success" || out["is_valid"] != true {
		t.Errorf("out = %v", out)
	}
	if got.Version != "v0.9" || !got.Strict || got.Resolver != reg || got.Params.SupportedCatalogIDs[0] != "acme:ui" {
		t.Errorf("options not forwarded: %+v", got)
	}
	if _, err := tl.(runnable).Run(ctx, map[string]any{"messages": []any{}}); err == nil {
		t.Error("validation error must surface as tool error")
	}
}
