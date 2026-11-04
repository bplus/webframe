// Package filedrop renders a file picker as a drop zone: a dashed <label>
// wrapping a visually hidden <input type="file">. A click anywhere on the
// zone opens the file picker, and a drop puts the dragged files into the
// same input (via a DataTransfer, in Alpine). Chosen files are listed under
// the zone with name, size, optional image thumbnail and a remove button.
// Files over MaxSize are refused with an inline alert; the rest of the pick
// stays selected.
//
// Form contract: the input is a plain file input named Name, so the form
// must be multipart (form.Props.Multipart or formpage.Props.Multipart). The
// handler reads r.FormFile(name) or, with Multiple,
// r.MultipartForm.File[name]. MaxSize is a client-side courtesy only; the
// server must enforce its own limit. A file input cannot be refilled, so a
// page re-rendered after a failed POST shows the zone empty; set
// ChooseAgain on that render and a line under the zone asks the user to
// choose the file again.
//
// Ids: ID names the <input type="file"> (the control a label points at and
// an error link focuses); the Alpine root is ID+"-root".
package filedrop

import (
	"fmt"
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/utils"
)

// Props configures FileDrop. Name is the submitted field. Attributes go on
// the hidden <input> (the focusable control), not the root.
type Props struct {
	// ID is the file input's id; the root is ID+"-root". When empty it is
	// "filedrop-"+Name, or a random id when Name is empty too.
	ID         string
	Class      string
	Attributes templ.Attributes

	Name string
	// Accept is the accept attribute, e.g. "image/*,.pdf".
	Accept string
	// Multiple allows more than one file. A new pick or drop replaces the
	// selection, as the native input does; a remove button drops one file.
	Multiple bool
	// MaxSize in bytes refuses larger files client-side with a message;
	// 0 means no client-side limit.
	MaxSize int64
	// Preview shows a thumbnail for image files in the list.
	Preview bool
	// Label is the zone's main line; Hint the muted line under it, which
	// defaults to the MaxSize when set.
	Label, Hint string
	Required    bool
	Disabled    bool
	// ChooseAgain shows "Choose the file again" ("Choose the files again"
	// with Multiple) under the zone. Set it when re-rendering a form after
	// a failed POST: the browser cannot refill a file input, so a file the
	// user picked before is gone.
	ChooseAgain bool
}

// baseID is the id stem: ID, else "filedrop-"+Name, else a fresh random
// id (see resolve).
func (p Props) baseID() string {
	if p.ID != "" {
		return p.ID
	}
	if p.Name != "" {
		return "filedrop-" + p.Name
	}
	return utils.RandomID()
}

// resolve pins a random id stem into ID when there is neither ID nor
// Name, so the root and the input agree within one render.
func (p Props) resolve() Props {
	if p.ID == "" && p.Name == "" {
		p.ID = utils.RandomID()
	}
	return p
}

// ControlID is the id of the hidden <input type="file">, for a label's for
// attribute or aria-describedby. It is the resolved ID.
func (p Props) ControlID() string {
	return p.baseID()
}

// rootID is the id of the Alpine root that wraps the zone and the list.
func (p Props) rootID() string {
	return p.baseID() + "-root"
}

// againID is the id of the ChooseAgain line.
func (p Props) againID() string {
	return p.baseID() + "-again"
}

// inputAttrs are the caller's Attributes for the file input, with the
// ChooseAgain line's id added in front of any aria-describedby, so a
// screen reader hears the line when the input takes focus.
func (p Props) inputAttrs() templ.Attributes {
	if !p.ChooseAgain {
		return p.Attributes
	}
	out := templ.Attributes{}
	for k, v := range p.Attributes {
		out[k] = v
	}
	desc := p.againID()
	if s, _ := p.Attributes["aria-describedby"].(string); s != "" {
		desc += " " + s
	}
	out["aria-describedby"] = desc
	return out
}

// againText is the ChooseAgain line.
func (p Props) againText() string {
	if p.Multiple {
		return "Choose the files again"
	}
	return "Choose the file again"
}

