package formpage

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/formfield"
	"github.com/bplus/webframe/components/pageheader"
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

// mustOrder asserts each want appears after the one before it.
func mustOrder(t *testing.T, out string, wants ...string) {
	t.Helper()
	pos := 0
	for _, want := range wants {
		i := strings.Index(out[pos:], want)
		if i < 0 {
			t.Fatalf("output missing %q after offset %d:\n%s", want, pos, out)
		}
		pos += i + len(want)
	}
}

// itemProps is the archetype create/edit form: two sections, a text, a
// select and a toggle field.
func itemProps() Props {
	return Props{
		Header: pageheader.Props{Title: "New item"},
		Action: "/items",
		Sections: []Section{
			{Title: "Basics", Description: "What it is", Fields: []formfield.Field{
				{Name: "name", Label: "Name", Required: true},
				{Name: "status", Label: "Status", Type: formfield.TypeSelect, Options: []formfield.Option{{Label: "Draft", Value: "draft"}, {Label: "Published", Value: "published"}}},
			}},
			{Title: "Options", Fields: []formfield.Field{
				{Name: "featured", Label: "Featured", Type: formfield.TypeToggle},
			}},
		},
		Cancel: &action.Action{Label: "Cancel", Href: "/items"},
	}
}

func TestBody_ValuesOverride(t *testing.T) {
	p := itemProps()
	p.Sections[0].Fields[0].Value = "Default"
	p.Values = map[string]string{"name": "Ada", "featured": "on", "nope": "ignored"}
	out := render(t, Body(p))
	mustContain(t, out, `value="Ada"`, `name="featured"`, `role="switch" checked`)
	mustNotContain(t, out, `value="Default"`, "ignored")

	// An absent key leaves Field.Value alone (edit-page prefill).
	p.Values = map[string]string{}
	mustContain(t, render(t, Body(p)), `value="Default"`)

	// A present empty value unchecks a toggle that defaulted to on.
	p.Sections[1].Fields[0].Value = "on"
	p.Values = map[string]string{"featured": ""}
	mustNotContain(t, render(t, Body(p)), `role="switch" checked`)
}

func TestBody_ErrorsBannerAndFields(t *testing.T) {
	p := itemProps()
	p.Errors = map[string]string{"name": "Name is required", "status": "Pick a status"}
	out := render(t, Body(p))
	mustContain(t, out,
		`role="alert"`,
		"Please fix the 2 errors below.",
		`href="#field-name"`, `href="#field-status"`,
		"Name: Name is required", "Status: Pick a status",
	)
	if n := strings.Count(out, `aria-invalid="true"`); n != 2 {
		t.Errorf("aria-invalid count = %d, want 2:\n%s", n, out)
	}
	// Banner comes before the fields, in section order.
	mustOrder(t, out, "Please fix", "Name: Name is required", "Status: Pick a status", `data-wf-field="name"`)
}

// TestBody_FileFieldAfterErrors: a form re-rendered with Errors cannot
// refill its file input, so the file field asks for the file again even
// when the error is on another field; without Errors it says nothing. The
// banner's link for an error on the file field lands on the file input.
func TestBody_FileFieldAfterErrors(t *testing.T) {
	p := itemProps()
	p.Multipart = true
	p.Sections[1].Fields = append(p.Sections[1].Fields, formfield.Field{Name: "art", Label: "Artwork", Type: formfield.TypeFile})
	mustNotContain(t, render(t, Body(p)), "Choose the file again")
	// An edit page's first load: the current file name in Values, no
	// Errors. Nothing has failed, so no line.
	p.Values = map[string]string{"art": "cover.png"}
	mustNotContain(t, render(t, Body(p)), "Choose the file again")
	p.Errors = map[string]string{"name": "Name is required"}
	mustContain(t, render(t, Body(p)), `id="field-art-again"`, "Choose the file again")
	p.Errors = map[string]string{"art": "Too big"}
	mustContain(t, render(t, Body(p)), `href="#field-art"`, `<input type="file" id="field-art"`, "Choose the file again")
}

