package v10

import (
	"fmt"

	"go.alis.build/adk/a2ui/internal/schema"
)

// checkList enforces per-list component rules: ids are non-empty and unique, and at most one
// component has id "root". It reports how many roots it found so the caller can track whether a
// surface (whose components may be split across createSurface.components and one or more
// updateComponents.components lists) ends up with a root at all.
func checkList(comps []any, basePath string) (int, []schema.Problem) {
	var out []schema.Problem
	seen := map[string]int{}
	roots := 0
	for j, c := range comps {
		obj, _ := c.(map[string]any)
		id, _ := obj["id"].(string)
		if id == "" {
			out = append(out, schema.Problem{Path: fmt.Sprintf("%s[%d].id", basePath, j), Message: "must not be empty"})
			continue
		}
		if k, dup := seen[id]; dup {
			out = append(out, schema.Problem{Path: fmt.Sprintf("%s[%d].id", basePath, j), Message: fmt.Sprintf("duplicate component id %q (also used at index %d)", id, k)})
			continue
		}
		seen[id] = j
		if id == "root" {
			roots++
		}
	}
	if roots > 1 {
		out = append(out, schema.Problem{Path: basePath, Message: `more than one component has id "root"`})
	}
	return roots, out
}

// semanticRules enforces what the spec states in prose rather than schema: surface ids are
// non-empty and created once per batch, component ids are unique per list with at most one
// "root" per list, and every surface created in the batch ends up with a root component either
// in createSurface.components or in a later updateComponents.
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
			if comps, ok := cs["components"].([]any); ok {
				roots, p := checkList(comps, fmt.Sprintf("messages[%d].createSurface.components", i))
				out = append(out, p...)
				if roots >= 1 {
					hasRoot[sid] = true
				}
			}
		}
		if uc, ok := m["updateComponents"].(map[string]any); ok {
			sid, _ := uc["surfaceId"].(string)
			if sid == "" {
				out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].updateComponents.surfaceId", i), Message: "must not be empty"})
			}
			comps, _ := uc["components"].([]any)
			roots, p := checkList(comps, fmt.Sprintf("messages[%d].updateComponents.components", i))
			out = append(out, p...)
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
			out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].createSurface", created[sid]), Message: fmt.Sprintf("surface %q has no component with id \"root\" in createSurface.components or any updateComponents of this batch", sid)})
		}
	}
	return out
}
