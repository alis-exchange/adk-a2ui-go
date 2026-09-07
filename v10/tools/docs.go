// Package tools provides the ADK tools for A2UI v1.0: a2ui_catalog, which returns the client's
// catalogs and authoring rules, and generate_a2ui_messages, which validates an agent-to-renderer
// message batch with [go.alis.build/adk/a2ui/v10.Validate] and echoes it on success. [NewToolset]
// exposes both only when the client negotiated this version; the root package's a2ui.NewToolset
// negotiates across all versions and is the usual entry point.
package tools
