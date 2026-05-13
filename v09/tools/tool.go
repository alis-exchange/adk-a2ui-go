package tools

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"go.alis.build/adk/a2ui/kit"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

const (
	// GenerateA2UIMessagesToolName is the ADK tool identifier for [GenerateA2UIMessages]. Agents and prompts should
	// refer to this constant when configuring allowed tools.
	GenerateA2UIMessagesToolName = "generate_a2ui_messages"

	// CatalogToolName is the ADK tool identifier for [A2UICatalogTool], registered alongside
	// [GenerateA2UIMessages] in [NewA2UIToolset].
	CatalogToolName = "a2ui_catalog"
)

// toolDescription is the human-readable instruction block shown to the model. It summarizes valid
// message shapes, emphasizes the required "root" component id, and includes minimal JSON examples.
var toolDescription = `
Validates an array of A2UI messages against the A2UI schema. Use this tool when showing components, forms, or custom UI surfaces.

**CRITICAL RENDERING RULES:**
- For every surfaceId created, at least one component in updateComponents.components MUST have "id": "root".
- The chat renderer mounts <ComponentNode id="root" />; without root the UI will not render.

**Example (minimal valid):**
{
  "messages": [
    {
      "version": "v0.9",
      "createSurface": {
        "surfaceId": "my-surface",
        "catalogId": "https://raw.githubusercontent.com/alis-exchange/a2ui-vuetify-renderer/main/catalog/vuetify-catalog.json"
      }
    },
    {
      "version": "v0.9",
      "updateComponents": {
        "surfaceId": "my-surface",
        "components": [
          { "component": "Card", "id": "root", "child": "text-1" },
          { "component": "Text", "id": "text-1", "text": "Hello" }
        ]
      }
    }
  ]
}

**Example (updateDataModel):**
{
  "messages": [
    {
      "version": "v0.9",
      "updateDataModel": {
        "surfaceId": "my-surface",
        "path": "/user/name",
        "value": "John Doe"
      }
    }
  ]
}

**Example (deleteSurface):**
{
  "messages": [
    {
      "version": "v0.9",
      "deleteSurface": {
        "surfaceId": "my-surface"
      }
    }
  ]
}

You MUST use the exact keys "createSurface", "updateComponents", "updateDataModel", or "deleteSurface".
`

// GenerateA2UIToolInput is the JSON argument shape for [GenerateA2UIMessages].
// Messages is a heterogeneous list of A2UI server-to-client message objects (each map typically
// contains exactly one of createSurface, updateComponents, updateDataModel, or deleteSurface).
type GenerateA2UIToolInput struct {
	Messages []map[string]any `json:"messages"`
}

// JSONSchema returns the tool argument JSON Schema for [GenerateA2UIToolInput] so functiontool can
// expose it to the model: an object with required property "messages" whose value matches the
// inlined A2UI v0.9 server-to-client list schema (see schema.go).
func (i *GenerateA2UIToolInput) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"messages": &A2UiServerToClientListSchema,
		},
		Required: []string{"messages"},
	}
}

// GenerateA2UIToolOutput represents the execution result of the [GenerateA2UIMessages] tool.
// Note: Validation failures are typically returned as Go tool errors to trigger immediate
// LLM retries, so this struct primarily serves as the "Wrapped Echo" success signal.
type GenerateA2UIToolOutput struct {
	// Status indicates the completion state. Returned as "success" when the UI is rendered.
	Status string `json:"status"`
	// IsValid is true when the echoed messages passed all A2UI JSON Schema and semantic checks.
	IsValid bool `json:"is_valid"`
	// Messages is the echoed server-to-client message list, forwarded to the client UI.
	Messages []map[string]any `json:"messages"`
}

// JSONSchema returns the tool result JSON Schema for [GenerateA2UIToolOutput].
// The descriptions are heavily optimized to prevent LLM tool-calling loops.
func (o *GenerateA2UIToolOutput) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"status": {
				Type:        "string",
				Description: "The execution status. A 'success' value means the UI was successfully generated and is valid according to the A2UI schema. Your task is 100% complete. Do not call this tool again. End your turn.",
				Enum:        []any{"success", "error"},
			},
			"is_valid": {
				Type:        "boolean",
				Description: "Confirmed true if the A2UI payload passed all strict schema validations and was accepted by the client.",
			},
			// Note: Assuming a2UiServerToClientListSchema is defined elsewhere in your package.
			"messages": &A2UiServerToClientListSchema,
		},
		Required: []string{"status", "is_valid", "messages"},
	}
}

