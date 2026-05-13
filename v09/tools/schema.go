package tools

import (
	"github.com/google/jsonschema-go/jsonschema"
)

// version is the A2UI protocol version string required in each message (const field in JSON Schema).
var version any = "v0.9"

// A2UiServerToClientListSchema is the JSON Schema for a top-level array of A2UI server-to-client
// messages (v0.9). It implements the A2UI specification at
// https://a2ui.org/specification/v0.9-a2ui/, expressed as
// [jsonschema.Schema] values for compile-time validation and runtime validation via
// [jsonschema.Schema.Resolve] and [*jsonschema.Resolved.Validate].
//
// The schema includes $defs for CreateSurface, UpdateComponents, UpdateDataModel, and
// DeleteSurface message shapes.
var A2UiServerToClientListSchema = jsonschema.Schema{
	Schema:      "https://json-schema.org/draft/2020-12/schema",
	ID:          "https://a2ui.org/specification/v0_9/server_to_client_list.json",
	Title:       "A2UI Server-to-Client Message List",
	Description: "A list of A2UI Server-to-Client messages.",
	Type:        "array",
	Items: &jsonschema.Schema{
		Title:       "A2UI Message Schema",
		Description: "Describes a JSON payload for an A2UI (Agent to UI) message, which is used to dynamically construct and update user interfaces.",
		Type:        "object",
		OneOf: []*jsonschema.Schema{
			{Ref: "#/$defs/CreateSurfaceMessage"},
			{Ref: "#/$defs/UpdateComponentsMessage"},
			{Ref: "#/$defs/UpdateDataModelMessage"},
			{Ref: "#/$defs/DeleteSurfaceMessage"},
		},
	},
	Defs: map[string]*jsonschema.Schema{
		"theme": {
			Type:        "object",
			Description: "Theme parameters for the surface. The exact structure depends on the active catalog provided in the context.",
		},
		"anyComponent": {
			Type:        "object",
			Description: "A UI component from the active catalog. The LLM should use the components provided in the prompt/context.",
			Properties: map[string]*jsonschema.Schema{
				"component": {
					Type:        "string",
					Description: "The name of the component type.",
				},
			},
			Required: []string{"component"},
			// Empty schema allows arbitrary additional properties (same intent as "additionalProperties": true in JSON).
			AdditionalProperties: &jsonschema.Schema{},
		},
		"CreateSurfaceMessage": {
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"version": {
					Const: &version,
				},
				"createSurface": {
					Type:        "object",
					Description: "Signals the client to create a new surface and begin rendering it. When this message is sent, the client will expect 'updateComponents' and/or 'updateDataModel' messages for the same surfaceId that define the component tree.",
					Properties: map[string]*jsonschema.Schema{
						"surfaceId": {
							Type:        "string",
							Description: "The unique identifier for the UI surface to be rendered.",
						},
						"catalogId": {
							Type:        "string",
							Description: "A string that uniquely identifies this catalog. It is recommended to prefix this with an internet domain that you own, to avoid conflicts e.g. mycompany.com:somecatalog'.",
						},
						"theme": {
							Ref:         "#/$defs/theme",
							Description: "Theme parameters for the surface (e.g., {'primaryColor': '#FF0000'}). These must validate against the 'theme' schema defined in the catalog.",
						},
						"sendDataModel": {
							Type:        "boolean",
							Description: "If true, the client will send the full data model of this surface in the metadata of every A2A message sent to the server that created the surface. Defaults to false.",
						},
					},
					Required:             []string{"surfaceId", "catalogId"},
					AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
				},
			},
			Required:             []string{"createSurface", "version"},
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		},
		"UpdateComponentsMessage": {
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"version": {
					Const: &version,
				},
				"updateComponents": {
					Type:        "object",
					Description: "Updates a surface with a new set of components. This message can be sent multiple times to update the component tree of an existing surface. One of the components in one of the components lists MUST have an 'id' of 'root' to serve as the root of the component tree. The createSurface message MUST have been previously sent with the 'catalogId' that is in this message.",
					Properties: map[string]*jsonschema.Schema{
						"surfaceId": {
							Type:        "string",
							Description: "The unique identifier for the UI surface to be updated.",
						},
						"components": {
							Type:        "array",
							Description: "A list containing all UI components for the surface.",
							MinItems:    jsonschema.Ptr(1),
							Items: &jsonschema.Schema{
								Ref: "#/$defs/anyComponent",
							},
						},
					},
					Required:             []string{"surfaceId", "components"},
					AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
				},
			},
			Required:             []string{"updateComponents", "version"},
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		},
		"UpdateDataModelMessage": {
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"version": {
					Const: &version,
				},
				"updateDataModel": {
					Type:        "object",
					Description: "Updates the data model for an existing surface. This message can be sent multiple times to update the data model. The createSurface message MUST have been previously sent with the 'catalogId' that is in this message.",
					Properties: map[string]*jsonschema.Schema{
						"surfaceId": {
							Type:        "string",
							Description: "The unique identifier for the UI surface this data model update applies to.",
						},
						"path": {
							Type:        "string",
							Description: "An optional path to a location within the data model (e.g., '/user/name'). If omitted, or set to '/', refers to the entire data model.",
						},
						"value": {
							Description:          "The data to be updated in the data model. If present, the value at 'path' is replaced (or created). If omitted, the key at 'path' is removed.",
							AdditionalProperties: &jsonschema.Schema{},
						},
					},
					Required:             []string{"surfaceId"},
					AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
				},
			},
			Required:             []string{"updateDataModel", "version"},
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		},
		"DeleteSurfaceMessage": {
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"version": {
					Const: &version,
				},
				"deleteSurface": {
					Type:        "object",
					Description: "Signals the client to delete the surface identified by 'surfaceId'. The createSurface message MUST have been previously sent with the 'catalogId' that is in this message.",
					Properties: map[string]*jsonschema.Schema{
						"surfaceId": {
							Type:        "string",
							Description: "The unique identifier for the UI surface to be deleted.",
						},
					},
					Required:             []string{"surfaceId"},
					AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
				},
			},
			Required:             []string{"deleteSurface", "version"},
			AdditionalProperties: &jsonschema.Schema{Not: &jsonschema.Schema{}},
		},
	},
}
