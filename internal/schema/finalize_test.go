package schema

import (
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
