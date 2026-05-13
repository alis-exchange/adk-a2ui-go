// Package kit contains version-agnostic helpers for A2UI client capability data:
// attaching it to a [context.Context] and parsing catalog fields from a capability
// params map.
//
// [WithA2UICapabilities] stores a capability map on the context so that versioned
// tool packages (e.g. [go.alis.build/adk/a2ui/v09/tools]) can read it back via
// [CapabilitiesFromContext] to decide whether to expose A2UI tools.
//
// [GetCatalogs] extracts supportedCatalogIds and inlineCatalogs from a capability
// params map in the shape described by the A2UI specification (https://a2ui.org/).
package kit