func (p Props) label() string {
	if p.Label != "" {
		return p.Label
	}
	if p.Multiple {
		return "Drop files here or choose"
	}
	return "Drop a file here or choose"
}

func (p Props) hint() string {
	if p.Hint != "" {
		return p.Hint
	}
	if p.MaxSize > 0 {
		if p.Multiple {
			return "Up to " + FormatSize(p.MaxSize) + " each"
		}
		return "Up to " + FormatSize(p.MaxSize)
	}
	return ""
}

// FormatSize writes n bytes for people: "512 B", "1.5 KB", "2 MB" (binary
// units, one decimal, a trailing ".0" dropped). It matches the client-side
// list's formatting.
func FormatSize(n int64) string {
	const (
		kb = 1 << 10
		mb = 1 << 20
		gb = 1 << 30
	)
	switch {
	case n < kb:
		return fmt.Sprintf("%d B", n)
	case n < mb:
		return trimZero(fmt.Sprintf("%.1f", float64(n)/kb)) + " KB"
	case n < gb:
		return trimZero(fmt.Sprintf("%.1f", float64(n)/mb)) + " MB"
	default:
		return trimZero(fmt.Sprintf("%.1f", float64(n)/gb)) + " GB"
	}
}

func trimZero(s string) string {
	return strings.TrimSuffix(s, ".0")
}

// alpineScript holds the x-data methods shared by every FileDrop;
// alpineData prepends the per-instance values. accept takes a pick or drop,
// refuses files over max and assigns the rest to the input through a
// DataTransfer. sync rebuilds the list and its thumbnail object URLs.
// destroy, called by Alpine when the root leaves the DOM (a 422 swap, say),
// revokes the object URLs so the thumbnails do not leak.
const alpineScript = `accept(list) { this.error = ''; const picked = Array.from(list || []); if (picked.length === 0) return; const dt = new DataTransfer(); const refused = []; for (const f of picked) { if (this.max > 0 && f.size > this.max) { refused.push(f.name); continue; } if (!this.multiple && dt.items.length > 0) break; dt.items.add(f); } this.$refs.input.files = dt.files; this.sync(); if (refused.length > 0) { this.error = refused.join(', ') + (refused.length > 1 ? ' are larger than ' + this.maxLabel + ' and were not added' : ' is larger than ' + this.maxLabel + ' and was not added'); } }, ` +
	`drop(e) { this.over = false; if (this.disabled) return; this.accept(e.dataTransfer.files); }, ` +
	`sync() { for (const f of this.files) { if (f.url) URL.revokeObjectURL(f.url); } this.files = Array.from(this.$refs.input.files).map(f => ({ name: f.name, size: f.size, url: this.preview && f.type.startsWith('image/') ? URL.createObjectURL(f) : '' })); }, ` +
	`remove(i) { const dt = new DataTransfer(); Array.from(this.$refs.input.files).forEach((f, j) => { if (j !== i) dt.items.add(f); }); this.$refs.input.files = dt.files; this.sync(); }, ` +
	`size(n) { if (n < 1024) return n + ' B'; if (n < 1048576) return (n / 1024).toFixed(1).replace(/\.0$/, '') + ' KB'; if (n < 1073741824) return (n / 1048576).toFixed(1).replace(/\.0$/, '') + ' MB'; return (n / 1073741824).toFixed(1).replace(/\.0$/, '') + ' GB'; }, ` +
	`destroy() { for (const f of this.files) { if (f.url) URL.revokeObjectURL(f.url); } this.files = []; }`

// alpineData is the root's x-data: the instance values (max, multiple,
// preview, disabled, the formatted limit) and the shared methods.
func (p Props) alpineData() string {
	return fmt.Sprintf("{ files: [], over: false, error: '', max: %d, multiple: %t, preview: %t, disabled: %t, maxLabel: '%s', %s }",
		p.MaxSize, p.Multiple, p.Preview, p.Disabled, utils.EscapeJS(FormatSize(p.MaxSize)), alpineScript)
}
