// Package tagsinput renders a free-text tag editor: a framed box holding
// one chip per tag (each with a remove button) and a text input that adds
// a tag on Enter, on the separator key (comma by default) and on blur;
// Backspace on the empty input removes the last chip. The tags are
// submitted as one hidden form field named Name, joined with Separator
// ("," by default), so the handler reads them with strings.Split:
//
//	var tags []string
//	for _, t := range strings.Split(r.FormValue("tags"), ",") {
//		if t = strings.TrimSpace(t); t != "" {
//			tags = append(tags, t)
//		}
//	}
//
// To re-render a failed POST, set Values to the parsed tags.
//
// The first paint is complete without JavaScript: chips, the hidden value
// and the count are server-rendered from Values. Alpine then reads the
// initial tags from the rendered chips and renders them from its own state,
// so the server markup is the single source of the initial state. Pasted
// "a, b, c" becomes three tags, and so does a pasted list with one tag per
// line (a text input would otherwise drop the newlines); duplicates are rejected case-insensitively;
// at MaxTags the input turns readonly (still focusable, so a phone keyboard
// stays open). Suggestions render a <datalist>.
//
// The chips are a real list: a flex <ul role="list" aria-label="Tags"> with
// one <li> per tag, and the text input sits after the list, outside it, so
// a screen reader announces "Tags, list, 3 items". The list is hidden while
// there are no tags. On phones (below md) each chip is 44px tall so its
// remove button is a 44px target; from md up it is a 24px button.
package tagsinput

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/utils"
)

// DefaultSeparator joins the tags in the hidden field when Props.Separator
// is empty.
const DefaultSeparator = ","

// Props configures a TagsInput. Name is required for the value to submit;
// everything else is optional with the defaults noted per field.
type Props struct {
	// ID is the text input's id, the target of a <label for>. It also
	// derives the ids of the hint, chip list, datalist and root. Defaults
	// to a random id; set it for stable markup.
	ID    string
	Class string
	// Attributes spread onto the text input. An aria-describedby here is
	// merged with the component's own hint id.
	Attributes templ.Attributes

	// Name is the hidden field's name; the submitted value is the tags
	// joined with Separator.
	Name string
	// Values are the current tags, rendered as chips in order after
	// trimming, dropping empties and case-insensitive duplicates.
	Values []string
	// Placeholder is the text input's placeholder.
	Placeholder string
	// MaxTags caps the number of tags; 0 means no cap. At the cap the text
	// input turns readonly with the placeholder "Limit reached" and the
	// hint shows "n/max".
	MaxTags int
	// Disabled disables the input, the remove buttons and the hidden
	// field, so nothing is submitted.
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

// resolve fills the defaults and normalizes Values the same way the client
// normalizes typed text, so the server-rendered chips, hidden value and
// count agree with what Alpine would produce.
func (p Props) resolve() Props {
	if p.ID == "" {
		p.ID = utils.RandomID()
	}
	if p.Separator == "" {
		p.Separator = DefaultSeparator
	}
	p.Values = normalize(p.Values)
	return p
}

// normalize trims, drops empties and dedupes case-insensitively, keeping
// the first spelling and the order of first appearance. nil in, nil out.
func normalize(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, v)
	}
	return out
}

// hintText is the sr-only instruction line: it names the key that adds a
// tag (Enter, plus the separator when it is one character the user can
// type) and Backspace.
func (p Props) hintText() string {
	switch {
	case p.Separator == DefaultSeparator:
		return "Enter or comma adds a tag; Backspace on an empty input removes the last one."
	case utf8.RuneCountInString(p.Separator) == 1:
		return "Enter or " + p.Separator + " adds a tag; Backspace on an empty input removes the last one."
	default:
		return "Enter adds a tag; Backspace on an empty input removes the last one."
	}
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
// wiring, then the caller's Attributes (minus aria-describedby, which
// describedBy merged). No JavaScript is built from strings here; Alpine
// reads server state from data attributes and the rendered chips.
func (p Props) inputAttrs() templ.Attributes {
	own := templ.Attributes{
		"x-model":         "text",
		"x-on:keydown":    "onKey($event)",
		"x-on:blur":       "commit()",
		"x-on:change":     "commit()",
		"x-on:paste":      "onPaste($event)",
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
// root's data attributes and the server-rendered chips, then removes those
// chips so the x-for renders them from state. A multi-character Separator
// adds a tag on Enter only.
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
	onPaste(e) {
		const text = (e.clipboardData || window.clipboardData)?.getData('text') || '';
		if (!/[\r\n]/.test(text)) return;
		e.preventDefault();
		const el = e.target;
		const at = el.selectionStart ?? this.text.length, end = el.selectionEnd ?? at;
		this.text = this.text.slice(0, at) + text.split(/\r?\n|\r/).join(this.sep) + this.text.slice(end);
		this.commit();
	},
	onKey(e) {
		if (e.key === 'Enter' || (this.sep.length === 1 && e.key === this.sep)) { e.preventDefault(); this.commit(); return }
		if (e.key === 'Backspace' && this.text === '' && this.tags.length > 0) { e.preventDefault(); this.tags.pop() }
	}
}`
