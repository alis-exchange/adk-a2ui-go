package schema

import (
	"fmt"
	"reflect"
	"testing"
)

func TestFinalizeDedupesAndSorts(t *testing.T) {
	in := []Problem{
		{Path: "messages[2]", Message: "b"},
		{Path: "messages[0]", Message: "a"},
		{Path: "messages[2]", Message: "b"},
		{Path: "messages[0]", Message: "z"},
	}
	want := []Problem{
		{Path: "messages[0]", Message: "a"},
		{Path: "messages[0]", Message: "z"},
		{Path: "messages[2]", Message: "b"},
	}
	got := Finalize(in)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Finalize(%v) = %v, want %v", in, got, want)
	}
}

// TestFinalizeNaturalOrder pins the order the model reads problems in: indices compare as
// numbers, so messages[10] follows messages[2] and components[10] follows components[9].
func TestFinalizeNaturalOrder(t *testing.T) {
	in := []Problem{
		{Path: "messages[10]", Message: "j"},
		{Path: "messages[2].updateComponents.components[10].id", Message: "c10"},
		{Path: "messages[2]", Message: "b"},
		{Path: "messages[2].updateComponents.components[9].id", Message: "c9"},
		{Path: "messages[1]", Message: "a"},
	}
	want := []Problem{
		{Path: "messages[1]", Message: "a"},
		{Path: "messages[2]", Message: "b"},
		{Path: "messages[2].updateComponents.components[9].id", Message: "c9"},
		{Path: "messages[2].updateComponents.components[10].id", Message: "c10"},
		{Path: "messages[10]", Message: "j"},
	}
	if got := Finalize(in); !reflect.DeepEqual(got, want) {
		t.Errorf("Finalize(%v) =\n%v\nwant\n%v", in, got, want)
	}
}

func TestFinalizeCaps(t *testing.T) {
	var in []Problem
	for i := 0; i < 25; i++ {
		in = append(in, Problem{Path: fmt.Sprintf("messages[%d]", i), Message: "boom"})
	}
	got := Finalize(in)
	if len(got) != maxProblems+1 {
		t.Fatalf("len = %d, want %d", len(got), maxProblems+1)
	}
	if got[maxProblems] != (Problem{Message: "... and 5 more"}) {
		t.Errorf("trailer = %+v", got[maxProblems])
	}
	if got[0].Path != "messages[0]" || got[maxProblems-1].Path != "messages[19]" {
		t.Errorf("kept the wrong window: %+v ... %+v", got[0], got[maxProblems-1])
	}
}
