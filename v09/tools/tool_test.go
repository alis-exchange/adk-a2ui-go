package tools

import (
	"context"
	"strings"
	"testing"

	"go.alis.build/adk/a2ui/kit"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool/toolconfirmation"
)

// fakeContext is the smallest agent.Context that lets a function tool run outside a runner.
// StrictContextMock panics on anything not overridden, so only the paths the tools touch are
// implemented: context values (via Ctx) and the confirmation check inside functiontool.Run.
type fakeContext struct {
	agent.StrictContextMock
}

func (f *fakeContext) ToolConfirmation() *toolconfirmation.ToolConfirmation { return nil }

func newFakeContext(ctx context.Context) *fakeContext {
	return &fakeContext{agent.NewStrictContextMock(ctx)}
}

// runnable mirrors the unexported interface ADK uses to execute function tools.
type runnable interface {
	Run(ctx agent.Context, args any) (map[string]any, error)
}

func withCapabilities(ctx context.Context) context.Context {
	return kit.WithA2UICapabilities(ctx, map[string]any{
		"supportedCatalogIds": []any{"https://example.com/catalog.json"},
		"inlineCatalogs":      []any{map[string]any{"name": "inline"}},
	})
}

func TestToolsetHiddenWithoutCapabilities(t *testing.T) {
	ts, err := NewA2UIToolset()
	if err != nil {
		t.Fatalf("NewA2UIToolset: %v", err)
	}
	tools, err := ts.Tools(newFakeContext(t.Context()))
	if err != nil {
		t.Fatalf("Tools: %v", err)
	}
	if len(tools) != 0 {
		t.Fatalf("expected no tools without capabilities, got %d", len(tools))
	}
}

func TestToolsetVisibleWithCapabilities(t *testing.T) {
	ts, err := NewA2UIToolset()
	if err != nil {
		t.Fatalf("NewA2UIToolset: %v", err)
	}
	tools, err := ts.Tools(newFakeContext(withCapabilities(t.Context())))
	if err != nil {
		t.Fatalf("Tools: %v", err)
	}
	want := []string{CatalogToolName, GenerateA2UIMessagesToolName}
	if len(tools) != len(want) {
		t.Fatalf("expected %d tools, got %d", len(want), len(tools))
	}
	for i, name := range want {
		if tools[i].Name() != name {
			t.Errorf("tool[%d] = %q, want %q", i, tools[i].Name(), name)
		}
	}
}

func TestCatalogToolReturnsCapabilities(t *testing.T) {
	tl, err := A2UICatalogTool()
	if err != nil {
		t.Fatalf("A2UICatalogTool: %v", err)
	}
	out, err := tl.(runnable).Run(newFakeContext(withCapabilities(t.Context())), map[string]any{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	urls, _ := out["catalog_urls"].([]any)
	if len(urls) != 1 || urls[0] != "https://example.com/catalog.json" {
		t.Errorf("catalog_urls = %v", out["catalog_urls"])
	}
	inline, _ := out["inline_catalogs"].([]any)
	if len(inline) != 1 || inline[0] != `{"name":"inline"}` {
		t.Errorf("inline_catalogs = %v", out["inline_catalogs"])
	}
}

func TestCatalogToolWithoutCapabilitiesReturnsEmpty(t *testing.T) {
	tl, err := A2UICatalogTool()
	if err != nil {
		t.Fatalf("A2UICatalogTool: %v", err)
	}
	out, err := tl.(runnable).Run(newFakeContext(t.Context()), map[string]any{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if urls, _ := out["catalog_urls"].([]any); len(urls) != 0 {
		t.Errorf("catalog_urls = %v, want empty", urls)
	}
}

func surfaceMessages(withRoot bool) []any {
	rootID := "card-1"
	if withRoot {
		rootID = "root"
	}
	return []any{
		map[string]any{
			"version": "v0.9",
			"createSurface": map[string]any{
				"surfaceId": "s1",
				"catalogId": "https://example.com/catalog.json",
			},
		},
		map[string]any{
			"version": "v0.9",
			"updateComponents": map[string]any{
				"surfaceId": "s1",
				"components": []any{
					map[string]any{"component": "Card", "id": rootID, "child": "t1"},
					map[string]any{"component": "Text", "id": "t1", "text": "hi"},
				},
			},
		},
	}
}

func TestGenerateAcceptsValidMessages(t *testing.T) {
	tl, err := GenerateA2UIMessages()
	if err != nil {
		t.Fatalf("GenerateA2UIMessages: %v", err)
	}
	out, err := tl.(runnable).Run(newFakeContext(t.Context()), map[string]any{"messages": surfaceMessages(true)})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out["status"] != "success" || out["is_valid"] != true {
		t.Errorf("unexpected output: %v", out)
	}
	if msgs, _ := out["messages"].([]any); len(msgs) != 2 {
		t.Errorf("messages not echoed: %v", out["messages"])
	}
}

func TestGenerateRejectsMissingRoot(t *testing.T) {
	tl, err := GenerateA2UIMessages()
	if err != nil {
		t.Fatalf("GenerateA2UIMessages: %v", err)
	}
	_, err = tl.(runnable).Run(newFakeContext(t.Context()), map[string]any{"messages": surfaceMessages(false)})
	if err == nil || !strings.Contains(err.Error(), "root") {
		t.Fatalf("expected root-component error, got %v", err)
	}
}

func TestGenerateRejectsWrongVersion(t *testing.T) {
	tl, err := GenerateA2UIMessages()
	if err != nil {
		t.Fatalf("GenerateA2UIMessages: %v", err)
	}
	msgs := surfaceMessages(true)
	msgs[0].(map[string]any)["version"] = "v1.0"
	_, err = tl.(runnable).Run(newFakeContext(t.Context()), map[string]any{"messages": msgs})
	if err == nil {
		t.Fatal("expected schema error for wrong version, got nil")
	}
}
