package schema

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInboundPrecheck(t *testing.T) {
	keys := []string{"action", "error"}
	action := map[string]any{"name": "n"}
	cases := []struct {
		name    string
		m       map[string]any
		version string
		path    string
		want    []string
	}{
		{"ok", map[string]any{"version": "v0.9", "action": action}, "v0.9", "", nil},
		{"ok unpinned", map[string]any{"version": "v0.9.1", "action": action}, "", "", nil},
		{"wrong version", map[string]any{"version": "v1.0", "action": action}, "v0.9", "",
			[]string{`version: must be "v0.9"`}},
		{"wrong version in list", map[string]any{"version": "v1.0", "action": action}, "v0.9", "messages[2]",
			[]string{`messages[2].version: must be "v0.9"`}},
		{"no key", map[string]any{"version": "v0.9"}, "", "",
			[]string{`a message must contain exactly one of "action", "error"`}},
		{"two keys", map[string]any{"version": "v0.9", "action": action, "error": map[string]any{}}, "", "",
			[]string{`a message must contain exactly one of "action", "error"`}},
		{"unknown key", map[string]any{"version": "v0.9", "bogus": 1}, "", "messages[0]",
			[]string{`messages[0]: unknown message key "bogus"; a message must contain exactly one of "action", "error"`}},
		{"known plus unknown", map[string]any{"version": "v0.9", "action": action, "zzz": 1, "aaa": 2}, "", "",
			[]string{`unknown message key "aaa", "zzz"; a message must contain exactly one of "action", "error"`}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, p := range InboundPrecheck(tc.m, tc.version, keys, tc.path) {
				got = append(got, p.String())
			}
			if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
				t.Errorf("got:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(tc.want, "\n"))
			}
		})
	}
}

func TestJSONType(t *testing.T) {
	cases := []struct {
		name string
		v    any
		want string
	}{
		{"null", nil, "null"},
		{"boolean", true, "boolean"},
		{"number float64", float64(1), "number"},
		{"number json.Number", json.Number("1"), "number"},
		{"string", "s", "string"},
		{"array", []any{}, "array"},
		{"object", map[string]any{}, "object"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := JSONType(tc.v); got != tc.want {
				t.Errorf("JSONType(%#v) = %q, want %q", tc.v, got, tc.want)
			}
		})
	}
}

func TestJoinPath(t *testing.T) {
	if got := JoinPath("", "version"); got != "version" {
		t.Errorf("JoinPath(\"\", version) = %q", got)
	}
	if got := JoinPath("messages[1]", "action.name"); got != "messages[1].action.name" {
		t.Errorf("got %q", got)
	}
}
