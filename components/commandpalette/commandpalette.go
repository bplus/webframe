// Package commandpalette is the app-wide "jump anywhere" dialog: a search
// input over a server-rendered, grouped list of links that Alpine filters
// as the user types. It opens on a keyboard shortcut (mod+k by default:
// Cmd+K on a Mac, Ctrl+K elsewhere) and from Trigger, the ghost button the
// shell shows in its topbar / sidebar footer.
//
// An app sets appshell.App.Palette to a *Props; the shell then renders
// Trigger in its right controls and appshell.Document renders Palette once
// near the end of <body>. App code never renders Palette or Trigger itself.
//
// Every item is a plain <a href>, so the list works without JS (open the
// dialog through its id and the links are ordinary links) and Enter on the
// active item is a plain navigation (the anchor's click; hx-boost applies
// when the page has it, nothing else is assumed). Filtering, the active
// row and the shortcut live in one inline Alpine x-data on the dialog
// root; the JS reads labels and keywords from the items' data attributes,
// never from values baked into the script.
//
// The browser-side public API is window.wf.palette: open(id), close(id),
// isOpen(id) (id optional: the page's first palette), registered once by
// the first palette's x-init. Opening goes through dialog.min.js
// (window.tui.dialog.open) and then focuses the input; closing goes
// through window.tui.dialog.close and returns focus to the element that
// was focused when the palette opened.
package commandpalette

import (
	"strconv"
	"strings"

	"github.com/a-h/templ"
)

// DefaultID is the element id Palette uses when Props.ID is empty; it is
// also the dialog.min.js instance id and what Trigger opens.
const DefaultID = "wf-palette"

// Default texts.
const (
	DefaultPlaceholder = "Type a command or search…"
	DefaultShortcut    = "mod+k"
	DefaultEmpty       = "No matches"
)

// Item is one entry in the palette: a link with a label, optional hidden
// search keywords, an optional group heading, an optional icon and an
// optional shortcut hint shown right-aligned (display only; the palette
// does not bind it).
type Item struct {
	Label    string
	Href     string
	Keywords string // extra words the filter matches, never shown
	Group    string // heading the item is listed under; "" is the untitled first group
	Shortcut string // right-aligned hint such as "G then H"; not bound
	Icon     templ.Component
}

// Props configures Palette and Trigger. The zero value renders an empty
// palette with the defaults; set at least Items.
type Props struct {
	ID         string // defaults to DefaultID; the dialog id Trigger and the shortcut open
	Class      string
	Attributes templ.Attributes

	Items       []Item
	Placeholder string // defaults to DefaultPlaceholder
	// Shortcut is the key chord that opens the palette, as modifiers and a
	// key joined with "+": "mod+k" (default) where mod is Cmd on a Mac and
	// Ctrl elsewhere; "ctrl", "alt", "shift" and "meta" are also accepted.
	Shortcut string
	Empty    string // text shown when no item matches; defaults to DefaultEmpty
}

// withDefaults returns p with empty fields set to the defaults.
func (p Props) withDefaults() Props {
	if p.ID == "" {
		p.ID = DefaultID
	}
	if p.Placeholder == "" {
		p.Placeholder = DefaultPlaceholder
	}
	if p.Shortcut == "" {
		p.Shortcut = DefaultShortcut
	}
	if p.Empty == "" {
		p.Empty = DefaultEmpty
	}
	return p
}

// group is one heading and its items, in render order.
type group struct {
	Name  string
	Items []Item
}

// groupItems buckets items by Group in order of first appearance; the
// untitled group ("") is a group like any other and keeps its position.
func groupItems(items []Item) []group {
	var groups []group
	index := map[string]int{}
	for _, it := range items {
		i, ok := index[it.Group]
		if !ok {
			i = len(groups)
			index[it.Group] = i
			groups = append(groups, group{Name: it.Group})
		}
		groups[i].Items = append(groups[i].Items, it)
	}
	return groups
}

// inputID, listID, optionID and groupID derive the element ids the ARIA
// wiring uses from the palette id.
func inputID(id string) string         { return id + "-input" }
func listID(id string) string          { return id + "-list" }
func optionID(id string, n int) string { return id + "-option-" + strconv.Itoa(n) }
func groupID(id string, n int) string  { return id + "-group-" + strconv.Itoa(n) }

