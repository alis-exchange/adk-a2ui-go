package tools

import (
	"github.com/google/jsonschema-go/jsonschema"

	"go.alis.build/adk/a2ui/kit"
)

// EnvelopeSchema returns the JSON Schema for a top-level array of A2UI server-to-client messages,
// with every message's "version" property constrained by the given wire version (kit.V09 or
// kit.V091). It implements the A2UI specification at https://a2ui.org/specification/v0.9-a2ui/
// (see spec/v0_9/json/server_to_client.json), expressed as [jsonschema.Schema] values for
// compile-time construction and runtime validation via [jsonschema.Schema.Resolve] and
// [*jsonschema.Resolved.Validate].
//
// This schema is also used, unmodified, as the ADK tool's InputSchema (see
// [go.alis.build/adk/a2ui/internal/toolkit.GenerateTool]), so it is validated on every tool call
// before the handler runs. For kit.V091 the "version" property therefore accepts either wire
// string structurally, exactly like [go.alis.build/adk/a2ui/internal/schema.Engine]'s own
// v0.9.1 patch: a genuine mismatch is left for [go.alis.build/adk/a2ui/v09.Validate]'s exact
// check to report as "must be %q", instead of surfacing as an opaque schema failure raised
// before the handler, and hence that validator, ever runs.
//
// The schema includes $defs for CreateSurface, UpdateComponents, UpdateDataModel, and
// DeleteSurface message shapes. A fresh tree is returned on every call: no *jsonschema.Schema
// pointer is shared within the tree, or across calls, since jsonschema-go's Resolve rejects a
// subschema reachable from more than one place.
func EnvelopeSchema(version string) *jsonschema.Schema {
	return &jsonschema.Schema{
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
					"id": {
						Type:        "string",
						Description: `Unique id of this component within the surface. Exactly one component per surface must use "root".`,
					},
					"component": {
						Type:        "string",
						Description: "The name of the component type.",
					},
				},
				Required: []string{"id", "component"},
				// Empty schema allows arbitrary additional properties (same intent as "additionalProperties": true in JSON).
				AdditionalProperties: &jsonschema.Schema{},
			},
			"CreateSurfaceMessage": {
				Type: "object",
				Properties: map[string]*jsonschema.Schema{
					"version": versionProperty(version),
					"createSurface": {
						Type:        "object",
						Description: "Signals the client to create a new surface and begin rendering it. It is an error to send 'createSurface' for a surfaceId that already exists without first deleting it. When this message is sent, the client will expect 'updateComponents' and/or 'updateDataModel' messages for the same surfaceId that define the component tree.",
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
					"version": versionProperty(version),
					"updateComponents": {
						Type:        "object",
						Description: "Updates a surface with a new set of components. This message can be sent multiple times to update the component tree of an existing surface. One of the components in one of the components lists MUST have an 'id' of 'root' to serve as the root of the component tree. A createSurface message MUST have been previously sent for the 'surfaceId' in this message; the surface's catalog is the one specified by that createSurface.",
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
					"version": versionProperty(version),
					"updateDataModel": {
						Type:        "object",
						Description: "Updates the data model for an existing surface. This message can be sent multiple times to update the data model. A createSurface message MUST have been previously sent for the 'surfaceId' in this message; the surface's catalog is the one specified by that createSurface.",
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
					"version": versionProperty(version),
					"deleteSurface": {
						Type:        "object",
						Description: "Signals the client to delete the surface identified by 'surfaceId'. A createSurface message MUST have been previously sent for the 'surfaceId' in this message; the surface's catalog is the one specified by that createSurface.",
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
}

// versionProperty returns the JSON Schema for a message's "version" property for the negotiated
// wire version. kit.V09 pins the wire string exactly, since it has no older sibling to tolerate.
// kit.V091 accepts either "v0.9" or "v0.9.1" structurally, mirroring the same patch
// [go.alis.build/adk/a2ui/internal/schema.Engine] applies to the embedded spec files for v0.9.1:
// this keeps a genuinely wrong version out of a hard schema failure (which would otherwise be
// raised by ADK's own argument validation, before the tool handler and hence
// [go.alis.build/adk/a2ui/v09.Validate] ever run) and lets that validator's exact check report
// the actionable "must be %q" message instead.
func versionProperty(version string) *jsonschema.Schema {
	if version == kit.V09 {
		v := any(version)
		return &jsonschema.Schema{Const: &v}
	}
	return &jsonschema.Schema{Enum: []any{kit.V09, kit.V091}}
}
