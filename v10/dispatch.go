package v10

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"go.alis.build/adk/a2ui/kit"
)

// Error codes Handle puts in an agentFunctionResponse when the handler did not choose one.
const (
	FunctionNotFound = "FUNCTION_NOT_FOUND"
	FunctionFailed   = "FUNCTION_FAILED"
)

// FunctionHandler runs one agent-side function. The returned value must be JSON-encodable
// (it is re-encoded into a plain tree before it is sent). Return a *FunctionError to choose
// the error code the renderer sees, or any error with a Code() string method; other errors are
// reported as FunctionFailed with the error text as the message. Never return a typed-nil
// pointer as the error: it is not nil, and a foreign error type's own methods may panic on it.
type FunctionHandler func(ctx context.Context, call *CallAgentFunction) (any, error)

// FunctionDispatcher maps the function names a renderer may call to handlers. Register the
// agent's functions once at startup, then hand every decoded CallAgentFunction to Handle and
// forward what it returns. Dispatch is by name only: the catalog a call names was already
// applied at decode time when it resolved. Safe for concurrent use.
type FunctionDispatcher struct {
	mu       sync.RWMutex
	handlers map[string]FunctionHandler
}

func NewFunctionDispatcher() *FunctionDispatcher {
	return &FunctionDispatcher{handlers: map[string]FunctionHandler{}}
}

// Register binds h to name, replacing any earlier handler for it. Like http.HandleFunc it
// panics on an empty name or a nil handler: both are programming errors made at startup.
func (d *FunctionDispatcher) Register(name string, h FunctionHandler) {
	if name == "" {
		panic("a2ui/v10: Register with empty function name")
	}
	if h == nil {
		panic(fmt.Sprintf("a2ui/v10: Register(%q) with nil handler", name))
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.handlers[name] = h
}

// Handle runs the handler registered for call and returns the agentFunctionResponse message
// to send back, carrying call.FunctionCallID. It never returns an error: an unknown function,
// a failing handler, and a result that cannot be encoded all become error responses, so the
// renderer always hears back. A handler that panics is not recovered.
func (d *FunctionDispatcher) Handle(ctx context.Context, call *CallAgentFunction) map[string]any {
	if call == nil {
		return NewAgentFunctionError("", FunctionFailed, "nil call")
	}
	name := call.CallFunction.Call
	d.mu.RLock()
	h, ok := d.handlers[name]
	d.mu.RUnlock()
	if !ok {
		return NewAgentFunctionError(call.FunctionCallID, FunctionNotFound, fmt.Sprintf("no agent function named %q", name))
	}
	value, err := h(ctx, call)
	if err != nil {
		code, message := errorCode(err)
		return NewAgentFunctionError(call.FunctionCallID, code, message)
	}
	plain, err := toJSONTree(value)
	if err != nil {
		return NewAgentFunctionError(call.FunctionCallID, FunctionFailed, "encode result: "+err.Error())
	}
	return NewAgentFunctionResponse(call.FunctionCallID, plain)
}

// errorCode picks the wire code and message for a handler error: a *FunctionError anywhere in
// the chain wins, then any error exposing Code(), then FunctionFailed. An empty code from
// either source falls back to FunctionFailed so the response stays meaningful. A typed-nil
// pointer error (var fe *FunctionError; return nil, fe) is answered without calling any method
// on it, since those would dereference nil inside a Handle that promises never to fail.
func errorCode(err error) (code, message string) {
	if isNilPointer(err) {
		return FunctionFailed, fmt.Sprintf("handler returned a typed-nil %T", err)
	}
	var fe *FunctionError
	var coded interface{ Code() string }
	switch {
	case errors.As(err, &fe) && fe != nil:
		code, message = fe.Code, fe.Message
	case errors.As(err, &coded) && !isNilPointer(coded):
		code, message = coded.Code(), err.Error()
	default:
		code, message = FunctionFailed, err.Error()
	}
	if code == "" {
		code = FunctionFailed
	}
	return code, message
}

// isNilPointer reports whether v is a non-nil interface holding a nil pointer.
func isNilPointer(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

// toJSONTree re-encodes a handler result as the plain map/slice/scalar tree the transport and
// Validate expect, so a struct or a typed slice comes out the way it would on the wire.
func toJSONTree(v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// NewCallRendererFunction builds the callRendererFunction message that asks the renderer to run
// call. The outbound schema requires call.CatalogID; it is left out when empty so Validate
// reports the omission at the call path rather than this constructor guessing a catalog.
// call.Args is sent whenever it is non-nil, an empty map included, because catalogs require the
// key. The function's catalog definition must allow agent callers (allowedCallers agentOnly or
// rendererOrAgent); Validate rejects a call to a rendererOnly function, which is every function
// of the basic catalog.
func NewCallRendererFunction(functionCallID string, call FunctionCall) map[string]any {
	cf := map[string]any{"call": call.Call}
	if call.CatalogID != "" {
		cf["catalogId"] = call.CatalogID
	}
	if call.Args != nil {
		cf["args"] = call.Args
	}
	return map[string]any{
		"version": kit.V10,
		"callRendererFunction": map[string]any{
			"functionCallId": functionCallID,
			"callFunction":   cf,
		},
	}
}

// NewAgentFunctionResponse builds the agentFunctionResponse that returns value for the call
// identified by functionCallID. A nil value is sent as an explicit null: the schema requires
// the "value" key whenever there is no error.
func NewAgentFunctionResponse(functionCallID string, value any) map[string]any {
	return map[string]any{
		"version": kit.V10,
		"agentFunctionResponse": map[string]any{
			"functionCallId": functionCallID,
			"value":          value,
		},
	}
}

// NewAgentFunctionError builds the agentFunctionResponse that reports a failure for the call
// identified by functionCallID.
func NewAgentFunctionError(functionCallID, code, message string) map[string]any {
	return map[string]any{
		"version": kit.V10,
		"agentFunctionResponse": map[string]any{
			"functionCallId": functionCallID,
			"error":          map[string]any{"code": code, "message": message},
		},
	}
}
