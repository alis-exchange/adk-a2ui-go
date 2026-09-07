package kit

import "strings"

// Problem is one validation finding at a rendered instance path such as
// "messages[1].updateComponents.components[0]". Path may be empty for batch-wide findings and
// for the "... and N more" trailer that caps a long list.
type Problem struct {
	Path    string
	Message string
}

// String renders the problem as one line, "path: message", or just the message when Path is empty.
func (p Problem) String() string {
	if p.Path == "" {
		return p.Message
	}
	return p.Path + ": " + p.Message
}

// ValidationError is what v09.Validate and v10.Validate return when a batch is not valid, and
// what the generate tool hands back to the model as its error text: a header line followed by
// one "- path: message" line per problem. It is the error type consumers match on, so a caller
// can tell a model mistake (a *ValidationError, worth reporting back to the model for a retry)
// from an agent-side configuration failure:
//
//	var ve *kit.ValidationError
//	if errors.As(err, &ve) {
//		for _, p := range ve.Problems { ... }
//	}
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
