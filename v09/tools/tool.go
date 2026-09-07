package tools

import (
	"fmt"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"

	"go.alis.build/adk/a2ui/internal/toolkit"
	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
	"go.alis.build/adk/a2ui/v09"
)

// Tool identifiers, stable across spec versions.
const (
	GenerateA2UIMessagesToolName = toolkit.GenerateToolName
	CatalogToolName              = toolkit.CatalogToolName
)

// IO types of the tools, shared with v10/tools.
type (
	GenerateA2UIToolInput  = toolkit.GenerateInput
	GenerateA2UIToolOutput = toolkit.GenerateOutput
	CatalogToolOutput      = toolkit.CatalogToolOutput
)

func checkVersion(version string) error {
	if version != kit.V09 && version != kit.V091 {
		return fmt.Errorf("v09/tools: version must be %q or %q, got %q", kit.V09, kit.V091, version)
	}
	return nil
}

func toolSpec(version string) toolkit.Spec {
	return toolkit.Spec{
		Major:       spec.MajorV09,
		Version:     version,
		Validate:    v09.Validate,
		Envelope:    EnvelopeSchema(version),
		MessageKeys: []string{"createSurface", "updateComponents", "updateDataModel", "deleteSurface"},
		Notes: []string{
			`createSurface requires "catalogId".`,
			`updateDataModel: "value" is optional; omit it to delete the key at "path".`,
			`createSurface may carry a "theme" object; it is validated against the catalog's theme schema.`,
		},
	}
}

// CatalogTool returns a2ui_catalog bound to the given negotiated params.
func CatalogTool(version string, params kit.VersionParams, opts kit.ToolOptions) (tool.Tool, error) {
	if err := checkVersion(version); err != nil {
		return nil, err
	}
	return toolkit.CatalogTool(toolSpec(version), params, opts)
}

// GenerateTool returns generate_a2ui_messages bound to the given negotiated params.
func GenerateTool(version string, params kit.VersionParams, opts kit.ToolOptions) (tool.Tool, error) {
	if err := checkVersion(version); err != nil {
		return nil, err
	}
	return toolkit.GenerateTool(toolSpec(version), params, opts)
}

// Tools returns [CatalogTool, GenerateTool].
func Tools(version string, params kit.VersionParams, opts kit.ToolOptions) ([]tool.Tool, error) {
	if err := checkVersion(version); err != nil {
		return nil, err
	}
	return toolkit.Tools(toolSpec(version), params, opts)
}

// NewToolset returns a toolset that exposes the two tools only when the client's capabilities
// (see kit.WithA2UICapabilities) include exactly this version. Prefer the root package's
// a2ui.NewToolset, which negotiates across versions.
func NewToolset(version string, opts kit.ToolOptions) (tool.Toolset, error) {
	if err := checkVersion(version); err != nil {
		return nil, err
	}
	return &toolset{version: version, opts: opts}, nil
}

type toolset struct {
	version string
	opts    kit.ToolOptions
}

func (t *toolset) Name() string { return toolkit.ToolsetName }

func (t *toolset) Tools(ctx agent.ReadonlyContext) ([]tool.Tool, error) {
	caps, ok := kit.CapabilitiesFromContext(ctx)
	if !ok {
		return nil, nil
	}
	params, ok := caps[t.version]
	if !ok {
		return nil, nil
	}
	return toolkit.Tools(toolSpec(t.version), params, t.opts)
}
