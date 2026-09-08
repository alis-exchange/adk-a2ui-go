package v10

import (
	"fmt"

	"go.alis.build/adk/a2ui/internal/render"
)

// RendererMessage is one decoded v1.0 renderer-to-agent message. Exactly one of Action,
// CallAgentFunction, RendererFunctionResponse and Error is set. Raw is the message as
// received (the map handed to the decoder, not a copy) for anything the typed fields do not
// carry, such as the extra properties a generic error may add.
type RendererMessage struct {
	Version                  string
	Action                   *Action
	CallAgentFunction        *CallAgentFunction
	RendererFunctionResponse *FunctionResponse
	Error                    *RendererError
	Raw                      map[string]any
}

// Action reports a user-initiated action from a component.
type Action struct {
	Name              string         `json:"name"`
	UserMessage       string         `json:"userMessage,omitzero"`
	SurfaceID         string         `json:"surfaceId"`
	SourceComponentID string         `json:"sourceComponentId"`
	Timestamp         string         `json:"timestamp"` // ISO 8601 as sent; not parsed
	Context           map[string]any `json:"context"`
	Extensions        map[string]any `json:"-"` // action.metadata.extensions; nil when absent, decode-only: json.Marshal does not write it back
}

// CallAgentFunction asks the agent to run a function on the renderer's behalf. The agent
// answers with an agentFunctionResponse carrying the same FunctionCallID; see
// [FunctionDispatcher].
type CallAgentFunction struct {
	SurfaceID      string       `json:"surfaceId"`
	FunctionCallID string       `json:"functionCallId"`
	CallFunction   FunctionCall `json:"callFunction"`
}

// FunctionCall is the wire shape shared by callAgentFunction and callRendererFunction: the
// function name, the catalog that defines it, and its arguments under "args". Args uses
// omitzero rather than omitempty so an explicitly empty {} survives marshalling; catalogs
// require the key.
type FunctionCall struct {
	Call      string         `json:"call"`
	CatalogID string         `json:"catalogId,omitzero"`
	Args      map[string]any `json:"args,omitzero"`
}

// FunctionResponse is the result of a call in either direction. Exactly one of Value and
// Error is set; Value may be nil (JSON null) when it is the one set. This type is for decoding
// only: Value has no omitzero, so marshalling a FunctionResponse directly would write Value
// beside Error even when Error is the one set. Build outbound messages with
// [NewAgentFunctionResponse] and [NewAgentFunctionError] instead.
type FunctionResponse struct {
	FunctionCallID string         `json:"functionCallId"`
	Value          any            `json:"value"`
	Error          *FunctionError `json:"error,omitzero"`
}

// FunctionError is the error object of a function response. It implements error so a
// [FunctionHandler] can return one to choose the code the renderer sees.
type FunctionError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *FunctionError) Error() string {
	if e == nil {
		return "nil *FunctionError"
	}
	if e.Message == "" {
		return render.Code(e.Code)
	}
	return render.Code(e.Code) + ": " + e.Message
}

// RendererError reports a renderer-side error. Path is set for the validation codes
// (VALIDATION_FAILED, UNALLOWED_PARENT, UNALLOWED_CHILD); a generic error names either the
// surface or the function call it concerns, never both.
type RendererError struct {
	Code           string `json:"code"`
	Message        string `json:"message"`
	SurfaceID      string `json:"surfaceId,omitzero"`
	Path           string `json:"path,omitzero"`
	FunctionCallID string `json:"functionCallId,omitzero"`
}

// String renders the action as one line to hand to the model.
func (a *Action) String() string {
	s := fmt.Sprintf("user action %q on surface %q from component %q %s", a.Name, a.SurfaceID, a.SourceComponentID, render.Context(a.Context))
	if a.UserMessage != "" {
		s += fmt.Sprintf(" (user said: %q)", a.UserMessage)
	}
	return s
}

// String renders the error as one line to hand to the model.
func (e *RendererError) String() string {
	where := ""
	switch {
	case e.SurfaceID != "":
		where = fmt.Sprintf(" on surface %q", e.SurfaceID)
	case e.FunctionCallID != "":
		where = fmt.Sprintf(" for function call %q", e.FunctionCallID)
	}
	return fmt.Sprintf("renderer error %s%s%s: %s", render.Code(e.Code), where, render.Path(e.Path), e.Message)
}
