package tools

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool/toolconfirmation"

	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/v09"
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
		version: map[string]any{"supportedCatalogIds": []any{v09.CatalogIDBasic}},
	})
}

func TestToolsetVisibility(t *testing.T) {
	ts, err := NewToolset(kit.V09, kit.ToolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if tools, _ := ts.Tools(fake(context.Background())); len(tools) != 0 {
		t.Error("visible without capabilities")
	}
	if tools, _ := ts.Tools(fake(capsCtx(t, kit.V10))); len(tools) != 0 {
		t.Error("visible for a different version")
	}
	tools, err := ts.Tools(fake(capsCtx(t, kit.V09)))
	if err != nil || len(tools) != 2 || tools[0].Name() != CatalogToolName || tools[1].Name() != GenerateA2UIMessagesToolName {
		t.Errorf("tools = %v, %v", tools, err)
	}
	if !strings.Contains(tools[1].Description(), `"version": "v0.9"`) || !strings.Contains(tools[1].Description(), v09.CatalogIDBasic) {
		t.Error("description must name the version and the negotiated catalog id")
	}
	if _, err := NewToolset(kit.V10, kit.ToolOptions{}); err == nil {
		t.Error("v1.0 must be rejected by the v09 package")
	}
}

func messages(version, component string) []any {
	return []any{
		map[string]any{"version": version, "createSurface": map[string]any{"surfaceId": "s", "catalogId": v09.CatalogIDBasic}},
		map[string]any{"version": version, "updateComponents": map[string]any{"surfaceId": "s", "components": []any{
			map[string]any{"id": "root", "component": "Card", "child": "t"},
			map[string]any{"id": "t", "component": "Text", "text": "hi", "colour": component},
		}}},
	}
}

func TestGenerateTool(t *testing.T) {
	params := kit.VersionParams{SupportedCatalogIDs: []string{v09.CatalogIDBasic}}
	tl, err := GenerateTool(kit.V091, params, kit.ToolOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx := fake(context.Background())
	good := messages("v0.9.1", "")
	good[1].(map[string]any)["updateComponents"].(map[string]any)["components"].([]any)[1] = map[string]any{"id": "t", "component": "Text", "text": "hi"}
	out, err := tl.(runnable).Run(ctx, map[string]any{"messages": good})
	if err != nil {
		t.Fatalf("valid v0.9.1 batch rejected: %v", err)
	}
	if out["status"] != "success" {
		t.Errorf("out = %v", out)
	}
	if _, err := tl.(runnable).Run(ctx, map[string]any{"messages": messages("v0.9.1", "red")}); err == nil || !strings.Contains(err.Error(), `unknown property "colour"`) {
		t.Errorf("catalog violation not reported: %v", err)
	}
	if _, err := tl.(runnable).Run(ctx, map[string]any{"messages": messages("v0.9", "")}); err == nil || !strings.Contains(err.Error(), `must be "v0.9.1"`) {
		t.Errorf("wrong version not reported: %v", err)
	}

	// The envelope describes rather than enforces, so an unknown message key passes ADK's
	// pre-validation of the raw arguments and reaches v09.Validate, which names it.
	unknownKey := []any{map[string]any{"version": "v0.9.1", "makeSurface": map[string]any{"surfaceId": "s"}}}
	if _, err := tl.(runnable).Run(ctx, map[string]any{"messages": unknownKey}); err == nil || !strings.Contains(err.Error(), `unknown message key "makeSurface"`) {
		t.Errorf("unknown message key not reported: %v", err)
	}
}

func TestCatalogTool(t *testing.T) {
	params := kit.VersionParams{SupportedCatalogIDs: []string{v09.CatalogIDBasic}}
	tl, err := CatalogTool(kit.V09, params, kit.ToolOptions{})
	if err != nil {
		t.Fatal(err)
	}
	out, err := tl.(runnable).Run(fake(context.Background()), map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	cats, _ := out["catalogs"].(map[string]any)
	if _, ok := cats[v09.CatalogIDBasic]; !ok {
		t.Errorf("basic catalog not returned: %v", out)
	}
}
