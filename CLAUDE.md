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
make ui-check       # Playwright: load /items and / at 390 and 1280, fail on horizontal overflow or console errors; needs `make dev` running, not part of check
npx playwright install chromium   # one-time, before the first ui-check (Playwright's own Chromium, ~170 MB)
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

**Demo app** (`cmd/demo/` + `demo/`): HTTP server on :3333 with the showcase gallery, the traffic stream page and the Items app (`/items` list, `/items/{id}` detail, `/items/new` and `/items/{id}/edit` forms), which runs on a second, sidebar-layout `appshell.App` and is the acceptance scenario for the page layer: page declarations and handlers in `demo/items_handlers.go`, no markup of its own. `/sidebar` redirects to `/items`. Routes also include HTMX interaction demos (click counter, search, validation, toasts) whose handlers render templ fragments in `demo/pages/fragments.templ`.

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
- `listquery` (module root, beside `assets`) — `Spec`, `Options`, `Parse`/`ParseValues`, `Spec.URL` with `WithSort`/`WithPage`/`WithPageSize`/`WithSearch`/`WithFilter`/`WithoutFilters`, `Apply[T]`: the sort/filter/page query-string contract (`sort`, `dir`, `page`, `per`, `q`, filter keys; `Options.Prefix` namespaces them); renders nothing
- `components/listpage` — `Props`, `Page`, `Results` (the swap region), `Respond`: header, intro, filter bar, toolbar, data table and pagination from one declaration
- `components/detailpage` — `Props`, `Tab` (inline or lazy `HxGet`), `Page`, `Respond`: header, summary description list, tabs, free content, optional aside
- `components/formpage` — `Props`, `Section`, `Page`, `Body` (the `<form>`, the swap region), `Respond`: sections of fields with `Values`/`Errors` by name and an error summary banner
- `components/datatable` — `Column`, `Row`, `Cell`, `DataTable`: declarative table on the `table` primitive with sort links from a `Spec`, `Hidden` (phone) columns, row actions, selection, an empty state that keeps the header
- `components/pagination` — `Pagination`: always-present summary line ("1–25 of 140 items"), Previous/Next, a windowed page list; links from `Spec.URL`
- `components/toolbar` — `Toolbar`: `Left`/`Right` action groups with children in the middle; collapses into a "⋯" menu per side on phone (`Overflow`)
- `components/descriptionlist` — `Item`, `DescriptionList`: label/value rows (`Href`, `Mono`, `Copyable`, `Badge`, `Component`), 1 or 2 columns
- `components/stattile` — `StatTile`: label, big value, sub, caption, optional `Delta`, `Spark` or `Ring`, `Href`
- `components/banner` — `Banner`: inline notice (`Info`/`Success`/`Warning`/`Error`, `Details`, `Actions`, `Dismissible`); renders nothing with no title, body or details; `banner.Flash` is batch 3
- `components/toggle` — `Toggle`: CSS-only switch (`<input type="checkbox" role="switch">`), no script
- `components/image` — `Image`: the component every app picture goes through (`Src` sanitized, `Alt` always emitted, `Width`/`Height`, lazy by default, `Eager`, `Fit`, `Rounded`); wflint forbids a raw `<img>`
- `components/figure` — `Figure`: media with an optional `<figcaption>` (the "icon + caption" scaffold)
- `form.Form` — the `<form>` element itself (`Action`, `Method` default post, `HxPost`/`HxGet`/`HxTarget`/`HxSwap`, `Hidden` inputs in sorted key order, `NoValidate`, `Multipart`)
- Additive props in batch 2: `badge` has `VariantSuccess`/`VariantWarning`/`VariantInfo` and `badge.Status(kind)` / `StatusLabel(kind)`, so apps never pick colors; `formfield.Field` has `Validate` (blur validation URL) and `TypeToggle`, and `formfield.RespondFeedback` is the validation handler's whole body; `filterbar.Props.Hidden` carries list state with every filter change and `filterbar.WithSelected` marks state from the query; `emptystate.Props.Action`; `layout.StackProps.Justify`, `layout.ThreeColumnProps.LeftLabel`/`RightLabel`; `dashboardgrid.Props.Flush`; `tabs.ContentProps.Sizing` (for a panel rendered standalone via `TabsID`); `selectbox.TriggerProps.ButtonAttributes`; `autocomplete.Props.TextName`; `appshell.App.Right`, `appshell.RenderParts`, `appshell.Must`
- `gameicon` has `Size`/`Tone`; `avatar` has `Size`; `tabs` has `Sizing` (`SizingTallest` default, `SizingFixed` + `Height` S/M/L, `SizingFit`); `hovercard` has `Placement` (default bottom-start) and its card is capped to the viewport height with inner scroll

