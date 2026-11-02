package settingspage

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

// settingsProps is the archetype settings screen: two groups, a text, a
// select and a toggle field, with a Cancel link.
func settingsProps() Props {
	return Props{
		Header: pageheader.Props{Title: "Settings"},
		Action: "/settings",
		Groups: []Group{
			{ID: "profile", Title: "Profile", Description: "How you appear", Fields: []formfield.Field{
				{Name: "name", Label: "Display name", Required: true},
				{Name: "theme", Label: "Theme", Type: formfield.TypeSelect, Options: []formfield.Option{{Label: "Light", Value: "light"}, {Label: "Dark", Value: "dark"}}},
			}},
			{ID: "notifications", Title: "Notifications", Fields: []formfield.Field{
				{Name: "digest", Label: "Weekly digest", Type: formfield.TypeToggle},
			}},
		},
		Cancel: &action.Action{Label: "Cancel", Href: "/"},
	}
}

func TestBody_GroupsAreSectionsWithFieldsets(t *testing.T) {
	out := render(t, Body(settingsProps()))
	mustContain(t, out, `<section id="profile"`, `<section id="notifications"`, "<h2", "Profile", "How you appear", "Notifications", "<fieldset")
	mustNotContain(t, out, "<h3", "<h1", "<legend")
	if n := strings.Count(out, "<fieldset"); n != 2 {
		t.Errorf("fieldset count = %d, want 2:\n%s", n, out)
	}
	mustOrder(t, out, `<section id="profile"`, `data-wf-field="name"`, `data-wf-field="theme"`, `<section id="notifications"`, `data-wf-field="digest"`)
}

func TestBody_ValuesOverride(t *testing.T) {
	p := settingsProps()
	p.Groups[0].Fields[0].Value = "Default"
	p.Values = map[string]string{"name": "Ada", "digest": "on", "nope": "ignored"}
	out := render(t, Body(p))
	mustContain(t, out, `value="Ada"`, `name="digest"`, `role="switch" checked`)
	mustNotContain(t, out, `value="Default"`, "ignored")

	// An absent key leaves Field.Value alone (prefill from the saved settings).
	p.Values = map[string]string{}
	mustContain(t, render(t, Body(p)), `value="Default"`)

	// A present empty value unchecks a toggle that defaulted to on.
	p.Groups[1].Fields[0].Value = "on"
	p.Values = map[string]string{"digest": ""}
	mustNotContain(t, render(t, Body(p)), `role="switch" checked`)
}

func TestBody_ErrorsBannerAndFields(t *testing.T) {
	p := settingsProps()
	p.Errors = map[string]string{"name": "Pick a name", "digest": "Not available", "_form": "Something else went wrong"}
	out := render(t, Body(p))
	mustContain(t, out,
		`role="alert"`,
		"Please fix the 3 errors below.",
		`href="#field-name"`, `href="#field-digest"`,
		"Display name: Pick a name", "Weekly digest: Not available", "Something else went wrong",
	)
	mustNotContain(t, out, `href="#field-_form"`)
	if n := strings.Count(out, `aria-invalid="true"`); n != 2 {
		t.Errorf("aria-invalid count = %d, want 2:\n%s", n, out)
	}
	// The banner is inside the form, before the groups, in group order; the
	// save bar is not part of the form.
	mustOrder(t, out, "<form", "Please fix", "Display name: Pick a name", "Weekly digest: Not available", `<section id="profile"`, "</form>")
	mustNotContain(t, out, "wf-settings-bar")

	p.Errors = map[string]string{"name": "Pick a name"}
	out = render(t, Body(p))
	mustContain(t, out, "Please fix the 1 error below.")
	mustNotContain(t, out, "errors below")
}

func TestBody_NoErrorsNoBanner(t *testing.T) {
	out := render(t, Body(settingsProps()))
	mustNotContain(t, out, `role="alert"`, "Please fix", `aria-invalid="true"`)
}

// TestBody_IsTheFormWithoutTheBar: Body is exactly the <form> element and
// the save bar is not inside it (it is Page's last child), so an htmx
// error re-render that swaps the form leaves the bar alone.
func TestBody_IsTheFormWithoutTheBar(t *testing.T) {
	for name, p := range map[string]Props{
		"plain":  settingsProps(),
		"empty":  {},
		"errors": func() Props { p := settingsProps(); p.Errors = map[string]string{"name": "x"}; return p }(),
	} {
		out := render(t, Body(p))
		if !strings.HasPrefix(out, "<form") || !strings.HasSuffix(out, "</form>") {
			t.Errorf("%s: Body is not exactly the <form> element:\n%s", name, out)
		}
		mustNotContain(t, out, "wf-settings-bar", `type="submit"`, "<span>Save changes</span>", "<span>Cancel</span>")
	}
}

