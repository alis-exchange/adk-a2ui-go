package v10

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"go.alis.build/adk/a2ui/kit"
)

// collectCalls returns every object in v that has a string "call", nested ones included:
// a function call inside another call's args is a FunctionCall of its own.
func collectCalls(v any) []map[string]any {
	var out []map[string]any
	switch x := v.(type) {
	case map[string]any:
		if _, ok := x["call"].(string); ok {
			out = append(out, x)
		}
		for _, child := range x {
			out = append(out, collectCalls(child)...)
		}
	case []any:
		for _, child := range x {
			out = append(out, collectCalls(child)...)
		}
	}
	return out
}

// surfaceOf returns the surfaceId a message body names, whatever its key.
func surfaceOf(m map[string]any) string {
	for k, v := range m {
		if k == "version" {
			continue
		}
		if body, ok := v.(map[string]any); ok {
			if sid, _ := body["surfaceId"].(string); sid != "" {
				return sid
			}
		}
	}
	return ""
}

// TestRoundTripOfficialFunctionCalls sends every function call the official v1.0 examples
// contain back to the agent as a callAgentFunction, decodes it strictly against the surface's
// catalog, answers it through a dispatcher, and asks the renderer to run it again through
// NewCallRendererFunction; each produced message must pass Validate.
func TestRoundTripOfficialFunctionCalls(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "spec", "v1_0", "testdata", "examples", "*.json"))
	if len(files) == 0 {
		t.Fatal("no examples; run scripts/sync-spec.sh")
	}
	ctx := context.Background()
	opts := kit.ValidateOptions{Version: kit.V10, Strict: true}
	d := NewFunctionDispatcher()
	echo := func(_ context.Context, c *CallAgentFunction) (any, error) { return c.CallFunction.Args, nil }
	total := 0
	for _, f := range files {
		base := filepath.Base(f)
		msgs := loadExample(t, f)
		catalogOf := map[string]string{}
		for _, m := range msgs {
			if cs, ok := m["createSurface"].(map[string]any); ok {
				sid, _ := cs["surfaceId"].(string)
				catalogOf[sid], _ = cs["catalogId"].(string)
			}
		}
		for _, m := range msgs {
			sid := surfaceOf(m)
			for _, src := range collectCalls(m) {
				total++
				id := fmt.Sprintf("rt-%d", total)
				wire := map[string]any{"call": src["call"], "args": src["args"]}
				if cid, _ := src["catalogId"].(string); cid != "" {
					wire["catalogId"] = cid
				} else {
					wire["catalogId"] = catalogOf[sid]
				}
				inbound := map[string]any{"version": kit.V10, "callAgentFunction": map[string]any{"surfaceId": sid, "functionCallId": id, "callFunction": wire}}
				decoded, err := DecodeRendererMessage(ctx, inbound, opts)
				if err != nil {
					t.Errorf("%s %s: decode: %v", base, id, err)
					continue
				}
				got := decoded.CallAgentFunction
				if !reflect.DeepEqual(got.CallFunction.Args, src["args"]) {
					t.Errorf("%s %s: args changed in decoding: %#v", base, id, got.CallFunction.Args)
				}
				d.Register(got.CallFunction.Call, echo)
				if err := Validate(ctx, []map[string]any{d.Handle(ctx, got)}, opts); err != nil {
					t.Errorf("%s %s: response: %v", base, id, err)
				}
				if err := Validate(ctx, []map[string]any{NewCallRendererFunction(id, got.CallFunction)}, opts); err != nil {
					t.Errorf("%s %s: callRendererFunction: %v", base, id, err)
				}
			}
		}
	}
	if total < 20 {
		t.Fatalf("found only %d function calls across the examples; the walker or the examples are broken", total)
	}
	t.Logf("round-tripped %d function calls", total)
}