Headings: `layout.Section` defaults to an h2 and `panel` to an h3; set `Level` to nest them. `formfield.Feedback` is swapped with `hx-swap="outerHTML"` targeting `#`+`Field.FeedbackID()` (innerHTML nests the wrapper and duplicates its id). `form.Form` is the `<form>` element app code uses (wflint forbids a raw one): inside it, `Required` puts `required` on the control and the browser enforces it; for `TypeSelect` the submitting control is a hidden input the browser never validates, so there `Validate` or a server-side `Error` is the enforcement. A required `autocomplete` inside a form blocks submit until an item is picked (`setCustomValidity` on the text box while the hidden value is empty).

**List pages:** the handler calls `listquery.Parse(r, opts)` to get a `Spec`, applies it to its data (`listquery.Apply` for a slice; `Spec.Offset()`/`Limit()` for SQL), declares a `listpage.Props` (header, filters, columns, rows, the `Spec`, `Total`) and calls `listpage.Respond(w, r, title, p)`. Sort headers, filter changes and page links are all htmx GETs to the same URL that swap only the results region (`listpage.Results`, id `ID+"-results"`) and push the URL; `Respond` serves that region alone when the request's `HX-Target` names it, the page body for any other plain htmx request, and the full document otherwise (`appshell.RenderParts`). DELETE and bulk handlers that fire from inside the list parse the `Spec` with `listquery.ParseCurrent` (the `HX-Current-URL` header's query: the page the user is looking at) and answer with `listpage.RespondResults`, which sets `HX-Replace-Url` to the served state so the address bar follows a clamped page. Every results-only response (`Respond` and `RespondResults`) carries, after the results, out-of-band copies (`hx-swap-oob`) of the filter bar's hidden `sort`/`dir`/`per` inputs and of the intro line, so a sort chosen from a table header survives the next filter change and the intro's counts follow a delete. The filter bar carries `sort`, `dir` and `per` as hidden inputs (never `page`), so a filter change keeps the sort and lands on page 1; its URL rewrite drops its own parameters and `page` (`filterbar.Props.Drop`) before writing the current values, so a cleared filter leaves the address bar; the empty state's "Clear filters" drives the bar's own reset (`window.tui.filterbar.reset`) so the controls clear with the results. `listquery` caps `page` at `listquery.MaxPage` (2^31-1) so `Offset()` cannot overflow.

**Form pages:** declare a `formpage.Props` (sections of `formfield.Field`, `Action`, `HxPost: true`) and call `formpage.Respond`; create and edit are the same declaration with different `Values`. On a failed POST the handler sets `Values` from the submitted form and `Errors` by field name and calls `Respond` again with status 200: htmx 2 does not swap a 4xx response, so a 422 would leave the stale form on screen. Every error renders under its field with `aria-invalid` and once more in the error banner at the top ("Please fix the N errors below."). On success the handler saves and redirects (303, or an `HX-Redirect` header with an empty 200 for an htmx submit) and renders no success message. `Field.Validate` is a URL: the control gets `hx-post`, `hx-trigger="blur changed"` (`change` for select, checkbox, radio, toggle), `hx-target` on its feedback slot and `hx-params={Name}`, so even inside a form only that field's name=value is posted; the handler reads `r.FormValue(name)` and its whole body is `formfield.RespondFeedback(w, r, field, msg)` (empty msg renders the empty slot, status 200 either way).

