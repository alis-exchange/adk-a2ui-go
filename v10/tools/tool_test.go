package tools

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool/toolconfirmation"

	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/v10"
)

type fakeContext struct{ agent.StrictContextMock }

func (f *fakeContext) ToolConfirmation() *toolconfirmation.ToolConfirmation { return nil }

func fake(ctx context.Context) *fakeContext { return &fakeContext{agent.NewStrictContextMock(ctx)} }

type runnable interface {
	Run(ctx agent.Context, args any) (map[string]any, error)
}

func capsCtx(t *testing.T, version string) context.Context {
	t.Helper()
	return kit.WithA2UICapabilities(context.Background(), map[string]any{
		version: map[string]any{"supportedCatalogIds": []any{v10.CatalogIDBasic}},
	})
}

func TestToolsetVisibility(t *testing.T) {
	ts, err := NewToolset(kit.ToolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if tools, _ := ts.Tools(fake(context.Background())); len(tools) != 0 {
		t.Error("visible without capabilities")
	}
	if tools, _ := ts.Tools(fake(capsCtx(t, kit.V09))); len(tools) != 0 {
		t.Error("visible for a different version")
	}
	tools, err := ts.Tools(fake(capsCtx(t, kit.V10)))
	if err != nil || len(tools) != 2 || tools[0].Name() != CatalogToolName || tools[1].Name() != GenerateA2UIMessagesToolName {
		t.Errorf("tools = %v, %v", tools, err)
	}
	if !strings.Contains(tools[1].Description(), `"version": "v1.0"`) || !strings.Contains(tools[1].Description(), v10.CatalogIDBasic) {
		t.Error("description must name the version and the negotiated catalog id")
	}
}

func TestGenerateTool(t *testing.T) {
	params := kit.VersionParams{SupportedCatalogIDs: []string{v10.CatalogIDBasic}}
	tl, err := GenerateTool(params, kit.ToolOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx := fake(context.Background())

	good := []any{
		map[string]any{"version": "v1.0", "createSurface": map[string]any{"surfaceId": "s", "catalogId": v10.CatalogIDBasic}},
		map[string]any{"version": "v1.0", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{
			map[string]any{"id": "root", "component": "Card", "child": "t"},
			map[string]any{"id": "t", "component": "Text", "text": "hi"},
		}}},
	}
	out, err := tl.(runnable).Run(ctx, map[string]any{"messages": good})
	if err != nil {
		t.Fatalf("valid v1.0 batch rejected: %v", err)
	}
	if out["status"] != "success" {
		t.Errorf("out = %v", out)
	}

	bad := []any{
		map[string]any{"version": "v1.0", "createSurface": map[string]any{"surfaceId": "s", "catalogId": v10.CatalogIDBasic}},
		map[string]any{"version": "v1.0", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{
			map[string]any{"id": "root", "component": "Card", "child": "t"},
			map[string]any{"id": "t", "component": "Text", "text": "hi", "colour": "red"},
		}}},
	}
	if _, err := tl.(runnable).Run(ctx, map[string]any{"messages": bad}); err == nil || !strings.Contains(err.Error(), `unknown property "colour"`) {
		t.Errorf("catalog violation not reported: %v", err)
	}

	// ADK pre-validates arguments against InputSchema before the handler runs. v1.0 has only one
	// wire version, so the "version" property carries a Const of "v1.0": a wrong version fails
	// that pre-validation and never reaches v10.Validate, so only assert that the call errors.
	wrongVersion := []any{
		map[string]any{"version": "v0.9", "createSurface": map[string]any{"surfaceId": "s", "catalogId": v10.CatalogIDBasic}},
	}
	if _, err := tl.(runnable).Run(ctx, map[string]any{"messages": wrongVersion}); err == nil {
		t.Error("wrong version must be rejected")
	}

	// updateDataModel requires "value" in the InputSchema too, so a batch lacking it likewise
	// fails ADK's own pre-validation before the handler runs; only assert that the call errors.
	missingValue := []any{
		map[string]any{"version": "v1.0", "updateDataModel": map[string]any{"surfaceId": "s"}},
	}
	if _, err := tl.(runnable).Run(ctx, map[string]any{"messages": missingValue}); err == nil {
		t.Error("updateDataModel without value must be rejected")
	}
}

func TestCatalogTool(t *testing.T) {
	params := kit.VersionParams{SupportedCatalogIDs: []string{v10.CatalogIDBasic}}
	tl, err := CatalogTool(params, kit.ToolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := tl.(runnable).Run(fake(context.Background()), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	cats, _ := out["catalogs"].(map[string]any)
	if _, ok := cats[v10.CatalogIDBasic]; !ok {
		t.Errorf("basic catalog not returned: %v", out)
	}
	if ins, _ := out["instructions"].(string); ins == "" {
		t.Error("instructions must be non-empty")
	}
}
