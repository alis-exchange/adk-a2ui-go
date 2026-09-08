package v10

import (
	"fmt"

	"go.alis.build/adk/a2ui/internal/schema"
)

// checkList enforces per-list component rules: ids are non-empty and unique (which also keeps a
// list to at most one component with id "root"), and a catalogId, when present, is not empty
// (the schema has no minLength, and an empty id would silently skip every catalog check). It
// reports how many roots it found (0 or 1) so the caller can track whether a surface, whose
// components may be split across createSurface.components and updateComponents lists, ends up
// with a root at all.
func checkList(comps []any, basePath string) (int, []schema.Problem) {
	var out []schema.Problem
	seen := map[string]int{}
	roots := 0
	for j, c := range comps {
		obj, _ := c.(map[string]any)
		if v, present := obj["catalogId"]; present && v == "" {
			out = append(out, schema.Problem{Path: fmt.Sprintf("%s[%d].catalogId", basePath, j), Message: "must not be empty"})
		}
		id, _ := obj["id"].(string)
		if id == "" {
			out = append(out, schema.Problem{Path: fmt.Sprintf("%s[%d].id", basePath, j), Message: "must not be empty"})
			continue
		}
		if k, dup := seen[id]; dup {
			out = append(out, schema.Problem{Path: fmt.Sprintf("%s[%d].id", basePath, j), Message: fmt.Sprintf("duplicate component id %q (also used at components[%d])", id, k)})
			continue
		}
		seen[id] = j
		if id == "root" {
			roots++
		}
	}
	// roots can only ever be 0 or 1: a second component with id "root" is a duplicate id and was
	// already reported above.
	return roots, out
}

// semanticRules enforces what the spec states in prose rather than schema: surface ids are
// non-empty and created once per batch, a surface is created before the batch updates or deletes
// it, catalog ids are never empty, component ids are unique per list, and every surface created
// in the batch ends up with a root component either in createSurface.components or in a later
// updateComponents.
func semanticRules(messages []map[string]any) []schema.Problem {
	var out []schema.Problem
	created := schema.FirstCreateIndex(messages)
	seenCreate := map[string]bool{}
	var createdOrder []string
	hasRoot := map[string]bool{}
	for i, m := range messages {
		if cs, ok := m["createSurface"].(map[string]any); ok {
			sid, _ := cs["surfaceId"].(string)
			if sid == "" {
				out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].createSurface.surfaceId", i), Message: "must not be empty"})
				continue
			}
			if seenCreate[sid] {
				out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].createSurface", i), Message: fmt.Sprintf("surface %q is created twice in this batch", sid)})
				continue
			}
			seenCreate[sid] = true
			createdOrder = append(createdOrder, sid)
			if v, present := cs["catalogId"]; present && v == "" {
				out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].createSurface.catalogId", i), Message: "must not be empty"})
			}
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
			out = append(out, schema.UsedBeforeCreate(created, sid, i, "updateComponents")...)
			comps, _ := uc["components"].([]any)
			roots, p := checkList(comps, fmt.Sprintf("messages[%d].updateComponents.components", i))
			out = append(out, p...)
			if roots >= 1 {
				hasRoot[sid] = true
			}
		}
		for _, key := range []string{"updateDataModel", "deleteSurface"} {
			if obj, ok := m[key].(map[string]any); ok {
				sid, _ := obj["surfaceId"].(string)
				if sid == "" {
					out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].%s.surfaceId", i, key), Message: "must not be empty"})
				}
				// deleteSurface is exempt: deleting and recreating a surface in one batch is the
				// spec's own way to reconfigure it, so a leading deleteSurface must not be flagged
				// as using the surface before its later createSurface.
				if key == "updateDataModel" {
					out = append(out, schema.UsedBeforeCreate(created, sid, i, key)...)
				}
			}
		}
		if cf, ok := m["callRendererFunction"].(map[string]any); ok {
			call, _ := cf["callFunction"].(map[string]any)
			if v, present := call["catalogId"]; present && v == "" {
				out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].callRendererFunction.callFunction.catalogId", i), Message: "must not be empty"})
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
