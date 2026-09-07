package tools

import "github.com/google/jsonschema-go/jsonschema"

// EnvelopeSchema is the model-facing shape of a v1.0 agent-to-renderer message list. It steers
// generation; enforcement is v10.Validate against the embedded official schema and catalogs.
// Property descriptions are copied from spec/v1_0/json/agent_to_renderer.json.
//
// A fresh tree is returned on every call, and every repeated sub-shape (the "no extra
// properties" marker and the generic component reference) is built by a small helper that
// allocates a new *jsonschema.Schema on each call: jsonschema-go's Resolve (which ADK runs on
// every tool's InputSchema) rejects a subschema reachable from more than one place in the tree,
// so no *jsonschema.Schema pointer here is ever shared between two parents. The one exception is
// the *any pointer backing every message's "version" Const, which is plain data, not a schema
// node, so sharing it is safe.
func EnvelopeSchema() *jsonschema.Schema {
	version := any("v1.0")
	noExtra := func() *jsonschema.Schema { return &jsonschema.Schema{Not: &jsonschema.Schema{}} }
	str := func(desc string) *jsonschema.Schema { return &jsonschema.Schema{Type: "string", Description: desc} }
	component := func() *jsonschema.Schema { return &jsonschema.Schema{Ref: "#/$defs/anyComponent"} }
	message := func(key string, body *jsonschema.Schema) *jsonschema.Schema {
		return &jsonschema.Schema{
			Type:                 "object",
			Properties:           map[string]*jsonschema.Schema{"version": {Const: &version}, key: body},
			Required:             []string{"version", key},
			AdditionalProperties: noExtra(),
		}
	}
	return &jsonschema.Schema{
		Schema:      "https://json-schema.org/draft/2020-12/schema",
		ID:          "https://a2ui.org/specification/v1_0/json/agent_to_renderer_list.json",
		Title:       "A2UI Agent-to-Renderer Message List",
		Description: "A list of A2UI agent-to-renderer messages.",
		Type:        "array",
		Items: &jsonschema.Schema{
			Type: "object",
			OneOf: []*jsonschema.Schema{
				{Ref: "#/$defs/CreateSurfaceMessage"}, {Ref: "#/$defs/UpdateComponentsMessage"},
				{Ref: "#/$defs/UpdateDataModelMessage"}, {Ref: "#/$defs/DeleteSurfaceMessage"},
				{Ref: "#/$defs/CallRendererFunctionMessage"}, {Ref: "#/$defs/AgentFunctionResponseMessage"},
			},
		},
		Defs: map[string]*jsonschema.Schema{
			"anyComponent": {
				Type:        "object",
				Description: "A UI component from the catalog named by its catalogId or the surface's catalogId. Use the component definitions returned by a2ui_catalog.",
				Properties: map[string]*jsonschema.Schema{
					"id":        str("Unique id of this component within the surface. Exactly one component per surface must use \"root\"."),
					"component": str("The component type name from the catalog. Must not be \"Surface\"."),
					"catalogId": str("Catalog id for this component, overriding the surface's default catalogId."),
				},
				Required:             []string{"id", "component"},
				AdditionalProperties: &jsonschema.Schema{},
			},
			"functionCall": {
				Type:        "object",
				Description: "A function call: \"call\" names the function, \"catalogId\" names its catalog, and the function's own arguments sit alongside as further properties.",
				Properties:  map[string]*jsonschema.Schema{"call": str("The name of the function to call."), "catalogId": str("The catalog ID for this function.")},
				Required:    []string{"call"},
				AdditionalProperties: &jsonschema.Schema{},
			},
			"CreateSurfaceMessage": message("createSurface", &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId":     str("The unique identifier for the UI surface to be rendered."),
					"catalogId":     str("Default catalog for components of this surface. If omitted, every component must set its own catalogId."),
					"sendDataModel": {Type: "boolean"},
					"components":    {Type: "array", MinItems: jsonschema.Ptr(1), Items: component(), Description: "Optional initial component tree; may contain the \"root\" component."},
					"dataModel":     {Type: "object", AdditionalProperties: &jsonschema.Schema{}, Description: "Optional initial data model."},
					"metadata":      {Type: "object", AdditionalProperties: &jsonschema.Schema{}},
				},
				Required:             []string{"surfaceId"},
				AdditionalProperties: noExtra(),
			}),
			"UpdateComponentsMessage": message("updateComponents", &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId":  str("The unique identifier for the UI surface to be updated."),
					"components": {Type: "array", MinItems: jsonschema.Ptr(1), Items: component()},
				},
				Required:             []string{"surfaceId", "components"},
				AdditionalProperties: noExtra(),
			}),
			"UpdateDataModelMessage": message("updateDataModel", &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId": str("The unique identifier for the UI surface this data model update applies to."),
					"path":      str("JSON Pointer into the data model, e.g. \"/user/name\". Omitted or \"/\" means the whole model."),
					"value":     {Description: "Required. The new value at path; null deletes the key.", AdditionalProperties: &jsonschema.Schema{}},
				},
				Required:             []string{"surfaceId", "value"},
				AdditionalProperties: noExtra(),
			}),
			"DeleteSurfaceMessage": message("deleteSurface", &jsonschema.Schema{
				Type:                 "object",
				Properties:           map[string]*jsonschema.Schema{"surfaceId": str("The unique identifier for the UI surface to be deleted.")},
				Required:             []string{"surfaceId"},
				AdditionalProperties: noExtra(),
			}),
			"CallRendererFunctionMessage": message("callRendererFunction", &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"functionCallId": str("Unique id for this call; the renderer echoes it in rendererFunctionResponse."),
					"callFunction":   {Ref: "#/$defs/functionCall"},
				},
				Required:             []string{"functionCallId", "callFunction"},
				AdditionalProperties: noExtra(),
			}),
			"AgentFunctionResponseMessage": message("agentFunctionResponse", &jsonschema.Schema{
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"functionCallId": str("The id from the renderer's callAgentFunction."),
					"value":          {Description: "The function result. Provide exactly one of value or error."},
					"error": {Type: "object", Properties: map[string]*jsonschema.Schema{"code": str("Error code."), "message": str("Human-readable error.")}, Required: []string{"code", "message"}},
				},
				Required:             []string{"functionCallId"},
				AdditionalProperties: noExtra(),
			}),
		},
	}
}