// TestBody_RadioErrorLinkTargetExists: an error on a radio field links to
// "#field-<name>", and the rendered body has an element with that id (the
// radiogroup), so the link lands somewhere.
func TestBody_RadioErrorLinkTargetExists(t *testing.T) {
	p := itemProps()
	p.Sections[1].Fields = append(p.Sections[1].Fields, formfield.Field{
		Name: "size", Label: "Size", Type: formfield.TypeRadio,
		Options: []formfield.Option{{Label: "Small", Value: "s"}, {Label: "Large", Value: "l"}},
	})
	p.Errors = map[string]string{"size": "Pick a size"}
	out := render(t, Body(p))
	mustContain(t, out, `href="#field-size"`, `id="field-size" role="radiogroup"`, "Size: Pick a size")
}

func TestBody_SingularError(t *testing.T) {
	p := itemProps()
	p.Errors = map[string]string{"name": "Name is required"}
	out := render(t, Body(p))
	mustContain(t, out, "Please fix the error below.")
	mustNotContain(t, out, "errors below")
}

func TestBody_UnmatchedErrorStillCounted(t *testing.T) {
	p := itemProps()
	p.Errors = map[string]string{"name": "Name is required", "_form": "Something else went wrong"}
	out := render(t, Body(p))
	mustContain(t, out, "Please fix the 2 errors below.", "Something else went wrong")
	mustNotContain(t, out, `href="#field-_form"`)
}

func TestBody_NoErrorsNoBanner(t *testing.T) {
	out := render(t, Body(itemProps()))
	mustNotContain(t, out, `role="alert"`, "Please fix", `aria-invalid="true"`)
}

func TestBody_ValidateDefaultApplied(t *testing.T) {
	p := itemProps()
	p.Validate = "/items/validate"
	p.Sections[0].Fields[1].Validate = "/items/validate-status"
	out := render(t, Body(p))
	mustContain(t, out, `hx-post="/items/validate"`, `hx-post="/items/validate-status"`)
	// name, featured get the default; status keeps its own: three validated controls.
	if n := strings.Count(out, `hx-target="#field-`); n != 3 {
		t.Errorf("validated controls = %d, want 3:\n%s", n, out)
	}
	// Without Props.Validate nothing is wired.
	mustNotContain(t, render(t, Body(itemProps())), "hx-post=")
}

func TestBody_SubmitDefaults(t *testing.T) {
	out := render(t, Body(itemProps()))
	mustContain(t, out, `type="submit"`, "<span>Save</span>", "bg-primary", `href="/items"`, "<span>Cancel</span>", "justify-end")
	mustOrder(t, out, "<span>Cancel</span>", `type="submit"`, "<span>Save</span>")

	p := itemProps()
	p.Submit = action.Action{Label: "Create item", Variant: action.VariantDestructive}
	out = render(t, Body(p))
	mustContain(t, out, "<span>Create item</span>", "bg-destructive")
	mustNotContain(t, out, "<span>Save</span>")
}

func TestBody_FormDefaults(t *testing.T) {
	p := itemProps()
	p.Hidden = map[string]string{"return": "/items"}
	out := render(t, Body(p))
	mustContain(t, out, `<form id="form"`, `action="/items"`, `method="post"`, `name="return"`, `<h3`, "Basics", "What it is", "Options", "<fieldset")
	mustNotContain(t, out, "hx-post=", "hx-target=", "<h1", "enctype")
	if !strings.HasPrefix(out, "<form") || !strings.HasSuffix(out, "</form>") {
		t.Errorf("Body is not exactly the <form> element:\n%s", out)
	}

	p.ID = "item-form"
	p.Method = "get"
	p.Multipart = true
	out = render(t, Body(p))
	mustContain(t, out, `<form id="item-form"`, `method="get"`, `enctype="multipart/form-data"`)
}

func TestBody_HxPostTargetsForm(t *testing.T) {
	p := itemProps()
	p.HxPost = true
	out := render(t, Body(p))
	mustContain(t, out, `hx-post="/items"`, `hx-target="#form"`, `hx-swap="outerHTML"`, `action="/items"`)

	p.ID = "item-form"
	mustContain(t, render(t, Body(p)), `hx-target="#item-form"`)
}

func TestPage_HeaderThenBody(t *testing.T) {
	out := render(t, Page(itemProps()))
	mustContain(t, out, "<h1", "New item", `<form id="form"`)
	mustOrder(t, out, "<h1", "New item", "<form")
}