// GenerateA2UIMessages builds an ADK [tool.Tool] that accepts [GenerateA2UIToolInput], validates
// args.Messages against the resolved JSON Schema, then applies extra semantic checks (root
// component per surface). Validation errors are returned as tool errors so the model can self-correct.
func GenerateA2UIMessages() (tool.Tool, error) {
	handler := func(ctx tool.Context, args *GenerateA2UIToolInput) (*GenerateA2UIToolOutput, error) {
		rs, err := A2UiServerToClientListSchema.Resolve(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve A2UI schema: %v", err)
		}

		if err := rs.Validate(args.Messages); err != nil {
			return nil, fmt.Errorf("validation failed. Please correct the following A2UI schema error and try again:\n- %s", err.Error())
		}

		if err := validateA2UISemantics(args.Messages); err != nil {
			return nil, fmt.Errorf("validation failed. %v", err)
		}

		return &GenerateA2UIToolOutput{
			Status:   "success",
			IsValid:  true,
			Messages: args.Messages,
		}, nil
	}

	return functiontool.New(functiontool.Config{
		Name:         GenerateA2UIMessagesToolName,
		Description:  toolDescription,
		InputSchema:  (&GenerateA2UIToolInput{}).JSONSchema(),
		OutputSchema: (&GenerateA2UIToolOutput{}).JSONSchema(),
	}, handler)
}

// A2UICatalogToolInput is the JSON input for [A2UICatalogTool]. The model sends an empty object;
// catalog data comes only from A2UI capabilities already on the tool context (see
// [kit.CapabilitiesFromContext]).
type A2UICatalogToolInput struct{}

// JSONSchema returns the input schema exposed to the model: an object with no required properties.
func (i *A2UICatalogToolInput) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:       "object",
		Properties: map[string]*jsonschema.Schema{},
	}
}

// A2UICatalogToolOutput is the JSON output of [A2UICatalogTool]. CatalogURLs lists values from the
// client's supportedCatalogIds capability (often absolute catalog URLs). InlineCatalogs holds
// full catalog documents from inlineCatalogs, in the same shape as [kit.GetCatalogs].
type A2UICatalogToolOutput struct {
	CatalogURLs    []string `json:"catalog_urls"`
	InlineCatalogs []string `json:"inline_catalogs"`
}

// JSONSchema returns the output schema exposed to the model for catalog_urls and inline_catalogs.
func (o *A2UICatalogToolOutput) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type: "object",
		Properties: map[string]*jsonschema.Schema{
			"catalog_urls": {
				Type:        "array",
				Description: "A list of valid URLs pointing to supported external A2UI catalogs.",
				Items: &jsonschema.Schema{
					Type:        "string",
					Description: "The absolute URL pointing to a JSON catalog definition.",
					Format:      "uri",
				},
			},
			"inline_catalogs": {
				Type:        "array",
				Description: "A collection of fully defined A2UI component catalogs, provided as stringified JSON. You MUST parse these strings to read the JSON schema definitions, components, and $defs.",
				Items: &jsonschema.Schema{
					Type:        "string",
					Description: "An individual A2UI catalog detailing available UI components, client-side functions, and shared definitions.",
					Properties: map[string]*jsonschema.Schema{
						"$schema": {
							Type:        "string",
							Description: "The JSON schema version URL.",
							Format:      "uri",
						},
						"$id": {
							Type:        "string",
							Description: "The unique identifier URL for the catalog definition.",
						},
						"title": {
							Type:        "string",
							Description: "The human-readable title of the component catalog.",
						},
						"description": {
							Type:        "string",
							Description: "A brief explanation of the catalog's purpose or underlying UI framework.",
						},
						"catalogId": {
							Type:        "string",
							Description: "The unique ID referencing this specific catalog.",
						},
						"components": {
							Type:        "object",
							Description: "A dynamic mapping of component names (e.g., 'CapabilitiesCard', 'UserRegistrationForm') to their respective JSON schemas.",
							AdditionalProperties: &jsonschema.Schema{
								Type:        "object",
								Description: "The schema definition for a specific UI component, including its allowed properties and required fields.",
							},
						},
						"functions": {
							Type:        "object",
							Description: "A dynamic mapping of supported client-side function names (e.g., 'add', 'formatString') to their expected arguments and return types.",
							AdditionalProperties: &jsonschema.Schema{
								Type:        "object",
								Description: "The schema definition for a specific function, detailing its 'call' constant, 'args', and 'returnType'.",
							},
						},
						"$defs": {
							Type:        "object",
							Description: "Reusable JSON schema definitions and common types shared across multiple components or functions.",
							AdditionalProperties: &jsonschema.Schema{
								Type:        "object",
								Description: "A shared JSON schema definition.",
							},
						},
					},
					Required: []string{"title", "catalogId", "components"},
				},
			},
		},
		Required: []string{"catalog_urls", "inline_catalogs"},
	}
}

