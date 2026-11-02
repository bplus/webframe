# Provenance of `static/js/*`

Method for each file: `git log --follow`
on the file (first commit that added it, and its message), reading the file
content itself (version strings, license banners, code style), and
cross-referencing the matching Go/`.templ` source when one exists.

**Sandbox limitation:** this investigation ran in an offline environment —
`WebFetch` and outbound network access were both unavailable, so no upstream
URL, changelog, or LICENSE file listed below was fetched and diffed live.
Versions and licenses marked "high confidence" come from strings embedded in
the vendored file itself or from a matching in-repo version comment (verified
against repo content, not guessed). Where a fact instead relies on prior
public knowledge of a well-known project's licensing (not something the repo
can attest to), that is flagged explicitly — re-verify against the linked
upstream before treating it as a compliance-grade answer.

## Third-party vendored libraries

| File | Source | Version | License | Upgrade path |
|---|---|---|---|---|
| `3d-force-graph.min.js` | [vasturiano/3d-force-graph](https://github.com/vasturiano/3d-force-graph) | **1.73.4** — first line of the file reads `// Version 1.73.4 3d-force-graph - https://github.com/vasturiano/3d-force-graph`; matches the commit message on `eaccae72` ("Vendored 3d-force-graph v1.73.4"). High confidence. | MIT (public knowledge of this project's licensing; not fetched/verified live in this sandbox — confirm at the upstream repo's `LICENSE` before relying on it). | Unminified build + source map are published as part of the `3d-force-graph` npm package (`dist/3d-force-graph.js`, `.js.map`) for the matching version tag; no source map is vendored in this repo. |
| `htmx.min.js` | [bigskysoftware/htmx](https://github.com/bigskysoftware/htmx) | **2.0.4** — the string `version:"2.0.4"` appears in the bundle's config object. High confidence. | BSD 2-Clause (public knowledge of htmx.org's licensing; not fetched/verified live in this sandbox). | Unminified `htmx.js` for the same tag ships in the `htmx.org` npm package / GitHub release assets; useful to pin down which `hx-*` idioms this codebase was written against before upgrading. |
| `alpine.min.js` | [alpinejs/alpine](https://github.com/alpinejs/alpine) | **3.14.8** — the string `version:"3.14.8"` appears in the bundle. High confidence. | MIT (public knowledge of Alpine's licensing; not fetched/verified live in this sandbox). | Unminified `dist/cdn.js` (or module build) for the same tag ships in the `alpinejs` npm package / GitHub release assets. |

## templUI-shipped component scripts

These 10 files have no version string of their own, but each pairs 1:1 with
a `components/<name>/<name>.templ` file whose header comment reads
`// templui component <name> - version: v1.3.0 installed by templui v1.3.0`
and links `https://templui.io/docs/components/<name>`. templUI (templui.io)
is a CLI that installs component source directly into the consuming repo
(shadcn-style) rather than being pulled in as a versioned dependency, so
there is no `go.mod`/`package.json` entry to cross-check against — the
`.templ` header comment is the only version record, and it agrees for all
ten files below.

| File | Source | Paired `.templ` source | Version | License |
|---|---|---|---|---|
| `avatar.min.js` | templUI | `components/avatar/avatar.templ` | templUI v1.3.0 | MIT (public knowledge of templui's licensing; not fetched/verified live in this sandbox — confirm at github.com/templui/templui's `LICENSE`). |
| `checkbox.min.js` | templUI | `components/checkbox/checkbox.templ` | templUI v1.3.0 | MIT (see note above). |
| `dialog.min.js` | templUI | `components/dialog/dialog.templ` | templUI v1.3.0 | MIT (see note above). |
| `dropdown.min.js` | templUI | `components/dropdown/dropdown.templ` | templUI v1.3.0 | MIT (see note above). |
| `input.min.js` | templUI | `components/input/input.templ` | templUI v1.3.0 | MIT (see note above). |
| `label.min.js` | templUI | `components/label/label.templ` | templUI v1.3.0 | MIT (see note above). |
| `popover.min.js` | templUI | `components/popover/popover.templ` | templUI v1.3.0 | MIT (see note above). |
| `selectbox.min.js` | templUI | `components/selectbox/selectbox.templ` | templUI v1.3.0 | MIT (see note above). |
| `tabs.min.js` | templUI | `components/tabs/tabs.templ` | templUI v1.3.0 | MIT (see note above). |
| `textarea.min.js` | templUI | `components/textarea/textarea.templ` | templUI v1.3.0 | MIT (see note above). |

Upgrade path: re-run the templUI CLI installer for a newer version of each
component (it overwrites the `.templ` + `.min.js` pair together); there is
no unminified variant to fetch separately since templUI installs source, not
a pre-built bundle — the "minified" scripts here are templUI's own shipped
build output, not a build step this repo performs itself.

## Local (written for webframe, not templUI, not third-party)

These 5 files carry none of the templUI header markers, and their
introducing commits directly describe original component work rather than
vendoring or installing something — evidence is strong enough to call these
confirmed local, not just inferred:

| File | Source | Evidence of local authorship | License |
|---|---|---|---|
| `graphwidget.min.js` | Local | Introduced in `eaccae72` ("feat: add GraphWidget component with vendored 3d-force-graph"), which explicitly distinguishes the *vendored* `3d-force-graph.min.js` from this hand-written glue script. No `.templ` version header (`components/graphwidget/graphwidget.templ` has none). Content is readable (not obfuscated), consistent with in-repo authorship. | Part of the webframe module; the repo has no separate top-level `LICENSE` file for its own code. |
| `trafficstream.min.js` | Local | Introduced in `46d1bab` ("Add traffic stream demo component"). No `.templ` version header. Readable, unminified-looking source. | Part of the webframe module (see above). |
| `theme-toggle.js` | Local | The only standalone `.js` file, and fully readable. Contains an internal comment referencing a repo commit (`cb57069`) as the source of the "light is the product default" rule, which only makes sense for code written in and referring to this repo's own history. | Part of the webframe module (see above). |
| `autocomplete.js` | Local | Written for webframe in page-layer batch 1b (autocomplete component, paired with `components/autocomplete/autocomplete.templ`). Unminified, first-party; depends only on HTMX's `htmx:beforeRequest` / `htmx:afterSwap` events. | Part of the webframe module (see above). |
| `confirm.js` | Local | Written for webframe in page-layer batch 3 (the `htmx:confirm` interceptor behind `action.Confirm`, paired with `components/confirmdialog/confirmdialog.templ`). Unminified, first-party; depends on HTMX's `htmx:confirm` event and the `window.tui.dialog` API of the templUI-shipped `dialog.min.js`. | Part of the webframe module (see above). |

## UNKNOWN provenance

These 2 files are **not** classified as local or third-party. They lack the
templUI header (unlike every confirmed templUI component in this repo), and
their introducing commit message describes them as original design
("Mirrors dialog component pattern", "HTMX integration via htmx.ajax, URL
param sync via history.replaceState") rather than an install — which points
away from "templUI-shipped." But that is not the same as confirming they
were *written* for webframe: templUI's live component catalog could not be
checked from this offline sandbox, so an unlisted or newer templUI
"filterbar"/"slideover" component cannot be ruled out. Per the ticket's
instruction to mark genuinely undeterminable rows as UNKNOWN rather than
guess, both stay UNKNOWN pending a check of templui.io's current catalog
with network access:

| File | Source | What's known | License |
|---|---|---|---|
| `filterbar.min.js` | UNKNOWN | Introduced alongside `slideover.min.js` in `5138e52` ("feat: add SlideOver and FilterBar components"). No `.templ` version header (`components/filterbar/filterbar.templ` has none). Commit message reads like original design, not an install — but templUI's catalog was not checked live. | UNKNOWN — pending source determination. |
| `slideover.min.js` | UNKNOWN | Same commit (`5138e52`); commit body says it "Mirrors dialog component pattern," i.e. modeled on the templUI-shipped `dialog.min.js` idiom. No `.templ` version header. Same caveat: templUI's live catalog was not checked. | UNKNOWN — pending source determination. |

## Summary

18 JavaScript assets live in `static/js/` (17 `.min.js` files plus the one
standalone `theme-toggle.js`; this PROVENANCE.md file itself is not counted).
Of those 18: 3 are third-party vendored libraries (versions and provenance
confirmed from in-file strings and commit messages), 10 are templUI
v1.3.0-shipped component scripts (versions confirmed via the paired
`.templ` header comment), 3 are confirmed local (identified by absence of
the templUI marker plus a commit message directly describing original
authorship), and 2 — `filterbar.min.js` and `slideover.min.js` — are
UNKNOWN: the same absence-of-templUI-header signal applies, but it only
rules out "templUI-shipped," it doesn't positively confirm "written for
webframe," and that distinction couldn't be settled without checking
templUI's live catalog, which this offline sandbox could not do. The two
caveats worth flagging to a human: (1) none of the license attributions
above were verified against a live-fetched LICENSE file, since this
investigation ran with no network access; (2) `filterbar.min.js` /
`slideover.min.js` need that same live check against templui.io's current
component list before their Source can move out of UNKNOWN.

## Page layer

Batch 1 of the page layer vendored no new JavaScript. This heading is the slot for later batches that do.

Batch 1b adds `autocomplete.js`, first-party and unminified, listed under
Local above. The counts in the Summary predate it.

Batch 2 (input #13, hover card height at phone width) makes one local edit
to the templUI-shipped `popover.min.js`: the positioning middleware list
(`offset, flip, shift[, arrow]`) gains Floating UI's `size` step after
`shift`, whose `apply` writes the available height on the final side to
the CSS variable `--wf-pop-max` on the floating element (the bundled
`window.FloatingUIDOM.size` was already present, just unused). Nothing else
in the file changed; re-installing the component from templUI would drop
this edit, so re-apply it (grep for `--wf-pop-max`) after any upgrade.

Batch 2 (Task B4, `filterbar.Props.Hidden`) makes one local edit to
`filterbar.min.js`: `resetAll` skips controls whose `type` is `hidden`
(`if(el.type==="hidden"){return}`) before clearing the rest. Hidden inputs
carry list state that is not a filter (sort, dir, per), so "Clear filters"
clears the filters and leaves them in place; `collectParams` already read
every `[data-tui-filterbar-control]` by `name`, so hidden inputs ride along
with no other change. Nothing else in the file changed.

Batch 2 review fix (F1, URL sync) makes a second local edit to
`filterbar.min.js`, in `triggerUpdate`'s URL block: before writing the
collected parameters into the page URL it now deletes every parameter named
by one of the bar's controls (`[data-tui-filterbar-control][name]` and
`[data-tui-filterbar-toggle][name]`, empty ones included) and every name in
the bar's optional `data-tui-filterbar-drop` attribute (comma-separated;
`filterbar.Props.Drop`), then sets the non-empty values as before. A cleared
control or a stale `page=` therefore leaves the URL, while parameters the bar
does not own are kept (the search string is edited, not replaced). Nothing
else in the file changed.

Batch 3 (Task A0) adds `confirm.js`, first-party (webframe, not templUI, not
third-party), committed empty so `appshell.Scripts` can include it; it is
listed under Local above.

Batch 3 (Task A3) fills `confirm.js`: unminified, about 80 lines, no
dependencies beyond htmx's `htmx:confirm` event and the public
`window.tui.dialog` API (`open`/`close` by instance id) that the
templUI-shipped `dialog.min.js` already exposes. It intercepts every
`htmx:confirm` that carries an `hx-confirm` question when the page has the
`#wf-confirm` host (`confirmdialog.Host`, rendered by `appshell.Document`),
fills the dialog's `data-wf-confirm-part` elements from the trigger's
`data-wf-confirm-*` attributes, opens it, and calls
`evt.detail.issueRequest(true)` on confirm. It watches the panel's
`data-tui-dialog-open` attribute (a MutationObserver) to learn about every
close `dialog.min.js` performs (Cancel, Escape, backdrop). `dialog.min.js`
itself is unchanged.

Batch 3 (Task A12, phone sort control) makes a third local edit to
`filterbar.min.js`, in three places: a new `applySets(bar)` function, before
`collectParams`, reads the selected `<option>` of every
`select[data-tui-filterbar-control]` in the bar and, for each of its
attributes named `data-tui-filterbar-set-<name>`, writes the attribute's
value into the bar's control whose `name` is `<name>` (compared
case-insensitively, since the HTML parser lowercases attribute names; the
select itself is skipped), so one option can set a second parameter
(`filterbar.FilterOption.Attributes`; listpage's Sort select sets `dir`).
`triggerUpdate` calls `applySets(bar)` before `collectParams(bar)`, so the
rule holds for every update, not only a change of that select (a filter
change in the default order must still send the matching `dir`). `resetAll`
now also skips controls carrying `data-tui-filterbar-state` (a control
promoted to carry list state in place of a hidden input, see
`filterbar.Props.Hidden`), alongside `type="hidden"`. Nothing else in the
file changed.

Batch 3 (Task A11, colors from theme tokens) edits the first-party
`trafficstream.min.js` in its `updateConnection` calls only: the class
names it swaps onto the connection dot change from palette classes to
tokens (`bg-amber-500` → `bg-warning` for Paused and Reconnecting,
`bg-emerald-500` → `bg-success` for Live, `bg-slate-400` →
`bg-muted-foreground` for Static). Because `input.css` uses `source(none)`
and never scans JS, those three classes are kept alive by an
`@source inline("bg-success bg-warning bg-muted-foreground")` line there;
a future class written only by a script needs the same by-hand entry.
Nothing else in the file changed.
