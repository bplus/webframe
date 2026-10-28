# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Webframe is a Go module (`github.com/bplus/webframe`) — a component library for server-rendered web UIs. It's consumed as a dependency by other Go projects. It is **not** a standalone application; the demo app exists only for development/showcase purposes.

## Commands

```bash
make build          # templ generate → tailwind compile → go build ./cmd/demo
make generate       # templ generate (produces *_templ.go from *.templ)
make css            # npx @tailwindcss/cli -i static/css/input.css -o static/css/output.css
make dev            # go run ./cmd/demo (serves on :3333)
make check          # check-generate → check-vet → check-test → check-ui
make check-generate # templ generate → css rebuild → drift check on *_templ.go and output.css
make check-vet      # go vet ./...
make check-test     # go test ./...
make check-ui       # go run ./cmd/wflint ./demo (UI lint: app code may only compose components)
go test ./...       # run all tests
go test ./components/copybutton/...  # single package
```

Build order matters: templ generate must run before CSS compilation or Go compilation.

## Architecture

**Component pattern**: each component lives in `components/<name>/` containing:
- `<name>.templ` — template with a `Props` struct and templ component function
- `*_templ.go` — generated Go (committed to git, do not hand-edit)
- Optional: `helpers.go`, `*_test.go`

**Props convention**: components accept a Props struct with `ID`, `Class`, `Attributes templ.Attributes` fields plus component-specific fields. Variadic `...Props` with zero-value defaults is common.

**Key utilities** (`utils/`):
- `TwMerge(classes...)` — always use this to combine Tailwind classes, never concatenate
- `EscapeJS(string)` — use for values injected into JS string literals
- `SanitizeColor(string)` — use for user-provided CSS colors
- `If[T]` / `IfElse[T]` — conditional helpers for templates
- `ScriptURL(path)` — appends cache-busting `?v=` timestamp

**JS integration**: Alpine.js for client-side reactivity (inline `x-data`/`x-on`), HTMX for server-driven interactions (`hx-*` attributes). A global `htmx:afterSwap` listener reinitializes Alpine on dynamic content.

**CSS**: Tailwind v4 with custom theme tokens in `static/css/input.css`. Light (Observatory) is the default palette; dark is an opt-in stored preference.

**Demo app** (`cmd/demo/` + `demo/`): HTTP server on :3333 with showcase gallery, traffic stream page and a sidebar-layout page (`/sidebar`, rendered through a second `appshell.App`). Routes include HTMX interaction demos (click counter, search, validation, toasts) whose handlers render templ fragments in `demo/pages/fragments.templ`.

### Page layer

Apps are built by composing components, never by writing markup. Packages:

- `components/action` — `Action` type shared by page headers, sections, panels and the user menu
- `components/appshell` — `Document`, `Shell`, `App`, `Middleware`, `Render`: the page chrome and the per-request wiring
- `components/breadcrumbs`, `components/pageheader` — trail and page title/actions block
- `components/layout` — `Stack`, `Inline`, `Section`, `Divider`, `TwoColumn`, `ThreeColumn`; `Section` takes `Level` (h2/h3/h4)
- `components/panel`, `components/swatch` — framed content block (also takes `Level`); color swatch
- `components/formfield` — `Field`, `FormField`, `Feedback(id, err, success)`, `Fieldset`: labelled control with help and validation feedback
- `components/autocomplete` — `Autocomplete`, `Results`, `Script`: HTMX type-ahead picker; first-party `static/js/autocomplete.js`, registered in `appshell.Scripts`
- `components/custompage`, `components/customblock` — the declared design-exemption fences (see below)
- `gameicon` has `Size`/`Tone`; `avatar` has `Size`; `tabs` has `Sizing` (`SizingTallest` default, `SizingFixed` + `Height` S/M/L, `SizingFit`); `hovercard` has `Placement` (default bottom-start)

App pattern: configure one `appshell.App`, wrap the mux in `App.Middleware`, render each page with `App.Render`:

```go
var App = &appshell.App{Name: "Demo", HomeHref: "/"}
http.ListenAndServe(addr, App.Middleware(mux))
err := App.Render(w, r, "Showcase", pages.Showcase()) // body only for a plain htmx request; full page otherwise, including boosted links and history restores
```

**`assets`** embeds `static/`: `assets.Mount(mux)` serves it, `assets.Handler()` returns the handler, `assets.URL(path)` builds a URL.

**`wflint`** (`internal/wflint`, run via `cmd/wflint`) lints app code (`.templ`, Go, CSS) and rejects raw block HTML, `class`, `style`, `<style>`, `Class:` in Props literals, class/style attribute maps, `templ.Raw`, Go string literals holding HTML markup (`html-from-go-string`), and Alpine class/style bindings. Only text-level elements and component calls are allowed. The sole exemption is a `CustomPage` or `CustomBlock` fence, which needs a written reason; the lint prints every fence on every run. Flags: `-json` for a JSON report; exit 1 on violations, 2 on bad usage. `make check-ui` runs `go run ./cmd/wflint ./demo` and is part of `make check`. The demo runs on the shell and passes with zero fences.

Known lint gaps (not detected today):

- `p.Class = "..."` and `attrs["class"] = ...` assignments; only `Class:` in a Props literal and `"class"` keys in an attribute-map literal are caught.
- Go expressions in `if`/`for`/`switch` conditions of a `.templ` file are not scanned for `templ.Raw` and the other Go-side rules.
- A fence `Reason` must be a double-quoted string literal; a named constant or a raw (backquoted) string is reported as a missing reason.
- `*_test.go` files are exempt from `html-from-go-string` by design (tests assert on rendered HTML).

### The rule for app code

App code never contains anything below the component boundary (no div, no class, no layout decision); composition means choosing and ordering components. The only place that boundary may be crossed is inside a CustomPage or CustomBlock, which must carry a written reason and is printed by every build. A missing component is never a reason to use the fence: stop and file it against webframe.

### CSS source pinning

`static/css/input.css` uses `@import "tailwindcss" source(none)` and pins sources to `components/**/*.templ` and `demo/**/*.templ` (demo Go code holds no class strings; the wflint `html-from-go-string` rule keeps it that way), so markdown and JS can never change `output.css`. Class-less text gets element typography from `:where(.wf-content)` rules there.

Legacy consumer classes: three older apps still write Tailwind classes in their own `internal/web/**/*.templ` files, and `source(none)` no longer scans them. `input.css` keeps them compiling with one `@source inline("...")` line under the "Legacy consumer classes" heading, listing every token those apps use (in `class="..."` attributes and `Class: "..."` Props literals) that is not otherwise in `output.css`. Remove an app's tokens when that app migrates to `appshell`; delete the line when all three have.

## Conventions

- Always commit `*_templ.go` when `.templ` files change — reviewers need to see generated HTML
- Keep components additive; don't break existing Props contracts
- Tests are render-focused: render to `bytes.Buffer`, assert on HTML string content
- Chart components (barchart, piechart, areachart, sparkline, metricring) are SVG-based, accept `float64` values, and sanitize color inputs
- `components/render_smoke_test.go` has one table row per `components/` directory; adding a component package without a row fails the test
