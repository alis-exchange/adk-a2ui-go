package v09

import (
	"fmt"

	"go.alis.build/adk/a2ui/internal/render"
)

// ClientMessage is one decoded v0.9 or v0.9.1 client-to-server message. Exactly one of Action
// and Error is set. Raw is the message as received (the map handed to the decoder, not a
// copy) for anything the typed fields do not carry, such as the extra properties a generic
// error may add.
type ClientMessage struct {
	Version string
	Action  *Action
	Error   *ClientError
	Raw     map[string]any
}

// Action reports a user-initiated action from a component.
type Action struct {
	Name              string         `json:"name"`
	SurfaceID         string         `json:"surfaceId"`
	SourceComponentID string         `json:"sourceComponentId"`
	Timestamp         string         `json:"timestamp"` // ISO 8601 as sent; not parsed
	Context           map[string]any `json:"context"`
}

// ClientError reports a client-side error. Path is set only for VALIDATION_FAILED.
type ClientError struct {
	Code      string `json:"code"`
	SurfaceID string `json:"surfaceId"`
	Message   string `json:"message"`
	Path      string `json:"path,omitzero"`
}

// String renders the action as one line to hand to the model.
func (a *Action) String() string {
	return fmt.Sprintf("user action %q on surface %q from component %q %s", a.Name, a.SurfaceID, a.SourceComponentID, render.Context(a.Context))
}

// String renders the error as one line to hand to the model.
func (e *ClientError) String() string {
	return fmt.Sprintf("renderer error %s on surface %q%s: %s", render.Code(e.Code), e.SurfaceID, render.Path(e.Path), e.Message)
}
