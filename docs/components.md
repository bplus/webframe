# Components

This is the catalog of every package in webframe, grouped by what you are
trying to build. Each row says what the package is, when to use it, and where
to see it running.

To see the examples, run the demo with `make dev` and open
http://localhost:3333. In the "See it" column, `/#datatable` means the
section with that anchor on the showcase page (the demo's home page), and
`/items` means that route in the demo. "not in demo" means the demo does not
show it yet.

## Start here

Most screens are one page type. Pick the one that fits, declare its `Props`,
and call its `Respond`.

- **One page inside the app's frame:** `appshell` with a `pageheader`. See
  [tutorial step 1](tutorial/01-hello.md).
- **A list of records** (sort, filter, search, pages): `listpage` with
  `listquery`. See [tutorial step 2](tutorial/02-list.md).
- **A create or edit form:** `formpage`. See [tutorial step 3](tutorial/03-form.md),
  and [step 4](tutorial/04-validation.md) for showing errors on bad input.
- **One record on its own page:** `detailpage`. See `/items/1` in the demo.
- **A dashboard of numbers and summaries:** `dashboardpage`. See `/dashboard`.
- **A settings screen:** `settingspage`. See `/settings`.
- **A list beside the selected item's detail, mail style:** `splitpane`. See `/browse`.
- **Sign in, signed out, access denied:** `authpage`. See `/signin`.
- **404, 403, 405 and 500 pages:** `errorpage`. See `/errors/404`.
- **Telling the user something happened:** `toast` after an htmx action,
  `flash` after a redirect. See [tutorial step 3](tutorial/03-form.md) for a
  flash.

## Page types

A page type renders a whole page from one declaration. App code fills in data
and never decides layout.

| Package | What it is and when to use it | See it |
|---|---|---|
| [`listpage`](https://pkg.go.dev/github.com/bplus/webframe/components/listpage) | A list page: header, filter bar, toolbar, data table and pagination, where each sort, filter or page change re-renders only the results over htmx. | `/items` |
| [`formpage`](https://pkg.go.dev/github.com/bplus/webframe/components/formpage) | A create or edit form page: sections of fields, an error banner and Cancel/Submit; create and edit are the same declaration with different values. | `/items/new`, `/items/1/edit` |
| [`detailpage`](https://pkg.go.dev/github.com/bplus/webframe/components/detailpage) | The page for one record: header, a summary list of fields, optional tabs (loaded inline or on demand), free content and an optional side column. | `/items/1` |
| [`dashboardpage`](https://pkg.go.dev/github.com/bplus/webframe/components/dashboardpage) | A dashboard: a page header, a row of stat tiles, then titled sections, all on one 12-column grid. | `/dashboard`, `/#dashboardpage` |
| [`settingspage`](https://pkg.go.dev/github.com/bplus/webframe/components/settingspage) | A settings screen: groups of fields in one form, optional content after it (such as a danger zone), and a save bar stuck to the bottom of the screen. | `/settings`, `/#settingspage` |
| [`splitpane`](https://pkg.go.dev/github.com/bplus/webframe/components/splitpane) | A list on the left and the selected item's detail on the right; on a phone it shows one of the two, chosen from the URL. | `/browse`, `/#splitpane` |
| [`authpage`](https://pkg.go.dev/github.com/bplus/webframe/components/authpage) | The pages shown outside the app's frame: sign in, signed out and forbidden, each one centered card with no navigation. | `/signin`, `/signed-out`, `/forbidden`, `/#authpage` |
| [`errorpage`](https://pkg.go.dev/github.com/bplus/webframe/components/errorpage) | 404, 403, 405 and 500 pages with the right status code, plus a wrapper that turns Go's plain-text 404 and 405 into these pages. | `/errors/404`, `/errors/500`, `/#errorpage`, `/#methodnotallowed` |
| [`custompage`](https://pkg.go.dev/github.com/bplus/webframe/components/custompage) | The escape hatch for a whole page of hand-written markup that the lint would reject; it needs a written reason and every build lists it. Use it only when no component can do the job. | not in demo (described at `/#custom`) |

## App shell and navigation

| Package | What it is and when to use it | See it |
|---|---|---|
| [`appshell`](https://pkg.go.dev/github.com/bplus/webframe/components/appshell) | The app's frame (top bar or sidebar navigation, the HTML document, the script bundle) and the per-request wiring: configure one `App`, wrap the router in `App.Middleware`, render pages with `App.Render`. | every page, `/#shell` |
| [`pageheader`](https://pkg.go.dev/github.com/bplus/webframe/components/pageheader) | The title block at the top of a page: title, optional back link, status badge and actions. | `/#shell`, every demo page |
| [`breadcrumbs`](https://pkg.go.dev/github.com/bplus/webframe/components/breadcrumbs) | A navigation trail where every item but the last is a link. | `/#shell` |
| [`commandpalette`](https://pkg.go.dev/github.com/bplus/webframe/components/commandpalette) | The app-wide "jump anywhere" search dialog, opened with Cmd+K (Ctrl+K) or a Search button; set `App.Palette` and the shell does the rest. | `/#commandpalette`, Cmd+K on any page |
| [`action`](https://pkg.go.dev/github.com/bplus/webframe/components/action) | The shared `Action` type for a button or link (label, variant, htmx request, confirmation) that headers, toolbars, panels and menus all take. | `/#actions` |
| [`toolbar`](https://pkg.go.dev/github.com/bplus/webframe/components/toolbar) | A row of actions in a left and a right group around optional middle content; on a phone the extra actions move into a "⋯" menu. | `/#toolbar`, `/items` |
| [`tabs`](https://pkg.go.dev/github.com/bplus/webframe/components/tabs) | A tab group: a row of tab buttons over content panels. Inside a detail page, use `detailpage`'s tabs instead. | `/#tabs`, `/items/1` |

## Layout

| Package | What it is and when to use it | See it |
|---|---|---|
| [`layout`](https://pkg.go.dev/github.com/bplus/webframe/components/layout) | The spacing and grouping pieces: `Stack` (one under another), `Inline` (side by side), `Section` (titled), `Divider`, `TwoColumn` and `ThreeColumn`. | `/#layout` |
| [`panel`](https://pkg.go.dev/github.com/bplus/webframe/components/panel) | A framed block of content with an optional title, actions, collapsing, and a flush mode for tables. | `/#panel` |
| [`card`](https://pkg.go.dev/github.com/bplus/webframe/components/card) | A bordered, rounded box with optional header, title, description, content and footer parts. For a grid of item cards, use `cardgrid`. | `/#cards` |
| [`dashboardgrid`](https://pkg.go.dev/github.com/bplus/webframe/components/dashboardgrid) | A column grid whose slots span a number of columns; it collapses to one column on a phone. `dashboardpage` uses it for you. | `/#stattiles`, `/#image` |
| [`aspectratio`](https://pkg.go.dev/github.com/bplus/webframe/components/aspectratio) | Holds its content at a fixed width-to-height ratio (square, 16:9, 3:4, 2:1), for images, video or maps. | `/#aspectratio` |
| [`separator`](https://pkg.go.dev/github.com/bplus/webframe/components/separator) | A horizontal or vertical divider line, optionally with a label. | `/#separator` |
| [`figure`](https://pkg.go.dev/github.com/bplus/webframe/components/figure) | One piece of media (icon, image, chart, swatch) with an optional caption under it. | `/#figure` |
| [`customblock`](https://pkg.go.dev/github.com/bplus/webframe/components/customblock) | The escape hatch for a block of hand-written markup inside a page; it needs a written reason and every build lists it. A missing component is a reason to file an issue, not to use this. | not in demo (described at `/#custom`) |

## Forms and inputs

Inside a `formpage` or `settingspage` you declare `formfield.Field` values and
pick the widget with `Field.Type`; the packages below are what those fields
render. Use them directly only outside a page type.

| Package | What it is and when to use it | See it |
|---|---|---|
| [`formfield`](https://pkg.go.dev/github.com/bplus/webframe/components/formfield) | A labelled form control with help text and validation feedback, from one `Field` value; `Fieldset` lays several out in columns. | `/#formfields`, `/items/new` |
| [`form`](https://pkg.go.dev/github.com/bplus/webframe/components/form) | The `<form>` element itself (action, method, htmx attributes, hidden inputs, file uploads); app code uses it instead of a raw form tag. | `/#form-elements`, `/#autocomplete` |
| [`button`](https://pkg.go.dev/github.com/bplus/webframe/components/button) | A styled button, or a link when it has an `Href`, in several variants and sizes. Page code usually passes an `action.Action` instead. | `/#buttons` |
| [`input`](https://pkg.go.dev/github.com/bplus/webframe/components/input) | A styled single-line text input (text, email, password, number and so on). | `/#form-elements` |
| [`textarea`](https://pkg.go.dev/github.com/bplus/webframe/components/textarea) | A styled multi-line text box. | `/#form-elements` |
| [`checkbox`](https://pkg.go.dev/github.com/bplus/webframe/components/checkbox) | A native checkbox, restyled. | `/#form-elements` |
| [`radio`](https://pkg.go.dev/github.com/bplus/webframe/components/radio) | A single native radio button, restyled. | `/#form-elements` |
| [`toggle`](https://pkg.go.dev/github.com/bplus/webframe/components/toggle) | An on/off switch built on a checkbox, with no script. | `/items/new` (Remixable), `/settings` |
| [`label`](https://pkg.go.dev/github.com/bplus/webframe/components/label) | A `<label>` tied to a form control. | `/#form-elements` |
| [`selectbox`](https://pkg.go.dev/github.com/bplus/webframe/components/selectbox) | A custom select menu with search, groups and multi-select. | `/#selectbox` |
| [`autocomplete`](https://pkg.go.dev/github.com/bplus/webframe/components/autocomplete) | A type-ahead picker that asks the server for suggestions as the user types and submits the picked item's value. | `/#autocomplete` |
| [`tagsinput`](https://pkg.go.dev/github.com/bplus/webframe/components/tagsinput) | A free-text tag editor that shows each tag as a removable chip and submits them as one comma-joined field. | `/#tagsinput`, `/items/new` |
| [`keyvalueeditor`](https://pkg.go.dev/github.com/bplus/webframe/components/keyvalueeditor) | Editable key and value rows with add and remove buttons, for things like environment variables. | `/#keyvalueeditor`, `/settings` |
| [`filedrop`](https://pkg.go.dev/github.com/bplus/webframe/components/filedrop) | A file picker drawn as a drop zone: click to pick, or drag files onto it; lists the chosen files. | `/#filedrop`, `/items/new` |
| [`datetimepicker`](https://pkg.go.dev/github.com/bplus/webframe/components/datetimepicker) | The browser's own date, date-and-time and time inputs, with a `Parse` for the handler side. | `/#datetimepicker`, `/items/new` |
| [`rating`](https://pkg.go.dev/github.com/bplus/webframe/components/rating) | A row of stars, read-only ("3 of 5") or as an input that submits the chosen number, with no script. | `/#rating`, `/items/new` |
| [`filterbar`](https://pkg.go.dev/github.com/bplus/webframe/components/filterbar) | A row of filter controls (selects, search box, toggle pills) that reload part of the page over htmx. `listpage` builds one for you. | `/items` |

## Data display

| Package | What it is and when to use it | See it |
|---|---|---|
| [`datatable`](https://pkg.go.dev/github.com/bplus/webframe/components/datatable) | A table declared as columns and rows, with sort links, a selection column, row actions and an empty state that keeps the header. `listpage` uses it for you. | `/#datatable`, `/items` |
| [`table`](https://pkg.go.dev/github.com/bplus/webframe/components/table) | The plain styled table parts (`Table`, `Header`, `Row`, `Cell`) that `datatable` is built on; reach for `datatable` first. | `/#panel` |
| [`pagination`](https://pkg.go.dev/github.com/bplus/webframe/components/pagination) | The "1–25 of 140 items" line plus Previous, Next and page links for one list. | `/#pagination`, `/items` |
| [`descriptionlist`](https://pkg.go.dev/github.com/bplus/webframe/components/descriptionlist) | Label and value rows for a record's fields, with optional links, badges and copy buttons. | `/#descriptionlist`, `/items/1` |
| [`cardgrid`](https://pkg.go.dev/github.com/bplus/webframe/components/cardgrid) | A grid of cards (a catalog, a library, a list of people), each with optional art, a title link, badges and footer actions. | `/#cardgrid`, `/dashboard` |
| [`timeline`](https://pkg.go.dev/github.com/bplus/webframe/components/timeline) | Dated events on a vertical line, newest first and grouped by day: an activity history or change log. | `/#timeline`, `/dashboard` |
| [`stattile`](https://pkg.go.dev/github.com/bplus/webframe/components/stattile) | One dashboard number in a tile, with an optional change ("+12%"), sparkline or ring. | `/#stattiles`, `/dashboard` |
| [`statcomparison`](https://pkg.go.dev/github.com/bplus/webframe/components/statcomparison) | A tile showing a current value against a previous one with an up, down or flat change. The arrow follows the direction; the color says whether that is good, and `LowerIsBetter` makes a fall green (open bugs, error rate). `stattile` reuses its arrows and color rule. | `/#statcomparison` |
| [`badge`](https://pkg.go.dev/github.com/bplus/webframe/components/badge) | A small inline label pill; `badge.Status` picks the color for a status word so apps never choose colors. | `/#badges` |
| [`dotlabel`](https://pkg.go.dev/github.com/bplus/webframe/components/dotlabel) | A compact label with a colored dot, for naming a thing whose kind should be easy to scan, such as in a table cell. | `/#datatable` |
| [`avatar`](https://pkg.go.dev/github.com/bplus/webframe/components/avatar) | A round picture of a person that falls back to initials or an icon. | `/#avatar` |
| [`image`](https://pkg.go.dev/github.com/bplus/webframe/components/image) | The component every picture goes through (the lint forbids a raw `<img>`): safe source, required alt text, lazy loading, size and fit. | `/#image` |
| [`icon`](https://pkg.go.dev/github.com/bplus/webframe/components/icon) | The Lucide icon set as Go variables, such as `icon.Activity`. | `/#icons` |
| [`swatch`](https://pkg.go.dev/github.com/bplus/webframe/components/swatch) | A small color square with an optional label. | `/#colors` |
| [`emptystate`](https://pkg.go.dev/github.com/bplus/webframe/components/emptystate) | A centered placeholder for an empty list or panel: icon, title, description and an optional action such as "Create item". | `/#datatable`, `/browse` |
| [`progressbar`](https://pkg.go.dev/github.com/bplus/webframe/components/progressbar) | A horizontal progress track with a fill and an optional label and percentage. `Variant` picks the fill color: default (primary), success, warning, destructive or info. | `/#progressbar` |
| [`lifecyclestepper`](https://pkg.go.dev/github.com/bplus/webframe/components/lifecyclestepper) | A horizontal row of stages joined by lines, each done, active, waiting on someone or still to come: a checkout flow, an approval chain, a task's lifecycle. On a phone the row wraps between stages, and a new row starts with a stage, not a line. | `/#lifecyclestepper` |

### Charts

All charts are inline SVG drawn on the server from `float64` values.

| Package | What it is and when to use it | See it |
|---|---|---|
| [`areachart`](https://pkg.go.dev/github.com/bplus/webframe/components/areachart) | One or more series drawn as lines with filled areas under them, in theme colors unless a series sets its own. It shrinks to fit its container. | `/#areachart` |
| [`barchart`](https://pkg.go.dev/github.com/bplus/webframe/components/barchart) | A bar chart, horizontal or vertical, in the order given. `ShowValues` prints each value; `MaxBars` folds the smallest bars into one "Other" bar. | `/#barchart` |
| [`piechart`](https://pkg.go.dev/github.com/bplus/webframe/components/piechart) | A donut (the default) or full pie chart. | not in demo |
| [`sparkline`](https://pkg.go.dev/github.com/bplus/webframe/components/sparkline) | A small trend line with no axes or labels; only the shape means anything. | inside a tile at `/#stattiles` |
| [`metricring`](https://pkg.go.dev/github.com/bplus/webframe/components/metricring) | A circular percentage gauge. | inside a tile at `/#stattiles` |
| [`graphwidget`](https://pkg.go.dev/github.com/bplus/webframe/components/graphwidget) | An interactive 3D force-directed graph of nodes and links, loaded as JSON from a URL; needs WebGL. | `/graph` (not linked from the menus) |

## Feedback

Telling the user what happened, asking before a risky action, and showing that
something is loading.

| Package | What it is and when to use it | See it |
|---|---|---|
| [`toast`](https://pkg.go.dev/github.com/bplus/webframe/components/toast) | A short notice that pops up at the top of the screen and leaves on its own; send one from an htmx handler with `toast.Send`. | `/#toast` |
| [`flash`](https://pkg.go.dev/github.com/bplus/webframe/components/flash) | A one-time message stored in a cookie before a redirect and shown once as a banner on the next page, such as "Saved". | `/#flash`, saving `/items/new` |
| [`banner`](https://pkg.go.dev/github.com/bplus/webframe/components/banner) | An inline notice inside the page (info, success, warning, error) that renders nothing when it has nothing to say. | `/#banner` |
| [`confirmdialog`](https://pkg.go.dev/github.com/bplus/webframe/components/confirmdialog) | The styled "Are you sure?" dialog. You never render it: set `Confirm` on an `action.Action` and the shell's one dialog asks. | `/#confirmdialog` |
| [`spinner`](https://pkg.go.dev/github.com/bplus/webframe/components/spinner) | A spinning ring for a request whose result has no known shape, usable as an htmx loading indicator. | `/#spinner` |
| [`skeleton`](https://pkg.go.dev/github.com/bplus/webframe/components/skeleton) | Grey placeholder shapes (lines, card, table, tile) the same height as the content they stand in for, so the page does not jump when it loads. | `/#skeleton` |

## Content

Showing or editing free-form material: text, code, logs, JSON and diffs.

| Package | What it is and when to use it | See it |
|---|---|---|
| [`prose`](https://pkg.go.dev/github.com/bplus/webframe/components/prose) | Styled long-form text: trusted HTML (rendered Markdown, help pages) written as is, or untrusted plain text escaped into paragraphs. The only sanctioned way to put content HTML on a page. | `/#prose`, `/items/1` |
| [`codeblock`](https://pkg.go.dev/github.com/bplus/webframe/components/codeblock) | Source code in a monospace box with optional language label, copy button, line numbers, wrapping and height cap; no syntax highlighting. | `/#codeblock` |
| [`logviewer`](https://pkg.go.dev/github.com/bplus/webframe/components/logviewer) | A fixed-height log box with a filter, a Follow switch and a download action; it can poll the server for new lines. | `/#logviewer`, `/logs` |
| [`jsonviewer`](https://pkg.go.dev/github.com/bplus/webframe/components/jsonviewer) | Any Go value shown as a collapsible JSON tree, with optional copy-path buttons. | `/#jsonviewer`, `/items/1` |
| [`diffview`](https://pkg.go.dev/github.com/bplus/webframe/components/diffview) | File diffs as tables, inline or side by side, built by hand or parsed from `git diff` output. | `/#diffview`, History tab on `/items/1` |
| [`copybutton`](https://pkg.go.dev/github.com/bplus/webframe/components/copybutton) | An icon button that copies a value to the clipboard. `codeblock`, `descriptionlist` and `jsonviewer` include it for you. | inside `/#codeblock`, ID row on `/items/1` |

## Overlays

Things that open on top of the page.

| Package | What it is and when to use it | See it |
|---|---|---|
| [`dialog`](https://pkg.go.dev/github.com/bplus/webframe/components/dialog) | A centered modal window opened by a trigger, with header, content and footer parts. For a yes/no question, use `action.Confirm` instead. | `/#dialog` |
| [`slideover`](https://pkg.go.dev/github.com/bplus/webframe/components/slideover) | A panel that slides in from a screen edge (left, right, or bottom as a sheet), for settings or details where a centered dialog would be wrong. | `/#bottomsheet` |
| [`dropdown`](https://pkg.go.dev/github.com/bplus/webframe/components/dropdown) | A menu that opens on click, with items, groups, separators and sub-menus. | `/#dropdown` |
| [`popover`](https://pkg.go.dev/github.com/bplus/webframe/components/popover) | Floating content anchored to a trigger, opened on click or hover; the base that dropdown, selectbox and hovercard are built on. | `/#popover` |
| [`hovercard`](https://pkg.go.dev/github.com/bplus/webframe/components/hovercard) | A preview card (title, description, details, link) shown while the user hovers something. | `/#hovercard` |

## Helpers

Packages at the module root that render nothing on their own.

| Package | What it is and when to use it | See it |
|---|---|---|
| [`listquery`](https://pkg.go.dev/github.com/bplus/webframe/listquery) | Reads a list's sort, direction, page, page size, search and filters from the URL and builds the links back; `Apply` sorts and pages a Go slice, `Offset`/`Limit` feed SQL. | `/items` |
| [`compose`](https://pkg.go.dev/github.com/bplus/webframe/compose) | Builds component trees in plain Go, without a `.templ` file: `compose.With` puts content inside a container component. | `/dashboard` (the pack counts), [tutorial step 1](tutorial/01-hello.md) |
| [`assets`](https://pkg.go.dev/github.com/bplus/webframe/assets) | Serves webframe's embedded CSS and JavaScript; call `assets.Mount(mux)` once. | every page |
