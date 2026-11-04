package lifecyclestepper

import (
	"strings"
	"testing"
)

// A connector must never begin a wrapped row. Each connector travels with
// the node after it, and the row is shifted left by one connector so the
// connector that leads each row (a real one after a wrap, an invisible one
// on the first row) sits in the clipped margin.
func TestRender_ConnectorTravelsWithNextNode(t *testing.T) {
	html := renderCompact(t, []Step{
		{Label: "A", State: StateDone},
		{Label: "B", State: StateDone},
		{Label: "C", State: StateFuture},
	})
	if !strings.Contains(html, "overflow:hidden") {
		t.Errorf("root must clip the leading connector of each row; html:\n%s", html)
	}
	// One step unit per node, each holding a connector then the node.
	if got := strings.Count(html, `data-wf-step`); got != 3 {
		t.Errorf("want 3 step units, got %d; html:\n%s", got, html)
	}
	// The second unit opens with the A->B connector, before B's label.
	i := strings.Index(html, ">B<")
	j := strings.LastIndex(html[:i], `data-wf-step`)
	if j < 0 {
		t.Fatalf("no step unit before B; html:\n%s", html)
	}
	if !strings.Contains(html[j:i], "background-color:var(--color-success);display:block") {
		t.Errorf("the A->B connector must sit in B's unit, before B; html:\n%s", html)
	}
}
