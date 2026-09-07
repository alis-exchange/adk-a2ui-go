package kit

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestParseCapabilitiesVersionKeyed(t *testing.T) {
	doc := map[string]any{
		"v0.9": map[string]any{
			"supportedCatalogIds": []any{"https://a/cat.json"},
			"inlineCatalogs":      []any{map[string]any{"catalogId": "inline:1"}},
		},
		"v1.0": map[string]any{"supportedCatalogIds": []string{"https://b/cat.json"}},
	}
	caps, err := ParseCapabilities(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := Capabilities{
		V09: {SupportedCatalogIDs: []string{"https://a/cat.json"}, InlineCatalogs: []map[string]any{{"catalogId": "inline:1"}}},
		V10: {SupportedCatalogIDs: []string{"https://b/cat.json"}},
	}
	if !reflect.DeepEqual(caps, want) {
		t.Errorf("got %+v", caps)
	}
}

func TestParseCapabilitiesLegacyFlatMap(t *testing.T) {
	caps, err := ParseCapabilities(map[string]any{"supportedCatalogIds": []any{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := caps[V09].SupportedCatalogIDs; !reflect.DeepEqual(got, []string{"x"}) {
		t.Errorf("legacy map not wrapped as v0.9: %+v", caps)
	}
}

func TestParseCapabilitiesMixedShape(t *testing.T) {
	doc := map[string]any{
		"supportedCatalogIds": []any{"x"},
		"v1.0":                map[string]any{"supportedCatalogIds": []any{"y"}},
	}
	caps, err := ParseCapabilities(doc)
	if err == nil || !strings.Contains(err.Error(), "mixes a legacy flat shape") {
		t.Fatalf("expected mixed-shape error, got %v", err)
	}
	want := Capabilities{V10: {SupportedCatalogIDs: []string{"y"}}}
	if !reflect.DeepEqual(caps, want) {
		t.Errorf("got %+v", caps)
	}
}

func TestParseCapabilitiesErrors(t *testing.T) {
	if _, err := ParseCapabilities(nil); err == nil {
		t.Error("nil doc should error")
	}
	caps, err := ParseCapabilities(map[string]any{"v0.9": "nope", "v1.0": map[string]any{"supportedCatalogIds": []any{1}}})
	if err == nil {
		t.Error("malformed versions should error")
	}
	if len(caps) != 0 {
		t.Errorf("malformed versions must be dropped, got %+v", caps)
	}
}

func TestContextRoundTrip(t *testing.T) {
	ctx := WithA2UICapabilities(context.Background(), map[string]any{"v1.0": map[string]any{"supportedCatalogIds": []any{"c"}}})
	caps, ok := CapabilitiesFromContext(ctx)
	if !ok || caps[V10].SupportedCatalogIDs[0] != "c" {
		t.Errorf("got %+v %v", caps, ok)
	}
	if _, ok := CapabilitiesFromContext(context.Background()); ok {
		t.Error("empty context should report no capabilities")
	}
}

func TestNegotiate(t *testing.T) {
	caps := Capabilities{V09: {}, V091: {}, V10: {}}
	if v, _, ok := Negotiate(caps, nil); !ok || v != V10 {
		t.Errorf("default preference should pick v1.0, got %q", v)
	}
	if v, _, ok := Negotiate(caps, []string{V091, V09}); !ok || v != V091 {
		t.Errorf("restricted preference should pick v0.9.1, got %q", v)
	}
	if _, _, ok := Negotiate(Capabilities{V09: {}}, []string{V10}); ok {
		t.Error("no mutual version should fail")
	}
}
