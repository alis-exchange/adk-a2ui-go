package kit

import "context"

// a2uiCapabilitiesCtxKey is the private context key type for A2UI capability maps.
// Using a struct type avoids collisions with other packages' context values.
type a2uiCapabilitiesCtxKey struct{}

// WithA2UICapabilities stores the A2UI v0.9 client capability map on the context.
// Transport adapters (A2A, AG-UI, etc.) call this after extracting capabilities
// from their transport-specific format. The capabilities map typically contains:
//   - "supportedCatalogIds": []any of catalog ID strings
//   - "inlineCatalogs": []any of inline catalog schema objects
//
// Tools and toolset filters read back the stored map via [CapabilitiesFromContext].
func WithA2UICapabilities(ctx context.Context, capabilities map[string]any) context.Context {
	return context.WithValue(ctx, a2uiCapabilitiesCtxKey{}, capabilities)
}

// CapabilitiesFromContext returns the v0.9 capability params map previously stored by
// [WithA2UICapabilities], and whether that store happened. If ok is false, the map must not be used.
func CapabilitiesFromContext(ctx context.Context) (map[string]any, bool) {
	capabilities, ok := ctx.Value(a2uiCapabilitiesCtxKey{}).(map[string]any)
	return capabilities, ok
}
