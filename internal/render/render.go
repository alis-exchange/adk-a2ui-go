// Package render holds the small formatting helpers the model-facing String methods of the
// inbound message types share across spec versions.
package render

import (
	"encoding/json"
	"fmt"
)

// Context renders an action context as compact JSON (keys sorted by encoding/json), or
// "with no context" when there is none.
func Context(ctx map[string]any) string {
	if len(ctx) == 0 {
		return "with no context"
	}
	b, err := json.Marshal(ctx)
	if err != nil {
		return fmt.Sprintf("with context %v", ctx)
	}
	return "with context " + string(b)
}

// Code renders an error code, naming its absence rather than printing nothing.
func Code(code string) string {
	if code == "" {
		return "(no code)"
	}
	return code
}

// Path renders " at <path>" for a JSON pointer, or nothing when there is none.
func Path(path string) string {
	if path == "" {
		return ""
	}
	return " at " + path
}
