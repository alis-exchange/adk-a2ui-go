package kit

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type stubResolver struct {
	cats map[string]map[string]any
	err  error
}

func (s stubResolver) ResolveCatalog(_ context.Context, id string) (map[string]any, bool, error) {
	if s.err != nil {
		return nil, false, s.err
	}
	c, ok := s.cats[id]
	return c, ok, nil
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	if err := r.Register(map[string]any{"title": "no id"}); err == nil {
		t.Error("catalog without catalogId must be rejected")
	}
	if err := r.RegisterJSON([]byte(`{"catalogId":"acme:ui","components":{}}`)); err != nil {
		t.Fatal(err)
	}
	c, ok, err := r.ResolveCatalog(context.Background(), "acme:ui")
	if err != nil || !ok || c["catalogId"] != "acme:ui" {
		t.Errorf("got %v %v %v", c, ok, err)
	}
	if _, ok, _ := r.ResolveCatalog(context.Background(), "missing"); ok {
		t.Error("unknown id should not resolve")
	}
}

func TestChain(t *testing.T) {
	first := stubResolver{cats: map[string]map[string]any{"a": {"catalogId": "a", "from": "first"}}}
	second := stubResolver{cats: map[string]map[string]any{"a": {"catalogId": "a", "from": "second"}, "b": {"catalogId": "b"}}}
	ch := Chain(first, second)
	c, ok, _ := ch.ResolveCatalog(context.Background(), "a")
	if !ok || c["from"] != "first" {
		t.Error("first resolver should win")
	}
	if _, ok, _ := ch.ResolveCatalog(context.Background(), "b"); !ok {
		t.Error("second resolver should be consulted")
	}
	boom := errors.New("boom")
	if _, _, err := Chain(stubResolver{err: boom}, second).ResolveCatalog(context.Background(), "b"); !errors.Is(err, boom) {
		t.Error("resolver errors must propagate")
	}
}

func TestAgentCapabilities(t *testing.T) {
	got := AgentCapabilities(V09, []string{"x"}, true)
	want := map[string]any{"v0.9": map[string]any{"supportedCatalogIds": []string{"x"}, "acceptsInlineCatalogs": true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v", got)
	}
}
