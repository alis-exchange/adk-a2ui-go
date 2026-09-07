// Package kit holds the version-agnostic pieces every A2UI spec version shares: storing the
// client's capabilities document on a context, negotiating a wire version, resolving catalog
// documents by id, and the option structs the version packages accept.
//
// A transport adapter extracts the capabilities object from its own metadata (A2A
// message.metadata["a2uiClientCapabilities"] for v0.9.x, ["a2uiRendererCapabilities"] for
// v1.0) and calls [WithA2UICapabilities]. The root package's toolset then calls [Negotiate].
//
// It also holds the module's error types. Every validation failure a version package reports is
// a [*ValidationError] carrying a [Problem] per finding, so a consumer can tell a model mistake
// worth retrying from an agent-side configuration failure:
//
//	var ve *kit.ValidationError
//	if errors.As(err, &ve) { ... }
//
// The same [ValidateOptions] drive the inbound decoders
// ([go.alis.build/adk/a2ui/v10.DecodeRendererMessage]), so a renderer's call to an agent
// function is checked against the same inline, registered, and built-in catalogs.
package kit
