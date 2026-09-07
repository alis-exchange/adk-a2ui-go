package schema

import (
	"encoding/json"
	"fmt"
	"sort"
)

// InboundPrecheck reports the two findings the inbound entry schemas render badly: a "version"
// that differs from the pinned one, and a message that does not carry exactly one of keys
// beside "version". client_to_server.json and renderer_to_agent.json express the key rule as a
// oneOf of bare required branches (not $refs), which Format cannot prune, so a message with no
// key or an unknown one would otherwise come back as one "missing property" line per key, and
// two keys as a maxProperties count. Callers stop at these findings; the schema pass would only
// add noise, the same rule Validate applies after its envelope pass.
func InboundPrecheck(m map[string]any, version string, keys []string, path string) []Problem {
	var out []Problem
	if got, _ := m["version"].(string); version != "" && got != version {
		out = append(out, Problem{Path: JoinPath(path, "version"), Message: fmt.Sprintf("must be %q", version)})
	}
	var present, unknown []string
	for k := range m {
		switch {
		case k == "version":
		case contains(keys, k):
			present = append(present, k)
		default:
			unknown = append(unknown, k)
		}
	}
	if len(present) == 1 && len(unknown) == 0 {
		return out
	}
	sort.Strings(unknown)
	msg := "a message must contain exactly one of " + quoteList(keys)
	if len(unknown) > 0 {
		msg = "unknown message key " + quoteList(unknown) + "; " + msg
	}
	return append(out, Problem{Path: path, Message: msg})
}

// JSONType names a decoded JSON value's type the way the validator's own messages do
// ("must be of type object, got array"), so hand-written problems read like schema ones.
func JSONType(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64, json.Number:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return fmt.Sprintf("%T", v)
}

// JoinPath appends a rendered segment to a path prefix, which may be empty for a message
// decoded on its own ("action.name") or "messages[i]" for one decoded from a list.
func JoinPath(prefix, segment string) string {
	if prefix == "" {
		return segment
	}
	return prefix + "." + segment
}
