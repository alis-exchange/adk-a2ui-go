package kit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// CatalogResolver supplies catalog documents by id. ok=false means the id is unknown to this
// resolver; err means the resolver itself failed and validation should abort.
type CatalogResolver interface {
	ResolveCatalog(ctx context.Context, catalogID string) (catalog map[string]any, ok bool, err error)
}

// Registry is a thread-safe in-memory CatalogResolver keyed by each catalog's own "catalogId".
// Register the catalogs your renderers use once at startup.
type Registry struct {
	mu       sync.RWMutex
	catalogs map[string]map[string]any
}

func NewRegistry() *Registry {
	return &Registry{catalogs: map[string]map[string]any{}}
}

// Register stores a deep copy of catalog under its "catalogId", so later mutations to the
// caller's map have no effect on the registry; a later Register with the same id replaces it.
func (r *Registry) Register(catalog map[string]any) error {
	id, _ := catalog["catalogId"].(string)
	if id == "" {
		return errors.New("kit: catalog has no string catalogId")
	}
	data, err := json.Marshal(catalog)
	if err != nil {
		return fmt.Errorf("kit: copy catalog %q: %w", id, err)
	}
	var cp map[string]any
	if err := json.Unmarshal(data, &cp); err != nil {
		return fmt.Errorf("kit: copy catalog %q: %w", id, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.catalogs[id] = cp
	return nil
}

// RegisterJSON parses data as a catalog document and registers it.
func (r *Registry) RegisterJSON(data []byte) error {
	var catalog map[string]any
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("kit: catalog JSON: %w", err)
	}
	return r.Register(catalog)
}

// ResolveCatalog looks up the catalog registered under id. The returned document is shared with
// every other caller and must not be mutated.
func (r *Registry) ResolveCatalog(_ context.Context, id string) (map[string]any, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.catalogs[id]
	return c, ok, nil
}

type chain []CatalogResolver

// Chain consults resolvers in order and returns the first hit. A resolver error stops the chain.
func Chain(resolvers ...CatalogResolver) CatalogResolver { return chain(resolvers) }

func (c chain) ResolveCatalog(ctx context.Context, id string) (map[string]any, bool, error) {
	for _, r := range c {
		if r == nil {
			continue
		}
		cat, ok, err := r.ResolveCatalog(ctx, id)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return cat, true, nil
		}
	}
	return nil, false, nil
}
