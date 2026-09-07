package schema

import (
	"encoding/json"
	"reflect"
	"testing"

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
			[]Problem{{comp, `unknown property "colour"`}}},
		{"missing required", wrap(`{"id":"root","component":"Text"}`),
			[]Problem{{comp, `missing property "text"`}}},
		{"bad enum", wrap(`{"id":"root","component":"Text","text":"hi","variant":"huge"}`),
			[]Problem{{comp + ".variant", `must be one of "h1", "h2", "h3", "h4", "h5", "caption", "body"`}}},
		{"unknown component", wrap(`{"id":"root","component":"Sparkle"}`),
			[]Problem{{comp, `unknown component "Sparkle" (catalog components: AudioPlayer, Button, Card, CheckBox, ChoicePicker, Column, DateTimeInput, Divider, Icon, Image, List, Modal, Row, Slider, Tabs, Text, TextField, Video)`}}},
		{"unknown message key", `[{"version":"v0.9","makeSurface":{"surfaceId":"s"}}]`,
			[]Problem{{"messages[0]", `unknown message key "makeSurface"; a message must contain exactly one of "createSurface", "deleteSurface", "updateComponents", "updateDataModel"`}}},
		{"wrong version", `[{"version":"v1.0","createSurface":{"surfaceId":"s","catalogId":"x"}}]`,
			[]Problem{{"messages[0].version", `must be "v0.9"`}}},
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
