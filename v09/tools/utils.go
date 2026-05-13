package tools

import "fmt"

// validateA2UISemantics checks constraints that are awkward to express purely in JSON Schema,
// such as requiring a root component for every surface that was created in the same message list.
//
// Rules enforced:
//   - createSurface messages must include a non-empty surfaceId.
//   - updateComponents messages must include surfaceId and a non-empty components array.
//   - For each surface that had createSurface, some later updateComponents for that surface must
//     include a component whose id is "root" (see hasRootComponentID).
//   - updateDataModel and deleteSurface messages must include a non-empty surfaceId.
func validateA2UISemantics(messages []map[string]any) error {
	type surfaceState struct {
		created bool
		hasRoot bool
	}
	surfaces := map[string]*surfaceState{}

	for i, m := range messages {
		if createSurfaceRaw, ok := m["createSurface"]; ok {
			createSurface, _ := createSurfaceRaw.(map[string]any)
			surfaceID, _ := createSurface["surfaceId"].(string)
			if surfaceID == "" {
				return fmt.Errorf("message[%d]: createSurface.surfaceId is required", i)
			}
			if _, exists := surfaces[surfaceID]; !exists {
				surfaces[surfaceID] = &surfaceState{}
			}
			surfaces[surfaceID].created = true
		}
		if updateComponentsRaw, ok := m["updateComponents"]; ok {
			updateComponents, _ := updateComponentsRaw.(map[string]any)
			surfaceID, _ := updateComponents["surfaceId"].(string)
			if surfaceID == "" {
				return fmt.Errorf("message[%d]: updateComponents.surfaceId is required", i)
			}
			if _, exists := surfaces[surfaceID]; !exists {
				surfaces[surfaceID] = &surfaceState{}
			}
			components, _ := updateComponents["components"].([]any)
			if len(components) == 0 {
				return fmt.Errorf("message[%d]: updateComponents.components must be non-empty", i)
			}
			for _, c := range components {
				if hasRootComponentID(c) {
					surfaces[surfaceID].hasRoot = true
					break
				}
			}
		}
		if updateDataModelRaw, ok := m["updateDataModel"]; ok {
			updateDataModel, _ := updateDataModelRaw.(map[string]any)
			surfaceID, _ := updateDataModel["surfaceId"].(string)
			if surfaceID == "" {
				return fmt.Errorf("message[%d]: updateDataModel.surfaceId is required", i)
			}
			if _, exists := surfaces[surfaceID]; !exists {
				surfaces[surfaceID] = &surfaceState{}
			}
		}
		if deleteSurfaceRaw, ok := m["deleteSurface"]; ok {
			deleteSurface, _ := deleteSurfaceRaw.(map[string]any)
			surfaceID, _ := deleteSurface["surfaceId"].(string)
			if surfaceID == "" {
				return fmt.Errorf("message[%d]: deleteSurface.surfaceId is required", i)
			}
		}
	}
	for surfaceID, state := range surfaces {
		if state.created && !state.hasRoot {
			return fmt.Errorf("surface %q: missing component with id 'root' in updateComponents", surfaceID)
		}
	}
	return nil
}

// hasRootComponentID reports whether a component value from JSON uses id "root", supporting both
// a flat object {"component":"...","id":"root",...} and a one-key wrapped shape {"TypeName":{...}}.
func hasRootComponentID(component any) bool {
	obj, ok := component.(map[string]any)
	if !ok {
		return false
	}
	if id, _ := obj["id"].(string); id == "root" {
		return true
	}
	for _, v := range obj {
		if inner, ok := v.(map[string]any); ok {
			if id, _ := inner["id"].(string); id == "root" {
				return true
			}
		}
	}
	return false
}
