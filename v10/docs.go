// Package v10 implements A2UI specification version 1.0 (https://a2ui.org/specification/v1.0-a2ui/).
//
// [Validate] checks an agent-to-renderer message batch against the official schema embedded in
// [go.alis.build/adk/a2ui/spec], the catalog each component or function call names, and the
// prose rules of the spec. The ADK tools live in [go.alis.build/adk/a2ui/v10/tools].
//
// v1.0 is marked a release candidate upstream; re-run scripts/sync-spec.sh when it tags.
package v10
