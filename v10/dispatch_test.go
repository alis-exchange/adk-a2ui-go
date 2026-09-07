package v10

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"go.alis.build/adk/a2ui/kit"
)

var strictV10 = kit.ValidateOptions{Version: kit.V10, Strict: true}

func mustValidate(t *testing.T, msg map[string]any) {
	t.Helper()
	if err := Validate(context.Background(), []map[string]any{msg}, strictV10); err != nil {
		t.Errorf("produced message must pass Validate: %v\n%v", err, msg)
	}
}

func responseOf(t *testing.T, msg map[string]any) map[string]any {
	t.Helper()
	if msg["version"] != kit.V10 {
		t.Errorf("version = %v", msg["version"])
	}
	r, ok := msg["agentFunctionResponse"].(map[string]any)
	if !ok {
		t.Fatalf("not an agentFunctionResponse: %v", msg)
	}
	return r
}

func errorOf(t *testing.T, msg map[string]any) (code, message string) {
	t.Helper()
	e, ok := responseOf(t, msg)["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error in %v", msg)
	}
	code, _ = e["code"].(string)
	message, _ = e["message"].(string)
	return code, message
}

func call(name string, args map[string]any) *CallAgentFunction {
	return &CallAgentFunction{SurfaceID: "s1", FunctionCallID: "fc-1", CallFunction: FunctionCall{Call: name, Args: args}}
}

type codedErr struct{ code string }

func (e codedErr) Error() string { return "coded failure" }
func (e codedErr) Code() string  { return e.code }

func TestHandleNotFound(t *testing.T) {
	d := NewFunctionDispatcher()
	msg := d.Handle(context.Background(), call("verifyProvider", nil))
	code, message := errorOf(t, msg)
	if code != FunctionNotFound || !strings.Contains(message, `"verifyProvider"`) {
		t.Errorf("got %s: %s", code, message)
	}
	if responseOf(t, msg)["functionCallId"] != "fc-1" {
		t.Error("functionCallId must be copied into the response")
	}
	mustValidate(t, msg)
}

func TestHandleErrors(t *testing.T) {
	d := NewFunctionDispatcher()
	d.Register("plain", func(context.Context, *CallAgentFunction) (any, error) { return nil, errors.New("boom") })
	d.Register("typed", func(context.Context, *CallAgentFunction) (any, error) {
		return nil, &FunctionError{Code: "PROVIDER_UNKNOWN", Message: "no such provider"}
	})
	d.Register("wrappedTyped", func(context.Context, *CallAgentFunction) (any, error) {
		return nil, errors.Join(errors.New("outer"), &FunctionError{Code: "INNER", Message: "inner"})
	})
	d.Register("coded", func(context.Context, *CallAgentFunction) (any, error) { return nil, codedErr{"RATE_LIMITED"} })
	d.Register("emptyCode", func(context.Context, *CallAgentFunction) (any, error) { return nil, codedErr{""} })
	cases := map[string][2]string{
		"plain":        {FunctionFailed, "boom"},
		"typed":        {"PROVIDER_UNKNOWN", "no such provider"},
		"wrappedTyped": {"INNER", "inner"},
		"coded":        {"RATE_LIMITED", "coded failure"},
		"emptyCode":    {FunctionFailed, "coded failure"},
	}
	for name, want := range cases {
		msg := d.Handle(context.Background(), call(name, nil))
		code, message := errorOf(t, msg)
		if code != want[0] || message != want[1] {
			t.Errorf("%s: got %s: %s, want %s: %s", name, code, message, want[0], want[1])
		}
		mustValidate(t, msg)
	}
}

