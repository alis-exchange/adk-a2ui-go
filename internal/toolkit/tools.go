package toolkit

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"

	"go.alis.build/adk/a2ui/internal/schema"
	"go.alis.build/adk/a2ui/kit"
)

// configErr labels a failure the model cannot fix. Tool errors are handed to the model as text,
// so a broken resolver or an unusable catalog would otherwise read like a malformed batch and
// send the model into a retry loop over a payload that was never the problem.
func configErr(err error) error {
	return fmt.Errorf("a2ui: agent-side configuration error, do not retry: %w", err)
}

// Tools returns [catalog tool, generate tool] for one negotiated version.
func Tools(s Spec, params kit.VersionParams, opts kit.ToolOptions) ([]tool.Tool, error) {
	c, err := CatalogTool(s, params, opts)
	if err != nil {
		return nil, err
	}
	g, err := GenerateTool(s, params, opts)
	if err != nil {
		return nil, err
	}
	return []tool.Tool{c, g}, nil
}

// CatalogTool returns a2ui_catalog for the negotiated params.
func CatalogTool(s Spec, params kit.VersionParams, opts kit.ToolOptions) (tool.Tool, error) {
	handler := func(ctx agent.Context, _ *CatalogInput) (*CatalogToolOutput, error) {
		out := &CatalogToolOutput{Version: s.Version, CatalogIDs: []string{}, Catalogs: map[string]string{}, Unresolved: []string{}}
		seen := map[string]bool{}
		for _, id := range params.SupportedCatalogIDs {
			if !seen[id] && id != "" {
				seen[id] = true
				out.CatalogIDs = append(out.CatalogIDs, id)
			}
		}
		for _, c := range params.InlineCatalogs {
			if id, _ := c["catalogId"].(string); id != "" && !seen[id] {
				seen[id] = true
				out.CatalogIDs = append(out.CatalogIDs, id)
			}
		}
		vopts := kit.ValidateOptions{Params: params, Resolver: opts.Resolver}
		var instructions []string
		for _, id := range out.CatalogIDs {
			cat, ok, err := schema.ResolveCatalog(ctx, s.Major, id, vopts)
			if err != nil {
				// A resolver that fails is the agent's problem, not the model's: an id it cannot
				// resolve is reported as "unresolved" instead.
				return nil, configErr(err)
			}
			if !ok {
				out.Unresolved = append(out.Unresolved, id)
				continue
			}
			b, err := json.Marshal(cat)
			if err != nil {
				return nil, configErr(err)
			}
			out.Catalogs[id] = string(b)
			if ins := schema.CatalogInstructions(s.Major, cat); ins != "" {
				instructions = append(instructions, ins)
			}
		}
		out.Instructions = strings.Join(instructions, "\n\n")
		return out, nil
	}
	return functiontool.New(functiontool.Config{
		Name:         CatalogToolName,
		Description:  "Returns the A2UI catalogs the client supports: every component, its required and optional properties, allowed values, and available functions, plus authoring instructions and the exact A2UI version string to use. Call this before generate_a2ui_messages whenever you need to build or change UI.",
		InputSchema:  &jsonschema.Schema{Type: "object", Properties: map[string]*jsonschema.Schema{}},
		OutputSchema: catalogOutputSchema(),
	}, handler)
}

// GenerateTool returns generate_a2ui_messages for the negotiated params. Validation errors are
// returned as tool errors so the model sees the problem list and retries.
func GenerateTool(s Spec, params kit.VersionParams, opts kit.ToolOptions) (tool.Tool, error) {
	vopts := kit.ValidateOptions{Version: s.Version, Params: params, Resolver: opts.Resolver, Strict: opts.Strict}
	handler := func(ctx agent.Context, in *GenerateInput) (*GenerateOutput, error) {
		if len(in.Messages) == 0 {
			// An empty batch is a model mistake with a one-line fix, not a schema failure.
			return nil, &kit.ValidationError{Problems: []kit.Problem{{Path: "messages", Message: "must contain at least one message"}}}
		}
		if err := s.Validate(ctx, in.Messages, vopts); err != nil {
			var ve *kit.ValidationError
			if errors.As(err, &ve) {
				return nil, err
			}
			return nil, configErr(err)
		}
		return &GenerateOutput{Status: "success", IsValid: true, Messages: in.Messages}, nil
	}
	example := ""
	if len(params.SupportedCatalogIDs) > 0 {
		example = params.SupportedCatalogIDs[0]
	} else if len(params.InlineCatalogs) > 0 {
		example, _ = params.InlineCatalogs[0]["catalogId"].(string)
	}
	return functiontool.New(functiontool.Config{
		Name:        GenerateToolName,
		Description: Description(s, example),
		InputSchema: &jsonschema.Schema{
			Type:       "object",
			Properties: map[string]*jsonschema.Schema{"messages": s.Envelope},
			Required:   []string{"messages"},
		},
		OutputSchema: generateOutputSchema(s.Envelope),
	}, handler)
}
