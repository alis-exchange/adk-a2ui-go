package tools

import "github.com/google/jsonschema-go/jsonschema"

// EnvelopeSchema is the model-facing shape of a v1.0 agent-to-renderer message list. Property
// descriptions are copied from spec/v1_0/json/agent_to_renderer.json.
//
// It steers generation; it does not enforce the spec. Enforcement is
// [go.alis.build/adk/a2ui/v10.Validate]'s job, against the embedded official schema and the
// negotiated catalogs. The split matters because ADK's functiontool validates a tool call's raw
// arguments against this schema before the handler runs and hands the model that failure
// verbatim: a constant, path-less string that says nothing about which message was wrong or how
// to fix it. Every mistake this schema could catch (wrong version, a missing "value", an unknown
// message key, an extra property) is already caught by Validate with a curated message and a
// JSON path, so the envelope deliberately keeps only what helps the model write the batch:
// structure, property names, and descriptions. It declares no "required" lists, no const on
// "version", and closes nothing off with additionalProperties, so a flawed batch reaches
// Validate and comes back as a fix-list instead of an opaque schema failure.
//
// The $defs hold the body of each message ("createSurface"'s object and so on), plus the shared
// "anyComponent" and "functionCall" shapes. A fresh tree is returned on every call, and every
// repeated sub-shape is built by a small helper that allocates a new *jsonschema.Schema each
// time: jsonschema-go's Resolve (which ADK runs on every tool's InputSchema) rejects a subschema
// reachable from more than one place in the tree, so no *jsonschema.Schema pointer here is ever
// shared between two parents.
func EnvelopeSchema() *jsonschema.Schema {
	str := func(desc string) *jsonschema.Schema { return &jsonschema.Schema{Type: "string", Description: desc} }
	component := func() *jsonschema.Schema { return &jsonschema.Schema{Ref: "#/$defs/anyComponent"} }
	return &jsonschema.Schema{
		Schema:      "https://json-schema.org/draft/2020-12/schema",
		ID:          "https://a2ui.org/specification/v1_0/json/agent_to_renderer_list.json",
		Title:       "A2UI Agent-to-Renderer Message List",
		Description: "A list of A2UI agent-to-renderer messages.",
		Type:        "array",
		Items: &jsonschema.Schema{
			Type:        "object",
			Description: `One A2UI agent-to-renderer message: the "version" plus exactly one message key.`,
			Properties: map[string]*jsonschema.Schema{
				"version":               str(`Required on every message. Must be "v1.0".`),
				"createSurface":         {Ref: "#/$defs/CreateSurfaceMessage"},
				"updateComponents":      {Ref: "#/$defs/UpdateComponentsMessage"},
				"updateDataModel":       {Ref: "#/$defs/UpdateDataModelMessage"},
				"deleteSurface":         {Ref: "#/$defs/DeleteSurfaceMessage"},
				"callRendererFunction":  {Ref: "#/$defs/CallRendererFunctionMessage"},
				"agentFunctionResponse": {Ref: "#/$defs/AgentFunctionResponseMessage"},
			},
		},
		Defs: map[string]*jsonschema.Schema{
			"anyComponent": {
				Type:        "object",
				Description: "A UI component from the catalog named by its catalogId or the surface's catalogId. Use the component definitions returned by a2ui_catalog.",
				Properties: map[string]*jsonschema.Schema{
					"id":        str("Required. Unique id of this component within the surface. Exactly one component per surface must use \"root\"."),
					"component": str("Required. The component type name from the catalog. Must not be \"Surface\"."),
					"catalogId": str("Catalog id for this component, overriding the surface's default catalogId."),
				},
				AdditionalProperties: &jsonschema.Schema{},
			},
			"functionCall": {
				Type:        "object",
				Description: "A function call: \"call\" names the function, \"catalogId\" names its catalog, and the function's arguments go in the \"args\" object as the catalog's function definition describes.",
				Properties: map[string]*jsonschema.Schema{
					"call":      str("Required. The name of the function to call."),
					"catalogId": str("The catalog ID for this function."),
					"args":      {Type: "object", AdditionalProperties: &jsonschema.Schema{}, Description: "The function's arguments, keyed by argument name, as defined by the catalog function."},
				},
				AdditionalProperties: &jsonschema.Schema{},
			},
			"CreateSurfaceMessage": {
				Type:        "object",
				Description: "Creates a new surface and begins rendering it.",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId":     str("Required. The unique identifier for the UI surface to be rendered."),
					"catalogId":     str("Default catalog for components of this surface. If omitted, every component must set its own catalogId."),
					"sendDataModel": {Type: "boolean"},
					"components":    {Type: "array", MinItems: jsonschema.Ptr(1), Items: component(), Description: "Optional initial component tree; may contain the \"root\" component."},
					"dataModel":     {Type: "object", AdditionalProperties: &jsonschema.Schema{}, Description: "Optional initial data model."},
					"metadata":      {Type: "object", AdditionalProperties: &jsonschema.Schema{}},
				},
			},
			"UpdateComponentsMessage": {
				Type:        "object",
				Description: "Replaces or extends the component tree of an existing surface.",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId":  str("Required. The unique identifier for the UI surface to be updated."),
					"components": {Type: "array", MinItems: jsonschema.Ptr(1), Items: component(), Description: "Required. The UI components for the surface."},
				},
			},
			"UpdateDataModelMessage": {
				Type:        "object",
				Description: "Updates the data model of an existing surface.",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId": str("Required. The unique identifier for the UI surface this data model update applies to."),
					"path":      str("JSON Pointer into the data model, e.g. \"/user/name\". Omitted or \"/\" means the whole model."),
					"value":     {Description: "Required. The new value at path; null deletes the key.", AdditionalProperties: &jsonschema.Schema{}},
				},
			},
			"DeleteSurfaceMessage": {
				Type:        "object",
				Description: "Deletes the surface identified by surfaceId.",
				Properties:  map[string]*jsonschema.Schema{"surfaceId": str("Required. The unique identifier for the UI surface to be deleted.")},
			},
			"CallRendererFunctionMessage": {
				Type:        "object",
				Description: "Calls a function the renderer's catalog provides.",
				Properties: map[string]*jsonschema.Schema{
					"functionCallId": str("Required. Unique id for this call; the renderer echoes it in rendererFunctionResponse."),
					"callFunction":   {Ref: "#/$defs/functionCall", Description: "Required. The function to call."},
				},
			},
			"AgentFunctionResponseMessage": {
				Type:        "object",
				Description: "Answers a callAgentFunction the renderer sent.",
				Properties: map[string]*jsonschema.Schema{
					"functionCallId": str("Required. The id from the renderer's callAgentFunction."),
					"value":          {Description: "The function result. Provide exactly one of value or error."},
					"error": {Type: "object", Description: "The failure. Provide exactly one of value or error.",
						Properties: map[string]*jsonschema.Schema{"code": str("Required. Error code."), "message": str("Required. Human-readable error.")}},
				},
			},
		},
	}
}