func TestHandleValues(t *testing.T) {
	type result struct {
		OK    bool     `json:"ok"`
		Names []string `json:"names"`
	}
	d := NewFunctionDispatcher()
	d.Register("struct", func(context.Context, *CallAgentFunction) (any, error) {
		return result{OK: true, Names: []string{"a"}}, nil
	})
	d.Register("nil", func(context.Context, *CallAgentFunction) (any, error) { return nil, nil })
	d.Register("echo", func(_ context.Context, c *CallAgentFunction) (any, error) { return c.CallFunction.Args, nil })
	d.Register("bad", func(context.Context, *CallAgentFunction) (any, error) { return make(chan int), nil })

	msg := d.Handle(context.Background(), call("struct", nil))
	if v := responseOf(t, msg)["value"]; !reflect.DeepEqual(v, map[string]any{"ok": true, "names": []any{"a"}}) {
		t.Errorf("struct value must be a plain JSON tree, got %#v", v)
	}
	mustValidate(t, msg)

	msg = d.Handle(context.Background(), call("nil", nil))
	if v, present := responseOf(t, msg)["value"]; !present || v != nil {
		t.Errorf("nil result must be sent as a present null value, got %#v (present=%v)", v, present)
	}
	mustValidate(t, msg)

	msg = d.Handle(context.Background(), call("echo", map[string]any{"providerId": "PRV-102"}))
	if v := responseOf(t, msg)["value"]; !reflect.DeepEqual(v, map[string]any{"providerId": "PRV-102"}) {
		t.Errorf("echo = %#v", v)
	}
	mustValidate(t, msg)

	msg = d.Handle(context.Background(), call("bad", nil))
	if code, message := errorOf(t, msg); code != FunctionFailed || !strings.Contains(message, "encode result") {
		t.Errorf("unencodable value: got %s: %s", code, message)
	}
	mustValidate(t, msg)
}

func TestHandleNilCall(t *testing.T) {
	msg := NewFunctionDispatcher().Handle(context.Background(), nil)
	if code, _ := errorOf(t, msg); code != FunctionFailed {
		t.Errorf("nil call: %v", msg)
	}
	mustValidate(t, msg)
}

func TestRegisterReplacesAndPanicsOnMisuse(t *testing.T) {
	d := NewFunctionDispatcher()
	d.Register("f", func(context.Context, *CallAgentFunction) (any, error) { return 1.0, nil })
	d.Register("f", func(context.Context, *CallAgentFunction) (any, error) { return 2.0, nil })
	if v := responseOf(t, d.Handle(context.Background(), call("f", nil)))["value"]; v != 2.0 {
		t.Errorf("later registration must win, got %v", v)
	}
	for name, fn := range map[string]func(){
		"empty name":  func() { d.Register("", func(context.Context, *CallAgentFunction) (any, error) { return nil, nil }) },
		"nil handler": func() { d.Register("g", nil) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s must panic", name)
				}
			}()
			fn()
		}()
	}
}

func TestConstructorsProduceValidMessages(t *testing.T) {
	withArgs := NewCallRendererFunction("fc-1", FunctionCall{Call: "formatDate", CatalogID: CatalogIDBasic, Args: map[string]any{"value": "2026-09-07", "format": "yyyy"}})
	mustValidate(t, withArgs)
	cf := withArgs["callRendererFunction"].(map[string]any)["callFunction"].(map[string]any)
	if cf["catalogId"] != CatalogIDBasic || cf["args"].(map[string]any)["format"] != "yyyy" {
		t.Errorf("callFunction = %v", cf)
	}

	emptyArgs := NewCallRendererFunction("fc-2", FunctionCall{Call: "formatDate", CatalogID: CatalogIDBasic, Args: map[string]any{}})
	if _, present := emptyArgs["callRendererFunction"].(map[string]any)["callFunction"].(map[string]any)["args"]; !present {
		t.Error("an empty args map must still be sent")
	}

	noArgs := NewCallRendererFunction("fc-3", FunctionCall{Call: "formatDate", CatalogID: CatalogIDBasic})
	if _, present := noArgs["callRendererFunction"].(map[string]any)["callFunction"].(map[string]any)["args"]; present {
		t.Error("nil args must not be sent")
	}
	if err := Validate(context.Background(), []map[string]any{noArgs}, strictV10); err == nil || !strings.Contains(err.Error(), `missing property "args"`) {
		t.Errorf("Validate must report the missing args: %v", err)
	}

	noCatalog := NewCallRendererFunction("fc-4", FunctionCall{Call: "formatDate", Args: map[string]any{"value": "x", "format": "y"}})
	if err := Validate(context.Background(), []map[string]any{noCatalog}, strictV10); err == nil || !strings.Contains(err.Error(), `missing property "catalogId"`) {
		t.Errorf("Validate must report the missing catalogId: %v", err)
	}

	mustValidate(t, NewAgentFunctionResponse("fc-5", map[string]any{"ok": true}))
	mustValidate(t, NewAgentFunctionResponse("fc-6", nil))
	mustValidate(t, NewAgentFunctionError("fc-7", FunctionFailed, "boom"))
}
