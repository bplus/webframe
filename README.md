# Webframe

Webframe is a Go-first component library for building server-rendered web UIs with `templ`, Tailwind v4, HTMX, and Alpine.

## What it contains

- Reusable UI components under `components/`
- Demo app under `cmd/demo` and `demo/`
- Prebuilt static assets under `static/`
- Tailwind and `templ` build steps via `Makefile`

## Development

Requirements:

- Go
- `templ`
- Node/npm for Tailwind CSS

Common commands:

```bash
make build
make dev
make generate
make css
```

The demo app runs on `http://localhost:3333`.

## Conventions

- Keep components additive and avoid breaking existing component contracts
- Each component should live in `components/<name>/`
- Prefer a small `Props` struct and a `Script()` templ when JS is required
- Add focused render tests for new components
- Commit generated `*_templ.go` output when templ files change

## New component: `trafficstream`

`trafficstream` is intended for operator/admin surfaces that need to watch live request and response activity.

Features:

- SSE-backed live feed
- Client-side filtering by text, provider, event, and direction
- Pause/resume and clear controls
- Bounded in-browser row retention
- Works well for `/admin/traffic` style pages

Minimal usage:

```go
@trafficstream.TrafficStream(trafficstream.Props{
    StreamURL: "/api/traffic/stream",
    Height: 560,
    ProviderOptions: []string{"anthropic", "openai", "gemini"},
})
@trafficstream.Script()
```
