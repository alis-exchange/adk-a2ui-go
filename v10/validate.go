package v10

import (
	"context"
	"fmt"

	"go.alis.build/adk/a2ui/internal/schema"
	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
)

// CatalogIDBasic is the canonical id of the spec's basic catalog for v1.0.
const CatalogIDBasic = "https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"

// refComponent and refFunctionCall are the full envelope-level definitions a component or
// function call must satisfy once its catalog is known: each combines the catalog's bare
// discriminated union (schema.RefAnyComponent / schema.RefAnyFunction) with the envelope's
// common properties (id/catalogId/accessibility/metadata, or call/catalogId) and closes the
// schema with unevaluatedProperties:false. Validating against the catalog union alone (as the
// bare Ref constants do) would miss unknown properties, since v1.0's basic catalog components
// declare no additionalProperties/unevaluatedProperties of their own -- that closure exists only
// once, centrally, on these two definitions.
const (
	refComponent    = "agent_to_renderer.json#/$defs/Component"
	refFunctionCall = "common_types.json#/$defs/FunctionCall"
)

// Validate checks messages and returns nil or a *schema.ValidationError listing every problem.
func Validate(ctx context.Context, messages []map[string]any, opts kit.ValidateOptions) error {
	if opts.Version != "" && opts.Version != kit.V10 {
		return fmt.Errorf("v10: unsupported version %q", opts.Version)
	}
	var problems []schema.Problem
	for i, m := range messages {
		if got, _ := m["version"].(string); opts.Version != "" && got != opts.Version {
			problems = append(problems, schema.Problem{Path: fmt.Sprintf("messages[%d].version", i), Message: fmt.Sprintf("must be %q", opts.Version)})
		}
	}
	inst := schema.ToInstance(messages)
	eng := schema.For(spec.MajorV10)
	envelope, err := eng.Compile(schema.CompileOptions{Entry: schema.EntryOutboundV10})
	if err != nil {
		return err
	}
	if err := envelope.Validate(inst); err != nil {
		// Structure is broken; component-level checks would only add noise.
		problems = append(problems, schema.Format(err, inst, "messages")...)
		return &schema.ValidationError{Problems: schema.Finalize(problems)}
	}
	catalogProblems, err := validateAgainstCatalogs(ctx, eng, messages, opts)
	if err != nil {
		return err
	}
	problems = append(problems, catalogProblems...)
	problems = append(problems, semanticRules(messages)...)
	problems = schema.Finalize(problems)
	if len(problems) == 0 {
		return nil
	}
	return &schema.ValidationError{Problems: problems}
}

// surfaceInfo records what this batch's createSurface said about a surface.
type surfaceInfo struct {
	index     int
	catalogID string
}

func createdSurfaces(messages []map[string]any) map[string]surfaceInfo {
	out := map[string]surfaceInfo{}
	for i, m := range messages {
		if cs, ok := m["createSurface"].(map[string]any); ok {
			sid, _ := cs["surfaceId"].(string)
			cid, _ := cs["catalogId"].(string)
			if _, dup := out[sid]; !dup {
				out[sid] = surfaceInfo{index: i, catalogID: cid}
			}
		}
	}
	return out
}

// catalogFor applies the v1.0 fallback: the component's own catalogId, else the surface's.
// cid is "" when neither is known for this batch. lookupPath is where a strict-mode "catalog
// not available" finding belongs if this turns out to be the first place this cid is
// encountered in message order: the component's own catalogId property when the component set
// one, or the creating surface's createSurface.catalogId when the id was inherited. lookup
// reports at most once per cid regardless of how many places later reuse it.
func catalogFor(comp map[string]any, componentPath string, surface surfaceInfo, created bool) (cid, lookupPath string) {
	if own, _ := comp["catalogId"].(string); own != "" {
		return own, componentPath + ".catalogId"
	}
	if created {
		return surface.catalogID, fmt.Sprintf("messages[%d].createSurface.catalogId", surface.index)
	}
	return "", ""
}

func validateAgainstCatalogs(ctx context.Context, eng *schema.Engine, messages []map[string]any, opts kit.ValidateOptions) ([]schema.Problem, error) {
	var problems []schema.Problem
	surfaces := createdSurfaces(messages)
	resolved := map[string]map[string]any{}
	missing := map[string]bool{}
	// lookup resolves cid, caching both hits and misses. A strict-mode "not available" finding is
	// reported at most once per cid per batch, attributed to the first place it is encountered in
	// message order (the path passed in on that first miss); every later call site that reuses
	// the same still-missing cid is silently skipped, regardless of what path it would have used.
	lookup := func(cid, path string) (map[string]any, bool, error) {
		if c, ok := resolved[cid]; ok {
			return c, true, nil
		}
		if missing[cid] {
			return nil, false, nil
		}
		c, ok, err := schema.ResolveCatalog(ctx, spec.MajorV10, cid, opts)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			missing[cid] = true
			if opts.Strict {
				problems = append(problems, schema.Problem{Path: path, Message: fmt.Sprintf("catalog %q is not available to this agent (not inline, registered, or built in)", cid)})
			}
			return nil, false, nil
		}
		resolved[cid] = c
		return c, true, nil
	}
	checkComponents := func(comps []any, sid, basePath string) error {
		info, created := surfaces[sid]
		for j, c := range comps {
			comp, _ := c.(map[string]any)
			path := fmt.Sprintf("%s[%d]", basePath, j)
			cid, lookupPath := catalogFor(comp, path, info, created)
			if cid == "" {
				if created { // surface exists in this batch but named no catalog: the component must
					problems = append(problems, schema.Problem{Path: path, Message: fmt.Sprintf("must set catalogId because surface %q was created without one", sid)})
				}
				continue
			}
			cat, ok, err := lookup(cid, lookupPath)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			s, err := eng.CompileRef(refComponent, cat, false)
			if err != nil {
				return err
			}
			if err := s.Validate(c); err != nil {
				problems = append(problems, schema.Format(err, c, path)...)
			}
		}
		return nil
	}

	for i, m := range messages {
		if cs, ok := m["createSurface"].(map[string]any); ok {
			sid, _ := cs["surfaceId"].(string)
			if comps, ok := cs["components"].([]any); ok {
				if err := checkComponents(comps, sid, fmt.Sprintf("messages[%d].createSurface.components", i)); err != nil {
					return nil, err
				}
			}
		}
		if uc, ok := m["updateComponents"].(map[string]any); ok {
			sid, _ := uc["surfaceId"].(string)
			comps, _ := uc["components"].([]any)
			if err := checkComponents(comps, sid, fmt.Sprintf("messages[%d].updateComponents.components", i)); err != nil {
				return nil, err
			}
		}
		if cf, ok := m["callRendererFunction"].(map[string]any); ok {
			call, _ := cf["callFunction"].(map[string]any)
			path := fmt.Sprintf("messages[%d].callRendererFunction.callFunction", i)
			cid, _ := call["catalogId"].(string)
			if cid == "" {
				continue // schema already requires it here
			}
			cat, ok, err := lookup(cid, path+".catalogId")
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			s, err := eng.CompileRef(refFunctionCall, cat, false)
			if err != nil {
				return nil, err
			}
			if err := s.Validate(call); err != nil {
				problems = append(problems, schema.Format(err, call, path)...)
			}
		}
	}
	return problems, nil
}
