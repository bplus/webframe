package authpage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a-h/templ"

	"github.com/bplus/webframe/components/action"
	"github.com/bplus/webframe/components/appshell"
	"github.com/bplus/webframe/components/formfield"
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

func text(s string) templ.Component {
	return templ.ComponentFunc(func(_ context.Context, w io.Writer) error {
		_, err := io.WriteString(w, s)
		return err
	})
}

// signInProps is the archetype sign-in page: email and password, one
// provider.
func signInProps() Props {
	return Props{
		AppName: "Acme",
		Action:  "/signin",
		Fields: []formfield.Field{
			{Name: "email", Label: "Email", Type: formfield.TypeEmail, Required: true},
			{Name: "password", Label: "Password", Type: formfield.TypePassword, Required: true},
		},
		Providers: []action.Action{{Label: "Continue with GitHub", Href: "/auth/github"}},
	}
}

func TestSignIn_Defaults(t *testing.T) {
	out := render(t, AuthPage(signInProps()))
	mustContain(t, out,
		"wf-authpage", "min-h-dvh", "p-4",
		"max-w-sm", "w-full",
		`<h1 class="text-xl font-semibold text-center">Acme</h1>`,
		`<form id="signin-form"`, `action="/signin"`, `method="post"`,
		`name="email"`, `type="email"`, `name="password"`, `type="password"`,
		`type="submit"`, "<span>Sign in</span>",
		">or</span>", `href="/auth/github"`, "Continue with GitHub",
	)
	mustNotContain(t, out, "wf-shell", "<nav", "role=\"alert\"", "You have been signed out.")
	// The submit button sits inside the form; the providers sit after it.
	mustOrder(t, out, "<form", `type="submit"`, "</form>", ">or</span>", "Continue with GitHub")
}

func TestSignIn_SubmitAndProvidersFullWidth(t *testing.T) {
	out := render(t, AuthPage(signInProps()))
	// The <button ...> tag that carries type="submit".
	end := strings.Index(out, `type="submit"`)
	start := strings.LastIndex(out[:end], "<button")
	submit := out[start : end+strings.Index(out[end:], ">")]
	if !strings.Contains(submit, "w-full") {
		t.Errorf("submit button is not full width: %s", submit)
	}
	prov := out[strings.Index(out, `href="/auth/github"`):]
	prov = prov[:strings.Index(prov, ">")]
	for _, want := range []string{"w-full", "border", "bg-background"} {
		if !strings.Contains(prov, want) {
			t.Errorf("provider link missing %q (outline, full width): %s", want, prov)
		}
	}
}

// TestSignIn_ProviderVariant: a Provider's own Variant wins when set; an
// empty one is outline.
func TestSignIn_ProviderVariant(t *testing.T) {
	p := signInProps()
	p.Providers = []action.Action{
		{Label: "Continue with Google", Href: "/auth/google", Variant: action.VariantPrimary},
		{Label: "Continue with GitHub", Href: "/auth/github"},
	}
	out := render(t, AuthPage(p))
	tag := func(href string) string {
		t.Helper()
		i := strings.Index(out, `href="`+href+`"`)
		if i < 0 {
			t.Fatalf("no link to %s:\n%s", href, out)
		}
		start := strings.LastIndex(out[:i], "<a")
		return out[start : i+strings.Index(out[i:], ">")]
	}
	google, github := tag("/auth/google"), tag("/auth/github")
	if !strings.Contains(google, "bg-primary") || strings.Contains(google, "bg-background") {
		t.Errorf("primary provider should keep its variant: %s", google)
	}
	if !strings.Contains(github, "bg-background") || strings.Contains(github, "bg-primary") {
		t.Errorf("provider with no variant should be outline: %s", github)
	}
	for _, tag := range []string{google, github} {
		if !strings.Contains(tag, "w-full") {
			t.Errorf("provider is not full width: %s", tag)
		}
	}
}

func TestSignIn_ProviderButtonWithHx(t *testing.T) {
	p := signInProps()
	p.Providers = []action.Action{{Label: "Magic link", HxPost: "/auth/magic", HxTarget: "#x"}}
	out := render(t, AuthPage(p))
	mustContain(t, out, `hx-post="/auth/magic"`, `hx-target="#x"`, "Magic link")
	mustNotContain(t, out, `href="/auth/magic"`)
}

func TestSignIn_NoProvidersNoDivider(t *testing.T) {
	p := signInProps()
	p.Providers = nil
	out := render(t, AuthPage(p))
	mustNotContain(t, out, ">or</span>")
}

