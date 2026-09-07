package v09

import (
	"context"
	"fmt"

	"go.alis.build/adk/a2ui/internal/schema"
	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
)

// CatalogIDBasic is the canonical id of the spec's basic catalog for v0.9.x.
const CatalogIDBasic = "https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json"

// Validate checks messages and returns nil or a *schema.ValidationError listing every problem
// found. opts.Version may be kit.V09, kit.V091, or empty (either).
func Validate(ctx context.Context, messages []map[string]any, opts kit.ValidateOptions) error {
	switch opts.Version {
	case "", kit.V09, kit.V091:
	default:
		return fmt.Errorf("v09: unsupported version %q", opts.Version)
	}
	v091 := opts.Version != kit.V09
	var problems []schema.Problem
	problems = append(problems, checkVersion(messages, opts.Version)...)

	inst := schema.ToInstance(messages)
	eng := schema.For(spec.MajorV09)
	envelope, err := eng.Compile(schema.CompileOptions{Entry: schema.EntryOutboundV09, V091: v091})
	if err != nil {
		return err
	}
	if err := envelope.Validate(inst); err != nil {
		// Structure is broken; component-level checks would only add noise.
		problems = append(problems, schema.Format(err, inst, "messages")...)
		return &schema.ValidationError{Problems: schema.Finalize(problems)}
	}

	catalogProblems, err := validateAgainstCatalogs(ctx, eng, messages, opts, v091)
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

func checkVersion(messages []map[string]any, want string) []schema.Problem {
	if want == "" {
		return nil
	}
	var out []schema.Problem
	for i, m := range messages {
		if got, _ := m["version"].(string); got != want {
			out = append(out, schema.Problem{Path: fmt.Sprintf("messages[%d].version", i), Message: fmt.Sprintf("must be %q", want)})
		}
	}
	return out
}

// validateAgainstCatalogs is pass 2: every component and theme is checked against the catalog
// its surface was created with in this batch. Surfaces not created here have an unknown catalog
// and are skipped; that is not a strict-mode error because the surface may legitimately
// pre-exist from an earlier turn.
func validateAgainstCatalogs(ctx context.Context, eng *schema.Engine, messages []map[string]any, opts kit.ValidateOptions, v091 bool) ([]schema.Problem, error) {
	var problems []schema.Problem
	surfaceCatalog := map[string]string{}
	catalogAt := map[string]int{} // catalogId -> message index of the createSurface, for strict-mode paths
	for i, m := range messages {
		if cs, ok := m["createSurface"].(map[string]any); ok {
			sid, _ := cs["surfaceId"].(string)
			cid, _ := cs["catalogId"].(string)
			surfaceCatalog[sid] = cid
			if _, seen := catalogAt[cid]; !seen {
				catalogAt[cid] = i
			}
		}
	}
	resolved := map[string]map[string]any{}
	missing := map[string]bool{}
	lookup := func(cid string) (map[string]any, bool, error) {
		if c, ok := resolved[cid]; ok {
			return c, true, nil
		}
		if missing[cid] {
			return nil, false, nil
		}
		c, ok, err := schema.ResolveCatalog(ctx, spec.MajorV09, cid, opts)
		if err != nil {
			return nil, false, err
		}
		if !ok {
			missing[cid] = true
			if opts.Strict {
				problems = append(problems, schema.Problem{
					Path:    fmt.Sprintf("messages[%d].createSurface.catalogId", catalogAt[cid]),
					Message: fmt.Sprintf("catalog %q is not available to this agent (not inline, registered, or built in)", cid),
				})
			}
			return nil, false, nil
		}
		resolved[cid] = c
		return c, true, nil
	}

	for i, m := range messages {
		if cs, ok := m["createSurface"].(map[string]any); ok {
			theme, has := cs["theme"]
			if !has {
				continue
			}
			cid, _ := cs["catalogId"].(string)
			cat, ok, err := lookup(cid)
			if err != nil || !ok {
				if err != nil {
					return nil, err
				}
				continue
			}
			s, err := eng.CompileRef(schema.RefTheme, cat, v091)
			if err != nil {
				return nil, err
			}
			if err := s.Validate(theme); err != nil {
				problems = append(problems, schema.Format(err, theme, fmt.Sprintf("messages[%d].createSurface.theme", i))...)
			}
			continue
		}
		uc, ok := m["updateComponents"].(map[string]any)
		if !ok {
			continue
		}
		sid, _ := uc["surfaceId"].(string)
		cid := surfaceCatalog[sid]
		if cid == "" {
			continue
		}
		cat, ok, err := lookup(cid)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		compSchema, err := eng.CompileRef(schema.RefAnyComponent, cat, v091)
		if err != nil {
			return nil, err
		}
		comps, _ := uc["components"].([]any)
		for j, c := range comps {
			if err := compSchema.Validate(c); err != nil {
				problems = append(problems, schema.Format(err, c, fmt.Sprintf("messages[%d].updateComponents.components[%d]", i, j))...)
			}
		}
	}
	return problems, nil
}
