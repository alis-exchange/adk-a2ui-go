package v10

import (
	"context"
	"encoding/json"
	"fmt"

	"go.alis.build/adk/a2ui/internal/schema"
	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
)

// rendererMessageKeys are the message keys renderer_to_agent.json allows beside "version".
var rendererMessageKeys = []string{"action", "callAgentFunction", "rendererFunctionResponse", "error"}

// DecodeRendererMessage validates one renderer-to-agent message and returns it typed. The
// envelope is checked against the embedded renderer_to_agent.json; a callAgentFunction whose
// callFunction names a catalogId is then checked against that catalog when it resolves (inline
// catalogs from opts.Params, then opts.Resolver, then the embedded basic catalog), with
// opts.Strict turning an unresolvable id into a problem, exactly as Validate treats an
// outbound callRendererFunction. A call without a catalogId is checked against the envelope
// only; whether the agent has such a function is the dispatcher's call. opts.Version is "" or
// kit.V10. Problems come back as a *kit.ValidationError with paths such as
// "callAgentFunction.callFunction.args"; any other error is an unsupported pinned version or a
// resolver failure. The error's header line is written for the outbound tool; inbound callers
// should read Problems.
func DecodeRendererMessage(ctx context.Context, m map[string]any, opts kit.ValidateOptions) (*RendererMessage, error) {
	msgs, err := decodeRendererMessages(ctx, []map[string]any{m}, opts, "")
	if err != nil {
		return nil, err
	}
	return &msgs[0], nil
}

// DecodeRendererMessages decodes a renderer-to-agent message list. Problems carry
// "messages[i]" paths, and the whole list is rejected when any message has one.
func DecodeRendererMessages(ctx context.Context, ms []map[string]any, opts kit.ValidateOptions) ([]RendererMessage, error) {
	return decodeRendererMessages(ctx, ms, opts, "messages")
}

func decodeRendererMessages(ctx context.Context, ms []map[string]any, opts kit.ValidateOptions, prefix string) ([]RendererMessage, error) {
	if opts.Version != "" && opts.Version != kit.V10 {
		return nil, fmt.Errorf("v10: unsupported version %q", opts.Version)
	}
	eng := schema.For(spec.MajorV10)
	envelope, err := eng.Compile(schema.CompileOptions{Entry: schema.EntryInboundV10})
	if err != nil {
		return nil, err
	}
	var problems []schema.Problem
	out := make([]RendererMessage, 0, len(ms))
	for i, m := range ms {
		path := prefix
		if prefix != "" {
			path = fmt.Sprintf("%s[%d]", prefix, i)
		}
		p := schema.InboundPrecheck(m, opts.Version, rendererMessageKeys, path)
		if len(p) == 0 {
			if err := envelope.Validate(m); err != nil {
				p = schema.Format(err, m, path)
			}
		}
		if len(p) == 0 {
			if p, err = checkAgentCall(ctx, eng, m, opts, path); err != nil {
				return nil, err
			}
		}
		if len(p) > 0 {
			problems = append(problems, p...)
			continue
		}
		rm, err := toRendererMessage(m)
		if err != nil {
			return nil, err
		}
		out = append(out, rm)
	}
	if problems = schema.Finalize(problems); len(problems) > 0 {
		return nil, &schema.ValidationError{Problems: problems}
	}
	return out, nil
}

// checkAgentCall guards the args shape for the typed conversion, then validates
// callAgentFunction.callFunction against the catalog it names when that catalog resolves. The
// stub catalog the envelope pass used lets any callFunction through, so this is the only place
// arguments are checked.
func checkAgentCall(ctx context.Context, eng *schema.Engine, m map[string]any, opts kit.ValidateOptions, path string) ([]schema.Problem, error) {
	caf, ok := m["callAgentFunction"].(map[string]any)
	if !ok {
		return nil, nil
	}
	call, _ := caf["callFunction"].(map[string]any)
	callPath := schema.JoinPath(path, "callAgentFunction.callFunction")
	if args, present := call["args"]; present {
		if _, isObject := args.(map[string]any); !isObject {
			return []schema.Problem{{Path: callPath + ".args", Message: "must be of type object, got " + jsonType(args)}}, nil
		}
	}
	cid, _ := call["catalogId"].(string)
	if cid == "" {
		return nil, nil
	}
	cat, ok, err := schema.ResolveCatalog(ctx, spec.MajorV10, cid, opts)
	if err != nil {
		return nil, err
	}
	if !ok {
		if opts.Strict {
			return []schema.Problem{{Path: callPath + ".catalogId", Message: fmt.Sprintf("catalog %q is not available to this agent (not inline, registered, or built in)", cid)}}, nil
		}
		return nil, nil
	}
	s, err := eng.CompileRef(refFunctionCall, cat, false)
	if err != nil {
		return nil, err
	}
	if err := s.Validate(call); err != nil {
		return schema.Format(err, call, callPath), nil
	}
	return nil, nil
}

// jsonType names a decoded JSON value's type the way the validator's messages do.
func jsonType(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case bool:
		return "boolean"
	case float64, json.Number:
		return "number"
	case string:
		return "string"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	}
	return fmt.Sprintf("%T", v)
}

// toRendererMessage converts a validated message into its typed form by way of JSON, which is
// what the map already is; the schema has fixed every field's type by now, so a failure here
// means a value that is not JSON at all (a channel, a func) and is reported as a plain error.
func toRendererMessage(m map[string]any) (RendererMessage, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return RendererMessage{}, fmt.Errorf("v10: encode message: %w", err)
	}
	var wire struct {
		Version string `json:"version"`
		Action  *struct {
			Action
			Metadata *struct {
				Extensions map[string]any `json:"extensions"`
			} `json:"metadata"`
		} `json:"action"`
		CallAgentFunction        *CallAgentFunction `json:"callAgentFunction"`
		RendererFunctionResponse *FunctionResponse  `json:"rendererFunctionResponse"`
		Error                    *RendererError     `json:"error"`
	}
	if err := json.Unmarshal(b, &wire); err != nil {
		return RendererMessage{}, fmt.Errorf("v10: decode message: %w", err)
	}
	out := RendererMessage{
		Version:                  wire.Version,
		CallAgentFunction:        wire.CallAgentFunction,
		RendererFunctionResponse: wire.RendererFunctionResponse,
		Error:                    wire.Error,
		Raw:                      m,
	}
	if wire.Action != nil {
		a := wire.Action.Action
		if wire.Action.Metadata != nil {
			a.Extensions = wire.Action.Metadata.Extensions
		}
		out.Action = &a
	}
	return out, nil
}
