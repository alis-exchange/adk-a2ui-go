package v10

import (
	"context"
	"fmt"
	"sort"

	"go.alis.build/adk/a2ui/internal/schema"
	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
)

// CatalogIDBasic is the canonical id of the spec's basic catalog for v1.0.
const CatalogIDBasic = "https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"

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
		return &schema.ValidationError{Problems: finalize(problems)}
	}
	catalogProblems, err := validateAgainstCatalogs(ctx, eng, messages, opts)
	if err != nil {
		return err
	}
	problems = append(problems, catalogProblems...)
	problems = append(problems, semanticRules(messages)...)
	problems = finalize(problems)
	if len(problems) == 0 {
		return nil
	}
	return &schema.ValidationError{Problems: problems}
}

// finalize drops problems whose (Path, Message) pair already appeared, keeping the first
// occurrence, and sorts the result stably by Path. The version check and the schema formatter
// can both report the same wrong-version finding for one message (e.g.
// messages[0].version: must be "v1.0"), and every return path must apply this the same way so
// the returned error is deterministic regardless of which pass produced the problems.
func finalize(problems []schema.Problem) []schema.Problem {
	seen := make(map[schema.Problem]bool, len(problems))
	out := problems[:0:0]
	for _, p := range problems {
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
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
// Returns "" when neither is known for this batch.
func catalogFor(comp map[string]any, surface surfaceInfo, created bool) string {
	if cid, _ := comp["catalogId"].(string); cid != "" {
		return cid
	}
	if created {
		return surface.catalogID
	}
	return ""
}

func validateAgainstCatalogs(ctx context.Context, eng *schema.Engine, messages []map[string]any, opts kit.ValidateOptions) ([]schema.Problem, error) {
	var problems []schema.Problem
	surfaces := createdSurfaces(messages)
	resolved := map[string]map[string]any{}
	missing := map[string]bool{}
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
			cid := catalogFor(comp, info, created)
			if cid == "" {
				if created { // surface exists in this batch but named no catalog: the component must
					problems = append(problems, schema.Problem{Path: path, Message: fmt.Sprintf("must set catalogId because surface %q was created without one", sid)})
				}
				continue
			}
			cat, ok, err := lookup(cid, path+".catalogId")
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			s, err := eng.CompileRef(schema.RefAnyComponent, cat, false)
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
			s, err := eng.CompileRef(schema.RefAnyFunction, cat, false)
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
