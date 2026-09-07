// Package toolkit builds the two ADK tools every A2UI version exposes, parameterised by a
// version Spec so v09/tools and v10/tools stay thin.
package toolkit

import (
	"context"

	"github.com/google/jsonschema-go/jsonschema"
	"go.alis.build/adk/a2ui/kit"
)

const (
	CatalogToolName  = "a2ui_catalog"
	GenerateToolName = "generate_a2ui_messages"
	ToolsetName      = "a2ui"
)

// GenerateInput is the argument shape of generate_a2ui_messages.
type GenerateInput struct {
	Messages []map[string]any `json:"messages"`
}

// GenerateOutput echoes validated messages; descriptions are tuned to stop tool-call loops.
type GenerateOutput struct {
	Status   string           `json:"status"`
	IsValid  bool             `json:"is_valid"`
	Messages []map[string]any `json:"messages"`
}

// CatalogInput is empty; catalog data comes from the negotiated capabilities.
type CatalogInput struct{}

// CatalogToolOutput lists the catalogs the client supports and their documents where known.
type CatalogToolOutput struct {
	Version      string            `json:"version"`
	CatalogIDs   []string          `json:"catalog_ids"`
	Catalogs     map[string]string `json:"catalogs"`
	Unresolved   []string          `json:"unresolved"`
	Instructions string            `json:"instructions"`
}

// ValidateFunc is v09.Validate or v10.Validate.
type ValidateFunc func(ctx context.Context, messages []map[string]any, opts kit.ValidateOptions) error

// Spec carries everything version-specific the tools need.
type Spec struct {
	Major       string // spec.MajorV09 or spec.MajorV10
	Version     string // wire version the model must emit
	Validate    ValidateFunc
	Envelope    *jsonschema.Schema // model-facing schema for the "messages" array
	MessageKeys []string           // allowed message keys, for the description
	Notes       []string           // version gotchas, one line each
}

func generateOutputSchema(envelope *jsonschema.Schema) *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"status": {
				Type:        "string",
				Description: "The execution status. A 'success' value means the UI was successfully generated and is valid. Your task is 100% complete. Do not call this tool again. End your turn.",
				Enum:        []any{"success", "error"},
			},
			"is_valid": {
				Type:        "boolean",
				Description: "True when the payload passed every schema, catalog, and semantic check and was accepted by the client.",
			},
			"messages": envelope,
		},
		Required: []string{"status", "is_valid", "messages"},
	}
}

func catalogOutputSchema() *jsonschema.Schema {
	// Each property gets its own *jsonschema.Schema instance (never a shared pointer): the
	// schema is resolved into a tree, and jsonschema.Resolve rejects a node reachable from more
	// than one place ("do not form a tree; ... appears more than once").
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"version":     {Type: "string", Description: "The A2UI version negotiated with the client. Every message you generate must carry exactly this version string."},
			"catalog_ids": {Type: "array", Items: &jsonschema.Schema{Type: "string"}, Description: "Catalog ids the client supports. Use one of these as catalogId."},
			"catalogs": {Type: "object", AdditionalProperties: &jsonschema.Schema{Type: "string"},
				Description: "Catalog documents by id, as JSON strings: component names, their required properties, allowed values, and available functions. Follow them exactly."},
			"unresolved":   {Type: "array", Items: &jsonschema.Schema{Type: "string"}, Description: "Catalog ids whose documents are not available; prefer an id from catalogs."},
			"instructions": {Type: "string", Description: "Authoring rules from the catalogs. Follow them."},
		},
		Required: []string{"version", "catalog_ids", "catalogs", "unresolved", "instructions"},
	}
}
