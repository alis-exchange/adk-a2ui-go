// Package schema compiles the embedded A2UI JSON schemas with a catalog injected, validates
// instances against them, and turns validator errors into problems a model can act on.
package schema

import "go.alis.build/adk/a2ui/kit"

// Problem and ValidationError live in kit so consumers can match on them without importing an
// internal package; these aliases keep the rest of this module reading against schema.Problem.
type (
	// Problem is one validation finding at a rendered instance path. See [kit.Problem].
	Problem = kit.Problem
	// ValidationError is returned to the model as the tool error text. See [kit.ValidationError].
	ValidationError = kit.ValidationError
)
