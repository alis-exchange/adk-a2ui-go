package v09

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.alis.build/adk/a2ui/internal/schema"
	"go.alis.build/adk/a2ui/kit"
)

type negative struct {
	Name         string           `json:"name"`
	Version      string           `json:"version"`
	Strict       bool             `json:"strict"`
	Messages     []map[string]any `json:"messages"`
	WantPaths    []string         `json:"wantPaths"`
	WantContains []string         `json:"wantContains"`
}

func TestNegatives(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("testdata", "negative", "*.json"))
	if len(files) == 0 {
		t.Fatal("no negative fixtures")
	}
	for _, f := range files {
		b, _ := os.ReadFile(f)
		var n negative
		if err := json.Unmarshal(b, &n); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		t.Run(n.Name, func(t *testing.T) {
			err := Validate(context.Background(), n.Messages, kit.ValidateOptions{Version: n.Version, Strict: n.Strict})
			var ve *schema.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want *schema.ValidationError, got %v", err)
			}
			text := ve.Error()
			for _, p := range n.WantPaths {
				found := false
				for _, pr := range ve.Problems {
					if pr.Path == p {
						found = true
					}
				}
				if !found {
					t.Errorf("path %q not reported in:\n%s", p, text)
				}
			}
			for _, s := range n.WantContains {
				if !strings.Contains(text, s) {
					t.Errorf("%q not in:\n%s", s, text)
				}
			}
		})
	}
}

func loadExample(t *testing.T, file string) []map[string]any {
	t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Messages []map[string]any `json:"messages"`
	}
	if err := json.Unmarshal(b, &doc); err != nil || doc.Messages == nil {
		var list []map[string]any
		if err := json.Unmarshal(b, &list); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		return list
	}
	return doc.Messages
}

func TestOfficialExamplesPassValidate(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "spec", "v0_9", "testdata", "examples", "*.json"))
	if len(files) == 0 {
		t.Fatal("no examples; run scripts/sync-spec.sh")
	}
	for _, version := range []string{kit.V09, kit.V091} {
		for _, f := range files {
			msgs := loadExample(t, f)
			for _, m := range msgs {
				m["version"] = version
			}
			if err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: version, Strict: true}); err != nil {
				t.Errorf("%s as %s: %v", filepath.Base(f), version, err)
			}
		}
	}
}

func TestDeleteThenRecreateIsAllowed(t *testing.T) {
	msgs := []map[string]any{
		{"version": "v0.9", "deleteSurface": map[string]any{"surfaceId": "s"}},
		{"version": "v0.9", "createSurface": map[string]any{"surfaceId": "s", "catalogId": CatalogIDBasic}},
		{"version": "v0.9", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{map[string]any{"id": "root", "component": "Text", "text": "hi"}}}},
	}
	if err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V09, Strict: true}); err != nil {
		t.Errorf("deleting and recreating a surface in one batch is the spec's reconfigure flow: %v", err)
	}
}

func TestGracefulUnknownCatalog(t *testing.T) {
	msgs := []map[string]any{
		{"version": "v0.9", "createSurface": map[string]any{"surfaceId": "s", "catalogId": "https://example.com/custom.json"}},
		{"version": "v0.9", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{
			map[string]any{"id": "root", "component": "Whatever", "anything": 1.0},
		}}},
	}
	if err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V09}); err != nil {
		t.Errorf("graceful mode should accept unknown catalog: %v", err)
	}
	msgs[1]["updateComponents"].(map[string]any)["components"] = []any{map[string]any{"component": "Whatever"}}
	if err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V09}); err == nil {
		t.Error("graceful mode must still require id")
	}
}

func TestInlineAndRegisteredCatalogsAreUsed(t *testing.T) {
	custom := map[string]any{
		"catalogId": "acme:ui",
		"$defs": map[string]any{
			"theme":        map[string]any{"type": "object"},
			"anyComponent": map[string]any{"oneOf": []any{map[string]any{"$ref": "#/components/Badge"}}},
		},
		"components": map[string]any{"Badge": map[string]any{
			"type":       "object",
			"properties": map[string]any{"id": map[string]any{"type": "string"}, "component": map[string]any{"const": "Badge"}, "label": map[string]any{"type": "string"}},
			"required":   []any{"id", "component", "label"}, "additionalProperties": false,
		}},
	}
	msgs := func(comp map[string]any) []map[string]any {
		return []map[string]any{
			{"version": "v0.9", "createSurface": map[string]any{"surfaceId": "s", "catalogId": "acme:ui"}},
			{"version": "v0.9", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{comp}}},
		}
	}
	good := map[string]any{"id": "root", "component": "Badge", "label": "x"}
	bad := map[string]any{"id": "root", "component": "Badge"}
	inline := kit.ValidateOptions{Version: kit.V09, Params: kit.VersionParams{InlineCatalogs: []map[string]any{custom}}, Strict: true}
	if err := Validate(context.Background(), msgs(good), inline); err != nil {
		t.Errorf("inline catalog: %v", err)
	}
	if err := Validate(context.Background(), msgs(bad), inline); err == nil || !strings.Contains(err.Error(), `missing property "label"`) {
		t.Errorf("inline catalog should enforce Badge.label, got %v", err)
	}
	reg := kit.NewRegistry()
	_ = reg.Register(custom)
	registered := kit.ValidateOptions{Version: kit.V09, Resolver: reg, Strict: true}
	if err := Validate(context.Background(), msgs(bad), registered); err == nil {
		t.Error("registered catalog should enforce Badge.label")
	}
}

func TestStrictSkipsSurfacesNotCreatedInBatch(t *testing.T) {
	msgs := []map[string]any{
		{"version": "v0.9", "updateComponents": map[string]any{"surfaceId": "pre-existing", "components": []any{
			map[string]any{"id": "root", "component": "Anything", "x": 1.0},
		}}},
	}
	if err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V09, Strict: true}); err != nil {
		t.Errorf("surface not created in this batch should skip catalog checks even in strict mode: %v", err)
	}
}
