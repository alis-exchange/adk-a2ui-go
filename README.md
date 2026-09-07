# A2UI Go — ADK Tools

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Transport-agnostic Go library for the **[A2UI](https://a2ui.org/)** (Agent-to-UI) protocol on the Google **[ADK](https://google.github.io/adk-docs/)** for Go v2. Works with any transport (A2A, AG-UI, custom).

## What it does

- **One negotiated toolset.** `a2ui.NewToolset()` reads the client's capabilities each turn, picks the newest A2UI version both sides support (**v1.0**, **v0.9.1**, or **v0.9**), and exposes `a2ui_catalog` and `generate_a2ui_messages` for it.
- **Official-schema validation.** The upstream JSON schemas and basic catalogs are embedded verbatim (`spec/`, pinned to an upstream commit in `spec/SOURCE`). Every generated message is validated against them **and against the negotiated catalog**, so a missing required property or a misspelled prop is caught before it reaches the renderer.
- **Catalog resolution.** Catalogs come from the client's inline catalogs, from catalogs you register, or from the built-in basic catalog. Unknown catalogs fall back to envelope checks unless you opt into strict mode. The library never fetches over the network.
- **Model-friendly errors.** Validation failures come back as a `*kit.ValidationError`: a short problem list with JSON paths, rendered as the tool error the model fixes and retries. Anything else the tools return is labelled an agent-side configuration error, so the model does not retry a payload that was never the problem.

## Installation

```bash
go get go.alis.build/adk/a2ui@latest
```

### Version compatibility

| a2ui-go | ADK Go module              | Go      | A2UI spec versions   |
| ------- | --------------------------- | ------- | --------------------- |
| `v1.x`  | `google.golang.org/adk/v2` | 1.26.6+ | v0.9, v0.9.1, v1.0   |
| `v0.x`  | `google.golang.org/adk`    | 1.26+   | v0.9                 |

The two ADK majors define different `tool.Toolset` and `agent.Context` types, so an agent must use the a2ui-go line that matches its ADK module. ADK Go v1 users stay on the [`v0` branch](https://github.com/alis-exchange/adk-a2ui-go/tree/v0).

## Getting started

### 1. Store the client's capabilities from your transport

```go
import "go.alis.build/adk/a2ui/kit"

// A2A: message.metadata["a2uiClientCapabilities"] (v0.9.x) or ["a2uiRendererCapabilities"] (v1.0).
// The value is keyed by version, e.g. {"v1.0": {"supportedCatalogIds": [...], "inlineCatalogs": [...]}}.
ctx = kit.WithA2UICapabilities(ctx, capabilitiesObject)
```

### 2. Add the toolset to your agent

```go
import "go.alis.build/adk/a2ui"

toolset, err := a2ui.NewToolset()
// Add to your ADK agent's Toolsets. Tools appear only when capabilities are present.
```

### 3. Register your renderer's catalog (optional)

```go
registry := kit.NewRegistry()
if err := registry.RegisterJSON(myCatalogJSON); err != nil { ... }

toolset, err := a2ui.NewToolset(
    a2ui.WithCatalogResolver(registry),
    a2ui.WithStrictCatalogs(), // unknown catalogId becomes a validation error
)
```

Strict mode only covers surfaces the batch itself creates. A message that updates a surface created in an earlier turn has no `catalogId` in this batch, so its components are checked against the envelope alone and no "catalog not available" problem is raised — the surface may legitimately pre-exist.

### Advertise what the agent supports

```go
import "go.alis.build/adk/a2ui/v10"

params := kit.AgentCapabilities(kit.V10, []string{v10.CatalogIDBasic}, true)
// Put params in your A2A AgentCard extension / transport metadata.
```

## Packages

| Import path                        | Role                                                          |
| ----------------------------------- | -------------------------------------------------------------- |
| `go.alis.build/adk/a2ui`           | `NewToolset` with version negotiation and options.            |
| `go.alis.build/adk/a2ui/kit`       | Capabilities, `Negotiate`, catalog resolvers, options.        |
| `go.alis.build/adk/a2ui/spec`      | Embedded upstream schemas and basic catalogs.                 |
| `go.alis.build/adk/a2ui/v09`       | `Validate` for v0.9 and v0.9.1 (framework-independent).       |
| `go.alis.build/adk/a2ui/v09/tools` | ADK tools for v0.9 and v0.9.1.                                |
| `go.alis.build/adk/a2ui/v10`       | `Validate` for v1.0.                                          |
| `go.alis.build/adk/a2ui/v10/tools` | ADK tools for v1.0.                                           |

## How a turn flows

1. The transport adapter stores capabilities on the context.
2. `a2ui.NewToolset` negotiates the version and exposes two tools.
3. The model calls `a2ui_catalog`, which returns every supported catalog document it can resolve plus authoring instructions.
4. The model calls `generate_a2ui_messages`. The batch is validated against the official schema (envelope), then each component against its catalog, then the spec's prose rules (at least one `root` component per created surface, ids unique within a list, no duplicate surfaces, v1.0 catalogId fallback). Problems come back as a tool error; success echoes the messages for the transport to forward.

## Migrating from v0.x

The `v1` line is a rewrite around version negotiation. What changed for consumers:

- `kit.CapabilitiesFromContext` returns a `kit.Capabilities` (a parsed, version-keyed map), not a raw document.
- `kit.GetCatalogs` is gone. Read `kit.VersionParams` from `kit.ParseCapabilities` or from `kit.Negotiate` instead.
- `v09/tools.NewA2UIToolset` is gone. Use `a2ui.NewToolset(...)` for negotiation across versions, or `v09/tools.NewToolset(version, opts)` to pin one version.
- The catalog tool's output shape is now `{version, catalog_ids, catalogs, unresolved, instructions}`.
- Validation errors are `*kit.ValidationError`; match them with `errors.As` and read `.Problems`.

## Updating the embedded spec

```bash
scripts/sync-spec.sh /path/to/A2UI   # copies json/, basic catalogs, and examples; writes spec/SOURCE
go test ./...                        # every official example must validate
```

## License

Apache 2.0 — see [LICENSE](LICENSE).
