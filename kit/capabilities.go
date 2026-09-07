package kit

import (
	"context"
	"errors"
	"fmt"
)

// Wire version identifiers as they appear in the "version" field of every A2UI message and
// as keys of a capabilities document.
const (
	V09  = "v0.9"
	V091 = "v0.9.1"
	V10  = "v1.0"
)

// KnownVersions lists every version this module implements, newest first. It is the default
// negotiation preference.
var KnownVersions = []string{V10, V091, V09}

// VersionParams is the per-version object of a capabilities document
// (client_capabilities.json for v0.9.x, renderer_capabilities.json for v1.0).
//
// InlineCatalogs holds the catalog documents themselves, not copies: parsing keeps the maps the
// caller's document already contains, and validation and the catalog tool read them on every
// turn. Do not mutate a catalog after handing it to [WithA2UICapabilities] or
// [WithCapabilities]; register a document you intend to change with a [Registry] instead, which
// stores a deep copy.
type VersionParams struct {
	SupportedCatalogIDs []string
	InlineCatalogs      []map[string]any
}

// Capabilities is a capabilities document keyed by wire version, e.g. {"v0.9": {...}}.
type Capabilities map[string]VersionParams

type capabilitiesKey struct{}

// WithA2UICapabilities stores the version-keyed object a transport receives under
// a2uiClientCapabilities (v0.9.x) or a2uiRendererCapabilities (v1.0). A legacy flat map with
// supportedCatalogIds at the top level is treated as {"v0.9": doc}; see [ParseCapabilities] for
// exactly when that wrap applies. Malformed versions are dropped; use ParseCapabilities first to
// see the errors.
//
// The document is not copied: see [VersionParams] on not mutating inline catalogs afterwards.
func WithA2UICapabilities(ctx context.Context, doc map[string]any) context.Context {
	caps, _ := ParseCapabilities(doc)
	if caps == nil {
		caps = Capabilities{}
	}
	return WithCapabilities(ctx, caps)
}

// WithCapabilities stores an already-parsed document.
func WithCapabilities(ctx context.Context, caps Capabilities) context.Context {
	return context.WithValue(ctx, capabilitiesKey{}, caps)
}

// CapabilitiesFromContext returns the stored document and whether one was stored.
func CapabilitiesFromContext(ctx context.Context) (Capabilities, bool) {
	caps, ok := ctx.Value(capabilitiesKey{}).(Capabilities)
	return caps, ok
}

// ParseCapabilities converts a raw document into Capabilities. A document is treated as the
// legacy flat shape (wrapped as {"v0.9": doc}) only when it has a top-level supportedCatalogIds
// and no top-level key that looks like a version (one starting with "v" followed by a digit).
// A top-level supportedCatalogIds is therefore the sole marker of the legacy shape: a legacy
// document carrying only inlineCatalogs is not recognised as one, and is parsed as version-keyed
// (so "inlineCatalogs" is reported as a stray top-level key rather than becoming v0.9 params).
// Otherwise the document is parsed as version-keyed, and any top-level
// supportedCatalogIds/inlineCatalogs are skipped and reported in the joined error instead of
// being silently dropped. Versions whose value is not an object, or whose supportedCatalogIds
// contains non-strings, are also omitted and reported in the joined error; well-formed versions
// are still returned.
//
// Inline catalogs are carried over by reference; see [VersionParams].
func ParseCapabilities(doc map[string]any) (Capabilities, error) {
	if doc == nil {
		return nil, errors.New("kit: nil capabilities document")
	}
	if _, legacy := doc["supportedCatalogIds"]; legacy && !hasVersionLikeKey(doc) {
		doc = map[string]any{V09: doc}
	}
	caps := Capabilities{}
	var errs []error
	for version, raw := range doc {
		if version == "supportedCatalogIds" || version == "inlineCatalogs" {
			errs = append(errs, fmt.Errorf("kit: capabilities document mixes a legacy flat shape with version keys; top-level %q ignored", version))
			continue
		}
		obj, ok := raw.(map[string]any)
		if !ok {
			errs = append(errs, fmt.Errorf("kit: capabilities[%q]: got %T, want object", version, raw))
			continue
		}
		params, err := parseParams(obj)
		if err != nil {
			errs = append(errs, fmt.Errorf("kit: capabilities[%q]: %w", version, err))
			continue
		}
		caps[version] = params
	}
	return caps, errors.Join(errs...)
}

// hasVersionLikeKey reports whether doc has a top-level key that looks like a wire version, i.e.
// one starting with "v" followed by a digit (e.g. "v0.9", "v1.0").
func hasVersionLikeKey(doc map[string]any) bool {
	for key := range doc {
		if len(key) >= 2 && key[0] == 'v' && key[1] >= '0' && key[1] <= '9' {
			return true
		}
	}
	return false
}

func parseParams(obj map[string]any) (VersionParams, error) {
	var p VersionParams
	switch ids := obj["supportedCatalogIds"].(type) {
	case nil:
	case []string:
		p.SupportedCatalogIDs = append([]string(nil), ids...)
	case []any:
		for i, v := range ids {
			s, ok := v.(string)
			if !ok {
				return p, fmt.Errorf("supportedCatalogIds[%d]: got %T, want string", i, v)
			}
			p.SupportedCatalogIDs = append(p.SupportedCatalogIDs, s)
		}
	default:
		return p, fmt.Errorf("supportedCatalogIds: got %T, want array", ids)
	}
	switch inline := obj["inlineCatalogs"].(type) {
	case nil:
	case []map[string]any:
		p.InlineCatalogs = append([]map[string]any(nil), inline...)
	case []any:
		for i, v := range inline {
			m, ok := v.(map[string]any)
			if !ok {
				return p, fmt.Errorf("inlineCatalogs[%d]: got %T, want object", i, v)
			}
			p.InlineCatalogs = append(p.InlineCatalogs, m)
		}
	default:
		return p, fmt.Errorf("inlineCatalogs: got %T, want array", inline)
	}
	return p, nil
}

// Negotiate returns the first version in preferred (default KnownVersions) that caps contains.
func Negotiate(caps Capabilities, preferred []string) (string, VersionParams, bool) {
	if preferred == nil {
		preferred = KnownVersions
	}
	for _, v := range preferred {
		if p, ok := caps[v]; ok {
			return v, p, true
		}
	}
	return "", VersionParams{}, false
}
