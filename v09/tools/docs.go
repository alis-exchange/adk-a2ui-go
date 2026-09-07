// Package tools provides the ADK tools for A2UI v0.9 and v0.9.1: a2ui_catalog, which returns
// the client's catalogs and authoring rules, and generate_a2ui_messages, which validates a
// server-to-client message batch with [go.alis.build/adk/a2ui/v09.Validate] and echoes it on
// success. [NewToolset] exposes both only when the client negotiated this version; the root
// package's a2ui.NewToolset negotiates across all versions and is the usual entry point.
package tools
