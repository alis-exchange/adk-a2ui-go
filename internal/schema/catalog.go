package schema

import (
	"sort"

	"go.alis.build/adk/a2ui/spec"
)

// withUnions returns catalog with the $defs entries the envelope schemas dereference
// (anyComponent, anyFunction, and for v0.9 theme) filled in where the catalog leaves them out.
// The spec's inline-catalog shapes do not carry them: v0.9's client_capabilities.json#/$defs/Catalog
// forbids $defs outright and v1.0's catalog_definition.json makes it optional, so a client's
// custom catalog would otherwise fail to compile ("json-pointer ... not found") and every turn
// on that client would be reported as an agent-side configuration error. Each union is built
// from what the catalog declares, the way the basic catalogs build theirs; a catalog that
// declares nothing for a union gets the permissive stub for it, since there is nothing to check
// against. Entries the catalog already has are kept; the caller's maps are never mutated.
func withUnions(major string, catalog map[string]any) map[string]any {
	defs, _ := catalog["$defs"].(map[string]any)
	stub := stubCatalog()["$defs"].(map[string]any)
	merged := make(map[string]any, len(defs)+3)
	for k, v := range defs {
		merged[k] = v
	}
	if _, ok := merged["anyComponent"]; !ok {
		merged["anyComponent"] = refUnion(catalog["components"], "#/components/", stub["anyComponent"])
	}
	if _, ok := merged["anyFunction"]; !ok {
		merged["anyFunction"] = functionUnion(catalog["functions"], stub["anyFunction"])
	}
	if _, ok := merged["theme"]; !ok && major == spec.MajorV09 {
		merged["theme"] = themeSchema(catalog["theme"], stub["theme"])
	}
	out := make(map[string]any, len(catalog)+1)
	for k, v := range catalog {
		out[k] = v
	}
	out["$defs"] = merged
	return out
}

// refUnion builds oneOf[{$ref: prefix+name}, ...] over the keys of an object of schemas, in name
// order so the compiled schema is deterministic. Format prunes these unions by the $ref
// basename, exactly as it does for the basic catalogs.
func refUnion(v any, prefix string, fallback any) any {
	m, _ := v.(map[string]any)
	if len(m) == 0 {
		return fallback
	}
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	branches := make([]any, len(names))
	for i, name := range names {
		branches[i] = map[string]any{"$ref": prefix + name}
	}
	return map[string]any{"oneOf": branches}
}

// functionUnion accepts both function shapes the spec uses: an object of call schemas keyed by
// name (the basic catalogs, catalog_definition.json) and v0.9's client_capabilities.json array of
// {name, parameters, returnType}, whose parameters schema describes "args".
func functionUnion(v any, fallback any) any {
	switch fns := v.(type) {
	case map[string]any:
		return refUnion(fns, "#/functions/", fallback)
	case []any:
		var branches []any
		for _, f := range fns {
			def, _ := f.(map[string]any)
			name, _ := def["name"].(string)
			if name == "" {
				continue
			}
			props := map[string]any{"call": map[string]any{"const": name}}
			if params, ok := def["parameters"]; ok {
				props["args"] = params
			}
			branches = append(branches, map[string]any{"type": "object", "properties": props, "required": []any{"call"}})
		}
		if len(branches) == 0 {
			return fallback
		}
		return map[string]any{"oneOf": branches}
	}
	return fallback
}

// themeSchema turns v0.9's inline theme (a map from theme property to its schema) into the
// object schema server_to_client.json validates createSurface.theme against. It stays open,
// like the basic catalog's theme.
func themeSchema(v any, fallback any) any {
	props, _ := v.(map[string]any)
	if len(props) == 0 {
		return fallback
	}
	return map[string]any{"type": "object", "properties": props, "additionalProperties": true}
}
