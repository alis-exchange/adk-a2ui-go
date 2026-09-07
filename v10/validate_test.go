package v10

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
	files, _ := filepath.Glob(filepath.Join("..", "spec", "v1_0", "testdata", "examples", "*.json"))
	if len(files) == 0 {
		t.Fatal("no examples; run scripts/sync-spec.sh")
	}
	for _, f := range files {
		msgs := loadExample(t, f)
		if err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V10, Strict: true}); err != nil {
			t.Errorf("%s: %v", filepath.Base(f), err)
		}
	}
}

func TestRootInsideCreateSurface(t *testing.T) {
	msgs := []map[string]any{{
		"version": "v1.0",
		"createSurface": map[string]any{
			"surfaceId": "s", "catalogId": CatalogIDBasic,
			"components": []any{map[string]any{"id": "root", "component": "Text", "text": "hi"}},
		},
	}}
	if err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V10, Strict: true}); err != nil {
		t.Errorf("root inside createSurface.components must be accepted: %v", err)
	}
}

func TestPerComponentCatalogID(t *testing.T) {
	msgs := []map[string]any{
		{"version": "v1.0", "createSurface": map[string]any{"surfaceId": "s"}},
		{"version": "v1.0", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{
			map[string]any{"id": "root", "component": "Text", "text": "hi", "catalogId": CatalogIDBasic},
		}}},
	}
	if err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V10, Strict: true}); err != nil {
		t.Errorf("per-component catalogId must satisfy the rule: %v", err)
	}
	msgs[1]["updateComponents"].(map[string]any)["components"].([]any)[0].(map[string]any)["text"] = 5.0
	if err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V10, Strict: true}); err == nil {
		t.Error("component must be validated against its own catalog")
	}
}

func TestCallRendererFunctionValidated(t *testing.T) {
	msgs := []map[string]any{{
		"version": "v1.0",
		"callRendererFunction": map[string]any{
			"functionCallId": "c1",
			"callFunction":   map[string]any{"call": "noSuchFunction", "catalogId": CatalogIDBasic},
		},
	}}
	err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V10, Strict: true})
	if err == nil || !strings.Contains(err.Error(), "messages[0].callRendererFunction.callFunction") {
		t.Errorf("unknown function should be reported at the call path, got %v", err)
	}
}
