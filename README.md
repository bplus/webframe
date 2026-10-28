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
make check-ui
```

The demo app runs on `http://localhost:3333`.

Run `make check` before pushing. It composes four sub-targets, in order:

- `check-generate` — regenerates templ output and Tailwind's `static/css/output.css`,
  then fails on drift in either
- `check-vet` — `go vet ./...`
- `check-test` — `go test ./...`
- `check-ui` — `go run ./cmd/wflint ./demo`, the UI lint (see Page layer)

The Tailwind CLI version is pinned via `@tailwindcss/cli` in `package.json`/
`package-lock.json` (currently 4.3.3) so `npx @tailwindcss/cli` produces a
byte-stable `output.css` instead of floating to whatever is latest.

## Page layer

The page layer (`appshell`, `pageheader`, `breadcrumbs`, `layout`, `panel`, `action`, `formfield`, `autocomplete`, and the `custompage`/`customblock` fences) lets an app build whole pages by composing components. `assets` embeds `static/` for serving. `wflint` enforces the rule below; `make check-ui` runs it and `make check` includes it. `tabs` take a `Sizing` (default `SizingTallest`, so switching never moves the page), `layout.Section` and `panel` take a `Level` (h2 to h4) so nesting reads as nesting, and `hovercard` takes a `Placement` (default bottom-start).

App code never contains anything below the component boundary (no div, no class, no layout decision); composition means choosing and ordering components. The only place that boundary may be crossed is inside a CustomPage or CustomBlock, which must carry a written reason and is printed by every build. A missing component is never a reason to use the fence: stop and file it against webframe.

```go
var App = &appshell.App{Name: "Demo", HomeHref: "/"}
http.ListenAndServe(addr, App.Middleware(mux))
// in a handler:
err := App.Render(w, r, "Showcase", pages.Showcase())
```

Consumers: several server-rendered Go apps.

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
