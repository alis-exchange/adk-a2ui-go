// Package a2ui is the root of module [go.alis.build/adk/a2ui], a Go library for the
// [A2UI] (Agent-to-UI) protocol. It provides transport-agnostic ADK tools and capability
// helpers for agents that render rich UI surfaces.
//
// The module is organized by A2UI spec version so multiple versions can coexist:
//
// # Subpackages
//
//   - [go.alis.build/adk/a2ui/kit] — Version-agnostic capability helpers: store and retrieve
//     A2UI client capabilities on [context.Context], and parse catalog fields from capability
//     maps. Transport adapters (A2A, AG-UI, etc.) call [kit.WithA2UICapabilities] after
//     extracting capabilities from their transport format.
//
//   - [go.alis.build/adk/a2ui/v09/tools] — ADK function tools and specification (https://a2ui.org/specification/v0.9-a2ui/) for A2UI
//     v0.9 server-to-client messages. Includes the generate_a2ui_messages tool, the
//     a2ui_catalog tool, and the NewA2UIToolset filtered toolset.
//
// [A2UI]: https://a2ui.org/
package a2ui
