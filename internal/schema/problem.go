// Package schema compiles the embedded A2UI JSON schemas with a catalog injected, validates
// instances against them, and turns validator errors into problems a model can act on.
package schema

import "strings"

// Problem is one validation finding at a rendered instance path such as
// "messages[1].updateComponents.components[0]". Path may be empty for batch-wide findings.
type Problem struct {
	Path    string
	Message string
}

func (p Problem) String() string {
	if p.Path == "" {
		return p.Message
	}
	return p.Path + ": " + p.Message
}

// ValidationError is returned to the model as the tool error text.
type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	var b strings.Builder
	b.WriteString("validation failed. Fix the following and call the tool again:")
	for _, p := range e.Problems {
		b.WriteString("\n- ")
		b.WriteString(p.String())
	}
	return b.String()
}