// TestPage_SaveBarAlwaysPresentAndLast: the save bar renders once on every
// page, after the form, as the page root's last child, and its submit
// button names the form it submits.
func TestPage_SaveBarAlwaysPresentAndLast(t *testing.T) {
	for name, p := range map[string]Props{
		"plain":  settingsProps(),
		"empty":  {},
		"errors": func() Props { p := settingsProps(); p.Errors = map[string]string{"name": "x"}; return p }(),
	} {
		out := render(t, Page(p))
		mustContain(t, out, `<div class="wf-settings flex flex-col gap-8">`, "wf-settings-bar", "sticky", "bottom-0", "z-20", "border-t", "bg-background/95", "backdrop-blur", "pb-[calc(0.75rem+env(safe-area-inset-bottom,0px))]", `type="submit"`, `form="settings"`, "<span>Save changes</span>", "bg-primary", "justify-end")
		if !strings.HasPrefix(out, `<div class="wf-settings`) || !strings.HasSuffix(out, "</div>") {
			t.Errorf("%s: Page is not one wf-settings root:\n%s", name, out)
		}
		mustOrder(t, out, "</form>", "wf-settings-bar", `type="submit"`)
		// The bar is the root's last child: nothing but closing tags follows it.
		tail := out[strings.LastIndex(out, `type="submit"`):]
		if strings.Contains(tail, "<fieldset") || strings.Contains(tail, "<section") || strings.Contains(tail, "<form") {
			t.Errorf("%s: content after the save bar:\n%s", name, tail)
		}
		if strings.Count(out, "wf-settings-bar") != 1 {
			t.Errorf("%s: save bar rendered %d times", name, strings.Count(out, "wf-settings-bar"))
		}
		// The form attribute is not inherited by Cancel, nor by the form itself.
		if strings.Count(out, `form="settings"`) != 1 {
			t.Errorf("%s: form= attribute count = %d, want 1 (the submit button):\n%s", name, strings.Count(out, `form="settings"`), out)
		}
	}
	// The submit button follows the form's id.
	p := settingsProps()
	p.ID = "prefs"
	mustContain(t, render(t, Page(p)), `<form id="prefs"`, `form="prefs"`)
}

func TestPage_AfterBetweenFormAndBar(t *testing.T) {
	p := settingsProps()
	p.After = templ.Raw(`<section id="danger">AFTERCONTENT</section>`)
	out := render(t, Page(p))
	mustOrder(t, out, "<h1", "<form", "</form>", `<section id="danger">`, "AFTERCONTENT", "wf-settings-bar", `type="submit"`)
	// After is not inside the form, so a submit never posts it and the
	// targeted re-render (Body) never redraws it.
	mustNotContain(t, render(t, Body(p)), "AFTERCONTENT")
	// Nil After renders nothing between the form and the bar.
	p.After = nil
	out = render(t, Page(p))
	between := out[strings.Index(out, "</form>")+len("</form>") : strings.Index(out, "wf-settings-bar")]
	if strings.TrimSpace(between) != `<div class="` {
		t.Errorf("content between the form and the bar without After: %q", between)
	}
}

func TestPage_SaveBarCancelGhostThenSubmit(t *testing.T) {
	out := render(t, Page(settingsProps()))
	mustContain(t, out, `href="/"`, "<span>Cancel</span>")
	mustOrder(t, out, "wf-settings-bar", "<span>Cancel</span>", `type="submit"`, "<span>Save changes</span>")
	// Cancel is a ghost button: no secondary fill.
	cancel := out[strings.Index(out, "wf-settings-bar"):strings.Index(out, "<span>Cancel</span>")]
	mustContain(t, cancel, "hover:bg-accent")
	mustNotContain(t, cancel, "bg-secondary")

	// A Cancel with its own Variant keeps it.
	p := settingsProps()
	p.Cancel = &action.Action{Label: "Back", Href: "/", Variant: action.VariantOutline}
	out = render(t, Page(p))
	mustContain(t, out, "<span>Back</span>", "border bg-background")

	// No Cancel: the bar holds the submit button alone.
	p.Cancel = nil
	out = render(t, Page(p))
	mustContain(t, out, "wf-settings-bar", `type="submit"`)
	mustNotContain(t, out, "<span>Cancel</span>", "<span>Back</span>")
}

