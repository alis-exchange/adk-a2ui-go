package schema

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"go.alis.build/adk/a2ui/spec"
)

func formatV09(t *testing.T, src string) []Problem {
	t.Helper()
	cat, _, _, _ := spec.BasicCatalog(spec.MajorV09)
	s, err := For(spec.MajorV09).Compile(CompileOptions{Entry: EntryOutboundV09, Catalog: cat})
	if err != nil {
		t.Fatal(err)
	}
	var inst []any
	if err := json.Unmarshal([]byte(src), &inst); err != nil {
		t.Fatal(err)
	}
	err = s.Validate(inst)
	if err == nil {
		t.Fatal("expected a validation error")
	}
	return Format(err, inst, "messages")
}

func wrap(component string) string {
	return `[{"version":"v0.9","createSurface":{"surfaceId":"s","catalogId":"` + basicV09 + `"}},
	         {"version":"v0.9","updateComponents":{"surfaceId":"s","components":[` + component + `]}}]`
}

func TestFormatGolden(t *testing.T) {
	const comp = "messages[1].updateComponents.components[0]"
	cases := []struct {
		name string
		src  string
		want []Problem
	}{
		{"unknown property", wrap(`{"id":"root","component":"Text","text":"hi","colour":"red"}`),
			[]Problem{{Path: comp, Message: `unknown property "colour"`}}},
		{"missing required", wrap(`{"id":"root","component":"Text"}`),
			[]Problem{{Path: comp, Message: `missing property "text"`}}},
		{"bad enum", wrap(`{"id":"root","component":"Text","text":"hi","variant":"huge"}`),
			[]Problem{{Path: comp + ".variant", Message: `must be one of "h1", "h2", "h3", "h4", "h5", "caption", "body"`}}},
		{"unknown component", wrap(`{"id":"root","component":"Sparkle"}`),
			[]Problem{{Path: comp, Message: `unknown component "Sparkle" (catalog components: AudioPlayer, Button, Card, CheckBox, ChoicePicker, Column, DateTimeInput, Divider, Icon, Image, List, Modal, Row, Slider, Tabs, Text, TextField, Video)`}}},
		{"unknown message key", `[{"version":"v0.9","makeSurface":{"surfaceId":"s"}}]`,
			[]Problem{{Path: "messages[0]", Message: `unknown message key "makeSurface"; a message must contain exactly one of "createSurface", "deleteSurface", "updateComponents", "updateDataModel"`}}},
		{"wrong version", `[{"version":"v1.0","createSurface":{"surfaceId":"s","catalogId":"x"}}]`,
			[]Problem{{Path: "messages[0].version", Message: `must be "v0.9"`}}},
		{"unknown properties sorted", `[{"version":"v0.9","createSurface":{"surfaceId":"s","catalogId":"` + basicV09 + `","zzz":1,"aaa":2,"mmm":3}}]`,
			[]Problem{{Path: "messages[0].createSurface", Message: `unknown properties "aaa", "mmm", "zzz"`}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatV09(t, tc.src)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

// TestFormatEmptyOneOfCausesDoesNotPanic covers an ambiguous oneOf (more than one subschema
// matched) where the validator reports *kind.OneOf with nil Causes. walkOneOf must not index
// branchNames[0] when no Reference-shaped causes were collected, and must instead let the
// generic walk fall back to the OneOf kind's own LocalizedString.
func TestFormatEmptyOneOfCausesDoesNotPanic(t *testing.T) {
	ve := &jsonschema.ValidationError{
		InstanceLocation: []string{"foo"},
		ErrorKind:        &kind.OneOf{Subschemas: []int{0, 1}},
		Causes:           nil,
	}
	instance := map[string]any{"foo": map[string]any{"bar": 1.0}}
	got := Format(ve, instance, "root")
	if len(got) != 1 {
		t.Fatalf("expected exactly one Problem, got %d: %+v", len(got), got)
	}
	if got[0].Path != "root.foo" {
		t.Errorf("got Path %q, want %q", got[0].Path, "root.foo")
	}
}
