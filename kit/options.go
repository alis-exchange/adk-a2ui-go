package kit

// ValidateOptions is shared by v09.Validate and v10.Validate.
type ValidateOptions struct {
	// Version is the exact wire version every message must carry. Empty accepts whatever the
	// schema allows (for v09 that is both "v0.9" and "v0.9.1").
	Version string
	// Params are the negotiated client parameters; their InlineCatalogs are the first place a
	// catalogId is looked up.
	Params VersionParams
	// Resolver supplies consumer-registered catalogs; may be nil.
	Resolver CatalogResolver
	// Strict makes an unresolvable catalogId a validation error instead of falling back to
	// envelope-only checks for that surface.
	Strict bool
}

// ToolOptions configures the ADK tools; the negotiated params are supplied at runtime.
type ToolOptions struct {
	Resolver CatalogResolver
	Strict   bool
}