func TestPage_SubmitOverridesAndIgnoresHrefHxForm(t *testing.T) {
	p := settingsProps()
	p.Submit = action.Action{Label: "Apply", Variant: action.VariantDestructive, Href: "/ignored", HxPost: "/ignored", HxTarget: "#ignored", Form: "other"}
	out := render(t, Page(p))
	mustContain(t, out, `type="submit"`, `form="settings"`, "<span>Apply</span>", "bg-destructive")
	mustNotContain(t, out, "<span>Save changes</span>", `href="/ignored"`, `hx-post="/ignored"`, `hx-target="#ignored"`, `form="other"`)
}

func TestBody_FormDefaultsAndOptions(t *testing.T) {
	p := settingsProps()
	p.Hidden = map[string]string{"return": "/"}
	out := render(t, Body(p))
	mustContain(t, out, `<form id="settings"`, `action="/settings"`, `method="post"`, `name="return"`, "flex-col", "gap-8")
	// wf-settings (the scroll-margin hook) is the page root's, not the form's.
	mustNotContain(t, out, "hx-post=", "hx-target=", "enctype", "wf-settings")

	p.ID = "prefs"
	p.Method = "get"
	p.Class = "max-w-2xl"
	p.Attributes = templ.Attributes{"data-x": "y"}
	p.HxPost = true
	out = render(t, Body(p))
	mustContain(t, out, `<form id="prefs"`, `method="get"`, "max-w-2xl", `data-x="y"`, `hx-post="/settings"`, `hx-target="#prefs"`, `hx-swap="outerHTML"`, `action="/settings"`)
}

func TestBody_ValidateDefaultApplied(t *testing.T) {
	p := settingsProps()
	p.Validate = "/settings/validate"
	p.Groups[0].Fields[1].Validate = "/settings/validate-theme"
	out := render(t, Body(p))
	mustContain(t, out, `hx-post="/settings/validate"`, `hx-post="/settings/validate-theme"`)
	if n := strings.Count(out, `hx-target="#field-`); n != 3 {
		t.Errorf("validated controls = %d, want 3:\n%s", n, out)
	}
	mustNotContain(t, render(t, Body(settingsProps())), "hx-post=")
}

func TestPage_HeaderThenBodyThenBar(t *testing.T) {
	out := render(t, Page(settingsProps()))
	mustContain(t, out, "<h1", "Settings", `<form id="settings"`, "wf-settings-bar")
	mustOrder(t, out, `class="wf-settings`, "<h1", "Settings", "<form", "</form>", "wf-settings-bar")
}

func TestField(t *testing.T) {
	p := settingsProps()
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
	// The declaration is not mutated.
	if p.Groups[0].Fields[0].Value != "" || p.Groups[0].Fields[0].Error != "" {
		t.Errorf("Field mutated the declaration: %+v", p.Groups[0].Fields[0])
	}
}

// serveRespond runs Respond through appshell Middleware with the given
// request headers and props and returns the recorded response.
func serveRespond(t *testing.T, p Props, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	a := &appshell.App{Name: "Demo", HomeHref: "/", Nav: []appshell.NavItem{{Label: "Settings", Href: "/settings"}}}
	h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := Respond(w, r, "Settings", p); err != nil {
			t.Errorf("Respond: %v", err)
		}
	}))
	req := httptest.NewRequest("POST", "/settings", nil)
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

func withErrors() Props {
	p := settingsProps()
	p.Values = map[string]string{"name": "", "theme": "dark", "digest": "on"}
	p.Errors = map[string]string{"name": "Pick a name"}
	return p
}

// TestRespond_TargetedForm422: an HxPost re-render with Errors answers 422
// with just the form, every submitted value kept, the error under its
// field and in the banner; After and the save bar are not in the part
// (they stay on the page untouched).
func TestRespond_TargetedForm422(t *testing.T) {
	for name, target := range map[string]string{"bare": "settings", "hashed": "#settings"} {
		p := withErrors()
		p.After = templ.Raw("<p>AFTERCONTENT</p>")
		rec := serveRespond(t, p, map[string]string{"HX-Request": "true", "HX-Target": target})
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s: status %d, want 422", name, rec.Code)
		}
		out := rec.Body.String()
		if !strings.HasPrefix(out, "<form") || !strings.HasSuffix(strings.TrimSpace(out), "</form>") {
			t.Errorf("%s: body is not the form alone:\n%s", name, out)
		}
		mustContain(t, out, "Please fix the 1 error below.", "Display name: Pick a name", `aria-invalid="true"`, `role="switch" checked`)
		mustNotContain(t, out, "<h1", "<html", "wf-topbar", "wf-settings-bar", "<span>Save changes</span>", "AFTERCONTENT")
		if v := rec.Header().Get("Vary"); !strings.Contains(v, "HX-Request") {
			t.Errorf("%s: Vary %q lost with the early status", name, v)
		}
	}
}

