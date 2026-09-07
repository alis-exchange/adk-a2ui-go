package a2ui

import (
	"fmt"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"

	"go.alis.build/adk/a2ui/internal/toolkit"
	"go.alis.build/adk/a2ui/kit"
	v09tools "go.alis.build/adk/a2ui/v09/tools"
	v10tools "go.alis.build/adk/a2ui/v10/tools"
)

// Option configures NewToolset.
type Option func(*config)

type config struct {
	versions []string
	tools    kit.ToolOptions
}

// WithVersions restricts negotiation to these wire versions, in preference order.
// Default: kit.KnownVersions (newest first).
func WithVersions(versions ...string) Option {
	return func(c *config) { c.versions = versions }
}

// WithCatalogResolver supplies catalogs beyond the client's inline ones and the built-in basic
// catalog, for example a kit.Registry holding your renderer's catalog.
func WithCatalogResolver(r kit.CatalogResolver) Option {
	return func(c *config) { c.tools.Resolver = r }
}

// WithStrictCatalogs makes an unresolvable catalogId a validation error instead of falling
// back to envelope-only checks.
func WithStrictCatalogs() Option {
	return func(c *config) { c.tools.Strict = true }
}

// NewToolset returns the version-negotiated A2UI toolset. On each turn it reads the client's
// capabilities (kit.WithA2UICapabilities), picks the newest version both sides support, and
// exposes that version's a2ui_catalog and generate_a2ui_messages tools. Without capabilities,
// or without a mutual version, it exposes nothing.
func NewToolset(opts ...Option) (tool.Toolset, error) {
	cfg := config{versions: kit.KnownVersions}
	for _, o := range opts {
		o(&cfg)
	}
	for _, v := range cfg.versions {
		switch v {
		case kit.V09, kit.V091, kit.V10:
		default:
			return nil, fmt.Errorf("a2ui: unknown version %q", v)
		}
	}
	return &toolset{cfg: cfg}, nil
}

type toolset struct{ cfg config }

func (t *toolset) Name() string { return toolkit.ToolsetName }

func (t *toolset) Tools(ctx agent.ReadonlyContext) ([]tool.Tool, error) {
	caps, ok := kit.CapabilitiesFromContext(ctx)
	if !ok {
		return nil, nil
	}
	version, params, ok := kit.Negotiate(caps, t.cfg.versions)
	if !ok {
		return nil, nil
	}
	if version == kit.V10 {
		return v10tools.Tools(params, t.cfg.tools)
	}
	return v09tools.Tools(version, params, t.cfg.tools)
}
