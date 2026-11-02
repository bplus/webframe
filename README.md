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
make ui-check   # Playwright measurement of /items and / at phone and desktop width; needs `make dev` running
```

The demo app runs on `http://localhost:3333`. `make ui-check` loads it in
Playwright's own Chromium (one-time `npx playwright install chromium`) at 390
and 1280 px, writes screenshots to `tools/shots/` (git-ignored) and fails on
horizontal overflow or console errors; see `tools/README.md`. It is not part
of `make check` because it needs the dev server.

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

The page layer (`appshell`, `pageheader`, `breadcrumbs`, `layout`, `panel`, `action`, `formfield`, `autocomplete`, and the `custompage`/`customblock` fences) lets an app build whole pages by composing components. `assets` embeds `static/` for serving. `wflint` enforces the rule below; `make check-ui` runs it and `make check` includes it. `tabs` takes a `Sizing` (default `SizingTallest`, so switching never moves the page), `layout.Section` and `panel` take a `Level` (h2 to h4) so nesting reads as nesting, and `hovercard` takes a `Placement` (default bottom-start).

Batch 2 (lists and forms) adds the three page types `listpage`, `detailpage` and `formpage`, each with a `Respond(w, r, title, props)`; the pieces they compose, `datatable`, `pagination`, `toolbar`, `descriptionlist`, `stattile`, `banner`, `toggle`, `image`, `figure` and `form.Form` (the `<form>` element itself; a raw one is forbidden); `badge.Status(kind)` so apps never pick status colors; and `listquery` at the module root, the sort/filter/page query-string contract (`sort`, `dir`, `page`, `per`, `q`, filter keys) that table headers, filter bars and pagination emit and handlers parse. A list page is `listquery.Parse` → declare `listpage.Props` → `listpage.Respond`; sort, filter and page links swap only the results region over htmx and push the URL. A form page re-renders with `Values` and `Errors` by field name on a failed POST (status 200, because htmx does not swap 4xx) and redirects on success (`HX-Redirect` for an htmx submit); `formfield.Field.Validate` wires blur validation to a handler whose body is `formfield.RespondFeedback`. The demo's Items app (`/items`) is the acceptance scenario.

**Changed defaults in batch 2:** `layout.Inline` wraps below `md` by default; a bare `?sort=key` without `dir` parses ascending; `appshell` adds `HX-Target` to `Vary`; `statcomparison`, `copybutton` and `form.Message` colors come from the theme tokens `text-success`/`text-destructive`/`text-info`; `avatar.Image` sanitizes `src`; new tokens `--color-success`, `--color-warning`, `--color-info`. wflint no longer allows a raw `<img>` (use `image`) and now allows the `download` attribute; the `html-from-go-string` rule only fires on a quoted attribute value or a tag with markup around it.

**Changed defaults in batch 1b:** `tabs` are now as tall as their tallest panel by default (set `Sizing: tabs.SizingFit` for the old grow-and-shrink behavior), hover cards open below by default (set `Placement` to choose), `layout.Section` Level 2 headings are `text-2xl` and the page title is `text-3xl`. Consumers pinned to v0.1.0 (one app uses tabs and hovercard) will see these changes when they upgrade.

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
