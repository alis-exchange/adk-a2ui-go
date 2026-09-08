package schema

import (
	"encoding/json"
	"reflect"
	"strings"
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

func functionCallSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	cat, _, _, err := spec.BasicCatalog(spec.MajorV10)
	if err != nil {
		t.Fatal(err)
	}
	s, err := For(spec.MajorV10).CompileRef("common_types.json#/$defs/FunctionCall", cat, false)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestFormatPrunesFunctionUnionByCall(t *testing.T) {
	s := functionCallSchema(t)
	basic := "https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"
	cases := []struct {
		name string
		call map[string]any
		want []string // exact problem lines, in order
	}{
		{"flat args", map[string]any{"call": "formatDate", "catalogId": basic, "value": "x", "format": "y"},
			[]string{`cf: missing property "args"`}},
		{"missing arg", map[string]any{"call": "formatDate", "catalogId": basic, "args": map[string]any{"value": "x"}},
			[]string{`cf.args: missing property "format"`}},
		{"unknown arg", map[string]any{"call": "formatDate", "catalogId": basic, "args": map[string]any{"value": "x", "format": "y", "zzz": 1}},
			[]string{`cf.args: unknown property "zzz"`}},
		{"unknown function", map[string]any{"call": "noSuchFunction", "catalogId": basic, "args": map[string]any{}},
			[]string{`cf: unknown function "noSuchFunction" (catalog functions: and, email, formatCurrency, formatDate, formatNumber, formatString, length, not, numeric, openUrl, or, pluralize, regex, required)`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.Validate(tc.call)
			if err == nil {
				t.Fatal("expected a validation error")
			}
			var got []string
			for _, p := range Format(err, tc.call, "cf") {
				got = append(got, p.String())
			}
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tc.want, "\n"))
			}
		})
	}
}

func TestFormatIndexFunctionBranch(t *testing.T) {
	s := functionCallSchema(t)
	call := map[string]any{"call": "@index", "args": map[string]any{"offset": "bad"}}
	err := s.Validate(call)
	if err == nil {
		t.Fatal("expected a validation error")
	}
	text := ""
	for _, p := range Format(err, call, "cf") {
		text += p.String() + "\n"
	}
	if !strings.Contains(text, "cf.args.offset: must be of type number") || strings.Contains(text, `must be "required"`) {
		t.Errorf("@index must select the IndexSystemFunction branch, got:\n%s", text)
	}
}

func checkRuleSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	cat, _, _, err := spec.BasicCatalog(spec.MajorV10)
	if err != nil {
		t.Fatal(err)
	}
	s, err := For(spec.MajorV10).CompileRef("common_types.json#/$defs/CheckRule", cat, false)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// CheckRule.condition is oneOf[DataBinding, FunctionCall]: a call there sits one union above the
// catalog's function union, and the pruning must find its way down instead of reporting the
// two branch names as "catalog functions".
func TestFormatPrunesCallsInsideCheckRule(t *testing.T) {
	s := checkRuleSchema(t)
	cases := []struct {
		name string
		rule map[string]any
		want []string
	}{
		{"missing arg", map[string]any{"condition": map[string]any{"call": "required", "args": map[string]any{}}},
			[]string{`rule.condition.args: missing property "value"`}},
		{"unknown function", map[string]any{"condition": map[string]any{"call": "nope", "args": map[string]any{}}},
			[]string{`rule.condition: unknown function "nope" (catalog functions: and, email, formatCurrency, formatDate, formatNumber, formatString, length, not, numeric, openUrl, or, pluralize, regex, required)`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := s.Validate(tc.rule)
			if err == nil {
				t.Fatal("expected a validation error")
			}
			var got []string
			for _, p := range Format(err, tc.rule, "rule") {
				got = append(got, p.String())
			}
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tc.want, "\n"))
			}
		})
	}
}