func TestRespond_NoErrorsIs200(t *testing.T) {
	rec := serveRespond(t, settingsProps(), map[string]string{"HX-Request": "true", "HX-Target": "settings"})
	if rec.Code != http.StatusOK {
		t.Errorf("status %d, want 200", rec.Code)
	}
	mustNotContain(t, rec.Body.String(), `role="alert"`)
}

func TestRespond_OtherTargetGetsPage(t *testing.T) {
	p := withErrors()
	p.After = templ.Raw("<p>AFTERCONTENT</p>")
	rec := serveRespond(t, p, map[string]string{"HX-Request": "true", "HX-Target": "main"})
	out := rec.Body.String()
	mustContain(t, out, "<h1", "Settings", "<form", "AFTERCONTENT", "wf-settings-bar")
	mustOrder(t, out, "</form>", "AFTERCONTENT", "wf-settings-bar")
	mustNotContain(t, out, "<html", "wf-topbar")
	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status %d, want 422", rec.Code)
	}
}

func TestRespond_FullDocument(t *testing.T) {
	for name, headers := range map[string]map[string]string{
		"plain":   nil,
		"boosted": {"HX-Request": "true", "HX-Boosted": "true", "HX-Target": "settings"},
	} {
		rec := serveRespond(t, settingsProps(), headers)
		out := rec.Body.String()
		if !strings.HasPrefix(strings.ToLower(out), "<!doctype") {
			t.Errorf("%s: not a full document:\n%.80s", name, out)
		}
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d, want 200", name, rec.Code)
		}
		mustContain(t, out, "<title>Settings", "<h1", `<form id="settings"`, "wf-settings-bar")
	}
}

func TestRespond_NoApp(t *testing.T) {
	req := httptest.NewRequest("POST", "/settings", nil)
	rec := httptest.NewRecorder()
	err := Respond(rec, req, "Settings", settingsProps())
	if err == nil || !strings.Contains(err.Error(), "Middleware") {
		t.Errorf("err = %v, want one naming Middleware", err)
	}
}

// TestBody_GroupContent: a Group's Content renders inside the <form> (so
// the save bar posts it with the fields), inside its own section after the
// fieldset; a group with Content and no Fields renders no fieldset; nil
// Content renders nothing beyond the fieldset.
func TestBody_GroupContent(t *testing.T) {
	p := settingsProps()
	p.Groups[1].Content = templ.Raw(`<p id="extra">EXTRA</p>`)
	p.Groups = append(p.Groups, Group{ID: "environment", Title: "Environment", Content: templ.Raw(`<p id="env">ENVCONTENT</p>`)})
	out := render(t, Body(p))
	// Inside the form: after the form opens, before it closes.
	mustOrder(t, out, "<form", `<section id="notifications"`, "<fieldset", `name="digest"`, "</fieldset>", `id="extra"`, "EXTRA", "</section>",
		`<section id="environment"`, "ENVCONTENT", "</section>", "</form>")
	// The content-only group has no fieldset: exactly one per group with fields.
	if n := strings.Count(out, "<fieldset"); n != 2 {
		t.Errorf("got %d fieldsets, want 2 (one per group with fields):\n%s", n, out)
	}
	env := out[strings.Index(out, `<section id="environment"`):]
	env = env[:strings.Index(env, "</section>")]
	mustNotContain(t, env, "<fieldset")
	// Content is part of the form, so a targeted re-render (Body) carries it.
	mustContain(t, render(t, Page(p)), "ENVCONTENT")
	// Nil Content: the section holds the fieldset and nothing after it.
	q := settingsProps()
	out = render(t, Body(q))
	sec := out[strings.Index(out, `<section id="notifications"`):]
	sec = sec[:strings.Index(sec, "</section>")]
	after := sec[strings.LastIndex(sec, "</fieldset>")+len("</fieldset>"):]
	if strings.TrimSpace(after) != "" {
		t.Errorf("content after the fieldset without Content: %q", after)
	}
}

// TestBody_Multipart: Multipart puts enctype on the <form>, as formpage
// does; off by default.
func TestBody_Multipart(t *testing.T) {
	mustNotContain(t, render(t, Body(settingsProps())), "multipart/form-data")
	p := settingsProps()
	p.Multipart = true
	mustContain(t, render(t, Body(p)), `enctype="multipart/form-data"`)
}
