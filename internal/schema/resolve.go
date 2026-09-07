package schema

import (
	"context"
	"fmt"
	"strings"

	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
)

// ResolveCatalog finds the document for catalogID in the order the design fixes: inline
// catalogs from the negotiated params, the consumer's resolver, then the embedded basic catalog.
// ok=false means nobody knew the id; err means a resolver failed.
func ResolveCatalog(ctx context.Context, major, catalogID string, opts kit.ValidateOptions) (map[string]any, bool, error) {
	if catalogID == "" {
		return nil, false, nil
	}
	for _, c := range opts.Params.InlineCatalogs {
		if id, _ := c["catalogId"].(string); id == catalogID {
			return c, true, nil
		}
	}
	if opts.Resolver != nil {
		c, ok, err := opts.Resolver.ResolveCatalog(ctx, catalogID)
		if err != nil {
			return nil, false, fmt.Errorf("catalog resolver for %q: %w", catalogID, err)
		}
		if ok {
			return c, true, nil
		}
	}
	for _, known := range spec.BasicCatalogIDs(major) {
		if known == catalogID {
			c, _, _, err := spec.BasicCatalog(major)
			if err != nil {
				return nil, false, err
			}
			return c, true, nil
		}
	}
	return nil, false, nil
}

// CatalogInstructions returns prompt guidance for a catalog: the embedded basic catalog's
// rules/instructions, or a custom catalog's "instructions" field.
func CatalogInstructions(major string, catalog map[string]any) string {
	id, _ := catalog["catalogId"].(string)
	for _, known := range spec.BasicCatalogIDs(major) {
		if known == id {
			_, _, ins, _ := spec.BasicCatalog(major)
			return ins
		}
	}
	s, _ := catalog["instructions"].(string)
	return strings.TrimSpace(s)
}