**Changed defaults in batch 1b:** `tabs` are now as tall as their tallest panel by default (set `Sizing: tabs.SizingFit` for the old grow-and-shrink behavior), hover cards open below by default (set `Placement` to choose), `layout.Section` Level 2 headings are `text-2xl` and the page title is `text-3xl`. Consumers pinned to v0.1.0 (one app uses tabs and hovercard) will see these changes when they upgrade.

**Changed defaults in batch 2:** `datatable`'s selection column is hidden below `md` (a dense phone row has no room for it and bulk actions are not offered there); `layout.Inline` wraps below `md` by default (`Wrap` makes it wrap at every width); `listquery` parses a bare `?sort=key` without `dir` as ascending even when `DefaultDir` is `Desc` (the default direction belongs to the default sort only); `appshell.Render`/`RenderParts` now add `HX-Target` to `Vary` (alongside `HX-Request`, `HX-Boosted`, `HX-History-Restore-Request`); `statcomparison`, `copybutton` and `form.Message` colors now come from theme tokens (`text-success`, `text-destructive`, `text-info`) instead of hard-coded palette classes; `avatar.Image` sanitizes `Src` like `image` does (an unsafe scheme renders `about:invalid`); `input.css` gains `--color-success`, `--color-warning`, `--color-info` (each with a `-foreground`, light and dark values, registered in `@theme`).

App pattern: configure one `appshell.App`, wrap the mux in `App.Middleware`, render each page with `App.Render`:

```go
var App = &appshell.App{Name: "Demo", HomeHref: "/"}
http.ListenAndServe(addr, App.Middleware(mux))
err := App.Render(w, r, "Showcase", pages.Showcase()) // body only for a plain htmx request; full page otherwise, including boosted links and history restores
// named swap regions (what listpage.Respond and formpage.Respond do):
err = App.RenderParts(w, r, "Items", page, map[string]templ.Component{"items-results": results}) // a plain htmx request whose HX-Target is "items-results" gets only that part
```

`appshell.Must(ctx)` returns the `App` stored by `Middleware` or an error naming `Middleware`; page types call it from their `Respond`. `App.Right` renders in the topbar's right group, before the user menu.

**`assets`** embeds `static/`: `assets.Mount(mux)` serves it, `assets.Handler()` returns the handler, `assets.URL(path)` builds a URL.

**`wflint`** (`internal/wflint`, run via `cmd/wflint`) lints app code (`.templ`, Go, CSS) and rejects raw block HTML, `class`, `style`, `<style>`, `Class:` in Props literals, class/style attribute maps, `templ.Raw`, Go string literals holding HTML markup (`html-from-go-string`), and Alpine class/style bindings. Only text-level elements and component calls are allowed: `p, a, span, strong, em, b, i, code, kbd, br, ul, ol, li, time, small, sup, sub`. `img` is not among them (batch 2): an image is a sized block of content, not a word in a sentence, so app code declares it through `image.Image` and the hint says so. Allowed attributes on those elements are `href, src, alt, title, id, name, datetime, target, rel, download` plus `aria-*`, `data-*`, `hx-*`, `x-*`; `download` was added in batch 2 (a link can sit inside a sentence and downloading is behavior, not design), while `role`, `tabindex`, `width`, `height` and `loading` stay forbidden (something focusable with a role is an interactive thing, and interactive things are components; sizing and loading belong to `image`). The `html-from-go-string` rule was tightened in batch 2 so usage text does not trip it: a `class`/`style` attribute inside a Go string is reported only with a quoted value (case-sensitive, so `"class=%s"` in a log line and `"style= ?"` in SQL pass), and a forbidden opening tag only when the same literal also holds a closing tag (`</`), a quoted attribute value (`="`), or a tag directly after another (`><`), so `"<input> is not allowed"` and `"<div> → layout.Stack"` pass while `"<div class=\"x\">"` and `"<table><tr>"` fail. The sole exemption is a `CustomPage` or `CustomBlock` fence, which needs a written reason; the lint prints every fence on every run. Flags: `-json` for a JSON report; exit 1 on violations, 2 on bad usage. `make check-ui` runs `go run ./cmd/wflint ./demo` and is part of `make check`. The demo runs on the shell and passes with zero fences.

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
