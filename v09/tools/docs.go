// Package tools provides Google ADK (Agent Development Kit) tools for generating and validating
// A2UI v0.9 server-to-client messages.
//
// This package is version-specific: it implements the v0.9 A2UI specification
// (https://a2ui.org/specification/v0.9-a2ui/), semantic validation rules,
// and ADK tools that produce v0.9 message arrays. Future A2UI spec versions will have their own
// packages (e.g. v10/tools).
//
// The primary entry points are [GenerateA2UIMessages], which returns a [google.golang.org/adk/tool.Tool]
// that validates [GenerateA2UIToolInput] and, on success, returns [GenerateA2UIToolOutput] echoing the
// messages, and [NewA2UIToolset], which wraps that tool in a filtered toolset
// exposed only when A2UI capabilities are present on the agent context (see
// [go.alis.build/adk/a2ui/kit.CapabilitiesFromContext], typically after
// [go.alis.build/adk/a2ui/kit.WithA2UICapabilities]).
//
// Schema validation uses [github.com/google/jsonschema-go/jsonschema]. Additional semantic checks
// (for example, requiring a component with id "root" for each created surface) are implemented in
// this package alongside the schema.
package tools
