package kit

// AgentCapabilities builds the object an agent advertises to clients, in the shape of
// server_capabilities.json (v0.9.x) and agent_capabilities.json (v1.0):
// {version: {"supportedCatalogIds": [...], "acceptsInlineCatalogs": bool}}.
func AgentCapabilities(version string, catalogIDs []string, acceptsInline bool) map[string]any {
	return map[string]any{
		version: map[string]any{
			"supportedCatalogIds":   catalogIDs,
			"acceptsInlineCatalogs": acceptsInline,
		},
	}
}