func TestSignIn_ValuesAndErrors(t *testing.T) {
	p := signInProps()
	p.Values = map[string]string{"email": "a@b.c"}
	p.Errors = map[string]string{"password": "Required"}
	out := render(t, AuthPage(p))
	mustContain(t, out,
		`value="a@b.c"`,
		`aria-invalid="true"`, "Required",
		`role="alert"`, "Please fix the 1 error below.",
		`href="#field-password"`, "Password: Required",
	)
	// The password field is the invalid one, not the email field.
	email := out[strings.Index(out, `name="email"`)-400 : strings.Index(out, `name="email"`)]
	if strings.Contains(email, `aria-invalid`) {
		t.Errorf("email control marked invalid:\n%s", email)
	}
}

func TestSignIn_ErrorBanner(t *testing.T) {
	p := signInProps()
	p.Error = "Email or password is incorrect."
	out := render(t, AuthPage(p))
	mustContain(t, out, `role="alert"`, "Email or password is incorrect.")
	mustNotContain(t, out, "Please fix", "<ul")
	mustOrder(t, out, "<h1", `role="alert"`, "<form")

	// With field errors too, Error is the title and the fields are listed.
	p.Errors = map[string]string{"email": "Unknown address", "other": "Something else"}
	out = render(t, AuthPage(p))
	mustContain(t, out, "Email or password is incorrect.", "Email: Unknown address", "<li>Something else</li>")
	mustNotContain(t, out, "Please fix")
}

func TestSignIn_SubmitOverrides(t *testing.T) {
	p := signInProps()
	p.Submit = action.Action{Label: "Let me in", Variant: action.VariantDestructive, Href: "/ignored", HxPost: "/ignored"}
	out := render(t, AuthPage(p))
	mustContain(t, out, `type="submit"`, "<span>Let me in</span>", "bg-destructive")
	mustNotContain(t, out, "/ignored")
}

func TestSignIn_LogoMessageFooter(t *testing.T) {
	p := signInProps()
	p.Logo = text("LOGO")
	p.Message = "Use your work account."
	p.Footer = text("No account? Create one.")
	out := render(t, AuthPage(p))
	mustOrder(t, out, "LOGO", "<h1", "Use your work account.", "<form", "No account? Create one.")
}

func TestSignIn_HeadingFallsBackToTitle(t *testing.T) {
	p := signInProps()
	p.AppName = ""
	out := render(t, AuthPage(p))
	mustContain(t, out, ">Sign in</h1>")
}

func TestSignIn_IDAndClass(t *testing.T) {
	p := signInProps()
	p.ID = "auth"
	p.Class = "bg-muted"
	p.Attributes = templ.Attributes{"data-x": "1"}
	out := render(t, AuthPage(p))
	mustContain(t, out, `id="auth"`, `<form id="auth-form"`, "bg-muted", `data-x="1"`)
}

func TestSignedOut_Defaults(t *testing.T) {
	out := render(t, AuthPage(Props{AppName: "Acme", Mode: ModeSignedOut, Action: "/signin"}))
	mustContain(t, out, "You have been signed out.", `href="/signin"`, "<span>Sign in again</span>", "w-full")
	mustNotContain(t, out, "<form", `type="submit"`, ">or</span>", `role="alert"`)
}

func TestSignedOut_SubmitHrefWins(t *testing.T) {
	out := render(t, AuthPage(Props{Mode: ModeSignedOut, Action: "/signin", Message: "Bye.", Submit: action.Action{Href: "/login", Label: "Back"}}))
	mustContain(t, out, "Bye.", `href="/login"`, "<span>Back</span>", ">Signed out</h1>")
	mustNotContain(t, out, `href="/signin"`, "You have been signed out.")
}

func TestForbidden_Defaults(t *testing.T) {
	out := render(t, AuthPage(Props{AppName: "Acme", Mode: ModeForbidden}))
	mustContain(t, out, "Your account does not have access.", `href="/"`, "<span>Go home</span>", "w-full")
	mustNotContain(t, out, "<form", ">or</span>")
}

func TestForbidden_HxAction(t *testing.T) {
	out := render(t, AuthPage(Props{Mode: ModeForbidden, Submit: action.Action{Label: "Sign out", HxPost: "/signout"}}))
	mustContain(t, out, `hx-post="/signout"`, "<span>Sign out</span>", ">Not allowed</h1>")
	mustNotContain(t, out, `href="/"`)
}

func TestTitle(t *testing.T) {
	for mode, want := range map[Mode]string{ModeSignIn: "Sign in", ModeSignedOut: "Signed out", ModeForbidden: "Not allowed", Mode("bogus"): "Sign in"} {
		if got := (Props{Mode: mode}).title(); got != want {
			t.Errorf("title(%q) = %q, want %q", mode, got, want)
		}
	}
}

