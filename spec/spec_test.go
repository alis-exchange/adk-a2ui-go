package spec

import "testing"

func TestEmbeddedFiles(t *testing.T) {
	for _, p := range []string{
		"v0_9/json/server_to_client.json", "v0_9/json/common_types.json", "v0_9/catalogs/basic/catalog.json",
		"v0_9/catalogs/basic/rules.txt", "v1_0/json/agent_to_renderer.json", "v1_0/json/catalog_definition.json",
		"v1_0/catalogs/basic/catalog.json",
	} {
		if _, err := FS.ReadFile(p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	if Source == "" {
		t.Error("Source is empty; run scripts/sync-spec.sh")
	}
}

func TestBasicCatalog(t *testing.T) {
	cat, id, ins, err := BasicCatalog(MajorV09)
	if err != nil {
		t.Fatal(err)
	}
	if id != "https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json" {
		t.Errorf("v0_9 id = %q", id)
	}
	if _, ok := cat["components"].(map[string]any); !ok {
		t.Error("v0_9 catalog has no components map")
	}
	if ins == "" {
		t.Error("v0_9 instructions (rules.txt) empty")
	}
	_, id10, ins10, err := BasicCatalog(MajorV10)
	if err != nil {
		t.Fatal(err)
	}
	if id10 != "https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json" || ins10 == "" {
		t.Errorf("v1_0 id=%q instructions empty=%v", id10, ins10 == "")
	}
	if _, _, _, err := BasicCatalog("v9_9"); err == nil {
		t.Error("unknown major should error")
	}
}

func TestMajorFor(t *testing.T) {
	for v, want := range map[string]string{"v0.9": MajorV09, "v0.9.1": MajorV09, "v1.0": MajorV10} {
		if got, ok := MajorFor(v); !ok || got != want {
			t.Errorf("MajorFor(%q) = %q,%v", v, got, ok)
		}
	}
	if _, ok := MajorFor("v2.0"); ok {
		t.Error("v2.0 should be unknown")
	}
}
