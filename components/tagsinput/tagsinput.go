// Package tagsinput renders a free-text tag editor: a framed box holding
// one chip per tag (each with a remove button) and a text input that adds
// a tag on Enter, on comma and on blur; Backspace on the empty input
// removes the last chip. The tags are submitted as one plain form field,
// <input type="hidden" name={Name}>, whose value is the tags joined with
// Separator ("," by default), so the handler reads them with strings.Split:
//
//	var tags []string
//	for _, t := range strings.Split(r.FormValue("tags"), ",") {
//		if t = strings.TrimSpace(t); t != "" {
//			tags = append(tags, t)
//		}
//	}
//
// (strings.Split of "" is []string{""}, which the empty check drops.) The
// handler re-renders a failed POST with Values set to the parsed tags and
// the chips come back; nothing else is needed.
//
// First paint without JavaScript is complete: the chips are server-rendered
// from Values, the hidden field already carries the joined value, the count
// is written, and the text input is readonly (placeholder "Limit reached",
// still focusable so a phone's keyboard stays open) when Values is at
// MaxTags.
// Alpine (x-data on the root, inline in this component) then takes over:
// its init() reads the tag list from the rendered <li data-tag> chips,
// removes them and re-renders the same chips from its own state, so the
// server markup is the single source of the initial state and nothing is
// written twice. Pasted "a, b, c" is split on Separator into three tags;
// duplicates are rejected case-insensitively; MaxTags disables the input at
// the cap and the count reads "n/max". Only the user's actions change the
// chip row: it grows when a tag is added and shrinks when one is removed.
//
// Phone width (390px): the chips wrap inside the frame and the text input
// takes the remaining width of the last row (it drops to its own full row
// when a long chip leaves too little space). The on-screen keyboard's
// Enter/Return key adds the tag because the component listens to keydown
// Enter; a comma on the keyboard adds too. Each chip's remove button is a
// 24px target inside the chip; the input itself is at least 28px tall.
// Suggestions render a <datalist>, which phones show as autocomplete
// suggestions above the keyboard.
package tagsinput

import (
	"fmt"
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/utils"
)

// DefaultSeparator joins the tags in the hidden field when Props.Separator
// is empty.
const DefaultSeparator = ","

// Props configures a TagsInput. Name is required for the value to submit;
// everything else is optional with the defaults noted per field.
type Props struct {
	// ID is the text input's id (what a <label for> points at; the
	// control that receives focus and aria-describedby). The hint is
	// ID+"-hint", the chip list ID+"-list", the datalist ID+"-options" and
	// the root ID+"-root". Defaults to utils.RandomID(); set it for stable
	// markup.
	ID    string
	Class string
	// Attributes spread onto the text input (as input.Props.Attributes
	// does), so a form field's aria-describedby and aria-invalid land on
	// the focusable control. An aria-describedby here is merged with the
	// component's own hint id.
	Attributes templ.Attributes

	// Name is the hidden field's name; the submitted value is the tags
	// joined with Separator.
	Name string
	// Values are the current tags, rendered as chips in order.
	Values []string
	// Placeholder is the text input's placeholder.
	Placeholder string
	// MaxTags caps the number of tags; 0 means no cap. At the cap the text
	// input turns readonly with the placeholder "Limit reached" and the
	// hint shows "n/max".
	MaxTags int
	// Disabled disables the input, the chip remove buttons and the hidden
	// field, so a disabled tags input submits nothing (as a disabled
	// <input> does).
	Disabled bool
	// Separator joins the tags in the hidden field and splits pasted text;
	// "" means DefaultSeparator. A tag never contains the separator.
	Separator string
	// Suggestions, when set, render a <datalist> the text input offers as
	// completions; picking one adds it as a tag.
	Suggestions []string
	// HasError marks the frame and the input invalid (destructive border),
	// as input.Props.HasError does.
	HasError bool
}

// resolve fills the defaults: a random ID and the default separator.
func (p Props) resolve() Props {
	if p.ID == "" {
		p.ID = utils.RandomID()
	}
	if p.Separator == "" {
		p.Separator = DefaultSeparator
	}
	return p
}

// HintID is the id of the hint paragraph (instructions and the n/max
// count), which the text input's aria-describedby names.
func (p Props) HintID() string { return p.ID + "-hint" }

func (p Props) rootID() string    { return p.ID + "-root" }
func (p Props) listID() string    { return p.ID + "-list" }
func (p Props) optionsID() string { return p.ID + "-options" }

// joined is the hidden field's initial value: Values joined with the
// separator.
func (p Props) joined() string {
	return strings.Join(p.Values, p.Separator)
}

// atMax reports whether Values already holds MaxTags tags.
func (p Props) atMax() bool {
	return p.MaxTags > 0 && len(p.Values) >= p.MaxTags
}