// A2UICatalogTool builds an ADK [tool.Tool] named [CatalogToolName] that reads A2UI extension
// capabilities from the context and returns supported catalog IDs and any inline catalogs. When
// no capabilities are present (for example before negotiation), the handler returns empty slices
// without error so the model can still call the tool safely.
func A2UICatalogTool() (tool.Tool, error) {
	handler := func(ctx tool.Context, input *A2UICatalogToolInput) (*A2UICatalogToolOutput, error) {
		a2uiCapabilities, ok := kit.CapabilitiesFromContext(ctx)
		if !ok {
			// No negotiated A2UI params on this turn; empty output is valid.
			return &A2UICatalogToolOutput{}, nil
		}

		catalogURLs, catalogsMap, err := kit.GetCatalogs(a2uiCapabilities)
		if err != nil {
			return nil, err
		}

		var stringifiedCatalogs []string
		for _, catalog := range catalogsMap {
			jsonBytes, err := json.Marshal(catalog)
			if err != nil {
				return nil, err
			}
			stringifiedCatalogs = append(stringifiedCatalogs, string(jsonBytes))
		}

		return &A2UICatalogToolOutput{
			CatalogURLs:    catalogURLs,
			InlineCatalogs: stringifiedCatalogs,
		}, nil
	}
	return functiontool.New(functiontool.Config{
		Name:         CatalogToolName,
		Description:  "Fetches the full A2UI component library capabilities. Use this tool when requested to build or modify a user interface. It returns both external catalog URLs and inline JSON schemas detailing all available UI components, their required fields, data types, and supported client-side functions (like math or string formatting).",
		InputSchema:  (&A2UICatalogToolInput{}).JSONSchema(),
		OutputSchema: (&A2UICatalogToolOutput{}).JSONSchema(),
	}, handler)
}

// NewA2UIToolset returns a named [tool.Toolset] containing [A2UICatalogTool] and
// [GenerateA2UIMessages] (catalog first, then message generation). The set is filtered so both
// tools are only visible when [kit.CapabilitiesFromContext] finds A2UI capabilities on the agent
// context (for example after extension negotiation).
func NewA2UIToolset() (tool.Toolset, error) {
	a2uiCatalogTool, err := A2UICatalogTool()
	if err != nil {
		return nil, err
	}

	a2uiTool, err := GenerateA2UIMessages()
	if err != nil {
		return nil, err
	}

	var toolSet tool.Toolset = &a2uiToolset{
		name:  "a2ui",
		tools: []tool.Tool{a2uiCatalogTool, a2uiTool},
	}
	toolSet = tool.FilterToolset(toolSet, func(ctx agent.ReadonlyContext, tool tool.Tool) bool {
		if _, ok := kit.CapabilitiesFromContext(ctx); !ok {
			return false
		}

		return true
	})

	return toolSet, nil
}

// a2uiToolset is a small adapter that implements [tool.Toolset] for a fixed slice of tools.
type a2uiToolset struct {
	name  string
	tools []tool.Tool
}

// Name returns the toolset name ("a2ui").
func (t *a2uiToolset) Name() string {
	return t.name
}

// Tools returns the tools registered in this toolset.
func (t *a2uiToolset) Tools(_ agent.ReadonlyContext) ([]tool.Tool, error) {
	return t.tools, nil
}

