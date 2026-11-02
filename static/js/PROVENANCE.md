# Provenance of `static/js/*`

Origin and license of every script in this directory. Checked 2026-11-02
against the upstream repositories and npm packages named below. "Identical"
means byte-for-byte equal to the upstream file at that version.

## Third-party libraries

| File | Origin | Version | License | How verified |
|---|---|---|---|---|
| `htmx.min.js` | [bigskysoftware/htmx](https://github.com/bigskysoftware/htmx) | 2.0.4 | 0BSD (Zero-Clause BSD) | Identical to npm `htmx.org@2.0.4/dist/htmx.min.js`. License read from `LICENSE` at tag `v2.0.4`. |
| `alpine.min.js` | [alpinejs/alpine](https://github.com/alpinejs/alpine) | 3.14.8 | MIT, Copyright © 2019-2021 Caleb Porzio and contributors | Identical to npm `alpinejs@3.14.8/dist/cdn.min.js`. License read from `LICENSE.md` at tag `v3.14.8`. |
| `3d-force-graph.min.js` | [vasturiano/3d-force-graph](https://github.com/vasturiano/3d-force-graph) | 1.73.4 | MIT, Copyright (c) 2017 Vasco Asturiano | Identical to npm `3d-force-graph@1.73.4/dist/3d-force-graph.min.js`. License read from `LICENSE` at tag `v1.73.4`. |

`3d-force-graph.min.js` is a UMD bundle that also contains its dependencies,
with no license banners of their own: three.js r168 (MIT, Copyright © 2010-2024
three.js authors), three-forcegraph 1.41.14, three-render-objects 1.29.5,
kapsule, accessor-fn, data-joint, d3-force-3d, the d3 packages, the ngraph
packages, tinycolor2, tween.js, polished and others. Licenses are MIT, ISC,
BSD-3-Clause (the ngraph packages) and Apache-2.0 (ColorBrewer schemes in
d3-scale-chromatic). The full resolved list with versions and copyright lines
is in `THIRD_PARTY_NOTICES.md`.

## templUI component scripts

Source: [templui/templui](https://github.com/templui/templui), v1.3.0,
`internal/components/<name>/<name>.min.js`. License: MIT, Copyright (c) 2023
Axel Adrian (`LICENSE` at tag `v1.3.0`). Each pairs with
`components/<name>/<name>.templ`, whose header reads
`templui component <name> - version: v1.3.0`.

| File | Origin | Version | License | How verified |
|---|---|---|---|---|
| `avatar.min.js` | templUI | v1.3.0 | MIT | Identical to upstream at `v1.3.0`. |
| `checkbox.min.js` | templUI | v1.3.0 | MIT | Identical to upstream at `v1.3.0`. |
| `dialog.min.js` | templUI | v1.3.0 | MIT | Identical to upstream at `v1.3.0`. |
| `dropdown.min.js` | templUI | v1.3.0 | MIT | Identical to upstream at `v1.3.0`. |
| `input.min.js` | templUI | v1.3.0 | MIT | Identical to upstream at `v1.3.0`. |
| `label.min.js` | templUI | v1.3.0 | MIT | Identical to upstream at `v1.3.0`. |
| `popover.min.js` | templUI, modified | v1.3.0 | MIT (templUI); bundles Floating UI, MIT, Copyright (c) 2021 Floating UI contributors | Upstream at `v1.3.0` plus one local edit (see Local modifications). Upstream `popover.js` imports `floating_ui_dom.js` and `floating_ui_core.js`, which the minified file bundles as `window.FloatingUIDOM` / `window.FloatingUICore`. |
| `selectbox.min.js` | templUI | v1.3.0 | MIT | Identical to upstream at `v1.3.0`. |
| `tabs.min.js` | templUI | v1.3.0 | MIT | Identical to upstream at `v1.3.0`. |
| `textarea.min.js` | templUI | v1.3.0 | MIT | Identical to upstream at `v1.3.0`. |
| `slideover.min.js` | templUI `dialog.min.js`, adapted for webframe | v1.3.0 base | MIT (templUI) | Added in `5138e52` ("Mirrors dialog component pattern"). It is templUI v1.3.0 `dialog.min.js` with `dialog` renamed to `slideover` and `content` renamed to `panel` in attribute names, plus one added `document.dispatchEvent(new CustomEvent("slideover-open", ...))`; 98% character match after those renames. templUI has no slideover or filterbar component at any tag (all tags through v2.0.0 checked); its `sheet` component ships no script. Unchanged since `5138e52`. |

## First-party (webframe, MIT)

Written in this repository; covered by webframe's own license.

| File | Origin | License | How verified |
|---|---|---|---|
| `filterbar.min.js` | webframe | MIT (webframe) | Added in `5138e52` together with `components/filterbar/filterbar.templ`; later edited in `96b044a`, `45bcbf6`, `b36d458`. Hand-written ES5 (`var`, named functions, readable identifiers such as `findBar`, `collectParams`, `triggerUpdate`), not minifier output; only the outer `(()=>{(function(){"use strict";` wrapper copies the templUI file shape. No matching file in any templUI tag, and GitHub code search for `data-tui-filterbar` finds only bplus repositories. |
| `graphwidget.min.js` | webframe | MIT (webframe) | Added in `eaccae72` as the glue script for the vendored 3d-force-graph. |
| `theme-toggle.js` | webframe | MIT (webframe) | Unminified; refers to this repository's own commit `cb57069`. |
| `autocomplete.js` | webframe | MIT (webframe) | Unminified; written with `components/autocomplete`. |
| `confirm.js` | webframe | MIT (webframe) | Unminified; written with `components/confirmdialog`. |
| `htmx-errors.js` | webframe | MIT (webframe) | Unminified; loaded by `appshell.Scripts`. |
| `splitpane.js` | webframe | MIT (webframe) | Unminified; split pane link handler. |

## Local modifications to third-party files

Re-installing these from templUI drops the edits; re-apply them after any
upgrade.

| File | Edit | Find it by |
|---|---|---|
| `popover.min.js` | Adds Floating UI's `size` step after `shift` in the positioning middleware; its `apply` writes the available height to the CSS variable `--wf-pop-max` on the floating element. Nothing else differs from upstream. | `--wf-pop-max` |
| `slideover.min.js` | Derived from templUI `dialog.min.js` as described above. | `data-tui-slideover` |

## Behavior notes for first-party scripts

- `filterbar.min.js`: `resetAll` skips controls with `type="hidden"` or
  `data-tui-filterbar-state` (list state such as sort, dir and per survives
  "Clear filters"). `triggerUpdate` runs `applySets(bar)` (a selected
  `<option>`'s `data-tui-filterbar-set-<name>` attributes write into the bar's
  control named `<name>`), then `collectParams(bar)`, then edits the page URL:
  it deletes every parameter named by a bar control or toggle and every name
  in `data-tui-filterbar-drop`, then sets the non-empty values.
- `graphwidget.min.js`: builds each graph with
  `ForceGraph3D({ controlType })(mount)`, the factory form 3d-force-graph 1.73
  documents (`new ForceGraph3D(el)` is the 1.74+ form and attaches nothing on
  1.73). A missing library, a construction error, no canvas after
  construction, a non-2xx or non-JSON DataURL answer and data without a
  `nodes` array each show the widget's error box and log `console.error`.
  `make graph-check` covers this in a browser.
- `confirm.js`, `htmx-errors.js`, `autocomplete.js`, `theme-toggle.js`,
  `splitpane.js`: each guards against a second run through `window.wf.<name>`
  (htmx re-executes `<script src>` tags in swapped content). The templUI
  scripts are not guarded.
- `htmx-errors.js`: on `htmx:beforeSwap` with `evt.detail.isError`, refuses
  the swap when the response is not `text/html` or the target sits inside
  `data-wf-swallow-errors`; otherwise leaves htmx's decision alone.
- `confirm.js`: answers `htmx:confirm` with the `#wf-confirm` dialog through
  `window.tui.dialog` (from `dialog.min.js`, unchanged).
