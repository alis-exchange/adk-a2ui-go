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

func TestStrictReportsMissingCatalogOnce(t *testing.T) {
	msgs := []map[string]any{
		{"version": "v1.0", "createSurface": map[string]any{"surfaceId": "s", "catalogId": "https://example.com/nope.json"}},
		{"version": "v1.0", "updateComponents": map[string]any{"surfaceId": "s", "components": []any{
			map[string]any{"id": "root", "component": "Text", "text": "hi"},
			map[string]any{"id": "b", "component": "Text", "text": "bye", "catalogId": "https://example.com/nope.json"},
		}}},
	}
	err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V10, Strict: true})
	var ve *schema.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want *schema.ValidationError, got %v", err)
	}
	if len(ve.Problems) != 1 {
		t.Fatalf("want exactly one problem (missing catalog reported once), got %d: %v", len(ve.Problems), ve.Problems)
	}
	p := ve.Problems[0]
	if !strings.Contains(p.Message, "not available") {
		t.Errorf("message = %q, want it to contain %q", p.Message, "not available")
	}
	if p.Path != "messages[0].createSurface.catalogId" {
		t.Errorf("path = %q, want %q", p.Path, "messages[0].createSurface.catalogId")
	}
}

// TestValidationErrorIsPublic proves a consumer can match a Validate failure without importing
// an internal package: the error a validator returns is a *kit.ValidationError, and its problem
// list is reachable through errors.As.
func TestValidationErrorIsPublic(t *testing.T) {
	msgs := []map[string]any{{"version": "v0.9", "deleteSurface": map[string]any{"surfaceId": "s"}}}
	err := Validate(context.Background(), msgs, kit.ValidateOptions{Version: kit.V10})
	var ve *kit.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want *kit.ValidationError, got %#v", err)
	}
	if len(ve.Problems) == 0 {
		t.Fatal("Problems is empty")
	}
	if ve.Problems[0].Path != "messages[0].version" || !strings.Contains(ve.Problems[0].Message, `must be "v1.0"`) {
		t.Errorf("Problems[0] = %+v", ve.Problems[0])
	}
	if !strings.Contains(err.Error(), "validation failed. Fix the following and call the tool again:") {
		t.Errorf("Error() = %q", err.Error())
	}
}
