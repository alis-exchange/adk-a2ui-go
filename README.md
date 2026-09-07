# A2UI Go — ADK Tools

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

Transport-agnostic Go libraries for the **[A2UI](https://a2ui.org/)** (Agent-to-UI) protocol, built for the Google **[ADK](https://google.github.io/adk-docs/)** (Agent Development Kit). Works with any transport layer — AG-UI SSE, A2A, or custom integrations.

## Features

- **ADK tools** (`v09/tools`) — `generate_a2ui_messages` [function tool](https://pkg.go.dev/google.golang.org/adk/v2/tool/functiontool) with specification derived from the [A2UI v0.9 server-to-client message list](https://a2ui.org/specification/v0.9-a2ui/), plus semantic checks (e.g. `id: "root"` per surface).
- **Capabilities** (`kit`) — Version-agnostic helpers to store and retrieve A2UI client capabilities on `context.Context`. Transport adapters call `kit.WithA2UICapabilities` after extracting capabilities from their transport-specific format; versioned tools read them back via `kit.CapabilitiesFromContext`.

## Packages

The module is organized by A2UI spec version so multiple versions can coexist. Version-agnostic code lives at the top level.

| Import path                       | Role                                                          |
| --------------------------------- | ------------------------------------------------------------- |
| `go.alis.build/adk/a2ui`          | Root package (documentation only; see `docs.go`).             |
| `go.alis.build/adk/a2ui/kit`      | Version-agnostic context keys and catalog parsing.            |
| `go.alis.build/adk/a2ui/v09/tools`| A2UI v0.9 ADK tools, specification (https://a2ui.org/specification/v0.9-a2ui/), and semantic validation.    |

## Architecture

1. **Transport adapter** — Each transport (A2A, AG-UI, etc.) extracts A2UI capabilities from its own format and calls [`kit.WithA2UICapabilities`](kit/capabilities.go) to store them on the Go context.
2. **Toolset visibility** — [`v09/tools.NewA2UIToolset`](v09/tools/tool.go) uses [`kit.CapabilitiesFromContext`](kit/capabilities.go) to filter tool visibility — A2UI tools are only exposed when capabilities are present.
3. **Catalog discovery** — The model calls `a2ui_catalog` to retrieve supported catalog IDs and inline schemas from the capabilities.
4. **Generation** — The model calls `generate_a2ui_messages` with a `messages` array; validation runs against the v0.9 schema in [`v09/tools/schema.go`](v09/tools/schema.go).

```
┌─────────────┐     ┌──────────────────┐     ┌──────────────┐
│  Transport   │────▶│ kit.WithA2UI-    │────▶│ ADK Agent +  │
│  Adapter     │     │ Capabilities()   │     │ v09 Toolset  │
│ (A2A/AG-UI)  │     └──────────────────┘     └──────┬───────┘
└─────────────┘                                      │
                                                     ▼
                                              ┌──────────────┐
                                              │ a2ui_catalog  │
                                              │ generate_a2ui │
                                              └──────────────┘
```

## Installation

```bash
go get go.alis.build/adk/a2ui@latest
```

### Version compatibility

| a2ui-go | ADK Go module              | Go      |
| ------- | -------------------------- | ------- |
| `v1.x`  | `google.golang.org/adk/v2` | 1.26.6+ |
| `v0.x`  | `google.golang.org/adk`    | 1.26+   |

The two ADK majors define different `tool.Toolset` and `agent.Context` types, so an agent must use the a2ui-go line that matches its ADK module. Agents still on ADK Go v1 should pin `v0.x`; that line is maintained on the [`v0` branch](https://github.com/alis-exchange/adk-a2ui-go/tree/v0).

## Getting started

### 1. Store capabilities from your transport

```go
import "go.alis.build/adk/a2ui/kit"

// In your transport adapter (A2A interceptor, AG-UI handler, etc.)
capabilities := map[string]any{
    "supportedCatalogIds": []any{"https://example.com/catalog.json"},
    "inlineCatalogs":      []any{catalogSchemaMap},
}
ctx = kit.WithA2UICapabilities(ctx, capabilities)
```

### 2. Add the v0.9 toolset to your agent

```go
import tools "go.alis.build/adk/a2ui/v09/tools"

toolset, err := tools.NewA2UIToolset()
// Add to your ADK agent's Toolsets
```

The toolset is automatically filtered — tools only appear when `kit.CapabilitiesFromContext` finds capabilities on the context.

## Documentation

- **Go doc comments** — Browse locally:

  ```bash
  go doc go.alis.build/adk/a2ui/...
  ```

- **Package overviews** — See `docs.go` at the repository root and under `kit/` and `v09/tools/` for package-level narratives.
- **Specification** — A2UI message shapes and semantics are defined by [A2UI](https://a2ui.org/) (v0.9 server-to-client list schema).

## License

Apache 2.0 — see [LICENSE](LICENSE).
