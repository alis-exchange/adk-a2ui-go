package v10

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.alis.build/adk/a2ui/kit"
)

type inboundCase struct {
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Strict       bool           `json:"strict"`
	Message      map[string]any `json:"message"`
	WantPaths    []string       `json:"wantPaths"`
	WantContains []string       `json:"wantContains"`
	WantString   string         `json:"wantString"`
}

// stringOf renders whichever field is set, the way a consumer feeding the model would.
func stringOf(t *testing.T, m *RendererMessage) string {
	t.Helper()
	set := 0
	got := ""
	if m.Action != nil {
		set++
		got = m.Action.String()
	}
	if m.CallAgentFunction != nil {
		set++
	}
	if m.RendererFunctionResponse != nil {
		set++
		if m.RendererFunctionResponse.Error != nil {
			got = m.RendererFunctionResponse.Error.Error()
		}
	}
	if m.Error != nil {
		set++
		got = m.Error.String()
	}
	if set != 1 {
		t.Fatalf("exactly one field must be set, got %d: %+v", set, m)
	}
	return got
}

func TestDecodeRendererMessageFixtures(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("testdata", "inbound", "*.json"))
	if len(files) == 0 {
		t.Fatal("no inbound fixtures")
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var c inboundCase
		if err := json.Unmarshal(b, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		t.Run(c.Name, func(t *testing.T) {
			msg, err := DecodeRendererMessage(context.Background(), c.Message, kit.ValidateOptions{Version: c.Version, Strict: c.Strict})
			if len(c.WantPaths) == 0 && len(c.WantContains) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got := stringOf(t, msg); got != c.WantString {
					t.Errorf("String() = %q, want %q", got, c.WantString)
				}
				return
			}
			var ve *kit.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want *kit.ValidationError, got %v", err)
			}
			text := ve.Error()
			for _, p := range c.WantPaths {
				found := false
				for _, pr := range ve.Problems {
					if pr.Path == p {
						found = true
					}
				}
				if !found {
					t.Errorf("path %q not reported in:\n%s", p, text)
				}
			}
			for _, s := range c.WantContains {
				if !strings.Contains(text, s) {
					t.Errorf("%q not in:\n%s", s, text)
				}
			}
		})
	}
}

