package schema

import (
	"bytes"
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.alis.build/adk/a2ui/spec"
)

// Entry schema files, relative to spec/<major>/json. Outbound entries are message lists
// (the tools validate a whole batch); inbound entries are single messages (a transport hands
// the agent one renderer message at a time, and the decoders add list handling themselves).
const (
	EntryOutboundV09 = "server_to_client_list.json"
	EntryOutboundV10 = "agent_to_renderer_list.json"
	EntryInboundV09  = "client_to_server.json"
	EntryInboundV10  = "renderer_to_agent.json"
)

// Catalog-relative references the spec uses; CompileRef wraps one of these.
const (
	RefAnyComponent = "catalog.json#/$defs/anyComponent"
	RefTheme        = "catalog.json#/$defs/theme"
)

const baseURL = "https://a2ui.org/specification/"

// CompileOptions selects an entry schema and the catalog to inject as catalog.json.
type CompileOptions struct {
	Entry   string         // one of the Entry* constants
	Catalog map[string]any // nil injects the permissive stub catalog
	V091    bool           // v0_9 only: accept "v0.9.1" as well as "v0.9"
}

// maxCachedSchemas bounds the compiled schemas each engine keeps. Cache keys include a digest
// of the catalog, and inline catalogs arrive from clients, so without a bound every distinct
// client catalog would stay compiled for the life of the process.
const maxCachedSchemas = 64

// Engine compiles schemas for one major version and keeps the most recently used ones.
type Engine struct {
	major string
	mu    sync.Mutex
	cache map[string]*list.Element // key -> element holding a cacheEntry
	order *list.List               // front is the most recently used
}

type cacheEntry struct {
	key    string
	schema *jsonschema.Schema
}

func newEngine(major string) *Engine {
	return &Engine{major: major, cache: map[string]*list.Element{}, order: list.New()}
}

var engines = map[string]*Engine{
	spec.MajorV09: newEngine(spec.MajorV09),
	spec.MajorV10: newEngine(spec.MajorV10),
}

// lookup returns the cached schema for key and marks it most recently used.
func (e *Engine) lookup(key string) (*jsonschema.Schema, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	el, ok := e.cache[key]
	if !ok {
		return nil, false
	}
	e.order.MoveToFront(el)
	return el.Value.(cacheEntry).schema, true
}

// store keeps s under key unless another goroutine stored one first (the same
// load-or-store rule sync.Map gave us: both callers end up with one pointer), then evicts the
// least recently used entries past maxCachedSchemas. It returns the schema now held for key.
func (e *Engine) store(key string, s *jsonschema.Schema) *jsonschema.Schema {
	e.mu.Lock()
	defer e.mu.Unlock()
	if el, ok := e.cache[key]; ok {
		e.order.MoveToFront(el)
		return el.Value.(cacheEntry).schema
	}
	e.cache[key] = e.order.PushFront(cacheEntry{key: key, schema: s})
	for e.order.Len() > maxCachedSchemas {
		oldest := e.order.Back()
		e.order.Remove(oldest)
		delete(e.cache, oldest.Value.(cacheEntry).key)
	}
	return s
}

// For returns the engine for an embedded major directory (spec.MajorV09 or spec.MajorV10).
func For(major string) *Engine { return engines[major] }

// Compile returns the compiled entry schema, injecting opts.Catalog (or the stub) as catalog.json.
func (e *Engine) Compile(opts CompileOptions) (*jsonschema.Schema, error) {
	key := "entry|" + opts.Entry + "|" + fmt.Sprint(opts.V091) + "|" + catalogKey(opts.Catalog)
	return e.compile(key, baseURL+e.major+"/"+opts.Entry, nil, opts)
}

// CompileRef compiles {"$ref": ref} against the given catalog, so one component, function call,
// or theme can be validated on its own.
func (e *Engine) CompileRef(ref string, catalog map[string]any, v091 bool) (*jsonschema.Schema, error) {
	key := "ref|" + ref + "|" + fmt.Sprint(v091) + "|" + catalogKey(catalog)
	loc := baseURL + e.major + "/wrapper.json"
	wrapper := map[string]any{"$schema": "https://json-schema.org/draft/2020-12/schema", "$ref": ref}
	return e.compile(key, loc, wrapper, CompileOptions{Catalog: catalog, V091: v091})
}

