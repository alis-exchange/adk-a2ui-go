// Package v10 implements A2UI specification version 1.0 (https://a2ui.org/specification/v1.0-a2ui/).
//
// [Validate] checks an agent-to-renderer message batch against the official schema embedded in
// [go.alis.build/adk/a2ui/spec], the catalog each component or function call names, and the
// prose rules of the spec. The ADK tools live in [go.alis.build/adk/a2ui/v10/tools].
//
// v1.0 is marked a release candidate upstream; re-run scripts/sync-spec.sh when it tags.
//
// [DecodeRendererMessage] and [DecodeRendererMessages] check what the renderer sends back
// ("action", "callAgentFunction", "rendererFunctionResponse", "error") against the embedded
// renderer_to_agent.json and, for a callAgentFunction naming a catalogId, against that catalog,
// and return typed [RendererMessage] values. [FunctionDispatcher] runs the agent's functions
// for callAgentFunction and returns the agentFunctionResponse to send; [NewCallRendererFunction]
// builds the request for a function the renderer should run. Function arguments always sit
// under "args". A function may be called across the wire only when its catalog definition allows
// that caller ("allowedCallers": agentOnly or rendererOrAgent for the agent; anything but
// agentOnly for the renderer); both Validate and the decoders enforce it.
package v10
