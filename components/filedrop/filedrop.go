// Package filedrop renders a file picker as a drop zone: a dashed <label>
// wrapping a visually hidden <input type="file">, so a click or tap anywhere
// on the zone opens the platform's file picker, and a drop on desktop puts
// the dragged files into the same input (Alpine assigns them through a
// DataTransfer). The chosen files are listed under the zone with name, size,
// an optional image thumbnail (an object URL, revoked when the file is
// removed) and a remove button. Files over MaxSize are refused with an inline
// role="alert" message in the destructive tone; the others in the same pick
// stay selected.
//
// Form contract: the input is a plain file input named Name, so the form must
// be multipart (form.Props.Multipart or formpage.Props.Multipart; without it
// the browser submits only the file names). The handler reads
// r.FormFile(name) or, with Multiple, r.MultipartForm.File[name]. MaxSize is
// a client-side courtesy only; the server must still enforce its own limit.
// A file input cannot be refilled from a value: a page re-rendered after a
// failed POST shows the zone empty.
//
// Phone width (390px): the zone is a full-width block and a tap opens the
// native picker (on iOS: photo library, camera or Files); drag and drop is
// desktop-only. The file list stacks one file per row with the name
// truncated and the size and remove button kept at the row's end.
package filedrop

import (
	"fmt"
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/utils"
)

// Props configures FileDrop. Name is the submitted field; everything else is
// optional. Class merges after the root's built-in classes; Attributes
// spread onto the hidden <input> (the focusable control), after its
// built-in attributes.
type Props struct {
	// ID is the root's id. ControlID derives the input's id from it (or from
	// Name).
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
	// Label is the zone's main line ("Drop files here or choose" by
	// default); Hint the muted line under it (by default the MaxSize when
	// set).
	Label, Hint string
	Required    bool
	Disabled    bool
}

// baseID is the id stem: ID, else "filedrop-"+Name, else "filedrop".
func (p Props) baseID() string {
	if p.ID != "" {
		return p.ID
	}
	if p.Name != "" {
		return "filedrop-" + p.Name
	}
	return "filedrop"
}

// ControlID is the id of the hidden <input type="file">, the control a
// label's for attribute and aria-describedby point at: baseID()+"-input".
func (p Props) ControlID() string {
	return p.baseID() + "-input"
}

// label returns the zone's main line.
func (p Props) label() string {
	if p.Label != "" {
		return p.Label
	}
	if p.Multiple {
		return "Drop files here or choose"
	}
	return "Drop a file here or choose"
}

// hint returns the muted line: Hint, else "Up to <MaxSize>" when a limit is
// set, else "".
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

// FormatSize writes n bytes for people: "512 B", "1.5 KB", "2 MB", "1.2 GB"
// (binary units, one decimal, a trailing ".0" dropped). The client-side list
// formats sizes the same way.
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

// alpineScript is the x-data body shared by every FileDrop; the per-instance
// values are prepended by alpineData. Methods:
//
//   - accept(list): the files from a pick or a drop; refuses the ones over
//     max, keeps the first one only unless multiple, assigns the rest to the
//     input through a DataTransfer and lists them; sets error when any was
//     refused (the message names the files and the limit).
//   - drop(event): the drop handler; ignored when disabled.
//   - sync(): rebuilds the list from the input's files, creating object URLs
//     for image thumbnails when preview is on and revoking the old ones.
//   - remove(i): drops the i-th file from the input and relists.
//   - size(n): formats bytes like FormatSize.
const alpineScript = `accept(list) { this.error = ''; const picked = Array.from(list || []); if (picked.length === 0) return; const dt = new DataTransfer(); const refused = []; for (const f of picked) { if (this.max > 0 && f.size > this.max) { refused.push(f.name); continue; } if (!this.multiple && dt.items.length > 0) break; dt.items.add(f); } this.$refs.input.files = dt.files; this.sync(); if (refused.length > 0) { this.error = refused.join(', ') + (refused.length > 1 ? ' are larger than ' + this.maxLabel + ' and were not added' : ' is larger than ' + this.maxLabel + ' and was not added'); } }, ` +
	`drop(e) { this.over = false; if (this.disabled) return; this.accept(e.dataTransfer.files); }, ` +
	`sync() { for (const f of this.files) { if (f.url) URL.revokeObjectURL(f.url); } this.files = Array.from(this.$refs.input.files).map(f => ({ name: f.name, size: f.size, url: this.preview && f.type.startsWith('image/') ? URL.createObjectURL(f) : '' })); }, ` +
	`remove(i) { const dt = new DataTransfer(); Array.from(this.$refs.input.files).forEach((f, j) => { if (j !== i) dt.items.add(f); }); this.$refs.input.files = dt.files; this.sync(); }, ` +
	`size(n) { if (n < 1024) return n + ' B'; if (n < 1048576) return (n / 1024).toFixed(1).replace(/\.0$/, '') + ' KB'; if (n < 1073741824) return (n / 1048576).toFixed(1).replace(/\.0$/, '') + ' MB'; return (n / 1073741824).toFixed(1).replace(/\.0$/, '') + ' GB'; }`

// alpineData is the root's x-data: the instance values (max, multiple,
// preview, disabled, the formatted limit) and the shared methods.
func (p Props) alpineData() string {
	return fmt.Sprintf("{ files: [], over: false, error: '', max: %d, multiple: %t, preview: %t, disabled: %t, maxLabel: '%s', %s }",
		p.MaxSize, p.Multiple, p.Preview, p.Disabled, utils.EscapeJS(FormatSize(p.MaxSize)), alpineScript)
}
