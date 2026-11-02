package demo

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bplus/webframe/demo/pages"
)

// HandleGraph renders the graph page (GET /graph) in the topbar shell.
func HandleGraph(w http.ResponseWriter, r *http.Request) {
	if err := App.Render(w, r, "Graph", pages.Graph()); err != nil {
		log.Printf("demo: render graph: %v", err)
	}
}

type graphNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type graphLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// HandleGraphData answers GET /api/graph with a small fixed graph in the
// shape graphwidget reads: {"nodes": [...], "links": [...]}.
func HandleGraphData(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Nodes []graphNode `json:"nodes"`
		Links []graphLink `json:"links"`
	}{
		Nodes: []graphNode{
			{ID: "web", Name: "Web server", Type: "service"},
			{ID: "db", Name: "Postgres", Type: "service"},
			{ID: "cache", Name: "Cache", Type: "tool"},
			{ID: "auth", Name: "Auth", Type: "concept"},
			{ID: "queue", Name: "Job queue", Type: "tool"},
			{ID: "worker", Name: "Worker", Type: "service"},
			{ID: "ops", Name: "On-call", Type: "person"},
			{ID: "outage", Name: "Outage", Type: "incident"},
		},
		Links: []graphLink{
			{Source: "web", Target: "db"},
			{Source: "web", Target: "cache"},
			{Source: "web", Target: "auth"},
			{Source: "web", Target: "queue"},
			{Source: "queue", Target: "worker"},
			{Source: "worker", Target: "db"},
			{Source: "outage", Target: "db"},
			{Source: "ops", Target: "outage"},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("demo: encode graph: %v", err)
	}
}
