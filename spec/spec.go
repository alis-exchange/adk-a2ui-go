// Package spec embeds the official A2UI specification files (JSON schemas and the basic
// catalog) verbatim, per major version directory, so validation runs against exactly what
// upstream publishes. Refresh with scripts/sync-spec.sh; [Source] records the upstream commit.
package spec

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"
)

// Major version directories inside the embedded tree.
const (
	MajorV09 = "v0_9" // serves wire versions "v0.9" and "v0.9.1"
	MajorV10 = "v1_0" // serves wire version "v1.0"
)

//go:embed SOURCE v0_9/json/*.json v0_9/catalogs/basic/catalog.json v0_9/catalogs/basic/rules.txt v1_0/json/*.json v1_0/catalogs/basic/catalog.json
var files embed.FS

// FS returns the embedded specification tree, rooted at the directory holding SOURCE, v0_9 and
// v1_0. It is a function rather than an exported variable so callers cannot swap the embedded
// files out from under the validators; read from it with [io/fs.ReadFile].
func FS() fs.FS { return files }

// source is the upstream commit the embedded files were copied from ("<short-sha> <date>").
var source = func() string {
	b, _ := files.ReadFile("SOURCE")
	return strings.TrimSpace(string(b))
}()

// Source returns the upstream commit the embedded files were copied from
// ("<short-sha> <date>"), as recorded by scripts/sync-spec.sh in spec/SOURCE.
func Source() string { return source }

// MajorFor maps a wire version to its embedded directory.
func MajorFor(version string) (string, bool) {
	switch version {
	case "v0.9", "v0.9.1":
		return MajorV09, true
	case "v1.0":
		return MajorV10, true
	}
	return "", false
}

// basicCatalogIDs lists, per major, the canonical basic catalog id first, then aliases the
// spec documents use for the same file (v0.9.1 docs refer to a v0_9_1 URL; the file is identical).
var basicCatalogIDs = map[string][]string{
	MajorV09: {
		"https://a2ui.org/specification/v0_9/catalogs/basic/catalog.json",
		"https://a2ui.org/specification/v0_9_1/catalogs/basic/catalog.json",
	},
	MajorV10: {"https://a2ui.org/specification/v1_0/catalogs/basic/catalog.json"},
}

// BasicCatalogIDs returns every id that resolves to the embedded basic catalog of major.
func BasicCatalogIDs(major string) []string {
	return append([]string(nil), basicCatalogIDs[major]...)
}

// BasicCatalog returns the embedded basic catalog for major, its canonical id, and the prompt
// guidance shipped with it: rules.txt for v0_9, the catalog's "instructions" field for v1_0.
func BasicCatalog(major string) (catalog map[string]any, catalogID, instructions string, err error) {
	b, err := files.ReadFile(major + "/catalogs/basic/catalog.json")
	if err != nil {
		return nil, "", "", fmt.Errorf("spec: no basic catalog for %q: %w", major, err)
	}
	if err := json.Unmarshal(b, &catalog); err != nil {
		return nil, "", "", fmt.Errorf("spec: basic catalog for %q: %w", major, err)
	}
	catalogID, _ = catalog["catalogId"].(string)
	var parts []string
	if rules, err := files.ReadFile(major + "/catalogs/basic/rules.txt"); err == nil {
		parts = append(parts, strings.TrimSpace(string(rules)))
	}
	if s, ok := catalog["instructions"].(string); ok && strings.TrimSpace(s) != "" {
		parts = append(parts, strings.TrimSpace(s))
	}
	return catalog, catalogID, strings.Join(parts, "\n\n"), nil
}
