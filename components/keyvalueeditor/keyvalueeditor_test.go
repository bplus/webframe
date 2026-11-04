package keyvalueeditor

import (
	"bytes"
	"context"
	"net/url"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func render(t *testing.T, c templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	if err := c.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	return buf.String()
}

func mustContain(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func mustNotContain(t *testing.T, out string, nots ...string) {
	t.Helper()
	for _, not := range nots {
		if strings.Contains(out, not) {
			t.Errorf("output has unexpected %q:\n%s", not, out)
		}
	}
}

// disabledAttr matches a bare disabled attribute (not the disabled: class
// variants).
var disabledAttr = regexp.MustCompile(`\sdisabled(\s|>)`)

func isDisabled(tag string) bool { return disabledAttr.MatchString(tag) }

// attrsOf returns the opening tag that contains marker.
func attrsOf(t *testing.T, out, marker string) string {
	t.Helper()
	i := strings.Index(out, marker)
	if i < 0 {
		t.Fatalf("marker %q missing:\n%s", marker, out)
	}
	start := strings.LastIndex(out[:i], "<")
	end := strings.Index(out[i:], ">")
	return out[start : i+end+1]
}

func TestKeyValueEditor_RowsAndNames(t *testing.T) {
	out := render(t, KeyValueEditor(Props{ID: "env", Name: "env", Rows: []Row{{"HOST", "db.local"}, {"PORT", "5432"}}}))
	mustContain(t, out,
		`<div id="env" class="w-full" x-data="{`,
		`data-max="0"`,
		`<div data-row role="listitem"`,
		`<template x-for="(row, i) in rows" :key="row.id">`,
		`x-on:click="removeAt(i)"`,
		`x-on:click="add()"`,
		`id="env-add"`,
		`> Add row</button>`,
	)
	if n := strings.Count(out, `<div data-row`); n != 3 { // two rows + the live template's
		t.Errorf("want 2 server rows + 1 template row, got %d:\n%s", n, out)
	}
	k0 := attrsOf(t, out, `id="env-key-0"`)
	mustContain(t, k0, `name="env.key"`, `value="HOST"`, `data-key`, `aria-label="Key"`, `placeholder="Key"`)
	v0 := attrsOf(t, out, `id="env-value-0"`)
	mustContain(t, v0, `name="env.value"`, `value="db.local"`, `data-value`, `aria-label="Value"`)
	k1 := attrsOf(t, out, `id="env-key-1"`)
	mustContain(t, k1, `value="PORT"`)
	v1 := attrsOf(t, out, `id="env-value-1"`)
	mustContain(t, v1, `value="5432"`)
	mustContain(t, out, `aria-label="Remove row 1"`, `aria-label="Remove row 2"`)
	// Key before value before the next row's key, so Parse's zip order
	// matches the submit order.
	if !(strings.Index(out, `id="env-key-0"`) < strings.Index(out, `id="env-value-0"`) &&
		strings.Index(out, `id="env-value-0"`) < strings.Index(out, `id="env-key-1"`) &&
		strings.Index(out, `id="env-key-1"`) < strings.Index(out, `<template x-for`)) {
		t.Errorf("rows out of order:\n%s", out)
	}
	// The live row binds its inputs to state and its ids to the row's
	// stable id, not its index, so removing a row renames no other row.
	mustContain(t, out,
		`x-model="row.key"`, `x-model="row.value"`,
		`x-bind:id="$root.id + &#39;-key-&#39; + row.id"`,
		`x-bind:id="$root.id + &#39;-value-&#39; + row.id"`,
		`x-bind:aria-label="&#39;Remove row &#39; + (i + 1)"`,
	)
	mustNotContain(t, attrsOf(t, out, `id="env"`), ` data-disabled`)
	if p := (Props{ID: "env"}); p.ControlID() != "env-key-0" {
		t.Errorf("ControlID = %q", p.ControlID())
	}
}

func TestKeyValueEditor_EmptyRowsStillHasAddButton(t *testing.T) {
	out := render(t, KeyValueEditor(Props{ID: "env", Name: "env"}))
	if n := strings.Count(out, `<div data-row`); n != 1 {
		t.Errorf("want only the template row, got %d:\n%s", n, out)
	}
	mustContain(t, out, `id="env-add"`, `x-bind:disabled="disabled || atMax"`)
	mustNotContain(t, out, `name="env.key" value=`)
}

func TestKeyValueEditor_MaxRows(t *testing.T) {
	below := render(t, KeyValueEditor(Props{ID: "e", Name: "e", Rows: []Row{{"a", "1"}}, MaxRows: 2}))
	mustContain(t, below, `data-max="2"`)
	if isDisabled(attrsOf(t, below, `id="e-add"`)) {
		t.Errorf("Add row should be enabled below the cap:\n%s", below)
	}

	atCap := render(t, KeyValueEditor(Props{ID: "e", Name: "e", Rows: []Row{{"a", "1"}, {"b", "2"}}, MaxRows: 2}))
	if !isDisabled(attrsOf(t, atCap, `id="e-add"`)) {
		t.Errorf("Add row should be disabled at the cap:\n%s", atCap)
	}
	// The rows themselves stay editable at the cap.
	if isDisabled(attrsOf(t, atCap, `id="e-key-0"`)) {
		t.Errorf("rows should stay enabled at the cap:\n%s", atCap)
	}
}

func TestKeyValueEditor_Disabled(t *testing.T) {
	out := render(t, KeyValueEditor(Props{ID: "e", Name: "e", Rows: []Row{{"a", "1"}}, Disabled: true}))
	mustContain(t, attrsOf(t, out, `id="e"`), ` data-disabled`)
	for _, m := range []string{`id="e-key-0"`, `id="e-value-0"`, `aria-label="Remove row 1"`, `id="e-add"`} {
		if tag := attrsOf(t, out, m); !isDisabled(tag) {
			t.Errorf("%s should be disabled: %s", m, tag)
		}
	}
}

func TestKeyValueEditor_CustomLabelsAndEscaping(t *testing.T) {
	out := render(t, KeyValueEditor(Props{
		ID: "e", Name: "e", AddLabel: "Add variable", KeyPlaceholder: "Variable", ValuePlaceholder: "Setting",
		Rows: []Row{{`<script>k</script>`, `"v" & more`}},
	}))
	mustContain(t, out,
		`> Add variable</button>`,
		`placeholder="Variable"`, `aria-label="Variable"`,
		`placeholder="Setting"`, `aria-label="Setting"`,
		`value="&lt;script&gt;k&lt;/script&gt;"`,
		`value="&#34;v&#34; &amp; more"`,
	)
	mustNotContain(t, out, `<script>k`)
}

func TestKeyValueEditor_AttributesOnRoot(t *testing.T) {
	out := render(t, KeyValueEditor(Props{ID: "e", Name: "e", Class: "mt-4", Attributes: templ.Attributes{"data-test": "kv"}}))
	mustContain(t, attrsOf(t, out, `id="e"`), `class="w-full mt-4"`, `data-test="kv"`)
}

func TestKeyValueEditor_DefaultID(t *testing.T) {
	out := render(t, KeyValueEditor(Props{Name: "e", Rows: []Row{{"a", "1"}}}))
	mustContain(t, out, `id="id-`, `-key-0"`, `-add"`)
}

func TestParse(t *testing.T) {
	for name, tc := range map[string]struct {
		form url.Values
		want []Row
	}{
		"zips in order": {
			form: url.Values{"env.key": {"HOST", "PORT"}, "env.value": {"db.local", "5432"}},
			want: []Row{{"HOST", "db.local"}, {"PORT", "5432"}},
		},
		"drops blank keys, keeps their position out": {
			form: url.Values{"env.key": {"A", "", "  ", "B"}, "env.value": {"1", "ignored", "ignored", "2"}},
			want: []Row{{"A", "1"}, {"B", "2"}},
		},
		"empty value kept": {
			form: url.Values{"env.key": {"A"}, "env.value": {""}},
			want: []Row{{"A", ""}},
		},
		"missing value is empty": {
			form: url.Values{"env.key": {"A", "B"}, "env.value": {"1"}},
			want: []Row{{"A", "1"}, {"B", ""}},
		},
		"trims keys, not values": {
			form: url.Values{"env.key": {" A "}, "env.value": {" 1 "}},
			want: []Row{{"A", " 1 "}},
		},
		"other names ignored": {
			form: url.Values{"other.key": {"X"}, "other.value": {"9"}},
			want: []Row{},
		},
		"absent": {form: url.Values{}, want: []Row{}},
	} {
		t.Run(name, func(t *testing.T) {
			got := Parse(tc.form, "env")
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Parse = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestParse_RoundTrip(t *testing.T) {
	rows := []Row{{"HOST", "db.local"}, {"PORT", "5432"}}
	// Simulate the browser: fields in document order.
	form := url.Values{}
	for _, r := range rows {
		form.Add("env.key", r.Key)
		form.Add("env.value", r.Value)
	}
	if got := Parse(form, "env"); !reflect.DeepEqual(got, rows) {
		t.Errorf("round trip = %#v", got)
	}
}
