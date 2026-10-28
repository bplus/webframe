package demo

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bplus/webframe/assets"
)

func TestShowcaseAndTrafficRenderThroughShell(t *testing.T) {
	mux := http.NewServeMux()
	assets.Mount(mux)
	mux.HandleFunc("GET /{$}", HandleShowcase)
	mux.HandleFunc("GET /traffic", HandleTraffic)
	srv := httptest.NewServer(App.Middleware(mux))
	defer srv.Close()

	get := func(path string) (int, string) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return resp.StatusCode, string(b)
	}

	for _, path := range []string{"/", "/traffic"} {
		code, body := get(path)
		if code != 200 {
			t.Errorf("GET %s = %d, want 200", path, code)
		}
		for _, want := range []string{"wf-topbar", " — Webframe</title>"} {
			if !strings.Contains(body, want) {
				t.Errorf("GET %s: body missing %q", path, want)
			}
		}
	}
	if code, _ := get("/static/css/output.css"); code != 200 {
		t.Errorf("GET /static/css/output.css = %d, want 200", code)
	}
}
