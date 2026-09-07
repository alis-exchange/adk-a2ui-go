package tools

import (
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"

	"go.alis.build/adk/a2ui/internal/toolkit"
	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
	"go.alis.build/adk/a2ui/v10"
)

// Tool identifiers, stable across spec versions.
const (
	GenerateA2UIMessagesToolName = toolkit.GenerateToolName
	CatalogToolName              = toolkit.CatalogToolName
)

// IO types of the tools, shared with v09/tools.
type (
	GenerateA2UIToolInput  = toolkit.GenerateInput
	GenerateA2UIToolOutput = toolkit.GenerateOutput
	CatalogToolOutput      = toolkit.CatalogToolOutput
)

func toolSpec() toolkit.Spec {
	return toolkit.Spec{
		Major:       spec.MajorV10,
		Version:     kit.V10,
		Validate:    v10.Validate,
		Envelope:    EnvelopeSchema(),
		MessageKeys: []string{"createSurface", "updateComponents", "updateDataModel", "deleteSurface", "callRendererFunction", "agentFunctionResponse"},
		Notes: []string{
			`updateDataModel requires "value"; send null to delete the key at "path".`,
			`createSurface has no "theme". Its "catalogId" is optional, but then every component must set its own "catalogId".`,
			`A component may never be named "Surface". The "root" component may be placed inside createSurface.components.`,
			`Function arguments go under "args" inside "callFunction", next to "call" and "catalogId", as the catalog's function definition describes.`,
		},
	}
}

// CatalogTool returns a2ui_catalog bound to the given negotiated params.
func CatalogTool(params kit.VersionParams, opts kit.ToolOptions) (tool.Tool, error) {
	return toolkit.CatalogTool(toolSpec(), params, opts)
}

// GenerateTool returns generate_a2ui_messages bound to the given negotiated params.
func GenerateTool(params kit.VersionParams, opts kit.ToolOptions) (tool.Tool, error) {
	return toolkit.GenerateTool(toolSpec(), params, opts)
}

// Tools returns [CatalogTool, GenerateTool].
func Tools(params kit.VersionParams, opts kit.ToolOptions) ([]tool.Tool, error) {
	return toolkit.Tools(toolSpec(), params, opts)
}

// NewToolset returns a toolset that exposes the two tools only when the client's capabilities
// (see kit.WithA2UICapabilities) include v1.0. Prefer the root package's a2ui.NewToolset, which
// negotiates across versions.
func NewToolset(opts kit.ToolOptions) (tool.Toolset, error) {
	return &toolset{opts: opts}, nil
}

type toolset struct {
	opts kit.ToolOptions
}

func (t *toolset) Name() string { return toolkit.ToolsetName }

func (t *toolset) Tools(ctx agent.ReadonlyContext) ([]tool.Tool, error) {
	caps, ok := kit.CapabilitiesFromContext(ctx)
	if !ok {
		return nil, nil
	}
	params, ok := caps[kit.V10]
	if !ok {
		return nil, nil
	}
	return toolkit.Tools(toolSpec(), params, t.opts)
}
