package kit

import (
	"errors"
	"fmt"
	"testing"
)

func TestProblemString(t *testing.T) {
	if got := (Problem{Path: "messages[0].version", Message: `must be "v1.0"`}).String(); got != `messages[0].version: must be "v1.0"` {
		t.Errorf("String() = %q", got)
	}
	if got := (Problem{Message: "... and 5 more"}).String(); got != "... and 5 more" {
		t.Errorf("String() without a path = %q", got)
	}
}

func TestValidationErrorRendering(t *testing.T) {
	err := &ValidationError{Problems: []Problem{
		{Path: "messages[0]", Message: `missing property "version"`},
		{Message: "... and 2 more"},
	}}
	want := "validation failed. Fix the following and call the tool again:" +
		"\n- messages[0]: missing property \"version\"" +
		"\n- ... and 2 more"
	if got := err.Error(); got != want {
		t.Errorf("Error() =\n%s\nwant\n%s", got, want)
	}
}

func TestValidationErrorIsMatchable(t *testing.T) {
	wrapped := fmt.Errorf("tool failed: %w", &ValidationError{Problems: []Problem{{Path: "messages", Message: "must contain at least one message"}}})
	var ve *ValidationError
	if !errors.As(wrapped, &ve) {
		t.Fatalf("errors.As failed for %v", wrapped)
	}
	if len(ve.Problems) != 1 || ve.Problems[0].Path != "messages" {
		t.Errorf("Problems = %+v", ve.Problems)
	}
}
