// Package keyvalueeditor renders editable key/value rows: one row per Row
// (a key input, a value input and a remove button) under an "Add row"
// button that appends an empty row. Because the key is itself editable the
// rows are not submitted as name[key]=value (the key would have to be known
// when the form renders); they are submitted as two parallel fields
// repeated once per row, in row order:
//
//	{Name}.key=HOST   {Name}.value=db.local
//	{Name}.key=PORT   {Name}.value=5432
//
// Parse zips them back into []Row and drops rows whose key is blank, so a
// row the user added and left empty is not an error:
//
//	func save(w http.ResponseWriter, r *http.Request) {
//		if err := r.ParseForm(); err != nil { /* 400 */ }
//		rows := keyvalueeditor.Parse(r.Form, "env") // []Row{{Key: "HOST", Value: "db.local"}, ...}
//		// ... re-render with Props{Rows: rows} on a validation failure
//	}
//
// The rows are server-rendered, so a plain submit works without JavaScript
// (the Add row and remove buttons need Alpine). Alpine's init() reads the
// rows from that markup and re-renders them from its own state. A disabled
// editor submits nothing, as disabled inputs do. On phones each row stacks
// key over value; from sm up a row is one line.
//
// Needs JavaScript: Add row and the remove buttons do nothing, so a new row
// cannot be added; existing rows can be edited, and clearing a key drops
// that row on save.
package keyvalueeditor

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/utils"
)

// DefaultAddLabel is the Add row button's text when Props.AddLabel is empty.
const DefaultAddLabel = "Add row"

// Row is one key/value pair.
type Row struct {
	Key, Value string
}

// Props configures a KeyValueEditor. Name is required for the rows to
// submit; everything else is optional with the defaults noted per field.
type Props struct {
	// ID is the root's id; the key input of row i is ID+"-key-"+i, the
	// value input ID+"-value-"+i, where i is the row's position in Rows.
	// In the browser a row keeps its id when another row is removed, and
	// a row the user adds takes the next unused number, so an id is never
	// reused within one page. Random when empty.
	ID         string
	Class      string
	Attributes templ.Attributes // spread on the root

	// Name prefixes the field names: {Name}.key and {Name}.value.
	Name string
	// Rows are the current pairs, rendered in order.
	Rows []Row
	// KeyPlaceholder and ValuePlaceholder are the inputs' placeholders
	// ("Key" and "Value" when empty).
	KeyPlaceholder, ValuePlaceholder string
	// MaxRows caps the number of rows; 0 means no cap. At the cap Add row
	// is disabled.
	MaxRows int
	// AddLabel is the Add row button's text; "" means DefaultAddLabel.
	AddLabel string
	// Disabled disables every input and button.
	Disabled bool
}

// resolve fills the defaults.
func (p Props) resolve() Props {
	if p.ID == "" {
		p.ID = utils.RandomID()
	}
	if p.AddLabel == "" {
		p.AddLabel = DefaultAddLabel
	}
	if p.KeyPlaceholder == "" {
		p.KeyPlaceholder = "Key"
	}
	if p.ValuePlaceholder == "" {
		p.ValuePlaceholder = "Value"
	}
	return p
}

// ControlID is the id of the first row's key input (ID+"-key-0"), the
// control a <label for> points at. When Rows is empty, or once the user
// removes that first row, no element carries it.
func (p Props) ControlID() string { return p.keyID(0) }

// KeyName and ValueName are the submitted field names, {Name}.key and
// {Name}.value, repeated once per row.
func (p Props) KeyName() string   { return p.Name + ".key" }
func (p Props) ValueName() string { return p.Name + ".value" }

func (p Props) keyID(i int) string   { return fmt.Sprintf("%s-key-%d", p.ID, i) }
func (p Props) valueID(i int) string { return fmt.Sprintf("%s-value-%d", p.ID, i) }
func (p Props) addID() string        { return p.ID + "-add" }

// removeLabel is the accessible name of row i's remove button (1-based).
func (p Props) removeLabel(i int) string { return fmt.Sprintf("Remove row %d", i+1) }

// atMax reports whether Rows already holds MaxRows rows.
func (p Props) atMax() bool {
	return p.MaxRows > 0 && len(p.Rows) >= p.MaxRows
}

// maxAttr is the data-max value Alpine reads ("0" = no cap).
func (p Props) maxAttr() string { return fmt.Sprint(p.MaxRows) }

// rowClass is one row: stacked on phones inside a light frame, one line
// from sm up.
const rowClass = "grid grid-cols-1 items-center gap-2 rounded-md border border-border p-2 sm:grid-cols-[1fr_1fr_auto] sm:border-0 sm:p-0"

// Parse zips form[name+".key"] and form[name+".value"] into rows, dropping
// rows whose trimmed key is blank. Keys are trimmed; values are kept as
// typed.
func Parse(form url.Values, name string) []Row {
	keys := form[name+".key"]
	values := form[name+".value"]
	rows := make([]Row, 0, len(keys))
	for i, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		v := ""
		if i < len(values) {
			v = values[i]
		}
		rows = append(rows, Row{Key: k, Value: v})
	}
	return rows
}

// alpineData is the x-data of the root. init() seeds the state from the
// server-rendered [data-row] elements and removes them, so the x-for
// renders the same rows in the same tick with no paint in between. Each
// row gets a stable id, numbered in Rows order so it matches the
// server-rendered input ids, used for :key and for the inputs' DOM ids.
const alpineData = `{
	rows: [], next: 0, max: 0, disabled: false,
	init() {
		const root = this.$el;
		this.max = Number(root.dataset.max) || 0;
		this.disabled = root.hasAttribute('data-disabled');
		const els = root.querySelectorAll('[data-row]');
		this.rows = Array.from(els, el => ({
			id: this.next++,
			key: el.querySelector('[data-key]').value,
			value: el.querySelector('[data-value]').value
		}));
		els.forEach(el => el.remove());
	},
	get atMax() { return this.max > 0 && this.rows.length >= this.max },
	add() {
		if (this.disabled || this.atMax) return;
		this.rows.push({ id: this.next++, key: '', value: '' });
		this.$nextTick(() => { const keys = this.$el.querySelectorAll('[data-key]'); keys[keys.length - 1]?.focus() });
	},
	removeAt(i) { if (!this.disabled) this.rows.splice(i, 1) }
}`
