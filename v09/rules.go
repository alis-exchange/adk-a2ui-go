package v09

import (
	"fmt"

	"go.alis.build/adk/a2ui/internal/schema"
)

// semanticRules enforces what the spec states in prose rather than schema: surface ids are
// non-empty and created once per batch, component ids are unique per list (which is also what
// keeps a list to at most one "root"), and every surface created in the batch ends up with a
// root component.
func semanticRules(messages []map[string]any) []schema.Problem {
	var out []schema.Problem
	created := map[string]int{}
	var createdOrder []string
	hasRoot := map[string]bool{}
	for i, m := range messages {
		if cs, ok := m["createSurface"].(map[string]any); ok {
			sid, _ := cs["surfaceId"].(string)
			if sid == "" {
				out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].createSurface.surfaceId", i), Message: "must not be empty"})
				continue
			}
			if _, dup := created[sid]; dup {
				out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].createSurface", i), Message: fmt.Sprintf("surface %q is created twice in this batch", sid)})
				continue
			}
			created[sid] = i
			createdOrder = append(createdOrder, sid)
		}
		if uc, ok := m["updateComponents"].(map[string]any); ok {
			sid, _ := uc["surfaceId"].(string)
			if sid == "" {
				out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].updateComponents.surfaceId", i), Message: "must not be empty"})
			}
			comps, _ := uc["components"].([]any)
			seen := map[string]int{}
			roots := 0
			for j, c := range comps {
				obj, _ := c.(map[string]any)
				id, _ := obj["id"].(string)
				if id == "" {
					out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].updateComponents.components[%d].id", i, j), Message: "must not be empty"})
					continue
				}
				if k, dup := seen[id]; dup {
					out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].updateComponents.components[%d].id", i, j), Message: fmt.Sprintf("duplicate component id %q (also used at components[%d])", id, k)})
					continue
				}
				seen[id] = j
				if id == "root" {
					roots++
				}
			}
			// roots can only ever be 0 or 1: a second component with id "root" is a duplicate id
			// and was already reported above. roots is counted only to answer "does this surface
			// end up with a root at all".
			if roots >= 1 {
				hasRoot[sid] = true
			}
		}
		for _, key := range []string{"updateDataModel", "deleteSurface"} {
			if obj, ok := m[key].(map[string]any); ok {
				if sid, _ := obj["surfaceId"].(string); sid == "" {
					out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].%s.surfaceId", i, key), Message: "must not be empty"})
				}
			}
		}
	}
	for _, sid := range createdOrder {
		if !hasRoot[sid] {
			out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].createSurface", created[sid]), Message: fmt.Sprintf("surface %q has no component with id \"root\" in any updateComponents of this batch", sid)})
		}
	}
	return out
}
