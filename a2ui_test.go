package a2ui

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool/toolconfirmation"

	"go.alis.build/adk/a2ui/kit"
)

type fakeContext struct{ agent.StrictContextMock }

func (f *fakeContext) ToolConfirmation() *toolconfirmation.ToolConfirmation { return nil }

func fake(ctx context.Context) *fakeContext { return &fakeContext{agent.NewStrictContextMock(ctx)} }

func caps(versions ...string) context.Context {
	doc := map[string]any{}
	for _, v := range versions {
		doc[v] = map[string]any{"supportedCatalogIds": []any{"cat:" + v}}
	}
	return kit.WithA2UICapabilities(context.Background(), doc)
}

func versionOf(t *testing.T, ctx context.Context, opts ...Option) string {
	t.Helper()
	ts, err := NewToolset(opts...)
	if err != nil {
		t.Fatal(err)
	}
	tools, err := ts.Tools(fake(ctx))
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) == 0 {
		return ""
	}
	if len(tools) != 2 || tools[0].Name() != "a2ui_catalog" || tools[1].Name() != "generate_a2ui_messages" {
		t.Fatalf("unexpected tools %v", tools)
	}
	d := tools[1].Description()
	for _, v := range kit.KnownVersions {
		if strings.Contains(d, `"version": "`+v+`"`) {
			return v
		}
	}
	t.Fatalf("no version in description")
	return ""
}

func TestNegotiatedToolset(t *testing.T) {
	if got := versionOf(t, context.Background()); got != "" {
		t.Errorf("no capabilities: got %q", got)
	}
	if got := versionOf(t, caps(kit.V09)); got != kit.V09 {
		t.Errorf("v0.9 only: got %q", got)
	}
	if got := versionOf(t, caps(kit.V09, kit.V10)); got != kit.V10 {
		t.Errorf("both: want newest, got %q", got)
	}
	if got := versionOf(t, caps(kit.V09, kit.V10), WithVersions(kit.V091, kit.V09)); got != kit.V09 {
		t.Errorf("restricted: got %q", got)
	}
	if got := versionOf(t, caps(kit.V091), WithVersions(kit.V10)); got != "" {
		t.Errorf("no mutual version: got %q", got)
	}
	legacy := kit.WithA2UICapabilities(context.Background(), map[string]any{"supportedCatalogIds": []any{"x"}})
	if got := versionOf(t, legacy); got != kit.V09 {
		t.Errorf("legacy flat map: got %q", got)
	}
	if ts, _ := NewToolset(); ts.Name() != "a2ui" {
		t.Error("toolset name")
	}
	if _, err := NewToolset(WithVersions("v7")); err == nil {
		t.Error("unknown version must be rejected")
	}
}
