package schema

import "fmt"

// FirstCreateIndex maps each surface id to the index of the first createSurface that names it
// in the batch. Surfaces the batch never creates are absent: they may exist from an earlier turn.
func FirstCreateIndex(messages []map[string]any) map[string]int {
	out := map[string]int{}
	for i, m := range messages {
		if cs, ok := m["createSurface"].(map[string]any); ok {
			if sid, _ := cs["surfaceId"].(string); sid != "" {
				if _, seen := out[sid]; !seen {
					out[sid] = i
				}
			}
		}
	}
	return out
}

// UsedBeforeCreate reports a message at index i, under key, that addresses a surface this batch
// only creates later. Both specs say a surface must be created before any updateComponents or
// updateDataModel addresses it; deleteSurface is deliberately not covered, since deleting and
// recreating a surface in one batch is the spec's own way to reconfigure it.
func UsedBeforeCreate(created map[string]int, sid string, i int, key string) []Problem {
	if ci, ok := created[sid]; ok && ci > i {
		return []Problem{{Path: fmt.Sprintf("messages[%d].%s", i, key), Message: fmt.Sprintf("surface %q is used before its createSurface at messages[%d]", sid, ci)}}
	}
	return nil
}
