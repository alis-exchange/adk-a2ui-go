package tools

import (
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
)

// EnvelopeSchema returns the model-facing JSON Schema for a top-level array of A2UI
// server-to-client messages, for the given wire version (kit.V09 or kit.V091). It describes the
// shape of the A2UI specification at https://a2ui.org/specification/v0.9-a2ui/ (see
// spec/v0_9/json/server_to_client.json), expressed as [jsonschema.Schema] values.
//
// This schema steers generation; it does not enforce the spec. Enforcement is
// [go.alis.build/adk/a2ui/v09.Validate]'s job, against the embedded official schema and the
// negotiated catalog. The split matters because ADK's functiontool validates a tool call's raw
// arguments against this schema before the handler runs and hands the model that failure
// verbatim: a constant, path-less string that says nothing about which message was wrong or how
// to fix it. Every mistake this schema could catch (wrong version, missing surfaceId, unknown
// message key, extra property) is already caught by Validate with a curated message and a JSON
// path, so the envelope deliberately keeps only what helps the model write the batch:
// structure, property names, and descriptions. It declares no "required" lists, no const or
// enum on "version", and closes nothing off with additionalProperties, so a flawed batch reaches
// Validate and comes back as a fix-list instead of an opaque schema failure.
//
// The $defs hold the body of each message ("createSurface"'s object and so on), plus the shared
// "theme" and "anyComponent" shapes. A fresh tree is returned on every call: no
// *jsonschema.Schema pointer is shared within the tree, or across calls, since jsonschema-go's
// Resolve rejects a subschema reachable from more than one place.
func EnvelopeSchema(version string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Schema:      "https://json-schema.org/draft/2020-12/schema",
		ID:          "https://a2ui.org/specification/v0_9/server_to_client_list.json",
		Title:       "A2UI Server-to-Client Message List",
		Description: "A list of A2UI Server-to-Client messages.",
		Type:        "array",
		Items: &jsonschema.Schema{
			Title:       "A2UI Message Schema",
			Description: `Describes a JSON payload for an A2UI (Agent to UI) message, which is used to dynamically construct and update user interfaces. Every message carries "version" and exactly one message key.`,
			Type:        "object",
			Properties: map[string]*jsonschema.Schema{
				"version": {
					Type:        "string",
					Description: fmt.Sprintf("Required on every message. Must be %q.", version),
				},
				"createSurface":    {Ref: "#/$defs/CreateSurfaceMessage"},
				"updateComponents": {Ref: "#/$defs/UpdateComponentsMessage"},
				"updateDataModel":  {Ref: "#/$defs/UpdateDataModelMessage"},
				"deleteSurface":    {Ref: "#/$defs/DeleteSurfaceMessage"},
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
					"id": {
						Type:        "string",
						Description: `Required. Unique id of this component within the surface. Exactly one component per surface must use "root".`,
					},
					"component": {
						Type:        "string",
						Description: "Required. The name of the component type.",
					},
				},
				// Empty schema allows arbitrary additional properties (same intent as "additionalProperties": true in JSON).
				AdditionalProperties: &jsonschema.Schema{},
			},
			"CreateSurfaceMessage": {
				Type:        "object",
				Description: "Signals the client to create a new surface and begin rendering it. It is an error to send 'createSurface' for a surfaceId that already exists without first deleting it. When this message is sent, the client will expect 'updateComponents' and/or 'updateDataModel' messages for the same surfaceId that define the component tree.",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId": {
						Type:        "string",
						Description: "Required. The unique identifier for the UI surface to be rendered.",
					},
					"catalogId": {
						Type:        "string",
						Description: "Required. A string that uniquely identifies this catalog. It is recommended to prefix this with an internet domain that you own, to avoid conflicts e.g. mycompany.com:somecatalog'.",
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
			},
			"UpdateComponentsMessage": {
				Type:        "object",
				Description: "Updates a surface with a new set of components. This message can be sent multiple times to update the component tree of an existing surface. One of the components in one of the components lists MUST have an 'id' of 'root' to serve as the root of the component tree. A createSurface message MUST have been previously sent for the 'surfaceId' in this message; the surface's catalog is the one specified by that createSurface.",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId": {
						Type:        "string",
						Description: "Required. The unique identifier for the UI surface to be updated.",
					},
					"components": {
						Type:        "array",
						Description: "Required. A list containing all UI components for the surface.",
						MinItems:    jsonschema.Ptr(1),
						Items: &jsonschema.Schema{
							Ref: "#/$defs/anyComponent",
						},
					},
				},
			},
			"UpdateDataModelMessage": {
				Type:        "object",
				Description: "Updates the data model for an existing surface. This message can be sent multiple times to update the data model. A createSurface message MUST have been previously sent for the 'surfaceId' in this message; the surface's catalog is the one specified by that createSurface.",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId": {
						Type:        "string",
						Description: "Required. The unique identifier for the UI surface this data model update applies to.",
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
			},
			"DeleteSurfaceMessage": {
				Type:        "object",
				Description: "Signals the client to delete the surface identified by 'surfaceId'. A createSurface message MUST have been previously sent for the 'surfaceId' in this message; the surface's catalog is the one specified by that createSurface.",
				Properties: map[string]*jsonschema.Schema{
					"surfaceId": {
						Type:        "string",
						Description: "Required. The unique identifier for the UI surface to be deleted.",
					},
				},
			},
		},
	}
}