func (e *Engine) compile(key, loc string, resource map[string]any, opts CompileOptions) (*jsonschema.Schema, error) {
	if s, ok := e.lookup(key); ok {
		return s, nil
	}
	// Compile outside the lock: it is slow and pure, and store reconciles a concurrent compile
	// of the same key.
	c := jsonschema.NewCompiler()
	c.UseLoader(&loader{major: e.major, catalog: opts.Catalog, v091: opts.V091})
	c.UseRegexpEngine(goRegexp)
	// Formats are annotations by default in draft 2020-12. The spec relies on them being
	// asserted: DateTimeInput's min/max are "if string then oneOf[format date, time,
	// date-time]", which every string satisfies without assertion, so oneOf always failed.
	c.AssertFormat()
	if resource != nil {
		doc, err := normalize(resource)
		if err != nil {
			return nil, err
		}
		if err := c.AddResource(loc, doc); err != nil {
			return nil, fmt.Errorf("schema: add %s: %w", loc, err)
		}
	}
	s, err := c.Compile(loc)
	if err != nil {
		return nil, fmt.Errorf("schema: compile %s: %w", loc, err)
	}
	return e.store(key, s), nil
}

// ToInstance converts tool input into the value shape the validator accepts.
func ToInstance(messages []map[string]any) []any {
	out := make([]any, len(messages))
	for i, m := range messages {
		out[i] = m
	}
	return out
}

// loader answers every $ref by the URL's basename: the embedded file of that name, or the
// injected catalog for catalog.json. Matching on basename tolerates upstream's inconsistent $id
// paths (v0_9_1 ids still say v0_9; v1_0 list schemas carry /json/, message schemas do not).
type loader struct {
	major   string
	catalog map[string]any
	v091    bool
}

func (l *loader) Load(u string) (any, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	base := path.Base(parsed.Path)
	if base == "catalog.json" {
		if l.catalog == nil {
			return normalize(stubCatalog())
		}
		return normalize(withUnions(l.major, l.catalog))
	}
	b, err := fs.ReadFile(spec.FS(), l.major+"/json/"+base)
	if err != nil {
		return nil, fmt.Errorf("schema: no embedded schema for %s: %w", u, err)
	}
	if l.v091 {
		// Upstream's entire v0.9 -> v0.9.1 delta, applied in memory (sync-spec.sh guards this).
		b = bytes.ReplaceAll(b, []byte(`"const": "v0.9"`), []byte(`"enum": ["v0.9", "v0.9.1"]`))
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(b))
}

// normalize round-trips a Go map through JSON so numbers become json.Number, which is what the
// compiler expects for schema keywords like minimum.
func normalize(doc map[string]any) (any, error) {
	b, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(b))
}

// stubCatalog stands in when the real catalog is unknown: it keeps the envelope rules and
// requires id and component, but lets any component name and any properties through.
// additionalProperties:true matters for v1.0, whose Component schema uses unevaluatedProperties.
// It also supplies the per-union fallbacks withUnions uses for a catalog that declares nothing
// for that union.
func stubCatalog() map[string]any {
	return map[string]any{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"$defs": map[string]any{
			"anyComponent": map[string]any{
				"type":     "object",
				"required": []any{"id", "component"},
				"properties": map[string]any{
					"id":        map[string]any{"type": "string"},
					"component": map[string]any{"type": "string"},
				},
				"additionalProperties": true,
			},
			"anyFunction": map[string]any{
				"type":                 "object",
				"required":             []any{"call"},
				"properties":           map[string]any{"call": map[string]any{"type": "string"}},
				"additionalProperties": true,
			},
			"theme": map[string]any{"type": "object"},
		},
	}
}

// unmarshalable counts catalogs that could not be marshalled, so each one gets its own cache
// key. Two distinct catalogs that both fail to marshal must never share a compiled schema.
var unmarshalable atomic.Uint64

// catalogKey is the cache key for a catalog document: its id plus a digest of its canonical JSON.
// A catalog that cannot be marshalled (a non-JSON value somewhere inside it) has no canonical
// form, so it gets a fresh, never-repeated key instead: compilation still happens and still
// fails loudly if the document is unusable, but no two failing catalogs can collide on one entry.
func catalogKey(c map[string]any) string {
	if c == nil {
		return "stub"
	}
	id, _ := c["catalogId"].(string)
	b, err := json.Marshal(c) // encoding/json sorts map keys, so this is canonical
	if err != nil {
		return fmt.Sprintf("%s!marshal-error:%d", id, unmarshalable.Add(1))
	}
	sum := sha256.Sum256(b)
	return id + ":" + hex.EncodeToString(sum[:8])
}

// goRegexp approximates the UAX #31 identifier classes v1.0 uses for extension keys, which
// Go's RE2 engine lacks, with the closest Unicode general categories.
var xidReplacer = strings.NewReplacer(
	`\p{XID_Start}`, `\p{L}\p{Nl}`,
	`\p{XID_Continue}`, `\p{L}\p{Nl}\p{Mn}\p{Mc}\p{Nd}\p{Pc}`,
)

func goRegexp(s string) (jsonschema.Regexp, error) {
	return regexp.Compile(xidReplacer.Replace(s))
}
