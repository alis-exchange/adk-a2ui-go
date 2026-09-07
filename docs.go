// Package a2ui is the root of module [go.alis.build/adk/a2ui], a Go library for the
// [A2UI] (Agent-to-UI) protocol on the Google ADK (google.golang.org/adk/v2).
//
// [NewToolset] is the usual entry point: a transport adapter stores the client's capabilities
// with [go.alis.build/adk/a2ui/kit.WithA2UICapabilities], and the toolset negotiates the newest
// A2UI version both sides support (v1.0, v0.9.1, or v0.9), exposing one catalog tool and one
// message-generation tool for it. Messages are validated against the official schemas embedded
// in [go.alis.build/adk/a2ui/spec] and against the negotiated catalog. What the renderer sends
// back is decoded and checked the same way by [go.alis.build/adk/a2ui/v10.DecodeRendererMessage]
// and [go.alis.build/adk/a2ui/v09.DecodeClientMessage], and
// [go.alis.build/adk/a2ui/v10.FunctionDispatcher] answers renderer-initiated function calls.
//
// # Subpackages
//
//   - [go.alis.build/adk/a2ui/kit]: capabilities, negotiation, catalog resolvers, options.
//   - [go.alis.build/adk/a2ui/spec]: embedded upstream schemas and basic catalogs.
//   - [go.alis.build/adk/a2ui/v09], [go.alis.build/adk/a2ui/v09/tools]: A2UI v0.9 and v0.9.1 (validate, decode, tools).
//   - [go.alis.build/adk/a2ui/v10], [go.alis.build/adk/a2ui/v10/tools]: A2UI v1.0 (validate, decode, dispatch, tools).
//
// [A2UI]: https://a2ui.org/
package a2ui