func TestDecodeRendererMessageTypedFields(t *testing.T) {
	ctx := context.Background()
	call := map[string]any{"version": "v1.0", "callAgentFunction": map[string]any{"surfaceId": "s1", "functionCallId": "fc-1",
		"callFunction": map[string]any{"call": "formatDate", "catalogId": CatalogIDBasic, "args": map[string]any{"value": "2026-09-07", "format": "yyyy"}}}}
	msg, err := DecodeRendererMessage(ctx, call, kit.ValidateOptions{Version: kit.V10, Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	caf := msg.CallAgentFunction
	if caf.SurfaceID != "s1" || caf.FunctionCallID != "fc-1" || caf.CallFunction.Call != "formatDate" || caf.CallFunction.CatalogID != CatalogIDBasic {
		t.Errorf("CallAgentFunction = %+v", caf)
	}
	if !reflect.DeepEqual(caf.CallFunction.Args, map[string]any{"value": "2026-09-07", "format": "yyyy"}) {
		t.Errorf("Args = %#v", caf.CallFunction.Args)
	}
	if msg.Raw["callAgentFunction"] == nil {
		t.Error("Raw must be the message as received")
	}

	action := map[string]any{"version": "v1.0", "action": map[string]any{"name": "order", "userMessage": "Placed", "surfaceId": "s1", "sourceComponentId": "btn",
		"timestamp": "2026-09-07T10:00:00Z", "context": map[string]any{"qty": 2.0}, "metadata": map[string]any{"extensions": map[string]any{"vendor_trace": "abc"}}}}
	msg, err = DecodeRendererMessage(ctx, action, kit.ValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if msg.Action.UserMessage != "Placed" || msg.Action.Extensions["vendor_trace"] != "abc" || msg.Action.Context["qty"] != 2.0 {
		t.Errorf("Action = %+v", msg.Action)
	}

	resp := map[string]any{"version": "v1.0", "rendererFunctionResponse": map[string]any{"functionCallId": "fc-1", "value": nil}}
	msg, err = DecodeRendererMessage(ctx, resp, kit.ValidateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if msg.RendererFunctionResponse.FunctionCallID != "fc-1" || msg.RendererFunctionResponse.Value != nil || msg.RendererFunctionResponse.Error != nil {
		t.Errorf("FunctionResponse = %+v", msg.RendererFunctionResponse)
	}

	resp = map[string]any{"version": "v1.0", "rendererFunctionResponse": map[string]any{"functionCallId": "fc-1", "value": map[string]any{"w": 1024.0}}}
	msg, _ = DecodeRendererMessage(ctx, resp, kit.ValidateOptions{})
	if v, _ := msg.RendererFunctionResponse.Value.(map[string]any); v["w"] != 1024.0 {
		t.Errorf("Value = %#v", msg.RendererFunctionResponse.Value)
	}
}

func TestDecodeRendererMessageInlineCatalogAndResolver(t *testing.T) {
	ctx := context.Background()
	custom := map[string]any{
		"catalogId": "https://example.com/agent.json",
		"$defs": map[string]any{
			"anyComponent": map[string]any{"type": "object"},
			"anyFunction":  map[string]any{"oneOf": []any{map[string]any{"$ref": "#/functions/verifyProvider"}}},
		},
		"functions": map[string]any{"verifyProvider": map[string]any{
			"type": "object", "returnType": "boolean",
			"properties": map[string]any{
				"call": map[string]any{"const": "verifyProvider"},
				"args": map[string]any{"type": "object", "properties": map[string]any{"providerId": map[string]any{"type": "string"}}, "required": []any{"providerId"}, "unevaluatedProperties": false},
			},
			"required": []any{"call", "args"},
		}},
	}
	m := map[string]any{"version": "v1.0", "callAgentFunction": map[string]any{"surfaceId": "s1", "functionCallId": "fc-1",
		"callFunction": map[string]any{"call": "verifyProvider", "catalogId": "https://example.com/agent.json", "args": map[string]any{"providerId": 7.0}}}}

	inline := kit.ValidateOptions{Strict: true, Params: kit.VersionParams{InlineCatalogs: []map[string]any{custom}}}
	_, err := DecodeRendererMessage(ctx, m, inline)
	if err == nil || !strings.Contains(err.Error(), "callAgentFunction.callFunction.args.providerId") {
		t.Errorf("inline catalog must validate the call, got %v", err)
	}

	reg := kit.NewRegistry()
	if err := reg.Register(custom); err != nil {
		t.Fatal(err)
	}
	_, err = DecodeRendererMessage(ctx, m, kit.ValidateOptions{Strict: true, Resolver: reg})
	if err == nil || !strings.Contains(err.Error(), "callAgentFunction.callFunction.args.providerId") {
		t.Errorf("registered catalog must validate the call, got %v", err)
	}

	_, err = DecodeRendererMessage(ctx, m, kit.ValidateOptions{Resolver: failingResolver{}})
	var ve *kit.ValidationError
	if err == nil || errors.As(err, &ve) {
		t.Errorf("a resolver failure is a plain error, got %v", err)
	}
}

type failingResolver struct{}

func (failingResolver) ResolveCatalog(context.Context, string) (map[string]any, bool, error) {
	return nil, false, errors.New("backend down")
}

func TestDecodeRendererMessageUnsupportedVersion(t *testing.T) {
	_, err := DecodeRendererMessage(context.Background(), map[string]any{"version": "v0.9"}, kit.ValidateOptions{Version: "v0.9"})
	var ve *kit.ValidationError
	if err == nil || errors.As(err, &ve) {
		t.Errorf("an unsupported pinned version is a plain error, got %v", err)
	}
}

func TestDecodeRendererMessagesListPaths(t *testing.T) {
	ok := map[string]any{"version": "v1.0", "error": map[string]any{"code": "X", "surfaceId": "s", "message": "m"}}
	bad := map[string]any{"version": "v1.0", "action": map[string]any{"name": "n"}}
	msgs, err := DecodeRendererMessages(context.Background(), []map[string]any{ok, ok}, kit.ValidateOptions{Version: kit.V10})
	if err != nil || len(msgs) != 2 {
		t.Fatalf("got %d messages, %v", len(msgs), err)
	}
	_, err = DecodeRendererMessages(context.Background(), []map[string]any{ok, bad}, kit.ValidateOptions{Version: kit.V10})
	var ve *kit.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want *kit.ValidationError, got %v", err)
	}
	found := false
	for _, p := range ve.Problems {
		if p.Path == "messages[1].action" {
			found = true
		}
	}
	if !found {
		t.Errorf("list problems must carry messages[i] paths:\n%s", ve)
	}
}

// TestToRendererMessageTypedConversionFailure exercises the fallback for a field the v1.0
// schema types today but a future upstream loosening could open up, the same way v0.9's
// generic error already has: a value the envelope pass let through that still does not fit
// the Go type. toRendererMessage must report that as a problem, not a plain error.
func TestToRendererMessageTypedConversionFailure(t *testing.T) {
	m := map[string]any{"version": "v1.0", "callAgentFunction": map[string]any{"surfaceId": "s1", "functionCallId": 5.0,
		"callFunction": map[string]any{"call": "formatDate", "args": map[string]any{}}}}
	_, p, err := toRendererMessage(m, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("want a non-nil problem")
	}
	if p.Path != "" {
		t.Errorf("Path = %q, want empty", p.Path)
	}
	if !strings.Contains(p.Message, "unexpected type") {
		t.Errorf("Message = %q, want it to contain %q", p.Message, "unexpected type")
	}
}

func TestRendererStringsOnZeroValues(t *testing.T) {
	if got := (&Action{}).String(); got != `user action "" on surface "" from component "" with no context` {
		t.Errorf("Action: %q", got)
	}
	if got := (&RendererError{}).String(); got != `renderer error (no code): ` {
		t.Errorf("RendererError: %q", got)
	}
	if got := (&FunctionError{Code: "X"}).Error(); got != "X" {
		t.Errorf("FunctionError without message: %q", got)
	}
	if got := (&FunctionError{}).Error(); got != "(no code)" {
		t.Errorf("FunctionError zero: %q", got)
	}
}
