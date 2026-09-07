package v09

import (
	"encoding/json"
	"fmt"

	"go.alis.build/adk/a2ui/internal/schema"
	"go.alis.build/adk/a2ui/kit"
	"go.alis.build/adk/a2ui/spec"
)

// clientMessageKeys are the message keys client_to_server.json allows beside "version".
var clientMessageKeys = []string{"action", "error"}

// DecodeClientMessage validates one client-to-server message against the embedded
// client_to_server.json and returns it typed. version is "", kit.V09 or kit.V091: a pinned
// version must match the message's "version"; "" accepts either wire version. Problems come
// back as a *kit.ValidationError with paths such as "action.name"; any other error means the
// pinned version is not one this package serves. The error's header line is written for the
// outbound tool; inbound callers should read Problems.
func DecodeClientMessage(m map[string]any, version string) (*ClientMessage, error) {
	msgs, err := decodeClientMessages([]map[string]any{m}, version, "")
	if err != nil {
		return nil, err
	}
	return &msgs[0], nil
}

// DecodeClientMessages decodes a client-to-server message list. Problems carry "messages[i]"
// paths, and the whole list is rejected when any message has one.
func DecodeClientMessages(ms []map[string]any, version string) ([]ClientMessage, error) {
	return decodeClientMessages(ms, version, "messages")
}

func decodeClientMessages(ms []map[string]any, version, prefix string) ([]ClientMessage, error) {
	switch version {
	case "", kit.V09, kit.V091:
	default:
		return nil, fmt.Errorf("v09: unsupported version %q", version)
	}
	envelope, err := schema.For(spec.MajorV09).Compile(schema.CompileOptions{Entry: schema.EntryInboundV09, V091: version != kit.V09})
	if err != nil {
		return nil, err
	}
	var problems []schema.Problem
	out := make([]ClientMessage, 0, len(ms))
	for i, m := range ms {
		path := prefix
		if prefix != "" {
			path = fmt.Sprintf("%s[%d]", prefix, i)
		}
		p := schema.InboundPrecheck(m, version, clientMessageKeys, path)
		if len(p) == 0 {
			if err := envelope.Validate(m); err != nil {
				p = schema.Format(err, m, path)
			}
		}
		if len(p) == 0 {
			// client_to_server.json types the generic error's "code" only as "not
			// VALIDATION_FAILED", so a number passes the envelope pass and would otherwise
			// blow up the typed conversion below; catch it here with the same message shape
			// the schema itself would use.
			if e, ok := m["error"].(map[string]any); ok {
				if code, present := e["code"]; present {
					if _, isString := code.(string); !isString {
						p = []schema.Problem{{Path: schema.JoinPath(path, "error.code"), Message: "must be of type string, got " + schema.JSONType(code)}}
					}
				}
			}
		}
		if len(p) > 0 {
			problems = append(problems, p...)
			continue
		}
		cm, cp, err := toClientMessage(m, path)
		if err != nil {
			return nil, err
		}
		if cp != nil {
			problems = append(problems, *cp)
			continue
		}
		out = append(out, cm)
	}
	if problems = schema.Finalize(problems); len(problems) > 0 {
		return nil, &schema.ValidationError{Problems: problems}
	}
	return out, nil
}

// toClientMessage converts a validated message into its typed form by way of JSON, which is
// what the map already is. The v0.9 schema does not type every field (the generic error's code,
// guarded above, is one gap of possibly several), so a value the envelope pass let through can
// still fail here; that surfaces as a validation problem at path, not a plain error. A
// json.Marshal failure (a value that is not JSON at all, such as a channel or a func) stays a
// plain error, since that is a caller-side bug rather than a renderer mistake.
func toClientMessage(m map[string]any, path string) (ClientMessage, *schema.Problem, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return ClientMessage{}, nil, fmt.Errorf("v09: encode message: %w", err)
	}
	var wire struct {
		Version string       `json:"version"`
		Action  *Action      `json:"action"`
		Error   *ClientError `json:"error"`
	}
	if err := json.Unmarshal(b, &wire); err != nil {
		return ClientMessage{}, &schema.Problem{Path: path, Message: "has a field of an unexpected type: " + err.Error()}, nil
	}
	return ClientMessage{Version: wire.Version, Action: wire.Action, Error: wire.Error, Raw: m}, nil, nil
}