func TestField(t *testing.T) {
	p := itemProps()
	p.Values = map[string]string{"name": "Ada"}
	p.Errors = map[string]string{"name": "bad"}
	p.Validate = "/v"
	f, ok := p.Field("name")
	if !ok || f.Value != "Ada" || f.Error != "bad" || f.Validate != "/v" {
		t.Errorf("Field(name) = %+v, %v", f, ok)
	}
	if _, ok := p.Field("missing"); ok {
		t.Error("Field(missing) reported found")
	}
}

// serveRespond runs Respond through appshell Middleware with the given
// request headers and returns the recorded response.
func serveRespond(t *testing.T, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	a := &appshell.App{Name: "Demo", Nav: []appshell.NavItem{{Label: "Items", Href: "/items"}}}
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := itemProps()
		p.Errors = map[string]string{"name": "Name is required"}
		if err := Respond(w, r, "New item", p); err != nil {
			t.Errorf("Respond: %v", err)
		}
	}))
	req := httptest.NewRequest("POST", "/items", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	return rec
}

func TestRespond_TargetedBody(t *testing.T) {
	for name, target := range map[string]string{"bare": "form", "hashed": "#form"} {
		out := serveRespond(t, map[string]string{"HX-Request": "true", "HX-Target": target}).Body.String()
		if !strings.HasPrefix(out, "<form") {
			t.Errorf("%s: body does not start with the form:\n%s", name, out)
		}
		mustContain(t, out, "Please fix the error below.", `aria-invalid="true"`)
		mustNotContain(t, out, "<h1", "<html", "wf-topbar")
	}
}

func TestRespond_OtherTargetGetsPage(t *testing.T) {
	out := serveRespond(t, map[string]string{"HX-Request": "true", "HX-Target": "main"}).Body.String()
	mustContain(t, out, "<h1", "New item", "<form")
	mustNotContain(t, out, "<html", "wf-topbar")
}

func TestRespond_FullDocument(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"plain":   nil,
		"boosted": {"HX-Request": "true", "HX-Boosted": "true", "HX-Target": "form"},
	} {
		out := serveRespond(t, headers).Body.String()
		if !strings.HasPrefix(strings.ToLower(out), "<!doctype") {
			t.Errorf("%s: not a full document:\n%.80s", name, out)
		}
		mustContain(t, out, "<title>New item", "<h1", `<form id="form"`)
	}
}

func TestRespond_NoApp(t *testing.T) {
	req := httptest.NewRequest("POST", "/items", nil)
	rec := httptest.NewRecorder()
	err := Respond(rec, req, "New item", itemProps())
	if err == nil || !strings.Contains(err.Error(), "Middleware") {
		t.Errorf("err = %v, want one naming Middleware", err)
	}
}

func TestBody_SubmitIgnoresHrefAndHx(t *testing.T) {
	p := itemProps()
	p.Submit = action.Action{Label: "Save", Href: "/ignored", HxPost: "/ignored", HxTarget: "#ignored"}
	out := render(t, Body(p))
	mustContain(t, out, `type="submit"`, "<span>Save</span>")
	mustNotContain(t, out, `href="/ignored"`, `hx-post="/ignored"`, `hx-target="#ignored"`)
}

// TestRespond_ErrorsAre422: a re-render with Errors answers 422 with the
// headers Render sets; without Errors it is 200.
func TestRespond_ErrorsAre422(t *testing.T) {
	app := &appshell.App{Name: "Demo", HomeHref: "/"}
	serve := func(p Props) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/items", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", p.id())
		app.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := Respond(w, r, "New", p); err != nil {
				t.Fatal(err)
			}
		})).ServeHTTP(rec, req)
		return rec
	}
	p := Props{ID: "item-form", Action: "/items", Sections: []Section{{Fields: []formfield.Field{{Name: "name", Label: "Name"}}}}}
	if rec := serve(p); rec.Code != http.StatusOK {
		t.Errorf("no errors: status %d, want 200", rec.Code)
	}
	p.Errors = map[string]string{"name": "Required"}
	rec := serve(p)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("with errors: status %d, want 422", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type %q lost with the early status", ct)
	}
	if v := rec.Header().Get("Vary"); !strings.Contains(v, "HX-Request") {
		t.Errorf("Vary %q lost with the early status", v)
	}
	if !strings.Contains(rec.Body.String(), `aria-invalid="true"`) {
		t.Errorf("error re-render lacks aria-invalid:\n%s", rec.Body.String())
	}
}