// Hint renders a shortcut spec as the text shown in Trigger: "mod+k" is
// "⌘K" (the Mac spelling; the listener itself uses Ctrl on other
// platforms), symbol modifiers (mod/meta ⌘, shift ⇧, alt ⌥) attach to the
// key with no separator, and a word modifier (Ctrl) joins with "+".
func Hint(shortcut string) string {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	if len(parts) == 0 {
		return ""
	}
	key := strings.ToUpper(parts[len(parts)-1])
	var b strings.Builder
	for _, m := range parts[:len(parts)-1] {
		switch m {
		case "mod", "meta", "cmd":
			b.WriteString("⌘")
		case "shift":
			b.WriteString("⇧")
		case "alt", "option":
			b.WriteString("⌥")
		case "ctrl", "control":
			b.WriteString("Ctrl+")
		}
	}
	b.WriteString(key)
	return b.String()
}

// keyShortcuts renders the spec as aria-keyshortcuts: "mod" expands to
// both Meta and Control alternatives, so "mod+k" is "Meta+K Control+K".
func keyShortcuts(shortcut string) string {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	if len(parts) == 0 {
		return ""
	}
	key := strings.ToUpper(parts[len(parts)-1])
	mods := parts[:len(parts)-1]
	one := func(mod string) string {
		var out []string
		for _, m := range mods {
			switch m {
			case "mod":
				out = append(out, mod)
			case "meta", "cmd":
				out = append(out, "Meta")
			case "ctrl", "control":
				out = append(out, "Control")
			case "shift":
				out = append(out, "Shift")
			case "alt", "option":
				out = append(out, "Alt")
			}
		}
		return strings.Join(append(out, key), "+")
	}
	for _, m := range mods {
		if m == "mod" {
			return one("Meta") + " " + one("Control")
		}
	}
	return one("")
}

// rootAttrs are the attributes on the dialog root: the palette marker the
// shortcut listener looks up, the shortcut spec, the Alpine state, then
// the caller's Attributes (which win on conflict).
func rootAttrs(p Props) templ.Attributes {
	attrs := templ.Attributes{
		"data-wf-palette":          "",
		"data-wf-palette-shortcut": p.Shortcut,
		"x-data":                   paletteData,
	}
	for k, v := range p.Attributes {
		attrs[k] = v
	}
	return attrs
}

// contentClass reshapes dialog.Content's centered panel. Below sm it is a
// top sheet: fixed to the top edge, spanning the width, rounded at the
// bottom only, at most 60dvh tall (so the list stays above a phone keyboard, which dvh does not account for) with the list scrolling inside, padded
// for the status bar's safe area. From sm up it is the dialog's usual
// centered panel at max-w-lg. The panel is a column (input row, list,
// empty text) with no padding of its own.
const contentClass = "flex flex-col gap-0 p-0 overflow-hidden " +
	"left-0 right-0 top-0 translate-x-0 translate-y-0 w-full max-w-none rounded-none rounded-b-xl max-h-[60dvh] pt-[env(safe-area-inset-top,0px)] " +
	"sm:left-[50%] sm:right-auto sm:top-[50%] sm:translate-x-[-50%] sm:translate-y-[-50%] sm:max-w-lg sm:rounded-lg sm:max-h-[80vh] sm:pt-0"