// count is the "n/max" text shown when MaxTags is set.
func (p Props) count() string {
	return fmt.Sprintf("%d/%d", len(p.Values), p.MaxTags)
}

// maxAttr is the data-max value Alpine reads ("0" = no cap).
func (p Props) maxAttr() string { return fmt.Sprint(p.MaxTags) }

// describedBy is the text input's aria-describedby: the hint id, followed
// by whatever the caller put in Attributes["aria-describedby"].
func (p Props) describedBy() string {
	if v, ok := p.Attributes["aria-describedby"]; ok {
		if s, _ := v.(string); s != "" {
			return p.HintID() + " " + s
		}
	}
	return p.HintID()
}

// inputAttrs are the attributes spread onto the text input: the Alpine
// wiring first, then the caller's Attributes (minus aria-describedby,
// which describedBy merged), so a caller can still override x-on hooks
// deliberately. Values typed by the user never reach these attributes;
// everything Alpine needs from the server is read from data attributes
// and the rendered chips, so no JS is built from strings here.
func (p Props) inputAttrs() templ.Attributes {
	own := templ.Attributes{
		"x-model":         "text",
		"x-on:keydown":    "onKey($event)",
		"x-on:blur":       "commit()",
		"x-on:change":     "commit()",
		"x-bind:disabled": "disabled",
		// At the cap the input stays focusable (readonly), so adding the
		// last tag does not drop focus and close a phone's keyboard; the
		// placeholder explains the state.
		"x-bind:readonly":    "atMax",
		"x-bind:placeholder": "atMax ? 'Limit reached' : placeholder",
		"aria-describedby":   p.describedBy(),
		"autocomplete":       "off",
	}
	if len(p.Suggestions) > 0 {
		own["list"] = p.optionsID()
	}
	caller := templ.Attributes{}
	for k, v := range p.Attributes {
		if k == "aria-describedby" {
			continue
		}
		caller[k] = v
	}
	return utils.MergeAttributes(own, caller)
}

// frameClass is the box around the chips and the input: the same border,
// radius and focus ring as a text input, with the ring moved to
// focus-within since the real focus is on the inner input.
func (p Props) frameClass() string {
	return utils.TwMerge(
		"flex flex-wrap items-center gap-1.5 rounded-md border border-input bg-transparent p-1.5 shadow-xs transition-[color,box-shadow]",
		"dark:bg-input/30",
		"focus-within:border-ring focus-within:ring-ring/50 focus-within:ring-[3px]",
		utils.If(p.HasError, "border-destructive ring-destructive/20 dark:ring-destructive/40"),
		utils.If(p.Disabled, "cursor-not-allowed opacity-50"),
	)
}

// textInputClass strips the inner input's own frame (the box has one) and
// makes it compact so a chip row and the input sit on one line.
const textInputClass = "h-7 min-w-32 flex-1 border-0 bg-transparent px-1 shadow-none focus-visible:border-0 focus-visible:ring-0 dark:bg-transparent"

// alpineData is the x-data of the root. init() seeds the state from the
// server-rendered markup: data-max/data-sep/data-disabled on the root and
// one <li data-tag> per chip, which it removes so the x-for below renders
// the same chips from state (same tick as the walk: no paint in between).
// commit() splits the text on the separator (so a paste adds several),
// trims, drops empties, rejects case-insensitive duplicates and stops at
// the cap; onKey handles Enter and comma (add) and Backspace on an empty
// input (remove the last chip).
const alpineData = `{
	tags: [], text: '', max: 0, sep: ',', disabled: false, placeholder: '',
	init() {
		const root = this.$el;
		this.max = Number(root.dataset.max) || 0;
		this.sep = root.dataset.sep || ',';
		this.disabled = root.hasAttribute('data-disabled');
		this.placeholder = root.dataset.placeholder || '';
		const chips = root.querySelectorAll('li[data-tag]');
		this.tags = Array.from(chips, li => li.dataset.tag);
		chips.forEach(li => li.remove());
	},
	get atMax() { return this.max > 0 && this.tags.length >= this.max },
	has(v) { const l = v.toLowerCase(); return this.tags.some(t => t.toLowerCase() === l) },
	commit() {
		if (this.disabled) return;
		for (const part of this.text.split(this.sep)) {
			const v = part.trim();
			if (v === '' || this.atMax || this.has(v)) continue;
			this.tags.push(v);
		}
		this.text = '';
	},
	removeAt(i) { if (!this.disabled) this.tags.splice(i, 1) },
	onKey(e) {
		if (e.key === 'Enter' || e.key === ',') { e.preventDefault(); this.commit(); return }
		if (e.key === 'Backspace' && this.text === '' && this.tags.length > 0) { e.preventDefault(); this.tags.pop() }
	}
}`