// TestBody_SectionContent: a Section's Content renders inside the <form>
// (so the submit button posts it with the fields), inside its own section
// after the fieldset and before the submit row; a section with Content and
// no Fields renders no fieldset; nil Content renders nothing beyond the
// fieldset.
func TestBody_SectionContent(t *testing.T) {
	p := itemProps()
	p.Sections[1].Content = templ.Raw(`<p id="extra">EXTRA</p>`)
	p.Sections = append(p.Sections, Section{Title: "Notes", Content: templ.Raw(`<p id="notes">NOTESCONTENT</p>`)})
	out := render(t, Body(p))
	mustOrder(t, out, "<form", "Options", "<fieldset", `name="featured"`, "</fieldset>", `id="extra"`, "EXTRA", "</section>",
		"Notes", "NOTESCONTENT", "</section>", `type="submit"`, "</form>")
	if n := strings.Count(out, "<fieldset"); n != 2 {
		t.Errorf("got %d fieldsets, want 2 (one per section with fields):\n%s", n, out)
	}
	notes := out[strings.Index(out, "NOTESCONTENT"):]
	notes = notes[:strings.Index(notes, "</section>")]
	mustNotContain(t, notes, "<fieldset")
	// Nil Content: the section holds the fieldset and nothing after it.
	out = render(t, Body(itemProps()))
	sec := out[strings.Index(out, "Options"):]
	sec = sec[:strings.Index(sec, "</section>")]
	after := sec[strings.LastIndex(sec, "</fieldset>")+len("</fieldset>"):]
	if strings.TrimSpace(after) != "" {
		t.Errorf("content after the fieldset without Content: %q", after)
	}
}

// An untitled section (no Title, no Description) has no heading to space
// out, so it must not render as a padded, bordered layout.Section: that
// left an empty band between the error banner and the first field. A
// titled section on the same form keeps its layout.Section.
func TestBody_UntitledSectionHasNoSectionPadding(t *testing.T) {
	p := Props{
		Action: "/books",
		Sections: []Section{
			{Fields: []formfield.Field{{Name: "title", Label: "Title"}}},
			{Title: "More", Fields: []formfield.Field{{Name: "author", Label: "Author"}}},
		},
		Errors: map[string]string{"title": "Enter a title."},
	}
	out := render(t, Body(p))
	first := out[:strings.Index(out, `name="title"`)]
	first = first[strings.Index(first, "Please fix"):]
	mustNotContain(t, first, "wf-section", "<section")
	if n := strings.Count(out, "wf-section"); n != 1 {
		t.Errorf("got %d wf-section elements, want 1 (the titled section only):\n%s", n, out)
	}
	mustOrder(t, out, "Please fix", `name="title"`, "<section", "More", `name="author"`, `type="submit"`)
}

// TestRespond_CustomIDTargetedBody: with Props.ID set, an htmx submit
// aimed at that id gets the form alone with 422, and the form carries the
// id and the matching hx-target.
func TestRespond_CustomIDTargetedBody(t *testing.T) {
	a := &appshell.App{Name: "Demo"}
	p := itemProps()
	p.ID = "edit-item"
	p.HxPost = true
	p.Errors = map[string]string{"name": "Name is required"}
	for name, target := range map[string]string{"bare": "edit-item", "hashed": "#edit-item"} {
		h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := Respond(w, r, "Edit item", p); err != nil {
				t.Errorf("Respond: %v", err)
			}
		}))
		req := httptest.NewRequest("POST", "/items/1", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Target", target)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: status %d, want 422", name, rec.Code)
		}
		out := rec.Body.String()
		if !strings.HasPrefix(out, `<form id="edit-item"`) || !strings.HasSuffix(strings.TrimSpace(out), "</form>") {
			t.Errorf("%s: body is not the form alone:\n%s", name, out)
		}
		mustContain(t, out, `hx-target="#edit-item"`, "Please fix the error below.")
		mustNotContain(t, out, "<h1", "<html", "wf-topbar")
	}
}