// paletteData is the Alpine x-data on the dialog root. It owns the query
// and the active row, filters the server-rendered items (label plus
// keywords, case-insensitive substring) by toggling the hidden attribute
// on items and groups, mirrors the active row into aria-selected /
// aria-activedescendant / data-active, and handles the input's keys.
// Its init registers the page-wide window.wf.palette API and the shortcut
// keydown listener exactly once (guarded on window.wf.palette, so a second
// palette or a boosted re-init adds nothing), then watches the panel's
// data-tui-dialog-open attribute so every close path (Escape, backdrop,
// close()) resets the query and returns focus to the opener.
const paletteData = `{
	q: '',
	active: 0,
	init() {
		// $root, not $el: a method called from a child's x-on handler sees that
		// child as $el (measured 2026-10-31: the filter never hid a row).
		const root = this.$root;
		if (!(window.wf && window.wf.palette)) {
			window.wf = window.wf || {};
			const api = window.wf.palette = {
				version: 1,
				opener: null,
				isMac: /Mac|iPhone|iPad|iPod/i.test((navigator.userAgentData && navigator.userAgentData.platform) || navigator.platform || ''),
				find(id) { return id ? document.getElementById(id) : document.querySelector('[data-wf-palette]') },
				dialog() { return window.tui && window.tui.dialog },
				open(id) {
					const el = api.find(id), d = api.dialog();
					if (!el || !d) return;
					api.opener = document.activeElement;
					d.open(el.id);
					const input = el.querySelector('[data-wf-palette-input]');
					if (input) { input.focus(); requestAnimationFrame(() => input.focus()) }
				},
				close(id) { const el = api.find(id), d = api.dialog(); if (el && d) d.close(el.id) },
				isOpen(id) { const el = api.find(id), d = api.dialog(); return !!(el && d && d.isOpen(el.id)) },
				matches(e, spec) {
					const parts = String(spec || 'mod+k').toLowerCase().split('+'), key = parts.pop();
					if (!e.key || e.key.toLowerCase() !== key) return false;
					const want = { ctrl: false, alt: false, shift: false, meta: false };
					for (const m of parts) { if (m === 'mod') want[api.isMac ? 'meta' : 'ctrl'] = true; else if (m in want) want[m] = true }
					return e.ctrlKey === want.ctrl && e.altKey === want.alt && e.shiftKey === want.shift && e.metaKey === want.meta
				}
			};
			document.addEventListener('keydown', e => {
				if (e.defaultPrevented || e.repeat) return;
				const el = api.find();
				if (!el || !api.matches(e, el.dataset.wfPaletteShortcut)) return;
				e.preventDefault();
				if (api.isOpen(el.id)) api.close(el.id); else api.open(el.id)
			})
		}
		const content = root.querySelector('[data-tui-dialog-content]');
		if (content) new MutationObserver(() => { if (content.getAttribute('data-tui-dialog-open') === 'false') this.closed() }).observe(content, { attributes: true, attributeFilter: ['data-tui-dialog-open'] });
		this.update()
	},
	items() { return Array.from(this.$root.querySelectorAll('[data-wf-palette-item]')) },
	visible() { return this.items().filter(li => !li.hidden) },
	update() {
		const q = this.q.trim().toLowerCase();
		this.items().forEach(li => { li.hidden = !!q && !((li.dataset.label || '') + ' ' + (li.dataset.keywords || '')).toLowerCase().includes(q) });
		this.$root.querySelectorAll('[data-wf-palette-group]').forEach(g => { g.hidden = !g.querySelector('[data-wf-palette-item]:not([hidden])') });
		const vis = this.visible();
		if (this.active >= vis.length) this.active = Math.max(0, vis.length - 1);
		let current = '';
		this.items().forEach(li => {
			const o = li.querySelector('[role=option]');
			if (!o) return;
			const on = !li.hidden && vis[this.active] === li;
			o.setAttribute('aria-selected', on ? 'true' : 'false');
			o.dataset.active = on ? 'true' : 'false';
			if (on) current = o.id
		});
		const input = this.$root.querySelector('[data-wf-palette-input]');
		if (input) input.setAttribute('aria-activedescendant', current);
		const empty = this.$root.querySelector('[data-wf-palette-empty]');
		if (empty) empty.hidden = vis.length > 0
	},
	changed(e) { if (e && e.target) this.q = e.target.value; this.active = 0; this.update() },
	move(i, n) {
		if (!n) return;
		this.active = ((i % n) + n) % n;
		this.update();
		const li = this.visible()[this.active];
		if (li && li.scrollIntoView) li.scrollIntoView({ block: 'nearest' })
	},
	key(e) {
		const n = this.visible().length;
		switch (e.key) {
			case 'ArrowDown': e.preventDefault(); this.move(this.active + 1, n); break;
			case 'ArrowUp': e.preventDefault(); this.move(this.active - 1, n); break;
			case 'Home': e.preventDefault(); this.move(0, n); break;
			case 'End': e.preventDefault(); this.move(n - 1, n); break;
			case 'Enter': {
				e.preventDefault();
				const li = this.visible()[this.active], o = li && li.querySelector('[role=option]');
				if (o) o.click();
				break
			}
			case 'Escape': e.preventDefault(); e.stopPropagation(); window.wf.palette.close(this.$root.id); break
		}
	},
	hover(e) {
		const li = e.target.closest && e.target.closest('[data-wf-palette-item]');
		if (!li) return;
		const i = this.visible().indexOf(li);
		if (i >= 0 && i !== this.active) { this.active = i; this.update() }
	},
	closed() {
		this.q = ''; this.active = 0; this.update();
		const api = window.wf && window.wf.palette, o = api && api.opener;
		if (api) api.opener = null;
		if (o && o !== document.body && document.body.contains(o) && typeof o.focus === 'function') o.focus()
	}
}`
