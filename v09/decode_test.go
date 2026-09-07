package v09

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.alis.build/adk/a2ui/kit"
)

// TestToClientMessageTypedConversionFailure exercises the fallback the guard in
// decodeClientMessages cannot reach through the public API once every known typed-conversion
// hole has a curated check in front of it: a field the schema pass let through but that still
// does not fit the Go type. toClientMessage must report that as a problem, not a plain error.
func TestToClientMessageTypedConversionFailure(t *testing.T) {
	m := map[string]any{"version": "v0.9", "action": map[string]any{"name": 5.0, "surfaceId": "s", "sourceComponentId": "c", "timestamp": "t", "context": map[string]any{}}}
	_, p, err := toClientMessage(m, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("want a non-nil problem")
	}
	if p.Path != "" {
		t.Errorf("Path = %q, want empty", p.Path)
	}
	if !strings.Contains(p.Message, "unexpected type") {
		t.Errorf("Message = %q, want it to contain %q", p.Message, "unexpected type")
	}
}

type inboundCase struct {
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Message      map[string]any `json:"message"`
	WantPaths    []string       `json:"wantPaths"`
	WantContains []string       `json:"wantContains"`
	WantString   string         `json:"wantString"`
}

func TestDecodeClientMessageFixtures(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("testdata", "inbound", "*.json"))
	if len(files) == 0 {
		t.Fatal("no inbound fixtures")
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var c inboundCase
		if err := json.Unmarshal(b, &c); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		t.Run(c.Name, func(t *testing.T) {
			msg, err := DecodeClientMessage(c.Message, c.Version)
			if len(c.WantPaths) == 0 && len(c.WantContains) == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				var got string
				switch {
				case msg.Action != nil && msg.Error == nil:
					got = msg.Action.String()
				case msg.Error != nil && msg.Action == nil:
					got = msg.Error.String()
				default:
					t.Fatalf("exactly one of Action and Error must be set: %+v", msg)
				}
				if got != c.WantString {
					t.Errorf("String() = %q, want %q", got, c.WantString)
				}
				if msg.Version != c.Message["version"] {
					t.Errorf("Version = %q", msg.Version)
				}
				return
			}
			var ve *kit.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("want *kit.ValidationError, got %v", err)
			}
			text := ve.Error()
			for _, p := range c.WantPaths {
				found := false
				for _, pr := range ve.Problems {
					if pr.Path == p {
						found = true
					}
				}
				if !found {
					t.Errorf("path %q not reported in:\n%s", p, text)
				}
			}
			for _, s := range c.WantContains {
				if !strings.Contains(text, s) {
					t.Errorf("%q not in:\n%s", s, text)
				}
			}
		})
	}
}

func TestDecodeClientMessageRawAndExtras(t *testing.T) {
	m := map[string]any{"version": "v0.9", "error": map[string]any{"code": "RENDER_FAILED", "surfaceId": "s1", "message": "m", "detail": map[string]any{"line": 3.0}}}
	msg, err := DecodeClientMessage(m, kit.V09)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Raw["error"].(map[string]any)["detail"] == nil {
		t.Error("Raw must carry the message as received, extras included")
	}
	if msg.Error.Code != "RENDER_FAILED" || msg.Error.SurfaceID != "s1" || msg.Error.Path != "" {
		t.Errorf("Error = %+v", msg.Error)
	}
}

func TestDecodeClientMessageUnsupportedVersion(t *testing.T) {
	m := map[string]any{"version": "v1.0", "action": map[string]any{}}
	_, err := DecodeClientMessage(m, "v1.0")
	var ve *kit.ValidationError
	if err == nil || errors.As(err, &ve) {
		t.Errorf("an unsupported pinned version is a plain error, got %v", err)
	}
}

func TestDecodeClientMessagesListPaths(t *testing.T) {
	ok := map[string]any{"version": "v0.9", "action": map[string]any{"name": "n", "surfaceId": "s", "sourceComponentId": "c", "timestamp": "2026-09-07T10:00:00Z", "context": map[string]any{}}}
	bad := map[string]any{"version": "v0.9", "action": map[string]any{"name": "n"}}
	msgs, err := DecodeClientMessages([]map[string]any{ok, ok}, kit.V09)
	if err != nil || len(msgs) != 2 {
		t.Fatalf("got %d messages, %v", len(msgs), err)
	}
	_, err = DecodeClientMessages([]map[string]any{ok, bad}, kit.V09)
	var ve *kit.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want *kit.ValidationError, got %v", err)
	}
	found := false
	for _, p := range ve.Problems {
		if p.Path == "messages[1].action" {
			found = true
		}
	}
	if !found {
		t.Errorf("list problems must carry messages[i] paths:\n%s", ve)
	}
	if _, err := DecodeClientMessages(nil, kit.V09); err != nil {
		t.Errorf("an empty list decodes to nothing, got %v", err)
	}
}

func TestClientStringsOnZeroValues(t *testing.T) {
	if got := (&Action{}).String(); got != `user action "" on surface "" from component "" with no context` {
		t.Errorf("Action zero value: %q", got)
	}
	if got := (&ClientError{}).String(); got != `renderer error (no code) on surface "": ` {
		t.Errorf("ClientError zero value: %q", got)
	}
}
