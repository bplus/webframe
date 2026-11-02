package demo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestGraphPageLoadsGraphScriptsOnlyThere: /graph carries both graphwidget
// scripts once; the showcase, which has no graph, carries neither (they
// are not part of appshell.Scripts). /api/graph answers the JSON shape the
// widget reads.
func TestGraphPageLoadsGraphScriptsOnlyThere(t *testing.T) {
	mux := http.NewServeMux()
	Routes(mux)
	srv := httptest.NewServer(App.Middleware(mux))
	defer srv.Close()

	get := func(path string) string {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, resp.StatusCode)
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return string(b)
	}

	graph := get("/graph")
	for _, js := range []string{"/static/js/3d-force-graph.min.js", "/static/js/graphwidget.min.js"} {
		if n := strings.Count(graph, js); n != 1 {
			t.Errorf("/graph loads %s %d times, want 1", js, n)
		}
	}
	if n := strings.Count(graph, "data-tui-graph-mount"); n != 2 {
		t.Errorf("/graph has %d graph mounts, want 2", n)
	}
	if !strings.Contains(graph, `data-tui-graph-data-url="/api/graph"`) {
		t.Errorf("/graph widgets do not point at /api/graph")
	}

	home := get("/")
	for _, js := range []string{"3d-force-graph.min.js", "graphwidget.min.js"} {
		if strings.Contains(home, js) {
			t.Errorf("showcase loads %s but has no graph", js)
		}
	}

	var data struct {
		Nodes []map[string]any `json:"nodes"`
		Links []map[string]any `json:"links"`
	}
	if err := json.Unmarshal([]byte(get("/api/graph")), &data); err != nil {
		t.Fatalf("/api/graph: %v", err)
	}
	if len(data.Nodes) == 0 || len(data.Links) == 0 {
		t.Errorf("/api/graph: %d nodes, %d links, want some of each", len(data.Nodes), len(data.Links))
	}
}