func TestField(t *testing.T) {
	p := signInProps()
	p.Values = map[string]string{"email": "a@b.c"}
	p.Errors = map[string]string{"email": "Nope"}
	f, ok := p.Field("email")
	if !ok || f.Value != "a@b.c" || f.Error != "Nope" {
		t.Errorf("Field(email) = %+v, %v", f, ok)
	}
	if _, ok := p.Field("missing"); ok {
		t.Error("Field(missing) found")
	}
	if p.Fields[0].Value != "" {
		t.Error("resolve mutated the caller's field")
	}
}

// serveRespond runs Respond for one request shape, under App.Middleware
// when app is non-nil.
func serveRespond(t *testing.T, app *appshell.App, p Props, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var h http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := Respond(w, r, p); err != nil {
			t.Errorf("Respond: %v", err)
		}
	})
	if app != nil {
		h = app.Middleware(h)
	}
	req := httptest.NewRequest("GET", "/signin", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestRespond_WithApp(t *testing.T) {
	app := &appshell.App{Name: "Demo", HomeHref: "/", Nav: []appshell.NavItem{{Label: "Inbox", Href: "/inbox"}}}
	p := signInProps()
	p.AppName = ""
	rec := serveRespond(t, app, p, nil)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	out := rec.Body.String()
	if !strings.HasPrefix(strings.ToLower(out), "<!doctype") {
		t.Errorf("not a full document:\n%.80s", out)
	}
	mustContain(t, out, "<title>Sign in — Demo</title>", ">Demo</h1>", `<form id="signin-form"`)
	mustNotContain(t, out, "wf-shell", "wf-topbar", "<nav", `id="wf-flash"`, "Inbox")
}

func TestRespond_WithApp_HXBodyOnly(t *testing.T) {
	app := &appshell.App{Name: "Demo"}
	out := serveRespond(t, app, signInProps(), map[string]string{"HX-Request": "true"}).Body.String()
	if !strings.HasPrefix(out, `<div`) {
		t.Errorf("fragment does not start with the card column:\n%.80s", out)
	}
	mustContain(t, out, "wf-authpage", `<form id="signin-form"`)
	mustNotContain(t, out, "<html", "<title>", "wf-shell")
}

func TestRespond_WithApp_Titles(t *testing.T) {
	app := &appshell.App{Name: "Demo"}
	for mode, want := range map[Mode]string{ModeSignedOut: "<title>Signed out — Demo</title>", ModeForbidden: "<title>Not allowed — Demo</title>"} {
		out := serveRespond(t, app, Props{Mode: mode}, nil).Body.String()
		mustContain(t, out, want)
		mustNotContain(t, out, "wf-shell", "<nav")
	}
}

func TestRespond_NoApp(t *testing.T) {
	req := httptest.NewRequest("GET", "/signin", nil)
	if _, ok := appshell.FromContext(req.Context()); ok {
		t.Fatal("test precondition: no App expected in context")
	}
	rec := serveRespond(t, nil, signInProps(), nil)
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d", rec.Code)
	}
	out := rec.Body.String()
	if !strings.HasPrefix(strings.ToLower(out), "<!doctype") {
		t.Errorf("not a full document:\n%.80s", out)
	}
	mustContain(t, out, "<title>Sign in — Acme</title>", ">Acme</h1>", "output.css", `<form id="signin-form"`)
	mustNotContain(t, out, "wf-shell", "wf-topbar", "<nav", `id="wf-flash"`)
}

func TestRespond_NoApp_NoAppName(t *testing.T) {
	p := signInProps()
	p.AppName = ""
	out := serveRespond(t, nil, p, nil).Body.String()
	mustContain(t, out, "<title>Sign in</title>", ">Sign in</h1>")
	mustNotContain(t, out, "Sign in — ")
}

func TestRespond_NoApp_HXBodyOnly(t *testing.T) {
	out := serveRespond(t, nil, signInProps(), map[string]string{"HX-Request": "true"}).Body.String()
	mustContain(t, out, "wf-authpage")
	mustNotContain(t, out, "<html", "<title>")
}

func TestSignIn_AutocompleteDefaults(t *testing.T) {
	out := render(t, AuthPage(Props{Action: "/signin", Fields: []formfield.Field{
		{Name: "email", Label: "Email", Type: formfield.TypeEmail},
		{Name: "password", Label: "Password", Type: formfield.TypePassword},
		{Name: "code", Label: "Code", Type: formfield.TypeText},
		{Name: "pw2", Label: "New password", Type: formfield.TypePassword, Attributes: templ.Attributes{"autocomplete": "new-password"}},
	}}))
	mustContain(t, out, `autocomplete="username"`, `autocomplete="current-password"`, `autocomplete="new-password"`)
	if n := strings.Count(out, `autocomplete="`); n != 3 {
		t.Errorf("autocomplete appears %d times, want 3 (not on the text field):\n%s", n, out)
	}
}
