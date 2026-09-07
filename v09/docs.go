// Package v09 implements A2UI specification versions 0.9 and 0.9.1
// (https://a2ui.org/specification/v0.9-a2ui/, https://a2ui.org/specification/v0.9.1-a2ui/).
// The two differ only in the accepted "version" string, so one package serves both.
//
// [Validate] checks a server-to-client message batch against the official schema embedded in
// [go.alis.build/adk/a2ui/spec], the negotiated catalog, and the prose rules of the spec.
// The ADK tools live in [go.alis.build/adk/a2ui/v09/tools].
//
// [DecodeClientMessage] and [DecodeClientMessages] check what the client sends back (an
// "action" or an "error") against the embedded client_to_server.json and return typed
// [ClientMessage] values whose [Action.String] and [ClientError.String] are written for a
// model prompt.
package v09
